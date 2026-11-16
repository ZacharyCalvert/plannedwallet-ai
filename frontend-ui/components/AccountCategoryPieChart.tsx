'use client';

import { useState, useMemo } from 'react';
import { AccountRecord, Category } from '../lib/types';
import { BalanceRange } from '../lib/accountTrend';
import { computeCategoryPieData, PieChartSlice } from '../lib/accountCategories';

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
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);

  const summary = useMemo(
    () => computeCategoryPieData(records, categories, rangeFilter, includeUncategorized),
    [records, categories, rangeFilter, includeUncategorized]
  );

  const segments = useMemo(() => buildArcSegments(summary.slices), [summary.slices]);
  const activeSlice = hoveredIdx !== null ? summary.slices[hoveredIdx] ?? null : null;

  return (
    <div className="trend-graph-section">
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Expenses by Category</h3>
          <p className="graph-subtitle">Breakdown of negative (spending) transactions by category</p>
        </div>
        <div className="expense-controls-right">
          <label className="uncategorized-toggle">
            <input
              type="checkbox"
              checked={includeUncategorized}
              onChange={(e) => {
                setHoveredIdx(null);
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
                  setHoveredIdx(null);
                  setRangeFilter(opt.value);
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {summary.slices.length === 0 ? (
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
                  onMouseEnter={() => setHoveredIdx(0)}
                  onMouseLeave={() => setHoveredIdx(null)}
                />
              ) : (
                segments.map((seg) => (
                  <path
                    key={seg.slice.categoryName}
                    d={seg.path}
                    fill={seg.slice.color}
                    stroke="#ffffff"
                    strokeWidth="2"
                    className={`pie-slice ${hoveredIdx === seg.index ? 'hovered' : ''}`}
                    onMouseEnter={() => setHoveredIdx(seg.index)}
                    onMouseLeave={() => setHoveredIdx(null)}
                  />
                ))
              )}
              <text x="150" y="142" textAnchor="middle" className="pie-center-label">
                {activeSlice ? activeSlice.categoryName : 'Total Spend'}
              </text>
              <text x="150" y="166" textAnchor="middle" className="pie-center-value">
                {formatCurrency(activeSlice ? activeSlice.amount : summary.totalExpenses)}
              </text>
              {activeSlice && (
                <text x="150" y="185" textAnchor="middle" className="pie-center-pct">
                  {activeSlice.percentage.toFixed(1)}%
                </text>
              )}
            </svg>
          </div>

          <div className="pie-legend-card">
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
                {summary.slices.map((slice, idx) => (
                  <tr
                    key={slice.categoryName}
                    className={hoveredIdx === idx ? 'legend-row-active' : ''}
                    onMouseEnter={() => setHoveredIdx(idx)}
                    onMouseLeave={() => setHoveredIdx(null)}
                  >
                    <td>
                      <span className="legend-swatch" style={{ backgroundColor: slice.color }} />
                      <span className="font-semibold">{slice.categoryName}</span>
                    </td>
                    <td className="text-right text-muted">{slice.recordCount}</td>
                    <td className="text-right font-semibold">{formatCurrency(slice.amount)}</td>
                    <td className="text-right text-muted">{slice.percentage.toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
