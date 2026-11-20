package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/plannedwallet/internal/models"
)

func (p *PostgresRepository) CreatePolicyAgreement(ctx context.Context, agreement *models.PolicyAgreement) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE policy_agreements SET active = FALSE WHERE user_id = $1 AND active = TRUE`, agreement.UserID); err != nil {
		return err
	}
	query := `INSERT INTO policy_agreements
	          (id, user_id, terms_sha256, privacy_sha256, terms_approved, privacy_approved, us_adult_consumer_confirmed, active, client_ip, agreed_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	if _, err := tx.ExecContext(
		ctx,
		query,
		agreement.ID,
		agreement.UserID,
		agreement.TermsSHA256,
		agreement.PrivacySHA256,
		agreement.TermsApproved,
		agreement.PrivacyApproved,
		agreement.USAdultConsumerConfirmed,
		agreement.Active,
		agreement.ClientIP,
		agreement.AgreedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func scanPolicyAgreement(scanner interface{ Scan(...any) error }) (*models.PolicyAgreement, error) {
	var a models.PolicyAgreement
	err := scanner.Scan(
		&a.ID,
		&a.UserID,
		&a.TermsSHA256,
		&a.PrivacySHA256,
		&a.TermsApproved,
		&a.PrivacyApproved,
		&a.USAdultConsumerConfirmed,
		&a.Active,
		&a.ClientIP,
		&a.AgreedAt,
	)
	return &a, err
}

func (p *PostgresRepository) GetActivePolicyAgreement(ctx context.Context, userID string) (*models.PolicyAgreement, error) {
	query := `SELECT id, user_id, terms_sha256, privacy_sha256, terms_approved, privacy_approved, us_adult_consumer_confirmed, active, client_ip, agreed_at
	          FROM policy_agreements
	          WHERE user_id = $1 AND active = TRUE
	          ORDER BY agreed_at DESC
	          LIMIT 1`
	a, err := scanPolicyAgreement(p.db.QueryRowContext(ctx, query, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

func (p *PostgresRepository) ListPolicyAgreementsByUserID(ctx context.Context, userID string) ([]*models.PolicyAgreement, error) {
	query := `SELECT id, user_id, terms_sha256, privacy_sha256, terms_approved, privacy_approved, us_adult_consumer_confirmed, active, client_ip, agreed_at
	          FROM policy_agreements
	          WHERE user_id = $1
	          ORDER BY agreed_at DESC`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*models.PolicyAgreement, 0)
	for rows.Next() {
		a, err := scanPolicyAgreement(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (p *PostgresRepository) DeactivateAllPolicyAgreements(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `UPDATE policy_agreements SET active = FALSE WHERE active = TRUE`)
	return err
}
