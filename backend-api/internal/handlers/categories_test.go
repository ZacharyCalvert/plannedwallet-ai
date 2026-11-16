package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

func doAuthedJSON(t *testing.T, r *gin.Engine, method, path, token string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := newTestRequest(method, path, bodyReader)
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCategoriesCRUDHandlers(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "hcat1", "hcat1@example.com", "SecurePass123!")

	wList := doAuthedJSON(t, r, http.MethodGet, "/api/v1/categories", token, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 listing categories, got %d", wList.Code)
	}
	var cats []models.CategoryResponse
	_ = json.Unmarshal(wList.Body.Bytes(), &cats)
	if len(cats) != 7 {
		t.Fatalf("expected 7 default categories, got %d", len(cats))
	}

	// Default category cannot be deleted
	wDelDefault := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/categories/"+cats[0].ID, token, nil)
	if wDelDefault.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting default category, got %d", wDelDefault.Code)
	}

	// Create custom category and check 400 / 409 / 200 / 204
	wBad := doAuthedJSON(t, r, http.MethodPost, "/api/v1/categories", token, map[string]string{})
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on missing name, got %d", wBad.Code)
	}

	wCreate := doAuthedJSON(t, r, http.MethodPost, "/api/v1/categories", token, models.CategoryCreateRequest{Name: "Hobbies"})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating category, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var created models.CategoryResponse
	_ = json.Unmarshal(wCreate.Body.Bytes(), &created)

	wDup := doAuthedJSON(t, r, http.MethodPost, "/api/v1/categories", token, models.CategoryCreateRequest{Name: "hobbies"})
	if wDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate category, got %d", wDup.Code)
	}

	wGet := doAuthedJSON(t, r, http.MethodGet, "/api/v1/categories/"+created.ID, token, nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 getting category, got %d", wGet.Code)
	}

	wDel := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/categories/"+created.ID, token, nil)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting category, got %d", wDel.Code)
	}
}

func TestAccountRecordCategoriesAndSummaryHandlers(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "hcat2", "hcat2@example.com", "SecurePass123!")
	accID := createAccount(t, r, token, "Monthly Savings", "savings", "Savings with categories")
	snapID := createAccount(t, r, token, "Net Worth Snap", "snapshot", "Snapshot account")

	wCats := doAuthedJSON(t, r, http.MethodGet, "/api/v1/categories", token, nil)
	var cats []models.CategoryResponse
	_ = json.Unmarshal(wCats.Body.Bytes(), &cats)
	catID := cats[0].ID

	val1, val2 := -80.0, -20.0
	wRec := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", token, models.CheckingRecordCreateRequest{
		Value: &val1, DateSent: "2026-11-10", CategoryID: &catID, Entity: "Utility Co",
	})
	if wRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating record with category, got %d", wRec.Code)
	}
	var rec models.CheckingRecordResponse
	_ = json.Unmarshal(wRec.Body.Bytes(), &rec)

	_ = doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", token, models.CheckingRecordCreateRequest{
		Value: &val2, DateSent: "2026-11-11", Entity: "Corner Shop",
	})

	// Summary endpoint works on savings account
	wSummary := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accID+"/records/categories", token, nil)
	if wSummary.Code != http.StatusOK {
		t.Fatalf("expected 200 from category summary, got %d: %s", wSummary.Code, wSummary.Body.String())
	}
	var summary models.AccountCategorySummaryResponse
	_ = json.Unmarshal(wSummary.Body.Bytes(), &summary)
	if summary.TotalExpenses != 100.0 || len(summary.Categories) != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	// Clearing category_id via PATCH sets it to null
	empty := ""
	wPatch := doAuthedJSON(t, r, http.MethodPatch, "/api/v1/accounts/"+accID+"/records/"+rec.ID, token, models.CheckingRecordPatchRequest{
		CategoryID: &empty,
	})
	if wPatch.Code != http.StatusOK {
		t.Fatalf("expected 200 patching record, got %d", wPatch.Code)
	}
	var patched models.CheckingRecordResponse
	_ = json.Unmarshal(wPatch.Body.Bytes(), &patched)
	if patched.CategoryID != nil {
		t.Fatalf("expected patched CategoryID to be nil, got %v", *patched.CategoryID)
	}

	// Invalid query param and snapshot account rejection
	wBadQuery := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accID+"/records/categories?cleared=maybe", token, nil)
	if wBadQuery.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid cleared param, got %d", wBadQuery.Code)
	}
	wSnap := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+snapID+"/records/categories", token, nil)
	if wSnap.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on snapshot account category summary, got %d", wSnap.Code)
	}
}
