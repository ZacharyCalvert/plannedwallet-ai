'use client';

import { useState, useEffect, useMemo, useRef } from 'react';
import { Account, AccountRecord } from '../lib/types';
import { DateInput } from './DateInput';
import {
  fetchAccountRecords,
  createAccountRecord,
  updateAccountRecord,
  patchAccountRecord,
} from '../lib/api';

interface AccountDetailProps {
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

export function AccountDetail({ account, token }: AccountDetailProps) {
  const [records, setRecords] = useState<AccountRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Selected record for edit mode (null means New Record mode)
  const [selectedRecordId, setSelectedRecordId] = useState<string | null>(null);

  // Ref to the amount input for focusing after submitting or canceling
  const amountInputRef = useRef<HTMLInputElement>(null);
  const [focusCounter, setFocusCounter] = useState(0);

  const focusAmountInput = () => {
    if (amountInputRef.current) {
      amountInputRef.current.focus();
      amountInputRef.current.select();
    }
  };

  useEffect(() => {
    if (focusCounter > 0) {
      focusAmountInput();
      const raf = requestAnimationFrame(focusAmountInput);
      const timer = setTimeout(focusAmountInput, 60);
      return () => {
        cancelAnimationFrame(raf);
        clearTimeout(timer);
      };
    }
  }, [focusCounter]);

  // Record entry/edit fields: amount, sent, cleared, Recipient, description
  const [amount, setAmount] = useState('');
  const [dateSent, setDateSent] = useState(getTodayDateString());
  const [dateCleared, setDateCleared] = useState('');
  const [recipient, setRecipient] = useState('');
  const [description, setDescription] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  const resetForm = () => {
    setSelectedRecordId(null);
    setAmount('');
    setDateSent(getTodayDateString());
    setDateCleared('');
    setRecipient('');
    setDescription('');
    setFormError('');
  };

  const loadRecords = async () => {
    try {
      setLoading(true);
      setError('');
      const data = await fetchAccountRecords(token, account.id);
      setRecords(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load records');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRecords();
    resetForm();
    setFocusCounter((c) => c + 1);
  }, [account.id, token]);

  // Expected balance assumes all records apply to the total value of the account
  const expectedBalance = records.reduce((sum, r) => sum + r.value, 0);

  // Current balance includes only what has been given a cleared date
  const currentBalance = records
    .filter((r) => r.date_cleared !== null && r.date_cleared !== undefined && r.date_cleared.trim() !== '')
    .reduce((sum, r) => sum + r.value, 0);

  // Compute running balance based on prior records + current record adjustment
  const { sortedRecords, recordBalances } = useMemo(() => {
    // 1. Sort chronologically (oldest to newest) to compute cumulative running balances
    const chronological = [...records].sort((a, b) => {
      if (a.date_sent !== b.date_sent) {
        return a.date_sent.localeCompare(b.date_sent);
      }
      const aCreated = a.created_at || '';
      const bCreated = b.created_at || '';
      if (aCreated !== bCreated) {
        return aCreated.localeCompare(bCreated);
      }
      return a.id.localeCompare(b.id);
    });

    let runningBalance = 0;
    const balances = new Map<string, number>();
    for (const rec of chronological) {
      // Balance reflects last known balance based on prior records + adjustment from current record
      runningBalance += rec.value;
      balances.set(rec.id, runningBalance);
    }

    // 2. Display order: most recent first (date_sent DESC, created_at DESC)
    const display = [...records].sort((a, b) => {
      if (a.date_sent !== b.date_sent) {
        return b.date_sent.localeCompare(a.date_sent);
      }
      const aCreated = a.created_at || '';
      const bCreated = b.created_at || '';
      if (aCreated !== bCreated) {
        return bCreated.localeCompare(aCreated);
      }
      return b.id.localeCompare(a.id);
    });

    return { sortedRecords: display, recordBalances: balances };
  }, [records]);

  const amountColorClass = getAmountColorClass(amount);

  const handleSelectRecord = (rec: AccountRecord) => {
    setSelectedRecordId(rec.id);
    setAmount(rec.value.toString());
    setDateSent(rec.date_sent);
    setDateCleared(rec.date_cleared || '');
    setRecipient(rec.entity);
    setDescription(rec.description || '');
    setFormError('');
    setFocusCounter((c) => c + 1);
  };

  const handleCancelEdit = () => {
    resetForm();
    setFocusCounter((c) => c + 1);
  };

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    const parsedAmount = parseAmount(amount);
    if (isNaN(parsedAmount) || parsedAmount === 0) {
      setFormError('Please enter a valid non-zero amount (positive for income, negative for spending, e.g. 1,250.00 or -45.50).');
      setFocusCounter((c) => c + 1);
      return;
    }

    if (!recipient.trim()) {
      setFormError('Recipient is required.');
      return;
    }

    if (!dateSent.trim()) {
      setFormError('Sent date is required.');
      return;
    }

    setSubmitting(true);
    try {
      const payload = {
        value: parsedAmount,
        date_sent: dateSent.trim(),
        date_cleared: dateCleared.trim() ? dateCleared.trim() : null,
        entity: recipient.trim(),
        description: description.trim() || undefined,
      };

      if (selectedRecordId) {
        const updated = await updateAccountRecord(token, account.id, selectedRecordId, payload);
        setRecords((prev) => prev.map((r) => (r.id === selectedRecordId ? updated : r)));
        resetForm();
      } else {
        const newRecord = await createAccountRecord(token, account.id, payload);
        // Prepend the new record and recalculate balances
        setRecords((prev) => [newRecord, ...prev]);
        resetForm();
      }

      // Move cursor focus back to Amount input box for the next new record
      setFocusCounter((c) => c + 1);
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : 'Failed to save record');
    } finally {
      setSubmitting(false);
    }
  };

