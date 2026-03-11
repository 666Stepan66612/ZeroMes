package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"auth-service/internal/auth/repository"
    "auth-service/internal/auth/service"
    "auth-service/internal/auth/transport"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwtAccessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if jwtAccessSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET is required")
	}

	jwtRefreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if jwtRefreshSecret == "" {
		log.Fatal("JWT_REFRESH_SECRET is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Unable to ping database:", err)
	}
	log.Println("Connected to PostgreSQL")

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB: 0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Unable to connect to Redis:", err)
	}
	log.Println("Connected to Redis")
	defer redisClient.Close()

	userRepo := repository.NewPostgresUserRepository(pool)
	tokenService := service.NewTokenService(jwtAccessSecret, jwtRefreshSecret, redisClient)
	authService := service.NewAuthService(userRepo, tokenService)
	handler := transport.NewHandler(authService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", handler.Register)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.HandleFunc("POST /auth/refresh", handler.RefreshToken)
	mux.HandleFunc("POST /auth/logout", handler.Logout)
	mux.HandleFunc("GET /auth/search", handler.Search)

	server := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<- quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server stopped")
}