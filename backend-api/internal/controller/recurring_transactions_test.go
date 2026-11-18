package controller_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

func intPtr(v int) *int           { return &v }
func boolPtr(v bool) *bool        { return &v }
func floatPtr(v float64) *float64 { return &v }
func strPtr(v string) *string     { return &v }

func setupRecurringTestController(t *testing.T) (*controller.Controller, *repository.MemoryRepository, string, string) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-key", 3600)
	ctrl := controller.NewController(repo, authSvc)

	ctx := context.Background()
	r1, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "rtowner", Email: "rtowner@example.com", Password: "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser 1 failed: %v", err)
	}
	r2, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "rtother", Email: "rtother@example.com", Password: "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser 2 failed: %v", err)
	}
	return ctrl, repo, r1.ID, r2.ID
}

func TestRecurringScheduleAndFrequencyValidation(t *testing.T) {
	ctrl, _, u1, _ := setupRecurringTestController(t)
	ctx := context.Background()

	validCases := []struct {
		name   string
		amount float64
		freq   string
		sch    *models.RecurringSchedule
	}{
		{"null schedule monthly bill", -50.00, models.RecurringFrequencyMonthly, nil},
		{"null schedule weekly income", 500.00, models.RecurringFrequencyWeekly, nil},
		{"null schedule yearly", -120.00, models.RecurringFrequencyYearly, nil},
		{"weekly Friday", -40.00, models.RecurringFrequencyWeekly, &models.RecurringSchedule{Type: "weekday", Scope: "week", Weekday: "Friday"}},
		{"bi-monthly 1st and 15th", 2500.00, models.RecurringFrequencyBiMonthly, &models.RecurringSchedule{Type: "first_fifteenth", Scope: "bi-monthly"}},
		{"bi-monthly 15th and last", 2500.00, models.RecurringFrequencyBiMonthly, &models.RecurringSchedule{Type: "fifteenth_last", Scope: "bi-monthly"}},
		{"monthly 3rd Thursday", -150.00, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(3), Weekday: "thursday"}},
		{"monthly 1st Monday", -1200.00, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "Monday"}},
		{"monthly 1st of month", -95.00, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(1)}},
		{"yearly 1st of year", -500.00, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)}},
		{"yearly 1st Sunday of year", -320.00, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "nth_weekday", Scope: "year", Nth: intPtr(1), Weekday: "sunday"}},
		{"yearly month_day Apr 14", 1000.00, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "month_day", Scope: "year", Month: intPtr(4), Day: intPtr(14)}},
	}

	for _, tc := range validCases {
		res, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
			Amount: floatPtr(tc.amount), Frequency: tc.freq, Schedule: tc.sch, Entity: tc.name,
		})
		if err != nil {
			t.Errorf("%s: expected valid schedule, got error: %v", tc.name, err)
		} else if res.Frequency != tc.freq || res.Amount != tc.amount {
			t.Errorf("%s: unexpected result %+v", tc.name, res)
		}
	}

	invalidCases := []struct {
		name   string
		amount float64
		freq   string
		sch    *models.RecurringSchedule
	}{
		{"zero amount", 0, models.RecurringFrequencyMonthly, nil},
		{"invalid frequency", -50, "daily", nil},
		{"bi-monthly wrong scope", 2500, models.RecurringFrequencyBiMonthly, &models.RecurringSchedule{Type: "first_fifteenth", Scope: "month"}},
		{"bi-monthly wrong type", 2500, models.RecurringFrequencyBiMonthly, &models.RecurringSchedule{Type: "day_of_period", Scope: "bi-monthly"}},
		{"1st Sunday of month on yearly", -50, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "sunday"}},
		{"1st of year on monthly", -50, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)}},
		{"monthly day 32", -50, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(32)}},
		{"yearly day 367", -50, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(367)}},
		{"yearly Feb 30", -50, models.RecurringFrequencyYearly, &models.RecurringSchedule{Type: "month_day", Scope: "year", Month: intPtr(2), Day: intPtr(30)}},
		{"monthly 6th weekday", -50, models.RecurringFrequencyMonthly, &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(6), Weekday: "tuesday"}},
		{"invalid weekday", -50, models.RecurringFrequencyWeekly, &models.RecurringSchedule{Type: "weekday", Scope: "week", Weekday: "funday"}},
	}

	for _, tc := range invalidCases {
		_, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
			Amount: floatPtr(tc.amount), Frequency: tc.freq, Schedule: tc.sch, Entity: tc.name,
		})
		if appErr, ok := err.(*controller.AppError); !ok || appErr.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400 AppError, got %v", tc.name, err)
		}
	}

	for _, badInterval := range []int{0, -1, -6} {
		_, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
			Amount: floatPtr(-50.00), Frequency: models.RecurringFrequencyMonthly, Interval: intPtr(badInterval), Entity: "Bad Interval",
		})
		if appErr, ok := err.(*controller.AppError); !ok || appErr.Status != http.StatusBadRequest {
			t.Errorf("interval %d: expected 400 AppError, got %v", badInterval, err)
		}
	}
}

