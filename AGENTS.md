# AGENTS.md - Repository Rules & Instructions

## Core Principles & Coding Standards
- **Back-End Development**: Written in Go (Golang) using the Gin web framework and `pq` / `pgx` driver for PostgreSQL.
- **Front-End Development**: Written in React using Next.js.
- **Database**: PostgreSQL for persistence, accompanied by a full in-memory repository implementation (`MemoryRepository`) for unit testing without external database dependencies.
- **Function Constraints**:
  - Maximum function length: **40 lines**.
  - Maximum cyclomatic complexity per function: **6**.
- **Observability**: User operations and API interactions log using W3C Trace Context (`traceparent` header). All HTTP requests require the `X-Client-ID` header.
- **Integration Tests**: Integration tests in `itests` execute against a backend with a completely fresh (wiped) database every time (`make refresh` or removing `./data` before test runs).
- **No Registered Business / Brand Names**: Avoid adding registered business, financial institution, or trademarked company names in the codebase. This applies to UI placeholders, input labels, descriptions, sample data, mocks, seed fixtures, unit tests, integration tests, OpenAPI contracts, and documentation. Apply preference to generic, descriptive, or fictional names (e.g., "Primary Everyday Checking", "Emergency Reserve Savings", "Day Trading Portfolio", "Brokerage Snapshot", "Supermarket", "Corner Grocery", "Employer Payroll").

---

## 1. Project Stack & Architecture
- **Tech Stack**:
  - Front-End: Next.js / React, Tailwind CSS.
  - Back-End: Golang, Gin router, standard library `crypto` / `bcrypt` for security.
  - Database: PostgreSQL 16 (dockerized).
  - Reverse Proxy: Nginx gateway.
  - Integration Tests: Python, Pytest, Playwright API client.
- **Containerization**: All services deploy within `docker-compose.yml` for local development, end-to-end testing, and CI/CD pipelines.

---

## 2. Folder & File Structure
```
├── backend-api/            # Golang backend microservice
│   ├── internal/
│   │   ├── auth/          # JWT issuing, TOTP (RFC 6238) generation and validation
│   │   ├── controller/    # Business logic, access control, validation (MVC controller)
│   │   ├── cors/          # CORS origin and host validation middleware
│   │   ├── database/      # SQL DDL migrations (init.sql) and connection pooling
│   │   ├── handlers/      # Gin HTTP request/response handlers (MVC view/handler)
│   │   ├── models/        # Domain entities, request/response DTOs, enums
│   │   ├── repository/    # Data access: Repository interface, PostgresRepo, MemoryRepo
│   │   └── trace/         # W3C trace context middleware and structured logging
│   ├── schema.yml         # OpenAPI 3.2.0 API contract definition
│   ├── schema.md          # Database schema and API specification documentation
│   ├── Dockerfile         # Multi-stage production container build
│   └── Makefile           # Build, test, and run commands for backend
├── frontend-ui/           # Next.js web application
│   ├── src/               # React components, pages, hooks, state management
│   ├── Dockerfile         # Container build for web UI
│   └── Makefile           # Build, test, and start commands for frontend
├── nginx-gateway/         # Nginx reverse proxy configuration and container
├── itests/                # Integration test suite (Playwright + Pytest)
│   ├── conftest.py        # Shared test fixtures, auth helpers, TOTP generator
│   ├── test_*.py          # Functional test suites (users, checking, assets, cors, trace)
│   └── Makefile           # Virtualenv setup and test runner
├── docker-compose.yml     # Local orchestration for postgres, backend, frontend, gateway
├── Makefile               # Top-level targets: start, stop, clean, refresh, test
├── schema.md              # Root copy of database and API specification
└── AGENTS.md              # Repository rules and guidance for AI agents
```

---

## 3. Account Types & Behavior
The backend classifies financial accounts under the `type` column in the `accounts` table:

1. **`checking`**:
   - Transactional account with checking records (`value`, `date_sent`, `date_cleared`, `entity`, `description`).
   - Balances calculated **bottom-up**: The oldest ("first") record matches its own transaction value; the newest record at the top reflects the cumulative balance of all cleared and pending transactions.
2. **`savings`**:
   - Identical transaction and balance calculation behavior to `checking`.
   - Serves as a distinct category/label in the UI and data layer.
3. **`credit card`**:
   - Identical transaction and balance calculation behavior to `checking`/`savings`.
   - Displayed with a red/pink visual indicator in the UI accounts list.
