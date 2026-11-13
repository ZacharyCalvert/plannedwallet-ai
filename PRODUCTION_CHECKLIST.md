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
   - When a shutdown signal is received (e.g. during a Kubernetes rolling update or autoscaler scale-down), the server stops accepting new connections and grants existing in-flight HTTP requests up to 30 seconds to complete cleanly before closing database handles and terminating.

3. **Mandatory High-Entropy JWT Secret**:
   - The application enforces a minimum key length of **256 bits (32 bytes)**: whenever `JWT_SECRET` is set in the environment and contains fewer than 32 bytes, the application logs a fatal error and terminates on startup (`log.Fatalf`).
   - In `PRODUCTION_ENV=true`, the application additionally performs startup validation on `JWT_SECRET`: if unset, empty, set to the default development secret (`super-secret-jwt-key-replace-in-production`, `dev-super-secret-jwt-key`), or containing the substring `"super secret"` / `"super-secret"`, the process **terminates immediately on startup** (`log.Fatalf`).

4. **Automated CAPTCHA Enforcement**:
   - When `PRODUCTION_ENV=true`, `REQUIRE_CAPTCHA` is automatically enforced as `true`, regardless of whether `REQUIRE_CAPTCHA` was explicitly set.
   - All user registration requests must include and pass a valid visual CAPTCHA verification challenge.

5. **Restricted Reverse Proxy Trust (`10.*.*.*`)**:
   - Gin is configured with `r.SetTrustedProxies([]string{"10.0.0.0/8"})`.
   - Only reverse proxies, Ingress controllers, or application load balancers residing within the internal RFC 1918 `10.0.0.0/8` private network (`10.*.*.*`) are permitted to forward client headers (`X-Forwarded-For`, `X-Real-IP`). External or untrusted proxy headers are rejected.

6. **Decoupled Database Schema Migrations**:
   - In `PRODUCTION_ENV=true`, the application **skips** executing `database/init.sql` on startup.
   - Schema creation and migrations must be performed out-of-band via a dedicated migration job before deploying new container replicas, preventing concurrent DDL execution locks during multi-replica rollouts.

7. **Azure Front Door ID (`EXPECTED_FDID`) Proxy Enforcement**:
   - When `PRODUCTION_ENV=true`, the application strictly mandates `EXPECTED_FDID`. If unset or empty, the application **terminates immediately on startup** (`log.Fatalf`).
   - Every incoming request (except OPTIONS preflight) must carry the `X-Azure-FDID` (or `X-FDID`) header matching `EXPECTED_FDID`. Direct requests bypassing Azure Front Door or originating from unauthorized proxies are immediately rejected with `403 Forbidden` (`INVALID_PROXY_HEADER`).

---

## 2. Environment Variables Reference

| Variable | Type | Default | Production Value | Description |
| :--- | :--- | :--- | :--- | :--- |
| `PRODUCTION_ENV` | Boolean | `false` | `true` | Enables production guardrails, graceful shutdown, and security checks. |
| `PORT` | Integer / String | `8080` | `8080` (or dynamic) | TCP port for the backend HTTP server. |
| `GIN_MODE` | String | `debug` | `release` | Sets Gin to release mode, disabling debug route dumps and verbose request logs. |
| `JWT_SECRET` | String | *Dev default* | `openssl rand -base64 32` | 256-bit+ random secret used for HMAC-SHA256 JWT signing. **Must be identical across all backend instances**. |
| `JWT_EXPIRATION_SECONDS`| Integer | `3600` | `900` to `3600` | Expiration lifetime of authentication tokens (15 minutes to 1 hour recommended). |
| `EXPECTED_FDID` | String | *None* | *Azure Front Door ID* | Required when `PRODUCTION_ENV=true`. Enforces that incoming requests carry `X-Azure-FDID` matching this ID. |
| `REQUIRE_CAPTCHA` | Boolean | `false` | `true` (auto in prod) | Enforces CAPTCHA verification on user registration. |
| `CORS_ALLOWED_CLIENTS` | String | `*.plannedwallet.com` | `https://app.example.com` | Comma-separated list of allowed origin URLs and hostnames. Requests from unlisted origins receive `403 Forbidden`. |
| `POSTGRES_HOST` | String | `localhost` | *Primary DB Host* | Hostname of the PostgreSQL cluster or RDS/Cloud SQL endpoint. |
| `POSTGRES_PORT` | Integer / String | `5432` | `5432` | Port of the PostgreSQL database. |
| `POSTGRES_USER` | String | `postgres` | *Dedicated DB User* | Database username (follow principle of least privilege). |
| `POSTGRES_PASSWORD` | String | `postgres` | *Secure Password* | Database password supplied via Secret Manager. |
| `POSTGRES_DB` | String | `plannedwallet` | `plannedwallet` | PostgreSQL database name. |
| `POSTGRES_SSLMODE` | String | `disable` | `require` / `verify-full` | Enforces TLS encryption for database connections in transit. |

