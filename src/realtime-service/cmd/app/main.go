package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"realtime-service/internal/connection/repository"
	"realtime-service/internal/connection/service"
	"realtime-service/internal/connection/transport"

	realtimepb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	redis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection" // for test
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "realtime-1"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to redis", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	redisRepo := repository.NewRedisRepository(redisClient)
	hub := service.NewHub(redisRepo, instanceID)

	consumer := service.NewKafkaConsumer(
		[]string{os.Getenv("KAFKA_BROKERS")},
		os.Getenv("KAFKA_TOPIC"),
		os.Getenv("KAFKA_GROUP_ID"),
		hub,
	)
	defer consumer.Close()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			slog.Error("kafka consumer error", "err", err)
		}
	}()

	jwtSecret := os.Getenv("JWT_SECRET")
	grpcServer := grpc.NewServer()
	grpcHandler := transport.NewConnectionHandler(hub, jwtSecret, redisClient)
	reflection.Register(grpcServer) // for test
	realtimepb.RegisterConnectionServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("realtime-service started", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	grpcServer.GracefulStop()
	hub.CloseAll(context.Background())
}
