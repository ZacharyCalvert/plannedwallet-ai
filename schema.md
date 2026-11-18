# Planned Wallet - Database Schema & API Specification

## 1. Database Schema

The database is PostgreSQL 16. All table schemas and constraints are defined in `backend-api/internal/database/init.sql`.

### 1.1 `users`
Stores registered user accounts with encrypted credentials and TOTP secrets.
- `id` (UUID, Primary Key)
- `username` (VARCHAR(50), Unique, Not Null)
- `email` (VARCHAR(255), Unique, Not Null)
- `password_hash` (VARCHAR(255), Not Null) - bcrypt hashed password
- `totp_secret` (VARCHAR(64), Not Null, Default '') - RFC 6238 TOTP base32 secret
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())

### 1.2 `accounts`
Stores financial accounts owned by an administrator/owner user.
- `id` (UUID, Primary Key)
- `admin_id` (UUID, Not Null, FK -> `users(id)` ON DELETE CASCADE)
- `name` (VARCHAR(100), Not Null)
- `type` (VARCHAR(50), Not Null) - Account type: `checking`, `savings`, `credit card`, `asset`, `snapshot`
- `description` (VARCHAR(255), Not Null)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())

### 1.3 `account_access`
Enforces multi-user access control and sharing permissions.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `user_id` (UUID, Not Null, FK -> `users(id)` ON DELETE CASCADE)
- `access_level` (VARCHAR(20), Not Null) - Allowed: `'read'`, `'edit'`, `'admin'`
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_account_access_account_user UNIQUE (account_id, user_id)`
- **Indexes**: `idx_account_access_user_id`, `idx_account_access_account_id`

### 1.4 `categories`
Stores transaction categories for `checking`, `savings`, and `credit card` records.
- `id` (UUID, Primary Key)
- `user_id` (UUID, Nullable, FK -> `users(id)` ON DELETE CASCADE) - `NULL` for global default categories (`Housing`, `Transportation`, `Groceries`, `Utilities`, `Healthcare`, `Insurance`, `Debt`)
- `name` (VARCHAR(100), Not Null)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraints**: `UNIQUE NULLS NOT DISTINCT (user_id, name)`, `uq_categories_user_lower_name (user_id, LOWER(name)) NULLS NOT DISTINCT`
- **Index**: `idx_categories_user_id`

### 1.5 `reconcile_reports`
Stores completed reconciliation reports for `checking`, `savings`, and `credit card` accounts.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `name` (VARCHAR(100), Not Null, Default '') - Optional label (e.g. `Jan2026`, `Year-End`)
- `report_date` (DATE, Not Null) - Report date (also applied as `date_cleared` for any attached record missing a cleared date)
- `starting_balance` (DOUBLE PRECISION, Not Null)
- `ending_balance` (DOUBLE PRECISION, Not Null) - Enforced to equal `starting_balance + SUM(attached records)`
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Index**: `idx_reconcile_reports_account_date (account_id, report_date DESC)`

### 1.6 `checking_records`
Stores transaction line items for `checking`, `savings`, and `credit card` accounts.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `category_id` (UUID, Nullable, FK -> `categories(id)` ON DELETE SET NULL) - Optional transaction category; set to `NULL` if a user-owned category is deleted
- `reconcile_report_id` (UUID, Nullable, FK -> `reconcile_reports(id)` ON DELETE SET NULL) - Links a record to at most one reconcile report; reconciled records lock `value`, `date_sent`, `date_cleared`, and `entity` (`409 Conflict` on edits)
- `value` (DOUBLE PRECISION, Not Null) - Positive (credits/deposits) or negative (debits/expenses)
- `date_sent` (DATE, Not Null)
- `date_cleared` (DATE, Nullable)
- `entity` (VARCHAR(255), Not Null) - Payee or payer
- `description` (VARCHAR(500), Not Null, Default '')
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Indexes**: `idx_checking_records_account_datesent (account_id, date_sent DESC)`, `idx_checking_records_category_id (category_id)`, `idx_checking_records_reconcile_report_id (reconcile_report_id)`

### 1.5 `assets`
Stores asset definitions (e.g. Bitcoin, Silver, Stocks).
- `id` (UUID, Primary Key)
- `user_id` (UUID, Nullable, FK -> `users(id)` ON DELETE CASCADE) - `NULL` for globally available assets
- `label` (VARCHAR(100), Not Null) - Asset ticker/symbol (e.g. `btc`, `slv-oz`)
- `description` (VARCHAR(200), Not Null, Default '')
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraints**:
  - `CONSTRAINT uq_assets_user_label UNIQUE NULLS NOT DISTINCT (user_id, label)`
  - `CREATE UNIQUE INDEX uq_assets_user_lower_label ON assets (user_id, LOWER(label)) NULLS NOT DISTINCT`
- **Indexes**: `idx_assets_user_id`, `idx_assets_label`

### 1.6 `asset_values`
Stores date-keyed valuation points for assets.
- `id` (UUID, Primary Key)
- `asset_id` (UUID, Not Null, FK -> `assets(id)` ON DELETE CASCADE)
- `value` (DOUBLE PRECISION, Not Null) - USD price per unit
- `value_date` (DATE, Not Null)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_asset_values_asset_date UNIQUE (asset_id, value_date)`
- **Index**: `idx_asset_values_asset_date (asset_id, value_date DESC)`

