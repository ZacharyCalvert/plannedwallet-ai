package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/plannedwallet/internal/models"
)

func TestReconcileReportHandlers(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "hrec1", "hrec1@example.com", "SecurePass123!")
	accID := createAccount(t, r, token, "Primary Checking", "checking", "Checking account")

	v1, v2 := 1000.0, -125.50
	wR1 := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", token, models.CheckingRecordCreateRequest{
		Value: &v1, DateSent: "2026-11-01", Entity: "Payroll",
	})
	wR2 := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/records", token, models.CheckingRecordCreateRequest{
		Value: &v2, DateSent: "2026-11-05", Entity: "Utility Co",
	})
	var r1, r2 models.CheckingRecordResponse
	_ = json.Unmarshal(wR1.Body.Bytes(), &r1)
	_ = json.Unmarshal(wR2.Body.Bytes(), &r2)

	// Balance mismatch returns 400
	start, wrongEnd, rightEnd := 0.0, 900.0, 874.50
	wMismatch := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/reconcile-reports", token, models.ReconcileReportCreateRequest{
		Name: "Nov2026", ReportDate: "2026-11-16", StartingBalance: &start, EndingBalance: &wrongEnd, RecordIDs: []string{r1.ID, r2.ID},
	})
	if wMismatch.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on balance mismatch, got %d", wMismatch.Code)
	}

	// Valid reconcile report returns 201
	wCreate := doAuthedJSON(t, r, http.MethodPost, "/api/v1/accounts/"+accID+"/reconcile-reports", token, models.ReconcileReportCreateRequest{
		Name: "Nov2026", ReportDate: "2026-11-16", StartingBalance: &start, EndingBalance: &rightEnd, RecordIDs: []string{r1.ID, r2.ID},
	})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating reconcile report, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var created models.ReconcileReportResponse
	_ = json.Unmarshal(wCreate.Body.Bytes(), &created)
	if created.RecordCount != 2 || created.Name != "Nov2026" {
		t.Fatalf("unexpected reconcile report response: %+v", created)
	}

	// List and Get reconcile reports
	wList := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accID+"/reconcile-reports", token, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 listing reconcile reports, got %d", wList.Code)
	}
	wGet := doAuthedJSON(t, r, http.MethodGet, "/api/v1/accounts/"+accID+"/reconcile-reports/"+created.ID, token, nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 getting reconcile report, got %d", wGet.Code)
	}

	// Reconciled record rejects amount/cleared/sent edits with 409, allows description edit
	badVal := -130.0
	wConflict := doAuthedJSON(t, r, http.MethodPatch, "/api/v1/accounts/"+accID+"/records/"+r2.ID, token, models.CheckingRecordPatchRequest{
		Value: &badVal,
	})
	if wConflict.Code != http.StatusConflict {
		t.Fatalf("expected 409 editing reconciled record amount, got %d", wConflict.Code)
	}
	newDesc := "Electric bill"
	wDesc := doAuthedJSON(t, r, http.MethodPatch, "/api/v1/accounts/"+accID+"/records/"+r2.ID, token, models.CheckingRecordPatchRequest{
		Description: &newDesc,
	})
	if wDesc.Code != http.StatusOK {
		t.Fatalf("expected 200 editing reconciled record description, got %d", wDesc.Code)
	}

	// Deleting reconcile report returns 204 and allows editing record amount again
	wDel := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/accounts/"+accID+"/reconcile-reports/"+created.ID, token, nil)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting reconcile report, got %d", wDel.Code)
	}
	wDelAgain := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/accounts/"+accID+"/reconcile-reports/"+created.ID, token, nil)
	if wDelAgain.Code != http.StatusNotFound {
		t.Fatalf("expected 404 deleting already-deleted reconcile report, got %d", wDelAgain.Code)
	}
	wUnlockPatch := doAuthedJSON(t, r, http.MethodPatch, "/api/v1/accounts/"+accID+"/records/"+r2.ID, token, models.CheckingRecordPatchRequest{
		Value: &badVal,
	})
	if wUnlockPatch.Code != http.StatusOK {
		t.Fatalf("expected 200 editing record after reconcile report deletion, got %d", wUnlockPatch.Code)
	}
}
