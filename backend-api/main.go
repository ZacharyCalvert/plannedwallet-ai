package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/captcha"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/cors"
	"github.com/plannedwallet/internal/database"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

const defaultJWTSecret = "super-secret-jwt-key-replace-in-production"

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func isProduction() bool {
	return strings.EqualFold(os.Getenv("PRODUCTION_ENV"), "true")
}

func isDefaultJWTSecret(secret string) bool {
	clean := strings.ToLower(strings.TrimSpace(secret))
	if clean == "" || clean == defaultJWTSecret || clean == "dev-super-secret-jwt-key" {
		return true
	}
	return strings.Contains(clean, "super-secret") || strings.Contains(clean, "super secret")
}

func validateExpectedFDID(expectedFDID string, isProd bool) {
	if isProd && strings.TrimSpace(expectedFDID) == "" {
		log.Fatalf("Fatal: PRODUCTION_ENV is true but EXPECTED_FDID is not configured. An expected Azure Front Door ID is required.")
	}
}

func getIncomingFDID(c *gin.Context) string {
	if id := c.GetHeader("X-Azure-FDID"); id != "" {
		return strings.TrimSpace(id)
	}
	return strings.TrimSpace(c.GetHeader("X-FDID"))
}

func fdidMiddleware(expectedFDID string, isProd bool) gin.HandlerFunc {
	cleanExpected := strings.TrimSpace(expectedFDID)
	return func(c *gin.Context) {
		if (!isProd && cleanExpected == "") || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/api/health" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		if !strings.EqualFold(getIncomingFDID(c), cleanExpected) {
			trace.LogError(c.Request.Context(), nil, "Access denied: missing or invalid Azure Front Door ID header")
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Error:     "INVALID_PROXY_HEADER",
				Message:   "Access denied: missing or invalid Azure Front Door ID.",
				Timestamp: time.Now().UTC(),
			})
			return
		}
		c.Next()
	}
}

func initJWTService(isProd bool) *auth.Service {
	jwtSecret := getEnv("JWT_SECRET", defaultJWTSecret)
	if isProd && isDefaultJWTSecret(jwtSecret) {
		log.Fatalf("Fatal: PRODUCTION_ENV is true but default JWT secret is in use. Secure JWT_SECRET required.")
	}
	jwtExpStr := getEnv("JWT_EXPIRATION_SECONDS", "3600")
	jwtExp, err := strconv.Atoi(jwtExpStr)
	if err != nil {
		jwtExp = 3600
	}
	return auth.NewService(jwtSecret, jwtExp)
}

func applyDatabaseSchema(db *sql.DB, isProd bool) {
	if isProd {
		log.Println("PRODUCTION_ENV is true: skipping init.sql database initialization.")
		return
	}
	log.Println("Initializing database schema from init.sql...")
	if err := database.InitDatabase(db); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	log.Println("Database initialized successfully.")
}

func initDatabase(isProd bool) repository.Repository {
	cfg := database.Config{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     getEnv("POSTGRES_PORT", "5432"),
		User:     getEnv("POSTGRES_USER", "postgres"),
		Password: getEnv("POSTGRES_PASSWORD", "postgres"),
		DBName:   getEnv("POSTGRES_DB", "plannedwallet"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}

	db, err := database.ConnectWithRetry(cfg, 5, time.Second)
	if err != nil {
		if isProd {
			log.Fatalf("Fatal: Database connection failed in PRODUCTION_ENV: %v", err)
		}
		log.Printf("Warning: Failed to connect to PostgreSQL (%v). Falling back to In-Memory repository.", err)
		return repository.NewMemoryRepository()
	}

	applyDatabaseSchema(db, isProd)
	return repository.NewPostgresRepository(db)
}

func configureProxies(r *gin.Engine, isProd bool) {
	if !isProd {
		return
	}
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}
}

func addHandlers(r *gin.Engine, repo repository.Repository, authService *auth.Service, isProd bool) {
	ctrl := controller.NewController(repo, authService)
	if isProd || getEnv("REQUIRE_CAPTCHA", "false") == "true" {
		ctrl.SetRequireCaptcha(true)
	}
	handler := handlers.NewHandler(ctrl, authService)
	handler.RegisterRoutes(r)
}

func setupGinEngine(repo repository.Repository, authService *auth.Service, isProd bool, expectedFDID string) *gin.Engine {
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(trace.Middleware())
	r.Use(cors.Middleware())
	r.Use(fdidMiddleware(expectedFDID, isProd))
	r.Use(gin.Recovery())
	configureProxies(r, isProd)
	addHandlers(r, repo, authService, isProd)
	return r
}

func cleanupStartupCaptchas(repo repository.Repository) {
	captcha.SetCustomStore(captcha.NewRepoStore(repo))
	if _, err := captcha.RunStartupCleanup(context.Background(), repo); err != nil {
		log.Printf("Warning: Startup CAPTCHA cleanup failed: %v", err)
	}
}

func runGracefulServer(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Planned Wallet API listening on %s (graceful shutdown enabled)", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server listen error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutdown signal received. Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}
	log.Println("Server exited gracefully.")
}

func runServer(srv *http.Server, isProd bool) {
	if isProd {
		runGracefulServer(srv)
		return
	}
	log.Printf("Planned Wallet API listening on %s (standard mode)", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server stopped: %v", err)
	}
}

func main() {
	isProd := isProduction()
	port := getEnv("PORT", "8080")
	expectedFDID := getEnv("EXPECTED_FDID", "")
	validateExpectedFDID(expectedFDID, isProd)

	authService := initJWTService(isProd)
	repo := initDatabase(isProd)
	cleanupStartupCaptchas(repo)

	r := setupGinEngine(repo, authService, isProd, expectedFDID)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	runServer(srv, isProd)
}
