'use client';

import { useState, useMemo } from 'react';
import {
  Account,
  Bill,
  BillCreateRequest,
  BillFrequency,
  BillSchedule,
  Category,
} from '../lib/types';
import {
  formatBillScheduleLabel,
  getAmountColorClass,
  getBillDisplayName,
  isValidBillIntervalInput,
  normalizeNegativeBillAmountInput,
  normalizePositiveBillIntervalInput,
  parseBillAmount,
  sortBillsForDisplay,
} from '../lib/bills';
import { DateInput, isoToDisplayDate } from './DateInput';
import { CopyLinkButton } from './CopyLinkButton';

type ScheduleMode = 'none' | 'weekday' | 'day_of_period' | 'nth_weekday' | 'month_day';

const WEEKDAYS = [
  'sunday',
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
];

const MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];

function isRecordSupportedAccount(acc: Account): boolean {
  const t = (acc.type || '').toLowerCase();
  return t === 'checking' || t === 'savings' || t === 'credit card' || t === 'credit_card';
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
  return isNegative ? `-${formatted}` : formatted;
}

function capitalize(s: string): string {
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : '';
}

interface BillsPaneProps {
  bills: Bill[];
  accounts: Account[];
  categories: Category[];
  applyingId: string | null;
  onCreateBill: (payload: BillCreateRequest) => Promise<void>;
  onUpdateBill: (billId: string, payload: BillCreateRequest) => Promise<void>;
  onDeleteBill: (billId: string) => Promise<void>;
  onApplyBill: (bill: Bill) => Promise<void>;
}

