package main

import (
	"api-gateway/internal/cores/middleware"
	"api-gateway/internal/gateway/service"
	"api-gateway/internal/gateway/transport"
	"os"
	"os/signal"
	"syscall"
	"log/slog"

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
	}
	defer messageClient.Close()

	realtimeClient, err := service.NewRealtimeClient(realtimeServiceAddr)
	if err != nil {
		slog.Error("failed to connect to realtime client", "err", err)
	}
	defer realtimeClient.Close()

	redisClient := redis.NewClient(&redis.Options{
    	Addr: os.Getenv("REDIS_ADDR"),
	})

	gatewaySvc := service.NewGatewayService(messageClient, realtimeClient)

	wsHandler := transport.NewWebSocketHandler(gatewaySvc)
	authProxy, err := transport.NewAuthProxy(authServiceURL)
	if err != nil {
		slog.Error("failed to connect to auth proxy", "err", err)
	}

	r := gin.Default()

	auth := r.Group("/auth")
	{
		auth.POST("/register", authProxy.Register)
		auth.POST("/login", authProxy.Login)
		auth.POST("/refresh", authProxy.Refresh)
		auth.POST("/logout", authProxy.Logout)
		auth.GET("/search", middleware.JWTMiddleware(jwtSecret, redisClient), authProxy.Search)
	}

	r.GET("/ws", middleware.JWTMiddleware(jwtSecret, redisClient), wsHandler.Handle)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := r.Run(":8083"); err != nil {
			slog.Error("failed to start server", "err", err)
		}
	}()

	<-quit
	slog.Info("shutting down api-gateway")
}