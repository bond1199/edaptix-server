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

	"github.com/edaptix/server/internal/config"
	"github.com/edaptix/server/internal/middleware"
	"github.com/edaptix/server/internal/pkg/logger"
	jwtpkg "github.com/edaptix/server/internal/pkg/jwt"
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

	// Register routes
	v1 := r.Group("/api/v1")

	// Public routes: auth
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", middleware.PlaceholderHandler("auth/register"))
		authGroup.POST("/login", middleware.PlaceholderHandler("auth/login"))
		authGroup.POST("/refresh", middleware.PlaceholderHandler("auth/refresh"))
		authGroup.POST("/sms/send", middleware.PlaceholderHandler("auth/sms/send"))
		authGroup.POST("/sms/verify", middleware.PlaceholderHandler("auth/sms/verify"))
	}

	// Authenticated routes
	authMw := middleware.AuthMiddleware(cfg.JWT.Secret)
	_ = jwtpkg.GenerateToken // reference to jwt package

	authenticatedGroups := map[string][]struct {
		method  string
		path    string
		handler string
	}{
		"init": {
			{"GET", "", "init"},
		},
		"learning": {
			{"GET", "", "learning"},
		},
		"tasks": {
			{"GET", "", "tasks"},
		},
		"grading": {
			{"GET", "", "grading"},
		},
		"risk": {
			{"GET", "", "risk"},
		},
		"parent": {
			{"GET", "", "parent"},
		},
		"student": {
			{"GET", "", "student"},
		},
	}

	for group, routes := range authenticatedGroups {
		g := v1.Group("/" + group)
		g.Use(authMw)
		for _, route := range routes {
			g.Handle(route.method, route.path, middleware.PlaceholderHandler(route.handler+"/"+route.path))
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
