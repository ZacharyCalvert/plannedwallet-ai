-- Planned Wallet Database Statistics (Read-Only)
-- Target: stage and production environments
-- Reports summary metrics (users, accounts) and exact record counts for all database tables.

\pset pager off
\pset footer off

-- 1. Summary Metrics: Number of Registered Users and Financial Accounts
SELECT 
    'Registered Users' AS metric,
    COUNT(*) AS total_count
FROM users
UNION ALL
SELECT 
    'Financial Accounts' AS metric,
    COUNT(*) AS total_count
FROM accounts;

-- 2. Total Number of Records for Each Table
SELECT 
    table_name,
    record_count
FROM (
    SELECT 'users' AS table_name, COUNT(*) AS record_count FROM users
    UNION ALL SELECT 'accounts', COUNT(*) FROM accounts
    UNION ALL SELECT 'account_access', COUNT(*) FROM account_access
    UNION ALL SELECT 'checking_records', COUNT(*) FROM checking_records
    UNION ALL SELECT 'assets', COUNT(*) FROM assets
    UNION ALL SELECT 'asset_values', COUNT(*) FROM asset_values
    UNION ALL SELECT 'asset_stores', COUNT(*) FROM asset_stores
    UNION ALL SELECT 'asset_store_records', COUNT(*) FROM asset_store_records
    UNION ALL SELECT 'account_assets', COUNT(*) FROM account_assets
    UNION ALL SELECT 'account_snapshots', COUNT(*) FROM account_snapshots
    UNION ALL SELECT 'captchas', COUNT(*) FROM captchas
    UNION ALL SELECT 'subscriptions', COUNT(*) FROM subscriptions
    UNION ALL SELECT 'perspectives', COUNT(*) FROM perspectives
    UNION ALL SELECT 'perspective_settings', COUNT(*) FROM perspective_settings
    UNION ALL SELECT 'perspective_accounts', COUNT(*) FROM perspective_accounts
    UNION ALL SELECT 'perspective_access', COUNT(*) FROM perspective_access
) table_counts
ORDER BY table_name ASC;
