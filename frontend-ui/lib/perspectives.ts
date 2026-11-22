import type {
  AccountRecord,
  Category,
  PerspectiveTrendPoint,
  RecurringTransaction,
} from './types.ts';
import type { BalanceRange } from './accountTrend.ts';
import { computeCategoryPieData, type CategoryPieSummary } from './accountCategories.ts';

export interface RunwayTrajectoryPoint {
  date: string;
  totalValue: number;
}

export interface RunwayProjectionResult {
  dailyIncome: number;
  dailyBills: number;
  dailyCustomSpend: number;
  netDailyChange: number;
  daysUntilZero: number | null;
  projectedZeroDate: string | null;
  isSustainable: boolean;
  points: RunwayTrajectoryPoint[];
}

export function getRecurringDailyMultiplier(frequency: string): number {
  switch (frequency) {
    case 'weekly':
      return 1 / 7;
    case 'biweekly':
      return 1 / 14;
    case 'bi-monthly':
      return 24 / 365.25;
    case 'monthly':
      return 12 / 365.25;
    case 'yearly':
      return 1 / 365.25;
    default:
      return 0;
  }
}

export function addDaysIso(isoDate: string, days: number): string {
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  dt.setUTCDate(dt.getUTCDate() + days);
  return dt.toISOString().slice(0, 10);
}