### 1.7 `asset_stores`
Maps asset classes to unique labeled storage locations / holdings within an asset account.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `asset_id` (UUID, Not Null, FK -> `assets(id)` ON DELETE CASCADE)
- `label` (VARCHAR(100), Not Null) - Unique label within account (e.g. "red", "blue", "attic box", "silver mattress")
- `description` (VARCHAR(500), Not Null, Default `''`)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraints**: `CONSTRAINT uq_asset_stores_account_label UNIQUE (account_id, label)`, `uq_asset_stores_account_lower_label (account_id, LOWER(label))`
- **Indexes**: `idx_asset_stores_account_id`, `idx_asset_stores_asset_id`

### 1.8 `asset_store_records`
Stores point-in-time quantity records by date for a given asset store.
- `id` (UUID, Primary Key)
- `asset_store_id` (UUID, Not Null, FK -> `asset_stores(id)` ON DELETE CASCADE)
- `quantity` (DOUBLE PRECISION, Not Null, `CHECK (quantity > 0)`)
- `record_date` (DATE, Not Null)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_asset_store_records_store_date UNIQUE (asset_store_id, record_date)`
- **Indexes**: `idx_asset_store_records_store_id`, `idx_asset_store_records_store_date (asset_store_id, record_date DESC)`

### 1.9 `account_assets`
Legacy asset mapping table retained for backward compatibility.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `asset_id` (UUID, Not Null, FK -> `assets(id)` ON DELETE CASCADE)
- `quantity` (DOUBLE PRECISION, Not Null, `CHECK (quantity > 0)`)
- `asset_date` (DATE, Not Null, Default `CURRENT_DATE`)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_account_assets_account_asset_date UNIQUE (account_id, asset_id, asset_date)`
- **Indexes**: `idx_account_assets_account_id`, `idx_account_assets_asset_id`, `idx_account_assets_account_date (account_id, asset_date DESC)`

### 1.10 `account_snapshots`
Stores manual point-in-time valuations for `snapshot` accounts.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `value` (DOUBLE PRECISION, Not Null) - Account valuation in USD
- `snapshot_date` (DATE, Not Null)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)`
- **Index**: `idx_account_snapshots_account_date (account_id, snapshot_date ASC)`

### 1.11 `captchas`
Shared backing store for ephemeral CAPTCHA challenges to support 12-factor stateless application instances. Expired challenges older than 5 minutes are purged on application startup and expired on retrieval.
- `id` (VARCHAR(64), Primary Key) - Challenge identifier
- `digits` (BYTEA, Not Null) - Challenge solution digits
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Index**: `idx_captchas_created_at (created_at)`

### 1.12 `subscriptions`
Stores paid subscriber membership records defining service periods for users.
- `id` (UUID, Primary Key)
- `user_id` (UUID, Not Null, FK -> `users(id)` ON DELETE CASCADE)
- `paid_date` (DATE, Not Null) - Date payment was recorded
- `subscriber_ends` (DATE, Not Null) - Final full day of paid services being active
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Indexes**: `idx_subscriptions_user_id`, `idx_subscriptions_user_ends (user_id, subscriber_ends DESC)`

### 1.13 `perspectives`
Stores multi-account aggregated perspective metadata.
- `id` (UUID, Primary Key)
- `user_id` (UUID, Not Null, FK -> `users(id)` ON DELETE CASCADE)
- `name` (VARCHAR(100), Not Null)
- `description` (VARCHAR(255), Not Null, Default `''`)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Index**: `idx_perspectives_user_id (user_id)`

### 1.14 `perspective_settings`
Stores analytical viewing preferences and calculation settings.
- `id` (UUID, Primary Key)
- `perspective_id` (UUID, Not Null, FK -> `perspectives(id)` ON DELETE CASCADE)
- `show_financial_trend` (BOOLEAN, Not Null, Default TRUE)
- `calculate_average_daily_spend` (BOOLEAN, Not Null, Default FALSE)
- `calculate_average_return_yoy` (BOOLEAN, Not Null, Default FALSE)
- `track_ruined_date` (BOOLEAN, Not Null, Default FALSE)
- `custom_daily_spend` (DOUBLE PRECISION, Nullable)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_perspective_settings_perspective UNIQUE (perspective_id)`
- **Index**: `idx_perspective_settings_perspective_id (perspective_id)`

