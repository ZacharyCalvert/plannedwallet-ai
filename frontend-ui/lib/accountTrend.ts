import type { AccountRecord } from './types.ts';

export type BalanceRange = '30d' | '60d' | '90d' | '1y' | 'all';
export type RollupPeriod = 'daily' | 'weekly' | 'monthly' | 'yearly';

export interface BalanceTrendPoint {
  date: string;
  balance: number;
  delta: number;
  transactionCount: number;
  sampleCount?: number;
  periodLabel?: string;
}

export interface RolledUpValuePoint {
  date: string;
  value: number;
  sampleCount: number;
  periodLabel: string;
}

export interface BalanceMetrics {
  first: BalanceTrendPoint;
  latest: BalanceTrendPoint;
  change: number;
  pctChange: number;
  minVal: number;
  maxVal: number;
}

export interface YAxisScale {
  yMin: number;
  yMax: number;
  step: number;
  ticks: number[];
}

function compareRecordsChronologically(a: AccountRecord, b: AccountRecord): number {
  if (a.date_sent !== b.date_sent) {
    return a.date_sent.localeCompare(b.date_sent);
  }
  const aCreated = a.created_at || '';
  const bCreated = b.created_at || '';
  if (aCreated !== bCreated) {
    return aCreated.localeCompare(bCreated);
  }
  return a.id.localeCompare(b.id);
}

