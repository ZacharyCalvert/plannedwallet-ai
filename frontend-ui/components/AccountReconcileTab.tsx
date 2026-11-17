'use client';

import { useState, useEffect, useMemo } from 'react';
import { AccountRecord, Category, ReconcileReport } from '../lib/types';
import { DateInput, isoToDisplayDate } from './DateInput';
import {
  fetchReconcileReports,
  createReconcileReport,
  deleteReconcileReport,
  patchAccountRecord,
} from '../lib/api';
import {
  filterReconcileCandidates,
  computeReconcileBalanceSummary,
  parseReconcileAmount,
  getNextSortConfig,
  sortReconcileCandidates,
  type ReconcileSortConfig,
  type ReconcileSortKey,
} from '../lib/accountReconcile';

interface AccountReconcileTabProps {
  accountId: string;
  token: string;
  records: AccountRecord[];
  categories: Category[];
  onRecordUpdated: (updated: AccountRecord) => void;
  onReportCreated: (report: ReconcileReport) => void;
  onReportDeleted: (reportId: string) => void;
}

function getTodayIso(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

export function AccountReconcileTab({
  accountId,
  token,
  records,
  categories,
  onRecordUpdated,
  onReportCreated,
  onReportDeleted,
}: AccountReconcileTabProps) {
  const [subView, setSubView] = useState<'new' | 'history'>('new');
  const [reports, setReports] = useState<ReconcileReport[]>([]);
  const [loadingReports, setLoadingReports] = useState(true);
  const [expandedReportId, setExpandedReportId] = useState<string | null>(null);
  const [confirmingDeleteId, setConfirmingDeleteId] = useState<string | null>(null);
  const [deletingReportId, setDeletingReportId] = useState<string | null>(null);

  // New report form state
  const [reportName, setReportName] = useState('');
  const [reportDate, setReportDate] = useState(getTodayIso());
  const [startingStr, setStartingStr] = useState('0.00');
  const [endingStr, setEndingStr] = useState('');
  const [cutoffDate, setCutoffDate] = useState('');
  const [amountFilter, setAmountFilter] = useState('');
  const [sortConfig, setSortConfig] = useState<ReconcileSortConfig>({
    key: 'date_sent',
    direction: 'asc',
  });
  const [selectedIds, setSelectedIds] = useState<Record<string, boolean>>({});

  // Inline amount editing state
  const [editingAmountId, setEditingAmountId] = useState<string | null>(null);
  const [editingAmountVal, setEditingAmountVal] = useState('');
  const [savingAmount, setSavingAmount] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoadingReports(true);
    fetchReconcileReports(token, accountId)
      .then((list) => {
        if (!cancelled) {
          setReports(list);
          if (list.length > 0) {
            setStartingStr(list[0].ending_balance.toFixed(2));
          }
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoadingReports(false);
      });
    return () => {
      cancelled = true;
    };
  }, [accountId, token]);

  const categoryNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const c of categories) map.set(c.id, c.name);
    return map;
  }, [categories]);

  const candidateRecords = useMemo(() => {
    const filtered = filterReconcileCandidates(records, cutoffDate, amountFilter);
    return sortReconcileCandidates(filtered, sortConfig, categoryNameById);
  }, [records, cutoffDate, amountFilter, sortConfig, categoryNameById]);

  const handleSortColumn = (key: ReconcileSortKey) => {
    setSortConfig((prev) => getNextSortConfig(prev, key));
  };

  const renderSortHeader = (label: string, key: ReconcileSortKey, alignRight = false) => {
    const isActive = sortConfig.key === key;
    const ariaSort = isActive
      ? sortConfig.direction === 'asc'
        ? 'ascending'
        : 'descending'
      : 'none';
    return (
      <th style={alignRight ? { textAlign: 'right' } : undefined} aria-sort={ariaSort}>
        <button
          type="button"
          className={`table-sort-btn ${alignRight ? 'table-sort-btn-right' : ''} ${
            isActive ? 'table-sort-btn-active' : ''
          }`}
          onClick={() => handleSortColumn(key)}
        >
          <span>{label}</span>
          <span className="table-sort-indicator" aria-hidden="true">
            {isActive ? (sortConfig.direction === 'asc' ? '▲' : '▼') : '⇅'}
          </span>
        </button>
      </th>
    );
  };

  const selectedRecordsList = useMemo(
    () => records.filter((r) => !r.reconcile_report_id && selectedIds[r.id]),
    [records, selectedIds]
  );

  const parsedStart = parseReconcileAmount(startingStr);
  const parsedEnd = parseReconcileAmount(endingStr);
  const balanceSummary = useMemo(
    () => computeReconcileBalanceSummary(parsedStart, parsedEnd, selectedRecordsList),
    [parsedStart, parsedEnd, selectedRecordsList]
  );

  const allFilteredSelected =
    candidateRecords.length > 0 && candidateRecords.every((r) => Boolean(selectedIds[r.id]));

  const toggleSelectAllFiltered = () => {
    setSelectedIds((prev) => {
      const next = { ...prev };
      for (const r of candidateRecords) {
        next[r.id] = !allFilteredSelected;
      }
      return next;
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const handleSaveInlineAmount = async (recId: string) => {
    setError('');
    const parsed = parseReconcileAmount(editingAmountVal);
    if (isNaN(parsed) || parsed === 0) {
      setError('Please enter a valid non-zero numeric amount.');
      return;
    }
    setSavingAmount(true);
    try {
      const updated = await patchAccountRecord(token, accountId, recId, { value: parsed });
      onRecordUpdated(updated);
      setEditingAmountId(null);
      setEditingAmountVal('');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to update transaction amount.');
    } finally {
      setSavingAmount(false);
    }
  };

  const handleCreateReport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMsg('');

    if (!reportDate.trim()) {
      setError('Reconcile report date is required.');
      return;
    }
    if (isNaN(parsedStart) || isNaN(parsedEnd)) {
      setError('Starting balance and ending balance must be valid numbers.');
      return;
    }
    if (!balanceSummary.isBalanced) {
      setError('Starting balance plus selected records must equal the target ending balance.');
      return;
    }

    setSubmitting(true);
    try {
      const created = await createReconcileReport(token, accountId, {
        name: reportName.trim() || undefined,
        report_date: reportDate.trim(),
        starting_balance: parsedStart,
        ending_balance: parsedEnd,
        record_ids: selectedRecordsList.map((r) => r.id),
      });
      setReports((prev) => [created, ...prev]);
      onReportCreated(created);
      setSelectedIds({});
      setReportName('');
      setStartingStr(created.ending_balance.toFixed(2));
      setEndingStr('');
      setSuccessMsg(
        `Reconcile report${created.name ? ` "${created.name}"` : ''} completed with ${created.record_count} record(s).`
      );
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create reconcile report.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteReport = async (report: ReconcileReport) => {
    setError('');
    setSuccessMsg('');
    setDeletingReportId(report.id);
    try {
      await deleteReconcileReport(token, accountId, report.id);
      const remaining = reports.filter((r) => r.id !== report.id);
      setReports(remaining);
      setConfirmingDeleteId(null);
      if (expandedReportId === report.id) {
        setExpandedReportId(null);
      }
      setStartingStr(remaining.length > 0 ? remaining[0].ending_balance.toFixed(2) : '0.00');
      onReportDeleted(report.id);
      setSuccessMsg(
        `Deleted reconcile report${report.name ? ` "${report.name}"` : ''}. Attached records remain in your account and are now unlocked.`
      );
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete reconcile report.');
    } finally {
      setDeletingReportId(null);
    }
  };

  return (
    <div className="reconcile-tab-section">
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Account Reconciliation</h3>
          <p className="graph-subtitle">
            Match transactions against a statement starting and ending balance, or review completed reports
          </p>
        </div>
        <div className="graph-range-pills" role="group" aria-label="Reconcile view">
          <button
            type="button"
            className={`pill-btn ${subView === 'new' ? 'active' : ''}`}
            onClick={() => setSubView('new')}
          >
            New Reconcile Report
          </button>
          <button
            type="button"
            className={`pill-btn ${subView === 'history' ? 'active' : ''}`}
            onClick={() => setSubView('history')}
          >
            Completed Reports ({reports.length})
          </button>
        </div>
      </div>

      {error && <div className="alert-error">{error}</div>}
      {successMsg && <div className="notice-banner">{successMsg}</div>}

      {subView === 'new' ? (
        <form onSubmit={handleCreateReport} className="reconcile-new-section">
          <div className="reconcile-meta-grid">
            <div className="field-group">
              <label htmlFor="reconcile-name">Report Name (Optional)</label>
              <input
                id="reconcile-name"
                type="text"
                maxLength={100}
                placeholder="e.g. Jan2026, Year-End"
                value={reportName}
                onChange={(e) => setReportName(e.target.value)}
              />
            </div>
            <div className="field-group">
              <label htmlFor="reconcile-date">Report Date</label>
              <DateInput
                id="reconcile-date"
                required
                defaultToday
                value={reportDate}
                onChange={setReportDate}
              />
            </div>
            <div className="field-group">
              <label htmlFor="reconcile-start">Starting Balance</label>
              <input
                id="reconcile-start"
                type="text"
                inputMode="decimal"
                required
                placeholder="0.00"
                value={startingStr}
                onChange={(e) => setStartingStr(e.target.value)}
              />
            </div>
            <div className="field-group">
              <label htmlFor="reconcile-end">Ending Balance</label>
              <input
                id="reconcile-end"
                type="text"
                inputMode="decimal"
                required
                placeholder="0.00"
                value={endingStr}
                onChange={(e) => setEndingStr(e.target.value)}
              />
            </div>
          </div>

          <div className="reconcile-summary-bar">
            <div className="reconcile-stat">
              <span className="metric-label">Starting</span>
              <span className="font-semibold">
                {isNaN(parsedStart) ? '—' : formatCurrency(parsedStart)}
              </span>
            </div>
            <div className="reconcile-stat">
              <span className="metric-label">Selected ({selectedRecordsList.length})</span>
              <span className="font-semibold">{formatCurrency(balanceSummary.selectedSum)}</span>
            </div>
            <div className="reconcile-stat">
              <span className="metric-label">Calculated Ending</span>
              <span className="font-semibold">
                {isNaN(balanceSummary.calculatedEnding) ? '—' : formatCurrency(balanceSummary.calculatedEnding)}
              </span>
            </div>
            <div className="reconcile-stat">
              <span className="metric-label">Target Ending</span>
              <span className="font-semibold">
                {isNaN(parsedEnd) ? '—' : formatCurrency(parsedEnd)}
              </span>
            </div>
            <div className="reconcile-stat">
              <span className="metric-label">Difference</span>
              <span
                className={`font-semibold ${
                  balanceSummary.isBalanced ? 'text-positive' : 'text-negative'
                }`}
              >
                {isNaN(balanceSummary.difference)
                  ? '—'
                  : balanceSummary.isBalanced
                  ? '$0.00 (Balanced)'
                  : formatCurrency(balanceSummary.difference)}
              </span>
            </div>
          </div>

          <div className="reconcile-filter-toolbar">
            <div className="reconcile-filter-inputs">
              <div className="field-group">
                <label htmlFor="reconcile-cutoff">Sent Cutoff Date (&ge;)</label>
                <DateInput
                  id="reconcile-cutoff"
                  value={cutoffDate}
                  onChange={setCutoffDate}
                />
              </div>
              <div className="field-group">
                <label htmlFor="reconcile-amount-filter">Filter by Amount</label>
                <input
                  id="reconcile-amount-filter"
                  type="text"
                  placeholder="e.g. 30 or -30.00"
                  value={amountFilter}
                  onChange={(e) => setAmountFilter(e.target.value)}
                />
              </div>
              {(cutoffDate || amountFilter) && (
                <button
                  type="button"
                  className="btn-secondary btn-sm"
                  onClick={() => {
                    setCutoffDate('');
                    setAmountFilter('');
                  }}
                >
                  Clear Filters
                </button>
              )}
            </div>

            <div className="reconcile-actions-right">
              <label className="uncategorized-toggle">
                <input
                  type="checkbox"
                  checked={allFilteredSelected}
                  disabled={candidateRecords.length === 0}
                  onChange={toggleSelectAllFiltered}
                />
                <span>Include all filtered ({candidateRecords.length})</span>
              </label>
              <button
                type="submit"
                className="btn-primary btn-sm"
                disabled={submitting || !balanceSummary.isBalanced}
              >
                {submitting ? 'Reconciling...' : 'Complete Reconcile Report'}
              </button>
            </div>
          </div>

          <div className="records-table-container">
            <table className="records-table">
              <thead>
                <tr>
                  <th style={{ width: '40px' }}>Include</th>
                  {renderSortHeader('Sent', 'date_sent')}
                  {renderSortHeader('Cleared', 'date_cleared')}
                  {renderSortHeader('Recipient', 'entity')}
                  {renderSortHeader('Category', 'category')}
                  {renderSortHeader('Amount', 'value', true)}
                  {renderSortHeader('Description', 'description')}
                </tr>
              </thead>
              <tbody>
                {candidateRecords.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="records-empty">
                      No unreconciled records match the current filters.
                    </td>
                  </tr>
                ) : (
                  candidateRecords.map((rec) => {
                    const checked = Boolean(selectedIds[rec.id]);
                    const isEditingAmt = editingAmountId === rec.id;
                    return (
                      <tr
                        key={rec.id}
                        className={`record-row ${checked ? 'record-row-selected' : ''}`}
                      >
                        <td>
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => toggleOne(rec.id)}
                            aria-label={`Include record ${rec.entity} ${isoToDisplayDate(rec.date_sent)}`}
                          />
                        </td>
                        <td>{isoToDisplayDate(rec.date_sent)}</td>
                        <td>
                          {rec.date_cleared ? (
                            <span className="cleared-date">{isoToDisplayDate(rec.date_cleared)}</span>
                          ) : (
                            <span className="badge-pending">Pending</span>
                          )}
                        </td>
                        <td className="font-semibold">{rec.entity}</td>
                        <td>
                          {rec.category_id && categoryNameById.has(rec.category_id) ? (
                            <span className="category-pill">{categoryNameById.get(rec.category_id)}</span>
                          ) : (
                            <span className="text-muted">Uncategorized</span>
                          )}
                        </td>
                        <td style={{ textAlign: 'right' }}>
                          {isEditingAmt ? (
                            <div className="reconcile-inline-amt">
                              <input
                                type="text"
                                inputMode="decimal"
                                value={editingAmountVal}
                                onChange={(e) => setEditingAmountVal(e.target.value)}
                                disabled={savingAmount}
                                aria-label="Edit record amount"
                              />
                              <button
                                type="button"
                                className="btn-primary btn-sm"
                                onClick={() => handleSaveInlineAmount(rec.id)}
                                disabled={savingAmount}
                              >
                                Save
                              </button>
                              <button
                                type="button"
                                className="btn-secondary btn-sm"
                                onClick={() => {
                                  setEditingAmountId(null);
                                  setEditingAmountVal('');
                                }}
                                disabled={savingAmount}
                              >
                                Cancel
                              </button>
                            </div>
                          ) : (
                            <div className="reconcile-amt-cell">
                              <span className={rec.value < 0 ? 'amount-negative' : 'amount-positive'}>
                                {formatCurrency(rec.value)}
                              </span>
                              <button
                                type="button"
                                className="btn-table-action"
                                onClick={() => {
                                  setEditingAmountId(rec.id);
                                  setEditingAmountVal(rec.value.toFixed(2));
                                }}
                                title="Edit amount via PATCH"
                              >
                                Edit
                              </button>
                            </div>
                          )}
                        </td>
                        <td className="text-muted">{rec.description || '—'}</td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </form>
      ) : (
        <div className="reconcile-history-list">
          {loadingReports ? (
            <div className="records-loading">Loading reconcile reports...</div>
          ) : reports.length === 0 ? (
            <div className="graph-empty-card">
              <h4>No Completed Reconcile Reports</h4>
              <p>Switch to New Reconcile Report to create your first reconciliation.</p>
            </div>
          ) : (
            reports.map((rep) => {
              const isExpanded = expandedReportId === rep.id;
              const isConfirming = confirmingDeleteId === rep.id;
              const isDeleting = deletingReportId === rep.id;
              return (
                <div key={rep.id} className="reconcile-report-card">
                  <div className="reconcile-report-header">
                    <div>
                      <h4>{rep.name || `Reconcile ${isoToDisplayDate(rep.report_date)}`}</h4>
                      <p className="text-muted text-sm">
                        Report Date: {isoToDisplayDate(rep.report_date)} &middot; {rep.record_count}{' '}
                        {rep.record_count === 1 ? 'record' : 'records'}
                      </p>
                    </div>
                    <div className="reconcile-report-balances">
                      <span>Start: <strong>{formatCurrency(rep.starting_balance)}</strong></span>
                      <span>End: <strong>{formatCurrency(rep.ending_balance)}</strong></span>
                      <button
                        type="button"
                        className="btn-secondary btn-sm"
                        onClick={() => setExpandedReportId(isExpanded ? null : rep.id)}
                      >
                        {isExpanded ? 'Hide Records' : 'View Records'}
                      </button>
                      <button
                        type="button"
                        className="btn-danger-outline btn-sm"
                        onClick={() => setConfirmingDeleteId(isConfirming ? null : rep.id)}
                        disabled={isDeleting}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                  {isConfirming && (
                    <div className="delete-confirm-inline">
                      <span>
                        Delete this reconcile report? Existing transaction records will NOT be deleted — they will simply be unlinked and unlocked for editing.
                      </span>
                      <div className="delete-confirm-actions">
                        <button
                          type="button"
                          className="btn-secondary btn-sm"
                          onClick={() => setConfirmingDeleteId(null)}
                          disabled={isDeleting}
                        >
                          Cancel
                        </button>
                        <button
                          type="button"
                          className="btn-danger-outline btn-sm"
                          onClick={() => handleDeleteReport(rep)}
                          disabled={isDeleting}
                        >
                          {isDeleting ? 'Deleting...' : 'Confirm Delete'}
                        </button>
                      </div>
                    </div>
                  )}
                  {isExpanded && (
                    <div className="records-table-container mt-3">
                      <table className="records-table">
                        <thead>
                          <tr>
                            <th>Sent</th>
                            <th>Cleared</th>
                            <th>Recipient</th>
                            <th>Category</th>
                            <th style={{ textAlign: 'right' }}>Amount</th>
                            <th>Description</th>
                          </tr>
                        </thead>
                        <tbody>
                          {(rep.records || []).map((rec) => (
                            <tr key={rec.id}>
                              <td>{isoToDisplayDate(rec.date_sent)}</td>
                              <td>{rec.date_cleared ? isoToDisplayDate(rec.date_cleared) : '—'}</td>
                              <td className="font-semibold">{rec.entity}</td>
                              <td>
                                {rec.category_id && categoryNameById.has(rec.category_id)
                                  ? categoryNameById.get(rec.category_id)
                                  : 'Uncategorized'}
                              </td>
                              <td style={{ textAlign: 'right' }}>{formatCurrency(rec.value)}</td>
                              <td className="text-muted">{rec.description || '—'}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
