package middleware

import (
	"net/http"
	"strings"

	pkgjwt "github.com/666Stepan66612/ZeroMes/pkg/jwt"
    "github.com/gin-gonic/gin"
)

func JWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie
		} else {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer")
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

		c.Set("userID", userID)
		c.Next()
	}
}