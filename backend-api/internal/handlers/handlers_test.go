package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/captcha"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/cors"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func newTestRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-Client-ID", "test-client")
	return req
}

func setupTestRouter() (*gin.Engine, *handlers.Handler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(trace.Middleware())
	r.Use(cors.Middleware())
	r.Use(gin.Recovery())

	repo := repository.NewMemoryRepository()
	captcha.SetCustomStore(captcha.NewRepoStore(repo))
	authService := auth.NewService("test-secret-key-12345", auth.DefaultExpirationSeconds)
	ctrl := controller.NewController(repo, authService)
	h := handlers.NewHandler(ctrl, authService)
	h.RegisterRoutes(r)
	return r, h
}

func TestHelloEndpoint(t *testing.T) {
	r, _ := setupTestRouter()
	req := newTestRequest(http.MethodGet, "/api/v1/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/hello, got %d", w.Code)
	}
	if w.Body.String() != "world" {
		t.Fatalf("expected 'world', got %q", w.Body.String())
	}
	if w.Header().Get("traceparent") == "" {
		t.Fatalf("expected traceparent header in response")
	}
}

func TestHealthEndpoint(t *testing.T) {
	r, _ := setupTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/health, got %d (body: %s)", w.Code, w.Body.String())
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "up" {
		t.Fatalf("expected status: up, got %q", res["status"])
	}
}

func TestMissingClientID_Rejected(t *testing.T) {
	r, _ := setupTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing X-Client-ID, got %d", w.Code)
	}
	if w.Header().Get("traceparent") == "" {
		t.Fatalf("expected traceparent header on error response")
	}
}

func registerAndGetToken(t *testing.T, r *gin.Engine, username, email, password string) string {
	t.Helper()

	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	req := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 during registration, got %d: %s", w.Code, w.Body.String())
	}

	var userRes models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &userRes)
	totpCode, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)

	tokenBody, _ := json.Marshal(models.TokenExchangeRequest{
		Password: password,
		TOTPCode: totpCode,
	})
	tokenReq := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/token", username), bytes.NewReader(tokenBody))
	tokenReq.Header.Set("Content-Type", "application/json")
	tw := httptest.NewRecorder()
	r.ServeHTTP(tw, tokenReq)
	if tw.Code != http.StatusOK {
		t.Fatalf("expected 200 during token exchange, got %d: %s", tw.Code, tw.Body.String())
	}

	var tokenRes models.TokenResponse
	if err := json.Unmarshal(tw.Body.Bytes(), &tokenRes); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	return tokenRes.AccessToken
}

func createAccount(t *testing.T, r *gin.Engine, token, name, accType, description string) string {
	t.Helper()

	body, _ := json.Marshal(models.AccountCreationRequest{
		Name:        name,
		Type:        accType,
		Description: description,
	})
	req := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 during account creation, got %d: %s", w.Code, w.Body.String())
	}

	var res models.AccountResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse account response: %v", err)
	}
	if res.Name != name {
		t.Fatalf("expected account name %q, got %q", name, res.Name)
	}
	return res.ID
}

func TestUserRegistration_ValidationAndConflict(t *testing.T) {
	r, _ := setupTestRouter()

	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: "alice_wonder",
		Email:    "alice@example.com",
		Password: "SecurePassword123!",
	})
	req := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var userRes models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &userRes)
	if userRes.Username != "alice_wonder" || userRes.Email != "alice@example.com" {
		t.Errorf("unexpected user data: %+v", userRes)
	}
	if userRes.TOTPSecret == "" || userRes.TOTPURL == "" {
		t.Errorf("expected TOTP secret and URL in response: %+v", userRes)
	}

	// Duplicate registration conflict (409)
	reqDup := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)
	if wDup.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", wDup.Code, wDup.Body.String())
	}
}

func TestUserRegistration_PasswordRules(t *testing.T) {
	r, _ := setupTestRouter()

	invalidPasswords := []struct {
		desc string
		pw   string
	}{
		{"less than 10 characters", "Short1!"},
		{"no uppercase", "lowercase123!"},
		{"no lowercase", "UPPERCASE123!"},
		{"no number", "NoNumbersHere!"},
		{"no special character", "NoSpecialChar123"},
	}

	for _, tc := range invalidPasswords {
		t.Run("password_"+tc.desc, func(t *testing.T) {
			badPwBody, _ := json.Marshal(models.UserRegistrationRequest{
				Username: "bob_builder",
				Email:    "bob@example.com",
				Password: tc.pw,
			})
			reqBadPw := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(badPwBody))
			reqBadPw.Header.Set("Content-Type", "application/json")
			wBadPw := httptest.NewRecorder()
			r.ServeHTTP(wBadPw, reqBadPw)
			if wBadPw.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for password rule (%s), got %d: %s", tc.desc, wBadPw.Code, wBadPw.Body.String())
			}
		})
	}
}

func TestTokenExchange_TOTPAndAuthentication(t *testing.T) {
	r, _ := setupTestRouter()

	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: "auth_user",
		Email:    "auth_user@example.com",
		Password: "SecurePassword123!",
	})
	reqReg := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	r.ServeHTTP(wReg, reqReg)

	var userRes models.UserResponse
	_ = json.Unmarshal(wReg.Body.Bytes(), &userRes)
	totpCode, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)

	// 1. Success with valid password and TOTP
	tokBody, _ := json.Marshal(models.TokenExchangeRequest{
		Password: "SecurePassword123!",
		TOTPCode: totpCode,
	})
	reqTok := newTestRequest(http.MethodPost, "/api/v1/users/auth_user/token", bytes.NewReader(tokBody))
	reqTok.Header.Set("Content-Type", "application/json")
	wTok := httptest.NewRecorder()
	r.ServeHTTP(wTok, reqTok)
	if wTok.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wTok.Code, wTok.Body.String())
	}

	// 2. Wrong TOTP code
	badTotpBody, _ := json.Marshal(models.TokenExchangeRequest{
		Password: "SecurePassword123!",
		TOTPCode: "000000",
	})
	reqBadTotp := newTestRequest(http.MethodPost, "/api/v1/users/auth_user/token", bytes.NewReader(badTotpBody))
	reqBadTotp.Header.Set("Content-Type", "application/json")
	wBadTotp := httptest.NewRecorder()
	r.ServeHTTP(wBadTotp, reqBadTotp)
	if wBadTotp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad TOTP, got %d", wBadTotp.Code)
	}
}

