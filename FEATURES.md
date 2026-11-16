# Statement of Work: Feature Inventory & System Capabilities

**Project:** Planned Wallet Personal Wealth & Financial Operations Platform  
**Target Audience:** Enterprise Clients, Stakeholders, Technical Leadership & Auditors  
**Document Version:** 1.0.0  
**Classification:** Technical Statement of Work (SOW) & Capabilities Matrix  

---

## 1. Executive Summary

Planned Wallet is an enterprise-grade, high-performance financial intelligence and account orchestration platform designed to track, trend, analyze, and forecast personal and entity-level net worth across diverse asset classes and debt instruments. Built upon a 12-factor cloud-native architecture, the platform pairs an ultra-low latency Golang backend with a reactive Next.js / TypeScript presentation layer, mediated by an Nginx reverse-proxy gateway and orchestrated via Docker and automated CI/CD pipelines.

The platform provides unified ledger management across transactional accounts (checking, savings, credit cards), multi-location physical and digital asset portfolios (precious metals, cryptocurrencies, collectibles), and point-in-time snapshot valuations (brokerage portfolios, retirement funds). Advanced analytical engines project runway trajectories under loss-of-income scenarios, downsample dense time-series metrics, and eliminate cross-account transfer artifacts to deliver financial observability.

---

## 2. Platform Architecture & Deliverable Artifacts

The engagement deliverable encompasses a complete, self-contained multi-tier codebase and production runtime infrastructure:

```
├── backend-api/            # Golang 1.22+ REST microservice (Gin framework)
│   ├── internal/
│   │   ├── auth/          # JWT issuing, RFC 6238 TOTP engine, BCrypt hashing
│   │   ├── captcha/       # Procedural visual CAPTCHA engine & memory store
│   │   ├── controller/    # Core business logic, RBAC, financial math, validations
│   │   ├── cors/          # Origin and host validation middleware
│   │   ├── database/      # SQL DDL schemas (init.sql) & connection pool management
│   │   ├── handlers/      # Gin HTTP request/response routing & serialisation
│   │   ├── models/        # Domain entities, DTOs, enums, JSON specifications
│   │   ├── repository/    # Dual-store data access (PostgreSQL & In-Memory parity)
│   │   └── trace/         # W3C Trace Context engine and structured audit logging
│   ├── schema.yml         # OpenAPI 3.2.0 formal contract specification
│   ├── schema.md          # Database schema and API specification documentation
│   ├── Dockerfile         # Multi-stage production container build (scratch/distroless)
│   └── Makefile           # Backend test, build, and audit runners
├── frontend-ui/           # Next.js / React 19 / TypeScript Web Application
│   ├── app/               # Next.js App Router (layout, pages, modular CSS stylesheets)
│   │   └── styles/        # Dedicated domain stylesheets (variables, auth, accounts, etc.)
│   ├── components/        # Isolated React UI components (charts, modals, ledgers)
│   ├── lib/               # Typed API client, W3C trace propagation, CSV parser, math
│   ├── Dockerfile         # Multi-stage Node/Nginx container build
│   └── Makefile           # Frontend test, lint, and bundle targets
├── nginx-gateway/         # Reverse proxy gateway & ingress boundary
│   ├── default.conf       # Route proxying (/api/v1 -> backend, / -> frontend)
│   └── Dockerfile         # Lightweight Alpine-based reverse proxy container
├── itests/                # End-to-End Integration Test Suite (Python, Pytest, Playwright)
│   ├── conftest.py        # Fresh database fixtures, auth generators, TOTP drivers
│   ├── test_*.py          # 35 comprehensive automated test suites
│   └── Makefile           # Isolated virtual environment test runner
├── .github/workflows/     # CI/CD Deployment & Database Migration Pipelines
│   ├── apply-sql.yml      # Concurrency-locked idempotent database migration runner
│   ├── backend-api.yml    # Concurrency-locked backend testing, container build & push
│   └── frontend-ui.yml    # Concurrency-locked frontend verification & deployment
├── docker-compose.yml     # Multi-container orchestration (Postgres 16, API, UI, Gateway)
├── Makefile               # Top-level orchestration (start, stop, clean, refresh, test)
├── schema.md              # Root database architecture and API documentation
├── SQL_CONFIG.md          # Production database provisioning & connection sizing guide
├── PRODREADY.md           # Production hardening, secrets, and environment directives
└── PRODUCTION_CHECKLIST.md# Operational readiness & deployment verification guide
```

