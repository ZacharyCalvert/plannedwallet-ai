package controller

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

// IsValidSHA256Hex returns true if s is a 64-character hexadecimal SHA-256 digest.
func IsValidSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func validatePolicyAgreementFlags(req models.PolicyAgreementRequest) error {
	if req.TermsApproved == nil || !*req.TermsApproved {
		return NewAppError(http.StatusBadRequest, "POLICY_TERMS_NOT_APPROVED", "Terms of Use must be approved.")
	}
	if req.PrivacyApproved == nil || !*req.PrivacyApproved {
		return NewAppError(http.StatusBadRequest, "POLICY_PRIVACY_NOT_APPROVED", "Privacy Policy must be approved.")
	}
	if req.USAdultConsumerConfirmed == nil || !*req.USAdultConsumerConfirmed {
		return NewAppError(http.StatusBadRequest, "POLICY_US_ADULT_CONSUMER_REQUIRED", "You must confirm you are at least 18 years of age and a customer in the United States.")
	}
	return nil
}

func validatePolicyDigest(raw, expected, invalidCode, mismatchCode, fieldLabel, docLabel string) (string, error) {
	sha := strings.ToLower(strings.TrimSpace(raw))
	if !IsValidSHA256Hex(sha) {
		return "", NewAppError(http.StatusBadRequest, invalidCode, fieldLabel+" must be a valid 64-character hex SHA-256 digest.")
	}
	if expected != "" && !strings.EqualFold(sha, expected) {
		return "", NewAppError(http.StatusBadRequest, mismatchCode, "Provided "+fieldLabel+" does not match the current "+docLabel+" policy.")
	}
	return sha, nil
}

func (c *Controller) validatePolicyAgreementRequest(req models.PolicyAgreementRequest) (string, string, error) {
	if err := validatePolicyAgreementFlags(req); err != nil {
		return "", "", err
	}
	termsSha, err := validatePolicyDigest(req.TermsSHA256, c.termsHash, "INVALID_TERMS_HASH", "TERMS_HASH_MISMATCH", "terms_sha256", "Terms of Use")
	if err != nil {
		return "", "", err
	}
	privacySha, err := validatePolicyDigest(req.PrivacySHA256, c.privacyHash, "INVALID_PRIVACY_HASH", "PRIVACY_HASH_MISMATCH", "privacy_sha256", "Privacy & Security")
	if err != nil {
		return "", "", err
	}
	return termsSha, privacySha, nil
}

func toPolicyAgreementResponse(a *models.PolicyAgreement) *models.PolicyAgreementResponse {
	return &models.PolicyAgreementResponse{
		ID:                       a.ID,
		UserID:                   a.UserID,
		TermsSHA256:              a.TermsSHA256,
		PrivacySHA256:            a.PrivacySHA256,
		TermsApproved:            a.TermsApproved,
		PrivacyApproved:          a.PrivacyApproved,
		USAdultConsumerConfirmed: a.USAdultConsumerConfirmed,
		Active:                   a.Active,
		ClientIP:                 a.ClientIP,
		AgreedAt:                 a.AgreedAt,
	}
}

func agreementSatisfiesPolicy(a *models.PolicyAgreement, expectedTermsHash, expectedPrivacyHash string) bool {
	if a == nil || !a.Active {
		return false
	}
	if !a.TermsApproved || !a.PrivacyApproved || !a.USAdultConsumerConfirmed {
		return false
	}
	if expectedTermsHash != "" && !strings.EqualFold(a.TermsSHA256, expectedTermsHash) {
		return false
	}
	return expectedPrivacyHash == "" || strings.EqualFold(a.PrivacySHA256, expectedPrivacyHash)
}

// IsUserPolicyApproved verifies an active ledger entry matching the current policy hashes exists for userID.
func (c *Controller) IsUserPolicyApproved(ctx context.Context, userID string) (bool, error) {
	active, err := c.repo.GetActivePolicyAgreement(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return agreementSatisfiesPolicy(active, c.termsHash, c.privacyHash), nil
}

func (c *Controller) resolveTokenTermsApproved(ctx context.Context, userID string) (bool, error) {
	if !c.enforcePolicy {
		return true, nil
	}
	return c.IsUserPolicyApproved(ctx, userID)
}

// RecordPolicyAgreement records an active policy and 18+ US-customer agreement in the ledger.
func (c *Controller) RecordPolicyAgreement(
	ctx context.Context,
	userID, targetUsername, clientIP string,
	req models.PolicyAgreementRequest,
) (*models.PolicyAgreementResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only record policy agreements for themselves.")
	if err != nil {
		return nil, err
	}
	termsSha, privacySha, err := c.validatePolicyAgreementRequest(req)
	if err != nil {
		return nil, err
	}
	ip := strings.TrimSpace(clientIP)
	if ip == "" {
		ip = "unknown"
	}
	entry := &models.PolicyAgreement{
		ID:                       uuid.New().String(),
		UserID:                   user.ID,
		TermsSHA256:              termsSha,
		PrivacySHA256:            privacySha,
		TermsApproved:            true,
		PrivacyApproved:          true,
		USAdultConsumerConfirmed: true,
		Active:                   true,
		ClientIP:                 ip,
		AgreedAt:                 time.Now().UTC(),
	}
	if err := c.repo.CreatePolicyAgreement(ctx, entry); err != nil {
		trace.LogError(ctx, err, "Failed to persist policy agreement for user %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to record policy agreement.")
	}
	trace.LogInfo(ctx, "Recorded active policy agreement for user %s (ip=%s, terms=%s, privacy=%s)", user.Username, ip, termsSha, privacySha)
	return toPolicyAgreementResponse(entry), nil
}

// ListPolicyAgreements returns all policy agreement ledger entries for the authenticated user.
func (c *Controller) ListPolicyAgreements(ctx context.Context, userID, targetUsername string) ([]*models.PolicyAgreementResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only view their own policy agreements.")
	if err != nil {
		return nil, err
	}
	list, err := c.repo.ListPolicyAgreementsByUserID(ctx, user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve policy agreements.")
	}
	res := make([]*models.PolicyAgreementResponse, 0, len(list))
	for _, a := range list {
		res = append(res, toPolicyAgreementResponse(a))
	}
	return res, nil
}
