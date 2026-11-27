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

func TestMobileEndpoints_ScopeDeviceBindingAndIsolation(t *testing.T) {
	r, authSvc := setupPasswordRouter(false)

	regBody := `{"username":"mob_http_u","email":"mob_http@example.com","password":"ComplexPassword123!"}`
	regReq := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	r.ServeHTTP(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d: %s", regRec.Code, regRec.Body.String())
	}
	var userResp models.UserResponse
	_ = json.Unmarshal(regRec.Body.Bytes(), &userResp)
	stdTok, _ := authSvc.GenerateToken(userResp.ID, userResp.Username, false, true)

	// Create checking account and snapshot account using standard JWT
	chkReq := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString(`{"name":"Primary Checking","type":"checking","description":"Checking"}`))
	chkReq.Header.Set("Content-Type", "application/json")
	chkReq.Header.Set("Authorization", "Bearer "+stdTok)
	chkRec := httptest.NewRecorder()
	r.ServeHTTP(chkRec, chkReq)
	var chkAcc models.AccountResponse
	_ = json.Unmarshal(chkRec.Body.Bytes(), &chkAcc)

	snpReq := newTestRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString(`{"name":"Brokerage","type":"snapshot","description":"Snapshot"}`))
	snpReq.Header.Set("Content-Type", "application/json")
	snpReq.Header.Set("Authorization", "Bearer "+stdTok)
	snpRec := httptest.NewRecorder()
	r.ServeHTTP(snpRec, snpReq)
	var snpAcc models.AccountResponse
	_ = json.Unmarshal(snpRec.Body.Bytes(), &snpAcc)

	// Standard token works for issuing mobile token with X-Mobile-Identifier
	deviceID := "ios-keychain-install-uuid-9876"
	issueMissingHdr := newTestRequest(http.MethodPost, "/api/v1/mobile/token", nil)
	issueMissingHdr.Header.Set("Authorization", "Bearer "+stdTok)
	recMissingHdr := httptest.NewRecorder()
	r.ServeHTTP(recMissingHdr, issueMissingHdr)
	if recMissingHdr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when X-Mobile-Identifier absent on /mobile/token, got %d", recMissingHdr.Code)
	}

	issueReq := newTestRequest(http.MethodPost, "/api/v1/mobile/token", nil)
	issueReq.Header.Set("Authorization", "Bearer "+stdTok)
	issueReq.Header.Set("X-Mobile-Identifier", deviceID)
	issueRec := httptest.NewRecorder()
	r.ServeHTTP(issueRec, issueReq)
	if issueRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 issuing mobile token, got %d: %s", issueRec.Code, issueRec.Body.String())
	}
	var mobTokResp models.MobileTokenResponse
	_ = json.Unmarshal(issueRec.Body.Bytes(), &mobTokResp)
	if mobTokResp.Limitation != "mobile" || mobTokResp.AccessToken == "" {
		t.Fatalf("unexpected mobile token response: %+v", mobTokResp)
	}

	// Mobile token MUST be rejected on standard endpoints (/api/v1/accounts, /api/v1/mobile/token)
	for _, path := range []struct{ method, url string }{
		{http.MethodGet, "/api/v1/accounts"},
		{http.MethodGet, "/api/v1/users/" + userResp.Username},
		{http.MethodPost, "/api/v1/mobile/token"},
	} {
		req := newTestRequest(path.method, path.url, nil)
		req.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
		req.Header.Set("X-Mobile-Identifier", deviceID)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for mobile token on %s %s, got %d", path.method, path.url, rec.Code)
		}
	}

	// Standard token MUST be rejected on /api/v1/mobile/accounts
	stdOnMobReq := newTestRequest(http.MethodGet, "/api/v1/mobile/accounts", nil)
	stdOnMobReq.Header.Set("Authorization", "Bearer "+stdTok)
	stdOnMobReq.Header.Set("X-Mobile-Identifier", deviceID)
	stdOnMobRec := httptest.NewRecorder()
	r.ServeHTTP(stdOnMobRec, stdOnMobReq)
	if stdOnMobRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for standard token on /mobile/accounts, got %d", stdOnMobRec.Code)
	}

	// Mobile token without or with wrong X-Mobile-Identifier rejected (401)
	wrongDevReq := newTestRequest(http.MethodGet, "/api/v1/mobile/accounts", nil)
	wrongDevReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	wrongDevReq.Header.Set("X-Mobile-Identifier", "another-device-uuid")
	wrongDevRec := httptest.NewRecorder()
	r.ServeHTTP(wrongDevRec, wrongDevReq)
	if wrongDevRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong X-Mobile-Identifier, got %d", wrongDevRec.Code)
	}

	// Matching device succeeds on GET /api/v1/mobile/accounts
	targetsReq := newTestRequest(http.MethodGet, "/api/v1/mobile/accounts", nil)
	targetsReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	targetsReq.Header.Set("X-Mobile-Identifier", deviceID)
	targetsRec := httptest.NewRecorder()
	r.ServeHTTP(targetsRec, targetsReq)
	if targetsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /mobile/accounts, got %d: %s", targetsRec.Code, targetsRec.Body.String())
	}
	var targets models.MobileTargetsResponse
	_ = json.Unmarshal(targetsRec.Body.Bytes(), &targets)
	if len(targets.Accounts) != 1 || targets.Accounts[0].ID != chkAcc.ID || len(targets.Categories) == 0 {
		t.Fatalf("unexpected mobile targets: %+v", targets)
	}

	// Unknown JSON field rejected (bindStrictJSON)
	badFieldReq := newTestRequest(http.MethodPost, "/api/v1/mobile/accounts/"+chkAcc.ID+"/records", bytes.NewBufferString(`{"value":-12.5,"date_sent":"2026-11-26","entity":"Market","description":"extra"}`))
	badFieldReq.Header.Set("Content-Type", "application/json")
	badFieldReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	badFieldReq.Header.Set("X-Mobile-Identifier", deviceID)
	badFieldRec := httptest.NewRecorder()
	r.ServeHTTP(badFieldRec, badFieldReq)
	if badFieldRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on extra field in mobile record, got %d", badFieldRec.Code)
	}

	// Valid mobile record succeeds and returns only id, account_id, created_at
	addBody := `{"value":-24.75,"date_sent":"2026-11-26","entity":"Corner Grocery","category_id":"` + targets.Categories[0].ID + `"}`
	addReq := newTestRequest(http.MethodPost, "/api/v1/mobile/accounts/"+chkAcc.ID+"/records", bytes.NewBufferString(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	addReq.Header.Set("X-Mobile-Identifier", deviceID)
	addRec := httptest.NewRecorder()
	r.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on mobile record create, got %d: %s", addRec.Code, addRec.Body.String())
	}

	// Snapshot account rejected (400)
	addSnapReq := newTestRequest(http.MethodPost, "/api/v1/mobile/accounts/"+snpAcc.ID+"/records", bytes.NewBufferString(addBody))
	addSnapReq.Header.Set("Content-Type", "application/json")
	addSnapReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	addSnapReq.Header.Set("X-Mobile-Identifier", deviceID)
	addSnapRec := httptest.NewRecorder()
	r.ServeHTTP(addSnapRec, addSnapReq)
	if addSnapRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on snapshot account mobile write, got %d", addSnapRec.Code)
	}

	// Password change revokes mobile token immediately (401)
	code, _ := auth.GenerateCode(userResp.TOTPSecret, time.Now().Unix()/30)
	pwBody := `{"old_password":"ComplexPassword123!","new_password":"ComplexPassword456!","new_password_confirm":"ComplexPassword456!","totp_code":"` + code + `"}`
	pwReq := newTestRequest(http.MethodPost, "/api/v1/users/"+userResp.Username+"/password", bytes.NewBufferString(pwBody))
	pwReq.Header.Set("Content-Type", "application/json")
	pwReq.Header.Set("Authorization", "Bearer "+stdTok)
	pwRec := httptest.NewRecorder()
	r.ServeHTTP(pwRec, pwReq)
	if pwRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on password change, got %d", pwRec.Code)
	}

	revokedReq := newTestRequest(http.MethodGet, "/api/v1/mobile/accounts", nil)
	revokedReq.Header.Set("Authorization", "Bearer "+mobTokResp.AccessToken)
	revokedReq.Header.Set("X-Mobile-Identifier", deviceID)
	revokedRec := httptest.NewRecorder()
	r.ServeHTTP(revokedRec, revokedReq)
	if revokedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after password change revoked mobile token, got %d", revokedRec.Code)
	}
}
