'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  Account,
  AccountRecord,
  Category,
  Perspective,
  PerspectiveView,
  RecurringTransaction,
} from '../lib/types';
import {
  deletePerspective,
  fetchAccountRecords,
  fetchPerspectiveView,
  linkPerspectiveAccount,
  linkPerspectiveRecurringTransaction,
  unlinkPerspectiveAccount,
  unlinkPerspectiveRecurringTransaction,
  updatePerspective,
  updatePerspectiveSettings,
} from '../lib/api';
import {
  BalanceRange,
  RollupPeriod,
  computeBalanceMetrics,
  computeYAxisScale,
  formatRollupXAxisLabel,
  rollupValuesByPeriod,
  selectHighlightPoints,
} from '../lib/accountTrend';
import {
  buildNonOverlappingXAxisLabels,
  buildPerspectiveCategoryRecords,
  computeRunwayProjection,
  filterPerspectiveTrendByRange,
} from '../lib/perspectives';
import { AccountCategoryPieChart } from './AccountCategoryPieChart';

interface PerspectiveDetailProps {
  perspectiveId: string;
  token: string;
  allAccounts: Account[];
  allRecurringTransactions: RecurringTransaction[];
  categories: Category[];
  onPerspectiveUpdated: (updated: Perspective) => void;
  onPerspectiveDeleted: (deletedId: string) => void;
  showPinToggle?: boolean;
  isPinned?: boolean;
  onTogglePin?: () => void;
}

type PerspectiveTab = 'trend' | 'categories' | 'settings';

const RANGE_OPTIONS: { value: BalanceRange; label: string }[] = [
  { value: 'all', label: 'All Time' },
  { value: '1y', label: '1 Year' },
  { value: '90d', label: '90 Days' },
  { value: '60d', label: '60 Days' },
  { value: '30d', label: '30 Days' },
];

const ROLLUP_OPTIONS: { value: RollupPeriod; label: string }[] = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
  { value: 'yearly', label: 'Yearly' },
];

const SVG_WIDTH = 840;
const SVG_HEIGHT = 340;
const PAD_LEFT = 85;
const PAD_RIGHT = 45;
const PAD_TOP = 30;
const PAD_BOTTOM = 45;
const PLOT_WIDTH = SVG_WIDTH - PAD_LEFT - PAD_RIGHT;
const PLOT_HEIGHT = SVG_HEIGHT - PAD_TOP - PAD_BOTTOM;

function formatCurrency(val: number): string {
  const isNeg = val < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
  return isNeg ? `-${formatted}` : formatted;
}

function formatYAxisCurrency(val: number): string {
  const clean = val === 0 ? 0 : val;
  const isNeg = clean < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(Math.abs(clean));
  return isNeg ? `-${formatted}` : formatted;
}

