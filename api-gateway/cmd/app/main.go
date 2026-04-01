package main

import (
	"api-gateway/internal/cores/middleware"
	"api-gateway/internal/gateway/service"
	"api-gateway/internal/gateway/transport"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	messageServiceAddr := os.Getenv("MESSAGE_SERVICE_ADDR")
	realtimeServiceAddr := os.Getenv("REALTIME_SERVICE_ADDR")
	jwtSecret := os.Getenv("JWT_ACCESS_SECRET")

	messageClient, err := service.NewMessageClient(messageServiceAddr)
	if err != nil {
		slog.Error("failed to connect to message client", "err", err)
		os.Exit(1)
	}
	defer messageClient.Close()

	realtimeClient, err := service.NewRealtimeClient(realtimeServiceAddr)
	if err != nil {
		slog.Error("failed to connect to realtime client", "err", err)
		os.Exit(1)
	}
	defer realtimeClient.Close()

	redisClient := redis.NewClient(&redis.Options{
    	Addr: os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	gatewaySvc := service.NewGatewayService(messageClient, realtimeClient)

	wsHandler := transport.NewWebSocketHandler(gatewaySvc)
	authProxy, err := transport.NewAuthProxy(authServiceURL)
	if err != nil {
		slog.Error("failed to connect to auth proxy", "err", err)
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	})
	
	authLimit := middleware.RateLimiter(redisClient, 120, time.Minute)

	auth := r.Group("/auth")
	{
		auth.POST("/register", authLimit, authProxy.Register)
		auth.POST("/login", authLimit, authProxy.Login)
		auth.POST("/refresh", authLimit, authProxy.Refresh)
		auth.POST("/logout", authLimit, authProxy.Logout)
		auth.GET("/search", middleware.JWTMiddleware(jwtSecret, redisClient), authProxy.Search)
		auth.POST("/change-password", authLimit, authProxy.ChangePassword)
	}

	r.GET("/ws", middleware.JWTMiddleware(jwtSecret, redisClient), wsHandler.Handle)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8083"
		}
		if err := r.Run(":" + port); err != nil {
			slog.Error("failed to start server", "err", err)
		}
	}()

	<-quit
	slog.Info("shutting down api-gateway")
}