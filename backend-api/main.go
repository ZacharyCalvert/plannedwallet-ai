package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/captcha"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/cors"
	"github.com/plannedwallet/internal/database"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func initJWTService() *auth.Service {
	jwtSecret := getEnv("JWT_SECRET", "super-secret-jwt-key-replace-in-production")
	jwtExpStr := getEnv("JWT_EXPIRATION_SECONDS", "3600")
	jwtExp, err := strconv.Atoi(jwtExpStr)
	if err != nil {
		jwtExp = 3600
	}
	return auth.NewService(jwtSecret, jwtExp)
}

func initDatabase() repository.Repository {
	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "postgres")
	pgPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	pgDB := getEnv("POSTGRES_DB", "plannedwallet")
	pgSSLMode := getEnv("POSTGRES_SSLMODE", "disable")

	db, err := database.ConnectWithRetry(database.Config{
		Host:     pgHost,
		Port:     pgPort,
		User:     pgUser,
		Password: pgPassword,
		DBName:   pgDB,
		SSLMode:  pgSSLMode,
	}, 5, time.Second)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL (%v). Falling back to In-Memory repository.", err)
		return repository.NewMemoryRepository()
	}

	log.Println("Successfully connected to PostgreSQL. Initializing database schema...")
	if err := database.InitDatabase(db); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	log.Println("Database initialized successfully.")
	return repository.NewPostgresRepository(db)
}

func addHandlers(r *gin.Engine, repo repository.Repository, authService *auth.Service) {
	ctrl := controller.NewController(repo, authService)
	if getEnv("REQUIRE_CAPTCHA", "false") == "true" {
		ctrl.SetRequireCaptcha(true)
	}
	handler := handlers.NewHandler(ctrl, authService)
	handler.RegisterRoutes(r)
}

func main() {
	port := getEnv("PORT", "8080")

	authService := initJWTService()
	repo := initDatabase()

	captcha.SetCustomStore(captcha.NewRepoStore(repo))
	if _, err := captcha.RunStartupCleanup(context.Background(), repo); err != nil {
		log.Printf("Warning: Startup CAPTCHA cleanup failed: %v", err)
	}

	r := gin.New()
	r.Use(trace.Middleware())
	r.Use(cors.Middleware())
	r.Use(gin.Recovery())
	addHandlers(r, repo, authService)

	log.Printf("Planned Wallet API listening on port :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