  const handleMarkCleared = async (recordId: string) => {
    try {
      const today = getTodayDateString();
      const updated = await patchAccountRecord(token, account.id, recordId, {
        date_cleared: today,
      });
      setRecords((prev) => prev.map((r) => (r.id === recordId ? updated : r)));
      if (selectedRecordId === recordId) {
        setDateCleared(today);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to mark record as cleared');
    }
  };

  return (
    <div className="card account-detail-card">
      {/* Top Header: Account Info on Left, Balances on Rightmost Side */}
      <div className="account-detail-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>{account.name}</h2>
            <span className="badge-checking">{account.type}</span>
          </div>
          <p className="account-desc-text">{account.description}</p>
          <p className="account-id-label">Account ID: {account.id}</p>
        </div>

        {/* Rightmost side balances */}
        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">Current balance</span>
            <span
              className={`balance-value ${
                currentBalance >= 0 ? 'balance-positive' : 'balance-negative'
              }`}
            >
              {formatCurrency(currentBalance)}
            </span>
            <span className="balance-subtext">Cleared records only</span>
          </div>
          <div className="balance-box expected-balance-box">
            <span className="balance-label">Expected balance</span>
            <span
              className={`balance-value ${
                expectedBalance >= 0 ? 'balance-positive' : 'balance-negative'
              }`}
            >
              {formatCurrency(expectedBalance)}
            </span>
            <span className="balance-subtext">All records included</span>
          </div>
        </div>
      </div>

      {error && <div className="alert-error mb-4">{error}</div>}

      {/* New / Edit Checking Record Form Row at the top */}
      <section className="new-record-section">
        <div className="section-title-row">
          <h3>{selectedRecordId ? 'Edit Record' : 'New Record'}</h3>
          <span className="section-subtitle">
            {selectedRecordId
              ? 'Update record details or set cleared date'
              : 'Add income (+) or spending (-) to this checking account'}
          </span>
        </div>

        {formError && <div className="alert-error mb-3">{formError}</div>}

        <form onSubmit={handleFormSubmit} className="new-record-form">
          <div className="new-record-inputs">
            {/* 1. Amount: positive/negative number input with dynamic green/red/grey text color */}
            <div className="field-group field-amount">
              <label htmlFor="record-amount">Amount</label>
              <div className={`amount-input-wrapper ${amountColorClass}`}>
                <span className={`currency-prefix ${amountColorClass}`}>$</span>
                <input
                  ref={amountInputRef}
                  id="record-amount"
                  type="text"
                  inputMode="decimal"
                  required
                  placeholder="0.00"
                  value={amount}
                  onChange={(e) => {
                    const val = e.target.value;
                    if (/^[0-9.,-]*$/.test(val)) {
                      setAmount(val);
                    }
                  }}
                  className={`amount-input ${amountColorClass}`}
                />
              </div>
            </div>

            {/* 2. Sent Date */}
            <div className="field-group field-sent">
              <label htmlFor="record-sent">Sent</label>
              <DateInput
                id="record-sent"
                required
                defaultToday
                value={dateSent}
                onChange={setDateSent}
                nextFieldId="record-cleared"
              />
            </div>

            {/* 3. Cleared Date */}
            <div className="field-group field-cleared">
              <label htmlFor="record-cleared">Cleared</label>
              <DateInput
                id="record-cleared"
                value={dateCleared}
                onChange={setDateCleared}
                nextFieldId="record-recipient"
              />
            </div>

            {/* 4. Recipient */}
            <div className="field-group field-recipient">
              <label htmlFor="record-recipient">Recipient</label>
              <input
                id="record-recipient"
                type="text"
                required
                placeholder="e.g. Grocery Store, Employer"
                value={recipient}
                onChange={(e) => setRecipient(e.target.value)}
              />
            </div>

            {/* 5. Description */}
            <div className="field-group field-description">
              <label htmlFor="record-description">Description</label>
              <input
                id="record-description"
                type="text"
                placeholder="Optional memo or notes"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
          </div>

          {/* Action buttons: Apply Change + Cancel when editing, Add Record when new */}
          <div className="new-record-actions">
            <button
              type="submit"
              className="btn-primary btn-add-record"
              disabled={submitting}
              onMouseDown={(e) => e.preventDefault()}
            >
              {submitting
                ? selectedRecordId
                  ? 'Saving...'
                  : 'Adding...'
                : selectedRecordId
                ? 'Apply Change'
                : 'Add Record'}
            </button>
            {selectedRecordId && (
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

      {/* Existing Records Presentation below */}
      <section className="existing-records-section">
        <div className="section-title-row">
          <h3>Account Records</h3>
          <span className="records-count-badge">{records.length} record{records.length === 1 ? '' : 's'}</span>
        </div>

        {loading ? (
          <div className="records-loading">Loading records...</div>
        ) : (
          <div className="records-table-container">
            <table className="records-table">
              <thead>
                <tr>
                  <th className="th-balance">Balance</th>
                  <th className="th-amount">Amount</th>
                  <th className="th-sent">Sent</th>
                  <th className="th-cleared">Cleared</th>
                  <th className="th-recipient">Recipient</th>
                  <th className="th-description">Description</th>
                  <th className="th-action">Status</th>
                </tr>
              </thead>
              <tbody>
                {sortedRecords.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="records-empty">
                      No records found for this account. Use the row above to add income or spending.
                    </td>
                  </tr>
                ) : (
                  sortedRecords.map((rec) => {
                    const isPositive = rec.value > 0;
                    const isNegative = rec.value < 0;
                    const isSelected = selectedRecordId === rec.id;
                    const recBalance = recordBalances.get(rec.id) ?? rec.value;
                    const isBalancePositive = recBalance > 0;
                    const isBalanceNegative = recBalance < 0;
                    return (
                      <tr
                        key={rec.id}
                        className={`record-row ${isSelected ? 'record-row-selected' : ''}`}
                        onClick={() => handleSelectRecord(rec)}
                      >
                        <td className="td-balance">
                          <span
                            className={`balance-cell ${
                              isBalancePositive
                                ? 'balance-positive'
                                : isBalanceNegative
                                ? 'balance-negative'
                                : 'balance-zero'
                            }`}
                          >
                            {formatCurrency(recBalance)}
                          </span>
                        </td>
                        <td className="td-amount">
                          <span
                            className={`amount-badge ${
                              isPositive
                                ? 'amount-positive'
                                : isNegative
                                ? 'amount-negative'
                                : 'amount-zero'
                            }`}
                          >
                            {isPositive ? `+${formatCurrency(rec.value)}` : formatCurrency(rec.value)}
                          </span>
                        </td>
                        <td className="td-sent">{rec.date_sent}</td>
                        <td className="td-cleared">
                          {rec.date_cleared ? (
                            <span className="cleared-date">{rec.date_cleared}</span>
                          ) : (
                            <span className="badge-pending">Pending</span>
                          )}
                        </td>
                        <td className="td-recipient">{rec.entity}</td>
                        <td className="td-description">{rec.description || '—'}</td>
                        <td className="td-action">
                          {!rec.date_cleared ? (
                            <button
                              type="button"
                              className="btn-clear-record"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleMarkCleared(rec.id);
                              }}
                              title="Mark as cleared today"
                            >
                              Mark Cleared
                            </button>
                          ) : (
                            <span className="badge-cleared-check">Cleared</span>
                          )}
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
  );
}