func TestRefreshTokenEndpoint(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "ref_user", "ref@example.com", "Password123!")

	req := newTestRequest(http.MethodPost, "/api/v1/users/ref_user/token/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 refreshing valid JWT, got %d: %s", w.Code, w.Body.String())
	}
	var tokRes models.TokenResponse
	_ = json.Unmarshal(w.Body.Bytes(), &tokRes)
	if tokRes.ExpiresIn != 900 || tokRes.TokenType != "Bearer" || tokRes.AccessToken == "" || tokRes.AccessToken == token {
		t.Fatalf("unexpected refresh response: %+v", tokRes)
	}

	reqAcc := newTestRequest(http.MethodGet, "/api/v1/accounts", nil)
	reqAcc.Header.Set("Authorization", "Bearer "+tokRes.AccessToken)
	wAcc := httptest.NewRecorder()
	r.ServeHTTP(wAcc, reqAcc)
	if wAcc.Code != http.StatusOK {
		t.Fatalf("expected refreshed token to access protected route, got %d", wAcc.Code)
	}

	reqOther := newTestRequest(http.MethodPost, "/api/v1/users/someone_else/token/refresh", nil)
	reqOther.Header.Set("Authorization", "Bearer "+token)
	wOther := httptest.NewRecorder()
	r.ServeHTTP(wOther, reqOther)
	if wOther.Code != http.StatusForbidden {
		t.Fatalf("expected 403 refreshing another username, got %d", wOther.Code)
	}

	reqUnauth := newTestRequest(http.MethodPost, "/api/v1/users/ref_user/token/refresh", nil)
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without JWT, got %d", wUnauth.Code)
	}
}

func TestAccountCreation(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "testuser1", "testuser1@example.com", "Password123!")

	// 1. Success creating checking account
	body, _ := json.Marshal(models.AccountCreationRequest{
		Name:        "Everyday Checking",
		Type:        "checking",
		Description: "Primary Checking",
	})
	req := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var accRes models.AccountResponse
	if err := json.Unmarshal(w.Body.Bytes(), &accRes); err != nil {
		t.Fatalf("failed to unmarshal account response: %v", err)
	}
	if accRes.Name != "Everyday Checking" || accRes.Type != "checking" || accRes.AdminID == "" {
		t.Errorf("unexpected account response: %+v", accRes)
	}

	// 2. Missing account name (400)
	missingNameBody, _ := json.Marshal(models.AccountCreationRequest{
		Name:        "",
		Type:        "checking",
		Description: "Primary Checking",
	})
	reqMissingName := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(missingNameBody))
	reqMissingName.Header.Set("Content-Type", "application/json")
	reqMissingName.Header.Set("Authorization", "Bearer "+token)
	wMissingName := httptest.NewRecorder()
	r.ServeHTTP(wMissingName, reqMissingName)
	if wMissingName.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing account name, got %d: %s", wMissingName.Code, wMissingName.Body.String())
	}

	// 3. Invalid account type
	badBody, _ := json.Marshal(models.AccountCreationRequest{
		Name:        "Bad Account",
		Type:        "unsupported_type",
		Description: "Bad Account",
	})
	reqBad := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(badBody))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("Authorization", "Bearer "+token)
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad account type, got %d: %s", wBad.Code, wBad.Body.String())
	}

	// 4. Unauthorized without token (401)
	reqNoAuth := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	reqNoAuth.Header.Set("Content-Type", "application/json")
	wNoAuth := httptest.NewRecorder()
	r.ServeHTTP(wNoAuth, reqNoAuth)
	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", wNoAuth.Code, wNoAuth.Body.String())
	}
}

func TestListAndGetAccounts(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "accuser", "accuser@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Everyday Checking", "checking", "My Everyday Checking")

	listReq := newTestRequest(http.MethodGet, "/api/v1/accounts", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, listReq)
	if lw.Code != http.StatusOK {
		t.Fatalf("expected 200 on list accounts, got %d: %s", lw.Code, lw.Body.String())
	}
	var accounts []*models.AccountResponse
	_ = json.Unmarshal(lw.Body.Bytes(), &accounts)
	if len(accounts) != 1 || accounts[0].ID != accountID {
		t.Fatalf("unexpected accounts list: %+v", accounts)
	}

	getReq := newTestRequest(http.MethodGet, "/api/v1/accounts/"+accountID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, getReq)
	if gw.Code != http.StatusOK {
		t.Fatalf("expected 200 on get account, got %d: %s", gw.Code, gw.Body.String())
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "owneruser", "owner@example.com", "Password123!")
	otherToken := registerAndGetToken(t, r, "otheruser", "other@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Everyday Checking", "checking", "My Everyday Checking")

	// Non-admin attempt to delete should return 403 Forbidden
	reqForbidden := newTestRequest(http.MethodDelete, "/api/v1/accounts/"+accountID, nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+otherToken)
	wForbidden := httptest.NewRecorder()
	r.ServeHTTP(wForbidden, reqForbidden)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when non-admin deletes account, got %d: %s", wForbidden.Code, wForbidden.Body.String())
	}

	// Admin delete should return 204 No Content
	reqDelete := newTestRequest(http.MethodDelete, "/api/v1/accounts/"+accountID, nil)
	reqDelete.Header.Set("Authorization", "Bearer "+token)
	wDelete := httptest.NewRecorder()
	r.ServeHTTP(wDelete, reqDelete)
	if wDelete.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete account, got %d: %s", wDelete.Code, wDelete.Body.String())
	}

	// Verify account is now 404
	reqGet := newTestRequest(http.MethodGet, "/api/v1/accounts/"+accountID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after account deletion, got %d", wGet.Code)
	}
}

