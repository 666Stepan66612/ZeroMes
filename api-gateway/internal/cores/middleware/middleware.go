package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	pkgjwt "github.com/666Stepan66612/ZeroMes/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func JWTMiddleware(secret string, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie
		} else {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return 
		}

		userID, err := pkgjwt.ValidateAccessToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return 
		}

		hash := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(hash[:])
		val, _ := redisClient.Get(context.Background(), "blacklist:"+tokenHash).Result()
		if val != "" {
    		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
    		return
		}

		c.Set("userID", userID)
		c.Next()
	}
}