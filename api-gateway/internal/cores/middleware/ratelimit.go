package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", c.FullPath(), c.ClientIP())

		pipe := redisClient.Pipeline()
		incr := pipe.Incr(context.Background(), key)
		pipe.Expire(context.Background(), key, window)
		if _, err := pipe.Exec(context.Background()); err != nil {
			c.Next()
			return
		}

		if incr.Val() > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, try again later",
			})
			return
		}

		c.Next()
	}
}