func TestCheckingRecordsCRUD(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "checkinguser", "checking@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Everyday Checking", "checking", "My Everyday Checking")

	// 1. Create a pending checking record (date_cleared is nil)
	val1 := -85.25
	createPayload, _ := json.Marshal(models.CheckingRecordCreateRequest{
		Value:       &val1,
		DateSent:    "2026-11-01",
		DateCleared: nil,
		Entity:      "Supermarket",
		Description: "Household supplies",
	})
	req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), bytes.NewReader(createPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on record create, got %d: %s", w.Code, w.Body.String())
	}

	var rec1 models.CheckingRecordResponse
	if err := json.Unmarshal(w.Body.Bytes(), &rec1); err != nil {
		t.Fatalf("failed to parse record response: %v", err)
	}
	if rec1.ID == "" || rec1.Value != -85.25 || rec1.DateCleared != nil || rec1.Entity != "Supermarket" {
		t.Errorf("unexpected record1 data: %+v", rec1)
	}

	// 2. Create a second cleared checking record
	val2 := 2500.00
	dateCleared := "2026-11-03"
	createPayload2, _ := json.Marshal(models.CheckingRecordCreateRequest{
		Value:       &val2,
		DateSent:    "2026-11-03",
		DateCleared: &dateCleared,
		Entity:      "Employer Payroll",
		Description: "Bi-weekly paycheck",
	})
	req2 := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), bytes.NewReader(createPayload2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 on second record create, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. List checking records (unfiltered)
	listReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, listReq)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d: %s", wList.Code, wList.Body.String())
	}
	var listRes []models.CheckingRecordResponse
	_ = json.Unmarshal(wList.Body.Bytes(), &listRes)
	if len(listRes) != 2 {
		t.Errorf("expected 2 records, got %d", len(listRes))
	}

	// 4. List with query filter: cleared=false
	filterReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/records?cleared=false", accountID), nil)
	filterReq.Header.Set("Authorization", "Bearer "+token)
	wFilter := httptest.NewRecorder()
	r.ServeHTTP(wFilter, filterReq)
	if wFilter.Code != http.StatusOK {
		t.Fatalf("expected 200 on filtered list, got %d: %s", wFilter.Code, wFilter.Body.String())
	}
	var pendingList []models.CheckingRecordResponse
	_ = json.Unmarshal(wFilter.Body.Bytes(), &pendingList)
	if len(pendingList) != 1 || pendingList[0].ID != rec1.ID {
		t.Errorf("expected 1 pending record (Supermarket), got: %+v", pendingList)
	}

	// 5. Get checking record by ID
	getReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/records/%s", accountID, rec1.ID), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 on get record, got %d: %s", wGet.Code, wGet.Body.String())
	}

	// 6. Patch record to mark as cleared
	clearedDate := "2026-11-05"
	patchBody, _ := json.Marshal(models.CheckingRecordPatchRequest{
		DateCleared: &clearedDate,
	})
	patchReq := newTestRequest(http.MethodPatch, fmt.Sprintf("/api/v1/accounts/%s/records/%s", accountID, rec1.ID), bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+token)
	wPatch := httptest.NewRecorder()
	r.ServeHTTP(wPatch, patchReq)
	if wPatch.Code != http.StatusOK {
		t.Fatalf("expected 200 on patch, got %d: %s", wPatch.Code, wPatch.Body.String())
	}
	var patchedRec models.CheckingRecordResponse
	_ = json.Unmarshal(wPatch.Body.Bytes(), &patchedRec)
	if patchedRec.DateCleared == nil || *patchedRec.DateCleared != "2026-11-05" {
		t.Errorf("expected date_cleared to be 2026-11-05, got %+v", patchedRec.DateCleared)
	}

	// 7. Put record (full update)
	updatedVal := -90.00
	putBody, _ := json.Marshal(models.CheckingRecordUpdateRequest{
		Value:       &updatedVal,
		DateSent:    "2026-11-01",
		DateCleared: &clearedDate,
		Entity:      "Supermarket Plus",
		Description: "Groceries and clothes",
	})
	putReq := newTestRequest(http.MethodPut, fmt.Sprintf("/api/v1/accounts/%s/records/%s", accountID, rec1.ID), bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)
	wPut := httptest.NewRecorder()
	r.ServeHTTP(wPut, putReq)
	if wPut.Code != http.StatusOK {
		t.Fatalf("expected 200 on put, got %d: %s", wPut.Code, wPut.Body.String())
	}
	var putRec models.CheckingRecordResponse
	_ = json.Unmarshal(wPut.Body.Bytes(), &putRec)
	if putRec.Value != -90.00 || putRec.Entity != "Supermarket Plus" {
		t.Errorf("unexpected updated record: %+v", putRec)
	}

	// 8. Delete record
	delReq := newTestRequest(http.MethodDelete, fmt.Sprintf("/api/v1/accounts/%s/records/%s", accountID, rec1.ID), nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, delReq)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// 9. Verify record is deleted (404)
	getDeletedReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/records/%s", accountID, rec1.ID), nil)
	getDeletedReq.Header.Set("Authorization", "Bearer "+token)
	wDeleted := httptest.NewRecorder()
	r.ServeHTTP(wDeleted, getDeletedReq)
	if wDeleted.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted record, got %d: %s", wDeleted.Code, wDeleted.Body.String())
	}
}

func TestSavingsAccountAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "savingsuser", "savings@example.com", "Password123!")
	accountID := createAccount(t, r, token, "High Yield Savings", "savings", "Emergency Fund")

	val := 1500.00
	createPayload, _ := json.Marshal(models.CheckingRecordCreateRequest{
		Value:       &val,
		DateSent:    "2026-11-01",
		Entity:      "Deposit",
		Description: "Initial deposit",
	})
	req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), bytes.NewReader(createPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on savings record create, got %d: %s", w.Code, w.Body.String())
	}

	listReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, listReq)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on savings records list, got %d: %s", wList.Code, wList.Body.String())
	}
}

func TestCreditCardAccountAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "ccuser", "cc@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Premier Card", "credit card", "Rewards Card")

	val := -75.00
	createPayload, _ := json.Marshal(models.CheckingRecordCreateRequest{
		Value:       &val,
		DateSent:    "2026-11-01",
		Entity:      "Gas Station",
		Description: "Fuel",
	})
	req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/records", accountID), bytes.NewReader(createPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on cc record create, got %d: %s", w.Code, w.Body.String())
	}
}

func createTestAsset(t *testing.T, r *gin.Engine, token, label, desc string) string {
	t.Helper()
	body, _ := json.Marshal(models.AssetCreateRequest{Label: label, Description: desc})
	req := newTestRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on asset create, got %d: %s", w.Code, w.Body.String())
	}
	var res models.AssetResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return res.ID
}

func TestAssetsAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "assetapi_u", "apiu@example.com", "Password123!")
	assetID := createTestAsset(t, r, token, "btc", "Bitcoin")

	// Verify conflict on duplicate label
	body, _ := json.Marshal(models.AssetCreateRequest{Label: "BTC", Description: "Duplicate"})
	req := newTestRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d: %s", w.Code, w.Body.String())
	}

	getReq := newTestRequest(http.MethodGet, "/api/v1/assets/"+assetID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, getReq)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 on get asset, got %d", wGet.Code)
	}
}

func TestAssetValuesMultiDayAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "multival_u", "multival@example.com", "Password123!")
	assetID := createTestAsset(t, r, token, "eth", "Ethereum")

	dates := []string{"2026-11-09", "2026-11-10", "2026-11-11"}
	for _, d := range dates {
		val := 3200.0
		body, _ := json.Marshal(models.AssetValueCreateRequest{Value: &val, ValueDate: d})
		req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/assets/%s/values", assetID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 on asset value create for %s, got %d: %s", d, w.Code, w.Body.String())
		}
	}

	// Duplicate date conflict test
	dupVal := 3250.0
	body, _ := json.Marshal(models.AssetValueCreateRequest{Value: &dupVal, ValueDate: "2026-11-10"})
	req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/assets/%s/values", assetID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, req)
	if wDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict on duplicate date, got %d", wDup.Code)
	}
}

func TestAccountAssetsAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "pm_user", "pm@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Precious Metals", "asset", "Metals Holdings")
	assetID := createTestAsset(t, r, token, "slv-oz", "Silver Ounce")

	// Set valuation
	val := 32.50
	valBody, _ := json.Marshal(models.AssetValueCreateRequest{Value: &val, ValueDate: "2026-11-10"})
	reqVal := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/assets/%s/values", assetID), bytes.NewReader(valBody))
	reqVal.Header.Set("Content-Type", "application/json")
	reqVal.Header.Set("Authorization", "Bearer "+token)
	wVal := httptest.NewRecorder()
	r.ServeHTTP(wVal, reqVal)

	// Add asset to account
	qty := 45.209
	addBody, _ := json.Marshal(models.AccountAssetAddRequest{AssetID: assetID, Quantity: &qty})
	reqAdd := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/assets", accountID), bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.Header.Set("Authorization", "Bearer "+token)
	wAdd := httptest.NewRecorder()
	r.ServeHTTP(wAdd, reqAdd)
	if wAdd.Code != http.StatusCreated {
		t.Fatalf("expected 201 on add account asset, got %d: %s", wAdd.Code, wAdd.Body.String())
	}

	// List assets in account
	reqList := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/assets", accountID), nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on list account assets, got %d", wList.Code)
	}

	// Summary endpoint
	reqSum := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/assets/summary", accountID), nil)
	reqSum.Header.Set("Authorization", "Bearer "+token)
	wSum := httptest.NewRecorder()
	r.ServeHTTP(wSum, reqSum)
	if wSum.Code != http.StatusOK {
		t.Fatalf("expected 200 on asset account summary, got %d", wSum.Code)
	}
}

func TestAccountAccessAPI(t *testing.T) {
	r, _ := setupTestRouter()
	tokenA := registerAndGetToken(t, r, "access_admin", "access_admin@example.com", "Password123!")

	regB, _ := json.Marshal(models.UserRegistrationRequest{Username: "collab_b", Email: "collab_b@example.com", Password: "Password123!"})
	reqB := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regB))
	reqB.Header.Set("Content-Type", "application/json")
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	var userB models.UserResponse
	_ = json.Unmarshal(wB.Body.Bytes(), &userB)

	accID := createAccount(t, r, tokenA, "Shared Vault", "asset", "Shared Holdings")

	grantBody, _ := json.Marshal(models.AccountAccessGrantRequest{UserID: userB.ID, AccessLevel: models.AccessLevelRead})
	reqGrant := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/access", accID), bytes.NewReader(grantBody))
	reqGrant.Header.Set("Content-Type", "application/json")
	reqGrant.Header.Set("Authorization", "Bearer "+tokenA)
	wGrant := httptest.NewRecorder()
	r.ServeHTTP(wGrant, reqGrant)
	if wGrant.Code != http.StatusOK {
		t.Fatalf("expected 200 on grant access, got %d: %s", wGrant.Code, wGrant.Body.String())
	}

	reqRevoke := newTestRequest(http.MethodDelete, fmt.Sprintf("/api/v1/accounts/%s/access/%s", accID, userB.ID), nil)
	reqRevoke.Header.Set("Authorization", "Bearer "+tokenA)
	wRevoke := httptest.NewRecorder()
	r.ServeHTTP(wRevoke, reqRevoke)
	if wRevoke.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on revoke access, got %d", wRevoke.Code)
	}
}

func TestAccountSnapshotsAPI(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "snapshot_api_user", "snapshot_api@example.com", "Password123!")
	accountID := createAccount(t, r, token, "Day Trading", "snapshot", "Stock trading portfolio")

	// Create snapshot
	val := 1200.00
	snapBody, _ := json.Marshal(models.AccountSnapshotCreateRequest{Value: &val, SnapshotDate: "2026-11-11"})
	req := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/snapshots", accountID), bytes.NewReader(snapBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on snapshot create, got %d: %s", w.Code, w.Body.String())
	}

	// List snapshots
	reqList := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/snapshots", accountID), nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on list snapshots, got %d: %s", wList.Code, wList.Body.String())
	}
}

func TestCaptchaEndpoints(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. GET /api/v1/captcha
	req := newTestRequest(http.MethodGet, "/api/v1/captcha", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/captcha, got %d: %s", w.Code, w.Body.String())
	}
	var captchaRes models.CaptchaResponse
	if err := json.Unmarshal(w.Body.Bytes(), &captchaRes); err != nil {
		t.Fatalf("failed to unmarshal captcha response: %v", err)
	}
	if captchaRes.ID == "" || !strings.HasPrefix(captchaRes.Image, "data:image/png;base64,") {
		t.Errorf("unexpected captcha response: %+v", captchaRes)
	}

	// 2. GET /api/v1/captcha/:id.png
	imgReq := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/captcha/%s.png", captchaRes.ID), nil)
	imgW := httptest.NewRecorder()
	r.ServeHTTP(imgW, imgReq)
	if imgW.Code != http.StatusOK {
		t.Fatalf("expected 200 for captcha image, got %d", imgW.Code)
	}
	if imgW.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png content type, got: %s", imgW.Header().Get("Content-Type"))
	}

	// 3. GET /api/v1/captcha/nonexistent.png -> 404
	errReq := newTestRequest(http.MethodGet, "/api/v1/captcha/nonexistent.png", nil)
	errW := httptest.NewRecorder()
	r.ServeHTTP(errW, errReq)
	if errW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid captcha image, got %d", errW.Code)
	}
}

func TestRegisterWithInvalidCaptcha(t *testing.T) {
	r, _ := setupTestRouter()
	body, _ := json.Marshal(models.UserRegistrationRequest{
		Username:        "captcha_user",
		Email:           "captcha_user@example.com",
		Password:        "Password123!",
		CaptchaID:       "some-challenge-id",
		CaptchaSolution: "wrong-solution",
	})
	req := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid captcha, got %d: %s", w.Code, w.Body.String())
	}
	var errRes models.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errRes)
	if errRes.Error != "INVALID_CAPTCHA" {
		t.Errorf("expected INVALID_CAPTCHA error code, got %s", errRes.Error)
	}
}

func TestPerspectivesAPI_Lifecycle(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "puser1", "p1@example.com", "Password123!")
	accID := createAccount(t, r, token, "Emergency Savings", "savings", "Emergency Fund")

	body, _ := json.Marshal(models.PerspectiveCreateRequest{
		Name: "Retirement Outlook", Description: "Long term view",
		AccountIDs: []string{accID},
	})
	req := newTestRequest(http.MethodPost, "/api/v1/perspectives", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on perspective create, got %d: %s", w.Code, w.Body.String())
	}
	var pRes models.PerspectiveResponse
	_ = json.Unmarshal(w.Body.Bytes(), &pRes)

	verifyPerspectiveEndpoints(t, r, token, pRes.ID)
}

func verifyPerspectiveEndpoints(t *testing.T, r *gin.Engine, token, pID string) {
	// List perspectives
	reqList := newTestRequest(http.MethodGet, "/api/v1/perspectives", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 on list perspectives, got %d", wList.Code)
	}

	// View perspective
	reqView := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/perspectives/%s/view", pID), nil)
	reqView.Header.Set("Authorization", "Bearer "+token)
	wView := httptest.NewRecorder()
	r.ServeHTTP(wView, reqView)
	if wView.Code != http.StatusOK {
		t.Fatalf("expected 200 on get perspective view, got %d: %s", wView.Code, wView.Body.String())
	}
}

