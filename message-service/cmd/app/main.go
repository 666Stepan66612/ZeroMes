package main

import (
	"context"
	"log"
	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"message-service/internal/messaging/repository"
	"message-service/internal/messaging/service"
	"message-service/internal/messaging/transport"
	worker "message-service/internal/cores/outbox-worker"
	"message-service/pkg/kafka"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc/reflection" // for test
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
	outboxRepo := repository.NewOutboxRepository(pgPool)
	outboxWorker := worker.NewOutboxWorker(outboxRepo, kafkaProducer)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go outboxWorker.Start(workerCtx)

	messageService := service.NewMessageService(messageRepo, kafkaProducer, outboxRepo)

	grpcServer := grpc.NewServer()
	grpcHandler := transport.NewGRPCHandler(messageService)
	pb.RegisterMessageServiceServer(grpcServer, grpcHandler)

	reflection.Register(grpcServer) // for test

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