export function PerspectiveDetail({
  perspectiveId,
  token,
  allAccounts,
  allRecurringTransactions,
  categories,
  onPerspectiveUpdated,
  onPerspectiveDeleted,
  showPinToggle = false,
  isPinned = false,
  onTogglePin,
}: PerspectiveDetailProps) {
  const [view, setView] = useState<PerspectiveView | null>(null);
  const [accountRecords, setAccountRecords] = useState<AccountRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [actionError, setActionError] = useState('');
  const [activeTab, setActiveTab] = useState<PerspectiveTab>('trend');
  const [rangeFilter, setRangeFilter] = useState<BalanceRange>('all');
  const [rollupPeriod, setRollupPeriod] = useState<RollupPeriod>('daily');
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [disabledRecurringIds, setDisabledRecurringIds] = useState<Set<string>>(() => new Set());
  const [deleting, setDeleting] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  // Settings / edit state
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [showCategories, setShowCategories] = useState(false);
  const [trackRuinedDate, setTrackRuinedDate] = useState(false);
  const [dailySpendStr, setDailySpendStr] = useState('');
  const [savingSettings, setSavingSettings] = useState(false);
  const [settingsMsg, setSettingsMsg] = useState('');

  const loadView = async (silent = false) => {
    if (!silent) {
      setLoading(true);
      setError('');
    }
    try {
      const data = await fetchPerspectiveView(token, perspectiveId);
      setView(data);
      setEditName(data.perspective.name);
      setEditDesc(data.perspective.description);
      setShowCategories(data.settings.show_categories);
      setTrackRuinedDate(data.settings.track_ruined_date);
      setDailySpendStr(
        data.settings.custom_daily_spend !== undefined && data.settings.custom_daily_spend !== null
          ? String(data.settings.custom_daily_spend)
          : ''
      );

      const recordTypes = new Set(['checking', 'savings', 'credit card', 'credit_card']);
      const recordAccounts = (data.accounts || []).filter((a) =>
        recordTypes.has((a.type || '').toLowerCase())
      );
      const recLists = await Promise.all(
        recordAccounts.map((a) => fetchAccountRecords(token, a.id).catch(() => []))
      );
      setAccountRecords(recLists.flat());
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load perspective');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setDisabledRecurringIds(new Set());
    loadView();
  }, [perspectiveId, token]);

  const linkedAccountIds = useMemo(
    () => new Set((view?.accounts || []).map((a) => a.id)),
    [view]
  );

  const linkedRecurring = useMemo(
    () => view?.recurring_transactions || [],
    [view]
  );

  const linkedRecurringIds = useMemo(
    () => new Set(linkedRecurring.map((r) => r.id)),
    [linkedRecurring]
  );

  const filteredTrend = useMemo(
    () => filterPerspectiveTrendByRange(view?.trend || [], rangeFilter),
    [view?.trend, rangeFilter]
  );

  const rolledUpTrend = useMemo(() => {
    const rolled = rollupValuesByPeriod(
      filteredTrend,
      rollupPeriod,
      (p) => p.date,
      (p) => p.total_value
    );
    return rolled.map((r) => ({
      date: r.date,
      balance: r.value,
      delta: 0,
      transactionCount: r.sampleCount,
      periodLabel: r.periodLabel,
    }));
  }, [filteredTrend, rollupPeriod]);

  const trendMetrics = useMemo(
    () => computeBalanceMetrics(rolledUpTrend),
    [rolledUpTrend]
  );

  const runwayProjection = useMemo(() => {
    if (!view || !view.settings.track_ruined_date) return null;
    const customSpend = view.settings.custom_daily_spend ?? 0;
    return computeRunwayProjection(
      view.current_total_value,
      customSpend,
      linkedRecurring,
      disabledRecurringIds
    );
  }, [view, linkedRecurring, disabledRecurringIds]);

  const combinedCategoryRecords = useMemo(
    () => buildPerspectiveCategoryRecords(accountRecords, linkedRecurring, disabledRecurringIds),
    [accountRecords, linkedRecurring, disabledRecurringIds]
  );

  if (loading) {
    return <div className="card empty-prompt"><p>Loading perspective...</p></div>;
  }

  if (error || !view) {
    return (
      <div className="card empty-prompt">
        <p className="text-danger">{error || 'Perspective not found.'}</p>
      </div>
    );
  }

  const toggleWhatIfRecurring = (id: string) => {
    setDisabledRecurringIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleToggleAccountLink = async (accId: string) => {
    setActionError('');
    try {
      if (linkedAccountIds.has(accId)) {
        await unlinkPerspectiveAccount(token, perspectiveId, accId);
      } else {
        await linkPerspectiveAccount(token, perspectiveId, accId);
      }
      await loadView(true);
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to update account link');
    }
  };

  const handleToggleRecurringLink = async (recId: string) => {
    setActionError('');
    try {
      if (linkedRecurringIds.has(recId)) {
        await unlinkPerspectiveRecurringTransaction(token, perspectiveId, recId);
      } else {
        await linkPerspectiveRecurringTransaction(token, perspectiveId, recId);
      }
      await loadView(true);
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Failed to update recurring link');
    }
  };

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSettingsMsg('');
    const trimmedName = editName.trim();
    if (!trimmedName) {
      setSettingsMsg('Perspective name is required.');
      return;
    }
    let parsedSpend: number | null = null;
    if (dailySpendStr.trim()) {
      const n = parseFloat(dailySpendStr.replace(/[$,\s]/g, ''));
      if (!Number.isFinite(n) || n < 0) {
        setSettingsMsg('Custom daily spend must be a valid non-negative number.');
        return;
      }
      parsedSpend = n;
    }

    setSavingSettings(true);
    try {
      const updated = await updatePerspective(token, perspectiveId, {
        name: trimmedName,
        description: editDesc.trim(),
      });
      await updatePerspectiveSettings(token, perspectiveId, {
        show_financial_trend: true,
        show_categories: showCategories,
        track_ruined_date: trackRuinedDate,
        custom_daily_spend: parsedSpend,
      });
      onPerspectiveUpdated(updated);
      setSettingsMsg('Perspective settings saved.');
      await loadView(true);
    } catch (err: unknown) {
      setSettingsMsg(err instanceof Error ? err.message : 'Failed to save settings.');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleConfirmDelete = async () => {
    setDeleting(true);
    try {
      await deletePerspective(token, perspectiveId);
      onPerspectiveDeleted(perspectiveId);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete perspective');
      setDeleting(false);
    }
  };

  const renderTrendChart = () => {
    const hist = rolledUpTrend;
    const proj = runwayProjection?.points || [];
    const allVals = [
      ...hist.map((p) => p.balance),
      ...proj.map((p) => p.totalValue),
    ];
    if (allVals.length === 0) {
      return (
        <div className="graph-empty-card">
          <h4>No Trend Data Yet</h4>
          <p>Link accounts with records, snapshots, or asset valuations to view the aggregate trend.</p>
        </div>
      );
    }

    let minVal = allVals[0];
    let maxVal = allVals[0];
    for (const v of allVals) {
      if (v < minVal) minVal = v;
      if (v > maxVal) maxVal = v;
    }

    const { yMin, yMax, ticks } = computeYAxisScale(minVal, maxVal);
    const yRange = yMax - yMin || 1;

    const allDates = [
      ...hist.map((p) => new Date(p.date).getTime()),
      ...proj.map((p) => new Date(p.date).getTime()),
    ];
    const firstTime = Math.min(...allDates);
    const lastTime = Math.max(...allDates);
    const timeSpan = lastTime - firstTime;

    const xForTime = (isoDate: string, fallbackIdx: number, totalCount: number) => {
      if (totalCount <= 1 && timeSpan <= 0) return PAD_LEFT + PLOT_WIDTH / 2;
      if (timeSpan > 0) {
        return PAD_LEFT + ((new Date(isoDate).getTime() - firstTime) / timeSpan) * PLOT_WIDTH;
      }
      return PAD_LEFT + (fallbackIdx / Math.max(1, totalCount - 1)) * PLOT_WIDTH;
    };
    const yForValue = (val: number) =>
      PAD_TOP + PLOT_HEIGHT - ((val - yMin) / yRange) * PLOT_HEIGHT;

    const plottedHist = hist.map((pt, idx) => ({
      x: xForTime(pt.date, idx, hist.length),
      y: yForValue(pt.balance),
      point: pt,
      index: idx,
    }));

    const linePath = plottedHist.reduce(
      (acc, pt, i) => `${acc} ${i === 0 ? 'M' : 'L'} ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`,
      ''
    );

    const clampedZeroNorm = Math.min(1, Math.max(0, (0 - yMin) / yRange));
    const zeroY = PAD_TOP + PLOT_HEIGHT - clampedZeroNorm * PLOT_HEIGHT;
    const areaPath =
      plottedHist.length > 0
        ? `${linePath} L ${plottedHist[plottedHist.length - 1].x.toFixed(1)} ${zeroY.toFixed(1)} L ${plottedHist[0].x.toFixed(1)} ${zeroY.toFixed(1)} Z`
        : '';

    const projPath =
      proj.length > 1
        ? proj
            .map(
              (pt, i) =>
                `${i === 0 ? 'M' : 'L'} ${xForTime(pt.date, i, proj.length).toFixed(1)} ${yForValue(pt.totalValue).toFixed(1)}`
            )
            .join(' ')
        : '';

    const crossesZero = yMin < 0 && yMax > 0;
    const gridLines = ticks.map((val) => ({
      y: yForValue(val),
      val,
      isZeroCross: crossesZero && val === 0,
    }));

    const highlightPoints = selectHighlightPoints(plottedHist, 10);
    const activePoint = hoveredIndex !== null ? plottedHist[hoveredIndex] ?? null : null;

    const xLabels = buildNonOverlappingXAxisLabels(
      firstTime,
      lastTime,
      PAD_LEFT,
      PLOT_WIDTH,
      (iso) => formatRollupXAxisLabel(iso, rollupPeriod),
      90
    );

    return (
      <div className="graph-canvas-wrapper">
        {activePoint && (
          <div
            className={`graph-floating-tooltip ${activePoint.y < 95 ? 'tooltip-below' : ''}`}
            style={{
              left: `${(activePoint.x / SVG_WIDTH) * 100}%`,
              top: `${(activePoint.y / SVG_HEIGHT) * 100}%`,
            }}
          >
            <div className="tooltip-date">
              {activePoint.point.periodLabel || activePoint.point.date}
            </div>
            <div className="tooltip-value">{formatCurrency(activePoint.point.balance)}</div>
          </div>
        )}

        <svg
          className="trend-svg"
          viewBox={`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`}
          preserveAspectRatio="none"
          role="img"
          aria-label="Perspective aggregate value and runway chart"
        >
          <defs>
            <linearGradient id={`persp-grad-${perspectiveId}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#2563eb" stopOpacity="0.24" />
              <stop offset="100%" stopColor="#2563eb" stopOpacity="0.02" />
            </linearGradient>
          </defs>

          {gridLines.map((gl, i) => (
            <g key={i}>
              <line
                x1={PAD_LEFT}
                y1={gl.y}
                x2={SVG_WIDTH - PAD_RIGHT}
                y2={gl.y}
                stroke={gl.isZeroCross ? '#94a3b8' : '#e2e8f0'}
                strokeWidth={gl.isZeroCross ? '1.5' : '1'}
                strokeDasharray={gl.isZeroCross ? undefined : '4 4'}
              />
              <text
                x={PAD_LEFT - 10}
                y={gl.y + 4}
                textAnchor="end"
                fontSize="11"
                fontWeight={gl.isZeroCross ? '600' : undefined}
                fill={gl.isZeroCross ? '#334155' : '#64748b'}
                fontFamily="inherit"
              >
                {formatYAxisCurrency(gl.val)}
              </text>
            </g>
          ))}

          {areaPath && <path d={areaPath} fill={`url(#persp-grad-${perspectiveId})`} />}
          {linePath && (
            <path
              d={linePath}
              fill="none"
              stroke="#2563eb"
              strokeWidth="3"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          )}

          {projPath && (
            <path
              d={projPath}
              fill="none"
              stroke={runwayProjection?.isSustainable ? '#10b981' : '#dc2626'}
              strokeWidth="2.5"
              strokeDasharray="6 5"
            />
          )}

          {highlightPoints.map((pt) => {
            const isHovered = hoveredIndex === pt.index;
            return (
              <g
                key={`${pt.point.date}-${pt.index}`}
                className="graph-point-group"
                onMouseEnter={() => setHoveredIndex(pt.index)}
                onMouseLeave={() => setHoveredIndex(null)}
              >
                <circle cx={pt.x} cy={pt.y} r="10" fill="transparent" />
                <circle
                  cx={pt.x}
                  cy={pt.y}
                  r={isHovered ? 6 : 3.5}
                  fill={isHovered ? '#ffffff' : '#2563eb'}
                  stroke="#2563eb"
                  strokeWidth={isHovered ? 3 : 1.5}
                />
              </g>
            );
          })}

          {xLabels.map((lbl, idx) => (
            <text
              key={`${lbl.label}-${idx}`}
              x={lbl.x}
              y={SVG_HEIGHT - 12}
              textAnchor={lbl.anchor}
              fontSize="11"
              fill="#64748b"
              fontFamily="inherit"
            >
              {lbl.label}
            </text>
          ))}
        </svg>
      </div>
    );
  };

  return (
    <div className="account-detail-container">
      <div className="account-header-card">
        <div className="account-header-top">
          <div>
            <div className="account-title-row">
              <h2>{view.perspective.name}</h2>
              <span className="badge-account-type badge-perspective">Perspective</span>
            </div>
            {view.perspective.description && (
              <p className="account-desc">{view.perspective.description}</p>
            )}
          </div>
          <div className="perspective-header-metrics">
            <div className="perspective-metric-box">
              <span className="text-muted text-sm">Aggregate Value</span>
              <strong className={view.current_total_value < 0 ? 'text-danger' : 'text-success'}>
                {formatCurrency(view.current_total_value)}
              </strong>
            </div>
            <button
              type="button"
              className="btn-danger-outline btn-sm"
              onClick={() => setConfirmDeleteOpen(true)}
            >
              Delete
            </button>
            {showPinToggle && onTogglePin && (
              <label className="pin-balance-toggle" title="Pin expected balance to top bar">
                <input
                  type="checkbox"
                  checked={isPinned}
                  onChange={onTogglePin}
                />
                <span>Pin to top</span>
              </label>
            )}
          </div>
        </div>
      </div>

      {actionError && (
        <div className="alert-error mb-3" role="alert">
          {actionError}
        </div>
      )}

      <div className="snapshot-tabs-bar">
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'trend' ? 'active' : ''}`}
          onClick={() => setActiveTab('trend')}
        >
          Trend &amp; Runway
        </button>
        {view.settings.show_categories && (
          <button
            type="button"
            className={`snapshot-tab-button ${activeTab === 'categories' ? 'active' : ''}`}
            onClick={() => setActiveTab('categories')}
          >
            Categories
          </button>
        )}
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </button>
      </div>

      {activeTab === 'trend' && (
        <div className="trend-graph-section">
          <div className="graph-controls-row">
            <div>
              <h3 className="graph-title">Aggregate Value Trend</h3>
              <p className="graph-subtitle">
                Combined balance across {view.accounts.length}{' '}
                {view.accounts.length === 1 ? 'account' : 'accounts'}
              </p>
            </div>
            <div className="graph-controls-right">
              <div className="graph-range-pills" role="group" aria-label="Perspective rollup granularity">
                {ROLLUP_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    type="button"
                    className={`pill-btn ${rollupPeriod === opt.value ? 'active' : ''}`}
                    onClick={() => {
                      setHoveredIndex(null);
                      setRollupPeriod(opt.value);
                    }}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
              <div className="graph-range-pills" role="group" aria-label="Perspective timeframe filter">
                {RANGE_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    type="button"
                    className={`pill-btn ${rangeFilter === opt.value ? 'active' : ''}`}
                    onClick={() => {
                      setHoveredIndex(null);
                      setRangeFilter(opt.value);
                    }}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {trendMetrics && (
            <div className="graph-metrics-grid">
              <div className="metric-tile">
                <span className="metric-label">Starting Value</span>
                <span className="metric-val">{formatCurrency(trendMetrics.first.balance)}</span>
                <span className="metric-sub">{trendMetrics.first.date}</span>
              </div>
              <div className="metric-tile">
                <span className="metric-label">Ending Value</span>
                <span className="metric-val">{formatCurrency(trendMetrics.latest.balance)}</span>
                <span className="metric-sub">{trendMetrics.latest.date}</span>
              </div>
              <div className="metric-tile">
                <span className="metric-label">Net Change</span>
                <span className={`metric-val ${trendMetrics.change >= 0 ? 'text-positive' : 'text-negative'}`}>
                  {trendMetrics.change >= 0 ? '+' : ''}
                  {formatCurrency(trendMetrics.change)}
                </span>
                <span className="metric-sub">
                  {trendMetrics.pctChange >= 0 ? '+' : ''}
                  {trendMetrics.pctChange.toFixed(2)}% over period
                </span>
              </div>
              <div className="metric-tile">
                <span className="metric-label">Value Range (Min - Max)</span>
                <span className="metric-val">
                  {formatCurrency(trendMetrics.minVal)} - {formatCurrency(trendMetrics.maxVal)}
                </span>
                <span className="metric-sub">
                  {rolledUpTrend.length} {rollupPeriod === 'daily' ? 'daily points' : `${rollupPeriod} buckets`}
                </span>
              </div>
            </div>
          )}

          {renderTrendChart()}

          {view.settings.track_ruined_date && runwayProjection && (
            <div className="perspective-runway-card mt-4">
              <div className="perspective-runway-summary">
                <div>
                  <h4>Zero-Balance Runway Trajectory</h4>
                  <p className="text-muted text-sm">
                    Combines custom daily spend ({formatCurrency(runwayProjection.dailyCustomSpend)}/day), linked bills (-{formatCurrency(runwayProjection.dailyBills)}/day), and linked income (+{formatCurrency(runwayProjection.dailyIncome)}/day).
                  </p>
                </div>
                <div className="perspective-runway-badge">
                  {runwayProjection.isSustainable ? (
                    <span className="text-success font-semibold">
                      Sustainable (+{formatCurrency(runwayProjection.netDailyChange)}/day)
                    </span>
                  ) : (
                    <span className="text-danger font-semibold">
                      Projected $0 Runway Date: {runwayProjection.projectedZeroDate} ({runwayProjection.daysUntilZero} days)
                    </span>
                  )}
                </div>
              </div>

              {linkedRecurring.length > 0 && (
                <div className="perspective-whatif-box mt-3">
                  <div className="text-sm font-semibold mb-2">
                    What-If Trajectory Controls (Toggle scheduled transactions in this projection):
                  </div>
                  <div className="perspective-checkbox-grid">
                    {linkedRecurring.map((tx) => {
                      const enabled = !disabledRecurringIds.has(tx.id);
                      return (
                        <label key={tx.id} className="perspective-checkbox-item">
                          <input
                            type="checkbox"
                            checked={enabled}
                            onChange={() => toggleWhatIfRecurring(tx.id)}
                          />
                          <span>{tx.entity}</span>
                          <span className={tx.amount < 0 ? 'text-danger text-sm' : 'text-success text-sm'}>
                            {tx.amount < 0 ? '-' : '+'}{formatCurrency(Math.abs(tx.amount))} ({tx.frequency})
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {activeTab === 'categories' && view.settings.show_categories && (
        <AccountCategoryPieChart records={combinedCategoryRecords} categories={categories} />
      )}

      {activeTab === 'settings' && (
        <div className="categories-tab-section">
          <section className="new-record-section">
            <h3>Perspective Settings</h3>
            {settingsMsg && <div className="alert-success mb-3">{settingsMsg}</div>}
            <form onSubmit={handleSaveSettings}>
              <div className="form-group">
                <label htmlFor="edit-persp-name">Perspective Name</label>
                <input
                  id="edit-persp-name"
                  type="text"
                  value={editName}
                  maxLength={100}
                  onChange={(e) => setEditName(e.target.value)}
                  required
                />
              </div>
              <div className="form-group">
                <label htmlFor="edit-persp-desc">Description</label>
                <input
                  id="edit-persp-desc"
                  type="text"
                  value={editDesc}
                  maxLength={255}
                  onChange={(e) => setEditDesc(e.target.value)}
                />
              </div>
              <div className="perspective-settings-checks">
                <label className="perspective-checkbox-item">
                  <input
                    type="checkbox"
                    checked={showCategories}
                    onChange={(e) => setShowCategories(e.target.checked)}
                  />
                  <span>Show Categories Pie Chart</span>
                </label>
                <label className="perspective-checkbox-item">
                  <input
                    type="checkbox"
                    checked={trackRuinedDate}
                    onChange={(e) => setTrackRuinedDate(e.target.checked)}
                  />
                  <span>Track Projected $0 Runway Date</span>
                </label>
              </div>
              {trackRuinedDate && (
                <div className="form-group mt-3">
                  <label htmlFor="edit-persp-spend">Custom Daily Spend ($/day)</label>
                  <input
                    id="edit-persp-spend"
                    type="text"
                    value={dailySpendStr}
                    onChange={(e) => setDailySpendStr(e.target.value)}
                    placeholder="e.g. 75.00"
                  />
                </div>
              )}
              <div className="form-actions mt-4">
                <button type="submit" className="btn-primary" disabled={savingSettings}>
                  {savingSettings ? 'Saving...' : 'Save Settings'}
                </button>
              </div>
            </form>
          </section>

          <div className="categories-lists-grid">
            <section className="categories-group-card">
              <div className="section-title-row">
                <h3>Linked Accounts</h3>
                <span className="records-count-badge">{view.accounts.length}</span>
              </div>
              <div className="perspective-checkbox-grid">
                {allAccounts.map((acc) => {
                  const typeSlug = (acc.type || '').toLowerCase().replace(/\s+/g, '-');
                  return (
                    <label
                      key={acc.id}
                      className={`perspective-checkbox-item account-card-${typeSlug}`}
                    >
                      <input
                        type="checkbox"
                        checked={linkedAccountIds.has(acc.id)}
                        onChange={() => handleToggleAccountLink(acc.id)}
                      />
                      <span className="perspective-checkbox-name">{acc.name}</span>
                      <span className={`badge-account-type badge-${typeSlug}`}>{acc.type}</span>
                    </label>
                  );
                })}
              </div>
            </section>

            <section className="categories-group-card">
              <div className="section-title-row">
                <h3>Linked Bills &amp; Income</h3>
                <span className="records-count-badge">{linkedRecurring.length}</span>
              </div>
              {allRecurringTransactions.length === 0 ? (
                <p className="text-muted text-sm">No recurring bills or income created yet.</p>
              ) : (
                <div className="perspective-checkbox-grid">
                  {allRecurringTransactions.map((tx) => (
                    <label key={tx.id} className="perspective-checkbox-item">
                      <input
                        type="checkbox"
                        checked={linkedRecurringIds.has(tx.id)}
                        onChange={() => handleToggleRecurringLink(tx.id)}
                      />
                      <span>{tx.entity}</span>
                      <span className="text-muted text-sm">
                        ({tx.amount < 0 ? 'Bill' : 'Income'} {formatCurrency(Math.abs(tx.amount))} / {tx.frequency})
                      </span>
                    </label>
                  ))}
                </div>
              )}
            </section>
          </div>
        </div>
      )}

      {confirmDeleteOpen && (
        <div className="modal-backdrop" onClick={() => !deleting && setConfirmDeleteOpen(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">Delete Perspective</h2>
            </div>
            <div className="modal-body">
              <p>
                Delete perspective <strong>{view.perspective.name}</strong>? Linked accounts, bills, and income will not be deleted.
              </p>
            </div>
            <div className="modal-actions mt-4">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setConfirmDeleteOpen(false)}
                disabled={deleting}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn-danger"
                onClick={handleConfirmDelete}
                disabled={deleting}
              >
                {deleting ? 'Deleting...' : 'Delete Perspective'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
