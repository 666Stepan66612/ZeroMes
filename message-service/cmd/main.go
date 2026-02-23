package main

import (
	"context"
	"log"
	pb "message-service/gen/messagepb"
	"message-service/internal/messaging/repository"
	"message-service/internal/messaging/service"
	"message-service/internal/messaging/transport"
	"message-service/pkg/kafka"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func main() {
	databaseURL := os.Getenv("POSTGRES_URL")
	if databaseURL == "" {
		log.Fatal("POSTGRES_URL is required")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		log.Fatal("KAFKA_BROKERS is required")
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	ctx := context.Background()

	pgPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to Database:", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		log.Fatal("Failed to ping Database", err)
	}
	log.Println("Connected to Database")

	kafkaProducer := kafka.NewProducer([]string{kafkaBrokers}, "messages.sent")
	defer kafkaProducer.Close()
	log.Println("Kafka producer initialized")

	messageRepo := repository.NewPostgresRepository(pgPool)

	messageService := service.NewMessageService(messageRepo, kafkaProducer)

	grpcServer := grpc.NewServer()
	grpcHandler := transport.NewGRPCHandler(messageService)
	pb.RegisterMessageServiceServer(grpcServer, grpcHandler)

	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("Failed to listen: ", err)
	}

	go func() {
		log.Println("gRPC server listening on: ", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Failed to serve: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Graceful shutdown start")

	grpcServer.GracefulStop()

	log.Println("Server stopped")
}
