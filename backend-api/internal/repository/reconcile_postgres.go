package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/plannedwallet/internal/models"
)

func scanReconcileReport(scanner interface{ Scan(...interface{}) error }) (*models.ReconcileReport, error) {
	var r models.ReconcileReport
	if err := scanner.Scan(&r.ID, &r.AccountID, &r.Name, &r.ReportDate, &r.StartingBalance, &r.EndingBalance, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func lockAndAttachReconcileRecords(ctx context.Context, tx *sql.Tx, report *models.ReconcileReport, recordIDs []string) error {
	for _, recID := range recordIDs {
		var existingReportID sql.NullString
		err := tx.QueryRowContext(
			ctx,
			`SELECT reconcile_report_id FROM checking_records WHERE id = $1 AND account_id = $2 FOR UPDATE`,
			recID, report.AccountID,
		).Scan(&existingReportID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if existingReportID.Valid && existingReportID.String != "" {
			return ErrConflict
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE checking_records
			 SET reconcile_report_id = $1, date_cleared = COALESCE(date_cleared, $2::date), updated_at = $3
			 WHERE id = $4 AND account_id = $5`,
			report.ID, report.ReportDate, report.UpdatedAt, recID, report.AccountID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepository) CreateReconcileReport(ctx context.Context, report *models.ReconcileReport, recordIDs []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO reconcile_reports (id, account_id, name, report_date, starting_balance, ending_balance, created_at, updated_at)
		 VALUES ($1, $2, $3, $4::date, $5, $6, $7, $8)`,
		report.ID, report.AccountID, report.Name, report.ReportDate, report.StartingBalance, report.EndingBalance, report.CreatedAt, report.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if err := lockAndAttachReconcileRecords(ctx, tx, report, recordIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresRepository) listRecordsForAccountReconciled(ctx context.Context, accountID string) (map[string][]*models.CheckingRecord, error) {
	query := `SELECT id, account_id, category_id, reconcile_report_id, value, TO_CHAR(date_sent, 'YYYY-MM-DD'), TO_CHAR(date_cleared, 'YYYY-MM-DD'), entity, description, created_at, updated_at
	          FROM checking_records
	          WHERE account_id = $1 AND reconcile_report_id IS NOT NULL
	          ORDER BY date_sent DESC, created_at DESC`
	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := make(map[string][]*models.CheckingRecord)
	for rows.Next() {
		rec, err := scanCheckingRecord(rows)
		if err != nil {
			return nil, err
		}
		if rec.ReconcileReportID != nil {
			grouped[*rec.ReconcileReportID] = append(grouped[*rec.ReconcileReportID], rec)
		}
	}
	return grouped, rows.Err()
}

func (p *PostgresRepository) GetReconcileReportByID(ctx context.Context, accountID, reportID string) (*models.ReconcileReport, []*models.CheckingRecord, error) {
	query := `SELECT id, account_id, name, TO_CHAR(report_date, 'YYYY-MM-DD'), starting_balance, ending_balance, created_at, updated_at
	          FROM reconcile_reports WHERE id = $1 AND account_id = $2`
	report, err := scanReconcileReport(p.db.QueryRowContext(ctx, query, reportID, accountID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	grouped, err := p.listRecordsForAccountReconciled(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	return report, grouped[report.ID], nil
}

func (p *PostgresRepository) ListReconcileReports(ctx context.Context, accountID string) ([]*models.ReconcileReport, map[string][]*models.CheckingRecord, error) {
	query := `SELECT id, account_id, name, TO_CHAR(report_date, 'YYYY-MM-DD'), starting_balance, ending_balance, created_at, updated_at
	          FROM reconcile_reports WHERE account_id = $1
	          ORDER BY report_date DESC, created_at DESC`
	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	reports := make([]*models.ReconcileReport, 0)
	for rows.Next() {
		rep, err := scanReconcileReport(rows)
		if err != nil {
			return nil, nil, err
		}
		reports = append(reports, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	grouped, err := p.listRecordsForAccountReconciled(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	return reports, grouped, nil
}

func (p *PostgresRepository) DeleteReconcileReport(ctx context.Context, accountID, reportID string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE checking_records SET reconcile_report_id = NULL, updated_at = NOW() WHERE account_id = $1 AND reconcile_report_id = $2`,
		accountID, reportID,
	); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM reconcile_reports WHERE id = $1 AND account_id = $2`, reportID, accountID)
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
	return tx.Commit()
}
