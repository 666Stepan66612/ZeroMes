package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	pkgjwt "github.com/666Stepan66612/ZeroMes/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func JWTMiddleware(secret string, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Try cookie first
		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie
			println("[JWT] Token from cookie:", token[:20]+"...")
		} else {
			println("[JWT] Cookie error:", err.Error())
			// Try Authorization header
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}

		// For WebSocket: try query parameter (since cookies may not be sent)
		if token == "" {
			token = c.Query("token")
			if token != "" {
				println("[JWT] Token from query")
			}
		}

		if token == "" {
			println("[JWT] No token found, returning 401")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID, err := pkgjwt.ValidateAccessToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if _, err := uuid.Parse(userID); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
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