func TestComputeRecurringDueInfo_SchedulesAndIntervals(t *testing.T) {
	refToday := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		rt             models.RecurringTransaction
		expectedDue    string
		expectedStatus string
	}{
		{
			name: "15th of the month advances from Nov 15 to Dec 15",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-11-15"),
				Schedule: &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(15)},
			},
			expectedDue: "2026-12-15", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "last of the month day 31 clamps Mar 31 to Apr 30",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-03-31"),
				Schedule: &models.RecurringSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(31)},
			},
			expectedDue: "2026-04-30", expectedStatus: models.RecurringStatusOverdue,
		},
		{
			name: "every 6 months without schedule from Nov 16 2026 yields May 16 2027",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyMonthly, Interval: 6, LastDateApplied: strPtr("2026-11-16"),
			},
			expectedDue: "2027-05-16", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "every 1st of the year advances Jan 1 2026 to Jan 1 2027",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyYearly, LastDateApplied: strPtr("2026-01-01"),
				Schedule: &models.RecurringSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)},
			},
			expectedDue: "2027-01-01", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "April 14th every year advances Apr 14 2026 to Apr 14 2027",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyYearly, LastDateApplied: strPtr("2026-04-14"),
				Schedule: &models.RecurringSchedule{Type: "month_day", Scope: "year", Month: intPtr(4), Day: intPtr(14)},
			},
			expectedDue: "2027-04-14", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "first Monday of month after Nov 2 goes to Dec 7 scheduled",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-11-02"),
				Schedule: &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "monday"},
			},
			expectedDue: "2026-12-07", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "Last Monday of the month from Oct 26 2026 yields Nov 30 2026",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-10-26"),
				Schedule: &models.RecurringSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(-1), Weekday: "monday"},
			},
			expectedDue: "2026-11-30", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "November 1st every year advances Nov 1 2026 to Nov 1 2027",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyYearly, LastDateApplied: strPtr("2026-11-01"),
				Schedule: &models.RecurringSchedule{Type: "month_day", Scope: "year", Month: intPtr(11), Day: intPtr(1)},
			},
			expectedDue: "2027-11-01", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "bi-monthly 1st and 15th after Nov 1 yields Nov 15",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyBiMonthly, LastDateApplied: strPtr("2026-11-01"),
				Schedule: &models.RecurringSchedule{Type: "first_fifteenth", Scope: "bi-monthly"},
			},
			expectedDue: "2026-11-15", expectedStatus: models.RecurringStatusOverdue,
		},
		{
			name: "bi-monthly 1st and 15th after Nov 15 yields Dec 1",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyBiMonthly, LastDateApplied: strPtr("2026-11-15"),
				Schedule: &models.RecurringSchedule{Type: "first_fifteenth", Scope: "bi-monthly"},
			},
			expectedDue: "2026-12-01", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "bi-monthly 15th and last after Nov 15 yields Nov 30",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyBiMonthly, LastDateApplied: strPtr("2026-11-15"),
				Schedule: &models.RecurringSchedule{Type: "fifteenth_last", Scope: "bi-monthly"},
			},
			expectedDue: "2026-11-30", expectedStatus: models.RecurringStatusScheduled,
		},
		{
			name: "bi-monthly 15th and last in non-leap Feb yields Feb 28",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyBiMonthly, LastDateApplied: strPtr("2026-02-15"),
				Schedule: &models.RecurringSchedule{Type: "fifteenth_last", Scope: "bi-monthly"},
			},
			expectedDue: "2026-02-28", expectedStatus: models.RecurringStatusOverdue,
		},
		{
			name: "bi-monthly 15th and last after Nov 30 yields Dec 15",
			rt: models.RecurringTransaction{
				Frequency: models.RecurringFrequencyBiMonthly, LastDateApplied: strPtr("2026-11-30"),
				Schedule: &models.RecurringSchedule{Type: "fifteenth_last", Scope: "bi-monthly"},
			},
			expectedDue: "2026-12-15", expectedStatus: models.RecurringStatusScheduled,
		},
	}

	for _, tc := range tests {
		due, status, _ := controller.ComputeRecurringDueInfo(&tc.rt, refToday)
		if due == nil || *due != tc.expectedDue || status != tc.expectedStatus {
			t.Errorf("%s: got due=%v status=%s, want due=%s status=%s", tc.name, due, status, tc.expectedDue, tc.expectedStatus)
		}
	}
}