function getLocalTodayIso(): string {
  const d = new Date();
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function subtractDaysIso(isoDate: string, days: number): string {
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  dt.setUTCDate(dt.getUTCDate() - days);
  return dt.toISOString().slice(0, 10);
}

function rangeToDays(range: Exclude<BalanceRange, 'all'>): number {
  switch (range) {
    case '30d':
      return 30;
    case '60d':
      return 60;
    case '90d':
      return 90;
    case '1y':
      return 365;
  }
}

export function buildAccountBalanceSeries(records: AccountRecord[]): BalanceTrendPoint[] {
  if (records.length === 0) return [];
  const sorted = [...records].sort(compareRecordsChronologically);
  const points: BalanceTrendPoint[] = [];
  let running = 0;

  for (const rec of sorted) {
    running += rec.value;
    const prev = points[points.length - 1];
    if (prev && prev.date === rec.date_sent) {
      prev.balance = running;
      prev.delta += rec.value;
      prev.transactionCount += 1;
    } else {
      points.push({
        date: rec.date_sent,
        balance: running,
        delta: rec.value,
        transactionCount: 1,
      });
    }
  }
  return points;
}

export function filterBalanceSeriesByRange(
  series: BalanceTrendPoint[],
  range: BalanceRange,
  referenceDateStr?: string
): BalanceTrendPoint[] {
  if (range === 'all' || series.length === 0) return series;

  const todayStr = referenceDateStr || getLocalTodayIso();
  const latestRecordDate = series[series.length - 1].date;
  const anchorStr = latestRecordDate > todayStr ? latestRecordDate : todayStr;
  const cutoffStr = subtractDaysIso(anchorStr, rangeToDays(range));

  let priorPoint: BalanceTrendPoint | null = null;
  const inWindow: BalanceTrendPoint[] = [];

  for (const pt of series) {
    if (pt.date < cutoffStr) {
      priorPoint = pt;
    } else {
      inWindow.push(pt);
    }
  }

  if (priorPoint && (inWindow.length === 0 || inWindow[0].date > cutoffStr)) {
    inWindow.unshift({
      date: cutoffStr,
      balance: priorPoint.balance,
      delta: 0,
      transactionCount: 0,
    });
  }
  if (priorPoint && inWindow.length === 1 && inWindow[0].date < anchorStr) {
    inWindow.push({
      date: anchorStr,
      balance: priorPoint.balance,
      delta: 0,
      transactionCount: 0,
    });
  }
  return inWindow;
}

export function computeBalanceMetrics(points: BalanceTrendPoint[]): BalanceMetrics | null {
  if (points.length === 0) return null;
  const first = points[0];
  const latest = points[points.length - 1];
  const change = latest.balance - first.balance;
  const pctChange = first.balance !== 0 ? (change / Math.abs(first.balance)) * 100 : 0;

  let minVal = points[0].balance;
  let maxVal = points[0].balance;
  for (const pt of points) {
    if (pt.balance < minVal) minVal = pt.balance;
    if (pt.balance > maxVal) maxVal = pt.balance;
  }
  return { first, latest, change, pctChange, minVal, maxVal };
}

function getMinMaxIndices<T>(points: T[], getValue: (pt: T) => number): number[] {
  let minIdx = 0;
  let maxIdx = 0;
  let minVal = getValue(points[0]);
  let maxVal = minVal;

  for (let i = 1; i < points.length; i++) {
    const val = getValue(points[i]);
    if (val < minVal) {
      minVal = val;
      minIdx = i;
    }
    if (val > maxVal) {
      maxVal = val;
      maxIdx = i;
    }
  }
  return minIdx === maxIdx ? [minIdx] : [minIdx, maxIdx];
}

function isFarEnoughFromAll<T extends { x: number }>(
  points: T[],
  candidateIdx: number,
  keptIndices: number[],
  minGapPx: number
): boolean {
  const cx = points[candidateIdx].x;
  for (const idx of keptIndices) {
    if (Math.abs(cx - points[idx].x) < minGapPx) {
      return false;
    }
  }
  return true;
}

export function selectHighlightPoints<T extends { x: number }>(
  points: T[],
  minGapPx = 10,
  getValue?: (pt: T) => number
): T[] {
  if (points.length <= 1 || minGapPx <= 0) return points;

  const keptIndices: number[] = getValue ? getMinMaxIndices(points, getValue) : [];
  const tryAdd = (idx: number) => {
    if (!keptIndices.includes(idx) && isFarEnoughFromAll(points, idx, keptIndices, minGapPx)) {
      keptIndices.push(idx);
    }
  };

  tryAdd(0);
  tryAdd(points.length - 1);
  for (let i = 1; i < points.length - 1; i++) {
    tryAdd(i);
  }

  keptIndices.sort((a, b) => a - b);
  return keptIndices.map((idx) => points[idx]);
}

function pickNiceDollarStep(span: number, targetIntervals = 5): number {
  const safeSpan = span > 0 ? span : 100;
  const rawStep = safeSpan / targetIntervals;
  const exp = Math.floor(Math.log10(rawStep));
  const mag = Math.pow(10, exp);
  const norm = rawStep / mag;
  const candidates = exp >= 1 ? [1, 2, 2.5, 5, 10] : [1, 2, 5, 10];
  let chosen = 10;
  for (const c of candidates) {
    if (c >= norm) {
      chosen = c;
      break;
    }
  }
  return Math.max(1, Math.round(chosen * mag));
}

export function computeYAxisScale(minVal: number, maxVal: number): YAxisScale {
  let lo = minVal;
  let hi = maxVal;
  if (lo === hi) {
    const pad = Math.max(Math.abs(lo) * 0.1, 50);
    lo = lo >= 0 ? Math.max(0, lo - pad) : lo - pad;
    hi = hi <= 0 ? Math.min(0, hi + pad) : hi + pad;
    if (lo === hi) hi = lo + 100;
  }

  const step = pickNiceDollarStep(hi - lo);
  let minIdx = Math.floor(lo / step);
  let maxIdx = Math.ceil(hi / step);

  if (minVal >= 0) minIdx = Math.max(0, minIdx);
  if (maxVal <= 0) maxIdx = Math.min(0, maxIdx);
  if (minIdx === maxIdx) maxIdx = minIdx + 1;

  const ticks: number[] = [];
  for (let i = minIdx; i <= maxIdx; i++) {
    const val = i * step;
    ticks.push(val === 0 ? 0 : val);
  }
  return {
    yMin: minIdx * step || 0,
    yMax: maxIdx * step || 0,
    step,
    ticks,
  };
}

function formatUtcShortDate(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  return dt.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function formatUtcMonthYear(ym: string): string {
  const [y, m] = ym.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, 1));
  return dt.toLocaleDateString('en-US', {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function getWeekMondayIso(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  const offset = (dt.getUTCDay() + 6) % 7;
  dt.setUTCDate(dt.getUTCDate() - offset);
  return dt.toISOString().slice(0, 10);
}

function getBucketInfo(
  isoDate: string,
  rollup: Exclude<RollupPeriod, 'daily'>
): { key: string; date: string; label: string } {
  if (rollup === 'weekly') {
    const monday = getWeekMondayIso(isoDate);
    return { key: monday, date: monday, label: `Week of ${formatUtcShortDate(monday)}` };
  }
  if (rollup === 'monthly') {
    const ym = isoDate.slice(0, 7);
    return { key: ym, date: `${ym}-01`, label: formatUtcMonthYear(ym) };
  }
  const yr = isoDate.slice(0, 4);
  return { key: yr, date: `${yr}-01-01`, label: yr };
}

export function rollupValuesByPeriod<T>(
  points: T[],
  rollup: RollupPeriod,
  getDate: (pt: T) => string,
  getValue: (pt: T) => number
): RolledUpValuePoint[] {
  if (points.length === 0) return [];
  if (rollup === 'daily') {
    return points.map((pt) => {
      const d = getDate(pt);
      return {
        date: d,
        value: getValue(pt),
        sampleCount: 1,
        periodLabel: formatUtcShortDate(d),
      };
    });
  }

  const buckets = new Map<
    string,
    { date: string; label: string; sum: number; count: number }
  >();
  for (const pt of points) {
    const bucket = getBucketInfo(getDate(pt), rollup);
    const cur = buckets.get(bucket.key) || {
      date: bucket.date,
      label: bucket.label,
      sum: 0,
      count: 0,
    };
    cur.sum += getValue(pt);
    cur.count += 1;
    buckets.set(bucket.key, cur);
  }

  return Array.from(buckets.values())
    .sort((a, b) => a.date.localeCompare(b.date))
    .map((b) => ({
      date: b.date,
      value: b.sum / b.count,
      sampleCount: b.count,
      periodLabel: b.label,
    }));
}

export function rollupAccountBalanceSeries(
  points: BalanceTrendPoint[],
  rollup: RollupPeriod
): BalanceTrendPoint[] {
  if (rollup === 'daily') return points;
  return rollupValuesByPeriod(
    points,
    rollup,
    (p) => p.date,
    (p) => p.balance
  ).map((r) => ({
    date: r.date,
    balance: r.value,
    delta: 0,
    transactionCount: 0,
    sampleCount: r.sampleCount,
    periodLabel: r.periodLabel,
  }));
}

export function formatRollupXAxisLabel(isoDate: string, rollup: RollupPeriod): string {
  if (rollup === 'monthly') return isoDate.slice(0, 7);
  if (rollup === 'yearly') return isoDate.slice(0, 4);
  return isoDate.slice(5);
}
