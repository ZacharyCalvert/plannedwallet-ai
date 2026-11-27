package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
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

const (
	defaultJWTSecret  = "super-secret-jwt-key-replace-in-production"
	minJWTSecretBytes = 32 // 256 bits
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func isProduction() bool {
	return strings.EqualFold(os.Getenv("PRODUCTION_ENV"), "true")
}

func isReadOnlyMode() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PW_READONLY")), "true")
}

func isPolicyEnforced(isProd bool) bool {
	return isProd || strings.EqualFold(strings.TrimSpace(os.Getenv("ENFORCE_POLICY")), "true")
}

func isSubscriberAlwaysOn() bool {
	return os.Getenv("SUBSCRIBER_ALWAYS_ON") == "true"
}

func parseSubscriberFreeDays(raw string) int {
	days, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || days <= 0 {
		return 0
	}
	return days
}

func checkEnforcePolicyForProd(rawEnforce string, isProd bool) error {
	if isProd && !strings.EqualFold(strings.TrimSpace(rawEnforce), "true") {
		return fmt.Errorf("PRODUCTION_ENV is true but ENFORCE_POLICY is not set to \"true\"")
	}
	return nil
}

func validatePolicyConfig(rawEnforce, rawTerms, rawPrivacy string, isProd bool) (string, string) {
	if err := checkEnforcePolicyForProd(rawEnforce, isProd); err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	enforced := isPolicyEnforced(isProd)
	return validatePolicyHash("PW_TERMS", rawTerms, enforced), validatePolicyHash("PW_PRIVACY", rawPrivacy, enforced)
}

func validatePolicyHash(envName, rawHash string, enforcePolicy bool) string {
	clean := strings.ToLower(strings.TrimSpace(rawHash))
	if enforcePolicy && !controller.IsValidSHA256Hex(clean) {
		log.Fatalf("Fatal: Policy enforcement is active (PRODUCTION_ENV or ENFORCE_POLICY) but %s is missing or invalid SHA-256 hex.", envName)
	}
	if clean != "" && !controller.IsValidSHA256Hex(clean) {
		log.Fatalf("Fatal: %s is set but is not a valid 64-character hex SHA-256 digest.", envName)
	}
	return clean
}

func isDefaultJWTSecret(secret string) bool {
	clean := strings.ToLower(strings.TrimSpace(secret))
	if clean == "" || clean == defaultJWTSecret || clean == "dev-super-secret-jwt-key" {
		return true
	}
	return strings.Contains(clean, "super-secret") || strings.Contains(clean, "super secret")
}

func validateExpectedOriginSecret(expectedSecret string, isProd bool) {
	if isProd && strings.TrimSpace(expectedSecret) == "" {
		log.Fatalf("Fatal: PRODUCTION_ENV is true but EXPECTED_ORIGIN_SECRET is not configured. An expected origin secret is required.")
	}
}

func getIncomingOriginSecret(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("X-Origin-Verify"))
}

func isOriginSecretValid(incoming, expected string) bool {
	if expected == "" || len(incoming) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(incoming), []byte(expected)) == 1
}

func isOriginExempt(c *gin.Context, cleanExpected string, isProd bool) bool {
	if (!isProd && cleanExpected == "") || c.Request.Method == http.MethodOptions {
		return true
	}
	return c.Request.URL.Path == "/api/health" || c.Request.URL.Path == "/health"
}

func originVerifyMiddleware(expectedSecret string, isProd bool) gin.HandlerFunc {
	cleanExpected := strings.TrimSpace(expectedSecret)
	return func(c *gin.Context) {
		if isOriginExempt(c, cleanExpected, isProd) {
			c.Next()
			return
		}
		if !isOriginSecretValid(getIncomingOriginSecret(c), cleanExpected) {
			trace.LogError(c.Request.Context(), nil, "Access denied: missing or invalid X-Origin-Verify header")
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Error:     "INVALID_PROXY_HEADER",
				Message:   "Access denied: missing or invalid origin verification header.",
				Timestamp: time.Now().UTC(),
			})
			return
		}
		c.Set("originVerified", true)
		c.Next()
	}
}

func checkJWTSecretBits(secret string) error {
	if secret != "" && len([]byte(secret)) < minJWTSecretBytes {
		return fmt.Errorf("JWT_SECRET is set but is less than 256 bits (32 bytes): length is %d bytes", len([]byte(secret)))
	}
	return nil
}