4. **`asset`** (or `assets`):
   - Represents a collection or portfolio of physical or digital assets (e.g. "Precious Metals" holding gold, silver, copper; crypto holding BTC/ETH; collectible items).
   - Contains 1 or more distinct asset holdings tracked in the `account_assets` table with positive quantities (`quantity > 0`).
   - Account worth is dynamically computed as:
     $$\text{Account Total Value} = \sum_{i} \left( \text{quantity}_i \times \text{latest\_value}_i \right)$$
     where `latest_value` is resolved from the most recent valuation in `asset_values` (`value_date DESC, created_at DESC LIMIT 1`).
5. **`snapshot`**:
   - Dedicated account type for point-in-time account valuations over time (e.g. trading accounts, net worth tracking).
   - Valuations are manually assigned per date via `account_snapshots` (`value`, `snapshot_date`).
   - Trends account worth over time (`snapshot_date ASC, created_at ASC`) without holding checking records or asset holdings.
   - Non-snapshot accounts reject snapshot operations with `400 Bad Request` (`INVALID_ACCOUNT_TYPE`).
   - Mutating snapshots requires `admin` or `edit` access; `read` access permits viewing/trending.

---

## 4. Multi-User Access Control (`account_access`)
All account endpoints enforce permissions via the `account_access` table:
- **Owner / Admin (`accounts.admin_id == user_id` or `access_level == "admin"`)**: Full permissions to view, mutate records/assets/snapshots, and grant or revoke access.
- **Editor (`access_level == "edit"`)**: Permitted to view and mutate account records, asset holdings, and snapshots (`POST`, `PUT`, `DELETE`). Denied from altering account access settings (`403 Forbidden`).
- **Reader (`access_level == "read"`)**: Permitted read-only access (`GET` records, assets, snapshots, summaries). Any mutation attempt returns `403 Forbidden`.
- **Unauthorized (no access entry)**: Returns `404 Not Found` (`ACCOUNT_NOT_FOUND`) to prevent enumeration or leaking the existence of private accounts.

---

## 5. Assets & Valuations Architecture
- **`assets` Table**: Stores asset definitions (`id`, `user_id`, `label`, `description`).
  - `user_id` can be `NULL` for globally available/system assets.
  - Unique constraint enforces unique labels per user or globally: `UNIQUE NULLS NOT DISTINCT (user_id, label)` and case-insensitively via `uq_assets_user_lower_label (user_id, LOWER(label))`.
- **`asset_values` Table**: Stores date-keyed unit prices (`id`, `asset_id`, `value`, `value_date`).
  - Enforces one valuation per asset per date: `UNIQUE (asset_id, value_date)`.
- **`account_assets` Table**: Maps assets into asset accounts (`id`, `account_id`, `asset_id`, `quantity`).
  - Quantity must be strictly positive (`CHECK (quantity > 0)`).
  - Unique constraint prevents duplicate asset entries within the same account: `UNIQUE (account_id, asset_id)`.
- **`account_snapshots` Table**: Stores point-in-time valuations for snapshot accounts (`id`, `account_id`, `value`, `snapshot_date`, `created_at`, `updated_at`).
  - Enforces one valuation per snapshot account per date: `CONSTRAINT uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)`.
  - Ordered chronologically by `snapshot_date ASC, created_at ASC` to provide time-series trending over time.

---

## 6. Guidelines for AI Agents & Developers
1. **Maintain In-Memory & Postgres Parity**: Every repository method implemented in `postgres.go` must also be implemented in `memory.go` with identical semantics and error types (`ErrNotFound`, `ErrConflict`), ensuring unit tests run fast and independently.
2. **Strict Function Size & Complexity Limits**:
   - Every function must remain $\le 40$ lines.
   - Cyclomatic complexity must remain $\le 6$.
   - Split orchestration, validation, and mapping into separate helper functions when needed.
3. **Traceability & Correlation**:
   - Log all significant lifecycle events, errors, and mutations with `trace.LogInfo` / `trace.LogError`.
   - Never remove or bypass `trace.Middleware()` or the required `X-Client-ID` header.
4. **Contract Synchronization**:
   - Whenever adding or updating database tables, endpoints, or payloads, update `schema.yml` (OpenAPI) and `schema.md` simultaneously.
5. **UI Modification Policy**:
   - Do NOT modify front-end files unless the user explicitly requests UI changes.
6. **Integration Test Hygiene**:
   - Every test run must be reproducible against a clean database. Use `make refresh` (which executes `docker compose down`, `rm -rf ./data`, `mkdir -p data`, `docker compose up -d --build`) to wipe the database volume before verifying integration suites.
7. **No Trademarked or Registered Business Names**:
   - Do not add additional real business or brand names into source code, comments, test assertions, seed scripts, OpenAPI examples, or UI placeholders/memos.
   - Use generic descriptors instead (e.g., "Brokerage Account", "Trading Portfolio", "Retail Bank", "Payroll Deposit", "Coffee Shop", "Supermarket").


