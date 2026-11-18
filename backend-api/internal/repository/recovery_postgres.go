package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/plannedwallet/internal/models"
)

func insertRecoveryKeyTx(ctx context.Context, tx *sql.Tx, key *models.UserRecoveryKey) error {
	query := `INSERT INTO user_recovery_keys (id, user_id, key_hash, key_hint, active, created_at, used_at, used_ip)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := tx.ExecContext(ctx, query,
		key.ID, key.UserID, key.KeyHash, key.KeyHint, key.Active, key.CreatedAt, key.UsedAt, key.UsedIP,
	)
	return err
}

func (p *PostgresRepository) CreateUserWithRecoveryKey(ctx context.Context, user *models.User, key *models.UserRecoveryKey) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	userQuery := `INSERT INTO users (id, username, email, password_hash, totp_secret, created_at)
	              VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.ExecContext(ctx, userQuery, user.ID, user.Username, user.Email, user.PasswordHash, user.TOTPSecret, user.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	if err := insertRecoveryKeyTx(ctx, tx, key); err != nil {
		return err
	}
	return tx.Commit()
}

func scanRecoveryKey(scanner interface{ Scan(...any) error }) (*models.UserRecoveryKey, error) {
	var rk models.UserRecoveryKey
	if err := scanner.Scan(&rk.ID, &rk.UserID, &rk.KeyHash, &rk.KeyHint, &rk.Active, &rk.CreatedAt, &rk.UsedAt, &rk.UsedIP); err != nil {
		return nil, err
	}
	return &rk, nil
}

func (p *PostgresRepository) GetActiveRecoveryKeyByUserID(ctx context.Context, userID string) (*models.UserRecoveryKey, error) {
	query := `SELECT id, user_id, key_hash, key_hint, active, created_at, used_at, used_ip
	          FROM user_recovery_keys
	          WHERE user_id = $1 AND active = TRUE
	          ORDER BY created_at DESC LIMIT 1`
	rk, err := scanRecoveryKey(p.db.QueryRowContext(ctx, query, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rk, nil
}

func (p *PostgresRepository) ListRecoveryKeysByUserID(ctx context.Context, userID string) ([]*models.UserRecoveryKey, error) {
	query := `SELECT id, user_id, key_hash, key_hint, active, created_at, used_at, used_ip
	          FROM user_recovery_keys
	          WHERE user_id = $1
	          ORDER BY created_at DESC`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.UserRecoveryKey
	for rows.Next() {
		rk, err := scanRecoveryKey(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rk)
	}
	return result, rows.Err()
}

func (p *PostgresRepository) CompleteAccountRecovery(ctx context.Context, userID, newPasswordHash, newTOTPSecret, consumedKeyID, usedIP string, usedAt time.Time, replacementKey *models.UserRecoveryKey) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE user_recovery_keys SET active = FALSE, used_at = $1, used_ip = $2 WHERE id = $3 AND user_id = $4 AND active = TRUE`,
		usedAt, usedIP, consumedKeyID, userID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = $1, totp_secret = $2 WHERE id = $3`, newPasswordHash, newTOTPSecret, userID); err != nil {
		return err
	}
	if err := insertRecoveryKeyTx(ctx, tx, replacementKey); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresRepository) RotateRecoveryKey(ctx context.Context, userID string, newKey *models.UserRecoveryKey) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE user_recovery_keys SET active = FALSE WHERE user_id = $1 AND active = TRUE`, userID); err != nil {
		return err
	}
	if err := insertRecoveryKeyTx(ctx, tx, newKey); err != nil {
		return err
	}
	return tx.Commit()
}
