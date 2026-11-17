package controller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
)

func assertAppStatus(t *testing.T, err error, wantStatus int) {
	t.Helper()
	var appErr *controller.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError with status %d, got %v", wantStatus, err)
	}
	if appErr.Status != wantStatus {
		t.Fatalf("expected status %d, got %d (%s)", wantStatus, appErr.Status, appErr.Code)
	}
}

func TestReconcileReportLifecycleAndImmutability(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	owner := registerTestUser(t, ctrl, "reconcileowner")
	acc, err := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{
		Name: "Credit Card", Type: "credit card", Description: "Monthly card",
	})
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	v1, v2 := -30.0, -70.0
	rec1, _ := ctrl.CreateCheckingRecord(ctx, owner.ID, acc.ID, models.CheckingRecordCreateRequest{
		Value: &v1, DateSent: "2026-11-02", Entity: "Bistro",
	})
	cleared := "2026-11-04"
	rec2, _ := ctrl.CreateCheckingRecord(ctx, owner.ID, acc.ID, models.CheckingRecordCreateRequest{
		Value: &v2, DateSent: "2026-11-03", DateCleared: &cleared, Entity: "Market",
	})

	startBal, badEnd, goodEnd := 0.0, -90.0, -100.0
	_, err = ctrl.CreateReconcileReport(ctx, owner.ID, acc.ID, models.ReconcileReportCreateRequest{
		Name: "Nov2026", ReportDate: "2026-11-16", StartingBalance: &startBal, EndingBalance: &badEnd, RecordIDs: []string{rec1.ID, rec2.ID},
	})
	assertAppStatus(t, err, http.StatusBadRequest)

	rep, err := ctrl.CreateReconcileReport(ctx, owner.ID, acc.ID, models.ReconcileReportCreateRequest{
		Name: "Nov2026", ReportDate: "2026-11-16", StartingBalance: &startBal, EndingBalance: &goodEnd, RecordIDs: []string{rec1.ID, rec2.ID},
	})
	if err != nil || rep.RecordCount != 2 {
		t.Fatalf("CreateReconcileReport failed: rep=%+v err=%v", rep, err)
	}

	// Uncleared record gets cleared date defaulted to report_date
	got1, _ := ctrl.GetCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID)
	if got1.ReconcileReportID == nil || got1.DateCleared == nil || *got1.DateCleared != "2026-11-16" {
		t.Fatalf("expected rec1 cleared date defaulted to 2026-11-16, got %+v", got1)
	}

	// Cannot attach already reconciled record to another report (409)
	nextEnd := -130.0
	_, err = ctrl.CreateReconcileReport(ctx, owner.ID, acc.ID, models.ReconcileReportCreateRequest{
		ReportDate: "2026-11-20", StartingBalance: &goodEnd, EndingBalance: &nextEnd, RecordIDs: []string{rec1.ID},
	})
	assertAppStatus(t, err, http.StatusConflict)

	// Editing amount, sent date, cleared date, or recipient returns 409 Conflict
	newVal := -35.0
	_, err = ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{Value: &newVal})
	assertAppStatus(t, err, http.StatusConflict)
	newDate := "2026-11-01"
	_, err = ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{DateSent: &newDate})
	assertAppStatus(t, err, http.StatusConflict)
	_, err = ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{DateCleared: &newDate})
	assertAppStatus(t, err, http.StatusConflict)
	newEnt := "Other Vendor"
	_, err = ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{Entity: &newEnt})
	assertAppStatus(t, err, http.StatusConflict)
	assertAppStatus(t, ctrl.DeleteCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID), http.StatusConflict)

	// Description and Category may still be modified
	cats, _ := ctrl.ListCategories(ctx, owner.ID)
	newDesc := "Dinner with tip"
	patched, err := ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{
		Description: &newDesc, CategoryID: &cats[0].ID,
	})
	if err != nil || patched.Description != newDesc || patched.CategoryID == nil || *patched.CategoryID != cats[0].ID {
		t.Fatalf("expected description and category patch to succeed on reconciled record: %+v err=%v", patched, err)
	}

	// Deleting reconcile report unlinks records without deleting them, allowing updates again
	if err := ctrl.DeleteReconcileReport(ctx, owner.ID, acc.ID, rep.ID); err != nil {
		t.Fatalf("DeleteReconcileReport failed: %v", err)
	}
	assertAppStatus(t, ctrl.DeleteReconcileReport(ctx, owner.ID, acc.ID, rep.ID), http.StatusNotFound)
	unlocked, err := ctrl.PatchCheckingRecord(ctx, owner.ID, acc.ID, rec1.ID, models.CheckingRecordPatchRequest{Value: &newVal})
	if err != nil || unlocked.Value != newVal || unlocked.ReconcileReportID != nil {
		t.Fatalf("expected record to be editable again after report deletion: %+v err=%v", unlocked, err)
	}
}

func TestReconcilePermissionsAndAccountTypes(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	owner := registerTestUser(t, ctrl, "recperm1")
	reader := registerTestUser(t, ctrl, "recperm2")
	sav, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{
		Name: "Savings", Type: "savings", Description: "Emergency fund",
	})
	snap, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{
		Name: "Brokerage", Type: "snapshot", Description: "Snapshot",
	})
	_, _ = ctrl.GrantAccountAccess(ctx, owner.ID, sav.ID, reader.ID, models.AccessLevelRead)

	zero := 0.0
	_, err := ctrl.CreateReconcileReport(ctx, reader.ID, sav.ID, models.ReconcileReportCreateRequest{
		ReportDate: "2026-11-16", StartingBalance: &zero, EndingBalance: &zero, RecordIDs: []string{},
	})
	assertAppStatus(t, err, http.StatusForbidden)

	assertAppStatus(t, ctrl.DeleteReconcileReport(ctx, reader.ID, sav.ID, "missing-rep"), http.StatusForbidden)

	_, err = ctrl.CreateReconcileReport(ctx, owner.ID, snap.ID, models.ReconcileReportCreateRequest{
		ReportDate: "2026-11-16", StartingBalance: &zero, EndingBalance: &zero, RecordIDs: []string{},
	})
	assertAppStatus(t, err, http.StatusBadRequest)
}
