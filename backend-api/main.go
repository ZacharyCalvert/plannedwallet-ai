package main

import (
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/database"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/repository"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-jwt-key-replace-in-production")
	jwtExpStr := getEnv("JWT_EXPIRATION_SECONDS", "3600")
	jwtExp, err := strconv.Atoi(jwtExpStr)
	if err != nil {
		jwtExp = 3600
	}

	authService := auth.NewService(jwtSecret, jwtExp)

	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "postgres")
	pgPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	pgDB := getEnv("POSTGRES_DB", "plannedwallet")
	pgSSLMode := getEnv("POSTGRES_SSLMODE", "disable")

	var repo repository.Repository

	db, err := database.Connect(database.Config{
		Host:     pgHost,
		Port:     pgPort,
		User:     pgUser,
		Password: pgPassword,
		DBName:   pgDB,
		SSLMode:  pgSSLMode,
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL (%v). Falling back to In-Memory repository.", err)
		repo = repository.NewMemoryRepository()
	} else {
		log.Println("Successfully connected to PostgreSQL. Running migrations...")
		if err := database.Migrate(db); err != nil {
			log.Fatalf("Database migration failed: %v", err)
		}
		log.Println("Database migrations applied successfully.")
		repo = repository.NewPostgresRepository(db)
	}

	r := gin.Default()

	// Health check / smoke test endpoint
	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "world")
	})

	handler := handlers.NewHandler(repo, authService)
	handler.RegisterRoutes(r)

	log.Printf("Planned Wallet API listening on port :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
