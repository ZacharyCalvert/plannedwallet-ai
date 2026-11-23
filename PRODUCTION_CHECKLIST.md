# Production Readiness & Deployment Checklist

This document outlines the architecture, environment configurations, and operational checklist required to deploy the Planned Wallet application to a secure, stable, and horizontally scalable production environment in accordance with **12-Factor Application** principles.

---

## 1. The `PRODUCTION_ENV` Flag

The backend API supports a unified boolean environment variable: `PRODUCTION_ENV` (`true` / `false`). When set to `true`, the application activates strict enterprise safety guardrails and production-only behaviors:

1. **Disabled In-Memory Fallback (Fast-Fail)**:
   - In local development, if PostgreSQL is unreachable, the backend logs a warning and falls back to an in-memory repository for developer convenience.
   - In `PRODUCTION_ENV=true`, fallback is **completely disabled**. If the database is unreachable, the process immediately exits with `log.Fatalf`. This prevents split-brain state, unpersisted writes, and data divergence across container replicas.

2. **Graceful HTTP Server Shutdown**:
   - The backend runs an `http.Server` configured to capture operating system termination signals (`SIGINT`, `SIGTERM`).
   - When a shutdown signal is received (e.g. during an ECS Fargate rolling deployment or autoscaler scale-in), the server stops accepting new connections and grants existing in-flight HTTP requests up to 30 seconds to complete cleanly before closing database handles and terminating.

3. **Mandatory High-Entropy JWT Secret**:
   - The application enforces a minimum key length of **256 bits (32 bytes)**: whenever `JWT_SECRET` is set in the environment and contains fewer than 32 bytes, the application logs a fatal error and terminates on startup (`log.Fatalf`).
   - In `PRODUCTION_ENV=true`, the application additionally performs startup validation on `JWT_SECRET`: if unset, empty, set to the default development secret (`super-secret-jwt-key-replace-in-production`, `dev-super-secret-jwt-key`), or containing the substring `"super secret"` / `"super-secret"`, the process **terminates immediately on startup** (`log.Fatalf`).

4. **Automated CAPTCHA Enforcement**:
   - When `PRODUCTION_ENV=true`, `REQUIRE_CAPTCHA` is automatically enforced as `true`, regardless of whether `REQUIRE_CAPTCHA` was explicitly set.
   - All user registration requests must include and pass a valid visual CAPTCHA verification challenge.

5. **Restricted Reverse Proxy Trust (Private VPC CIDRs)**:
   - Gin is configured with `r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "::1/128"})`.
   - Only internal VPC proxies/load balancers within RFC 1918 private networks or loopback are permitted to forward `X-Forwarded-For` / `X-Real-IP` during `c.ClientIP()` fallback. When `CloudFront-Viewer-Address` is forwarded by CloudFront and `X-Origin-Verify` is verified, [`resolveClientIP`](backend-api/internal/handlers/policy.go) uses `CloudFront-Viewer-Address` directly before falling back to `c.ClientIP()`.

6. **Decoupled Database Schema Migrations**:
   - In `PRODUCTION_ENV=true`, the application **skips** executing `database/init.sql` on startup.
   - Schema creation and migrations must be performed out-of-band via a dedicated migration job before deploying new container replicas, preventing concurrent DDL execution locks during multi-replica rollouts.

7. **CloudFront Origin Secret (`EXPECTED_ORIGIN_SECRET`) Proxy Enforcement**:
   - When `PRODUCTION_ENV=true`, the application strictly mandates `EXPECTED_ORIGIN_SECRET`. If unset or empty, the application **terminates immediately on startup** (`log.Fatalf`).
   - Every incoming request (except OPTIONS preflight and `/api/health` / `/health`) must carry the `X-Origin-Verify` header matching `EXPECTED_ORIGIN_SECRET` using constant-time comparison (`crypto/subtle.ConstantTimeCompare`). Direct requests bypassing CloudFront or originating from unauthorized proxies are immediately rejected with `403 Forbidden` (`INVALID_PROXY_HEADER`).
   - Verified CloudFront requests also authorize [`resolveClientIP`](backend-api/internal/handlers/policy.go) to record the client IP from `CloudFront-Viewer-Address` (validated with `net.ParseIP`) instead of edge/proxy container IPs.