export function BillsPane({
  bills,
  accounts,
  categories,
  applyingId,
  onCreateBill,
  onUpdateBill,
  onDeleteBill,
  onApplyBill,
}: BillsPaneProps) {
  const [editingBillId, setEditingBillId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Form fields
  const [amountStr, setAmountStr] = useState('');
  const [name, setName] = useState('');
  const [recipient, setRecipient] = useState('');
  const [description, setDescription] = useState('');
  const [frequency, setFrequency] = useState<BillFrequency>('monthly');
  const [intervalStr, setIntervalStr] = useState('1');
  const [lastDatePaid, setLastDatePaid] = useState('');
  const [reminder, setReminder] = useState(true);
  const [accountId, setAccountId] = useState('');
  const [categoryId, setCategoryId] = useState('');

  // Schedule fields
  const [scheduleMode, setScheduleMode] = useState<ScheduleMode>('none');
  const [weekday, setWeekday] = useState('monday');
  const [nthStr, setNthStr] = useState('1');
  const [dayStr, setDayStr] = useState('1');
  const [monthStr, setMonthStr] = useState('1');
  const [useYearlyNthMonth, setUseYearlyNthMonth] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');
  const [notice, setNotice] = useState('');

  const eligibleAccounts = useMemo(
    () => accounts.filter(isRecordSupportedAccount),
    [accounts]
  );

  const accountNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const a of accounts) map.set(a.id, a.name);
    return map;
  }, [accounts]);

  const categoryNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const c of categories) map.set(c.id, c.name);
    return map;
  }, [categories]);

  const sortedBills = useMemo(() => sortBillsForDisplay(bills), [bills]);
  const amountColorClass = getAmountColorClass(amountStr);

  const resetForm = () => {
    setEditingBillId(null);
    setAmountStr('');
    setName('');
    setRecipient('');
    setDescription('');
    setFrequency('monthly');
    setIntervalStr('1');
    setLastDatePaid('');
    setReminder(true);
    setAccountId('');
    setCategoryId('');
    setScheduleMode('none');
    setWeekday('monday');
    setNthStr('1');
    setDayStr('1');
    setMonthStr('1');
    setUseYearlyNthMonth(false);
    setFormError('');
  };

  const handleFrequencyChange = (nextFreq: BillFrequency) => {
    setFrequency(nextFreq);
    if (nextFreq === 'weekly' && scheduleMode !== 'none' && scheduleMode !== 'weekday') {
      setScheduleMode('none');
    } else if (
      nextFreq === 'monthly' &&
      scheduleMode !== 'none' &&
      scheduleMode !== 'day_of_period' &&
      scheduleMode !== 'nth_weekday'
    ) {
      setScheduleMode('none');
    } else if (nextFreq === 'yearly' && scheduleMode === 'weekday') {
      setScheduleMode('none');
    }
    if (nextFreq !== 'yearly') {
      setUseYearlyNthMonth(false);
    }
  };

  const startEdit = (bill: Bill) => {
    setEditingBillId(bill.id);
    setAmountStr(String(bill.amount));
    setName(bill.name || '');
    setRecipient(bill.recipient);
    setDescription(bill.description || '');
    setFrequency(bill.frequency);
    setIntervalStr(String(bill.interval || 1));
    setLastDatePaid(bill.last_date_paid || '');
    setReminder(bill.reminder);
    setAccountId(bill.account_id || '');
    setCategoryId(bill.category_id || '');
    setFormError('');
    setNotice('');

    if (!bill.schedule) {
      setScheduleMode('none');
      setWeekday('monday');
      setNthStr('1');
      setDayStr('1');
      setMonthStr('1');
      setUseYearlyNthMonth(false);
      return;
    }

    setScheduleMode(bill.schedule.type);
    setWeekday(bill.schedule.weekday || 'monday');
    setNthStr(bill.schedule.nth != null ? String(bill.schedule.nth) : '1');
    setDayStr(bill.schedule.day != null ? String(bill.schedule.day) : '1');
    setMonthStr(bill.schedule.month != null ? String(bill.schedule.month) : '1');
    setUseYearlyNthMonth(bill.frequency === 'yearly' && bill.schedule.month != null);
  };

  const buildSchedulePayload = (): BillSchedule | null => {
    if (scheduleMode === 'none') return null;
    if (frequency === 'weekly') {
      return { type: 'weekday', scope: 'week', weekday };
    }
    if (frequency === 'monthly') {
      if (scheduleMode === 'day_of_period') {
        return { type: 'day_of_period', scope: 'month', day: Number(dayStr) };
      }
      return {
        type: 'nth_weekday',
        scope: 'month',
        nth: Number(nthStr),
        weekday,
      };
    }
    // Yearly
    if (scheduleMode === 'day_of_period') {
      return { type: 'day_of_period', scope: 'year', day: Number(dayStr) };
    }
    if (scheduleMode === 'month_day') {
      return { type: 'month_day', scope: 'year', month: Number(monthStr), day: Number(dayStr) };
    }
    return {
      type: 'nth_weekday',
      scope: 'year',
      nth: Number(nthStr),
      weekday,
      month: useYearlyNthMonth ? Number(monthStr) : null,
    };
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');
    setNotice('');

    const parsedAmt = parseBillAmount(amountStr);
    if (isNaN(parsedAmt) || parsedAmt === 0) {
      setFormError('Please enter a valid non-zero amount (e.g., -125.00).');
      return;
    }
    if (parsedAmt > 0) {
      setFormError('Bill amount must be negative (e.g., -125.00).');
      return;
    }
    const parsedInt = parseInt(intervalStr, 10);
    if (isNaN(parsedInt) || parsedInt < 1) {
      setFormError('Interval must be at least 1.');
      return;
    }
    const trimmedName = name.trim();
    if (Array.from(trimmedName).length >= 20) {
      setFormError('Bill Name must be under 20 characters.');
      return;
    }
    if (!recipient.trim()) {
      setFormError('Recipient is required.');
      return;
    }

    const payload: BillCreateRequest = {
      account_id: accountId || null,
      category_id: accountId && categoryId ? categoryId : null,
      amount: parsedAmt,
      last_date_paid: lastDatePaid.trim() || null,
      frequency,
      interval: parsedInt,
      schedule: buildSchedulePayload(),
      name: trimmedName,
      recipient: recipient.trim(),
      description: description.trim(),
      reminder,
    };

    const label = trimmedName || payload.recipient;
    setSubmitting(true);
    try {
      if (editingBillId) {
        await onUpdateBill(editingBillId, payload);
        setNotice(`Updated bill "${label}".`);
      } else {
        await onCreateBill(payload);
        setNotice(`Added bill "${label}".`);
      }
      resetForm();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : 'Failed to save bill.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleConfirmDelete = async (billId: string) => {
    setDeletingId(billId);
    setFormError('');
    setNotice('');
    try {
      await onDeleteBill(billId);
      if (editingBillId === billId) resetForm();
      setConfirmDeleteId(null);
      setNotice('Bill deleted.');
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : 'Failed to delete bill.');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="card account-detail-card">
      <div className="account-detail-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>Bills</h2>
            <span className="badge-account-type badge-bills">Recurring</span>
            <CopyLinkButton className="ml-auto" />
          </div>
          <p className="account-desc-text">
            Manage recurring bills, due-date schedules, reminder alerts, and one-click account application
          </p>
        </div>
        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">Active Bills</span>
            <span className="balance-value balance-zero">{bills.length}</span>
            <span className="balance-subtext">Scheduled &amp; tracked</span>
          </div>
        </div>
      </div>

      <section className="new-record-section">
        <div className="section-title-row">
          <h3>{editingBillId ? 'Edit Bill' : 'New Bill'}</h3>
          <span className="section-subtitle">
            {editingBillId
              ? 'Update bill amount, cadence, schedule, or linked account'
              : 'Configure recurring schedule, reminder flag, and optional account link'}
          </span>
        </div>

        {formError && <div className="alert-error mb-3">{formError}</div>}
        {notice && <div className="notice-banner mb-3">{notice}</div>}

        <form onSubmit={handleSubmit} className="new-record-form">
          <div className="bill-form-primary">
            <div className="field-group field-amount">
              <label htmlFor="bill-amount">Amount</label>
              <div className={`amount-input-wrapper ${amountColorClass}`}>
                <span className={`currency-prefix ${amountColorClass}`}>$</span>
                <input
                  id="bill-amount"
                  type="text"
                  inputMode="decimal"
                  required
                  placeholder="0.00"
                  value={amountStr}
                  onChange={(e) => {
                    const val = e.target.value;
                    if (/^[0-9.,-]*$/.test(val)) {
                      setAmountStr(val);
                    }
                  }}
                  onBlur={() => setAmountStr((prev) => normalizeNegativeBillAmountInput(prev))}
                  className={`amount-input ${amountColorClass}`}
                />
              </div>
            </div>

            <div className="field-group">
              <label htmlFor="bill-name">Name</label>
              <input
                id="bill-name"
                type="text"
                maxLength={19}
                placeholder="e.g. Electric"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="field-group">
              <label htmlFor="bill-recipient">Recipient</label>
              <input
                id="bill-recipient"
                type="text"
                maxLength={255}
                required
                placeholder="e.g. City Electric Utility"
                value={recipient}
                onChange={(e) => setRecipient(e.target.value)}
              />
            </div>

            <div className="field-group">
              <label htmlFor="bill-frequency">Frequency</label>
              <select
                id="bill-frequency"
                value={frequency}
                onChange={(e) => handleFrequencyChange(e.target.value as BillFrequency)}
              >
                <option value="weekly">Weekly</option>
                <option value="monthly">Monthly</option>
                <option value="yearly">Yearly</option>
              </select>
            </div>

            <div className="field-group">
              <label htmlFor="bill-interval" title="Repeat interval (e.g., 1 = every period, 6 = every 6 periods)">Interval</label>
              <input
                id="bill-interval"
                type="number"
                min={1}
                step={1}
                required
                value={intervalStr}
                onKeyDown={(e) => {
                  if (['-', '+', 'e', 'E', '.', ','].includes(e.key)) {
                    e.preventDefault();
                  }
                }}
                onChange={(e) => {
                  const val = e.target.value;
                  if (isValidBillIntervalInput(val)) {
                    setIntervalStr(val);
                  }
                }}
                onBlur={() => setIntervalStr((prev) => normalizePositiveBillIntervalInput(prev))}
              />
            </div>

            <div className="field-group">
              <label htmlFor="bill-last-paid">Last Date Paid</label>
              <DateInput
                id="bill-last-paid"
                value={lastDatePaid}
                onChange={setLastDatePaid}
              />
            </div>
          </div>

          <div className="bill-form-grid mt-3">
            <div className="field-group">
              <label htmlFor="bill-schedule-mode">Schedule Rule</label>
              <select
                id="bill-schedule-mode"
                value={scheduleMode}
                onChange={(e) => setScheduleMode(e.target.value as ScheduleMode)}
              >
                <option value="none">None (use frequency + last paid)</option>
                {frequency === 'weekly' && <option value="weekday">Day of week</option>}
                {frequency === 'monthly' && (
                  <>
                    <option value="day_of_period">Day of month (e.g. 1st, 15th, 31st)</option>
                    <option value="nth_weekday">Nth weekday of month</option>
                  </>
                )}
                {frequency === 'yearly' && (
                  <>
                    <option value="month_day">Specific Month &amp; Day</option>
                    <option value="day_of_period">Day of year (1–366)</option>
                    <option value="nth_weekday">Nth weekday of year / month</option>
                  </>
                )}
              </select>
            </div>

            {scheduleMode === 'weekday' && (
              <div className="field-group">
                <label htmlFor="bill-weekday">Weekday</label>
                <select
                  id="bill-weekday"
                  value={weekday}
                  onChange={(e) => setWeekday(e.target.value)}
                >
                  {WEEKDAYS.map((wd) => (
                    <option key={wd} value={wd}>{capitalize(wd)}</option>
                  ))}
                </select>
              </div>
            )}

            {scheduleMode === 'day_of_period' && (
              <div className="field-group">
                <label htmlFor="bill-day-of-period">
                  {frequency === 'yearly' ? 'Day of Year (1–366)' : 'Day of Month (1–31, 31 = Last)'}
                </label>
                <input
                  id="bill-day-of-period"
                  type="number"
                  min={1}
                  max={frequency === 'yearly' ? 366 : 31}
                  required
                  value={dayStr}
                  onChange={(e) => setDayStr(e.target.value)}
                />
              </div>
            )}

            {scheduleMode === 'month_day' && (
              <>
                <div className="field-group">
                  <label htmlFor="bill-month">Month</label>
                  <select
                    id="bill-month"
                    value={monthStr}
                    onChange={(e) => setMonthStr(e.target.value)}
                  >
                    {MONTHS.map((mName, idx) => (
                      <option key={mName} value={String(idx + 1)}>{mName}</option>
                    ))}
                  </select>
                </div>
                <div className="field-group">
                  <label htmlFor="bill-month-day">Day</label>
                  <input
                    id="bill-month-day"
                    type="number"
                    min={1}
                    max={31}
                    required
                    value={dayStr}
                    onChange={(e) => setDayStr(e.target.value)}
                  />
                </div>
              </>
            )}

            {scheduleMode === 'nth_weekday' && (
              <>
                <div className="field-group">
                  <label htmlFor="bill-nth">Occurrence</label>
                  <select
                    id="bill-nth"
                    value={nthStr}
                    onChange={(e) => setNthStr(e.target.value)}
                  >
                    <option value="1">1st</option>
                    <option value="2">2nd</option>
                    <option value="3">3rd</option>
                    <option value="4">4th</option>
                    <option value="5">5th</option>
                    <option value="-1">Last</option>
                  </select>
                </div>
                <div className="field-group">
                  <label htmlFor="bill-nth-weekday">Weekday</label>
                  <select
                    id="bill-nth-weekday"
                    value={weekday}
                    onChange={(e) => setWeekday(e.target.value)}
                  >
                    {WEEKDAYS.map((wd) => (
                      <option key={wd} value={wd}>{capitalize(wd)}</option>
                    ))}
                  </select>
                </div>
                {frequency === 'yearly' && (
                  <div className="field-group">
                    <label htmlFor="bill-yearly-month">Specific Month (Optional)</label>
                    <select
                      id="bill-yearly-month"
                      value={useYearlyNthMonth ? monthStr : ''}
                      onChange={(e) => {
                        if (!e.target.value) {
                          setUseYearlyNthMonth(false);
                        } else {
                          setUseYearlyNthMonth(true);
                          setMonthStr(e.target.value);
                        }
                      }}
                    >
                      <option value="">Any month in year (1st..53rd)</option>
                      {MONTHS.map((mName, idx) => (
                        <option key={mName} value={String(idx + 1)}>{mName}</option>
                      ))}
                    </select>
                  </div>
                )}
              </>
            )}
          </div>

          <div className="bill-form-grid mt-3">
            <div className="field-group">
              <label htmlFor="bill-account">Linked Account (Optional)</label>
              <select
                id="bill-account"
                value={accountId}
                onChange={(e) => {
                  const nextAcc = e.target.value;
                  setAccountId(nextAcc);
                  if (!nextAcc) setCategoryId('');
                }}
              >
                <option value="">Unlinked</option>
                {eligibleAccounts.map((acc) => (
                  <option key={acc.id} value={acc.id}>
                    {acc.name} ({acc.type})
                  </option>
                ))}
              </select>
            </div>

            <div className="field-group">
              <label htmlFor="bill-category">Category (Requires Account)</label>
              <select
                id="bill-category"
                value={accountId ? categoryId : ''}
                disabled={!accountId}
                onChange={(e) => setCategoryId(e.target.value)}
              >
                <option value="">Uncategorized</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}{c.is_default ? ' (Default)' : ''}
                  </option>
                ))}
              </select>
            </div>

            <div className="field-group" style={{ gridColumn: 'span 2' }}>
              <label htmlFor="bill-description">Description</label>
              <input
                id="bill-description"
                type="text"
                maxLength={500}
                placeholder="Optional memo for created transaction records"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>

            <div className="field-group" style={{ justifyContent: 'flex-end' }}>
              <label className="uncategorized-toggle" style={{ height: '38px' }}>
                <input
                  type="checkbox"
                  checked={reminder}
                  onChange={(e) => setReminder(e.target.checked)}
                />
                <span>Reminder alert</span>
              </label>
            </div>
          </div>

          <div className="new-record-actions">
            <button type="submit" className="btn-primary btn-sm" disabled={submitting}>
              {submitting ? 'Saving...' : editingBillId ? 'Save Bill' : 'Add Bill'}
            </button>
            {editingBillId && (
              <button type="button" className="btn-secondary btn-sm" onClick={resetForm}>
                Cancel Edit
              </button>
            )}
          </div>
        </form>
      </section>

      <section className="records-table-section">
        <div className="section-title-row">
          <h3>Configured Bills <span className="records-count-badge">{bills.length}</span></h3>
        </div>
        <div className="records-table-container">
          <table className="records-table">
            <thead>
              <tr>
                <th>Bill</th>
                <th style={{ textAlign: 'right' }}>Amount</th>
                <th>Schedule</th>
                <th>Next Due</th>
                <th>Last Paid</th>
                <th>Account / Category</th>
                <th>Reminder</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sortedBills.length === 0 ? (
                <tr>
                  <td colSpan={8} className="records-empty">
                    No bills configured yet. Use the form above to add your first bill.
                  </td>
                </tr>
              ) : (
                sortedBills.map((bill) => {
                  const isApplying = applyingId === bill.id;
                  const isConfirming = confirmDeleteId === bill.id;
                  const isDeleting = deletingId === bill.id;
                  const accName = bill.account_id ? accountNameById.get(bill.account_id) : null;
                  const catName = bill.category_id ? categoryNameById.get(bill.category_id) : null;
                  return (
                    <tr key={bill.id} className={editingBillId === bill.id ? 'record-row-selected' : ''}>
                      <td>
                        <div className="font-semibold">{getBillDisplayName(bill)}</div>
                        {bill.name && bill.name.trim() !== '' && (
                          <div className="text-sm">{bill.recipient}</div>
                        )}
                        {bill.description && (
                          <div className="text-muted text-sm">{bill.description}</div>
                        )}
                      </td>
                      <td
                        className={bill.amount < 0 ? 'amount-negative td-amount' : 'amount-positive td-amount'}
                        style={{ textAlign: 'right' }}
                      >
                        {formatCurrency(bill.amount)}
                      </td>
                      <td>{formatBillScheduleLabel(bill)}</td>
                      <td>
                        <div className="bill-due-cell">
                          <span>
                            {bill.next_due_date ? isoToDisplayDate(bill.next_due_date) : '—'}
                          </span>
                          <span className={`badge-bill-status badge-due-${bill.due_status}`}>
                            {capitalize(bill.due_status)}
                          </span>
                        </div>
                      </td>
                      <td>
                        {bill.last_date_paid ? isoToDisplayDate(bill.last_date_paid) : <span className="text-muted">Never</span>}
                      </td>
                      <td>
                        {accName ? (
                          <div>
                            <div>{accName}</div>
                            <div className="text-muted text-sm">{catName || 'Uncategorized'}</div>
                          </div>
                        ) : (
                          <span className="text-muted">Unlinked</span>
                        )}
                      </td>
                      <td>{bill.reminder ? 'On' : 'Off'}</td>
                      <td>
                        {isConfirming ? (
                          <div className="delete-confirm-actions">
                            <button
                              type="button"
                              className="btn-secondary btn-sm"
                              onClick={() => setConfirmDeleteId(null)}
                              disabled={isDeleting}
                            >
                              Cancel
                            </button>
                            <button
                              type="button"
                              className="btn-danger-outline btn-sm"
                              onClick={() => handleConfirmDelete(bill.id)}
                              disabled={isDeleting}
                            >
                              {isDeleting ? 'Deleting...' : 'Confirm'}
                            </button>
                          </div>
                        ) : (
                          <div className="bill-row-actions">
                            {bill.account_id && (
                              <button
                                type="button"
                                className="btn-primary btn-sm"
                                disabled={isApplying}
                                onClick={() => onApplyBill(bill)}
                              >
                                {isApplying ? 'Applying...' : 'Apply'}
                              </button>
                            )}
                            <button
                              type="button"
                              className="btn-secondary btn-sm"
                              onClick={() => startEdit(bill)}
                            >
                              Edit
                            </button>
                            <button
                              type="button"
                              className="btn-danger-outline btn-sm"
                              onClick={() => setConfirmDeleteId(bill.id)}
                            >
                              Delete
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
