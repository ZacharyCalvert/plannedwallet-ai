-- Destroy Stage Data
-- Target Environment: stage ONLY
-- Description: Permanently removes all accounts and all users in the stage database.
-- Foreign key ON DELETE CASCADE automatically purges all child records:
-- checking_records, account_access, asset_stores, asset_store_records,
-- account_assets, account_snapshots, perspective_accounts, user-scoped assets,
-- asset_values, subscriptions, perspectives, perspective_settings, perspective_access.

BEGIN;

-- Purge all financial accounts (cascades checking records, holdings, stores, snapshots, perspective links)
DELETE FROM accounts;

-- Purge all registered users (cascades user assets, subscriptions, perspectives, access grants)
DELETE FROM users;

COMMIT;
