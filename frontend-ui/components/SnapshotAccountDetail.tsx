'use client';

import { useState, useEffect, useMemo, useRef } from 'react';
import { Account, AccountSnapshot } from '../lib/types';
import { DateInput, displayDateToIso } from './DateInput';
import { SnapshotTrendGraph } from './SnapshotTrendGraph';
import {
  fetchAccountSnapshots,
  createAccountSnapshot,
  updateAccountSnapshot,
  deleteAccountSnapshot,
} from '../lib/api';

interface SnapshotAccountDetailProps {
  account: Account;
  token: string;
}

function getTodayDateString(): string {
  const d = new Date();
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
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

function parseAmount(amtStr: string): number {
  const clean = amtStr.replace(/,/g, '').trim();
  if (!/^-?\d*(\.\d+)?$/.test(clean) || clean === '' || clean === '-') {
    return NaN;
  }
  return parseFloat(clean);
}

function getAmountColorClass(amtStr: string): string {
  const clean = amtStr.replace(/,/g, '').trim();
  if (!clean || clean === '-' || clean === '+') {
    return 'amount-text-zero';
  }
  const num = parseFloat(clean);
  if (isNaN(num) || num === 0) {
    return 'amount-text-zero';
  }
  return num > 0 ? 'amount-text-positive' : 'amount-text-negative';
}

export function SnapshotAccountDetail({ account, token }: SnapshotAccountDetailProps) {
  const [snapshots, setSnapshots] = useState<AccountSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [formError, setFormError] = useState('');
  const [activeTab, setActiveTab] = useState<'snapshots' | 'graph'>('snapshots');

  // Add / Edit Form State
  const [selectedSnapshotId, setSelectedSnapshotId] = useState<string | null>(null);
  const [valueStr, setValueStr] = useState('');
  const [snapshotDate, setSnapshotDate] = useState(getTodayDateString());
  const [submitting, setSubmitting] = useState(false);

  const valueInputRef = useRef<HTMLInputElement>(null);
  const [focusCounter, setFocusCounter] = useState(0);

  const focusValueInput = () => {
    if (valueInputRef.current) {
      valueInputRef.current.focus();
      valueInputRef.current.select();
    }
  };

  useEffect(() => {
    if (focusCounter > 0) {
      focusValueInput();
      const raf = requestAnimationFrame(focusValueInput);
      const timer = setTimeout(focusValueInput, 50);
      return () => {
        cancelAnimationFrame(raf);
        clearTimeout(timer);
      };
    }
  }, [focusCounter]);

  // Focus value input when tab switches to snapshots
  useEffect(() => {
    if (activeTab === 'snapshots') {
      setFocusCounter((c) => c + 1);
    }
  }, [activeTab]);

  // Load snapshots for this account
  const loadSnapshots = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await fetchAccountSnapshots(token, account.id);
      setSnapshots(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load account snapshots');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSnapshots();
    handleCancelEdit();
    setFocusCounter((c) => c + 1);
  }, [account.id, token]);

  // Chronologically sorted snapshots (oldest -> newest)
  const sortedSnapshots = useMemo(() => {
    return [...snapshots].sort((a, b) => {
      const cmp = a.snapshot_date.localeCompare(b.snapshot_date);
      if (cmp !== 0) return cmp;
      return (a.created_at || '').localeCompare(b.created_at || '');
    });
  }, [snapshots]);

  // Latest snapshot is the last element in chronologically sorted list
  const latestSnapshot = useMemo(() => {
    return sortedSnapshots.length > 0 ? sortedSnapshots[sortedSnapshots.length - 1] : null;
  }, [sortedSnapshots]);

  // Reverse chronologically sorted snapshots for table display (newest at top)
  const displaySnapshots = useMemo(() => {
    return [...sortedSnapshots].reverse();
  }, [sortedSnapshots]);

  // Map of snapshot ID to difference from previous chronological valuation
  const diffMap = useMemo(() => {
    const map = new Map<string, number | null>();
    for (let i = 0; i < sortedSnapshots.length; i++) {
      if (i === 0) {
        map.set(sortedSnapshots[i].id, null);
      } else {
        const diff = sortedSnapshots[i].value - sortedSnapshots[i - 1].value;
        map.set(sortedSnapshots[i].id, diff);
      }
    }
    return map;
  }, [sortedSnapshots]);

  const handleSelectForEdit = (snap: AccountSnapshot) => {
    setSelectedSnapshotId(snap.id);
    setValueStr(snap.value.toString());
    setSnapshotDate(snap.snapshot_date);
    setFormError('');
    setFocusCounter((c) => c + 1);
  };

  const handleCancelEdit = () => {
    setSelectedSnapshotId(null);
    setValueStr('');
    setSnapshotDate(getTodayDateString());
    setFormError('');
    setFocusCounter((c) => c + 1);
  };

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    const num = parseAmount(valueStr);
    if (isNaN(num)) {
      setFormError('Please enter a valid numeric valuation.');
      setFocusCounter((c) => c + 1);
      return;
    }

    let dateToUse = snapshotDate;
    if (dateToUse && !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      const iso = displayDateToIso(dateToUse);
      if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
        dateToUse = iso;
      }
    }

    if (!dateToUse || !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      setFormError('Please select a valid date (YYYY-MM-DD).');
      setFocusCounter((c) => c + 1);
      return;
    }

    setSubmitting(true);
    try {
      if (selectedSnapshotId) {
        const updated = await updateAccountSnapshot(token, account.id, selectedSnapshotId, {
          value: num,
          snapshot_date: dateToUse,
        });
        setSnapshots((prev) =>
          prev.map((s) => (s.id === selectedSnapshotId ? updated : s))
        );
        handleCancelEdit();
      } else {
        const created = await createAccountSnapshot(token, account.id, {
          value: num,
          snapshot_date: dateToUse,
        });
        setSnapshots((prev) => [...prev, created]);
        handleCancelEdit();
      }
      setFocusCounter((c) => c + 1);
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : 'Failed to save snapshot valuation');
      setFocusCounter((c) => c + 1);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (e: React.MouseEvent, snapshotId: string) => {
    e.stopPropagation();
    if (!confirm('Are you sure you want to delete this snapshot valuation?')) {
      return;
    }

    try {
      await deleteAccountSnapshot(token, account.id, snapshotId);
      setSnapshots((prev) => prev.filter((s) => s.id !== snapshotId));
      if (selectedSnapshotId === snapshotId) {
        handleCancelEdit();
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete snapshot');
    }
  };

  const amountColorClass = getAmountColorClass(valueStr);

  return (
    <div className="card account-detail-card">
      {/* Top Header: Account Info on Left, Balances on Rightmost Side */}
      <div className="account-detail-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>{account.name}</h2>
            <span className="badge-account-type badge-snapshot">
              {account.type}
            </span>
          </div>
          <p className="account-desc-text">{account.description}</p>
          <p className="account-id-label">Account ID: {account.id}</p>
        </div>

        {/* Rightmost Balances */}
        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">Latest Valuation</span>
            <span
              className={`balance-value ${
                latestSnapshot && latestSnapshot.value >= 0
                  ? 'balance-positive'
                  : 'balance-negative'
              }`}
            >
              {latestSnapshot ? formatCurrency(latestSnapshot.value) : '$0.00'}
            </span>
            <span className="balance-subtext">
              {latestSnapshot ? `As of ${latestSnapshot.snapshot_date}` : 'No valuations recorded'}
            </span>
          </div>

          <div className="balance-box expected-balance-box">
            <span className="balance-label">Total Snapshots</span>
            <span className="balance-value balance-positive">
              {snapshots.length}
            </span>
            <span className="balance-subtext">Point-in-time entries</span>
          </div>
        </div>
      </div>

      {error && <div className="alert-error mb-4">{error}</div>}

      {/* Central Pane Tab Bar */}
      <div className="snapshot-tabs-bar">
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'snapshots' ? 'active' : ''}`}
          onClick={() => setActiveTab('snapshots')}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <line x1="8" y1="6" x2="21" y2="6" />
            <line x1="8" y1="12" x2="21" y2="12" />
            <line x1="8" y1="18" x2="21" y2="18" />
            <line x1="3" y1="6" x2="3.01" y2="6" />
            <line x1="3" y1="12" x2="3.01" y2="12" />
            <line x1="3" y1="18" x2="3.01" y2="18" />
          </svg>
          Snapshots
          <span className="tab-count-pill">{snapshots.length}</span>
        </button>

        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'graph' ? 'active' : ''}`}
          onClick={() => setActiveTab('graph')}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
          </svg>
          Trend Graph
        </button>
      </div>

      {/* TAB 1: Add and Edit Snapshots View */}
      {activeTab === 'snapshots' && (
        <div className="tab-pane-snapshots">
          {/* New / Edit Snapshot Form */}
          <section className="new-record-section">
            <div className="section-title-row">
              <h3>{selectedSnapshotId ? 'Edit Snapshot' : 'New Snapshot'}</h3>
              <span className="section-subtitle">
                {selectedSnapshotId
                  ? 'Update valuation amount or effective date'
                  : 'Assign a manual point-in-time valuation for this account'}
              </span>
            </div>

            {formError && <div className="alert-error mb-3">{formError}</div>}

            <form onSubmit={handleFormSubmit} className="new-record-form">
              <div className="new-record-inputs">
                {/* 1. Valuation Value */}
                <div className="field-group field-amount">
                  <label htmlFor="snapshot-value">Account Valuation</label>
                  <div className={`amount-input-wrapper ${amountColorClass}`}>
                    <span className={`currency-prefix ${amountColorClass}`}>$</span>
                    <input
                      ref={valueInputRef}
                      id="snapshot-value"
                      type="text"
                      inputMode="decimal"
                      required
                      placeholder="e.g. 1,200.00"
                      value={valueStr}
                      onChange={(e) => {
                        const val = e.target.value;
                        if (/^[0-9.,-]*$/.test(val)) {
                          setValueStr(val);
                        }
                      }}
                      className={`amount-input ${amountColorClass}`}
                    />
                  </div>
                </div>

                {/* 2. Snapshot Date */}
                <div className="field-group field-sent">
                  <label htmlFor="snapshot-date">Valuation Date</label>
                  <DateInput
                    id="snapshot-date"
                    required
                    defaultToday
                    value={snapshotDate}
                    onChange={setSnapshotDate}
                  />
                </div>
              </div>

              <div className="new-record-actions">
                <button
                  type="submit"
                  className="btn-primary btn-add-record"
                  disabled={submitting}
                  onMouseDown={(e) => e.preventDefault()}
                >
                  {submitting
                    ? selectedSnapshotId
                      ? 'Saving...'
                      : 'Adding...'
                    : selectedSnapshotId
                    ? 'Apply Change'
                    : 'Record Snapshot'}
                </button>
                {selectedSnapshotId && (
                  <button
                    type="button"
                    className="btn-secondary btn-cancel-edit"
                    onClick={handleCancelEdit}
                    onMouseDown={(e) => e.preventDefault()}
                    disabled={submitting}
                  >
                    Cancel
                  </button>
                )}
              </div>
            </form>
          </section>

          {/* Existing Snapshots Table */}
          <section className="existing-records-section">
            <div className="section-title-row">
              <h3>Snapshot History</h3>
              <span className="records-count-badge">
                {snapshots.length} snapshot{snapshots.length === 1 ? '' : 's'}
              </span>
            </div>

            {loading ? (
              <div className="records-loading">Loading snapshots...</div>
            ) : (
              <div className="records-table-container">
                <table className="records-table">
                  <thead>
                    <tr>
                      <th className="th-sent">Date</th>
                      <th className="th-amount">Valuation</th>
                      <th className="th-balance">Change from Prior</th>
                      <th className="th-description">Recorded At</th>
                      <th className="th-action">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {displaySnapshots.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="records-empty">
                          No snapshots recorded yet. Use the form above to record your first valuation.
                        </td>
                      </tr>
                    ) : (
                      displaySnapshots.map((snap) => {
                        const isSelected = selectedSnapshotId === snap.id;
                        const diff = diffMap.get(snap.id);
                        const isPositiveDiff = diff !== null && diff !== undefined && diff > 0;
                        const isNegativeDiff = diff !== null && diff !== undefined && diff < 0;

                        return (
                          <tr
                            key={snap.id}
                            className={`record-row ${isSelected ? 'record-row-selected' : ''}`}
                            onClick={() => handleSelectForEdit(snap)}
                          >
                            <td className="td-sent">
                              <span className="snapshot-date-badge">
                                {snap.snapshot_date}
                              </span>
                            </td>
                            <td className="td-amount">
                              <span
                                className={`amount-badge ${
                                  snap.value >= 0 ? 'amount-positive' : 'amount-negative'
                                }`}
                              >
                                {formatCurrency(snap.value)}
                              </span>
                            </td>
                            <td className="td-balance">
                              {diff === null || diff === undefined ? (
                                <span className="balance-zero">— (First entry)</span>
                              ) : (
                                <span
                                  className={`balance-cell ${
                                    isPositiveDiff
                                      ? 'balance-positive'
                                      : isNegativeDiff
                                      ? 'balance-negative'
                                      : 'balance-zero'
                                  }`}
                                >
                                  {isPositiveDiff ? '+' : ''}
                                  {formatCurrency(diff)}
                                </span>
                              )}
                            </td>
                            <td className="td-description">
                              <span className="snapshot-timestamp">
                                {new Date(snap.created_at).toLocaleString('en-US', {
                                  month: 'short',
                                  day: 'numeric',
                                  year: 'numeric',
                                  hour: '2-digit',
                                  minute: '2-digit',
                                })}
                              </span>
                            </td>
                            <td className="td-action">
                              <div className="snapshot-action-buttons">
                                <button
                                  type="button"
                                  className="btn-edit-snapshot"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleSelectForEdit(snap);
                                  }}
                                  title="Edit valuation"
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-delete-snapshot"
                                  onClick={(e) => handleDelete(e, snap.id)}
                                  title="Delete snapshot"
                                >
                                  Delete
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>
      )}

      {/* TAB 2: Trend Graph View */}
      {activeTab === 'graph' && (
        <div className="tab-pane-graph">
          <SnapshotTrendGraph snapshots={snapshots} />
        </div>
      )}
    </div>
  );
}