---

## 3. Comprehensive Feature Breakdown

### 3.1 Multi-Account Financial Architecture
The system natively categorizes and manages financial assets across five distinct account models, enforcing type-specific invariant validation and calculation semantics:

1. **Checking Accounts (`checking`)**:
   - Transactional ledgers capturing income and expenditures.
   - **Bottom-Up Balance Calculation**: Balances calculate chronologically from the oldest transaction matching its original value, through to the newest record reflecting the cumulative running balance of cleared and pending transactions.
   - Distinctive cyan visual theme (`--checking: #0284c7`).
2. **Savings Accounts (`savings`)**:
   - Capital preservation reserves utilizing identical bottom-up balance semantics to checking accounts.
   - Distinctive emerald visual theme (`--savings: #059669`).
3. **Credit Card Accounts (`credit card`)**:
   - Revolving credit and debt tracking ledgers.
   - Directionality semantics: Outgoing amounts (negative values) reflect expenses/spend; incoming amounts (positive values) reflect statement payments.
   - Dedicated violet UI styling (`--credit-card: #8b5cf6`, light violet background, violet border) maintaining a non-stressful yet distinct visual hierarchy from deposit accounts.
4. **Physical & Digital Asset Portfolios (`asset`)**:
   - Aggregated holdings across physical commodities (precious metals), cryptocurrencies, and collectibles.
   - Separates asset class catalog definitions from specific storage locations and balance records.
   - Dynamic valuation model aggregating total account worth based on current holdings multiplied by latest unit valuations.
   - Distinctive teal visual theme (`--asset: #0d9488`).
5. **Snapshot Trend Accounts (`snapshot`)**:
   - Point-in-time valuation tracking for external investment, trading, and retirement portfolios.
   - Pure chronological valuation curve without individual transaction ledger overhead.
   - Strict rejection of transaction records or asset holdings on snapshot accounts (`400 INVALID_ACCOUNT_TYPE`).
   - Distinctive purple visual theme (`--snapshot: #7c3aed`).

---

### 3.2 Transaction Ledger & Ingestion Engine
- **Record Entry & Inline Management**:
  - Full CRUD operations supporting transaction value, sent date (`date_sent`), cleared date (`date_cleared`), entity / merchant, and memo description.
  - **Single-Click Clearance**: Rapid "Mark Cleared" workflow setting the cleared timestamp to today's date.
  - **Dynamic Sorting Invariants**:
    1. Uncleared / pending records prioritize at the top of the ledger.
    2. Uncleared records sort by sent date ascending (oldest first) to surface overdue transactions.
    3. Cleared records sort by clear date descending (newest first).
- **Spreadsheet-Grade Currency Formatting**:
  - Automatic normalization and parsing of currency strings containing `$`, commas, and whitespace.
  - **Parenthesis Accounting Notation**: Robust detection of accounting parenthesis formatting where `$(105.10)` or `(105.10)` seamlessly converts to `-105.10`, while `$105.10` parses as positive `105.10`.
- **High-Capacity Bulk CSV Import**:
  - Client-side streaming parser and modal workflow supporting batches up to 10,000 records per upload.
  - Multi-column auto-mapping for transaction amounts, dates, entities, descriptions, and clearance statuses.
  - Rigorous data validation: Rejection of financial records prior to the year 2000, rejection of 3-digit years, and mandatory 4-digit ISO-compatible date handling.

---

### 3.3 Multi-Storage Asset Tracking & Valuation Matrix
- **Asset Catalog Classes (`assets`)**:
  - Global and user-scoped catalog entries defining tradable or collectible asset types (e.g., Gold Ounces, Cryptographic Coins).
  - Enforced uniqueness constraints ensuring zero duplicate labels per user or system-wide (`UNIQUE (user_id, LOWER(label))`).
- **Date-Keyed Unit Valuations (`asset_values`)**:
  - Historical pricing repository capturing date-keyed unit values.
  - Enforces one definitive valuation per asset per calendar date.
- **Multi-Location Storage Mapping (`asset_stores`)**:
  - Enables partitioning asset accounts into distinct holding locations, cold-storage keys, depository vaults, or physical containers.
  - Unique storage label enforcement per account.
