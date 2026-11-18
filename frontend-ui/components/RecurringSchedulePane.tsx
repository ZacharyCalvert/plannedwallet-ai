'use client';

import { useState, useMemo } from 'react';
import {
  Account,
  Category,
  RecurringFrequency,
  RecurringSchedule,
  RecurringTransaction,
  RecurringTransactionRequest,
} from '../lib/types';
import {
  formatBillScheduleLabel,
  formatRecurringIntervalLabel,
  getAmountColorClass,
  getBillDisplayName,
  isValidBillIntervalInput,
  normalizeNegativeBillAmountInput,
  normalizePositiveBillIntervalInput,
  normalizePositiveIncomeAmountInput,
  parseBillAmount,
  sortBillsForDisplay,
} from '../lib/bills';
import { DateInput, isoToDisplayDate } from './DateInput';

type ScheduleMode =
  | 'none'
  | 'weekday'
  | 'day_of_period'
  | 'nth_weekday'
  | 'month_day'
  | 'first_fifteenth'
  | 'fifteenth_last';

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

interface PaneCopyConfig {
  title: string;
  description: string;
  countLabel: string;
  countSubtext: string;
  newTitle: string;
  editTitle: string;
  newSubtitle: string;
  editSubtitle: string;
  namePlaceholder: string;
  counterpartyLabel: string;
  counterpartyPlaceholder: string;
  lastDateLabel: string;
  emptyAmountErr: string;
  signAmountErr: string;
  missingCounterpartyErr: string;
  nameTooLongErr: string;
  saveFailedErr: string;
  deleteFailedErr: string;
  deletedNotice: string;
  addSubmitLabel: string;
  saveSubmitLabel: string;
  tableHeading: string;
  firstColumnHeader: string;
  emptyTableText: string;
  addedNotice: (label: string) => string;
  updatedNotice: (label: string) => string;
}

const COPY_BY_KIND: Record<'bill' | 'income', PaneCopyConfig> = {
  bill: {
    title: 'Bills',
    description: 'Manage recurring bills, due-date schedules, reminder alerts, and one-click account application',
    countLabel: 'Active Bills',
    countSubtext: 'Scheduled & tracked',
    newTitle: 'New Bill',
    editTitle: 'Edit Bill',
    newSubtitle: 'Configure recurring schedule, reminder flag, and optional account link',
    editSubtitle: 'Update bill amount, cadence, schedule, or linked account',
    namePlaceholder: 'e.g. Electric',
    counterpartyLabel: 'Recipient',
    counterpartyPlaceholder: 'e.g. City Electric Utility',
    lastDateLabel: 'Last Date Paid',
    emptyAmountErr: 'Please enter a valid non-zero amount (e.g., -125.00).',
    signAmountErr: 'Bill amount must be negative (e.g., -125.00).',
    missingCounterpartyErr: 'Recipient is required.',
    nameTooLongErr: 'Bill Name must be under 20 characters.',
    saveFailedErr: 'Failed to save bill.',
    deleteFailedErr: 'Failed to delete bill.',
    deletedNotice: 'Bill deleted.',
    addSubmitLabel: 'Add Bill',
    saveSubmitLabel: 'Save Bill',
    tableHeading: 'Configured Bills',
    firstColumnHeader: 'Bill',
    emptyTableText: 'No bills configured yet. Use the form above to add your first bill.',
    addedNotice: (label) => `Added bill "${label}".`,
    updatedNotice: (label) => `Updated bill "${label}".`,
  },
  income: {
    title: 'Income',
    description: 'Manage recurring income streams, deposit schedules, reminder alerts, and one-click account application',
    countLabel: 'Active Income',
    countSubtext: 'Scheduled & tracked',
    newTitle: 'New Income',
    editTitle: 'Edit Income',
    newSubtitle: 'Configure recurring schedule, reminder flag, and optional account link',
    editSubtitle: 'Update income amount, cadence, schedule, or linked account',
    namePlaceholder: 'e.g. Paycheck',
    counterpartyLabel: 'Source',
    counterpartyPlaceholder: 'e.g. Employer Payroll',
    lastDateLabel: 'Last Date Received',
    emptyAmountErr: 'Please enter a valid non-zero amount (e.g., 2,500.00).',
    signAmountErr: 'Income amount must be positive (e.g., 2,500.00).',
    missingCounterpartyErr: 'Source is required.',
    nameTooLongErr: 'Income Name must be under 20 characters.',
    saveFailedErr: 'Failed to save income.',
    deleteFailedErr: 'Failed to delete income.',
    deletedNotice: 'Income deleted.',
    addSubmitLabel: 'Add Income',
    saveSubmitLabel: 'Save Income',
    tableHeading: 'Configured Income',
    firstColumnHeader: 'Income',
    emptyTableText: 'No income configured yet. Use the form above to add your first income entry.',
    addedNotice: (label) => `Added income "${label}".`,
    updatedNotice: (label) => `Updated income "${label}".`,
  },
};

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

