'use client';

import { useState, useEffect, useMemo, useRef } from 'react';
import { Account, AccountRecord, Category } from '../lib/types';
import { DateInput, isoToDisplayDate } from './DateInput';
import CsvUploadModal from './CsvUploadModal';
import { DeleteAccountModal } from './DeleteAccountModal';
import { AccountHeaderInfo } from './AccountHeaderInfo';
import { AccountBalanceTrendGraph } from './AccountBalanceTrendGraph';
import { AccountCategoryPieChart } from './AccountCategoryPieChart';
import { AccountCategoriesTab } from './AccountCategoriesTab';
import { AccountReconcileTab } from './AccountReconcileTab';
import { RecordSearchModal } from './RecordSearchModal';
import { findLastCategoryForRecipient } from '../lib/accountCategories';
import { isSearchHotkeyEvent } from '../lib/recordSearch';
import { getAmountColorClass } from '../lib/bills';
import {
  fetchAccountRecords,
  createAccountRecord,
  updateAccountRecord,
  patchAccountRecord,
  deleteAccountRecord,
  fetchCategories,
} from '../lib/api';

interface AccountDetailProps {
  account: Account;
  token: string;
  urlTab?: string;
  onTabChange?: (tab: string, replace?: boolean) => void;
  onAccountUpdated?: (updated: Account) => void;
  onAccountDeleted?: (deletedId: string) => void;
  refreshVersion?: number;
}

type AccountTab = 'records' | 'trend' | 'expenses' | 'categories' | 'reconcile';

