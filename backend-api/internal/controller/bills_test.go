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

func setupBillTestController(t *testing.T) (*controller.Controller, *repository.MemoryRepository, string, string) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-key", 3600)
	ctrl := controller.NewController(repo, authSvc)

	ctx := context.Background()
	r1, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "billowner", Email: "billowner@example.com", Password: "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser 1 failed: %v", err)
	}
	r2, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "billother", Email: "billother@example.com", Password: "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser 2 failed: %v", err)
	}
	return ctrl, repo, r1.ID, r2.ID
}

func TestBillScheduleAndFrequencyValidation(t *testing.T) {
	ctrl, _, u1, _ := setupBillTestController(t)
	ctx := context.Background()

	validCases := []struct {
		name string
		freq string
		sch  *models.BillSchedule
	}{
		{"null schedule monthly", models.BillFrequencyMonthly, nil},
		{"null schedule weekly", models.BillFrequencyWeekly, nil},
		{"null schedule yearly", models.BillFrequencyYearly, nil},
		{"weekly Friday", models.BillFrequencyWeekly, &models.BillSchedule{Type: "weekday", Scope: "week", Weekday: "Friday"}},
		{"monthly 3rd Thursday", models.BillFrequencyMonthly, &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(3), Weekday: "thursday"}},
		{"monthly 1st Monday", models.BillFrequencyMonthly, &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "Monday"}},
		{"monthly 1st of month", models.BillFrequencyMonthly, &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(1)}},
		{"yearly 1st of year", models.BillFrequencyYearly, &models.BillSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)}},
		{"yearly 1st Sunday of year", models.BillFrequencyYearly, &models.BillSchedule{Type: "nth_weekday", Scope: "year", Nth: intPtr(1), Weekday: "sunday"}},
		{"yearly month_day Apr 15", models.BillFrequencyYearly, &models.BillSchedule{Type: "month_day", Scope: "year", Month: intPtr(4), Day: intPtr(15)}},
	}

	for _, tc := range validCases {
		res, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
			Amount: floatPtr(-50.00), Frequency: tc.freq, Schedule: tc.sch, Recipient: tc.name,
		})
		if err != nil {
			t.Errorf("%s: expected valid schedule, got error: %v", tc.name, err)
		} else if res.Frequency != tc.freq {
			t.Errorf("%s: expected frequency %q, got %q", tc.name, tc.freq, res.Frequency)
		}
	}

	invalidCases := []struct {
		name string
		freq string
		sch  *models.BillSchedule
	}{
		{"invalid frequency", "daily", nil},
		{"1st Sunday of month on yearly", models.BillFrequencyYearly, &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "sunday"}},
		{"1st of year on monthly", models.BillFrequencyMonthly, &models.BillSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)}},
		{"month_day on monthly", models.BillFrequencyMonthly, &models.BillSchedule{Type: "month_day", Scope: "month", Month: intPtr(3), Day: intPtr(1)}},
		{"nth_weekday on weekly", models.BillFrequencyWeekly, &models.BillSchedule{Type: "nth_weekday", Scope: "week", Nth: intPtr(1), Weekday: "monday"}},
		{"monthly day 32", models.BillFrequencyMonthly, &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(32)}},
		{"yearly day 367", models.BillFrequencyYearly, &models.BillSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(367)}},
		{"yearly Feb 30", models.BillFrequencyYearly, &models.BillSchedule{Type: "month_day", Scope: "year", Month: intPtr(2), Day: intPtr(30)}},
		{"monthly 6th weekday", models.BillFrequencyMonthly, &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(6), Weekday: "tuesday"}},
		{"invalid weekday", models.BillFrequencyWeekly, &models.BillSchedule{Type: "weekday", Scope: "week", Weekday: "funday"}},
	}

	for _, tc := range invalidCases {
		_, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
			Amount: floatPtr(-50.00), Frequency: tc.freq, Schedule: tc.sch, Recipient: tc.name,
		})
		if err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
			continue
		}
		if appErr, ok := err.(*controller.AppError); !ok || appErr.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400 AppError, got %v", tc.name, err)
		}
	}

	// Interval must be >= 1; default is 1 when nil
	for _, badInterval := range []int{0, -1, -6} {
		_, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
			Amount: floatPtr(-50.00), Frequency: models.BillFrequencyMonthly, Interval: intPtr(badInterval), Recipient: "Bad Interval",
		})
		if appErr, ok := err.(*controller.AppError); !ok || appErr.Status != http.StatusBadRequest {
			t.Errorf("interval %d: expected 400 AppError, got %v", badInterval, err)
		}
	}
}

