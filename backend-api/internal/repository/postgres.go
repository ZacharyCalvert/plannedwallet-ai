package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	_, err := p.db.ExecContext(ctx, query, access.ID, access.AccountID, access.UserID, string(access.AccessLevel), access.CreatedAt, access.UpdatedAt)
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
	          WHERE account_id = $1
	          ORDER BY created_at ASC`
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