func validateJWTSecret(rawSecret string, isProd bool) {
	if err := checkJWTSecretBits(rawSecret); err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	if isProd && isDefaultJWTSecret(rawSecret) {
		log.Fatalf("Fatal: PRODUCTION_ENV is true but default JWT secret is in use. Secure JWT_SECRET required.")
	}
}

func initJWTService(isProd bool) *auth.Service {
	rawSecret := os.Getenv("JWT_SECRET")
	validateJWTSecret(rawSecret, isProd)

	jwtSecret := rawSecret
	if jwtSecret == "" {
		jwtSecret = defaultJWTSecret
	}
	jwtExpStr := getEnv("JWT_EXPIRATION_SECONDS", strconv.Itoa(auth.DefaultExpirationSeconds))
	jwtExp, err := strconv.Atoi(jwtExpStr)
	if err != nil {
		jwtExp = auth.DefaultExpirationSeconds
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
	defaultSSLMode := "disable"
	if isProd {
		defaultSSLMode = "require"
	}
	cfg := database.Config{
		Host:     getEnv("POSTGRES_HOST", getEnv("DB_HOST", "localhost")),
		Port:     getEnv("POSTGRES_PORT", getEnv("DB_PORT", "5432")),
		User:     getEnv("POSTGRES_USER", getEnv("DB_USER", "postgres")),
		Password: getEnv("POSTGRES_PASSWORD", getEnv("DB_PASSWORD", "postgres")),
		DBName:   getEnv("POSTGRES_DB", getEnv("DB_NAME", "plannedwallet")),
		SSLMode:  getEnv("POSTGRES_SSLMODE", getEnv("DB_SSLMODE", defaultSSLMode)),
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
	trustedVPCProxies := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
	}
	if err := r.SetTrustedProxies(trustedVPCProxies); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}
}

func addHandlers(r *gin.Engine, repo repository.Repository, authService *auth.Service, isProd bool) {
	ctrl := controller.NewController(repo, authService)
	if isProd || getEnv("REQUIRE_CAPTCHA", "false") == "true" {
		ctrl.SetRequireCaptcha(true)
	}
	ctrl.SetEnforcePolicy(isPolicyEnforced(isProd))
	ctrl.SetSubscriberAlwaysOn(isSubscriberAlwaysOn())
	ctrl.SetSubscriberFreeDays(parseSubscriberFreeDays(os.Getenv("SUBSCRIBER_FREE_DAYS")))
	if rawTerms := strings.TrimSpace(os.Getenv("PW_TERMS")); rawTerms != "" {
		ctrl.SetTermsHash(rawTerms)
	}
	if rawPrivacy := strings.TrimSpace(os.Getenv("PW_PRIVACY")); rawPrivacy != "" {
		ctrl.SetPrivacyHash(rawPrivacy)
	}
	handler := handlers.NewHandler(ctrl, authService)
	handler.SetReadOnly(isReadOnlyMode())
	handler.RegisterRoutes(r)
}

func setupGinEngine(repo repository.Repository, authService *auth.Service, isProd bool, expectedOriginSecret string) *gin.Engine {
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(trace.Middleware())
	r.Use(cors.Middleware())
	r.Use(originVerifyMiddleware(expectedOriginSecret, isProd))
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

func cleanupStartupMobileTokens(repo repository.Repository) {
	deleted, err := repo.DeleteExpiredMobileTokens(context.Background(), time.Now().UTC())
	if err != nil {
		log.Printf("Warning: Startup mobile token cleanup failed: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("Startup cleanup removed %d expired mobile token(s).", deleted)
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
	expectedOriginSecret := getEnv("EXPECTED_ORIGIN_SECRET", "")
	validateExpectedOriginSecret(expectedOriginSecret, isProd)
	validatePolicyConfig(os.Getenv("ENFORCE_POLICY"), os.Getenv("PW_TERMS"), os.Getenv("PW_PRIVACY"), isProd)

	authService := initJWTService(isProd)
	repo := initDatabase(isProd)
	cleanupStartupCaptchas(repo)
	cleanupStartupMobileTokens(repo)

	r := setupGinEngine(repo, authService, isProd, expectedOriginSecret)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	runServer(srv, isProd)
}