func TestComputeBillDueInfo_SchedulesAndReminders(t *testing.T) {
	refToday := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		bill           models.Bill
		expectedDue    string
		expectedStatus string
	}{
		{
			name: "third Thursday of month upcoming in 2 days",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-10-15"),
				Schedule: &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(3), Weekday: "thursday"},
			},
			expectedDue: "2026-11-19", expectedStatus: models.BillStatusUpcoming,
		},
		{
			name: "first Monday of month after Nov 2 goes to Dec 7 scheduled",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-11-02"),
				Schedule: &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(1), Weekday: "monday"},
			},
			expectedDue: "2026-12-07", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "1st of month overdue when last paid Sep 1",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-09-01"),
				Schedule: &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(1)},
			},
			expectedDue: "2026-10-01", expectedStatus: models.BillStatusOverdue,
		},
		{
			name: "1st Sunday of the year advances to Jan 3 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyYearly, LastDatePaid: strPtr("2026-01-04"),
				Schedule: &models.BillSchedule{Type: "nth_weekday", Scope: "year", Nth: intPtr(1), Weekday: "sunday"},
			},
			expectedDue: "2027-01-03", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "null schedule monthly clamps Jan 31 to Feb 28",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-01-31"),
			},
			expectedDue: "2026-02-28", expectedStatus: models.BillStatusOverdue,
		},
		{
			name: "weekly without schedule adds 7 days",
			bill: models.Bill{
				Frequency: models.BillFrequencyWeekly, LastDatePaid: strPtr("2026-11-15"),
			},
			expectedDue: "2026-11-22", expectedStatus: models.BillStatusUpcoming,
		},
		{
			name: "every 6 months without schedule from Nov 16 2026 yields May 16 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, Interval: 6, LastDatePaid: strPtr("2026-11-16"),
			},
			expectedDue: "2027-05-16", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "every 6 months with 16th-of-month schedule from Nov 16 2026 yields May 16 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, Interval: 6, LastDatePaid: strPtr("2026-11-16"),
				Schedule: &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(16)},
			},
			expectedDue: "2027-05-16", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "every 6 months from May 31 2026 clamps to Nov 30 2026",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, Interval: 6, LastDatePaid: strPtr("2026-05-31"),
			},
			expectedDue: "2026-11-30", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "every 2 weeks Friday schedule from Nov 13 2026 yields Nov 27 2026",
			bill: models.Bill{
				Frequency: models.BillFrequencyWeekly, Interval: 2, LastDatePaid: strPtr("2026-11-13"),
				Schedule: &models.BillSchedule{Type: "weekday", Scope: "week", Weekday: "friday"},
			},
			expectedDue: "2026-11-27", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "every 2 years 1st Sunday of year from Jan 4 2026 yields Jan 2 2028",
			bill: models.Bill{
				Frequency: models.BillFrequencyYearly, Interval: 2, LastDatePaid: strPtr("2026-01-04"),
				Schedule: &models.BillSchedule{Type: "nth_weekday", Scope: "year", Nth: intPtr(1), Weekday: "sunday"},
			},
			expectedDue: "2028-01-02", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "15th of the month advances from Nov 15 to Dec 15",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-11-15"),
				Schedule: &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(15)},
			},
			expectedDue: "2026-12-15", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "last of the month day 31 clamps Jan 31 to Feb 28 and Mar 31 to Apr 30",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-03-31"),
				Schedule: &models.BillSchedule{Type: "day_of_period", Scope: "month", Day: intPtr(31)},
			},
			expectedDue: "2026-04-30", expectedStatus: models.BillStatusOverdue,
		},
		{
			name: "every 1st of the year advances Jan 1 2026 to Jan 1 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyYearly, LastDatePaid: strPtr("2026-01-01"),
				Schedule: &models.BillSchedule{Type: "day_of_period", Scope: "year", Day: intPtr(1)},
			},
			expectedDue: "2027-01-01", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "April 14th every year advances Apr 14 2026 to Apr 14 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyYearly, LastDatePaid: strPtr("2026-04-14"),
				Schedule: &models.BillSchedule{Type: "month_day", Scope: "year", Month: intPtr(4), Day: intPtr(14)},
			},
			expectedDue: "2027-04-14", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "Last Monday of the month from Oct 26 2026 yields Nov 30 2026",
			bill: models.Bill{
				Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-10-26"),
				Schedule: &models.BillSchedule{Type: "nth_weekday", Scope: "month", Nth: intPtr(-1), Weekday: "monday"},
			},
			expectedDue: "2026-11-30", expectedStatus: models.BillStatusScheduled,
		},
		{
			name: "November 1st every year advances Nov 1 2026 to Nov 1 2027",
			bill: models.Bill{
				Frequency: models.BillFrequencyYearly, LastDatePaid: strPtr("2026-11-01"),
				Schedule: &models.BillSchedule{Type: "month_day", Scope: "year", Month: intPtr(11), Day: intPtr(1)},
			},
			expectedDue: "2027-11-01", expectedStatus: models.BillStatusScheduled,
		},
	}

	for _, tc := range tests {
		due, status, _ := controller.ComputeBillDueInfo(&tc.bill, refToday)
		if due == nil || *due != tc.expectedDue || status != tc.expectedStatus {
			t.Errorf("%s: got due=%v status=%s, want due=%s status=%s", tc.name, due, status, tc.expectedDue, tc.expectedStatus)
		}
	}
}