---

## 3. Twelve-Factor Deployment Architecture at Scale

### Factor I: One Codebase, Multiple Deploys
- Maintain a single git repository. Deployments across local, staging, and production use identical container images with differing environment variable configurations.

### Factor II: Explicit Dependencies
- Backend Go dependencies are strictly vendored and locked via `go.mod` and `go.sum`.
- Frontend Node.js dependencies are locked via `package.json` and `package-lock.json`. Container builds use `npm ci` for deterministic reproduction.

### Factor III: Store Config in the Environment
- Never bake secrets, domain names, or credentials into container images.
- Inject secrets (`JWT_SECRET`, `POSTGRES_PASSWORD`) at container startup using Kubernetes Secrets, HashiCorp Vault, AWS Secrets Manager, or GCP Secret Manager.

### Factor IV: Backing Services as Attached Resources
- Treat PostgreSQL as an attached network resource.
- Use managed PostgreSQL (with automated failover, multi-AZ replication, and Point-in-Time Recovery).
- If autoscaling beyond 4–5 backend replicas, deploy a connection pooler such as **PgBouncer** or an application-aware database proxy to stay within PostgreSQL's `max_connections` limit.

### Factor V: Decouple Build, Release, Run
- Multi-stage Docker builds produce lean, minimal runtime images:
  - Backend: Alpine runtime with unprivileged non-root user.
  - Frontend: Static export compiled during build and served via lightweight Nginx.
- Execute database DDL migrations (`init.sql`) as an isolated release phase step (e.g., Kubernetes `initContainers` or CI/CD pre-deploy job) prior to starting new application instances.

### Factor VI: Stateless Processes
- Backend processes are completely stateless.
- **CAPTCHAs**: Persisted in PostgreSQL (`captchas` table) via `captcha.NewRepoStore(repo)`. Any backend replica can verify a challenge generated by any other replica.
- **Periodic CAPTCHA Cleanup**: Run a recurring job (Kubernetes `CronJob` or database pg_cron) calling `DeleteExpiredCaptchas(5 * time.Minute)` to prevent table growth.
- **Authentication**: JWTs are verified statelessly using `JWT_SECRET`. No centralized session cache is required.

### Factor VII: Port Binding
- The backend self-hosts via Go standard library `http.Server` bound to `PORT`.
- Nginx or cloud Ingress routes incoming traffic to the backend port.

### Factor VIII: Concurrency via Process Scaling
- Scale throughput horizontally by increasing replica count (Horizontal Pod Autoscaler based on CPU utilization and request volume).
- Because backend nodes share no in-memory state, instances scale up and down dynamically without user interruption.

### Factor IX: Disposability (Fast Startup & Graceful Shutdown)
- Fast startup: Backend instances initialize database pools and register handlers in under 1 second.
- Fast-fail: In `PRODUCTION_ENV=true`, invalid configuration or unresolvable databases immediately terminate the pod so orchestrators do not route user traffic to dead instances.
- Graceful shutdown: Intercepts `SIGTERM` and allows in-flight requests up to 30 seconds to finish before exiting.

### Factor X: Dev/Prod Parity
- Local development leverages Docker Compose with PostgreSQL 16, mirroring the production relational engine and table schemas.

### Factor XI: Logs as Event Streams
- Output structured logs directly to `stdout` and `stderr`.
- Requests include W3C Trace Context (`traceparent`) and `X-Client-ID`. Log shippers (Fluentbit, Promtail, Vector, or CloudWatch agent) forward log streams to central aggregation platforms for correlation.

### Factor XII: Admin Processes
- Run administrative and one-off tasks (database migrations, data backfills) as one-off jobs in identical container environments.

