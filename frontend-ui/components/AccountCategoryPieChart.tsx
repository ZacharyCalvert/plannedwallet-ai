'use client';

import { useState, useMemo } from 'react';
import { AccountRecord, Category } from '../lib/types';
import { BalanceRange } from '../lib/accountTrend';
import {
  computeCategoryPieData,
  filterAndRecomputePieData,
  getPieSliceKey,
  PieChartSlice,
} from '../lib/accountCategories';

interface AccountCategoryPieChartProps {
  records: AccountRecord[];
  categories: Category[];
}

const RANGE_OPTIONS: { value: BalanceRange; label: string }[] = [
  { value: 'all', label: 'All Time' },
  { value: '1y', label: '1 Year' },
  { value: '90d', label: '90 Days' },
  { value: '60d', label: '60 Days' },
  { value: '30d', label: '30 Days' },
];

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(val);
}

interface ArcSegment {
  slice: PieChartSlice;
  path: string;
  index: number;
}

function polarToCartesian(cx: number, cy: number, r: number, angleDeg: number) {
  const rad = ((angleDeg - 90) * Math.PI) / 180;
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) };
}

function describeDonutSlice(cx: number, cy: number, rOuter: number, rInner: number, startAngle: number, endAngle: number): string {
  const outerStart = polarToCartesian(cx, cy, rOuter, startAngle);
  const outerEnd = polarToCartesian(cx, cy, rOuter, endAngle);
  const innerStart = polarToCartesian(cx, cy, rInner, endAngle);
  const innerEnd = polarToCartesian(cx, cy, rInner, startAngle);
  const largeArc = endAngle - startAngle > 180 ? 1 : 0;

  return [
    `M ${outerStart.x.toFixed(2)} ${outerStart.y.toFixed(2)}`,
    `A ${rOuter} ${rOuter} 0 ${largeArc} 1 ${outerEnd.x.toFixed(2)} ${outerEnd.y.toFixed(2)}`,
    `L ${innerStart.x.toFixed(2)} ${innerStart.y.toFixed(2)}`,
    `A ${rInner} ${rInner} 0 ${largeArc} 0 ${innerEnd.x.toFixed(2)} ${innerEnd.y.toFixed(2)}`,
    'Z',
  ].join(' ');
}

function buildArcSegments(slices: PieChartSlice[]): ArcSegment[] {
  const segments: ArcSegment[] = [];
  let cursorAngle = 0;
  slices.forEach((slice, idx) => {
    const sweep = (slice.percentage / 100) * 360;
    const clampedSweep = Math.min(sweep, 359.99);
    const path = describeDonutSlice(150, 150, 118, 66, cursorAngle, cursorAngle + clampedSweep);
    segments.push({ slice, path, index: idx });
    cursorAngle += sweep;
  });
  return segments;
}

