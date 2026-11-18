import type { RecurringFrequency, RecurringTransaction } from './types.ts';

const MONTH_NAMES = [
  '',
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

function ordinal(n: number): string {
  if (n === -1) return 'Last';
  const mod100 = n % 100;
  if (mod100 >= 11 && mod100 <= 13) return `${n}th`;
  switch (n % 10) {
    case 1:
      return `${n}st`;
    case 2:
      return `${n}nd`;
    case 3:
      return `${n}rd`;
    default:
      return `${n}th`;
  }
}

function capitalize(s: string): string {
  if (!s) return '';
  return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
}

export function getLocalTodayIso(ref: Date = new Date()): string {
  const y = ref.getFullYear();
  const m = String(ref.getMonth() + 1).padStart(2, '0');
  const d = String(ref.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

function parseIsoUtcMs(isoDate: string): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec((isoDate || '').trim());
  if (!match) return null;
  const [, y, m, d] = match;
  return Date.UTC(Number(y), Number(m) - 1, Number(d));
}

export function getBillDaysUntilDue(nextDueDateIso: string | null | undefined, todayIso?: string): number | null {
  if (!nextDueDateIso) return null;
  const dueMs = parseIsoUtcMs(nextDueDateIso);
  const todayMs = parseIsoUtcMs(todayIso || getLocalTodayIso());
  if (dueMs === null || todayMs === null) return null;
  return Math.round((dueMs - todayMs) / 86400000);
}

/**
 * Determines whether a reminder banner should appear for a recurring bill or income on todayIso:
 * - Overdue (today > next_due_date, i.e. daysUntil < 0)
 * - Weekly item scheduled within the next 48 hours (0..2 days)
 * - Monthly item scheduled within the next 1 week (0..7 days)
 * - Yearly item scheduled within the next 30 days (0..30 days)
 */
export type RecurringScheduleLike = Pick<
  RecurringTransaction,
  'frequency' | 'interval' | 'schedule'
>;

export type RecurringReminderLike = Pick<
  RecurringTransaction,
  'reminder' | 'next_due_date' | 'frequency'
>;

export type RecurringDisplayLike = {
  next_due_date: string | null;
  name?: string;
  entity?: string;
};

export function shouldShowBillReminder(rt: RecurringReminderLike, todayIso?: string): boolean {
  if (!rt.reminder || !rt.next_due_date) {
    return false;
  }
  const days = getBillDaysUntilDue(rt.next_due_date, todayIso);
  if (days === null) {
    return false;
  }
  if (days < 0) {
    return true;
  }
  switch (rt.frequency) {
    case 'weekly':
      return days <= 2;
    case 'bi-monthly':
      return days <= 3;
    case 'monthly':
      return days <= 7;
    case 'yearly':
      return days <= 30;
    default:
      return false;
  }
}

export function parseBillAmount(raw: string): number {
  let clean = raw.replace(/[$,\s]/g, '').trim();
  if (!clean) return NaN;
  const isParenNeg = clean.startsWith('(') && clean.endsWith(')');
  if (isParenNeg) {
    clean = '-' + clean.slice(1, -1);
  }
  if (!/^[+-]?\d*(\.\d+)?$/.test(clean)) return NaN;
  const val = parseFloat(clean);
  return Number.isFinite(val) ? val : NaN;
}

export function normalizeNegativeBillAmountInput(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return raw;
  const parsed = parseBillAmount(trimmed);
  if (isNaN(parsed) || parsed <= 0) {
    return raw;
  }
  const withoutPlus = trimmed.startsWith('+') ? trimmed.slice(1).trim() : trimmed;
  return withoutPlus.startsWith('-') ? withoutPlus : `-${withoutPlus}`;
}

export function normalizePositiveIncomeAmountInput(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return raw;
  const parsed = parseBillAmount(trimmed);
  if (isNaN(parsed) || parsed >= 0) {
    return trimmed.startsWith('+') ? trimmed.slice(1).trim() : raw;
  }
  return trimmed.startsWith('-') ? trimmed.slice(1).trim() : trimmed;
}

export function isValidBillIntervalInput(raw: string): boolean {
  return raw === '' || /^[1-9]\d*$/.test(raw);
}

export function normalizePositiveBillIntervalInput(raw: string): string {
  if (!/^[1-9]\d*$/.test(raw.trim())) {
    return '1';
  }
  return raw.trim();
}

export function formatRecurringIntervalLabel(
  frequency: RecurringFrequency,
  intervalRaw: string
): string {
  const parsed = parseInt(intervalRaw, 10);
  const n = Number.isFinite(parsed) && parsed >= 1 ? parsed : 1;
  switch (frequency) {
    case 'weekly':
      return `Every ${n} ${n === 1 ? 'week' : 'weeks'}`;
    case 'bi-monthly':
      return n === 1 ? 'Twice a month' : `Every ${n} half-months`;
    case 'monthly':
      return `Every ${n} ${n === 1 ? 'month' : 'months'}`;
    case 'yearly':
      return `Every ${n} ${n === 1 ? 'year' : 'years'}`;
    default:
      return `Every ${n} ${n === 1 ? 'period' : 'periods'}`;
  }
}

function formatCadencePrefix(frequency: RecurringFrequency, interval: number): string {
  const step = interval > 1 ? interval : 1;
  if (frequency === 'bi-monthly') {
    return step === 1 ? 'Bi-monthly' : `Bi-monthly (every ${step})`;
  }
  const unit = frequency === 'weekly' ? 'week' : frequency === 'monthly' ? 'month' : 'year';
  return step === 1 ? `Every ${unit}` : `Every ${step} ${unit}s`;
}

export function formatBillScheduleLabel(bill: RecurringScheduleLike): string {
  const step = bill.interval > 1 ? bill.interval : 1;
  const cadence = formatCadencePrefix(bill.frequency, step);
  const s = bill.schedule;
  if (!s) {
    return bill.frequency === 'bi-monthly'
      ? `${cadence} (1st and 15th)`
      : `${cadence} (from last paid)`;
  }
  if (s.type === 'first_fifteenth') {
    return step === 1 ? '1st and 15th of the month' : `1st and 15th (${cadence.toLowerCase()})`;
  }
  if (s.type === 'fifteenth_last') {
    return step === 1
      ? '15th and last day of the month'
      : `15th and last day (${cadence.toLowerCase()})`;
  }
  if (s.type === 'weekday' && s.weekday) {
    return `${cadence} on ${capitalize(s.weekday)}`;
  }
  if (s.type === 'day_of_period' && s.day) {
    const scopeLabel = s.scope === 'year' ? 'of the year' : 'of the month';
    const dayText = s.scope === 'month' && s.day === 31 ? 'Last day (31st)' : ordinal(s.day);
    return step === 1 ? `${dayText} ${scopeLabel}` : `${dayText} day (${cadence.toLowerCase()})`;
  }
  if (s.type === 'month_day' && s.month && s.day) {
    const monthName = MONTH_NAMES[s.month] || `Month ${s.month}`;
    return step === 1 ? `${monthName} ${ordinal(s.day)} every year` : `${monthName} ${ordinal(s.day)} (${cadence.toLowerCase()})`;
  }
  if (s.type === 'nth_weekday' && s.nth && s.weekday) {
    const wd = capitalize(s.weekday);
    const nthLabel = ordinal(s.nth);
    if (s.scope === 'year' && s.month) {
      const mName = MONTH_NAMES[s.month] || `Month ${s.month}`;
      return `${nthLabel} ${wd} of ${mName} (${cadence.toLowerCase()})`;
    }
    const scopeText = s.scope === 'year' ? 'of the year' : 'of the month';
    return step === 1 ? `${nthLabel} ${wd} ${scopeText}` : `${nthLabel} ${wd} (${cadence.toLowerCase()})`;
  }
  return cadence;
}

export function getBillDisplayName(bill: Pick<RecurringDisplayLike, 'name' | 'entity'>): string {
  const shortName = (bill.name || '').trim();
  return shortName || bill.entity || '';
}

export function getAmountColorClass(amtStr: string): string {
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

export function sortBillsForDisplay<T extends RecurringDisplayLike>(bills: T[]): T[] {
  return [...bills].sort((a, b) => {
    const aDue = a.next_due_date || '9999-12-31';
    const bDue = b.next_due_date || '9999-12-31';
    if (aDue !== bDue) return aDue.localeCompare(bDue);
    return getBillDisplayName(a).localeCompare(getBillDisplayName(b), undefined, {
      sensitivity: 'base',
    });
  });
}
