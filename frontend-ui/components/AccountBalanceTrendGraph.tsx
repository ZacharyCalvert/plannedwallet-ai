'use client';

import { useState, useMemo, useRef, useEffect } from 'react';
import { AccountRecord } from '../lib/types';
import {
  BalanceRange,
  RollupPeriod,
  BalanceTrendPoint,
  buildAccountBalanceSeries,
  filterBalanceSeriesByRange,
  rollupAccountBalanceSeries,
  computeBalanceMetrics,
  selectHighlightPoints,
  computeYAxisScale,
  formatRollupXAxisLabel,
} from '../lib/accountTrend';

interface AccountBalanceTrendGraphProps {
  records: AccountRecord[];
  accountType: string;
}

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
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

function formatYAxisCurrency(val: number): string {
  const clean = val === 0 ? 0 : val;
  const isNegative = clean < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(Math.abs(clean));
  return isNegative ? `-${formatted}` : formatted;
}

function formatDateDisplay(isoDate: string): string {
  if (!isoDate) return '';
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dateObj = new Date(y, m - 1, d);
  return dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function getAccentColor(accountType: string): string {
  const normalized = (accountType || '').toLowerCase();
  if (normalized === 'savings') return '#059669';
  if (normalized === 'credit card' || normalized === 'credit_card') return '#e11d48';
  return '#0284c7';
}

function computeChartGeometry(points: BalanceTrendPoint[]) {
  if (points.length === 0) return null;
  let minVal = points[0].balance;
  let maxVal = points[0].balance;
  for (const pt of points) {
    if (pt.balance < minVal) minVal = pt.balance;
    if (pt.balance > maxVal) maxVal = pt.balance;
  }

  const { yMin, yMax, ticks } = computeYAxisScale(minVal, maxVal);
  const yRange = yMax - yMin || 1;
  const firstTime = new Date(points[0].date).getTime();
  const lastTime = new Date(points[points.length - 1].date).getTime();
  const timeSpan = lastTime - firstTime;

  const plotted = points.map((pt, idx) => {
    const x =
      points.length === 1
        ? PAD_LEFT + PLOT_WIDTH / 2
        : timeSpan > 0
        ? PAD_LEFT + ((new Date(pt.date).getTime() - firstTime) / timeSpan) * PLOT_WIDTH
        : PAD_LEFT + (idx / (points.length - 1)) * PLOT_WIDTH;
    const yNorm = (pt.balance - yMin) / yRange;
    const y = PAD_TOP + PLOT_HEIGHT - yNorm * PLOT_HEIGHT;
    return { x, y, point: pt, index: idx };
  });

  const linePath = plotted.reduce(
    (acc, pt, i) => `${acc} ${i === 0 ? 'M' : 'L'} ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`,
    ''
  );
  const clampedZeroNorm = Math.min(1, Math.max(0, (0 - yMin) / yRange));
  const zeroY = PAD_TOP + PLOT_HEIGHT - clampedZeroNorm * PLOT_HEIGHT;
  const areaPath = `${linePath} L ${plotted[plotted.length - 1].x.toFixed(1)} ${zeroY.toFixed(1)} L ${plotted[0].x.toFixed(1)} ${zeroY.toFixed(1)} Z`;
  const crossesZero = yMin < 0 && yMax > 0;
  const gridLines = ticks.map((val) => ({
    y: PAD_TOP + PLOT_HEIGHT - ((val - yMin) / yRange) * PLOT_HEIGHT,
    val,
    isZeroCross: crossesZero && val === 0,
  }));

  return { points: plotted, linePath, areaPath, gridLines };
}

function shouldShowXLabel(index: number, total: number): boolean {
  if (total <= 6 || index === 0 || index === total - 1) return true;
  return (
    index === Math.floor(total / 2) ||
    index === Math.floor(total / 4) ||
    index === Math.floor((3 * total) / 4)
  );
}

export function AccountBalanceTrendGraph({ records, accountType }: AccountBalanceTrendGraphProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [rangeFilter, setRangeFilter] = useState<BalanceRange>('all');
  const [rollupPeriod, setRollupPeriod] = useState<RollupPeriod>('daily');
  const [svgRenderedWidth, setSvgRenderedWidth] = useState<number>(SVG_WIDTH);
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const el = svgRef.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    const updateWidth = () => {
      if (el.clientWidth > 0) setSvgRenderedWidth(el.clientWidth);
    };
    updateWidth();
    const observer = new ResizeObserver(updateWidth);
    observer.observe(el);
    return () => observer.disconnect();
  }, [records.length, rangeFilter, rollupPeriod]);

  const fullSeries = useMemo(() => buildAccountBalanceSeries(records), [records]);
  const rangeFiltered = useMemo(
    () => filterBalanceSeriesByRange(fullSeries, rangeFilter),
    [fullSeries, rangeFilter]
  );
  const filtered = useMemo(
    () => rollupAccountBalanceSeries(rangeFiltered, rollupPeriod),
    [rangeFiltered, rollupPeriod]
  );
  const metrics = useMemo(() => computeBalanceMetrics(filtered), [filtered]);
  const chartData = useMemo(() => computeChartGeometry(filtered), [filtered]);
  const highlightPoints = useMemo(() => {
    if (!chartData) return [];
    const minGapSvgUnits = Math.max(10, 10 * (SVG_WIDTH / Math.max(svgRenderedWidth, 1)));
    return selectHighlightPoints(chartData.points, minGapSvgUnits, (pt) => pt.point.balance);
  }, [chartData, svgRenderedWidth]);

  const accentColor = getAccentColor(accountType);
  const gradientId = `accountTrendGrad-${(accountType || 'checking').replace(/\s+/g, '-')}`;
  const activePoint = hoveredIndex !== null && chartData ? chartData.points[hoveredIndex] : null;

  return (
    <div className="trend-graph-section">
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Balance Over Time</h3>
          <p className="graph-subtitle">
            {rollupPeriod === 'daily'
              ? 'Cumulative account balance across recorded transactions'
              : `Average balance rolled up ${rollupPeriod}`}
          </p>
        </div>
        <div className="graph-pills-cluster">
          <div className="graph-range-pills" role="group" aria-label="Rollup period">
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
          <div className="graph-range-pills" role="group" aria-label="Timeframe filter">
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

      {metrics && (
        <div className="graph-metrics-grid">
          <div className="metric-tile">
            <span className="metric-label">Starting Balance</span>
            <span className="metric-val">{formatCurrency(metrics.first.balance)}</span>
            <span className="metric-sub">{metrics.first.date}</span>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Ending Balance</span>
            <span className="metric-val">{formatCurrency(metrics.latest.balance)}</span>
            <span className="metric-sub">{metrics.latest.date}</span>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Net Change</span>
            <span className={`metric-val ${metrics.change >= 0 ? 'text-positive' : 'text-negative'}`}>
              {metrics.change >= 0 ? '+' : ''}
              {formatCurrency(metrics.change)}
            </span>
            <span className="metric-sub">
              {metrics.pctChange >= 0 ? '+' : ''}
              {metrics.pctChange.toFixed(2)}% over period
            </span>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Balance Range</span>
            <span className="metric-val">
              {formatCurrency(metrics.minVal)} - {formatCurrency(metrics.maxVal)}
            </span>
            <span className="metric-sub">
              {filtered.length} {rollupPeriod === 'daily' ? 'daily points' : `${rollupPeriod} buckets`}
            </span>
          </div>
        </div>
      )}

      {fullSeries.length === 0 ? (
        <div className="graph-empty-card">
          <div className="graph-empty-icon">📈</div>
          <h4>No Records Available</h4>
          <p>Add transactions in the Records tab to visualize your balance trend over time.</p>
        </div>
      ) : filtered.length === 1 ? (
        <div className="graph-single-card">
          <div className="graph-single-callout">
            <span className="single-val">{formatCurrency(filtered[0].balance)}</span>
            <span className="single-date">As of {formatDateDisplay(filtered[0].date)}</span>
          </div>
          <p className="single-subtext">
            Add transactions on at least two different dates to display a continuous balance trend line.
          </p>
        </div>
      ) : chartData ? (
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
                {activePoint.point.periodLabel || formatDateDisplay(activePoint.point.date)}
              </div>
              <div className="tooltip-value">{formatCurrency(activePoint.point.balance)}</div>
              {rollupPeriod !== 'daily' && activePoint.point.sampleCount ? (
                <div className="tooltip-diff">
                  Avg of {activePoint.point.sampleCount}{' '}
                  {activePoint.point.sampleCount === 1 ? 'daily point' : 'daily points'}
                </div>
              ) : (
                activePoint.point.transactionCount > 0 && (
                  <div className="tooltip-diff">
                    {activePoint.point.delta >= 0 ? '+' : ''}
                    {formatCurrency(activePoint.point.delta)} ({activePoint.point.transactionCount}{' '}
                    {activePoint.point.transactionCount === 1 ? 'record' : 'records'})
                  </div>
                )
              )}
            </div>
          )}

          <svg ref={svgRef} className="trend-svg" viewBox={`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`} preserveAspectRatio="none">
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={accentColor} stopOpacity="0.28" />
                <stop offset="100%" stopColor={accentColor} stopOpacity="0.02" />
              </linearGradient>
            </defs>

            {chartData.gridLines.map((gl, i) => (
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

            <path d={chartData.areaPath} fill={`url(#${gradientId})`} />
            <path
              d={chartData.linePath}
              fill="none"
              stroke={accentColor}
              strokeWidth="3.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />

            {highlightPoints.map((pt) => {
              const isHovered = hoveredIndex === pt.index;
              return (
                <g
                  key={pt.point.date}
                  className="point-interactive-group"
                  onMouseEnter={() => setHoveredIndex(pt.index)}
                  onMouseLeave={() => setHoveredIndex(null)}
                >
                  <circle cx={pt.x} cy={pt.y} r="18" fill="#000000" opacity="0" pointerEvents="all" className="point-hit-target" />
                  <circle
                    cx={pt.x}
                    cy={pt.y}
                    r={isHovered ? 7 : 4.5}
                    fill={isHovered ? accentColor : '#ffffff'}
                    stroke={accentColor}
                    strokeWidth={isHovered ? 3 : 2}
                    pointerEvents="none"
                    className="point-circle"
                  />
                </g>
              );
            })}

            <line
              x1={PAD_LEFT}
              y1={PAD_TOP + PLOT_HEIGHT}
              x2={SVG_WIDTH - PAD_RIGHT}
              y2={PAD_TOP + PLOT_HEIGHT}
              stroke="#cbd5e1"
              strokeWidth="1.5"
            />

            {chartData.points.map((pt, i) =>
              shouldShowXLabel(i, chartData.points.length) ? (
                <text
                  key={pt.point.date}
                  x={pt.x}
                  y={PAD_TOP + PLOT_HEIGHT + 20}
                  textAnchor="middle"
                  fontSize="11"
                  fill="#64748b"
                  fontFamily="inherit"
                >
                  {formatRollupXAxisLabel(pt.point.date, rollupPeriod)}
                </text>
              ) : null
            )}
          </svg>
        </div>
      ) : null}
    </div>
  );
}
