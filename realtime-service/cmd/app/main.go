package main

import (
	"context"
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
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	instanceID := os.Getenv("INSTANCE_ID")
    if instanceID == "" {
        instanceID = "realtime-1"
    }

	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB: 0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		os.Exit(1)
	}
	defer redisClient.Close()

	redisRepo := repository.NewRedisRepository(redisClient)
	hub := service.NewHub(redisRepo, instanceID)

	consumer := service.NewKafkaConsumer(
		[]string{os.Getenv("KAKFA_BROKERS")},
		os.Getenv("KAFKA_TOPIC"),
		os.Getenv("KAFKA_TOPIC"),
		hub,
	)
	defer consumer.Close()

	go func() {
		if err := consumer.Start(ctx); err != nil {

		}
	}()
	
	grpcServer := grpc.NewServer()
	grpcHandler := transport.NewConncetionHandler(hub)
	realtimepb.RegisterConnectionServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		os.Exit(1)
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {

		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	grpcServer.GracefulStop()
	hub.CloseAll(context.Background())
}
