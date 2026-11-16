import type { AccountRecord } from './types.ts';

export type BalanceRange = '30d' | '60d' | '90d' | '1y' | 'all';

export interface BalanceTrendPoint {
  date: string;
  balance: number;
  delta: number;
  transactionCount: number;
}

export interface BalanceMetrics {
  first: BalanceTrendPoint;
  latest: BalanceTrendPoint;
  change: number;
  pctChange: number;
  minVal: number;
  maxVal: number;
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

export function selectHighlightPoints<T extends { x: number }>(
  points: T[],
  minGapPx = 10
): T[] {
  if (points.length <= 1 || minGapPx <= 0) return points;
  const kept: T[] = [points[0]];

  for (let i = 1; i < points.length; i++) {
    const pt = points[i];
    const prev = kept[kept.length - 1];
    if (pt.x - prev.x >= minGapPx) {
      kept.push(pt);
    }
  }

  const last = points[points.length - 1];
  if (kept[kept.length - 1] !== last && kept.length >= 2) {
    if (last.x - kept[kept.length - 2].x >= minGapPx) {
      kept[kept.length - 1] = last;
    }
  }
  return kept;
}
