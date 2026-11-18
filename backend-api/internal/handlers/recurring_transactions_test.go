package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/plannedwallet/internal/models"
)

func TestRecurringTransactionsHandlers_LifecycleAndCascades(t *testing.T) {
	r, _ := setupTestRouter()
	token := registerAndGetToken(t, r, "hrt1", "hrt1@example.com", "SecurePass123!")
	otherTok := registerAndGetToken(t, r, "hrt2", "hrt2@example.com", "SecurePass123!")
	accID := createAccount(t, r, token, "Everyday Checking", "checking", "Recurring checking")

	wCat := doAuthedJSON(t, r, http.MethodPost, "/api/v1/categories", token, models.CategoryCreateRequest{Name: "Heating Utility"})
	if wCat.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating category, got %d", wCat.Code)
	}
	var cat models.CategoryResponse
	_ = json.Unmarshal(wCat.Body.Bytes(), &cat)

	// Invalid schedule on yearly frequency returns 400
	negAmt := -88.50
	wBad := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions", token, models.RecurringTransactionRequest{
		Amount:    &negAmt,
		Frequency: models.RecurringFrequencyYearly,
		Schedule:  &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtrH(1), Weekday: "sunday"},
		Entity:    "Invalid Bill",
	})
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for monthly scope on yearly frequency, got %d", wBad.Code)
	}

	// Short name >= 20 characters returns 400
	wLongName := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions", token, models.RecurringTransactionRequest{
		Amount:    &negAmt,
		Frequency: models.RecurringFrequencyMonthly,
		Name:      "TwentyCharactersName",
		Entity:    "Electric Utility",
	})
	if wLongName.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for 20-char name, got %d", wLongName.Code)
	}

	// Create negative recurring transaction (bill) and positive recurring transaction (income)
	applied := "2026-10-01"
	rem := true
	wBill := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions", token, models.RecurringTransactionRequest{
		AccountID:       &accID,
		CategoryID:      &cat.ID,
		Amount:          &negAmt,
		LastDateApplied: &applied,
		Frequency:       models.RecurringFrequencyMonthly,
		Schedule:        &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtrH(1)},
		Name:            "Gas",
		Entity:          "Natural Gas Utility",
		Description:     "Monthly heating bill",
		Reminder:        &rem,
	})
	if wBill.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating bill, got %d: %s", wBill.Code, wBill.Body.String())
	}
	var createdBill models.RecurringTransactionResponse
	_ = json.Unmarshal(wBill.Body.Bytes(), &createdBill)
	if createdBill.Name != "Gas" {
		t.Fatalf("expected name='Gas', got %q", createdBill.Name)
	}

	posAmt := 2500.00
	wIncome := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions", token, models.RecurringTransactionRequest{
		AccountID:       &accID,
		Amount:          &posAmt,
		LastDateApplied: &applied,
		Frequency:       models.RecurringFrequencyMonthly,
		Schedule:        &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtrH(15)},
		Name:            "Paycheck",
		Entity:          "Employer Payroll",
		Description:     "Monthly paycheck",
		Reminder:        &rem,
	})
	if wIncome.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating income, got %d: %s", wIncome.Code, wIncome.Body.String())
	}
	var createdIncome models.RecurringTransactionResponse
	_ = json.Unmarshal(wIncome.Body.Bytes(), &createdIncome)

	// Non-owner receives 404
	if wOther := doAuthedJSON(t, r, http.MethodGet, "/api/v1/recurring-transactions/"+createdBill.ID, otherTok, nil); wOther.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner accessing recurring transaction, got %d", wOther.Code)
	}

	// Reminders returns both overdue bill and overdue income
	wRem := doAuthedJSON(t, r, http.MethodGet, "/api/v1/recurring-transactions/reminders", token, nil)
	if wRem.Code != http.StatusOK {
		t.Fatalf("expected 200 on reminders, got %d", wRem.Code)
	}
	var remList []models.RecurringTransactionResponse
	_ = json.Unmarshal(wRem.Body.Bytes(), &remList)
	if len(remList) != 2 {
		t.Fatalf("expected 2 reminders (bill + income), got %d", len(remList))
	}

	// Apply both to account and verify checking record signs & last_date_applied
	applyDate := "2026-11-17"
	wApplyInc := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions/"+createdIncome.ID+"/apply", token, models.RecurringTransactionApplyRequest{
		Date: &applyDate,
	})
	if wApplyInc.Code != http.StatusCreated {
		t.Fatalf("expected 201 applying income, got %d: %s", wApplyInc.Code, wApplyInc.Body.String())
	}
	var appliedInc models.RecurringTransactionApplyResponse
	_ = json.Unmarshal(wApplyInc.Body.Bytes(), &appliedInc)
	if appliedInc.Record.Value != 2500.00 || appliedInc.RecurringTransaction.LastDateApplied == nil || *appliedInc.RecurringTransaction.LastDateApplied != applyDate {
		t.Fatalf("unexpected apply income response: %+v", appliedInc)
	}

	// Category delete and account delete cascade
	if w := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/categories/"+cat.ID, token, nil); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting category, got %d", w.Code)
	}
	var afterCat models.RecurringTransactionResponse
	wGet := doAuthedJSON(t, r, http.MethodGet, "/api/v1/recurring-transactions/"+createdBill.ID, token, nil)
	_ = json.Unmarshal(wGet.Body.Bytes(), &afterCat)
	if afterCat.CategoryID != nil || afterCat.AccountID == nil {
		t.Fatalf("expected category_id=nil and account_id kept, got %+v", afterCat)
	}

	if w := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/accounts/"+accID, token, nil); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting account, got %d", w.Code)
	}
	var afterAcc models.RecurringTransactionResponse
	wGet2 := doAuthedJSON(t, r, http.MethodGet, "/api/v1/recurring-transactions/"+createdBill.ID, token, nil)
	_ = json.Unmarshal(wGet2.Body.Bytes(), &afterAcc)
	if afterAcc.AccountID != nil || afterAcc.CategoryID != nil {
		t.Fatalf("expected account_id=nil and category_id=nil after account delete, got %+v", afterAcc)
	}

	// Update + Delete
	wUpd := doAuthedJSON(t, r, http.MethodPut, "/api/v1/recurring-transactions/"+createdIncome.ID, token, models.RecurringTransactionRequest{
		Amount:    &posAmt,
		Frequency: models.RecurringFrequencyMonthly,
		Name:      "Salary",
		Entity:    "Employer Payroll",
	})
	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 updating recurring transaction, got %d", wUpd.Code)
	}
	if wDel := doAuthedJSON(t, r, http.MethodDelete, "/api/v1/recurring-transactions/"+createdIncome.ID, token, nil); wDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting recurring transaction, got %d", wDel.Code)
	}

	// Bi-monthly payroll with 15th and last day flavor
	biPaid := "2026-11-15"
	wBi := doAuthedJSON(t, r, http.MethodPost, "/api/v1/recurring-transactions", token, models.RecurringTransactionRequest{
		Amount:          &posAmt,
		LastDateApplied: &biPaid,
		Frequency:       models.RecurringFrequencyBiMonthly,
		Schedule:        &models.RecurringSchedule{Type: "fifteenth_last", Scope: "bi-monthly"},
		Name:            "BiMonthly",
		Entity:          "Employer Payroll",
	})
	if wBi.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating bi-monthly recurring transaction, got %d: %s", wBi.Code, wBi.Body.String())
	}
	var biRes models.RecurringTransactionResponse
	_ = json.Unmarshal(wBi.Body.Bytes(), &biRes)
	if biRes.Frequency != "bi-monthly" || biRes.NextDueDate == nil || *biRes.NextDueDate != "2026-11-30" {
		t.Fatalf("expected bi-monthly next_due_date=2026-11-30, got %+v", biRes)
	}
}

func intPtrH(v int) *int { return &v }