function resolveAccountTab(tab?: string): AccountTab {
  if (tab === 'trend' || tab === 'expenses' || tab === 'categories' || tab === 'reconcile') return tab;
  return 'records';
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

function isRecordCleared(r: AccountRecord): boolean {
  return Boolean(r.date_cleared && r.date_cleared.trim() !== '');
}

function compareRecordsForDisplay(a: AccountRecord, b: AccountRecord): number {
  const aCleared = isRecordCleared(a);
  const bCleared = isRecordCleared(b);

  // 1. Not cleared -> top priority, Cleared -> second priority
  if (!aCleared && bCleared) {
    return -1;
  }
  if (aCleared && !bCleared) {
    return 1;
  }

  // 2. Both are Not Cleared:
  // Sort by date sent with oldest records staying at the top.
  // For records with the same sent date, sort by largest number at the top.
  if (!aCleared && !bCleared) {
    if (a.date_sent !== b.date_sent) {
      return a.date_sent.localeCompare(b.date_sent);
    }
    if (a.value !== b.value) {
      return b.value - a.value;
    }
    const aCreated = a.created_at || '';
    const bCreated = b.created_at || '';
    if (aCreated !== bCreated) {
      return aCreated.localeCompare(bCreated);
    }
    return a.id.localeCompare(b.id);
  }

  // 3. Both are Cleared:
  // Sort by date cleared with latest clear date at the top.
  // For records with the same clear date, sort by the largest number being at the top.
  const aDateCleared = a.date_cleared?.trim() || '';
  const bDateCleared = b.date_cleared?.trim() || '';
  if (aDateCleared !== bDateCleared) {
    return bDateCleared.localeCompare(aDateCleared);
  }
  if (a.value !== b.value) {
    return b.value - a.value;
  }
  const aCreated = a.created_at || '';
  const bCreated = b.created_at || '';
  if (aCreated !== bCreated) {
    return bCreated.localeCompare(aCreated);
  }
  return a.id.localeCompare(b.id);
}

export function AccountDetail({
  account,
  token,
  urlTab,
  onTabChange,
  onAccountUpdated,
  onAccountDeleted,
  refreshVersion = 0,
}: AccountDetailProps) {
  const [records, setRecords] = useState<AccountRecord[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<AccountTab>(() => resolveAccountTab(urlTab));
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const normalizedType = (account.type || '').toLowerCase();
  const isCreditCard = normalizedType === 'credit card' || normalizedType === 'credit_card';

  useEffect(() => {
    const resolved = resolveAccountTab(urlTab);
    setActiveTab(resolved);
    if (!urlTab) {
      onTabChange?.(resolved, true);
    }
  }, [urlTab]);

  const selectTab = (nextTab: AccountTab) => {
    setActiveTab(nextTab);
    onTabChange?.(nextTab);
    if (nextTab === 'records') {
      setFocusCounter((c) => c + 1);
    }
  };

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

  // Record entry/edit fields: amount, sent, cleared, Recipient, category, description
  const [amount, setAmount] = useState('');
  const [dateSent, setDateSent] = useState(getTodayDateString());
  const [dateCleared, setDateCleared] = useState('');
  const [recipient, setRecipient] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [categoryManuallySelected, setCategoryManuallySelected] = useState(false);
  const [description, setDescription] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');
  const [csvModalOpen, setCsvModalOpen] = useState(false);
  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const [deleteTxModalOpen, setDeleteTxModalOpen] = useState(false);
  const [deletingTx, setDeletingTx] = useState(false);
  const [deleteTxError, setDeleteTxError] = useState('');

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (isSearchHotkeyEvent(e, target?.tagName, target?.isContentEditable)) {
        e.preventDefault();
        setSearchModalOpen(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const resetForm = () => {
    setSelectedRecordId(null);
    setAmount('');
    setDateSent(getTodayDateString());
    setDateCleared('');
    setRecipient('');
    setCategoryId('');
    setCategoryManuallySelected(false);
    setDescription('');
    setFormError('');
    setDeleteTxModalOpen(false);
    setDeleteTxError('');
  };

  const loadCategories = async () => {
    try {
      const list = await fetchCategories(token);
      setCategories(list);
    } catch {
      // Categories failure handled gracefully
    }
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
    loadCategories();
    resetForm();
    setFocusCounter((c) => c + 1);
  }, [account.id]);

  useEffect(() => {
    if (refreshVersion > 0) {
      loadRecords();
      loadCategories();
    }
  }, [refreshVersion]);

  const categoryNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const c of categories) {
      map.set(c.id, c.name);
    }
    return map;
  }, [categories]);

  // Expected balance assumes all records apply to the total value of the account
  const expectedBalance = records.reduce((sum, r) => sum + r.value, 0);

  // Cleared balance includes only what has been given a cleared date
  const currentBalance = records
    .filter((r) => r.date_cleared !== null && r.date_cleared !== undefined && r.date_cleared.trim() !== '')
    .reduce((sum, r) => sum + r.value, 0);

  // Compute display records and running balance matching the record ordering
  const { sortedRecords, recordBalances } = useMemo(() => {
    // 1. Display order:
    // Not cleared -> top priority (oldest date_sent first, largest value first on ties)
    // Cleared -> second priority (latest date_cleared first, largest value first on ties)
    const display = [...records].sort(compareRecordsForDisplay);

    // 2. Balance calculations match the ordering of the records calculated bottom-up:
    // The lowest record at the bottom reflects its own value,
    // and balances accumulate upwards so the top record reflects all values added together.
    let runningBalance = 0;
    const balances = new Map<string, number>();
    for (let i = display.length - 1; i >= 0; i--) {
      const rec = display[i];
      runningBalance += rec.value;
      balances.set(rec.id, runningBalance);
    }

    return { sortedRecords: display, recordBalances: balances };
  }, [records]);

  const amountColorClass = getAmountColorClass(amount);

  const handleSelectRecord = (rec: AccountRecord) => {
    setSelectedRecordId(rec.id);
    setAmount(rec.value.toString());
    setDateSent(rec.date_sent);
    setDateCleared(rec.date_cleared || '');
    setRecipient(rec.entity);
    setCategoryId(rec.category_id || '');
    setCategoryManuallySelected(true);
    setDescription(rec.description || '');
    setFormError('');
    setFocusCounter((c) => c + 1);
  };

  const handleRecipientChange = (nextRecipient: string) => {
    setRecipient(nextRecipient);
    if (!selectedRecordId && !categoryManuallySelected) {
      const lastCatId = findLastCategoryForRecipient(records, nextRecipient);
      if (lastCatId && categoryNameById.has(lastCatId)) {
        setCategoryId(lastCatId);
      } else {
        setCategoryId('');
      }
    }
  };

  const handleCancelEdit = () => {
    resetForm();
    setFocusCounter((c) => c + 1);
  };

  const handleOpenRemoveTxModal = () => {
    if (!selectedRecordId || isEditingReconciled) return;
    setDeleteTxError('');
    setDeleteTxModalOpen(true);
  };

  const handleConfirmRemoveTransaction = async () => {
    if (!selectedRecordId || isEditingReconciled) return;
    setDeleteTxError('');
    setDeletingTx(true);
    try {
      await deleteAccountRecord(token, account.id, selectedRecordId);
      setRecords((prev) => prev.filter((r) => r.id !== selectedRecordId));
      resetForm();
      setFocusCounter((c) => c + 1);
    } catch (err: unknown) {
      setDeleteTxError(err instanceof Error ? err.message : 'Failed to remove transaction');
    } finally {
      setDeletingTx(false);
    }
  };

  const selectedRecord = useMemo(
    () => (selectedRecordId ? records.find((r) => r.id === selectedRecordId) ?? null : null),
    [records, selectedRecordId]
  );
  const isEditingReconciled = Boolean(selectedRecord?.reconcile_report_id);

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    if (selectedRecordId && isEditingReconciled) {
      setSubmitting(true);
      try {
        const updated = await patchAccountRecord(token, account.id, selectedRecordId, {
          category_id: categoryId ? categoryId : null,
          description: description.trim(),
        });
        setRecords((prev) => prev.map((r) => (r.id === selectedRecordId ? updated : r)));
        resetForm();
        setFocusCounter((c) => c + 1);
      } catch (err: unknown) {
        setFormError(err instanceof Error ? err.message : 'Failed to update reconciled record');
      } finally {
        setSubmitting(false);
      }
      return;
    }

    const parsedAmount = parseAmount(amount);
    if (isNaN(parsedAmount) || parsedAmount === 0) {
      setFormError(
        isCreditCard
          ? 'Please enter a valid non-zero amount (positive for payment, negative for purchase, e.g. 1,250.00 or -45.50).'
          : 'Please enter a valid non-zero amount (positive for income, negative for spending, e.g. 1,250.00 or -45.50).'
      );
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
        category_id: categoryId ? categoryId : null,
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
        <AccountHeaderInfo
          account={account}
          token={token}
          onAccountUpdated={onAccountUpdated}
          onDeleteClick={() => setDeleteModalOpen(true)}
        />

        {/* Rightmost side balances */}
        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">Cleared balance</span>
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

      {/* Account View Tab Bar */}
      <div className="snapshot-tabs-bar">
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'records' ? 'active' : ''}`}
          onClick={() => selectTab('records')}
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
          Records
          <span className="tab-count-pill">{records.length}</span>
        </button>

        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'trend' ? 'active' : ''}`}
          onClick={() => selectTab('trend')}
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
          Trend
        </button>

        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'expenses' ? 'active' : ''}`}
          onClick={() => selectTab('expenses')}
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
            <path d="M21.21 15.89A10 10 0 1 1 8 2.83" />
            <path d="M22 12A10 10 0 0 0 12 2v10z" />
          </svg>
          Expense Pie
        </button>

        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'categories' ? 'active' : ''}`}
          onClick={() => selectTab('categories')}
        >
          Categories
          <span className="tab-count-pill">{categories.length}</span>
        </button>

        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'reconcile' ? 'active' : ''}`}
          onClick={() => selectTab('reconcile')}
        >
          Reconcile
        </button>

        <button
          type="button"
          className="snapshot-tab-button record-search-tab-btn"
          onClick={() => setSearchModalOpen(true)}
          title="Search and edit records (/, Cmd/Ctrl+F, or Cmd/Ctrl+K)"
        >
          Search (/)
        </button>
      </div>

      {activeTab === 'trend' ? (
        <AccountBalanceTrendGraph records={records} accountType={account.type} />
      ) : activeTab === 'expenses' ? (
        <AccountCategoryPieChart records={records} categories={categories} />
      ) : activeTab === 'categories' ? (
        <AccountCategoriesTab
          token={token}
          accountId={account.id}
          records={records}
          categories={categories}
          onCategoryCreated={(cat) => setCategories((prev) => [...prev, cat])}
          onCategoryDeleted={(deletedId, transitionId) => {
            setCategories((prev) => prev.filter((c) => c.id !== deletedId));
            setRecords((prev) =>
              prev.map((r) => (r.category_id === deletedId ? { ...r, category_id: transitionId } : r))
            );
            if (categoryId === deletedId) {
              setCategoryId(transitionId || '');
            }
          }}
          onRecordsUpdated={(updatedList) => {
            const byId = new Map(updatedList.map((r) => [r.id, r]));
            setRecords((prev) => prev.map((r) => byId.get(r.id) ?? r));
          }}
        />
      ) : activeTab === 'reconcile' ? (
        <AccountReconcileTab
          accountId={account.id}
          token={token}
          records={records}
          categories={categories}
          onRecordUpdated={(updated) =>
            setRecords((prev) => prev.map((r) => (r.id === updated.id ? updated : r)))
          }
          onReportCreated={(report) => {
            const byId = new Map(report.records.map((r) => [r.id, r]));
            setRecords((prev) => prev.map((r) => byId.get(r.id) ?? r));
          }}
          onReportDeleted={(reportId) => {
            setRecords((prev) =>
              prev.map((r) =>
                r.reconcile_report_id === reportId ? { ...r, reconcile_report_id: null } : r
              )
            );
          }}
        />
      ) : (
        <>
          {/* New / Edit Checking Record Form Row at the top */}
          <section className="new-record-section">
        <div className="section-title-row">
          <h3>{selectedRecordId ? 'Edit Record' : 'New Record'}</h3>
          <span className="section-subtitle">
            {isEditingReconciled
              ? 'This record is reconciled: only Category and Description can be updated'
              : selectedRecordId
              ? 'Update record details or set cleared date'
              : isCreditCard
              ? 'Add payment (+) or purchase (-) to this account'
              : 'Add income (+) or spending (-) to this account'}
          </span>
        </div>

        {formError && <div className="alert-error mb-3">{formError}</div>}
        {isEditingReconciled && (
          <div className="notice-banner mb-3">
            <strong>Reconciled Record:</strong> Amount, Sent Date, Cleared Date, and Recipient are locked. Description and Category may still be edited.
          </div>
        )}

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
                  disabled={isEditingReconciled}
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
                disabled={isEditingReconciled}
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
                disabled={isEditingReconciled}
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
                disabled={isEditingReconciled}
                placeholder="e.g. Grocery Store, Employer"
                value={recipient}
                onChange={(e) => handleRecipientChange(e.target.value)}
              />
            </div>
          </div>

          {/* Second Row: Category and Description */}
          <div className="new-record-inputs-secondary">
            {/* 5. Category */}
            <div className="field-group field-category">
              <label htmlFor="record-category">Category</label>
              <select
                id="record-category"
                value={categoryId}
                onChange={(e) => {
                  setCategoryId(e.target.value);
                  setCategoryManuallySelected(true);
                }}
              >
                <option value="">Uncategorized</option>
                <optgroup label="Default Categories">
                  {categories
                    .filter((c) => c.is_default)
                    .map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                </optgroup>
                {categories.some((c) => !c.is_default) && (
                  <optgroup label="Custom Categories">
                    {categories
                      .filter((c) => !c.is_default)
                      .map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                  </optgroup>
                )}
              </select>
            </div>

            {/* 6. Description */}
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

          {/* Action buttons: Apply Change + Remove Transaction (if unreconciled) + Cancel when editing, Add Record when new */}
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
            {selectedRecordId && !isEditingReconciled && (
              <button
                type="button"
                className="btn-danger-outline btn-remove-transaction"
                onClick={handleOpenRemoveTxModal}
                onMouseDown={(e) => e.preventDefault()}
                disabled={submitting || deletingTx}
              >
                Remove Transaction
              </button>
            )}
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
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <h3>Account Records</h3>
            <span className="records-count-badge">{records.length} record{records.length === 1 ? '' : 's'}</span>
          </div>
          <button
            type="button"
            className="btn-secondary btn-sm btn-import-csv"
            onClick={() => setCsvModalOpen(true)}
            title="Import checking records from CSV"
          >
            &#128196; Import CSV
          </button>
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
                  <th className="th-category">Category</th>
                  <th className="th-description">Description</th>
                  <th className="th-action">Status</th>
                </tr>
              </thead>
              <tbody>
                {sortedRecords.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="records-empty">
                      No records found for this account.{' '}
                      {isCreditCard
                        ? 'Use the row above to add a payment or purchase.'
                        : 'Use the row above to add income or spending.'}
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
                        <td className="td-category">
                          {rec.category_id && categoryNameById.has(rec.category_id) ? (
                            <span className="category-pill">{categoryNameById.get(rec.category_id)}</span>
                          ) : (
                            <span className="text-muted">Uncategorized</span>
                          )}
                        </td>
                        <td className="td-description">{rec.description || '—'}</td>
                        <td className="td-action">
                          {rec.reconcile_report_id ? (
                            <span className="badge-reconciled">Reconciled</span>
                          ) : !rec.date_cleared ? (
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
        </>
      )}

      <CsvUploadModal
        isOpen={csvModalOpen}
        onClose={() => setCsvModalOpen(false)}
        token={token}
        accountId={account.id}
        accountName={account.name}
        accountType={account.type}
        existingRecords={records}
        existingCategories={categories}
        onSuccess={() => {
          loadRecords();
          loadCategories();
        }}
      />

      <DeleteAccountModal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        token={token}
        account={account}
        onSuccess={(deletedId) => {
          setDeleteModalOpen(false);
          onAccountDeleted?.(deletedId);
        }}
      />

      {deleteTxModalOpen && selectedRecord && !isEditingReconciled && (
        <div
          className="modal-backdrop"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-transaction-modal-title"
        >
          <div className="modal-content modal-danger">
            <div className="modal-header">
              <h3 id="delete-transaction-modal-title">Remove Transaction</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setDeleteTxModalOpen(false)}
                disabled={deletingTx}
                aria-label="Close"
              >
                ×
              </button>
            </div>

            <div className="modal-body">
              <div className="warning-banner">
                <span className="warning-icon" aria-hidden="true">⚠️</span>
                <div>
                  <strong>Warning: This action cannot be undone!</strong>
                </div>
              </div>

              <p className="mt-3">
                Are you sure you want to permanently remove the transaction for{' '}
                <strong>{selectedRecord.entity}</strong> ({formatCurrency(selectedRecord.value)}) sent on{' '}
                <strong>{isoToDisplayDate(selectedRecord.date_sent)}</strong>?
              </p>

              <p className="text-muted font-sm mt-2">
                Removing this record will recalculate your account balance immediately.
              </p>

              {deleteTxError && <p className="text-danger font-sm mt-3">{deleteTxError}</p>}
            </div>

            <div className="modal-actions mt-4">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setDeleteTxModalOpen(false)}
                disabled={deletingTx}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn-danger"
                onClick={handleConfirmRemoveTransaction}
                disabled={deletingTx}
              >
                {deletingTx ? 'Removing...' : 'Yes, Remove Transaction'}
              </button>
            </div>
          </div>
        </div>
      )}
      <RecordSearchModal
        isOpen={searchModalOpen}
        onClose={() => setSearchModalOpen(false)}
        token={token}
        accountId={account.id}
        accountType={account.type}
        records={records}
        categories={categories}
        onRecordUpdated={(updated) =>
          setRecords((prev) => prev.map((r) => (r.id === updated.id ? updated : r)))
        }
      />
    </div>
  );
}