### 1.15 `perspective_accounts`
Maps financial accounts into aggregated perspectives.
- `id` (UUID, Primary Key)
- `perspective_id` (UUID, Not Null, FK -> `perspectives(id)` ON DELETE CASCADE)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_perspective_accounts_perspective_account UNIQUE (perspective_id, account_id)`
- **Indexes**: `idx_perspective_accounts_perspective_id`, `idx_perspective_accounts_account_id`

### 1.16 `perspective_access`
Multi-user sharing and role-based permissions (`read`, `edit`, `admin`) for perspectives.
- `id` (UUID, Primary Key)
- `perspective_id` (UUID, Not Null, FK -> `perspectives(id)` ON DELETE CASCADE)
- `user_id` (UUID, Not Null, FK -> `users(id)` ON DELETE CASCADE)
- `access_level` (VARCHAR(20), Not Null) - `'read'`, `'edit'`, `'admin'`
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Constraint**: `CONSTRAINT uq_perspective_access_perspective_user UNIQUE (perspective_id, user_id)`
- **Indexes**: `idx_perspective_access_perspective_id`, `idx_perspective_access_user_id`

---

## 2. Account Types & Behavior

1. **`checking`**: Transactional account tracking checking records. Cumulative balances are calculated bottom-up.
2. **`savings`**: Transactional savings account tracking checking records identically to `checking`.
3. **`credit card`**: Transactional credit card account tracking transactions identically to `checking`/`savings`.
4. **`asset`**: Holds collections of assets and quantities (`account_assets`). Current worth is computed as $\sum (\text{quantity} \times \text{latest\_value})$.
5. **`snapshot`**: Tracks point-in-time account valuations over time (`account_snapshots`). Ordered chronologically by `snapshot_date ASC, created_at ASC` to trend account worth over time without tracking individual transactions or asset quantities.

---

## 3. API Specification

All HTTP requests require the `X-Client-ID` header. Protected endpoints require the `Authorization: Bearer <JWT>` header. The JWT payload includes `user_id`, `username`, and `subscriber` (boolean flag denoting active paid subscription status).

### 3.1 Health
- `GET /api/health`: Database-confirmed health check endpoint (`{"status": "up"}`).
- `GET /api/v1/hello`: Liveness smoke test.

### 3.2 Authentication, Security & Users
- `GET /api/v1/captcha`: Generate a new visual CAPTCHA challenge (returns unique ID and base64 PNG data URI) to prevent automated bot account creations.
- `GET /api/v1/captcha/{captchaId}`: Stream raw PNG image bytes for a CAPTCHA challenge.
- `POST /api/v1/users`: User self-registration with password complexity, uniqueness checks, and CAPTCHA challenge verification (`captcha_id`, `captcha_solution`).
- `POST /api/v1/users/{username}/token`: TOTP-authenticated JWT token exchange (issues a 15-minute / 900s JWT by default).
- `POST /api/v1/users/{username}/token/refresh`: Exchanges an existing, valid Bearer JWT for a freshly signed 15-minute JWT without re-entering credentials.

### 3.3 Accounts
- `POST /api/v1/accounts`: Create account (`checking`, `savings`, `credit card`, `asset`, `snapshot`).
- `GET /api/v1/accounts`: List accounts accessible by the user.
- `GET /api/v1/accounts/{accountId}`: Get account metadata.
- `PATCH /api/v1/accounts/{accountId}`: Partially update an account (`name` and/or `description`; requires `edit` or `admin` access; rejects immutable/unknown fields).
- `DELETE /api/v1/accounts/{accountId}`: Permanently delete account and cascade all associated records, holdings, snapshots, and perspective links (creator/admin only).

### 3.4 Account Access
- `POST /api/v1/accounts/{accountId}/access`: Grant or update user access permissions (`read`, `edit`, `admin`).
- `GET /api/v1/accounts/{accountId}/access/{userId}`: View user access level.
- `DELETE /api/v1/accounts/{accountId}/access/{userId}`: Revoke user access.

### 3.5 Checking Records & Categories
- `POST /api/v1/accounts/{accountId}/records`: Add transaction record (with optional `category_id`).
- `GET /api/v1/accounts/{accountId}/records`: List transaction records with cumulative running balances.
- `GET /api/v1/accounts/{accountId}/records/categories`: Get expense totals grouped by category (for pie chart visualization).
- `GET /api/v1/accounts/{accountId}/records/{recordId}`: Get single transaction record.
- `PUT /api/v1/accounts/{accountId}/records/{recordId}`: Update transaction record.
- `PATCH /api/v1/accounts/{accountId}/records/{recordId}`: Patch transaction record.
- `DELETE /api/v1/accounts/{accountId}/records/{recordId}`: Delete transaction record.
- `GET /api/v1/categories`: List accessible categories (global defaults, user-created, and categories used in accessible/shared accounts).
- `POST /api/v1/categories`: Create a user-scoped category.
- `GET /api/v1/categories/{categoryId}`: Get a single accessible category.
- `DELETE /api/v1/categories/{categoryId}`: Delete a user-created category (updates associated records to `category_id = null`; default categories cannot be deleted).
- `POST /api/v1/accounts/{accountId}/reconcile-reports`: Create a reconcile report (enforces `starting_balance + sum(records) == ending_balance`, defaults missing `date_cleared` to `report_date`, and locks reconciled records from `value`/`date_sent`/`date_cleared`/`entity` edits with `409 Conflict`).
- `GET /api/v1/accounts/{accountId}/reconcile-reports`: List completed reconcile reports for an account.
- `GET /api/v1/accounts/{accountId}/reconcile-reports/{reportId}`: Get a single reconcile report and its attached records.
- `DELETE /api/v1/accounts/{accountId}/reconcile-reports/{reportId}`: Delete a reconcile report, unlinking attached records (`reconcile_report_id = null`) without deleting the records.

### 3.6 Assets & Valuations
- `POST /api/v1/assets`: Create asset definition.
- `GET /api/v1/assets`: List assets (user-defined and system/global).
- `GET /api/v1/assets/{assetId}`: Get asset definition.
- `PUT /api/v1/assets/{assetId}`: Update asset label/description.
- `DELETE /api/v1/assets/{assetId}`: Delete asset definition (cascades valuations and holdings).
- `POST /api/v1/assets/{assetId}/values`: Record daily valuation for asset.
- `GET /api/v1/assets/{assetId}/values`: List daily valuations for asset.
- `GET /api/v1/assets/{assetId}/values/{valueId}`: Get specific valuation.
- `PUT /api/v1/assets/{assetId}/values/{valueId}`: Update valuation value or date.
- `DELETE /api/v1/assets/{assetId}/values/{valueId}`: Delete valuation.

### 3.7 Account Assets (Holdings)
- `POST /api/v1/accounts/{accountId}/assets`: Add asset holding to asset account.
- `GET /api/v1/accounts/{accountId}/assets`: List asset holdings and valuations in account.
- `GET /api/v1/accounts/{accountId}/assets/summary`: Get aggregated account valuation and breakdown.
- `GET /api/v1/accounts/{accountId}/assets/trend`: Retrieve time-series valuation points (`date`, `total_value`) tracking account worth over time.
- `GET /api/v1/accounts/{accountId}/assets/{assetId}`: Get specific asset holding.
- `PUT /api/v1/accounts/{accountId}/assets/{assetId}`: Update quantity held.
- `DELETE /api/v1/accounts/{accountId}/assets/{assetId}`: Remove asset from account.

### 3.8 Account Snapshots (Valuation Trending)
- `POST /api/v1/accounts/{accountId}/snapshots`: Record point-in-time valuation (`value`, `snapshot_date`).
- `GET /api/v1/accounts/{accountId}/snapshots`: List snapshots ordered chronologically (`snapshot_date ASC, created_at ASC`), optional `start_date` and `end_date` filters.
- `GET /api/v1/accounts/{accountId}/snapshots/{snapshotId}`: Get specific snapshot record.
- `PUT /api/v1/accounts/{accountId}/snapshots/{snapshotId}`: Update snapshot valuation or date.
- `DELETE /api/v1/accounts/{accountId}/snapshots/{snapshotId}`: Delete snapshot record.

### 3.9 Perspectives & Aggregated Analytics
- `POST /api/v1/perspectives`: Create a new perspective viewing window (`name`, `description`, optional `account_ids`, optional `settings`).
- `GET /api/v1/perspectives`: List perspectives accessible by the authenticated user (owned and shared).
- `GET /api/v1/perspectives/{perspectiveId}`: Get perspective details and settings.
- `PUT /api/v1/perspectives/{perspectiveId}`: Update perspective name or description.
- `DELETE /api/v1/perspectives/{perspectiveId}`: Delete perspective (admin only).
- `GET /api/v1/perspectives/{perspectiveId}/settings`: Get perspective settings.
- `PUT /api/v1/perspectives/{perspectiveId}/settings`: Update calculation and display toggles (`show_financial_trend`, `calculate_average_daily_spend`, `calculate_average_return_yoy`, `track_ruined_date`, `custom_daily_spend`).
- `GET /api/v1/perspectives/{perspectiveId}/accounts`: List accounts linked to perspective.
- `POST /api/v1/perspectives/{perspectiveId}/accounts`: Link an account to perspective (`account_id`).
- `DELETE /api/v1/perspectives/{perspectiveId}/accounts/{accountId}`: Unlink an account from perspective.
- `GET /api/v1/perspectives/{perspectiveId}/access`: List user access grants on perspective.
- `GET /api/v1/perspectives/{perspectiveId}/access/{userId}`: View access grant for a specific user.
- `POST /api/v1/perspectives/{perspectiveId}/access`: Grant or update user access level (`read`, `edit`, `admin`).
- `DELETE /api/v1/perspectives/{perspectiveId}/access/{userId}`: Revoke access for a user.
- `GET /api/v1/perspectives/{perspectiveId}/view`: Retrieve comprehensive aggregated viewing window (total portfolio value, historical trend line, average daily spend, year-over-year return rate, and ruined date runway forecast).

### 3.10 System & Feature Flags
- `GET /api/v1/features`: Returns enabled backend feature flags (e.g., `{"features": ["read_only"]}` when `PW_READONLY=true`, otherwise `{"features": []}`).
  - When `PW_READONLY=true`, write operations (`POST`, `PUT`, `PATCH`, `DELETE`) return `503 Service Unavailable` (`READ_ONLY_MODE`) while retaining read (`GET`) routes, user login (`POST /api/v1/users/{username}/token`), and JWT refresh (`POST /api/v1/users/{username}/token/refresh`).

### 3.11 Recurring Transactions & Reminders (`recurring_transactions`)
- Recurring transactions are strictly user-scoped (`recurring_transactions.user_id`) and represent both bills (`amount < 0`) and income (`amount > 0`) through a unified table and API.
- Supports optional short display `name` (`VARCHAR(19)`, strictly under 20 characters, e.g., `"Electric"` or `"Paycheck"` while `entity` holds `"Electric Utility"` or `"Employer Payroll"`), non-zero `amount` (`CHECK (amount <> 0)`), `frequency` (`weekly`, `bi-monthly`, `monthly`, `yearly`), `interval` (`>= 1`, default `1`, e.g., `frequency="monthly"` + `interval=6` for every 6 months), optional structured `schedule` rules validated against `frequency` (`weekday`, `first_fifteenth`, `fifteenth_last`, `day_of_period`, `nth_weekday`, `month_day`), `last_date_applied`, `reminder` alert flag, optional `account_id` (`checking`, `savings`, `credit card`), and optional `category_id` (requires `account_id`).
- Deleting a linked category sets `recurring_transactions.category_id = NULL`; deleting a linked account sets both `recurring_transactions.account_id = NULL` and `recurring_transactions.category_id = NULL` while preserving the recurring transaction.
- `POST /api/v1/recurring-transactions`: Create a recurring transaction.
- `GET /api/v1/recurring-transactions`: List recurring transactions owned by the authenticated user with computed `next_due_date`, `due_status`, and `days_until_due`.
- `GET /api/v1/recurring-transactions/reminders`: List `reminder=true` recurring transactions whose computed `due_status` is `upcoming` or `overdue`.
- `GET /api/v1/recurring-transactions/{recurringId}`: Get recurring transaction details.
- `PUT /api/v1/recurring-transactions/{recurringId}`: Update a recurring transaction.
- `DELETE /api/v1/recurring-transactions/{recurringId}`: Delete a recurring transaction.
- `POST /api/v1/recurring-transactions/{recurringId}/apply`: Creates a transaction record in the target checking/savings/credit card account and updates `recurring_transactions.last_date_applied`.



