package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/plannedwallet/internal/models"
)

func scanBill(scanner interface{ Scan(...interface{}) error }) (*models.Bill, error) {
	var b models.Bill
	var accID, catID, lastPaid, rawSched sql.NullString
	err := scanner.Scan(
		&b.ID, &b.UserID, &accID, &catID, &b.Amount, &lastPaid,
		&b.Frequency, &b.Interval, &rawSched, &b.Recipient, &b.Description, &b.Reminder,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if accID.Valid {
		b.AccountID = &accID.String
	}
	if catID.Valid {
		b.CategoryID = &catID.String
	}
	if lastPaid.Valid {
		b.LastDatePaid = &lastPaid.String
	}
	if rawSched.Valid && rawSched.String != "" {
		var sched models.BillSchedule
		if err := json.Unmarshal([]byte(rawSched.String), &sched); err != nil {
			return nil, err
		}
		b.Schedule = &sched
	}
	return &b, nil
}

func marshalBillSchedule(s *models.BillSchedule) (interface{}, error) {
	if s == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (p *PostgresRepository) CreateBill(ctx context.Context, bill *models.Bill) error {
	schedArg, err := marshalBillSchedule(bill.Schedule)
	if err != nil {
		return err
	}
	query := `INSERT INTO bills (id, user_id, account_id, category_id, amount, last_date_paid, frequency, interval, schedule, recipient, description, reminder, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9::jsonb, $10, $11, $12, $13, $14)`
	_, err = p.db.ExecContext(ctx, query,
		bill.ID, bill.UserID, bill.AccountID, bill.CategoryID, bill.Amount,
		bill.LastDatePaid, bill.Frequency, bill.Interval, schedArg, bill.Recipient,
		bill.Description, bill.Reminder, bill.CreatedAt, bill.UpdatedAt,
	)
	return err
}

func (p *PostgresRepository) GetBillByID(ctx context.Context, userID, billID string) (*models.Bill, error) {
	query := `SELECT id, user_id, account_id, category_id, amount, TO_CHAR(last_date_paid, 'YYYY-MM-DD'),
	                 frequency, interval, schedule::text, recipient, description, reminder, created_at, updated_at
	          FROM bills WHERE id = $1 AND user_id = $2`
	bill, err := scanBill(p.db.QueryRowContext(ctx, query, billID, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return bill, nil
}

func (p *PostgresRepository) ListBillsByUserID(ctx context.Context, userID string) ([]*models.Bill, error) {
	query := `SELECT id, user_id, account_id, category_id, amount, TO_CHAR(last_date_paid, 'YYYY-MM-DD'),
	                 frequency, interval, schedule::text, recipient, description, reminder, created_at, updated_at
	          FROM bills WHERE user_id = $1 ORDER BY created_at DESC, recipient ASC`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*models.Bill, 0)
	for rows.Next() {
		bill, err := scanBill(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, bill)
	}
	return result, rows.Err()
}

func (p *PostgresRepository) UpdateBill(ctx context.Context, bill *models.Bill) error {
	schedArg, err := marshalBillSchedule(bill.Schedule)
	if err != nil {
		return err
	}
	query := `UPDATE bills
	          SET account_id = $1, category_id = $2, amount = $3, last_date_paid = $4::date,
	              frequency = $5, interval = $6, schedule = $7::jsonb, recipient = $8, description = $9,
	              reminder = $10, updated_at = $11
	          WHERE id = $12 AND user_id = $13`
	res, err := p.db.ExecContext(ctx, query,
		bill.AccountID, bill.CategoryID, bill.Amount, bill.LastDatePaid,
		bill.Frequency, bill.Interval, schedArg, bill.Recipient, bill.Description,
		bill.Reminder, bill.UpdatedAt, bill.ID, bill.UserID,
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

func (p *PostgresRepository) DeleteBill(ctx context.Context, userID, billID string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM bills WHERE id = $1 AND user_id = $2`, billID, userID)
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

func (p *PostgresRepository) ApplyBillToAccount(ctx context.Context, bill *models.Bill, record *models.CheckingRecord) error {
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
		`UPDATE bills SET last_date_paid = $1::date, updated_at = $2 WHERE id = $3 AND user_id = $4`,
		bill.LastDatePaid, bill.UpdatedAt, bill.ID, bill.UserID,
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