8. **Policy Ledger & Document Checksum Enforcement (`ENFORCE_POLICY` / `PW_TERMS` / `PW_PRIVACY`)**:
   - When `PRODUCTION_ENV=true`, **`ENFORCE_POLICY` MUST be explicitly set to `"true"`**, **`PW_TERMS`** MUST be configured with the exact 64-character SHA-256 hex digest of the live `frontend-ui/public/terms-of-use.pdf` asset, and **`PW_PRIVACY`** MUST be configured with the exact 64-character SHA-256 hex digest of the live `frontend-ui/public/privacy-and-security-policy.pdf` asset; otherwise the process **terminates immediately on startup** (`log.Fatalf`).
   - Signed-in users without an active `policy_agreements` row matching `PW_TERMS`, `PW_PRIVACY`, and 18+ U.S. adult consumer confirmation (`us_adult_consumer_confirmed`) receive a JWT with `terms_approved: false` and receive `403 POLICY_APPROVAL_REQUIRED` on data routes until they submit `POST /api/v1/users/{username}/policy-agreements` and refresh their JWT.

---

## 2. Environment Variables Reference

| Variable | Type | Default | Production Value | Description |
| :--- | :--- | :--- | :--- | :--- |
| `PRODUCTION_ENV` | Boolean | `false` | `true` | Enables production guardrails, graceful shutdown, and security checks. |
| `PORT` | Integer / String | `8080` | `8080` (or dynamic) | TCP port for the backend HTTP server. |
| `GIN_MODE` | String | `debug` | `release` | Sets Gin to release mode, disabling debug route dumps and verbose request logs. |
| `JWT_SECRET` | String | *Dev default* | `openssl rand -base64 32` | 256-bit+ random secret used for HMAC-SHA256 JWT signing. **Must be identical across all backend instances**. |
| `JWT_EXPIRATION_SECONDS`| Integer | `900` | `900` | Expiration lifetime of authentication tokens in seconds (15 minutes default). |
| `EXPECTED_ORIGIN_SECRET` | String | *None* | *48-char CloudFront Origin Secret* | Required when `PRODUCTION_ENV=true`. Enforces constant-time comparison against `X-Origin-Verify` and authorizes `CloudFront-Viewer-Address` capture. |
| `REQUIRE_CAPTCHA` | Boolean | `false` | `true` (auto in prod) | Enforces CAPTCHA verification on user registration. |
| `ENFORCE_POLICY` | Boolean | `false` | `true` (required in prod) | **Must be `"true"` when `PRODUCTION_ENV=true`**. Enforces active `policy_agreements` approval matching `PW_TERMS`, `PW_PRIVACY`, and 18+ U.S. adult consumer confirmation before allowing data access. |
| `PW_TERMS` | String | *None* | *64-char SHA-256 hex* | Required when `PRODUCTION_ENV=true` or `ENFORCE_POLICY=true`. Exact SHA-256 hex checksum of `frontend-ui/public/terms-of-use.pdf`. |
| `PW_PRIVACY` | String | *None* | *64-char SHA-256 hex* | Required when `PRODUCTION_ENV=true` or `ENFORCE_POLICY=true`. Exact SHA-256 hex checksum of `frontend-ui/public/privacy-and-security-policy.pdf`. |
| `PW_READONLY` | Boolean | `false` | `false` (`true` during maintenance) | Enables read-only maintenance mode (`read_only` feature flag and `503 READ_ONLY_MODE` on mutations while keeping login and token refresh working). |
| `SUBSCRIBER_ALWAYS_ON` | String (`"true"`) | `false` | `false` | When explicitly set to `"true"`, forces the `subscriber` claim in issued/refreshed JWTs to `true`; any other value defaults to `false`. |
| `SUBSCRIBER_FREE_DAYS` | Integer | `0` | `0` (or trial days) | When set to a positive integer, grants first-time registered users a `subscriptions` record for that many days (`paid_date` = account creation date). |
| `CORS_ALLOWED_CLIENTS` | String | `*.plannedwallet.com` | `https://app.example.com` | Comma-separated list of allowed origin URLs and hostnames. Requests from unlisted origins receive `403 Forbidden`. |
| `POSTGRES_HOST` | String | `localhost` | *RDS Primary Endpoint* | Hostname of the Amazon RDS PostgreSQL 16 instance (`DB_HOST` alias also supported). |
| `POSTGRES_PORT` | Integer / String | `5432` | `5432` | Port of the PostgreSQL database (`DB_PORT` alias also supported). |
| `POSTGRES_USER` | String | `postgres` | `pwadmin` | Database username (`DB_USER` alias also supported). |
| `POSTGRES_PASSWORD` | String | `postgres` | *SSM Parameter Store Secret* | Database password injected into ECS tasks from SSM Parameter Store (`DB_PASSWORD` alias also supported). |
| `POSTGRES_DB` | String | `plannedwallet` | `plannedwallet` | PostgreSQL database name (`DB_NAME` alias also supported). |
| `POSTGRES_SSLMODE` | String | `disable` (`require` in prod) | `require` / `verify-full` | Enforces TLS encryption for Amazon RDS (`rds.force_ssl = 1`; defaults to `require` when `PRODUCTION_ENV=true`). |

