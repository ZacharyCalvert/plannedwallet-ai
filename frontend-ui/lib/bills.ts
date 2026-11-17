import type { Bill } from './types.ts';

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
 * Determines whether a reminder banner should appear for a bill on todayIso:
 * - Overdue (today > next_due_date, i.e. daysUntil < 0)
 * - Weekly bill scheduled within the next 48 hours (0..2 days)
 * - Monthly bill scheduled within the next 1 week (0..7 days)
 * - Yearly bill scheduled within the next 30 days (0..30 days)
 */
export function shouldShowBillReminder(bill: Bill, todayIso?: string): boolean {
  if (!bill.reminder || !bill.next_due_date) {
    return false;
  }
  const days = getBillDaysUntilDue(bill.next_due_date, todayIso);
  if (days === null) {
    return false;
  }
  if (days < 0) {
    return true;
  }
  switch (bill.frequency) {
    case 'weekly':
      return days <= 2;
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

export function isValidBillIntervalInput(raw: string): boolean {
  return raw === '' || /^[1-9]\d*$/.test(raw);
}

export function normalizePositiveBillIntervalInput(raw: string): string {
  if (!/^[1-9]\d*$/.test(raw.trim())) {
    return '1';
  }
  return raw.trim();
}

function formatCadencePrefix(frequency: Bill['frequency'], interval: number): string {
  const step = interval > 1 ? interval : 1;
  const unit = frequency === 'weekly' ? 'week' : frequency === 'monthly' ? 'month' : 'year';
  return step === 1 ? `Every ${unit}` : `Every ${step} ${unit}s`;
}

export function formatBillScheduleLabel(bill: Bill): string {
  const step = bill.interval > 1 ? bill.interval : 1;
  const cadence = formatCadencePrefix(bill.frequency, step);
  const s = bill.schedule;
  if (!s) {
    return `${cadence} (from last paid)`;
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

export function getBillDisplayName(bill: Pick<Bill, 'name' | 'recipient'>): string {
  const shortName = (bill.name || '').trim();
  return shortName || bill.recipient;
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

export function sortBillsForDisplay(bills: Bill[]): Bill[] {
  return [...bills].sort((a, b) => {
    const aDue = a.next_due_date || '9999-12-31';
    const bDue = b.next_due_date || '9999-12-31';
    if (aDue !== bDue) return aDue.localeCompare(bDue);
    return getBillDisplayName(a).localeCompare(getBillDisplayName(b), undefined, {
      sensitivity: 'base',
    });
  });
}