interface RecurringSchedulePaneProps {
  kind: 'bill' | 'income';
  items: RecurringTransaction[];
  accounts: Account[];
  categories: Category[];
  applyingId: string | null;
  onCreate: (payload: RecurringTransactionRequest) => Promise<void>;
  onUpdate: (id: string, payload: RecurringTransactionRequest) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onApply: (item: RecurringTransaction) => Promise<void>;
}

export function RecurringSchedulePane({
  kind,
  items,
  accounts,
  categories,
  applyingId,
  onCreate,
  onUpdate,
  onDelete,
  onApply,
}: RecurringSchedulePaneProps) {
  const copy = COPY_BY_KIND[kind];
  const isIncome = kind === 'income';

  const [editingId, setEditingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const [amountStr, setAmountStr] = useState('');
  const [name, setName] = useState('');
  const [counterparty, setCounterparty] = useState('');
  const [description, setDescription] = useState('');
  const [frequency, setFrequency] = useState<RecurringFrequency>('monthly');
  const [intervalStr, setIntervalStr] = useState('1');
  const [lastDate, setLastDate] = useState('');
  const [reminder, setReminder] = useState(true);
  const [accountId, setAccountId] = useState('');
  const [categoryId, setCategoryId] = useState('');

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

  const sortedItems = useMemo(() => sortBillsForDisplay(items), [items]);
  const amountColorClass = getAmountColorClass(amountStr);

  const resetForm = () => {
    setEditingId(null);
    setAmountStr('');
    setName('');
    setCounterparty('');
    setDescription('');
    setFrequency('monthly');
    setIntervalStr('1');
    setLastDate('');
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

  const handleFrequencyChange = (nextFreq: RecurringFrequency) => {
    setFrequency(nextFreq);
    if (nextFreq === 'bi-monthly') {
      if (scheduleMode !== 'first_fifteenth' && scheduleMode !== 'fifteenth_last') {
        setScheduleMode('first_fifteenth');
      }
      setUseYearlyNthMonth(false);
      return;
    }
    if (nextFreq === 'weekly' && scheduleMode !== 'none' && scheduleMode !== 'weekday') {
      setScheduleMode('none');
    } else if (
      nextFreq === 'monthly' &&
      scheduleMode !== 'none' &&
      scheduleMode !== 'day_of_period' &&
      scheduleMode !== 'nth_weekday'
    ) {
      setScheduleMode('none');
    } else if (
      nextFreq === 'yearly' &&
      (scheduleMode === 'weekday' ||
        scheduleMode === 'first_fifteenth' ||
        scheduleMode === 'fifteenth_last')
    ) {
      setScheduleMode('none');
    }
    if (nextFreq !== 'yearly') {
      setUseYearlyNthMonth(false);
    }
  };

  const startEdit = (item: RecurringTransaction) => {
    setEditingId(item.id);
    setAmountStr(String(item.amount));
    setName(item.name || '');
    setCounterparty(item.entity);
    setDescription(item.description || '');
    setFrequency(item.frequency);
    setIntervalStr(String(item.interval || 1));
    setLastDate(item.last_date_applied || '');
    setReminder(item.reminder);
    setAccountId(item.account_id || '');
    setCategoryId(item.category_id || '');
    setFormError('');
    setNotice('');

    if (!item.schedule) {
      setScheduleMode('none');
      setWeekday('monday');
      setNthStr('1');
      setDayStr('1');
      setMonthStr('1');
      setUseYearlyNthMonth(false);
      return;
    }

    setScheduleMode(item.schedule.type);
    setWeekday(item.schedule.weekday || 'monday');
    setNthStr(item.schedule.nth != null ? String(item.schedule.nth) : '1');
    setDayStr(item.schedule.day != null ? String(item.schedule.day) : '1');
    setMonthStr(item.schedule.month != null ? String(item.schedule.month) : '1');
    setUseYearlyNthMonth(item.frequency === 'yearly' && item.schedule.month != null);
  };

  const buildSchedulePayload = (): RecurringSchedule | null => {
    if (frequency === 'bi-monthly') {
      return {
        type: scheduleMode === 'fifteenth_last' ? 'fifteenth_last' : 'first_fifteenth',
        scope: 'bi-monthly',
      };
    }
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
      setFormError(copy.emptyAmountErr);
      return;
    }
    if (isIncome ? parsedAmt < 0 : parsedAmt > 0) {
      setFormError(copy.signAmountErr);
      return;
    }
    const parsedInt = parseInt(intervalStr, 10);
    if (isNaN(parsedInt) || parsedInt < 1) {
      setFormError('Interval must be at least 1.');
      return;
    }
    const trimmedName = name.trim();
    if (Array.from(trimmedName).length >= 20) {
      setFormError(copy.nameTooLongErr);
      return;
    }
    if (!counterparty.trim()) {
      setFormError(copy.missingCounterpartyErr);
      return;
    }

    const payload: RecurringTransactionRequest = {
      account_id: accountId || null,
      category_id: accountId && categoryId ? categoryId : null,
      amount: parsedAmt,
      last_date_applied: lastDate.trim() || null,
      frequency,
      interval: parsedInt,
      schedule: buildSchedulePayload(),
      name: trimmedName,
      entity: counterparty.trim(),
      description: description.trim(),
      reminder,
    };

    const label = trimmedName || payload.entity;
    setSubmitting(true);
    try {
      if (editingId) {
        await onUpdate(editingId, payload);
        setNotice(copy.updatedNotice(label));
      } else {
        await onCreate(payload);
        setNotice(copy.addedNotice(label));
      }
      resetForm();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : copy.saveFailedErr);
    } finally {
      setSubmitting(false);
    }
  };

  const handleConfirmDelete = async (id: string) => {
    setDeletingId(id);
    setFormError('');
    setNotice('');
    try {
      await onDelete(id);
      if (editingId === id) resetForm();
      setConfirmDeleteId(null);
      setNotice(copy.deletedNotice);
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : copy.deleteFailedErr);
    } finally {
      setDeletingId(null);
    }
  };

  const normalizeAmountOnBlur = () => {
    setAmountStr((prev) =>
      isIncome ? normalizePositiveIncomeAmountInput(prev) : normalizeNegativeBillAmountInput(prev)
    );
  };

  return (
    <div className="card account-detail-card">
      <div className="account-detail-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>{copy.title}</h2>
          </div>
          <p className="account-desc-text">{copy.description}</p>
        </div>
        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">{copy.countLabel}</span>
            <span className="balance-value balance-zero">{items.length}</span>
            <span className="balance-subtext">{copy.countSubtext}</span>
          </div>
        </div>
      </div>

      <section className="new-record-section">
        <div className="section-title-row">
          <h3>{editingId ? copy.editTitle : copy.newTitle}</h3>
          <span className="section-subtitle">
            {editingId ? copy.editSubtitle : copy.newSubtitle}
          </span>
        </div>

        {formError && <div className="alert-error mb-3">{formError}</div>}
        {notice && <div className="notice-banner mb-3">{notice}</div>}

        <form onSubmit={handleSubmit} className="new-record-form">
          <div className="bill-form-primary">
            <div className="field-group field-amount">
              <label htmlFor={`${kind}-amount`}>Amount</label>
              <div className={`amount-input-wrapper ${amountColorClass}`}>
                <span className={`currency-prefix ${amountColorClass}`}>$</span>
                <input
                  id={`${kind}-amount`}
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
                  onBlur={normalizeAmountOnBlur}
                  className={`amount-input ${amountColorClass}`}
                />
              </div>
            </div>

            <div className="field-group">
              <label htmlFor={`${kind}-name`}>Name</label>
              <input
                id={`${kind}-name`}
                type="text"
                maxLength={19}
                placeholder={copy.namePlaceholder}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="field-group">
              <label htmlFor={`${kind}-counterparty`}>{copy.counterpartyLabel}</label>
              <input
                id={`${kind}-counterparty`}
                type="text"
                maxLength={255}
                required
                placeholder={copy.counterpartyPlaceholder}
                value={counterparty}
                onChange={(e) => setCounterparty(e.target.value)}
              />
            </div>

            <div className="field-group">
              <label htmlFor={`${kind}-frequency`}>Frequency</label>
              <select
                id={`${kind}-frequency`}
                value={frequency}
                onChange={(e) => handleFrequencyChange(e.target.value as RecurringFrequency)}
              >
                <option value="weekly">Weekly</option>
                <option value="bi-monthly">Bi-Monthly</option>
                <option value="monthly">Monthly</option>
                <option value="yearly">Yearly</option>
              </select>
            </div>

            <div className="field-group">
              <label
                htmlFor={`${kind}-interval`}
                title="Repeat interval (e.g., 1 = every period, 6 = every 6 periods)"
              >
                {formatRecurringIntervalLabel(frequency, intervalStr)}
              </label>
              <input
                id={`${kind}-interval`}
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
              <label htmlFor={`${kind}-last-date`}>{copy.lastDateLabel}</label>
              <DateInput
                id={`${kind}-last-date`}
                value={lastDate}
                onChange={setLastDate}
              />
            </div>
          </div>

          <div className="bill-form-grid mt-3">
            <div className="field-group">
              <label htmlFor={`${kind}-schedule-mode`}>Schedule Rule</label>
              <select
                id={`${kind}-schedule-mode`}
                value={scheduleMode}
                onChange={(e) => setScheduleMode(e.target.value as ScheduleMode)}
              >
                {frequency === 'bi-monthly' ? (
                  <>
                    <option value="first_fifteenth">1st and 15th of month</option>
                    <option value="fifteenth_last">15th and last day of month</option>
                  </>
                ) : (
                  <>
                    <option value="none">None (use frequency + last date)</option>
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
                  </>
                )}
              </select>
            </div>

            {scheduleMode === 'weekday' && (
              <div className="field-group">
                <label htmlFor={`${kind}-weekday`}>Weekday</label>
                <select
                  id={`${kind}-weekday`}
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
                <label htmlFor={`${kind}-day-of-period`}>
                  {frequency === 'yearly' ? 'Day of Year (1–366)' : 'Day of Month (1–31, 31 = Last)'}
                </label>
                <input
                  id={`${kind}-day-of-period`}
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
                  <label htmlFor={`${kind}-month`}>Month</label>
                  <select
                    id={`${kind}-month`}
                    value={monthStr}
                    onChange={(e) => setMonthStr(e.target.value)}
                  >
                    {MONTHS.map((mName, idx) => (
                      <option key={mName} value={String(idx + 1)}>{mName}</option>
                    ))}
                  </select>
                </div>
                <div className="field-group">
                  <label htmlFor={`${kind}-month-day`}>Day</label>
                  <input
                    id={`${kind}-month-day`}
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
                  <label htmlFor={`${kind}-nth`}>Occurrence</label>
                  <select
                    id={`${kind}-nth`}
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
                  <label htmlFor={`${kind}-nth-weekday`}>Weekday</label>
                  <select
                    id={`${kind}-nth-weekday`}
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
                    <label htmlFor={`${kind}-yearly-month`}>Specific Month (Optional)</label>
                    <select
                      id={`${kind}-yearly-month`}
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
              <label htmlFor={`${kind}-account`}>Linked Account (Optional)</label>
              <select
                id={`${kind}-account`}
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
              <label htmlFor={`${kind}-category`}>Category (Requires Account)</label>
              <select
                id={`${kind}-category`}
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

            <div className="field-group bill-form-span-2">
              <label htmlFor={`${kind}-description`}>Description</label>
              <input
                id={`${kind}-description`}
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
            <button type="submit" className="btn-primary" disabled={submitting}>
              {submitting ? 'Saving...' : editingId ? copy.saveSubmitLabel : copy.addSubmitLabel}
            </button>
            {editingId && (
              <button
                type="button"
                className="btn-secondary"
                disabled={submitting}
                onClick={resetForm}
              >
                Cancel Edit
              </button>
            )}
          </div>
        </form>
      </section>

      <section className="records-table-section">
        <div className="section-title-row">
          <h3>{copy.tableHeading} <span className="records-count-badge">{items.length}</span></h3>
        </div>
        <div className="records-table-container">
          <table className="records-table">
            <thead>
              <tr>
                <th>{copy.firstColumnHeader}</th>
                <th style={{ textAlign: 'right' }}>Amount</th>
                <th>Schedule</th>
                <th>Next Due</th>
                <th>{copy.lastDateLabel}</th>
                <th>Account / Category</th>
                <th>Reminder</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sortedItems.length === 0 ? (
                <tr>
                  <td colSpan={8} className="records-empty">
                    {copy.emptyTableText}
                  </td>
                </tr>
              ) : (
                sortedItems.map((item) => {
                  const isApplying = applyingId === item.id;
                  const isConfirming = confirmDeleteId === item.id;
                  const isDeleting = deletingId === item.id;
                  const accName = item.account_id ? accountNameById.get(item.account_id) : null;
                  const catName = item.category_id ? categoryNameById.get(item.category_id) : null;
                  return (
                    <tr key={item.id} className={editingId === item.id ? 'record-row-selected' : ''}>
                      <td>
                        <div className="font-semibold">{getBillDisplayName(item)}</div>
                        {item.name && item.name.trim() !== '' && (
                          <div className="text-sm">{item.entity}</div>
                        )}
                        {item.description && (
                          <div className="text-muted text-sm">{item.description}</div>
                        )}
                      </td>
                      <td
                        className={item.amount < 0 ? 'amount-negative td-amount' : 'amount-positive td-amount'}
                        style={{ textAlign: 'right' }}
                      >
                        {formatCurrency(item.amount)}
                      </td>
                      <td>{formatBillScheduleLabel(item)}</td>
                      <td>
                        <div className="bill-due-cell">
                          <span>
                            {item.next_due_date ? isoToDisplayDate(item.next_due_date) : '—'}
                          </span>
                          <span className={`badge-bill-status badge-due-${item.due_status}`}>
                            {capitalize(item.due_status)}
                          </span>
                        </div>
                      </td>
                      <td>
                        {item.last_date_applied ? isoToDisplayDate(item.last_date_applied) : <span className="text-muted">Never</span>}
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
                      <td>{item.reminder ? 'On' : 'Off'}</td>
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
                              onClick={() => handleConfirmDelete(item.id)}
                              disabled={isDeleting}
                            >
                              {isDeleting ? 'Deleting...' : 'Confirm'}
                            </button>
                          </div>
                        ) : (
                          <div className="bill-row-actions">
                            {item.account_id && (
                              <button
                                type="button"
                                className="btn-primary btn-sm"
                                disabled={isApplying}
                                onClick={() => onApply(item)}
                              >
                                {isApplying ? 'Applying...' : 'Apply'}
                              </button>
                            )}
                            <button
                              type="button"
                              className="btn-secondary btn-sm"
                              onClick={() => startEdit(item)}
                            >
                              Edit
                            </button>
                            <button
                              type="button"
                              className="btn-danger-outline btn-sm"
                              onClick={() => setConfirmDeleteId(item.id)}
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