---

## 3. Twelve-Factor Deployment Architecture at Scale

### Factor I: One Codebase, Multiple Deploys
- Maintain a single git repository. Deployments across local, staging, and production use identical container images with differing environment variable configurations.

### Factor II: Explicit Dependencies
- Backend Go dependencies are strictly vendored and locked via `go.mod` and `go.sum`.
- Frontend Node.js dependencies are locked via `package.json` and `package-lock.json`. Container builds use `npm ci` for deterministic reproduction.

### Factor III: Store Config in the Environment
- Never bake secrets, domain names, or credentials into container images.
- Inject secrets (`JWT_SECRET`, `EXPECTED_ORIGIN_SECRET`, `POSTGRES_PASSWORD`) into ECS Fargate task definitions using **AWS Systems Manager (SSM) Parameter Store** or **AWS Secrets Manager**.

### Factor IV: Backing Services as Attached Resources
- Treat PostgreSQL as an attached network resource.
- Use managed PostgreSQL (**Amazon RDS PostgreSQL 16** with automated backups, Multi-AZ failover, and `rds.force_ssl = 1`).
- If autoscaling beyond 4–5 backend tasks, deploy **Amazon RDS Proxy** or **PgBouncer** to stay within PostgreSQL's `max_connections` limit.

### Factor V: Decouple Build, Release, Run
- Multi-stage Docker builds produce lean, minimal runtime images:
  - Backend: Scratch runtime compiled statically from Go (`linux/amd64`) and pushed with immutable Git SHA tags to **Amazon ECR**.
  - Frontend: Next.js static export (`out/`) synced directly to a private **Amazon S3** bucket behind **CloudFront OAC**.
- Execute database DDL migrations (`init.sql`) as an isolated pre-deploy step via an ephemeral **ECS Fargate `RunTask`** (`ECS_MIGRATION_TASK_DEFINITION`) prior to deploying new ECS backend tasks.

