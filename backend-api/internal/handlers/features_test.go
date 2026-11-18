package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
)

func TestFeaturesEndpoint_NormalAndReadOnly(t *testing.T) {
	r, h := setupTestRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newTestRequest(http.MethodGet, "/api/v1/features", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/features, got %d", w.Code)
	}
	var res models.FeaturesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode features: %v", err)
	}
	if len(res.Features) != 0 {
		t.Fatalf("expected empty features slice by default, got %v", res.Features)
	}

	h.SetReadOnly(true)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, newTestRequest(http.MethodGet, "/api/v1/features", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/features in read-only mode, got %d", w2.Code)
	}
	var res2 models.FeaturesResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &res2); err != nil {
		t.Fatalf("failed to decode features: %v", err)
	}
	if len(res2.Features) != 1 || res2.Features[0] != models.FeatureReadOnly {
		t.Fatalf("expected [%q], got %v", models.FeatureReadOnly, res2.Features)
	}
}

func TestReadOnlyMiddleware_BlocksMutationsAllowsReadsAndLogin(t *testing.T) {
	r, h := setupTestRouter()

	// Pre-seed user, account, and record while normal mode is active.
	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: "ro_user1", Email: "ro_user1@example.com", Password: "SecurePass123!",
	})
	wReg := httptest.NewRecorder()
	reqReg := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wReg, reqReg)
	var userRes models.UserResponse
	_ = json.Unmarshal(wReg.Body.Bytes(), &userRes)

	code, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)
	tokBody, _ := json.Marshal(models.TokenExchangeRequest{Password: "SecurePass123!", TOTPCode: code})
	wTok := httptest.NewRecorder()
	reqTok := newTestRequest(http.MethodPost, "/api/v1/users/ro_user1/token", bytes.NewReader(tokBody))
	reqTok.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wTok, reqTok)
	var tokRes models.TokenResponse
	_ = json.Unmarshal(wTok.Body.Bytes(), &tokRes)

	accID := createAccount(t, r, tokRes.AccessToken, "Everyday Checking", "checking", "Checking account")
	val := -45.00
	wRec := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", tokRes.AccessToken, models.CheckingRecordCreateRequest{
		Value: &val, DateSent: "2026-11-01", Entity: "Corner Grocery",
	})
	var rec models.CheckingRecordResponse
	_ = json.Unmarshal(wRec.Body.Bytes(), &rec)

	// Enable read-only maintenance mode.
	h.SetReadOnly(true)

	// Login must still succeed so existing users can inspect data.
	loginCode, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)
	loginBody, _ := json.Marshal(models.TokenExchangeRequest{Password: "SecurePass123!", TOTPCode: loginCode})
	wLogin := httptest.NewRecorder()
	reqLogin := newTestRequest(http.MethodPost, "/api/v1/users/ro_user1/token", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 on login during read-only mode, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	// Valid JWT refresh must also succeed in read-only mode so active sessions stay alive.
	reqRefresh := newTestRequest(http.MethodPost, "/api/v1/users/ro_user1/token/refresh", nil)
	reqRefresh.Header.Set("Authorization", "Bearer "+tokRes.AccessToken)
	wRefresh := httptest.NewRecorder()
	r.ServeHTTP(wRefresh, reqRefresh)
	if wRefresh.Code != http.StatusOK {
		t.Fatalf("expected 200 on token refresh during read-only mode, got %d: %s", wRefresh.Code, wRefresh.Body.String())
	}

	// Read endpoints succeed.
	wList := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accID+"/records", tokRes.AccessToken, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 reading records during read-only mode, got %d", wList.Code)
	}

	// Public user creation and all authenticated write verbs return 503 READ_ONLY_MODE.
	assertReadOnlyBlocked(t, r, http.MethodPost, "/api/v1/users", "", models.UserRegistrationRequest{
		Username: "ro_user2", Email: "ro_user2@example.com", Password: "SecurePass123!",
	})
	assertReadOnlyBlocked(t, r, http.MethodPost, "/api/v1/accounts", tokRes.AccessToken, models.AccountCreationRequest{
		Name: "Blocked Account", Type: "checking",
	})
	assertReadOnlyBlocked(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", tokRes.AccessToken, models.CheckingRecordCreateRequest{
		Value: &val, DateSent: "2026-11-02", Entity: "Coffee Shop",
	})
	assertReadOnlyBlocked(t, r, http.MethodPut, "/api/v1/accounts/"+accID+"/records/"+rec.ID, tokRes.AccessToken, models.CheckingRecordCreateRequest{
		Value: &val, DateSent: "2026-11-02", Entity: "Edited",
	})
	desc := "Patch attempt"
	assertReadOnlyBlocked(t, r, http.MethodPatch, "/api/v1/accounts/"+accID+"/records/"+rec.ID, tokRes.AccessToken, models.CheckingRecordPatchRequest{
		Description: &desc,
	})
	patchAccountName := "Blocked Rename"
	assertReadOnlyBlocked(t, r, http.MethodPatch, "/api/v1/accounts/"+accID, tokRes.AccessToken, models.AccountPatchRequest{
		Name: &patchAccountName,
	})
	assertReadOnlyBlocked(t, r, http.MethodDelete, "/api/v1/accounts/"+accID+"/records/"+rec.ID, tokRes.AccessToken, nil)
	assertReadOnlyBlocked(t, r, http.MethodDelete, "/api/v1/users/ro_user1", tokRes.AccessToken, nil)
}

func assertReadOnlyBlocked(t *testing.T, r http.Handler, method, path, token string, payload interface{}) {
	t.Helper()
	var bodyReader *bytes.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := newTestRequest(method, path, bodyReader)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("%s %s expected 503 during read-only mode, got %d (%s)", method, path, w.Code, w.Body.String())
	}
	var errRes models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errRes); err != nil {
		t.Fatalf("expected JSON ErrorResponse on %s %s: %v", method, path, err)
	}
	if errRes.Error != "READ_ONLY_MODE" {
		t.Fatalf("expected error READ_ONLY_MODE, got %q", errRes.Error)
	}
}