- **Date-Keyed Quantity Records (`asset_store_records`)**:
  - Point-in-time holding quantities tracked by date per storage location.
  - Strict positive quantity constraints (`quantity > 0`).
- **Real-Time Dynamic Valuation Engine**:
  - Automatically calculates account valuation by resolving the latest unit price per holding:
    $$\text{Account Worth} = \sum_{i} \left( \text{quantity}_i \times \text{latest\_value}_i \right)$$
  - Interactive multi-tab detail view: Asset Classes, Valuation History, Storage Locations, Quantity Records, and Time-Series Trend Charts.

---

### 3.4 Snapshot Accounts & Chronological Valuation Curves
- Point-in-time valuation records (`value`, `snapshot_date`) with date-uniqueness constraints per account (`uq_account_snapshots_account_date`).
- **Automated Differential Tracking**: Computes valuation variance ($\pm \Delta$) against the immediately preceding chronological entry.
- **Interactive SVG Trend Visualizer**: Embedded responsive vector charting displaying valuation curves, data points, and date-range tooltips.

---

### 3.5 Perspectives & Aggregated Financial Intelligence
Perspectives provide unified multi-account lenses aggregating disparate accounts into combined trend and runway projections:
- **Multi-Account Aggregation**: Combines transaction histories and valuation snapshots across arbitrary combinations of checking, savings, credit cards, assets, and snapshot accounts.
- **Transfer Spend Deduplication (35-Day Credit Card Heuristic)**:
  - Eliminates artificial inflation in spending calculations caused by debt payoffs.
  - When a debit (spend) in a deposit account (checking/savings) identically matches a credit (payment) in a credit card account within a 35-day window, the algorithm classifies it as an internal transfer and purges it from aggregate expense metrics.
- **Adaptive Graph Downsampling**:
  - Enforces high visual clarity on complex time series: Downsamples data to guarantee no more than 1 data point per 10 pixels along the horizontal X-axis.
  - Computes windowed arithmetic averages during downsampled buckets to eliminate graph noise while preserving trend fidelity.
- **Loss of Income Runway & "Ruined Date" Forecasting**:
  - Projects capital depletion trajectories based on current liquid reserves against trailing average daily spend across 30-day, 90-day, and 365-day moving windows.
  - Supports custom daily spend override simulations (e.g., modeling austerity budgets during career transitions).
  - Graphically renders the trajectory toward financial exhaustion with collision-aware popup tooltips.
- **Dynamic View Configuration**: Configurable toggles for financial trends, return-on-investment YoY analytics, daily spend metrics, and runway modeling.

---

### 3.6 Account Deletion Lifecycle & Re-Creation Guarantee
- **Strict Creator-Only Authorization**:
  - Account deletion is exclusively restricted to the account's original creator (`accounts.admin_id == user_id`).
  - Read-only, editor, or even granted admin accounts are prohibited from executing account deletion, yielding an immediate `403 Forbidden` (`FORBIDDEN`).
- **Full Cascade Cleanup**:
  - Atomic database cascades permanently eliminate orphan data across `account_access`, `checking_records`, `asset_stores`, `asset_store_records`, `account_assets`, `account_snapshots`, and `perspective_accounts`.
  - In-memory repository mirrors cascade semantics identically.
- **Explicit UI Warning Safeguards**:
  - Dedicated `DeleteAccountModal` dialog warning users with an unambiguous advisory:  
    `⚠️ Warning: This action is permanent and cannot be undone!`
  - Explicit confirmation requiring affirmative user intent before triggering destructive network calls.
- **Collision-Free Re-Creation**:
  - Deleting an account immediately releases all naming and resource locks, allowing the user to immediately re-create an account with the identical name and type without collision.

---

## 4. Security Controls & Defensive Architecture

Planned Wallet implements defense-in-depth principles across transport, identity, access control, and application layers:

### 4.1 Authentication & Credential Hardening
- **BCrypt Password Hashing**: Passwords are cryptographically salted and hashed using BCrypt at standard work factor.
- **Strict Password Complexity Policy**:
  - Minimum length: 10 characters.
  - Mandatory uppercase, lowercase, numeric, and special character requirements.
- **RFC 6238 TOTP Multi-Factor Authentication**:
  - Every registered account generates a high-entropy Base32 TOTP secret.
  - Token exchange (`POST /api/v1/users/{username}/token`) mandates both correct password and a valid, synchronized 6-digit TOTP code.