func TestBillAccountCategoryAndSharingRules(t *testing.T) {
	ctrl, _, u1, u2 := setupBillTestController(t)
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

	// Category without account fails
	if _, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-75), Frequency: models.BillFrequencyMonthly, Recipient: "Electric Co", CategoryID: &cat1.ID,
	}); err == nil {
		t.Fatal("expected error linking category without account")
	}

	// Snapshot account rejected
	if _, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-75), Frequency: models.BillFrequencyMonthly, Recipient: "Electric Co", AccountID: &snap.ID,
	}); err == nil {
		t.Fatal("expected error linking bill to snapshot account")
	}

	// Other user's custom category rejected
	if _, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-75), Frequency: models.BillFrequencyMonthly, Recipient: "Electric Co", AccountID: &chk.ID, CategoryID: &cat2.ID,
	}); err == nil {
		t.Fatal("expected error using another user's private category")
	}

	// Valid linked bill + custom category + reminder + short name
	bill, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-110.25), Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-10-01"),
		AccountID: &chk.ID, CategoryID: &cat1.ID, Name: "Electric", Recipient: "Electric Utility", Description: "Power service", Reminder: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}
	if bill.Name != "Electric" {
		t.Fatalf("expected bill Name='Electric', got %q", bill.Name)
	}

	// Short name must be strictly under 20 characters (19 allowed, 20 rejected)
	if _, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-45), Frequency: models.BillFrequencyMonthly, Name: "1234567890123456789", Recipient: "Water Utility",
	}); err != nil {
		t.Fatalf("expected 19-char bill name to succeed, got %v", err)
	}
	if _, err := ctrl.CreateBill(ctx, u1, models.BillCreateRequest{
		Amount: floatPtr(-45), Frequency: models.BillFrequencyMonthly, Name: "12345678901234567890", Recipient: "Water Utility",
	}); err == nil {
		t.Fatal("expected 20-char bill name to be rejected")
	}

	// UpdateBill preserves updated short name and recipient stays intact on apply
	updatedBill, err := ctrl.UpdateBill(ctx, u1, bill.ID, models.BillUpdateRequest{
		Amount: floatPtr(-110.25), Frequency: models.BillFrequencyMonthly, LastDatePaid: strPtr("2026-10-01"),
		AccountID: &chk.ID, CategoryID: &cat1.ID, Name: "Power", Recipient: "Electric Utility", Description: "Power service", Reminder: boolPtr(true),
	})
	if err != nil || updatedBill.Name != "Power" {
		t.Fatalf("expected updated bill Name='Power', got %+v err=%v", updatedBill, err)
	}

	// Sharing prohibited: u2 cannot access or mutate u1's bill
	if _, err := ctrl.GetBill(ctx, u2, bill.ID); err != controller.ErrBillNotFound {
		t.Fatalf("expected ErrBillNotFound for non-owner user, got %v", err)
	}

	// Apply bill creates transaction in linked checking account and updates last_date_paid
	applyRes, err := ctrl.ApplyBillToAccount(ctx, u1, bill.ID, models.BillApplyRequest{Date: strPtr("2026-11-17")})
	if err != nil {
		t.Fatalf("ApplyBillToAccount failed: %v", err)
	}
	if applyRes.Record.Value != -110.25 || applyRes.Record.DateSent != "2026-11-17" || *applyRes.Record.CategoryID != cat1.ID {
		t.Fatalf("unexpected applied record: %+v", applyRes.Record)
	}
	if applyRes.Bill.LastDatePaid == nil || *applyRes.Bill.LastDatePaid != "2026-11-17" {
		t.Fatalf("expected bill LastDatePaid=2026-11-17, got %v", applyRes.Bill.LastDatePaid)
	}

	// Delete custom category -> bill remains valid with category_id = nil
	if err := ctrl.DeleteCategory(ctx, u1, cat1.ID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}
	afterCatDel, err := ctrl.GetBill(ctx, u1, bill.ID)
	if err != nil || afterCatDel.CategoryID != nil || afterCatDel.AccountID == nil {
		t.Fatalf("expected CategoryID=nil and AccountID kept, got %+v err=%v", afterCatDel, err)
	}

	// Delete linked account -> bill remains valid with account_id = nil and category_id = nil
	if err := ctrl.DeleteAccount(ctx, u1, chk.ID); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	afterAccDel, err := ctrl.GetBill(ctx, u1, bill.ID)
	if err != nil || afterAccDel.AccountID != nil || afterAccDel.CategoryID != nil {
		t.Fatalf("expected AccountID=nil and CategoryID=nil, got %+v err=%v", afterAccDel, err)
	}
}
