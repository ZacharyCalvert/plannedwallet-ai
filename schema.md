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

### 1.4 `checking_records`
Stores transaction line items for `checking`, `savings`, and `credit card` accounts.
- `id` (UUID, Primary Key)
- `account_id` (UUID, Not Null, FK -> `accounts(id)` ON DELETE CASCADE)
- `value` (DOUBLE PRECISION, Not Null) - Positive (credits/deposits) or negative (debits/expenses)
- `date_sent` (DATE, Not Null)
- `date_cleared` (DATE, Nullable)
- `entity` (VARCHAR(255), Not Null) - Payee or payer
- `description` (VARCHAR(500), Not Null, Default '')
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- `updated_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Index**: `idx_checking_records_account_datesent (account_id, date_sent DESC)`

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

### 1.9 `captchas`
Shared backing store for ephemeral CAPTCHA challenges to support 12-factor stateless application instances. Expired challenges older than 5 minutes are purged on application startup and expired on retrieval.
- `id` (VARCHAR(64), Primary Key) - Challenge identifier
- `digits` (BYTEA, Not Null) - Challenge solution digits
- `created_at` (TIMESTAMPTZ, Not Null, Default NOW())
- **Index**: `idx_captchas_created_at (created_at)`

---

## 2. Account Types & Behavior

1. **`checking`**: Transactional account tracking checking records. Cumulative balances are calculated bottom-up.
2. **`savings`**: Transactional savings account tracking checking records identically to `checking`.
3. **`credit card`**: Transactional credit card account tracking transactions identically to `checking`/`savings`.
4. **`asset`**: Holds collections of assets and quantities (`account_assets`). Current worth is computed as $\sum (\text{quantity} \times \text{latest\_value})$.
5. **`snapshot`**: Tracks point-in-time account valuations over time (`account_snapshots`). Ordered chronologically by `snapshot_date ASC, created_at ASC` to trend account worth over time without tracking individual transactions or asset quantities.

---

## 3. API Specification

All HTTP requests require the `X-Client-ID` header. Protected endpoints require the `Authorization: Bearer <JWT>` header.

### 3.1 Health
- `GET /api/v1/hello`: Liveness smoke test.

### 3.2 Authentication, Security & Users
- `GET /api/v1/captcha`: Generate a new visual CAPTCHA challenge (returns unique ID and base64 PNG data URI) to prevent automated bot account creations.
- `GET /api/v1/captcha/{captchaId}`: Stream raw PNG image bytes for a CAPTCHA challenge.
- `POST /api/v1/users`: User self-registration with password complexity, uniqueness checks, and CAPTCHA challenge verification (`captcha_id`, `captcha_solution`).
- `POST /api/v1/users/{username}/token`: TOTP-authenticated JWT token exchange.

### 3.3 Accounts
- `POST /api/v1/accounts`: Create account (`checking`, `savings`, `credit card`, `asset`, `snapshot`).
- `GET /api/v1/accounts`: List accounts accessible by the user.
- `GET /api/v1/accounts/{accountId}`: Get account metadata.

### 3.4 Account Access
- `POST /api/v1/accounts/{accountId}/access`: Grant or update user access permissions (`read`, `edit`, `admin`).
- `GET /api/v1/accounts/{accountId}/access/{userId}`: View user access level.
- `DELETE /api/v1/accounts/{accountId}/access/{userId}`: Revoke user access.

### 3.5 Checking Records
- `POST /api/v1/accounts/{accountId}/records`: Add transaction record.
- `GET /api/v1/accounts/{accountId}/records`: List transaction records with cumulative running balances.
- `GET /api/v1/accounts/{accountId}/records/{recordId}`: Get single transaction record.
- `PUT /api/v1/accounts/{accountId}/records/{recordId}`: Update transaction record.
- `PATCH /api/v1/accounts/{accountId}/records/{recordId}`: Patch transaction record.
- `DELETE /api/v1/accounts/{accountId}/records/{recordId}`: Delete transaction record.

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