### Factor VI: Stateless Processes
- Backend processes are completely stateless.
- **CAPTCHAs**: Persisted in PostgreSQL (`captchas` table) via `captcha.NewRepoStore(repo)`. Any backend replica can verify a challenge generated by any other replica.
- **Periodic CAPTCHA Cleanup**: Run a recurring job (e.g., EventBridge-scheduled ECS task or `pg_cron`) calling `DeleteExpiredCaptchas(5 * time.Minute)` to prevent table growth.
- **Authentication**: JWTs are verified statelessly using `JWT_SECRET`. No centralized session cache is required.

### Factor VII: Port Binding
- The backend self-hosts via Go standard library `http.Server` bound to `PORT` (`8080`).
- **API Gateway HTTP API** routes `/api/*` requests over a **Private VPC Link** to the ECS Fargate service (or Nginx in local Docker Compose).

### Factor VIII: Concurrency via Process Scaling
- Scale throughput horizontally by increasing the ECS Fargate service desired count and Application Auto Scaling targets (CPU/request volume).
- Because backend tasks share no in-memory state, tasks scale out and in dynamically without user interruption.

### Factor IX: Disposability (Fast Startup & Graceful Shutdown)
- Fast startup: Backend instances initialize database pools and register handlers in under 1 second.
- Fast-fail: In `PRODUCTION_ENV=true`, invalid configuration (`ENFORCE_POLICY != "true"`, missing `EXPECTED_ORIGIN_SECRET` or `PW_TERMS`, insecure `JWT_SECRET`) or unresolvable databases immediately terminate the task so ECS/ALB health checks do not route traffic to unhealthy tasks.
- Graceful shutdown: Intercepts `SIGTERM` and allows in-flight requests up to 30 seconds to finish before exiting.

### Factor X: Dev/Prod Parity
- Local development leverages Docker Compose with PostgreSQL 16, mirroring the production RDS PostgreSQL engine and table schemas.

### Factor XI: Logs as Event Streams
- Output structured logs directly to `stdout` and `stderr`.
- Requests include W3C Trace Context (`traceparent`) and `X-Client-ID`. ECS `awslogs` streams container output directly to **Amazon CloudWatch Logs** for correlation.

### Factor XII: Admin Processes
- Run administrative and one-off tasks (database migrations, data backfills) as one-off jobs in identical container environments.

---

## 4. Network, Ingress & Security Hardening

### Reverse Proxy & Ingress Requirements
1. **CloudFront Origin Validation**:
   - AWS CloudFront injects the `X-Origin-Verify` request header containing the KMS-encrypted 48-character origin secret.
   - Inject `EXPECTED_ORIGIN_SECRET` into the ECS Fargate `backend` task definition from SSM Parameter Store. Backend instances reject any traffic missing this header or failing constant-time comparison (`403 Forbidden`), while exempting `/api/health` for internal ECS/NLB/ALB target health checks.
   - If utilizing Nginx as an internal gateway or sidecar, ensure `proxy_set_header X-Origin-Verify $http_x_origin_verify;` is configured to preserve this header.