func TestPerspectiveSharingAPI(t *testing.T) {
	r, _ := setupTestRouter()
	tokA := registerAndGetToken(t, r, "owner_persp", "owner_p@example.com", "Password123!")
	tokB := registerAndGetToken(t, r, "guest_persp", "guest_p@example.com", "Password123!")

	// Create perspective as owner
	body, _ := json.Marshal(models.PerspectiveCreateRequest{Name: "Shared Perspective"})
	req := newTestRequest(http.MethodPost, "/api/v1/perspectives", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var pRes models.PerspectiveResponse
	_ = json.Unmarshal(w.Body.Bytes(), &pRes)

	// Guest tries to view - should be 404
	reqGuest := newTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/perspectives/%s", pRes.ID), nil)
	reqGuest.Header.Set("Authorization", "Bearer "+tokB)
	wGuest := httptest.NewRecorder()
	r.ServeHTTP(wGuest, reqGuest)
	if wGuest.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unshared perspective, got %d", wGuest.Code)
	}

	sharePerspectiveWithUser(t, r, tokA, tokB, pRes.ID)
}

func sharePerspectiveWithUser(t *testing.T, r *gin.Engine, tokA, tokB, pID string) {
	// Owner shares with Guest: resolve guest user ID
	reqMe := newTestRequest(http.MethodGet, "/api/v1/accounts", nil)
	reqMe.Header.Set("Authorization", "Bearer "+tokB)
	wMe := httptest.NewRecorder()
	r.ServeHTTP(wMe, reqMe)

	// Share with guest
	var shareBody = []byte(`{"user_id": "00000000-0000-0000-0000-000000000001", "access_level": "read"}`)
	reqShare := newTestRequest(http.MethodPost, fmt.Sprintf("/api/v1/perspectives/%s/access", pID), bytes.NewReader(shareBody))
	reqShare.Header.Set("Content-Type", "application/json")
	reqShare.Header.Set("Authorization", "Bearer "+tokA)
	wShare := httptest.NewRecorder()
	r.ServeHTTP(wShare, reqShare)
	// Even if user not found, returns 404 USER_NOT_FOUND
	if wShare.Code != http.StatusNotFound && wShare.Code != http.StatusOK {
		t.Fatalf("unexpected code on share: %d", wShare.Code)
	}
}

func doPatchAccountRaw(r *gin.Engine, token, accountID, rawJSON string) *httptest.ResponseRecorder {
	req := newTestRequest(http.MethodPatch, "/api/v1/accounts/"+accountID, strings.NewReader(rawJSON))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func registerUserWithID(t *testing.T, r *gin.Engine, username, email string) (string, string) {
	t.Helper()
	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: username, Email: email, Password: "Password123!",
	})
	req := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var u models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &u)
	code, _ := auth.GenerateCode(u.TOTPSecret, time.Now().Unix()/30)
	tokBody, _ := json.Marshal(models.TokenExchangeRequest{Password: "Password123!", TOTPCode: code})
	tReq := newTestRequest(http.MethodPost, "/api/v1/users/"+username+"/token", bytes.NewReader(tokBody))
	tReq.Header.Set("Content-Type", "application/json")
	tw := httptest.NewRecorder()
	r.ServeHTTP(tw, tReq)
	var tok models.TokenResponse
	_ = json.Unmarshal(tw.Body.Bytes(), &tok)
	return u.ID, tok.AccessToken
}

func assertAccountPatchSuccess(t *testing.T, r *gin.Engine, token, accID, rawJSON, wantName, wantDesc string) {
	t.Helper()
	w := doPatchAccountRaw(r, token, accID, rawJSON)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on patch %s, got %d: %s", rawJSON, w.Code, w.Body.String())
	}
	var got models.AccountResponse
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Name != wantName || got.Description != wantDesc || got.Type != "checking" {
		t.Fatalf("unexpected account payload %+v (want name=%q desc=%q)", got, wantName, wantDesc)
	}
}

