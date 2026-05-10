package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ConsulCenter Consul配置中心
type ConsulCenter struct {
	client     *api.Client
	prefix     string
	cache      sync.Map
	mu         sync.RWMutex
	watching   bool
	cancel     context.CancelFunc
	onChangeCb ConfigChangeCallback
}

// NewConsulCenter 创建Consul配置中心
func NewConsulCenter(cfg ConsulConfig, env string) (*ConsulCenter, error) {
	consulConfig := api.DefaultConfig()
	if cfg.Address != "" {
		consulConfig.Address = cfg.Address
	}
	if cfg.Token != "" {
		consulConfig.Token = cfg.Token
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("create consul client failed: %w", err)
	}

	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = env + "/" + prefix
	}

	return &ConsulCenter{
		client: client,
		prefix: prefix,
	}, nil
}

// LoadFromConsul 从Consul拉取配置并覆盖Viper本地配置
func (cc *ConsulCenter) LoadFromConsul() error {
	pairs, _, err := cc.client.KV().List(cc.prefix, nil)
	if err != nil {
		return fmt.Errorf("consul list keys failed: %w", err)
	}

	if len(pairs) == 0 {
		zap.L().Info("no config found in consul", zap.String("prefix", cc.prefix))
		return nil
	}

	for _, pair := range pairs {
		if pair.Value == nil {
			continue
		}
		key := strings.TrimPrefix(pair.Key, cc.prefix)
		if key == "" {
			continue
		}

		value := string(pair.Value)
		cc.cache.Store(key, value)

		var jsonVal interface{}
		if err := json.Unmarshal(pair.Value, &jsonVal); err == nil {
			cc.setViperFromJSON(key, jsonVal)
		} else {
			viper.Set(key, value)
		}
	}

	zap.L().Info("consul config loaded", zap.Int("keys", len(pairs)))
	return nil
}

// setViperFromJSON 将JSON值展开设置到Viper
func (cc *ConsulCenter) setViperFromJSON(prefix string, val interface{}) {
	switch v := val.(type) {
	case map[string]interface{}:
		for k, child := range v {
			fullKey := prefix + "." + k
			cc.setViperFromJSON(fullKey, child)
		}
	default:
		viper.Set(prefix, v)
	}
}

// Get 从Consul缓存获取配置值
func (cc *ConsulCenter) Get(key string) (string, bool) {
	if val, ok := cc.cache.Load(key); ok {
		return val.(string), true
	}
	return "", false
}

// GetInt 获取整数配置
func (cc *ConsulCenter) GetInt(key string) int {
	if val, ok := cc.Get(key); ok {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
			return i
		}
	}
	return viper.GetInt(key)
}

// GetBool 获取布尔配置
func (cc *ConsulCenter) GetBool(key string) bool {
	if val, ok := cc.Get(key); ok {
		s := strings.ToLower(val)
		return s == "true" || s == "1" || s == "yes"
	}
	return viper.GetBool(key)
}

// WatchConfig 启动配置变更监听（长轮询）
func (cc *ConsulCenter) WatchConfig(ctx context.Context) {
	if cc.watching {
		return
	}
	cc.watching = true

	watchCtx, cancel := context.WithCancel(ctx)
	cc.cancel = cancel

	go func() {
		defer func() {
			cc.watching = false
			zap.L().Info("consul config watch stopped")
		}()

		var lastIndex uint64

		for {
			select {
			case <-watchCtx.Done():
				return
			default:
			}

			pairs, meta, err := cc.client.KV().List(cc.prefix, &api.QueryOptions{
				WaitIndex: lastIndex,
				WaitTime:  30 * time.Second,
			})

			if err != nil {
				zap.L().Error("consul watch failed", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}

			if meta.LastIndex == lastIndex {
				continue
			}
			lastIndex = meta.LastIndex

			changed := false
			for _, pair := range pairs {
				if pair.Value == nil {
					continue
				}
				key := strings.TrimPrefix(pair.Key, cc.prefix)
				if key == "" {
					continue
				}

				newVal := string(pair.Value)
				if oldVal, ok := cc.cache.Load(key); !ok || oldVal.(string) != newVal {
					cc.cache.Store(key, newVal)
					changed = true

					var jsonVal interface{}
					if err := json.Unmarshal(pair.Value, &jsonVal); err == nil {
						cc.setViperFromJSON(key, jsonVal)
					} else {
						viper.Set(key, newVal)
					}
				}
			}

			if changed {
				zap.L().Info("consul config updated", zap.Uint64("index", lastIndex))
				if cc.onChangeCb != nil {
					cc.onChangeCb()
				}
			}
		}
	}()

	zap.L().Info("consul config watch started", zap.String("prefix", cc.prefix))
}

// StopWatch 停止监听
func (cc *ConsulCenter) StopWatch() {
	if cc.cancel != nil {
		cc.cancel()
	}
}

// ConfigChangeCallback 配置变更回调类型
type ConfigChangeCallback func()

// OnChange 注册配置变更回调
func (cc *ConsulCenter) OnChange(callback ConfigChangeCallback) {
	cc.onChangeCb = callback
}

// ApplyToViper 将Consul配置应用到Viper（启动时调用）
func (cc *ConsulCenter) ApplyToViper() error {
	return cc.LoadFromConsul()
}

// IsConnected 检查Consul连接是否正常
func (cc *ConsulCenter) IsConnected() bool {
	_, err := cc.client.Agent().Self()
	return err == nil
}