---

## 4. Network, Ingress & Security Hardening

### Reverse Proxy & Ingress Requirements
1. **Azure Front Door Origin Validation**:
   - Azure Front Door injects the `X-Azure-FDID` request header containing your unique Front Door profile resource ID.
   - Set `EXPECTED_FDID` to this profile ID. Backend instances reject any traffic missing this header or carrying a mismatched ID (`403 Forbidden`).
   - If utilizing Nginx as an internal gateway or sidecar, ensure `proxy_set_header X-Azure-FDID $http_x_azure_fdid;` is configured to preserve this header.
2. **HTTPS / TLS Termination**:
   - Terminate SSL/TLS on port 443 with valid certificates (e.g., Let's Encrypt / cert-manager).
   - Port 80 must return a permanent redirect (`301 Moved Permanently`) to HTTPS.
3. **Preserve Required Headers**:
   - `X-Client-ID`: Mandatory for all mutating and non-OPTIONS backend requests. Ensure edge proxies do not strip this header.
   - `X-Azure-FDID`: Mandatory in production for origin authentication.
   - `traceparent` and `tracestate`: W3C distributed tracing context headers. Must be forwarded upstream.
   - `X-Forwarded-For`, `X-Forwarded-Proto`, `Host`: Standard proxy headers.
4. **Security Response Headers**:
   - `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload` (HSTS)
   - `X-Content-Type-Options: nosniff`
   - `X-Frame-Options: DENY`
   - `Referrer-Policy: strict-origin-when-cross-origin`
   - `Content-Security-Policy: default-src 'self'; ...`
4. **Rate Limiting**:
   - Configure edge rate limiting on sensitive authentication routes:
     - `/api/v1/users/*/token` (Login)
     - `/api/v1/users` (Registration)
     - `/api/v1/captcha` (Challenge generation)
5. **NTP Clock Synchronization**:
   - Two-Factor Authentication (TOTP, RFC 6238) allows a $\pm 1$ step (30s) drift window. All host nodes and database instances must run active NTP synchronization (e.g. `chrony`) to prevent 2FA validation failures.

---

## 5. Frontend Delivery & Caching Strategy

The frontend is exported as a static client application:
- **Static Hashed Assets (`/_next/static/*`)**:
  - Configure edge and browser cache: `Cache-Control: public, max-age=31536000, immutable`.
- **HTML Entrypoints (`/index.html`, `/`)**:
  - Configure edge and browser cache: `Cache-Control: no-cache, must-revalidate` so that users receive updated application bundles immediately after deployment.
- **Edge CDN**:
  - Host static assets on an edge storage bucket with global CDN distribution (e.g., Cloud CDN, CloudFront, or Fastly), routing `/api/*` directly to the backend API cluster.

---

## 6. Pre-Flight Production Checklist

Before promoting a release to production, verify each of the following items:

- [ ] **`PRODUCTION_ENV` is set to `true`** on all backend containers.
- [ ] **`EXPECTED_FDID` is configured** with your Azure Front Door profile ID.
- [ ] **`JWT_SECRET` is set to a secure, random string** (not default) and matches across all replicas.
- [ ] **`POSTGRES_SSLMODE` is set to `require` or `verify-full`**.
- [ ] **`CORS_ALLOWED_CLIENTS` matches the exact production frontend domain(s)**.
- [ ] **Database migrations (`init.sql`) have executed** in a pre-deployment step.
- [ ] **In-memory fallback is disabled** (verified via `PRODUCTION_ENV=true`).
- [ ] **`REQUIRE_CAPTCHA` is enabled** (verified via `PRODUCTION_ENV=true`).
- [ ] **Gin trusted proxies are configured** to internal network ranges (`10.0.0.0/8`).
- [ ] **`SIGTERM` graceful shutdown is active** with connection draining.
- [ ] **NTP time synchronization is running** on all container host nodes.
- [ ] **Periodic CAPTCHA cleanup job is scheduled** to run every 5–15 minutes.
- [ ] **Rate limiting is enabled at the edge proxy** for login and registration endpoints.
- [ ] **Reverse proxy forwards `X-Client-ID` and `traceparent` headers** without dropping them.
- [ ] **Health checks (Liveness & Readiness probes)** are configured on container orchestration.
