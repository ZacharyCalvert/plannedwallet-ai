package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func setupPasswordRouter(requireCaptcha bool) (*gin.Engine, *auth.Service) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(trace.Middleware())
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-123456", 900)
	ctrl := controller.NewController(repo, authSvc)
	ctrl.SetRequireCaptcha(requireCaptcha)
	h := handlers.NewHandler(ctrl, authSvc)
	h.RegisterRoutes(r)
	return r, authSvc
}

func doPasswordChangeRequest(r *gin.Engine, token, username, body string) *httptest.ResponseRecorder {
	req := newTestRequest(http.MethodPost, "/api/v1/users/"+username+"/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestPasswordChangeHandler_ValidationAndSuccess(t *testing.T) {
	r, authSvc := setupPasswordRouter(false)

	regBody := `{"username":"pw_handler_u","email":"pw_handler@example.com","password":"InitialPassword123!"}`
	regReq := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	r.ServeHTTP(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d: %s", regRec.Code, regRec.Body.String())
	}
	var userResp models.UserResponse
	_ = json.Unmarshal(regRec.Body.Bytes(), &userResp)

	tok, _ := authSvc.GenerateToken(userResp.ID, userResp.Username, false, true)
	validCode, _ := auth.GenerateCode(userResp.TOTPSecret, time.Now().Unix()/30)
	wrongCode := "000000"
	if validCode == wrongCode {
		wrongCode = "999999"
	}

	// Missing / invalid JWT -> 401
	if rec := doPasswordChangeRequest(r, "", userResp.Username, `{}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without JWT, got %d", rec.Code)
	}
	if rec := doPasswordChangeRequest(r, "not-a-jwt", userResp.Username, `{}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid JWT, got %d", rec.Code)
	}

	// Missing required fields -> 400
	if rec := doPasswordChangeRequest(r, tok, userResp.Username, `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty payload, got %d", rec.Code)
	}

	// Mismatched confirmation -> 400
	mismatchBody := `{"old_password":"InitialPassword123!","new_password":"NewPassword456!","new_password_confirm":"OtherPassword456!","totp_code":"` + validCode + `"}`
	if rec := doPasswordChangeRequest(r, tok, userResp.Username, mismatchBody); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for password mismatch, got %d", rec.Code)
	}

	// Wrong old password -> 401
	wrongOldBody := `{"old_password":"WrongOldPassword123!","new_password":"NewPassword456!","new_password_confirm":"NewPassword456!","totp_code":"` + validCode + `"}`
	if rec := doPasswordChangeRequest(r, tok, userResp.Username, wrongOldBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong old password, got %d", rec.Code)
	}

	// Wrong TOTP -> 401
	wrongTOTPBody := `{"old_password":"InitialPassword123!","new_password":"NewPassword456!","new_password_confirm":"NewPassword456!","totp_code":"` + wrongCode + `"}`
	if rec := doPasswordChangeRequest(r, tok, userResp.Username, wrongTOTPBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong TOTP, got %d", rec.Code)
	}

	// Valid change -> 200
	okBody := `{"old_password":"InitialPassword123!","new_password":"NewPassword456!","new_password_confirm":"NewPassword456!","totp_code":"` + validCode + `"}`
	okRec := doPasswordChangeRequest(r, tok, userResp.Username, okBody)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on password change, got %d: %s", okRec.Code, okRec.Body.String())
	}

	// Old password fails /token, new password succeeds /token
	oldTokReq := newTestRequest(http.MethodPost, "/api/v1/users/"+userResp.Username+"/token", bytes.NewBufferString(`{"password":"InitialPassword123!","totp_code":"`+validCode+`"}`))
	oldTokReq.Header.Set("Content-Type", "application/json")
	oldTokRec := httptest.NewRecorder()
	r.ServeHTTP(oldTokRec, oldTokReq)
	if oldTokRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with old password, got %d", oldTokRec.Code)
	}

	newTokReq := newTestRequest(http.MethodPost, "/api/v1/users/"+userResp.Username+"/token", bytes.NewBufferString(`{"password":"NewPassword456!","totp_code":"`+validCode+`"}`))
	newTokReq.Header.Set("Content-Type", "application/json")
	newTokRec := httptest.NewRecorder()
	r.ServeHTTP(newTokRec, newTokReq)
	if newTokRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with new password, got %d: %s", newTokRec.Code, newTokRec.Body.String())
	}

	// Unknown user, bad password, and bad TOTP on /token return identical 401 status, error code, and message
	var unknownRec, badPwRec, badTotpRec *httptest.ResponseRecorder
	for _, tc := range []struct {
		target **httptest.ResponseRecorder
		user   string
		body   string
	}{
		{&unknownRec, "nonexistent_user", `{"password":"NewPassword456!","totp_code":"` + validCode + `"}`},
		{&badPwRec, userResp.Username, `{"password":"WrongPassword456!","totp_code":"` + validCode + `"}`},
		{&badTotpRec, userResp.Username, `{"password":"NewPassword456!","totp_code":"` + wrongCode + `"}`},
	} {
		req := newTestRequest(http.MethodPost, "/api/v1/users/"+tc.user+"/token", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		*tc.target = rec
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on failed login for %s, got %d", tc.user, rec.Code)
		}
	}
	var errUnknown, errBadPw, errBadTotp models.ErrorResponse
	_ = json.Unmarshal(unknownRec.Body.Bytes(), &errUnknown)
	_ = json.Unmarshal(badPwRec.Body.Bytes(), &errBadPw)
	_ = json.Unmarshal(badTotpRec.Body.Bytes(), &errBadTotp)
	if errUnknown.Error != errBadPw.Error || errUnknown.Error != errBadTotp.Error ||
		errUnknown.Message != errBadPw.Message || errUnknown.Message != errBadTotp.Message {
		t.Fatalf("expected identical login error responses, got unknown=%+v badPw=%+v badTotp=%+v", errUnknown, errBadPw, errBadTotp)
	}
}
