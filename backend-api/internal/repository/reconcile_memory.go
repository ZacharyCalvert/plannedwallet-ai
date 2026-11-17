package repository

import (
	"context"
	"sort"
	"time"

	"github.com/plannedwallet/internal/models"
)

func (m *MemoryRepository) CreateReconcileReport(_ context.Context, report *models.ReconcileReport, recordIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	targets := make([]*models.CheckingRecord, 0, len(recordIDs))
	for _, recID := range recordIDs {
		rec, ok := m.checkingRecords[recID]
		if !ok || rec.AccountID != report.AccountID {
			return ErrNotFound
		}
		if rec.ReconcileReportID != nil && *rec.ReconcileReportID != "" {
			return ErrConflict
		}
		targets = append(targets, rec)
	}

	copiedReport := *report
	m.reconcileReports[report.ID] = &copiedReport
	for _, rec := range targets {
		repID := report.ID
		rec.ReconcileReportID = &repID
		if rec.DateCleared == nil || *rec.DateCleared == "" {
			cleared := report.ReportDate
			rec.DateCleared = &cleared
		}
		rec.UpdatedAt = report.UpdatedAt
	}
	return nil
}

func (m *MemoryRepository) groupedReconciledRecordsLocked(accountID string) map[string][]*models.CheckingRecord {
	grouped := make(map[string][]*models.CheckingRecord)
	for _, rec := range m.checkingRecords {
		if rec.AccountID != accountID || rec.ReconcileReportID == nil {
			continue
		}
		copied := *rec
		grouped[*rec.ReconcileReportID] = append(grouped[*rec.ReconcileReportID], &copied)
	}
	for repID := range grouped {
		recs := grouped[repID]
		sort.Slice(recs, func(i, j int) bool {
			if recs[i].DateSent == recs[j].DateSent {
				return recs[i].CreatedAt.After(recs[j].CreatedAt)
			}
			return recs[i].DateSent > recs[j].DateSent
		})
	}
	return grouped
}

func (m *MemoryRepository) GetReconcileReportByID(_ context.Context, accountID, reportID string) (*models.ReconcileReport, []*models.CheckingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rep, ok := m.reconcileReports[reportID]
	if !ok || rep.AccountID != accountID {
		return nil, nil, ErrNotFound
	}
	copied := *rep
	grouped := m.groupedReconciledRecordsLocked(accountID)
	return &copied, grouped[reportID], nil
}

func (m *MemoryRepository) ListReconcileReports(_ context.Context, accountID string) ([]*models.ReconcileReport, map[string][]*models.CheckingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	reports := make([]*models.ReconcileReport, 0)
	for _, rep := range m.reconcileReports {
		if rep.AccountID == accountID {
			copied := *rep
			reports = append(reports, &copied)
		}
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].ReportDate == reports[j].ReportDate {
			return reports[i].CreatedAt.After(reports[j].CreatedAt)
		}
		return reports[i].ReportDate > reports[j].ReportDate
	})
	return reports, m.groupedReconciledRecordsLocked(accountID), nil
}

func (m *MemoryRepository) DeleteReconcileReport(_ context.Context, accountID, reportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.reconcileReports[reportID]
	if !ok || rep.AccountID != accountID {
		return ErrNotFound
	}
	delete(m.reconcileReports, reportID)
	now := time.Now().UTC()
	for _, rec := range m.checkingRecords {
		if rec.AccountID == accountID && rec.ReconcileReportID != nil && *rec.ReconcileReportID == reportID {
			rec.ReconcileReportID = nil
			rec.UpdatedAt = now
		}
	}
	return nil
}
