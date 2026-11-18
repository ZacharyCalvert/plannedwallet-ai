JWT_SECRET:
Requirement: Must be a cryptographically random, high-entropy secret (e.g., 256-bit base64-encoded string: openssl rand -base64 32).

Scale Consideration: Must be injected identically into all backend instances via a dedicated secret manager (e.g., HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, or Kubernetes Secrets).
  
Code Fallback Warning: In main.go, the code falls back to "super-secret-jwt-key-replace-in-production" if unset. In production, ensure deployment manifests mandate this variable and fail fast if missing.
  
JWT_EXPIRATION_SECONDS:
Configure token TTL (default is 900 seconds / 15 minutes in auth.go). Combined with client activity-based token refresh, this limits exposure on inactive sessions.

PRODUCTION_ENV:
Set PRODUCTION_ENV=true in production. Enforces Gin release mode, PostgreSQL connectivity on startup (no MemoryRepository fallback), non-default 256-bit+ JWT_SECRET, EXPECTED_FDID header validation, REQUIRE_CAPTCHA=true, skipping startup init.sql, and policy/terms enforcement.

EXPECTED_FDID:
Azure Front Door profile ID required when PRODUCTION_ENV=true. Requests missing or mismatching X-Azure-FDID / X-FDID fail with 403 INVALID_PROXY_HEADER, and verified AFD requests allow X-Azure-ClientIP capture.

ENFORCE_POLICY & PW_TERMS:
ENFORCE_POLICY (default false) MUST be explicitly set to "true" whenever PRODUCTION_ENV=true (otherwise startup fails fatally). Requires an active policy_agreements row matching PW_TERMS (64-char SHA-256 hex of frontend-ui/public/terms-of-use.pdf) and U.S. consumer confirmation. PW_TERMS is mandatory at startup whenever policy enforcement is active.

PW_READONLY:
Set PW_READONLY=true during maintenance windows to publish the read_only feature flag and block mutating requests (503 READ_ONLY_MODE) while keeping GET endpoints, login, and token refresh active.

REQUIRE_CAPTCHA:
Set REQUIRE_CAPTCHA=true in production (auto-enabled when PRODUCTION_ENV=true, defaults to "false"). Protects user registration endpoints against automated bot account creation.
  
GIN_MODE:
Set GIN_MODE=release. This disables verbose debug logging, internal route dumping, and enables Gin's optimized production router.

PORT:
Allows orchestrators to bind the container port dynamically (defaults to 8080).

Periodic CAPTCHA Cleanup:
captcha.RunStartupCleanup() (captcha.go) runs only when a backend process starts.
In production instances running continuously for weeks or months, schedule a periodic background task (or database cron / Kubernetes CronJob) executing DeleteExpiredCaptchas(5 * time.Minute) to prevent table bloat.