export function getTodayIsoDate(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function computeRecurringDailyBreakdown(
  recurringTxs: RecurringTransaction[],
  disabledIds: ReadonlySet<string> = new Set()
): { dailyIncome: number; dailyBills: number; netDailyRecurring: number } {
  let dailyIncome = 0;
  let dailyBills = 0;

  for (const tx of recurringTxs) {
    if (disabledIds.has(tx.id)) continue;
    const rate = Math.abs(tx.amount) * getRecurringDailyMultiplier(tx.frequency);
    if (tx.amount > 0) {
      dailyIncome += rate;
    } else if (tx.amount < 0) {
      dailyBills += rate;
    }
  }

  return {
    dailyIncome,
    dailyBills,
    netDailyRecurring: dailyIncome - dailyBills,
  };
}

export function computeRunwayProjection(
  currentTotalValue: number,
  customDailySpend: number,
  recurringTxs: RecurringTransaction[],
  disabledRecurringIds: ReadonlySet<string> = new Set(),
  startDateIso: string = getTodayIsoDate()
): RunwayProjectionResult {
  const dailyCustomSpend = Math.max(0, Number.isFinite(customDailySpend) ? customDailySpend : 0);
  const { dailyIncome, dailyBills, netDailyRecurring } = computeRecurringDailyBreakdown(
    recurringTxs,
    disabledRecurringIds
  );
  const netDailyChange = netDailyRecurring - dailyCustomSpend;

  if (currentTotalValue <= 0) {
    return {
      dailyIncome,
      dailyBills,
      dailyCustomSpend,
      netDailyChange,
      daysUntilZero: 0,
      projectedZeroDate: startDateIso,
      isSustainable: false,
      points: [{ date: startDateIso, totalValue: currentTotalValue }],
    };
  }

  if (netDailyChange >= -1e-9) {
    const horizonDays = 365;
    const points: RunwayTrajectoryPoint[] = [];
    const steps = 6;
    for (let i = 0; i <= steps; i++) {
      const dayOffset = Math.round((horizonDays * i) / steps);
      points.push({
        date: addDaysIso(startDateIso, dayOffset),
        totalValue: currentTotalValue + netDailyChange * dayOffset,
      });
    }
    return {
      dailyIncome,
      dailyBills,
      dailyCustomSpend,
      netDailyChange,
      daysUntilZero: null,
      projectedZeroDate: null,
      isSustainable: true,
      points,
    };
  }

  const dailyBurn = Math.abs(netDailyChange);
  const daysUntilZero = Math.ceil(currentTotalValue / dailyBurn);
  const projectedZeroDate = addDaysIso(startDateIso, daysUntilZero);
  const steps = Math.min(12, Math.max(2, daysUntilZero));
  const points: RunwayTrajectoryPoint[] = [];

  for (let i = 0; i <= steps; i++) {
    const dayOffset = Math.round((daysUntilZero * i) / steps);
    const projected = Math.max(0, currentTotalValue - dailyBurn * dayOffset);
    points.push({
      date: addDaysIso(startDateIso, dayOffset),
      totalValue: i === steps ? 0 : projected,
    });
  }

  return {
    dailyIncome,
    dailyBills,
    dailyCustomSpend,
    netDailyChange,
    daysUntilZero,
    projectedZeroDate,
    isSustainable: false,
    points,
  };
}

export function filterPerspectiveTrendByRange(
  trend: PerspectiveTrendPoint[],
  range: BalanceRange,
  referenceDateIso: string = getTodayIsoDate()
): PerspectiveTrendPoint[] {
  if (range === 'all' || trend.length === 0) return trend;
  const daysMap: Record<Exclude<BalanceRange, 'all'>, number> = {
    '30d': 30,
    '60d': 60,
    '90d': 90,
    '1y': 365,
  };
  const cutoff = addDaysIso(referenceDateIso, -daysMap[range]);
  const filtered = trend.filter((pt) => pt.date >= cutoff);
  return filtered.length > 0 ? filtered : trend.slice(-1);
}

export function buildPerspectiveCategoryRecords(
  accountRecords: AccountRecord[],
  linkedRecurring: RecurringTransaction[],
  disabledRecurringIds: ReadonlySet<string> = new Set(),
  referenceDateIso: string = getTodayIsoDate()
): AccountRecord[] {
  const syntheticRecurring: AccountRecord[] = linkedRecurring
    .filter((tx) => !disabledRecurringIds.has(tx.id))
    .map((tx) => ({
      id: `recurring-${tx.id}`,
      account_id: tx.account_id || 'perspective',
      category_id: tx.category_id ?? null,
      value: -Math.abs(tx.amount),
      date_sent: referenceDateIso,
      date_cleared: referenceDateIso,
      entity: tx.entity,
      description: tx.description || '',
      created_at: tx.created_at,
      updated_at: tx.updated_at,
    }));

  return [...accountRecords, ...syntheticRecurring];
}

export function computePerspectiveCategoryPie(
  accountRecords: AccountRecord[],
  linkedRecurring: RecurringTransaction[],
  categories: Category[],
  range: BalanceRange,
  includeUncategorized: boolean,
  disabledRecurringIds: ReadonlySet<string> = new Set(),
  referenceDateIso: string = getTodayIsoDate()
): CategoryPieSummary {
  const combined = buildPerspectiveCategoryRecords(
    accountRecords,
    linkedRecurring,
    disabledRecurringIds,
    referenceDateIso
  );
  return computeCategoryPieData(combined, categories, range, includeUncategorized, referenceDateIso);
}

export interface XAxisTickLabel {
  x: number;
  label: string;
  anchor: 'start' | 'middle' | 'end';
}

export function buildNonOverlappingXAxisLabels(
  firstTimeMs: number,
  lastTimeMs: number,
  padLeft: number,
  plotWidth: number,
  formatLabelFn: (isoDate: string) => string,
  minSpacingPx: number = 85
): XAxisTickLabel[] {
  if (!Number.isFinite(firstTimeMs) || !Number.isFinite(lastTimeMs)) {
    return [];
  }
  const firstIso = new Date(firstTimeMs).toISOString().slice(0, 10);
  const lastIso = new Date(lastTimeMs).toISOString().slice(0, 10);
  if (lastTimeMs <= firstTimeMs) {
    return [{ x: padLeft + plotWidth / 2, label: formatLabelFn(firstIso), anchor: 'middle' }];
  }

  const maxSlots = Math.max(2, Math.floor(plotWidth / minSpacingPx) + 1);
  const stepCount = Math.min(6, maxSlots - 1);
  const labels: XAxisTickLabel[] = [
    { x: padLeft, label: formatLabelFn(firstIso), anchor: 'start' },
  ];
  const endX = padLeft + plotWidth;

  for (let i = 1; i < stepCount; i++) {
    const ratio = i / stepCount;
    const x = padLeft + ratio * plotWidth;
    const prevX = labels[labels.length - 1].x;
    if (x - prevX < minSpacingPx || endX - x < minSpacingPx) {
      continue;
    }
    const tickIso = new Date(Math.round(firstTimeMs + ratio * (lastTimeMs - firstTimeMs)))
      .toISOString()
      .slice(0, 10);
    const formatted = formatLabelFn(tickIso);
    if (formatted !== labels[labels.length - 1].label && formatted !== formatLabelFn(lastIso)) {
      labels.push({ x, label: formatted, anchor: 'middle' });
    }
  }

  if (endX - labels[labels.length - 1].x >= minSpacingPx) {
    labels.push({ x: endX, label: formatLabelFn(lastIso), anchor: 'end' });
  }
  return labels;
}
