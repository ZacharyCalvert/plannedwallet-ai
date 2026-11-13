JWT_SECRET:
Requirement: Must be a cryptographically random, high-entropy secret (e.g., 256-bit base64-encoded string: openssl rand -base64 32).

Scale Consideration: Must be injected identically into all backend instances via a dedicated secret manager (e.g., HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, or Kubernetes Secrets).
  
Code Fallback Warning: In main.go, the code falls back to "super-secret-jwt-key-replace-in-production" if unset. In production, ensure deployment manifests mandate this variable and fail fast if missing.
  
JWT_EXPIRATION_SECONDS:
Configure token TTL (default is 3600 seconds / 1 hour in main.go). For sensitive financial applications, shorter-lived tokens (e.g., 900–1800 seconds) combined with client re-authentication minimize the exposure window.
  
REQUIRE_CAPTCHA:
Set REQUIRE_CAPTCHA=true in production (defaults to "false" in main.go). This protects user registration endpoints against automated credential stuffing and bot account creation.
  
GIN_MODE:
Set GIN_MODE=release. This disables verbose debug logging, internal route dumping, and enables Gin's optimized production router.

PORT:
Allows orchestrators to bind the container port dynamically (defaults to 8080).

Periodic CAPTCHA Cleanup:
captcha.RunStartupCleanup() (captcha.go) runs only when a backend process starts.
In production instances running continuously for weeks or months, schedule a periodic background task (or database cron / Kubernetes CronJob) executing DeleteExpiredCaptchas(5 * time.Minute) to prevent table bloat.