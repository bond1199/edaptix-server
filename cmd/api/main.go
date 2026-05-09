package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edaptix/server/internal/ai/llm"
	"github.com/edaptix/server/internal/ai/ocr"
	"github.com/edaptix/server/internal/config"
	"github.com/edaptix/server/internal/handler"
	"github.com/edaptix/server/internal/middleware"
	"github.com/edaptix/server/internal/pkg/logger"
	"github.com/edaptix/server/internal/pkg/storage"
	"github.com/edaptix/server/internal/repository"
	"github.com/edaptix/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	zapLogger := logger.NewLogger(cfg.Log)
	defer zapLogger.Sync()

	// Replace global logger
	zap.ReplaceGlobals(zapLogger)

	// Initialize DB
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		zapLogger.Fatal("failed to connect database", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		zapLogger.Fatal("failed to get underlying sql.DB", zap.Error(err))
	}
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	defer sqlDB.Close()

	zapLogger.Info("database connected")

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		zapLogger.Fatal("failed to connect redis", zap.Error(err))
	}
	defer rdb.Close()

	zapLogger.Info("redis connected")

	// Initialize MinIO
	minioProvider, err := storage.NewMinIOProvider(cfg.MinIO)
	if err != nil {
		zapLogger.Fatal("failed to initialize MinIO", zap.Error(err))
	}
	zapLogger.Info("minio connected")

	// Initialize OCR Pipeline
	ocrPipeline := ocr.NewPipeline(cfg.AI.OCR)
	zapLogger.Info("ocr pipeline initialized", zap.String("engine", cfg.AI.OCR.Engine))

	// Initialize DeepSeek LLM Client
	llmClient := llm.NewDeepSeekClient(cfg.AI)
	zapLogger.Info("deepseek client initialized",
		zap.String("flash_model", cfg.AI.Flash.ModelName),
		zap.String("pro_model", cfg.AI.Pro.ModelName),
	)

	// Initialize repositories
	userRepo := repository.NewUserRepo(db)
	treeRepo := repository.NewKnowledgeTreeRepo(db)
	dataRepo := repository.NewLearningDataRepo(db)

	// Initialize services
	userSvc := service.NewUserService(
		userRepo, rdb,
		cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.JWT.RefreshExpireHours,
	)

	knowledgeTreeSvc := service.NewKnowledgeTreeService(
		treeRepo,
		dataRepo,
		userRepo,
		ocrPipeline,
		llmClient,
		minioProvider,
		db,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userSvc)
	uploadHandler := handler.NewUploadHandler(minioProvider)
	initHandler := handler.NewInitHandler(knowledgeTreeSvc)

	// Create Gin engine
	gin.SetMode(gin.ReleaseMode)
	if cfg.App.Env == "development" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// Register middlewares
	r.Use(middleware.Recovery(zapLogger))
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())

	rateLimiter := middleware.NewRateLimiter(rdb)
	r.Use(rateLimiter.Middleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// === Public routes ===
	v1 := r.Group("/api/v1")

	// Auth routes
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/sms/send", authHandler.SendSMS)
		authGroup.POST("/sms/verify", authHandler.VerifySMS)
	}

	// === Authenticated routes ===
	authMw := middleware.AuthMiddleware(cfg.JWT.Secret)

	// Init routes
	initGroup := v1.Group("/init")
	initGroup.Use(authMw)
	{
		initGroup.GET("/status", initHandler.GetInitStatus)
		initGroup.POST("/catalog", initHandler.InitFromCatalog)  // 新增：教材目录初始化
		initGroup.POST("/complete", initHandler.CompleteInit)
	}

	// Knowledge tree routes
	treeGroup := v1.Group("/knowledge-tree")
	treeGroup.Use(authMw)
	{
		treeGroup.GET("/:id", initHandler.GetKnowledgeTree)
	}

	// Upload routes
	uploadGroup := v1.Group("/upload")
	uploadGroup.Use(authMw)
	{
		uploadGroup.POST("/image", uploadHandler.UploadImage)
	}

	// Placeholder routes (to be implemented)
	placeholderGroups := map[string][]struct {
		method  string
		path    string
		name    string
	}{
		"learning": {
			{"POST", "/upload", "learning/upload"},
			{"GET", "/uploads", "learning/uploads"},
			{"GET", "/errors", "learning/errors"},
			{"GET", "/stats", "learning/stats"},
		},
		"tasks": {
			{"GET", "/today", "tasks/today"},
			{"GET", "/history", "tasks/history"},
			{"GET", "/:id", "tasks/detail"},
			{"POST", "/:id/start", "tasks/start"},
			{"POST", "/:id/answer", "tasks/answer"},
			{"POST", "/:id/finish", "tasks/finish"},
		},
		"grading": {
			{"GET", "/results/:task_id", "grading/results"},
			{"GET", "/detail/:item_id", "grading/detail"},
		},
		"risk": {
			{"POST", "/violation", "risk/violation"},
			{"GET", "/records", "risk/records"},
			{"GET", "/integrity", "risk/integrity"},
		},
		"parent": {
			{"POST", "/bind", "parent/bind"},
			{"GET", "/dashboard", "parent/dashboard"},
			{"GET", "/subject/:subject", "parent/subject-detail"},
			{"GET", "/reports", "parent/reports"},
			{"GET", "/risk-records", "parent/risk-records"},
		},
		"student": {
			{"GET", "/bind-code", "student/bind-code"},
			{"POST", "/unbind", "student/unbind"},
		},
	}

	for group, routes := range placeholderGroups {
		g := v1.Group("/" + group)
		g.Use(authMw)
		for _, route := range routes {
			g.Handle(route.method, route.path, middleware.PlaceholderHandler(route.name))
		}
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		zapLogger.Info("starting server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("listen failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Fatal("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("server exited")
}