2. **HTTPS / TLS Termination**:
   - Terminate SSL/TLS on port 443 with valid certificates (e.g., AWS ACM / Let's Encrypt).
   - Port 80 must return a permanent redirect (`301 Moved Permanently`) to HTTPS.
3. **Preserve Required Headers**:
   - `X-Client-ID`: Mandatory for all mutating and non-OPTIONS backend requests. Ensure edge proxies do not strip this header.
   - `X-Origin-Verify`: Mandatory in production for CloudFront origin authentication.
   - `CloudFront-Viewer-Address`: Injected by CloudFront and preserved for client IP recording in `policy_agreements` whenever `X-Origin-Verify` is verified.
   - `traceparent` and `tracestate`: W3C distributed tracing context headers. Must be forwarded upstream.
   - `X-Forwarded-For`, `X-Forwarded-Proto`, `Host`: Standard proxy headers.
4. **Security Response Headers**:
   - `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload` (HSTS)
   - `X-Content-Type-Options: nosniff`
   - `X-Frame-Options: DENY`
   - `Referrer-Policy: strict-origin-when-cross-origin`
   - `Content-Security-Policy: default-src 'self'; ...`
5. **Rate Limiting**:
   - Configure AWS WAF v2 edge rate limiting on sensitive authentication routes:
     - `/api/v1/users/*/token` (Login)
     - `/api/v1/users/*/recover` (Account Recovery)
     - `/api/v1/users` (Registration)
     - `/api/v1/captcha` (Challenge generation)
6. **NTP Clock Synchronization**:
   - Two-Factor Authentication (TOTP, RFC 6238) allows a $\pm 1$ step (30s) drift window. Managed AWS Fargate/RDS infrastructure maintains synchronized time (`Amazon Time Sync Service`) to prevent 2FA validation failures.

---

## 5. Frontend Delivery & Caching Strategy

The frontend is exported as a static client application and hosted in a private **Amazon S3** bucket (`s3-pw-${env}-web-<suffix>`) behind **CloudFront OAC + AWS WAF v2**:
- **Static Hashed Assets (`/_next/static/*`, JS, CSS, images)**:
  - Uploaded with `Cache-Control: public, max-age=31536000, immutable`.
- **HTML Entrypoints (`/index.html`, `/404.html`, `*.html`)**:
  - Uploaded with `Cache-Control: public, max-age=0, must-revalidate` and invalidated via CloudFront (`/*`) so users receive updated application bundles immediately after deployment.
- **API Routing**:
  - CloudFront routes `/api/*` to **API Gateway HTTP API**, which forwards traffic via a **Private VPC Link** to the ECS Fargate backend service while injecting `X-Origin-Verify` and `CloudFront-Viewer-Address`.

---

## 6. Pre-Flight Production Checklist

Before promoting a release to production, verify each of the following items:

- [ ] **`PRODUCTION_ENV` is set to `true`** and **`ENFORCE_POLICY` is set to `"true"`** on all backend ECS task definitions.
- [ ] **`EXPECTED_ORIGIN_SECRET` is configured** from SSM Parameter Store matching the CloudFront `X-Origin-Verify` header.
- [ ] **`PW_TERMS` matches the SHA-256 hex digest** of the deployed `frontend-ui/public/terms-of-use.pdf`.
- [ ] **`JWT_SECRET` is set to a secure, random 256-bit+ string** (not default) and injected identically across all ECS tasks.
- [ ] **`POSTGRES_SSLMODE` is set to `require` or `verify-full`** (RDS `rds.force_ssl = 1`).
- [ ] **`CORS_ALLOWED_CLIENTS` / `CORS_ALLOWED_ORIGINS` matches the exact production frontend domain(s)**.
- [ ] **Database migrations (`init.sql`) have executed** via ephemeral ECS Fargate `RunTask` (`ECS_MIGRATION_TASK_DEFINITION`) before backend deploy.
- [ ] **In-memory fallback is disabled** (verified via `PRODUCTION_ENV=true`).
- [ ] **`REQUIRE_CAPTCHA` is enabled** (verified via `PRODUCTION_ENV=true`).
- [ ] **Gin trusted proxies are configured** to VPC internal ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) and CloudFront's Origin Request Policy forwards `CloudFront-Viewer-Address`.
- [ ] **`SIGTERM` graceful shutdown is active** with ECS task draining.
- [ ] **Periodic CAPTCHA cleanup job is scheduled** to run every 5–15 minutes.
- [ ] **AWS WAF v2 rate limiting is enabled at CloudFront** for login, recovery, registration, and CAPTCHA endpoints.
- [ ] **API Gateway + CloudFront forward `X-Client-ID`, `X-Origin-Verify`, `CloudFront-Viewer-Address`, and `traceparent` headers** without dropping them.
- [ ] **ECS health checks (`/api/health`) and deploy gate (`/api/v1/version`)** are active.
