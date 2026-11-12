package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/plannedwallet/internal/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func (p *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, username, email, password_hash, totp_secret, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := p.db.ExecContext(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash, user.TOTPSecret, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (p *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, created_at
	          FROM users
	          WHERE LOWER(username) = LOWER($1)`
	row := p.db.QueryRowContext(ctx, query, username)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TOTPSecret, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, created_at
	          FROM users
	          WHERE LOWER(email) = LOWER($1)`
	row := p.db.QueryRowContext(ctx, query, email)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TOTPSecret, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, created_at
	          FROM users
	          WHERE id = $1`
	row := p.db.QueryRowContext(ctx, query, id)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.TOTPSecret, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	query := `INSERT INTO accounts (id, admin_id, name, type, description, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := p.db.ExecContext(ctx, query, account.ID, account.AdminID, account.Name, account.Type, account.Description, account.CreatedAt, account.UpdatedAt)
	return err
}

func (p *PostgresRepository) GetAccountByID(ctx context.Context, id string) (*models.Account, error) {
	query := `SELECT id, admin_id, name, type, description, created_at, updated_at
	          FROM accounts
	          WHERE id = $1`
	row := p.db.QueryRowContext(ctx, query, id)
	var a models.Account
	if err := row.Scan(&a.ID, &a.AdminID, &a.Name, &a.Type, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (p *PostgresRepository) ListAccountsByUserID(ctx context.Context, userID string) ([]*models.Account, error) {
	query := `SELECT DISTINCT a.id, a.admin_id, a.name, a.type, a.description, a.created_at, a.updated_at
	          FROM accounts a
	          LEFT JOIN account_access aa ON a.id = aa.account_id
	          WHERE a.admin_id = $1 OR aa.user_id = $1
	          ORDER BY a.created_at ASC`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.AdminID, &a.Name, &a.Type, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, rows.Err()
}

func (p *PostgresRepository) GrantAccountAccess(ctx context.Context, access *models.AccountAccess) error {
	query := `INSERT INTO account_access (id, account_id, user_id, access_level, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          ON CONFLICT (account_id, user_id)
	          DO UPDATE SET access_level = EXCLUDED.access_level, updated_at = EXCLUDED.updated_at`
	_, err := p.db.ExecContext(ctx, query, access.ID, access.AccountID, access.UserID, access.AccessLevel, access.CreatedAt, access.UpdatedAt)
	return err
}

func (p *PostgresRepository) GetAccountAccess(ctx context.Context, accountID, userID string) (*models.AccountAccess, error) {
	query := `SELECT id, account_id, user_id, access_level, created_at, updated_at
	          FROM account_access
	          WHERE account_id = $1 AND user_id = $2`
	row := p.db.QueryRowContext(ctx, query, accountID, userID)
	var a models.AccountAccess
	var lvl string
	if err := row.Scan(&a.ID, &a.AccountID, &a.UserID, &lvl, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.AccessLevel = models.AccessLevel(lvl)
	return &a, nil
}

func (p *PostgresRepository) ListAccountAccess(ctx context.Context, accountID string) ([]*models.AccountAccess, error) {
	query := `SELECT id, account_id, user_id, access_level, created_at, updated_at
	          FROM account_access
	          WHERE account_id = $1`
	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.AccountAccess
	for rows.Next() {
		var a models.AccountAccess
		var lvl string
		if err := rows.Scan(&a.ID, &a.AccountID, &a.UserID, &lvl, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.AccessLevel = models.AccessLevel(lvl)
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (p *PostgresRepository) RevokeAccountAccess(ctx context.Context, accountID, userID string) error {
	query := `DELETE FROM account_access WHERE account_id = $1 AND user_id = $2`
	res, err := p.db.ExecContext(ctx, query, accountID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) CreateCheckingRecord(ctx context.Context, record *models.CheckingRecord) error {
	query := `INSERT INTO checking_records (id, account_id, value, date_sent, date_cleared, entity, description, created_at, updated_at)
	          VALUES ($1, $2, $3, $4::date, $5::date, $6, $7, $8, $9)`
	_, err := p.db.ExecContext(ctx, query,
		record.ID,
		record.AccountID,
		record.Value,
		record.DateSent,
		record.DateCleared,
		record.Entity,
		record.Description,
		record.CreatedAt,
		record.UpdatedAt,
	)
	return err
}

func (p *PostgresRepository) GetCheckingRecordByID(ctx context.Context, accountID, recordID string) (*models.CheckingRecord, error) {
	query := `SELECT id, account_id, value, TO_CHAR(date_sent, 'YYYY-MM-DD'), TO_CHAR(date_cleared, 'YYYY-MM-DD'), entity, description, created_at, updated_at
	          FROM checking_records
	          WHERE id = $1 AND account_id = $2`
	row := p.db.QueryRowContext(ctx, query, recordID, accountID)
	var r models.CheckingRecord
	var dateCleared sql.NullString
	if err := row.Scan(&r.ID, &r.AccountID, &r.Value, &r.DateSent, &dateCleared, &r.Entity, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if dateCleared.Valid {
		val := dateCleared.String
		r.DateCleared = &val
	}
	return &r, nil
}

func (p *PostgresRepository) ListCheckingRecords(ctx context.Context, accountID string, cleared *bool, startDate, endDate *string) ([]*models.CheckingRecord, error) {
	var sb strings.Builder
	sb.WriteString(`SELECT id, account_id, value, TO_CHAR(date_sent, 'YYYY-MM-DD'), TO_CHAR(date_cleared, 'YYYY-MM-DD'), entity, description, created_at, updated_at
	                FROM checking_records
	                WHERE account_id = $1`)

	args := []interface{}{accountID}
	paramIdx := 2

	if cleared != nil {
		if *cleared {
			sb.WriteString(fmt.Sprintf(" AND date_cleared IS NOT NULL"))
		} else {
			sb.WriteString(fmt.Sprintf(" AND date_cleared IS NULL"))
		}
	}
	if startDate != nil && *startDate != "" {
		sb.WriteString(fmt.Sprintf(" AND date_sent >= $%d::date", paramIdx))
		args = append(args, *startDate)
		paramIdx++
	}
	if endDate != nil && *endDate != "" {
		sb.WriteString(fmt.Sprintf(" AND date_sent <= $%d::date", paramIdx))
		args = append(args, *endDate)
		paramIdx++
	}

	sb.WriteString(" ORDER BY date_sent DESC, created_at DESC")

	rows, err := p.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*models.CheckingRecord
	for rows.Next() {
		var r models.CheckingRecord
		var dateCleared sql.NullString
		if err := rows.Scan(&r.ID, &r.AccountID, &r.Value, &r.DateSent, &dateCleared, &r.Entity, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if dateCleared.Valid {
			val := dateCleared.String
			r.DateCleared = &val
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

func (p *PostgresRepository) UpdateCheckingRecord(ctx context.Context, record *models.CheckingRecord) error {
	query := `UPDATE checking_records
	          SET value = $1, date_sent = $2::date, date_cleared = $3::date, entity = $4, description = $5, updated_at = $6
	          WHERE id = $7 AND account_id = $8`
	res, err := p.db.ExecContext(ctx, query,
		record.Value,
		record.DateSent,
		record.DateCleared,
		record.Entity,
		record.Description,
		record.UpdatedAt,
		record.ID,
		record.AccountID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) DeleteCheckingRecord(ctx context.Context, accountID, recordID string) error {
	query := `DELETE FROM checking_records WHERE id = $1 AND account_id = $2`
	res, err := p.db.ExecContext(ctx, query, recordID, accountID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) CreateAsset(ctx context.Context, asset *models.Asset) error {
	query := `INSERT INTO assets (id, user_id, label, description, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := p.db.ExecContext(ctx, query, asset.ID, asset.UserID, asset.Label, asset.Description, asset.CreatedAt, asset.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (p *PostgresRepository) GetAssetByID(ctx context.Context, id string) (*models.Asset, error) {
	query := `SELECT id, user_id, label, description, created_at, updated_at
	          FROM assets
	          WHERE id = $1`
	row := p.db.QueryRowContext(ctx, query, id)
	var a models.Asset
	var userID sql.NullString
	if err := row.Scan(&a.ID, &userID, &a.Label, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if userID.Valid {
		uid := userID.String
		a.UserID = &uid
	}
	return &a, nil
}

func buildAssetsQuery(userID, label *string) (string, []interface{}) {
	var sb strings.Builder
	sb.WriteString(`SELECT id, user_id, label, description, created_at, updated_at FROM assets WHERE 1=1`)
	var args []interface{}
	paramIdx := 1
	if userID != nil && *userID != "" {
		sb.WriteString(fmt.Sprintf(" AND (user_id = $%d OR user_id IS NULL)", paramIdx))
		args = append(args, *userID)
		paramIdx++
	}
	if label != nil && *label != "" {
		sb.WriteString(fmt.Sprintf(" AND LOWER(label) = LOWER($%d)", paramIdx))
		args = append(args, *label)
	}
	sb.WriteString(" ORDER BY LOWER(label) ASC, created_at ASC")
	return sb.String(), args
}

func scanAssetRows(rows *sql.Rows) ([]*models.Asset, error) {
	var records []*models.Asset
	for rows.Next() {
		var a models.Asset
		var uid sql.NullString
		if err := rows.Scan(&a.ID, &uid, &a.Label, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if uid.Valid {
			u := uid.String
			a.UserID = &u
		}
		records = append(records, &a)
	}
	return records, rows.Err()
}

func (p *PostgresRepository) ListAssets(ctx context.Context, userID, label *string) ([]*models.Asset, error) {
	query, args := buildAssetsQuery(userID, label)
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (p *PostgresRepository) UpdateAsset(ctx context.Context, asset *models.Asset) error {
	query := `UPDATE assets
	          SET label = $1, description = $2, updated_at = $3
	          WHERE id = $4`
	res, err := p.db.ExecContext(ctx, query, asset.Label, asset.Description, asset.UpdatedAt, asset.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) DeleteAsset(ctx context.Context, id string) error {
	query := `DELETE FROM assets WHERE id = $1`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) CreateAssetValue(ctx context.Context, av *models.AssetValue) error {
	query := `INSERT INTO asset_values (id, asset_id, value, value_date, created_at, updated_at)
	          VALUES ($1, $2, $3, $4::date, $5, $6)`
	_, err := p.db.ExecContext(ctx, query, av.ID, av.AssetID, av.Value, av.ValueDate, av.CreatedAt, av.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (p *PostgresRepository) GetAssetValueByID(ctx context.Context, assetID, valueID string) (*models.AssetValue, error) {
	query := `SELECT id, asset_id, value, TO_CHAR(value_date, 'YYYY-MM-DD'), created_at, updated_at
	          FROM asset_values
	          WHERE id = $1 AND asset_id = $2`
	row := p.db.QueryRowContext(ctx, query, valueID, assetID)
	var av models.AssetValue
	if err := row.Scan(&av.ID, &av.AssetID, &av.Value, &av.ValueDate, &av.CreatedAt, &av.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &av, nil
}

func buildAssetValuesQuery(assetID string, startDate, endDate *string) (string, []interface{}) {
	var sb strings.Builder
	sb.WriteString(`SELECT id, asset_id, value, TO_CHAR(value_date, 'YYYY-MM-DD'), created_at, updated_at FROM asset_values WHERE asset_id = $1`)
	args := []interface{}{assetID}
	paramIdx := 2
	if startDate != nil && *startDate != "" {
		sb.WriteString(fmt.Sprintf(" AND value_date >= $%d::date", paramIdx))
		args = append(args, *startDate)
		paramIdx++
	}
	if endDate != nil && *endDate != "" {
		sb.WriteString(fmt.Sprintf(" AND value_date <= $%d::date", paramIdx))
		args = append(args, *endDate)
	}
	sb.WriteString(" ORDER BY value_date DESC, created_at DESC")
	return sb.String(), args
}

func scanAssetValueRows(rows *sql.Rows) ([]*models.AssetValue, error) {
	var records []*models.AssetValue
	for rows.Next() {
		var av models.AssetValue
		if err := rows.Scan(&av.ID, &av.AssetID, &av.Value, &av.ValueDate, &av.CreatedAt, &av.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, &av)
	}
	return records, rows.Err()
}

func (p *PostgresRepository) ListAssetValues(ctx context.Context, assetID string, startDate, endDate *string) ([]*models.AssetValue, error) {
	query, args := buildAssetValuesQuery(assetID, startDate, endDate)
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetValueRows(rows)
}

func (p *PostgresRepository) UpdateAssetValue(ctx context.Context, av *models.AssetValue) error {
	query := `UPDATE asset_values
	          SET value = $1, value_date = $2::date, updated_at = $3
	          WHERE id = $4 AND asset_id = $5`
	res, err := p.db.ExecContext(ctx, query, av.Value, av.ValueDate, av.UpdatedAt, av.ID, av.AssetID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) DeleteAssetValue(ctx context.Context, assetID, valueID string) error {
	query := `DELETE FROM asset_values WHERE id = $1 AND asset_id = $2`
	res, err := p.db.ExecContext(ctx, query, valueID, assetID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) AddAccountAsset(ctx context.Context, aa *models.AccountAsset) error {
	query := `INSERT INTO account_assets (id, account_id, asset_id, quantity, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := p.db.ExecContext(ctx, query, aa.ID, aa.AccountID, aa.AssetID, aa.Quantity, aa.CreatedAt, aa.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (p *PostgresRepository) GetAccountAsset(ctx context.Context, accountID, identifier string) (*models.AccountAsset, error) {
	query := `SELECT id, account_id, asset_id, quantity, created_at, updated_at
	          FROM account_assets
	          WHERE account_id = $1 AND (asset_id = $2 OR id = $2)`
	row := p.db.QueryRowContext(ctx, query, accountID, identifier)
	var aa models.AccountAsset
	if err := row.Scan(&aa.ID, &aa.AccountID, &aa.AssetID, &aa.Quantity, &aa.CreatedAt, &aa.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &aa, nil
}

func scanAccountAssetDetailRows(rows *sql.Rows) ([]*models.AccountAssetDetail, error) {
	var details []*models.AccountAssetDetail
	for rows.Next() {
		var d models.AccountAssetDetail
		var valDate sql.NullString
		err := rows.Scan(
			&d.ID, &d.AccountID, &d.AssetID, &d.Quantity, &d.CreatedAt, &d.UpdatedAt,
			&d.Label, &d.Description, &d.LatestValue, &valDate,
		)
		if err != nil {
			return nil, err
		}
		if valDate.Valid {
			d.LatestValueDate = valDate.String
		}
		details = append(details, &d)
	}
	return details, rows.Err()
}

func (p *PostgresRepository) ListAccountAssets(ctx context.Context, accountID string) ([]*models.AccountAssetDetail, error) {
	query := `SELECT 
	            aa.id, aa.account_id, aa.asset_id, aa.quantity, aa.created_at, aa.updated_at,
	            a.label, a.description,
	            COALESCE(latest_val.value, 0.0) AS latest_value,
	            TO_CHAR(latest_val.value_date, 'YYYY-MM-DD') AS latest_value_date
	          FROM account_assets aa
	          JOIN assets a ON aa.asset_id = a.id
	          LEFT JOIN LATERAL (
	            SELECT av.value, av.value_date
	            FROM asset_values av
	            WHERE av.asset_id = aa.asset_id
	            ORDER BY av.value_date DESC, av.created_at DESC
	            LIMIT 1
	          ) latest_val ON true
	          WHERE aa.account_id = $1
	          ORDER BY LOWER(a.label) ASC`
	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccountAssetDetailRows(rows)
}

func (p *PostgresRepository) UpdateAccountAsset(ctx context.Context, aa *models.AccountAsset) error {
	query := `UPDATE account_assets
	          SET quantity = $1, updated_at = $2
	          WHERE id = $3 AND account_id = $4`
	res, err := p.db.ExecContext(ctx, query, aa.Quantity, aa.UpdatedAt, aa.ID, aa.AccountID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) DeleteAccountAsset(ctx context.Context, accountID, identifier string) error {
	query := `DELETE FROM account_assets WHERE account_id = $1 AND (asset_id = $2 OR id = $2)`
	res, err := p.db.ExecContext(ctx, query, accountID, identifier)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) CreateAccountSnapshot(ctx context.Context, snapshot *models.AccountSnapshot) error {
	query := `INSERT INTO account_snapshots (id, account_id, value, snapshot_date, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := p.db.ExecContext(ctx, query, snapshot.ID, snapshot.AccountID, snapshot.Value, snapshot.SnapshotDate, snapshot.CreatedAt, snapshot.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (p *PostgresRepository) GetAccountSnapshotByID(ctx context.Context, accountID, snapshotID string) (*models.AccountSnapshot, error) {
	query := `SELECT id, account_id, value, snapshot_date, created_at, updated_at
	          FROM account_snapshots
	          WHERE account_id = $1 AND id = $2`
	var s models.AccountSnapshot
	var snapDate time.Time
	err := p.db.QueryRowContext(ctx, query, accountID, snapshotID).Scan(
		&s.ID, &s.AccountID, &s.Value, &snapDate, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.SnapshotDate = snapDate.Format("2006-01-02")
	return &s, nil
}

func (p *PostgresRepository) ListAccountSnapshots(ctx context.Context, accountID string, startDate, endDate *string) ([]*models.AccountSnapshot, error) {
	var sb strings.Builder
	sb.WriteString(`SELECT id, account_id, value, snapshot_date, created_at, updated_at FROM account_snapshots WHERE account_id = $1`)
	args := []interface{}{accountID}
	paramIdx := 2
	if startDate != nil && *startDate != "" {
		sb.WriteString(fmt.Sprintf(" AND snapshot_date >= $%d::date", paramIdx))
		args = append(args, *startDate)
		paramIdx++
	}
	if endDate != nil && *endDate != "" {
		sb.WriteString(fmt.Sprintf(" AND snapshot_date <= $%d::date", paramIdx))
		args = append(args, *endDate)
	}
	sb.WriteString(" ORDER BY snapshot_date ASC, created_at ASC")

	rows, err := p.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.AccountSnapshot
	for rows.Next() {
		var s models.AccountSnapshot
		var snapDate time.Time
		if err := rows.Scan(&s.ID, &s.AccountID, &s.Value, &snapDate, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.SnapshotDate = snapDate.Format("2006-01-02")
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (p *PostgresRepository) UpdateAccountSnapshot(ctx context.Context, snapshot *models.AccountSnapshot) error {
	query := `UPDATE account_snapshots
	          SET value = $1, snapshot_date = $2, updated_at = $3
	          WHERE account_id = $4 AND id = $5`
	res, err := p.db.ExecContext(ctx, query, snapshot.Value, snapshot.SnapshotDate, snapshot.UpdatedAt, snapshot.AccountID, snapshot.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) DeleteAccountSnapshot(ctx context.Context, accountID, snapshotID string) error {
	query := `DELETE FROM account_snapshots WHERE account_id = $1 AND id = $2`
	res, err := p.db.ExecContext(ctx, query, accountID, snapshotID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) SetCaptcha(ctx context.Context, id string, digits []byte) error {
	query := `INSERT INTO captchas (id, digits, created_at)
	          VALUES ($1, $2, NOW())
	          ON CONFLICT (id) DO UPDATE SET digits = $2, created_at = NOW()`
	_, err := p.db.ExecContext(ctx, query, id, digits)
	return err
}

func (p *PostgresRepository) GetCaptcha(ctx context.Context, id string, clear bool) ([]byte, error) {
	if clear {
		query := `DELETE FROM captchas WHERE id = $1 AND created_at >= NOW() - INTERVAL '5 minutes' RETURNING digits`
		var digits []byte
		err := p.db.QueryRowContext(ctx, query, id).Scan(&digits)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		return digits, nil
	}
	query := `SELECT digits FROM captchas WHERE id = $1 AND created_at >= NOW() - INTERVAL '5 minutes'`
	var digits []byte
	err := p.db.QueryRowContext(ctx, query, id).Scan(&digits)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return digits, nil
}

func (p *PostgresRepository) DeleteExpiredCaptchas(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	query := `DELETE FROM captchas WHERE created_at < $1`
	res, err := p.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
