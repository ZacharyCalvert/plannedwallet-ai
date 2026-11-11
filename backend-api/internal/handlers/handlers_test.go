package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
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
	authService := auth.NewService("test-secret-key-12345", 3600)
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
		Entity:      "Target",
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
	if rec1.ID == "" || rec1.Value != -85.25 || rec1.DateCleared != nil || rec1.Entity != "Target" {
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
		t.Errorf("expected 1 pending record (Target), got: %+v", pendingList)
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
		Entity:      "Super Target Store",
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
	if putRec.Value != -90.00 || putRec.Entity != "Super Target Store" {
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