- **Cryptographic JWT Token Issuance**:
  - High-entropy HS256 JWT tokens containing user claims, subscription tiers, and strict time-to-live expiration (`JWT_EXPIRATION_SECONDS`).

---

### 4.2 Automated Anti-Bot & CAPTCHA System
- Procedural algorithmic visual CAPTCHA engine operating with zero third-party tracking or external CDN dependencies.
- Generates distortion-mapped, noise-injected PNG images with time-decayed challenge tokens.
- Mandatory CAPTCHA enforcement on user registration endpoints in production (`REQUIRE_CAPTCHA=true`).
- In-memory and database-backed challenge expiration with automatic startup cleanup and time-to-live invalidation.

---

### 4.3 Multi-Tenant Isolation & Role-Based Access Control (RBAC)
- **Account Access Matrix (`account_access`)**:
  - `admin`: Complete authority over account records, holdings, snapshots, and user access delegation.
  - `edit`: Permission to create, modify, and delete transactions, assets, and snapshots. Blocked from altering access lists or deleting the account.
  - `read`: Read-only visibility over transactions and balances. All mutation attempts rejected with `403 Forbidden`.
- **Zero-Knowledge Anti-Enumeration Policy**:
  - Unauthenticated or unauthorized requests targeting private accounts receive `404 Not Found` (`ACCOUNT_NOT_FOUND`) rather than `403 Forbidden`, preventing malicious actor probing or enumeration of account IDs.

---

### 4.4 Perimeter & Edge Defenses
- **W3C Trace Context Standard**:
  - End-to-end distributed tracing across all layers via the standard `traceparent` HTTP header (`00-{trace_id}-{span_id}-{flags}`).
  - Automatic trace ID generation on the client, passed through the Nginx gateway, propagated into backend Gin contexts, and bound to structured application logs.
- **Mandatory Client Identification (`X-Client-ID`)**:
  - All API interactions require the `X-Client-ID` header. Requests omitting the header are rejected with `400 Bad Request`.
- **Azure Front Door Security Invariant (`X-Azure-FDID`)**:
  - Production deployments validate the reverse proxy / ingress header against the expected Front Door identifier, dropping spoofed external traffic.
- **Strict CORS & Host Protection**:
  - Preflight `OPTIONS` and runtime origin inspection ensuring requests originate only from whitelisted hosts and origins.
- **Deployment Concurrency Protections**:
  - GitHub Actions deployment workflows for `backend-api`, `frontend-ui`, and `apply-sql` enforce strict concurrency grouping (`cancel-in-progress: false`).
  - Guarantees that two deployments or database migration runs can never execute simultaneously, eliminating migration race conditions and schema corruption.

---

## 5. Branding, Compliance & Design Standards

- **Strict "No Registered Business / Brand Names" Policy**:
  - Total exclusion of registered trademarks, financial institution names, commercial banks, or proprietary brand marks across the entire repository.
  - Applies to all UI labels, form placeholders, unit tests, integration test mocks, database seeds, OpenAPI contracts, and documentation.
  - Universal adoption of descriptive, generic terminology (e.g., "Primary Everyday Checking", "Emergency Reserve Savings", "Precious Metals Depository", "Brokerage Snapshot", "Corner Grocery", "Payroll Deposit").
- **Cohesive Design System & Color Tokens**:
  - Distinctive visual indicators for financial accounts preventing operational errors:
    - Checking: Sky blue accent (`#0284c7`, background `#e0f2fe`)
    - Savings: Emerald green accent (`#059669`, background `#d1fae5`)
    - Credit Card: Calm violet accent (`#8b5cf6`, border `#ddd6fe`, background `#f5f3ff`)
    - Assets: Deep teal accent (`#0d9488`, background `#f0fdfa`)
    - Snapshots: Royal purple accent (`#7c3aed`, background `#f5f3ff`)
  - Full responsive layout support (mobile, tablet, desktop) with modular CSS stylesheets.

---

## 6. API Specification & Endpoint Catalog

The platform exposes an OpenAPI 3.2.0 compliant REST interface, fully detailed below:

### 6.1 Authentication & System Health
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/hello` | Service ping endpoint returning standard greeting message. | Public |
| `GET` | `/api/v1/health` | Deep health check confirming database connectivity and responsiveness. | Public |
| `GET` | `/api/v1/captcha` | Generates a new visual CAPTCHA challenge returning ID and base64 PNG data. | Public |
| `GET` | `/api/v1/captcha/{captchaId}` | Streams raw PNG image bytes for a specific CAPTCHA challenge ID. | Public |
| `POST` | `/api/v1/users` | Registers a new user with password complexity and CAPTCHA validation. | Public |
| `POST` | `/api/v1/users/{username}/token` | Exchanges username, password, and TOTP code for a JWT access token. | Public |

### 6.2 Financial Accounts & Access Control
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/accounts` | Creates a new account (`checking`, `savings`, `credit card`, `asset`, `snapshot`). | Bearer |
| `GET` | `/api/v1/accounts` | Lists all financial accounts accessible to the authenticated user. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}` | Retrieves detailed metadata for a specific account. | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}` | Permanently deletes account and cascades child records (creator-only). | Bearer |
| `POST` | `/api/v1/accounts/{accountId}/access` | Grants or updates user access level (`read`, `edit`, `admin`). | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/access/{userId}` | Views a specific user's access level for an account. | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}/access/{userId}` | Revokes a user's access permissions to an account. | Bearer |

### 6.3 Checking, Savings & Credit Card Records
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/accounts/{accountId}/records` | Inserts a transaction record under a checking, savings, or credit card account. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/records` | Retrieves records with cumulative running balances and optional date filters. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/records/{recordId}` | Retrieves a single transaction record by ID. | Bearer |
| `PUT` | `/api/v1/accounts/{accountId}/records/{recordId}` | Replaces all fields of an existing transaction record. | Bearer |
| `PATCH` | `/api/v1/accounts/{accountId}/records/{recordId}` | Partially updates record status (e.g., toggling clearance timestamp). | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}/records/{recordId}` | Permanently removes a transaction record from the account ledger. | Bearer |

