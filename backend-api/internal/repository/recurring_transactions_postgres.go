package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/plannedwallet/internal/models"
)

func scanRecurringTransaction(scanner interface{ Scan(...interface{}) error }) (*models.RecurringTransaction, error) {
	var rt models.RecurringTransaction
	var accID, catID, lastApplied, rawSched sql.NullString
	err := scanner.Scan(
		&rt.ID, &rt.UserID, &accID, &catID, &rt.Amount, &lastApplied,
		&rt.Frequency, &rt.Interval, &rawSched, &rt.Name, &rt.Entity, &rt.Description, &rt.Reminder,
		&rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if accID.Valid {
		rt.AccountID = &accID.String
	}
	if catID.Valid {
		rt.CategoryID = &catID.String
	}
	if lastApplied.Valid {
		rt.LastDateApplied = &lastApplied.String
	}
	if rawSched.Valid && rawSched.String != "" {
		var sched models.RecurringSchedule
		if err := json.Unmarshal([]byte(rawSched.String), &sched); err != nil {
			return nil, err
		}
		rt.Schedule = &sched
	}
	return &rt, nil
}

func marshalRecurringSchedule(s *models.RecurringSchedule) (interface{}, error) {
	if s == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

// CreateRecurringTransaction persists a new recurring transaction row in PostgreSQL.
func (p *PostgresRepository) CreateRecurringTransaction(ctx context.Context, rt *models.RecurringTransaction) error {
	schedArg, err := marshalRecurringSchedule(rt.Schedule)
	if err != nil {
		return err
	}
	query := `INSERT INTO recurring_transactions (id, user_id, account_id, category_id, amount, last_date_applied, frequency, interval, schedule, name, entity, description, reminder, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9::jsonb, $10, $11, $12, $13, $14, $15)`
	_, err = p.db.ExecContext(ctx, query,
		rt.ID, rt.UserID, rt.AccountID, rt.CategoryID, rt.Amount,
		rt.LastDateApplied, rt.Frequency, rt.Interval, schedArg, rt.Name, rt.Entity,
		rt.Description, rt.Reminder, rt.CreatedAt, rt.UpdatedAt,
	)
	return err
}

// GetRecurringTransactionByID retrieves a user-owned recurring transaction row or ErrNotFound.
func (p *PostgresRepository) GetRecurringTransactionByID(ctx context.Context, userID, recurringID string) (*models.RecurringTransaction, error) {
	query := `SELECT id, user_id, account_id, category_id, amount, TO_CHAR(last_date_applied, 'YYYY-MM-DD'),
	                 frequency, interval, schedule::text, name, entity, description, reminder, created_at, updated_at
	          FROM recurring_transactions WHERE id = $1 AND user_id = $2`
	rt, err := scanRecurringTransaction(p.db.QueryRowContext(ctx, query, recurringID, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rt, nil
}

// ListRecurringTransactionsByUserID lists all user-owned recurring transactions ordered newest-first.
func (p *PostgresRepository) ListRecurringTransactionsByUserID(ctx context.Context, userID string) ([]*models.RecurringTransaction, error) {
	query := `SELECT id, user_id, account_id, category_id, amount, TO_CHAR(last_date_applied, 'YYYY-MM-DD'),
	                 frequency, interval, schedule::text, name, entity, description, reminder, created_at, updated_at
	          FROM recurring_transactions WHERE user_id = $1 ORDER BY created_at DESC, entity ASC`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*models.RecurringTransaction, 0)
	for rows.Next() {
		rt, err := scanRecurringTransaction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rt)
	}
	return result, rows.Err()
}

// UpdateRecurringTransaction updates an existing user-owned recurring transaction row.
func (p *PostgresRepository) UpdateRecurringTransaction(ctx context.Context, rt *models.RecurringTransaction) error {
	schedArg, err := marshalRecurringSchedule(rt.Schedule)
	if err != nil {
		return err
	}
	query := `UPDATE recurring_transactions
	          SET account_id = $1, category_id = $2, amount = $3, last_date_applied = $4::date,
	              frequency = $5, interval = $6, schedule = $7::jsonb, name = $8, entity = $9, description = $10,
	              reminder = $11, updated_at = $12
	          WHERE id = $13 AND user_id = $14`
	res, err := p.db.ExecContext(ctx, query,
		rt.AccountID, rt.CategoryID, rt.Amount, rt.LastDateApplied,
		rt.Frequency, rt.Interval, schedArg, rt.Name, rt.Entity, rt.Description,
		rt.Reminder, rt.UpdatedAt, rt.ID, rt.UserID,
	)
	if err != nil {
		return err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRecurringTransaction deletes a user-owned recurring transaction row.
func (p *PostgresRepository) DeleteRecurringTransaction(ctx context.Context, userID, recurringID string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM recurring_transactions WHERE id = $1 AND user_id = $2`, recurringID, userID)
	if err != nil {
		return err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return ErrNotFound
	}
	return nil
}

// ApplyRecurringTransactionToAccount inserts the checking record and updates last_date_applied atomically.
func (p *PostgresRepository) ApplyRecurringTransactionToAccount(ctx context.Context, rt *models.RecurringTransaction, record *models.CheckingRecord) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO checking_records (id, account_id, category_id, reconcile_report_id, value, date_sent, date_cleared, entity, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::date, $7::date, $8, $9, $10, $11)`,
		record.ID, record.AccountID, record.CategoryID, record.ReconcileReportID, record.Value,
		record.DateSent, record.DateCleared, record.Entity, record.Description, record.CreatedAt, record.UpdatedAt,
	)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(
		ctx,
		`UPDATE recurring_transactions SET last_date_applied = $1::date, updated_at = $2 WHERE id = $3 AND user_id = $4`,
		rt.LastDateApplied, rt.UpdatedAt, rt.ID, rt.UserID,
	)
	if err != nil {
		return err
	}
	cnt, err := res.RowsAffected()
	if err != nil || cnt == 0 {
		if err != nil {
			return err
		}
		return ErrNotFound
	}
	return tx.Commit()
}
