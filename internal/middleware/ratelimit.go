package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/edaptix/server/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Middleware applies global and per-user rate limiting.
// Global: 1000 req/s, Per-user: 100 req/min
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Global rate limit: 1000 req/s using sliding window
		if !rl.allowGlobal(ctx) {
			response.Error(c, http.StatusTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		// Per-user rate limit: 100 req/min (by user_id if authenticated, else by IP)
		key := c.ClientIP()
		if userID, exists := c.Get(ContextUserID); exists {
			key = fmt.Sprintf("user:%v", userID)
		}
		if !rl.allowPerUser(ctx, key) {
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

// allowGlobal uses a fixed window counter for global 1000 req/s
func (rl *RateLimiter) allowGlobal(ctx context.Context) bool {
	now := time.Now()
	key := fmt.Sprintf("ratelimit:global:%d", now.Unix())

	count, err := rl.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true // fail open on Redis error
	}
	if count == 1 {
		rl.rdb.Expire(ctx, key, 2*time.Second)
	}
	return count <= 1000
}

// allowPerUser uses a sliding window for per-user 100 req/min
func (rl *RateLimiter) allowPerUser(ctx context.Context, key string) bool {
	now := time.Now()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/60)

	count, err := rl.rdb.Incr(ctx, windowKey).Result()
	if err != nil {
		return true // fail open on Redis error
	}
	if count == 1 {
		rl.rdb.Expire(ctx, windowKey, 2*time.Minute)
	}
	return count <= 100
}