### 6.4 Asset Catalog & Valuations
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/assets` | Creates a new asset class definition (scoped to user or system-wide). | Bearer |
| `GET` | `/api/v1/assets` | Lists all defined asset classes available to the user. | Bearer |
| `GET` | `/api/v1/assets/{assetId}` | Retrieves a single asset class definition. | Bearer |
| `PUT` | `/api/v1/assets/{assetId}` | Updates asset class label and description. | Bearer |
| `DELETE` | `/api/v1/assets/{assetId}` | Deletes asset class, cascading associated valuations and holdings. | Bearer |
| `POST` | `/api/v1/assets/{assetId}/values` | Records a date-keyed unit valuation for an asset class. | Bearer |
| `GET` | `/api/v1/assets/{assetId}/values` | Lists historical unit valuations for an asset class. | Bearer |
| `GET` | `/api/v1/assets/{assetId}/values/{valueId}` | Retrieves a specific date-keyed valuation record. | Bearer |
| `PUT` | `/api/v1/assets/{assetId}/values/{valueId}` | Updates a valuation unit price or date. | Bearer |
| `DELETE` | `/api/v1/assets/{assetId}/values/{valueId}` | Removes a unit valuation record. | Bearer |

### 6.5 Asset Storage Locations & Quantity Records
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/accounts/{accountId}/stores` | Creates a labeled storage location/vault under an asset account. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores` | Lists all storage locations associated with an asset account. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/summary` | Returns calculated account valuation and breakdown across all stores. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/trend` | Returns time-series valuation points tracking account worth over time. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/{storeId}` | Retrieves metadata for a specific storage location. | Bearer |
| `PUT` | `/api/v1/accounts/{accountId}/stores/{storeId}` | Updates label or description of a storage location. | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}/stores/{storeId}` | Deletes storage location and cascades all point-in-time quantity records. | Bearer |
| `POST` | `/api/v1/accounts/{accountId}/stores/{storeId}/records` | Records a point-in-time quantity of an asset held at a specific store. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/{storeId}/records` | Lists historical quantity records for a specific store. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/{storeId}/records/{recordId}` | Retrieves a specific store quantity record. | Bearer |
| `PUT` | `/api/v1/accounts/{accountId}/stores/{storeId}/records/{recordId}` | Updates held quantity or date for a store record. | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}/stores/{storeId}/records/{recordId}` | Deletes a store quantity record. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/stores/{storeId}/trend` | Returns historical valuation trend specific to an individual store location. | Bearer |

### 6.6 Account Snapshots
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/accounts/{accountId}/snapshots` | Records a point-in-time account valuation (`value`, `snapshot_date`). | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/snapshots` | Lists chronological snapshots with optional date range filters. | Bearer |
| `GET` | `/api/v1/accounts/{accountId}/snapshots/{snapshotId}` | Retrieves a specific valuation snapshot record. | Bearer |
| `PUT` | `/api/v1/accounts/{accountId}/snapshots/{snapshotId}` | Modifies valuation value or snapshot date. | Bearer |
| `DELETE` | `/api/v1/accounts/{accountId}/snapshots/{snapshotId}` | Deletes a snapshot valuation entry. | Bearer |

### 6.7 Perspectives & Analytical Aggregations
| Method | Path | Summary & Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/perspectives` | Creates an analytical perspective aggregating multiple financial accounts. | Bearer |
| `GET` | `/api/v1/perspectives` | Lists perspectives accessible to the user (owned and shared). | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}` | Retrieves perspective metadata, settings, and linked account IDs. | Bearer |
| `PUT` | `/api/v1/perspectives/{perspectiveId}` | Updates perspective name and description. | Bearer |
| `DELETE` | `/api/v1/perspectives/{perspectiveId}` | Deletes perspective configuration (admin-only, preserves underlying accounts). | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}/settings` | Retrieves calculation toggles and spend simulation settings. | Bearer |
| `PUT` | `/api/v1/perspectives/{perspectiveId}/settings` | Updates financial trend, YoY, ruined date, and spend override settings. | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}/accounts` | Lists all account objects linked to the perspective. | Bearer |
| `POST` | `/api/v1/perspectives/{perspectiveId}/accounts` | Links an additional financial account into the perspective. | Bearer |
| `DELETE` | `/api/v1/perspectives/{perspectiveId}/accounts/{accountId}` | Unlinks an account from the perspective. | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}/access` | Lists all user access grants for the perspective. | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}/access/{userId}` | Views a specific user's access level for the perspective. | Bearer |
| `POST` | `/api/v1/perspectives/{perspectiveId}/access` | Grants access permissions to another user for the perspective. | Bearer |
| `DELETE` | `/api/v1/perspectives/{perspectiveId}/access/{userId}` | Revokes a user's access to the perspective. | Bearer |
| `GET` | `/api/v1/perspectives/{perspectiveId}/view` | Returns fully calculated time-series trend, runway metrics, and spend averages. | Bearer |

---

## 7. Quality Assurance, Testing & Code Health Standards

The codebase adheres to rigorous engineering constraints enforced via automated test runners and linters:

- **Strict Complexity & Function Boundaries**:
  - Maximum function length: **$\le 40$ lines**.
  - Maximum cyclomatic complexity per function: **$\le 6$**.
  - All complex workflows decomposed into clean, single-responsibility helper functions.
- **Repository Dual-Store Parity**:
  - Every data access method in PostgreSQL (`postgres.go`) is identically implemented in the in-memory mock repository (`memory.go`) with matching error semantics (`ErrNotFound`, `ErrConflict`).
  - Unit tests execute instantaneously in-memory without external database dependencies.
- **Automated Integration Test Harness**:
  - 35 end-to-end integration test suites (`itests`) written in Python utilizing Pytest and Playwright's API client.
  - Automated tests execute against completely clean database volumes (`make refresh`), validating:
    - User registration, password complexity rejection, and duplicate prevention.
    - TOTP exchange, token validity, and subscriber claims.
    - CAPTCHA generation, rendering, and solution verification.
    - Checking, savings, and credit card ledger integrity and running balances.
    - Asset classes, multi-day valuations, store partitioning, and holding records.
    - Snapshot accounts, duplicate date rejection, and time-series trending.
    - Perspective creation, multi-account linking, transfer deduplication, and runway analytics.
    - Security invariants: SQL injection resistance, perimeter header requirements, CORS preflight controls, and user isolation.
    - Account deletion access enforcement (non-creator rejection) and immediate collision-free re-creation.
- **Continuous Integration & Static Analysis**:
  - `go vet` and Go unit tests passing with zero warnings.
  - TypeScript strict compilation (`tsc --noEmit`) and Turbopack Next.js build verification passing with zero errors.