func TestRecurringAccountCategoryNameAndApply(t *testing.T) {
	ctrl, _, u1, u2 := setupRecurringTestController(t)
	ctx := context.Background()

	chk, err := ctrl.CreateAccount(ctx, u1, models.AccountCreationRequest{Name: "Checking", Type: "checking", Description: "Checking account"})
	if err != nil {
		t.Fatalf("CreateAccount checking failed: %v", err)
	}
	snap, err := ctrl.CreateAccount(ctx, u1, models.AccountCreationRequest{Name: "Brokerage", Type: "snapshot", Description: "Snapshot account"})
	if err != nil {
		t.Fatalf("CreateAccount snapshot failed: %v", err)
	}
	cat1, err := ctrl.CreateCategory(ctx, u1, models.CategoryCreateRequest{Name: "Electric Bill Cat"})
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	cat2, err := ctrl.CreateCategory(ctx, u2, models.CategoryCreateRequest{Name: "Private Other Cat"})
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Category without account fails, snapshot fails, other user's category fails
	if _, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-75), Frequency: models.RecurringFrequencyMonthly, Entity: "Electric Co", CategoryID: &cat1.ID,
	}); err == nil {
		t.Fatal("expected error linking category without account")
	}
	if _, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-75), Frequency: models.RecurringFrequencyMonthly, Entity: "Electric Co", AccountID: &snap.ID,
	}); err == nil {
		t.Fatal("expected error linking recurring transaction to snapshot account")
	}
	if _, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-75), Frequency: models.RecurringFrequencyMonthly, Entity: "Electric Co", AccountID: &chk.ID, CategoryID: &cat2.ID,
	}); err == nil {
		t.Fatal("expected error using another user's private category")
	}

	// Short name must be strictly under 20 chars
	if _, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-45), Frequency: models.RecurringFrequencyMonthly, Name: "1234567890123456789", Entity: "Water Utility",
	}); err != nil {
		t.Fatalf("expected 19-char name to succeed, got %v", err)
	}
	if _, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-45), Frequency: models.RecurringFrequencyMonthly, Name: "12345678901234567890", Entity: "Water Utility",
	}); err == nil {
		t.Fatal("expected 20-char name to be rejected")
	}

	// Negative bill + Positive income stored and applied through unified API
	bill, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(-110.25), Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-10-01"),
		AccountID: &chk.ID, CategoryID: &cat1.ID, Name: "Electric", Entity: "Electric Utility", Description: "Power service", Reminder: boolPtr(true),
	})
	if err != nil || bill.Name != "Electric" {
		t.Fatalf("CreateRecurringTransaction bill failed: %+v err=%v", bill, err)
	}
	income, err := ctrl.CreateRecurringTransaction(ctx, u1, models.RecurringTransactionRequest{
		Amount: floatPtr(2500.00), Frequency: models.RecurringFrequencyMonthly, LastDateApplied: strPtr("2026-10-01"),
		AccountID: &chk.ID, Name: "Paycheck", Entity: "Employer Payroll", Description: "Direct deposit", Reminder: boolPtr(true),
	})
	if err != nil || income.Name != "Paycheck" {
		t.Fatalf("CreateRecurringTransaction income failed: %+v err=%v", income, err)
	}

	if _, err := ctrl.GetRecurringTransaction(ctx, u2, bill.ID); err != controller.ErrRecurringTransactionNotFound {
		t.Fatalf("expected ErrRecurringTransactionNotFound for non-owner user, got %v", err)
	}

	applyBill, err := ctrl.ApplyRecurringTransactionToAccount(ctx, u1, bill.ID, models.RecurringTransactionApplyRequest{Date: strPtr("2026-11-17")})
	if err != nil || applyBill.Record.Value != -110.25 || applyBill.Record.Entity != "Electric Utility" {
		t.Fatalf("unexpected apply bill result: %+v err=%v", applyBill, err)
	}
	applyInc, err := ctrl.ApplyRecurringTransactionToAccount(ctx, u1, income.ID, models.RecurringTransactionApplyRequest{Date: strPtr("2026-11-17")})
	if err != nil || applyInc.Record.Value != 2500.00 || applyInc.Record.Entity != "Employer Payroll" {
		t.Fatalf("unexpected apply income result: %+v err=%v", applyInc, err)
	}

	// Category deletion clears category_id; account deletion clears account_id and category_id
	if err := ctrl.DeleteCategory(ctx, u1, cat1.ID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}
	afterCatDel, err := ctrl.GetRecurringTransaction(ctx, u1, bill.ID)
	if err != nil || afterCatDel.CategoryID != nil || afterCatDel.AccountID == nil {
		t.Fatalf("expected CategoryID=nil and AccountID kept, got %+v err=%v", afterCatDel, err)
	}
	if err := ctrl.DeleteAccount(ctx, u1, chk.ID); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	afterAccDel, err := ctrl.GetRecurringTransaction(ctx, u1, bill.ID)
	if err != nil || afterAccDel.AccountID != nil || afterAccDel.CategoryID != nil {
		t.Fatalf("expected AccountID=nil and CategoryID=nil, got %+v err=%v", afterAccDel, err)
	}
}