func TestPatchAccount_NameAndDescriptionValidation(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "rename_user", "rename@example.com", "Password123!")
	accID := createAccount(t, r, token, "Primary Everyday Checking", "checking", "Monthly bills account")

	assertAccountPatchSuccess(t, r, token, accID, `{"name":"Household Checking"}`, "Household Checking", "Monthly bills account")
	assertAccountPatchSuccess(t, r, token, accID, `{"description":"Shared household budget"}`, "Household Checking", "Shared household budget")
	assertAccountPatchSuccess(t, r, token, accID, `{"name":"Operating Checking","description":"Daily expenses"}`, "Operating Checking", "Daily expenses")

	badPayloads := []string{
		`{}`,
		`{"name":""}`,
		`{"name":"   "}`,
		`{"description":""}`,
		`{"description":"   "}`,
		`{"type":"savings"}`,
		`{"name":"Valid","type":"savings"}`,
		`{"name":"Valid","admin_id":"00000000-0000-0000-0000-000000000001"}`,
		`{"name":"` + strings.Repeat("x", 101) + `"}`,
		`{"description":"` + strings.Repeat("d", 256) + `"}`,
	}
	for _, payload := range badPayloads {
		if w := doPatchAccountRaw(r, token, accID, payload); w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for payload %s, got %d: %s", payload, w.Code, w.Body.String())
		}
	}
	if w := doPatchAccountRaw(r, "", accID, `{"name":"Unauth"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestPatchAccount_AccessControl(t *testing.T) {
	r, _ := setupTestRouter()
	_, ownerTok := registerUserWithID(t, r, "patch_owner", "po@example.com")
	readerID, readerTok := registerUserWithID(t, r, "patch_reader", "pr@example.com")
	editorID, editorTok := registerUserWithID(t, r, "patch_editor", "pe@example.com")
	_, outsiderTok := registerUserWithID(t, r, "patch_out", "pout@example.com")
	accID := createAccount(t, r, ownerTok, "Shared Checking", "checking", "Shared account")

	for _, grant := range []string{
		fmt.Sprintf(`{"user_id":%q,"access_level":"read"}`, readerID),
		fmt.Sprintf(`{"user_id":%q,"access_level":"edit"}`, editorID),
	} {
		req := newTestRequest(http.MethodPost, "/api/v1/accounts/"+accID+"/access", strings.NewReader(grant))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ownerTok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 granting access, got %d: %s", w.Code, w.Body.String())
		}
	}

	if w := doPatchAccountRaw(r, readerTok, accID, `{"name":"Reader Attempt"}`); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for read access, got %d", w.Code)
	}
	if w := doPatchAccountRaw(r, outsiderTok, accID, `{"name":"Outsider Attempt"}`); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unshared user, got %d", w.Code)
	}
	if w := doPatchAccountRaw(r, editorTok, accID, `{"name":"Editor Renamed Checking"}`); w.Code != http.StatusOK {
		t.Fatalf("expected 200 for edit access, got %d: %s", w.Code, w.Body.String())
	}
}

func doUserReq(r *gin.Engine, method, token, username string) *httptest.ResponseRecorder {
	req := newTestRequest(method, "/api/v1/users/"+username, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetAndDeleteUserEndpoint(t *testing.T) {
	r, _ := setupTestRouter()
	tokA := registerAndGetToken(t, r, "del_a", "dela@example.com", "Password123!")
	tokB := registerAndGetToken(t, r, "del_b", "delb@example.com", "Password123!")
	accA := createAccount(t, r, tokA, "Checking A", "checking", "Personal checking")

	if w := doUserReq(r, http.MethodGet, "", "del_a"); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 getting user without JWT, got %d", w.Code)
	}
	if w := doUserReq(r, http.MethodGet, tokB, "del_a"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 getting another user, got %d", w.Code)
	}
	wGet := doUserReq(r, http.MethodGet, tokA, "del_a")
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 getting own user, got %d: %s", wGet.Code, wGet.Body.String())
	}
	var prof models.UserResponse
	_ = json.Unmarshal(wGet.Body.Bytes(), &prof)
	if prof.Username != "del_a" || prof.Email != "dela@example.com" || prof.TOTPSecret != "" {
		t.Fatalf("unexpected profile payload: %+v", prof)
	}

	if w := doUserReq(r, http.MethodDelete, "", "del_a"); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without JWT, got %d", w.Code)
	}
	if w := doUserReq(r, http.MethodDelete, tokB, "del_a"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting another user, got %d", w.Code)
	}
	if w := doUserReq(r, http.MethodDelete, tokA, "del_a"); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting own user, got %d: %s", w.Code, w.Body.String())
	}
	if w := doUserReq(r, http.MethodGet, tokA, "del_a"); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 getting deleted user, got %d", w.Code)
	}
	if w := doUserReq(r, http.MethodDelete, tokA, "del_a"); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 deleting already-deleted user, got %d", w.Code)
	}
	if wAcc := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accA, tokB, nil); wAcc.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cascaded account, got %d", wAcc.Code)
	}
}

func setupPolicyTestRouter(termsHash, privacyHash string) (*gin.Engine, repository.Repository, *auth.Service) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(trace.Middleware(), cors.Middleware(), gin.Recovery())
	repo := repository.NewMemoryRepository()
	captcha.SetCustomStore(captcha.NewRepoStore(repo))
	authService := auth.NewService("test-secret-key-12345", auth.DefaultExpirationSeconds)
	ctrl := controller.NewController(repo, authService)
	ctrl.SetEnforcePolicy(true)
	ctrl.SetTermsHash(termsHash)
	ctrl.SetPrivacyHash(privacyHash)
	handlers.NewHandler(ctrl, authService).RegisterRoutes(r)
	return r, repo, authService
}

func assertTokenTermsClaim(t *testing.T, svc *auth.Service, token string, expected bool) {
	t.Helper()
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.TermsApproved != expected {
		t.Fatalf("expected token terms_approved=%v, got %v", expected, claims.TermsApproved)
	}
}

func TestPolicyAgreementAndEnforcementEndpoints(t *testing.T) {
	termsHash := "b08f3d8b1cd44544888a78ed13316c0910725944bc46808ab542e36d9b2fd84c"
	privacyHash := "a731e26873a4229a02f875965c59a4f7b5de6c029b7e24de6907ed5b9aa6a1a7"
	r, repo, svc := setupPolicyTestRouter(termsHash, privacyHash)
	tok := registerAndGetToken(t, r, "pol_u", "polu@example.com", "Password123!")
	assertTokenTermsClaim(t, svc, tok, false)

	wBlocked := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts", tok, nil)
	if wBlocked.Code != http.StatusForbidden || !strings.Contains(wBlocked.Body.String(), "POLICY_APPROVAL_REQUIRED") {
		t.Fatalf("expected 403 POLICY_APPROVAL_REQUIRED, got %d: %s", wBlocked.Code, wBlocked.Body.String())
	}
	badTerms := map[string]any{"terms_approved": true, "privacy_approved": true, "us_adult_consumer_confirmed": true, "terms_sha256": strings.Repeat("0", 64), "privacy_sha256": privacyHash}
	if wBad := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/pol_u/policy-agreements", tok, badTerms); wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on mismatched terms SHA-256, got %d: %s", wBad.Code, wBad.Body.String())
	}
	badPrivacy := map[string]any{"terms_approved": true, "privacy_approved": true, "us_adult_consumer_confirmed": true, "terms_sha256": termsHash, "privacy_sha256": strings.Repeat("0", 64)}
	if wBad := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/pol_u/policy-agreements", tok, badPrivacy); wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on mismatched privacy SHA-256, got %d: %s", wBad.Code, wBad.Body.String())
	}
	goodBody := map[string]any{"terms_approved": true, "privacy_approved": true, "us_adult_consumer_confirmed": true, "terms_sha256": termsHash, "privacy_sha256": privacyHash}
	if wOK := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/pol_u/policy-agreements", tok, goodBody); wOK.Code != http.StatusCreated {
		t.Fatalf("expected 201 recording policy agreement, got %d: %s", wOK.Code, wOK.Body.String())
	}
	wRef := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/pol_u/token/refresh", tok, nil)
	if wRef.Code != http.StatusOK {
		t.Fatalf("expected 200 refreshing token after policy approval, got %d: %s", wRef.Code, wRef.Body.String())
	}
	var refreshed models.TokenResponse
	_ = json.Unmarshal(wRef.Body.Bytes(), &refreshed)
	assertTokenTermsClaim(t, svc, refreshed.AccessToken, true)

	if wAllowed := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts", refreshed.AccessToken, nil); wAllowed.Code != http.StatusOK {
		t.Fatalf("expected 200 accessing accounts after approval, got %d: %s", wAllowed.Code, wAllowed.Body.String())
	}
	_ = repo.DeactivateAllPolicyAgreements(context.Background())
	if wDeact := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts", refreshed.AccessToken, nil); wDeact.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after ledger deactivation, got %d", wDeact.Code)
	}
}

func postPolicyWithOriginViewerIP(r *gin.Engine, tok, username, termsHash, privacyHash, headerVal string) models.PolicyAgreementResponse {
	payload, _ := json.Marshal(map[string]any{
		"terms_approved": true, "privacy_approved": true, "us_adult_consumer_confirmed": true, "terms_sha256": termsHash, "privacy_sha256": privacyHash,
	})
	req := newTestRequest(http.MethodPost, "/api/v1/users/"+username+"/policy-agreements", bytes.NewReader(payload))
	req.RemoteAddr = "10.0.0.12:443"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("CloudFront-Viewer-Address", headerVal)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out models.PolicyAgreementResponse
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func TestRecordPolicyAgreement_ResolvesOriginViewerIP(t *testing.T) {
	termsHash := "b08f3d8b1cd44544888a78ed13316c0910725944bc46808ab542e36d9b2fd84c"
	privacyHash := "a731e26873a4229a02f875965c59a4f7b5de6c029b7e24de6907ed5b9aa6a1a7"
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var verifiedOrigin bool
	r.Use(trace.Middleware(), func(c *gin.Context) {
		if verifiedOrigin {
			c.Set("originVerified", true)
		}
		c.Next()
	})
	repo := repository.NewMemoryRepository()
	captcha.SetCustomStore(captcha.NewRepoStore(repo))
	svc := auth.NewService("test-secret-key-12345", auth.DefaultExpirationSeconds)
	ctrl := controller.NewController(repo, svc)
	ctrl.SetTermsHash(termsHash)
	ctrl.SetPrivacyHash(privacyHash)
	handlers.NewHandler(ctrl, svc).RegisterRoutes(r)

	tok := registerAndGetToken(t, r, "ip_user", "ip@example.com", "Password123!")
	if got := postPolicyWithOriginViewerIP(r, tok, "ip_user", termsHash, privacyHash, "198.51.100.99"); got.ClientIP != "10.0.0.12" {
		t.Fatalf("expected spoofed CloudFront-Viewer-Address ignored when origin unverified, got %q", got.ClientIP)
	}
	verifiedOrigin = true
	if got := postPolicyWithOriginViewerIP(r, tok, "ip_user", termsHash, privacyHash, "198.51.100.99:54321"); got.ClientIP != "198.51.100.99" {
		t.Fatalf("expected CloudFront-Viewer-Address IPv4 used when origin verified, got %q", got.ClientIP)
	}
	if got := postPolicyWithOriginViewerIP(r, tok, "ip_user", termsHash, privacyHash, "2001:db8::42"); got.ClientIP != "2001:db8::42" {
		t.Fatalf("expected CloudFront-Viewer-Address IPv6 used when origin verified, got %q", got.ClientIP)
	}
	if got := postPolicyWithOriginViewerIP(r, tok, "ip_user", termsHash, privacyHash, "2600:1700:15d0:4d60:283b:5aa1:2bb4:df0f:60834"); got.ClientIP != "2600:1700:15d0:4d60:283b:5aa1:2bb4:df0f" {
		t.Fatalf("expected CloudFront-Viewer-Address unbracketed IPv6+port parsed cleanly, got %q", got.ClientIP)
	}
	if got := postPolicyWithOriginViewerIP(r, tok, "ip_user", termsHash, privacyHash, "not-an-ip"); got.ClientIP != "10.0.0.12" {
		t.Fatalf("expected fallback to RemoteAddr on malformed CloudFront-Viewer-Address, got %q", got.ClientIP)
	}

	// Fallback to X-Forwarded-For when CloudFront-Viewer-Address is absent and proxy is trusted (skipping private 10.x hops)
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	payload, _ := json.Marshal(map[string]any{
		"terms_approved": true, "privacy_approved": true, "us_adult_consumer_confirmed": true, "terms_sha256": termsHash, "privacy_sha256": privacyHash,
	})
	reqXFF := newTestRequest(http.MethodPost, "/api/v1/users/ip_user/policy-agreements", bytes.NewReader(payload))
	reqXFF.RemoteAddr = "10.0.1.5:443"
	reqXFF.Header.Set("Content-Type", "application/json")
	reqXFF.Header.Set("Authorization", "Bearer "+tok)
	reqXFF.Header.Set("X-Forwarded-For", "10.0.0.2, 203.0.113.77, 172.31.10.5")
	wXFF := httptest.NewRecorder()
	r.ServeHTTP(wXFF, reqXFF)
	var outXFF models.PolicyAgreementResponse
	_ = json.Unmarshal(wXFF.Body.Bytes(), &outXFF)
	if outXFF.ClientIP != "203.0.113.77" {
		t.Fatalf("expected X-Forwarded-For fallback public IP '203.0.113.77', got %q", outXFF.ClientIP)
	}

	// Fallback to RFC 7239 Forwarded header when CloudFront-Viewer-Address and X-Forwarded-For are absent
	reqFwd := newTestRequest(http.MethodPost, "/api/v1/users/ip_user/policy-agreements", bytes.NewReader(payload))
	reqFwd.RemoteAddr = "10.10.1.123:56344"
	reqFwd.Header.Set("Content-Type", "application/json")
	reqFwd.Header.Set("Authorization", "Bearer "+tok)
	reqFwd.Header.Set("Forwarded", "for=2600:1700:15d0:4d60:283b:5aa1:2bb4:df0f,for=15.158.51.111;host=example.execute-api.us-east-2.amazonaws.com;proto=https")
	wFwd := httptest.NewRecorder()
	r.ServeHTTP(wFwd, reqFwd)
	var outFwd models.PolicyAgreementResponse
	_ = json.Unmarshal(wFwd.Body.Bytes(), &outFwd)
	if outFwd.ClientIP != "2600:1700:15d0:4d60:283b:5aa1:2bb4:df0f" {
		t.Fatalf("expected Forwarded fallback IPv6 '2600:1700:15d0:4d60:283b:5aa1:2bb4:df0f', got %q", outFwd.ClientIP)
	}
}

func registerUserFull(t *testing.T, r *gin.Engine, username, email, password string) models.UserResponse {
	t.Helper()
	body, _ := json.Marshal(models.UserRegistrationRequest{Username: username, Email: email, Password: password})
	req := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("registerUserFull expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var res models.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return res
}

func exchangeTokenWithSecret(t *testing.T, r *gin.Engine, username, password, totpSecret string) string {
	t.Helper()
	code, _ := auth.GenerateCode(totpSecret, time.Now().Unix()/30)
	tokBody, _ := json.Marshal(models.TokenExchangeRequest{Password: password, TOTPCode: code})
	wTok := httptest.NewRecorder()
	reqTok := newTestRequest(http.MethodPost, "/api/v1/users/"+username+"/token", bytes.NewReader(tokBody))
	reqTok.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wTok, reqTok)
	var tokRes models.TokenResponse
	_ = json.Unmarshal(wTok.Body.Bytes(), &tokRes)
	return tokRes.AccessToken
}

func TestAccountRecoveryAndRotationEndpoints(t *testing.T) {
	r, _ := setupTestRouter()
	uA := registerUserFull(t, r, "rec_h_a", "recha@example.com", "Password123!")
	uB := registerUserFull(t, r, "rec_h_b", "rechb@example.com", "Password123!")

	recReq, _ := json.Marshal(models.AccountRecoveryRequest{RecoveryPhrase: uA.RecoveryPhrase, NewPassword: "RecoveredPass2!"})
	wRec := httptest.NewRecorder()
	req := newTestRequest(http.MethodPost, "/api/v1/users/rec_h_a/recover", bytes.NewReader(recReq))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wRec, req)
	if wRec.Code != http.StatusOK {
		t.Fatalf("expected 200 recovering account, got %d: %s", wRec.Code, wRec.Body.String())
	}
	var recRes models.AccountRecoveryResponse
	_ = json.Unmarshal(wRec.Body.Bytes(), &recRes)

	wReplay := httptest.NewRecorder()
	reqReplay := newTestRequest(http.MethodPost, "/api/v1/users/rec_h_a/recover", bytes.NewReader(recReq))
	reqReplay.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wReplay, reqReplay)
	if wReplay.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 replaying spent phrase, got %d", wReplay.Code)
	}

	tok := exchangeTokenWithSecret(t, r, "rec_h_a", "RecoveredPass2!", recRes.TOTPSecret)
	codeA, _ := auth.GenerateCode(recRes.TOTPSecret, time.Now().Unix()/30)
	rotPayload := map[string]string{"password": "RecoveredPass2!", "totp_code": codeA}
	if wCross := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/"+uB.Username+"/recovery-key/rotate", tok, rotPayload); wCross.Code != http.StatusForbidden {
		t.Fatalf("expected 403 rotating another user's key, got %d", wCross.Code)
	}
	if wRot := doAuthedJSON(t, r, http.MethodPost, "/api/v1/users/rec_h_a/recovery-key/rotate", tok, rotPayload); wRot.Code != http.StatusOK {
		t.Fatalf("expected 200 rotating own recovery key, got %d: %s", wRot.Code, wRot.Body.String())
	}
}

func TestPasswordExceeding72BytesReturns400InvalidPassword(t *testing.T) {
	r, _ := setupTestRouter()
	longPass := "A1!" + strings.Repeat("a", 70)

	regBody, _ := json.Marshal(models.UserRegistrationRequest{
		Username: "long_pw_u", Email: "longpw@example.com", Password: longPass,
	})
	reqReg := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	r.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusBadRequest || !strings.Contains(wReg.Body.String(), "INVALID_PASSWORD") {
		t.Fatalf("expected 400 INVALID_PASSWORD on registration >72 bytes, got %d: %s", wReg.Code, wReg.Body.String())
	}

	u := registerUserFull(t, r, "long_pw_rec", "longrec@example.com", "Password123!")
	recBody, _ := json.Marshal(models.AccountRecoveryRequest{RecoveryPhrase: u.RecoveryPhrase, NewPassword: longPass})
	reqRec := newTestRequest(http.MethodPost, "/api/v1/users/long_pw_rec/recover", bytes.NewReader(recBody))
	reqRec.Header.Set("Content-Type", "application/json")
	wRec := httptest.NewRecorder()
	r.ServeHTTP(wRec, reqRec)
	if wRec.Code != http.StatusBadRequest || !strings.Contains(wRec.Body.String(), "INVALID_PASSWORD") {
		t.Fatalf("expected 400 INVALID_PASSWORD on recovery >72 bytes, got %d: %s", wRec.Code, wRec.Body.String())
	}
}

func TestAccountCreationAndPatchRejectsFiveOrMoreDigits(t *testing.T) {
	r, _ := setupTestRouter()
	tok := registerAndGetToken(t, r, "digits_h_u", "digitsh@example.com", "Password123!")

	// 4 digits accepted
	wOK := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts", tok, map[string]string{
		"name": "Rewards 1234", "type": "credit card", "description": "Ending 9876",
	})
	if wOK.Code != http.StatusCreated {
		t.Fatalf("expected 201 for 4-digit account name/desc, got %d: %s", wOK.Code, wOK.Body.String())
	}
	var created models.AccountResponse
	_ = json.Unmarshal(wOK.Body.Bytes(), &created)

	// 5 digits rejected on POST
	wBadPost := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts", tok, map[string]string{
		"name": "Card 12345", "type": "credit card", "description": "Personal",
	})
	if wBadPost.Code != http.StatusBadRequest || !strings.Contains(wBadPost.Body.String(), "PROHIBITED_SENSITIVE_DIGITS") {
		t.Fatalf("expected 400 PROHIBITED_SENSITIVE_DIGITS on POST, got %d: %s", wBadPost.Code, wBadPost.Body.String())
	}

	// 5 digits rejected on PATCH
	wBadPatch := doAuthedJSON(t, r, http.MethodPatch, "/api/v1/accounts/"+created.ID, tok, map[string]string{
		"description": "Routing 12345",
	})
	if wBadPatch.Code != http.StatusBadRequest || !strings.Contains(wBadPatch.Body.String(), "PROHIBITED_SENSITIVE_DIGITS") {
		t.Fatalf("expected 400 PROHIBITED_SENSITIVE_DIGITS on PATCH, got %d: %s", wBadPatch.Code, wBadPatch.Body.String())
	}
}

func TestBulkCreateCheckingRecordsHandler(t *testing.T) {
	r, _ := setupTestRouter()
	tok := registerAndGetToken(t, r, "bulk_h_u", "bulkh@example.com", "Password123!")

	wAcc := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts", tok, map[string]string{
		"name": "Bulk Handler Checking", "type": "checking", "description": "Bulk HTTP test",
	})
	if wAcc.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating account, got %d", wAcc.Code)
	}
	var acc models.AccountResponse
	_ = json.Unmarshal(wAcc.Body.Bytes(), &acc)

	// Valid bulk request
	val := -12.34
	wOK := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+acc.ID+"/records/bulk", tok, models.CheckingRecordBulkCreateRequest{
		Records: []models.CheckingRecordCreateRequest{
			{Value: &val, DateSent: "2026-11-10", Entity: "Coffee Shop", Description: "Latte"},
			{Value: &val, DateSent: "2026-11-11", Entity: "Corner Grocery", Description: "Lunch"},
		},
	})
	if wOK.Code != http.StatusCreated {
		t.Fatalf("expected 201 for bulk create, got %d: %s", wOK.Code, wOK.Body.String())
	}
	var bulkRes models.CheckingRecordBulkCreateResponse
	_ = json.Unmarshal(wOK.Body.Bytes(), &bulkRes)
	if bulkRes.CreatedCount != 2 || len(bulkRes.Records) != 2 {
		t.Fatalf("expected 2 created records, got %+v", bulkRes)
	}

	// Over 1000 limit rejected
	over := make([]models.CheckingRecordCreateRequest, models.MaxBulkCheckingRecords+1)
	for i := range over {
		over[i] = models.CheckingRecordCreateRequest{Value: &val, DateSent: "2026-11-10", Entity: "Market"}
	}
	wOver := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+acc.ID+"/records/bulk", tok, models.CheckingRecordBulkCreateRequest{
		Records: over,
	})
	if wOver.Code != http.StatusBadRequest || !strings.Contains(wOver.Body.String(), "BULK_LIMIT_EXCEEDED") {
		t.Fatalf("expected 400 BULK_LIMIT_EXCEEDED, got %d: %s", wOver.Code, wOver.Body.String())
	}

	// Invalid date rejected
	wBadDate := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+acc.ID+"/records/bulk", tok, models.CheckingRecordBulkCreateRequest{
		Records: []models.CheckingRecordCreateRequest{{Value: &val, DateSent: "11/10/2026", Entity: "Market"}},
	})
	if wBadDate.Code != http.StatusBadRequest || !strings.Contains(wBadDate.Body.String(), "INVALID_DATE_SENT") {
		t.Fatalf("expected 400 INVALID_DATE_SENT, got %d: %s", wBadDate.Code, wBadDate.Body.String())
	}
}