export function AccountCategoryPieChart({ records, categories }: AccountCategoryPieChartProps) {
  const [rangeFilter, setRangeFilter] = useState<BalanceRange>('all');
  const [includeUncategorized, setIncludeUncategorized] = useState(true);
  const [hoveredKey, setHoveredKey] = useState<string | null>(null);
  const [disabledKeys, setDisabledKeys] = useState<Set<string>>(() => new Set());

  const allSummary = useMemo(
    () => computeCategoryPieData(records, categories, rangeFilter, includeUncategorized),
    [records, categories, rangeFilter, includeUncategorized]
  );

  const activeSummary = useMemo(
    () => filterAndRecomputePieData(allSummary, disabledKeys),
    [allSummary, disabledKeys]
  );

  const activeShareByKey = useMemo(
    () => new Map(activeSummary.slices.map((s) => [getPieSliceKey(s), s.percentage])),
    [activeSummary.slices]
  );

  const segments = useMemo(() => buildArcSegments(activeSummary.slices), [activeSummary.slices]);
  const activeSlice =
    hoveredKey !== null
      ? activeSummary.slices.find((s) => getPieSliceKey(s) === hoveredKey) ?? null
      : null;

  const toggleCategoryKey = (key: string) => {
    setDisabledKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
        if (hoveredKey === key) {
          setHoveredKey(null);
        }
      }
      return next;
    });
  };

  return (
    <div className="trend-graph-section">
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Expenses by Category</h3>
          <p className="graph-subtitle">
            Breakdown of negative (spending) transactions by category — click a row to hide or show it
          </p>
        </div>
        <div className="expense-controls-right">
          <label className="uncategorized-toggle">
            <input
              type="checkbox"
              checked={includeUncategorized}
              onChange={(e) => {
                setHoveredKey(null);
                setIncludeUncategorized(e.target.checked);
              }}
            />
            <span>Include uncategorized spend</span>
          </label>
          <div className="graph-range-pills" role="group" aria-label="Expense timeframe filter">
            {RANGE_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className={`pill-btn ${rangeFilter === opt.value ? 'active' : ''}`}
                onClick={() => {
                  setHoveredKey(null);
                  setRangeFilter(opt.value);
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {allSummary.slices.length === 0 ? (
        <div className="graph-empty-card">
          <h4>No Matching Expenses</h4>
          <p>
            {includeUncategorized
              ? 'No expense transactions were recorded in the selected timeframe.'
              : 'No categorized expense transactions were recorded in the selected timeframe.'}
          </p>
        </div>
      ) : (
        <div className="pie-chart-layout">
          <div className="pie-svg-card">
            {activeSummary.slices.length === 0 ? (
              <div className="graph-empty-card">
                <h4>All Categories Hidden</h4>
                <p>Click any greyed-out category row to re-enable it in the pie chart.</p>
              </div>
            ) : (
              <svg viewBox="0 0 300 300" className="pie-svg" role="img" aria-label="Expense categories pie chart">
                {segments.length === 1 ? (
                  <circle
                    cx="150"
                    cy="150"
                    r="92"
                    fill="none"
                    stroke={segments[0].slice.color}
                    strokeWidth="52"
                    className="pie-slice"
                    onMouseEnter={() => setHoveredKey(getPieSliceKey(segments[0].slice))}
                    onMouseLeave={() => setHoveredKey(null)}
                  />
                ) : (
                  segments.map((seg) => {
                    const key = getPieSliceKey(seg.slice);
                    return (
                      <path
                        key={seg.slice.categoryName}
                        d={seg.path}
                        fill={seg.slice.color}
                        stroke="#ffffff"
                        strokeWidth="2"
                        className={`pie-slice ${hoveredKey === key ? 'hovered' : ''}`}
                        onMouseEnter={() => setHoveredKey(key)}
                        onMouseLeave={() => setHoveredKey(null)}
                      />
                    );
                  })
                )}
                <text x="150" y="142" textAnchor="middle" className="pie-center-label">
                  {activeSlice ? activeSlice.categoryName : 'Total Spend'}
                </text>
                <text x="150" y="166" textAnchor="middle" className="pie-center-value">
                  {formatCurrency(activeSlice ? activeSlice.amount : activeSummary.totalExpenses)}
                </text>
                {activeSlice && (
                  <text x="150" y="185" textAnchor="middle" className="pie-center-pct">
                    {activeSlice.percentage.toFixed(1)}%
                  </text>
                )}
              </svg>
            )}
          </div>

          <div className="pie-legend-card">
            {disabledKeys.size > 0 && (
              <div className="pie-legend-header">
                <span className="text-muted text-sm">
                  {disabledKeys.size} {disabledKeys.size === 1 ? 'category' : 'categories'} hidden
                </span>
                <button
                  type="button"
                  className="btn-secondary btn-sm"
                  onClick={() => setDisabledKeys(new Set())}
                >
                  Show All
                </button>
              </div>
            )}
            <table className="pie-legend-table">
              <thead>
                <tr>
                  <th>Category</th>
                  <th className="text-right">Records</th>
                  <th className="text-right">Amount</th>
                  <th className="text-right">Share</th>
                </tr>
              </thead>
              <tbody>
                {allSummary.slices.map((slice) => {
                  const key = getPieSliceKey(slice);
                  const isDisabled = disabledKeys.has(key);
                  const activePct = activeShareByKey.get(key);
                  const rowClasses = [
                    'legend-row-clickable',
                    !isDisabled && hoveredKey === key ? 'legend-row-active' : '',
                    isDisabled ? 'legend-row-disabled' : '',
                  ]
                    .filter(Boolean)
                    .join(' ');

                  return (
                    <tr
                      key={slice.categoryName}
                      className={rowClasses}
                      role="button"
                      tabIndex={0}
                      aria-pressed={!isDisabled}
                      title={
                        isDisabled
                          ? `Click to show "${slice.categoryName}" in pie chart`
                          : `Click to hide "${slice.categoryName}" from pie chart`
                      }
                      onClick={() => toggleCategoryKey(key)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          toggleCategoryKey(key);
                        }
                      }}
                      onMouseEnter={() => !isDisabled && setHoveredKey(key)}
                      onMouseLeave={() => setHoveredKey(null)}
                    >
                      <td>
                        <span
                          className="legend-swatch"
                          style={{ backgroundColor: isDisabled ? '#cbd5e1' : slice.color }}
                        />
                        <span className="font-semibold legend-category-name">{slice.categoryName}</span>
                      </td>
                      <td className="text-right text-muted">{slice.recordCount}</td>
                      <td className="text-right font-semibold">{formatCurrency(slice.amount)}</td>
                      <td className="text-right text-muted">
                        {isDisabled ? 'Hidden' : `${(activePct ?? slice.percentage).toFixed(1)}%`}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
