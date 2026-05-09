package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/edaptix/server/internal/dto/request"
	"github.com/edaptix/server/internal/dto/response"
	"github.com/edaptix/server/internal/model"
	"github.com/edaptix/server/internal/pkg/jwt"
	"github.com/edaptix/server/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo             *repository.UserRepo
	rdb              *redis.Client
	jwtSecret        string
	jwtExpire        int
	jwtRefreshExpire int
}

func NewUserService(repo *repository.UserRepo, rdb *redis.Client, jwtSecret string, jwtExpire, jwtRefreshExpire int) *UserService {
	return &UserService{
		repo:             repo,
		rdb:              rdb,
		jwtSecret:        jwtSecret,
		jwtExpire:        jwtExpire,
		jwtRefreshExpire: jwtRefreshExpire,
	}
}

// gradeStage 根据年级计算学段
func gradeStage(grade int) string {
	switch {
	case grade >= 1 && grade <= 6:
		return "primary"
	case grade >= 7 && grade <= 9:
		return "junior"
	case grade >= 10 && grade <= 12:
		return "senior"
	default:
		return "unknown"
	}
}

func (s *UserService) Register(ctx context.Context, req *request.RegisterRequest) (*response.RegisterResponse, error) {
	// 检查手机号是否已存在
	existing, err := s.repo.FindByPhone(ctx, req.Phone)
	if err != nil && err != gorm.ErrRecordNotFound {
		zap.L().Error("查询用户失败", zap.String("phone", req.Phone), zap.Error(err))
		return nil, fmt.Errorf("查询用户失败")
	}
	if existing != nil {
		return nil, fmt.Errorf("手机号已注册")
	}

	// 校验短信验证码
	valid, err := s.VerifySMS(ctx, req.Phone, req.SMSCode)
	if err != nil {
		return nil, fmt.Errorf("验证码校验失败: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	// bcrypt哈希密码 cost=12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		zap.L().Error("密码哈希失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	// 创建User记录
	user := &model.User{
		Phone:        req.Phone,
		PasswordHash: string(hashedPassword),
		Role:         "student",
		Status:       1,
		Initialized:  false,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		zap.L().Error("创建用户失败", zap.Error(err))
		return nil, fmt.Errorf("创建用户失败")
	}

	// 创建StudentProfile
	profile := &model.StudentProfile{
		UserID:     user.ID,
		RealName:   req.RealName,
		Grade:      int16(req.Grade),
		GradeStage: gradeStage(req.Grade),
	}
	if err := s.repo.CreateStudentProfile(ctx, profile); err != nil {
		zap.L().Error("创建学生档案失败", zap.Int64("user_id", user.ID), zap.Error(err))
		return nil, fmt.Errorf("创建学生档案失败")
	}

	// 生成JWT Token
	token, err := jwt.GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtExpire)
	if err != nil {
		zap.L().Error("生成Token失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	refreshToken, err := jwt.GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtRefreshExpire)
	if err != nil {
		zap.L().Error("生成RefreshToken失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	return &response.RegisterResponse{
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtExpire * 3600),
		Initialized:  user.Initialized,
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *request.LoginRequest) (*response.LoginResponse, error) {
	// 查找用户
	user, err := s.repo.FindByPhone(ctx, req.Phone)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("手机号或密码错误")
		}
		zap.L().Error("查询用户失败", zap.String("phone", req.Phone), zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("手机号或密码错误")
	}

	// 生成JWT Token
	token, err := jwt.GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtExpire)
	if err != nil {
		zap.L().Error("生成Token失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	refreshToken, err := jwt.GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtRefreshExpire)
	if err != nil {
		zap.L().Error("生成RefreshToken失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	return &response.LoginResponse{
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtExpire * 3600),
		Initialized:  user.Initialized,
	}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*response.LoginResponse, error) {
	// 解析refreshToken
	claims, err := jwt.ParseToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("无效的RefreshToken")
	}

	// 查找用户确认存在
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 生成新Token对
	token, err := jwt.GenerateToken(claims.UserID, claims.Role, s.jwtSecret, s.jwtExpire)
	if err != nil {
		zap.L().Error("生成Token失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	newRefreshToken, err := jwt.GenerateToken(claims.UserID, claims.Role, s.jwtSecret, s.jwtRefreshExpire)
	if err != nil {
		zap.L().Error("生成RefreshToken失败", zap.Error(err))
		return nil, fmt.Errorf("系统错误")
	}

	return &response.LoginResponse{
		UserID:       user.ID,
		Token:        token,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.jwtExpire * 3600),
		Initialized:  user.Initialized,
	}, nil
}

func (s *UserService) SendSMS(ctx context.Context, phone string) (string, error) {
	// 生成6位随机验证码
	code, err := randomCode(6)
	if err != nil {
		zap.L().Error("生成验证码失败", zap.Error(err))
		return "", fmt.Errorf("系统错误")
	}

	// 存入Redis key=sms:{phone} TTL=5min
	key := fmt.Sprintf("sms:%s", phone)
	if err := s.rdb.Set(ctx, key, code, 5*time.Minute).Err(); err != nil {
		zap.L().Error("存储验证码失败", zap.String("phone", phone), zap.Error(err))
		return "", fmt.Errorf("系统错误")
	}

	// 开发环境直接返回验证码（正式环境调短信API）
	zap.L().Info("发送验证码", zap.String("phone", phone), zap.String("code", code))
	return code, nil
}

func (s *UserService) VerifySMS(ctx context.Context, phone, code string) (bool, error) {
	// 开发环境固定123456直接通过
	if code == "123456" {
		return true, nil
	}

	key := fmt.Sprintf("sms:%s", phone)
	stored, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		zap.L().Error("获取验证码失败", zap.String("phone", phone), zap.Error(err))
		return false, fmt.Errorf("系统错误")
	}

	if stored != code {
		return false, nil
	}

	// 验证成功后删除验证码
	s.rdb.Del(ctx, key)
	return true, nil
}

func randomCode(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte('0' + n.Int64())
	}
	return string(result), nil
}
