package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/handlers"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
	"github.com/gin-gonic/gin"
)

func setupSettingsRouter(alwaysOn bool) (*gin.Engine, *controller.Controller, *auth.Service) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(trace.Middleware())
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-123456", 900)
	ctrl := controller.NewController(repo, authSvc)
	ctrl.SetSubscriberAlwaysOn(alwaysOn)
	h := handlers.NewHandler(ctrl, authSvc)
	h.RegisterRoutes(r)
	return r, ctrl, authSvc
}

func TestUserSettingsHandlers_GetAndPatch(t *testing.T) {
	r, _, authSvc := setupSettingsRouter(true)
	tok, _ := authSvc.GenerateToken("u-settings-1", "settings_alice", true, true)

	// Register user in repo by creating via API first
	regBody := `{"username":"settings_alice","email":"settings_alice@example.com","password":"ValidPassword123!"}`
	regReq := newTestRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	r.ServeHTTP(regRec, regReq)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d: %s", regRec.Code, regRec.Body.String())
	}
	var userResp models.UserResponse
	_ = json.Unmarshal(regRec.Body.Bytes(), &userResp)
	tok, _ = authSvc.GenerateToken(userResp.ID, userResp.Username, true, true)

	// GET defaults
	getReq := newTestRequest(http.MethodGet, "/api/v1/users/settings_alice/settings", nil)
	getReq.Header.Set("Authorization", "Bearer "+tok)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET settings, got %d: %s", getRec.Code, getRec.Body.String())
	}

	// PATCH daily_update_order
	snapKey := "snapshot:11111111-1111-4111-8111-111111111111"
	assetKey := "asset:22222222-2222-4222-8222-222222222222"
	patchBody := `{"daily_update_order":["` + snapKey + `","` + assetKey + `"]}`
	patchReq := newTestRequest(http.MethodPatch, "/api/v1/users/settings_alice/settings", bytes.NewBufferString(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+tok)
	patchRec := httptest.NewRecorder()
	r.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on PATCH settings, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	var patched models.UserSettings
	_ = json.Unmarshal(patchRec.Body.Bytes(), &patched)
	if len(patched.Settings.DailyUpdateOrder) != 2 || patched.Settings.DailyUpdateOrder[0] != snapKey {
		t.Fatalf("unexpected patched daily_update_order: %+v", patched.Settings.DailyUpdateOrder)
	}
}
