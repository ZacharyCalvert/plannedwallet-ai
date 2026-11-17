package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/plannedwallet/internal/models"
)

func TestBillsHandlers_LifecycleAndCascades(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "hbill1", "hbill1@example.com", "SecurePass123!")
	otherTok := registerAndGetToken(t, r, "hbill2", "hbill2@example.com", "SecurePass123!")
	accID := createAccount(t, r, token, "Everyday Checking", "checking", "Bills checking")

	wCat := doAuthedJSON(t, r, http.MethodPost, "/api/v1/categories", token, models.CategoryCreateRequest{Name: "Heating Utility"})
	if wCat.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating category, got %d", wCat.Code)
	}
	var cat models.CategoryResponse
	_ = json.Unmarshal(wCat.Body.Bytes(), &cat)

	// Invalid schedule on yearly frequency returns 400
	amt := -88.50
	wBad := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills", token, models.BillCreateRequest{
		Amount:    &amt,
		Frequency: models.BillFrequencyYearly,
		Schedule:  &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtrH(1), Weekday: "sunday"},
		Recipient: "Invalid Bill",
	})
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for monthly scope on yearly frequency, got %d", wBad.Code)
	}

	// Valid bill linked to account + category + reminder + short name
	paid := "2026-10-01"
	rem := true
	wCreate := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills", token, models.BillCreateRequest{
		AccountID:    &accID,
		CategoryID:   &cat.ID,
		Amount:       &amt,
		LastDatePaid: &paid,
		Frequency:    models.BillFrequencyMonthly,
		Schedule:     &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtrH(1)},
		Name:         "Gas",
		Recipient:    "Natural Gas Utility",
		Description:  "Monthly heating bill",
		Reminder:     &rem,
	})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating bill, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var created models.BillResponse
	_ = json.Unmarshal(wCreate.Body.Bytes(), &created)
	if created.Name != "Gas" {
		t.Fatalf("expected bill name='Gas', got %q", created.Name)
	}

	// Short name must be under 20 characters over HTTP
	wLongName := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills", token, models.BillCreateRequest{
		Amount:    &amt,
		Frequency: models.BillFrequencyMonthly,
		Name:      "TwentyCharactersName",
		Recipient: "Electric Utility",
	})
	if wLongName.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for 20-char bill name, got %d", wLongName.Code)
	}

	// Non-owner receives 404 (bills cannot be shared)
	wOther := doAuthedJSON(t, r, http.MethodGet, "/api/v1/bills/"+created.ID, otherTok, nil)
	if wOther.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner accessing bill, got %d", wOther.Code)
	}

	// Reminders endpoint returns overdue/upcoming reminder bills
	wRem := doAuthedJSON(t, r, http.MethodGet, "/api/v1/bills/reminders", token, nil)
	if wRem.Code != http.StatusOK {
		t.Fatalf("expected 200 on reminders, got %d", wRem.Code)
	}
	var remList []models.BillResponse
	_ = json.Unmarshal(wRem.Body.Bytes(), &remList)
	if len(remList) != 1 || remList[0].ID != created.ID {
		t.Fatalf("expected 1 reminder matching created bill, got %+v", remList)
	}

	// Apply to account creates checking record and updates last_date_paid
	applyDate := "2026-11-17"
	wApply := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills/"+created.ID+"/apply", token, models.BillApplyRequest{
		Date: &applyDate,
	})
	if wApply.Code != http.StatusCreated {
		t.Fatalf("expected 201 applying bill, got %d: %s", wApply.Code, wApply.Body.String())
	}
	var applied models.BillApplyResponse
	_ = json.Unmarshal(wApply.Body.Bytes(), &applied)
	if applied.Record.Value != -88.50 || applied.Bill.LastDatePaid == nil || *applied.Bill.LastDatePaid != applyDate {
		t.Fatalf("unexpected apply response: %+v", applied)
	}

	// Deleting custom category clears bill category_id without deleting bill
	if w := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/categories/"+cat.ID, token, nil); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting category, got %d", w.Code)
	}
	var afterCat models.BillResponse
	wGet := doAuthedJSON(t, r, http.MethodGet, "/api/v1/bills/"+created.ID, token, nil)
	_ = json.Unmarshal(wGet.Body.Bytes(), &afterCat)
	if afterCat.CategoryID != nil || afterCat.AccountID == nil {
		t.Fatalf("expected category_id=nil and account_id kept, got %+v", afterCat)
	}

	// Deleting linked account clears both account_id and category_id
	if w := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/accounts/"+accID, token, nil); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting account, got %d", w.Code)
	}
	var afterAcc models.BillResponse
	wGet2 := doAuthedJSON(t, r, http.MethodGet, "/api/v1/bills/"+created.ID, token, nil)
	_ = json.Unmarshal(wGet2.Body.Bytes(), &afterAcc)
	if afterAcc.AccountID != nil || afterAcc.CategoryID != nil {
		t.Fatalf("expected account_id=nil and category_id=nil after account delete, got %+v", afterAcc)
	}

	// Interval validation & every-6-months due calculation over HTTP
	wBadInt := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills", token, models.BillCreateRequest{
		Amount: &amt, Frequency: models.BillFrequencyMonthly, Interval: intPtrH(0), Recipient: "Bad Interval",
	})
	if wBadInt.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for interval=0, got %d", wBadInt.Code)
	}

	semiPaid := "2026-11-16"
	wSemi := doAuthedJSON(t, r, http.MethodPost, "/api/v1/bills", token, models.BillCreateRequest{
		Amount:       &amt,
		LastDatePaid: &semiPaid,
		Frequency:    models.BillFrequencyMonthly,
		Interval:     intPtrH(6),
		Recipient:    "Semi-Annual Insurance",
	})
	if wSemi.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating 6-month bill, got %d: %s", wSemi.Code, wSemi.Body.String())
	}
	var semi models.BillResponse
	_ = json.Unmarshal(wSemi.Body.Bytes(), &semi)
	if semi.Interval != 6 || semi.NextDueDate == nil || *semi.NextDueDate != "2027-05-16" {
		t.Fatalf("expected interval=6 and next_due_date=2027-05-16, got %+v", semi)
	}
}

func intPtrH(v int) *int { return &v }
