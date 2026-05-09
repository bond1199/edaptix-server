package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type AppConfig struct {
	App      AppConfigStruct      `mapstructure:"app"`
	Database DatabaseConfig       `mapstructure:"database"`
	Redis    RedisConfig          `mapstructure:"redis"`
	RabbitMQ RabbitMQConfig       `mapstructure:"rabbitmq"`
	MinIO    MinIOConfig          `mapstructure:"minio"`
	Consul   ConsulConfig         `mapstructure:"consul"`
	JWT      JWTConfig            `mapstructure:"jwt"`
	AI       AIConfig             `mapstructure:"ai"`
	Log      LogConfig            `mapstructure:"log"`
	Tracing  TracingConfig        `mapstructure:"tracing"`
}

type AppConfigStruct struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		c.Host, c.Port, c.User, c.Password, c.DBName)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	PublicURL string `mapstructure:"public_url"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type ConsulConfig struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
	Prefix  string `mapstructure:"prefix"`
}

type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	ExpireHours        int    `mapstructure:"expire_hours"`
	RefreshExpireHours int    `mapstructure:"refresh_expire_hours"`
}

type AIConfig struct {
	Flash AIModelConfig `mapstructure:"flash"`
	Pro   AIModelConfig `mapstructure:"pro"`
	OCR   OCRConfig     `mapstructure:"ocr"`
}

type AIModelConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	APIKey    string `mapstructure:"api_key"`
	Timeout   int    `mapstructure:"timeout"`
	ModelName string `mapstructure:"model_name"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

type OCRConfig struct {
	Engine       string `mapstructure:"engine"`
	Fallback     string `mapstructure:"fallback"`
	BaiduAPIKey  string `mapstructure:"baidu_api_key"`
	BaiduSecretKey string `mapstructure:"baidu_secret_key"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

var C AppConfig
var ConsulCC *ConsulCenter

func Load() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("internal/config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	viper.SetEnvPrefix("EDAPTIX")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := viper.Unmarshal(&C); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 尝试连接Consul拉取远程配置（连接失败则使用本地配置）
	if C.Consul.Address != "" {
		cc, err := NewConsulCenter(C.Consul)
		if err != nil {
			zap.L().Warn("consul connect failed, using local config", zap.Error(err))
		} else {
			if err := cc.ApplyToViper(); err != nil {
				zap.L().Warn("consul load config failed, using local config", zap.Error(err))
			} else {
				// 重新解析，让Consul配置覆盖本地值
				_ = viper.Unmarshal(&C)
				ConsulCC = cc
				zap.L().Info("consul config applied")
			}
		}
	}

	return &C, nil
}
