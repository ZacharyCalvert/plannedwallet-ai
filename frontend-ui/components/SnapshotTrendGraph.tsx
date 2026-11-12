'use client';

import { useState, useMemo } from 'react';
import { AccountSnapshot } from '../lib/types';

interface SnapshotTrendGraphProps {
  snapshots: AccountSnapshot[];
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

function formatDateDisplay(isoDate: string): string {
  if (!isoDate) return '';
  const [y, m, d] = isoDate.split('-');
  const dateObj = new Date(parseInt(y, 10), parseInt(m, 10) - 1, parseInt(d, 10));
  return dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function SnapshotTrendGraph({ snapshots }: SnapshotTrendGraphProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [rangeFilter, setRangeFilter] = useState<'all' | '30d' | '90d' | '1y'>('all');

  // Sort snapshots chronologically
  const sorted = useMemo(() => {
    return [...snapshots].sort((a, b) => {
      const cmp = a.snapshot_date.localeCompare(b.snapshot_date);
      if (cmp !== 0) return cmp;
      return (a.created_at || '').localeCompare(b.created_at || '');
    });
  }, [snapshots]);

  // Apply range filter
  const filtered = useMemo(() => {
    if (rangeFilter === 'all' || sorted.length === 0) return sorted;
    const latestDate = new Date(sorted[sorted.length - 1].snapshot_date);
    let daysToSubtract = 30;
    if (rangeFilter === '90d') daysToSubtract = 90;
    if (rangeFilter === '1y') daysToSubtract = 365;

    const cutoff = new Date(latestDate);
    cutoff.setDate(cutoff.getDate() - daysToSubtract);
    const cutoffStr = cutoff.toISOString().split('T')[0];

    const result = sorted.filter((s) => s.snapshot_date >= cutoffStr);
    return result.length > 0 ? result : sorted;
  }, [sorted, rangeFilter]);

  // Metrics
  const metrics = useMemo(() => {
    if (filtered.length === 0) return null;
    const first = filtered[0];
    const latest = filtered[filtered.length - 1];
    const change = latest.value - first.value;
    const pctChange = first.value !== 0 ? (change / Math.abs(first.value)) * 100 : 0;

    let minVal = filtered[0].value;
    let maxVal = filtered[0].value;
    for (const s of filtered) {
      if (s.value < minVal) minVal = s.value;
      if (s.value > maxVal) maxVal = s.value;
    }

    return {
      first,
      latest,
      change,
      pctChange,
      minVal,
      maxVal,
    };
  }, [filtered]);

  // SVG dimensions & plotting math
  const svgWidth = 840;
  const svgHeight = 340;
  const padLeft = 85;
  const padRight = 45;
  const padTop = 30;
  const padBottom = 45;
  const plotWidth = svgWidth - padLeft - padRight;
  const plotHeight = svgHeight - padTop - padBottom;

  const chartData = useMemo(() => {
    if (filtered.length === 0) return null;

    let minVal = filtered[0].value;
    let maxVal = filtered[0].value;
    for (const s of filtered) {
      if (s.value < minVal) minVal = s.value;
      if (s.value > maxVal) maxVal = s.value;
    }

    let yMin = minVal;
    let yMax = maxVal;
    if (yMin === yMax) {
      yMin = minVal > 0 ? minVal * 0.9 : minVal - 100;
      yMax = maxVal > 0 ? maxVal * 1.1 : maxVal + 100;
    } else {
      const margin = (maxVal - minVal) * 0.12;
      yMin = minVal - margin;
      yMax = maxVal + margin;
    }

    const yRange = yMax - yMin;

    const firstTime = new Date(filtered[0].snapshot_date).getTime();
    const lastTime = new Date(filtered[filtered.length - 1].snapshot_date).getTime();
    const timeSpan = lastTime - firstTime;

    const points = filtered.map((s, idx) => {
      let x = padLeft;
      if (filtered.length === 1) {
        x = padLeft + plotWidth / 2;
      } else if (timeSpan > 0) {
        const t = new Date(s.snapshot_date).getTime();
        x = padLeft + ((t - firstTime) / timeSpan) * plotWidth;
      } else {
        x = padLeft + (idx / (filtered.length - 1)) * plotWidth;
      }

      const yNorm = (s.value - yMin) / (yRange || 1);
      const y = padTop + plotHeight - yNorm * plotHeight;

      return { x, y, snapshot: s, index: idx };
    });

    // Build SVG path
    const linePath = points.reduce((acc, pt, i) => {
      return `${acc} ${i === 0 ? 'M' : 'L'} ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`;
    }, '');

    // Build SVG area fill path
    const firstPt = points[0];
    const lastPt = points[points.length - 1];
    const baselineY = padTop + plotHeight;
    const areaPath = `${linePath} L ${lastPt.x.toFixed(1)} ${baselineY} L ${firstPt.x.toFixed(1)} ${baselineY} Z`;

    // 4 Y grid lines
    const gridLines = [0, 0.333, 0.666, 1].map((ratio) => {
      const val = yMin + ratio * yRange;
      const y = padTop + plotHeight - ratio * plotHeight;
      return { y, val };
    });

    return { points, linePath, areaPath, gridLines, yMin, yMax };
  }, [filtered, padLeft, padRight, padTop, padBottom, plotWidth, plotHeight]);

  const activePoint = hoveredIndex !== null && chartData ? chartData.points[hoveredIndex] : null;

  return (
    <div className="trend-graph-section">
      {/* Header and Filter Buttons */}
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Account Value Over Time</h3>
          <p className="graph-subtitle">
            Visual progression of manual point-in-time valuations
          </p>
        </div>
        <div className="graph-range-pills">
          <button
            type="button"
            className={`pill-btn ${rangeFilter === 'all' ? 'active' : ''}`}
            onClick={() => setRangeFilter('all')}
          >
            All Time
          </button>
          <button
            type="button"
            className={`pill-btn ${rangeFilter === '1y' ? 'active' : ''}`}
            onClick={() => setRangeFilter('1y')}
          >
            1 Year
          </button>
          <button
            type="button"
            className={`pill-btn ${rangeFilter === '90d' ? 'active' : ''}`}
            onClick={() => setRangeFilter('90d')}
          >
            90 Days
          </button>
          <button
            type="button"
            className={`pill-btn ${rangeFilter === '30d' ? 'active' : ''}`}
            onClick={() => setRangeFilter('30d')}
          >
            30 Days
          </button>
        </div>
      </div>

      {/* Metric Tiles */}
      {metrics && (
        <div className="graph-metrics-grid">
          <div className="metric-tile">
            <span className="metric-label">Initial Valuation</span>
            <span className="metric-val">{formatCurrency(metrics.first.value)}</span>
            <span className="metric-sub">{metrics.first.snapshot_date}</span>
          </div>

          <div className="metric-tile">
            <span className="metric-label">Latest Valuation</span>
            <span className="metric-val">{formatCurrency(metrics.latest.value)}</span>
            <span className="metric-sub">{metrics.latest.snapshot_date}</span>
          </div>

          <div className="metric-tile">
            <span className="metric-label">Net Trend</span>
            <span
              className={`metric-val ${
                metrics.change >= 0 ? 'text-positive' : 'text-negative'
              }`}
            >
              {metrics.change >= 0 ? '+' : ''}
              {formatCurrency(metrics.change)}
            </span>
            <span className="metric-sub">
              {metrics.pctChange >= 0 ? '+' : ''}
              {metrics.pctChange.toFixed(2)}% over period
            </span>
          </div>

          <div className="metric-tile">
            <span className="metric-label">Recorded Range</span>
            <span className="metric-val">
              {formatCurrency(metrics.minVal)} - {formatCurrency(metrics.maxVal)}
            </span>
            <span className="metric-sub">{filtered.length} total points</span>
          </div>
        </div>
      )}

      {/* Graph Area */}
      {sorted.length === 0 ? (
        <div className="graph-empty-card">
          <div className="graph-empty-icon">📈</div>
          <h4>No Valuations Recorded</h4>
          <p>Switch to the Snapshots tab to record account values and trend your portfolio over time.</p>
        </div>
      ) : sorted.length === 1 ? (
        <div className="graph-single-card">
          <div className="graph-single-callout">
            <span className="single-val">{formatCurrency(sorted[0].value)}</span>
            <span className="single-date">Recorded on {formatDateDisplay(sorted[0].snapshot_date)}</span>
          </div>
          <p className="single-subtext">
            Add at least two snapshots on different dates to generate a continuous trend line.
          </p>
        </div>
      ) : chartData ? (
        <div className="graph-canvas-wrapper">
          {/* Floating Tooltip Display */}
          {activePoint && (
            <div
              className="graph-floating-tooltip"
              style={{
                left: `${(activePoint.x / svgWidth) * 100}%`,
                top: `${(activePoint.y / svgHeight) * 100}%`,
              }}
            >
              <div className="tooltip-date">
                {formatDateDisplay(activePoint.snapshot.snapshot_date)}
              </div>
              <div className="tooltip-value">
                {formatCurrency(activePoint.snapshot.value)}
              </div>
              {activePoint.index > 0 && (
                <div className="tooltip-diff">
                  {(() => {
                    const prev = chartData.points[activePoint.index - 1].snapshot;
                    const diff = activePoint.snapshot.value - prev.value;
                    const sign = diff >= 0 ? '+' : '';
                    return `${sign}${formatCurrency(diff)} from ${prev.snapshot_date}`;
                  })()}
                </div>
              )}
            </div>
          )}

          <svg
            className="trend-svg"
            viewBox={`0 0 ${svgWidth} ${svgHeight}`}
            preserveAspectRatio="none"
          >
            <defs>
              <linearGradient id="trendGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.28" />
                <stop offset="100%" stopColor="#7c3aed" stopOpacity="0.01" />
              </linearGradient>
            </defs>

            {/* Grid Lines and Y-Axis Labels */}
            {chartData.gridLines.map((gl, i) => (
              <g key={i}>
                <line
                  x1={padLeft}
                  y1={gl.y}
                  x2={svgWidth - padRight}
                  y2={gl.y}
                  stroke="#e2e8f0"
                  strokeWidth="1"
                  strokeDasharray="4 4"
                />
                <text
                  x={padLeft - 10}
                  y={gl.y + 4}
                  textAnchor="end"
                  fontSize="11"
                  fill="#64748b"
                  fontFamily="inherit"
                >
                  {formatCurrency(gl.val)}
                </text>
              </g>
            ))}

            {/* Gradient Area Fill */}
            <path d={chartData.areaPath} fill="url(#trendGradient)" />

            {/* Main Trend Polyline */}
            <path
              d={chartData.linePath}
              fill="none"
              stroke="#7c3aed"
              strokeWidth="3.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />

            {/* Interactive Data Points */}
            {chartData.points.map((pt) => {
              const isHovered = hoveredIndex === pt.index;
              return (
                <g key={pt.snapshot.id}>
                  {/* Invisible larger target for smooth hovering */}
                  <circle
                    cx={pt.x}
                    cy={pt.y}
                    r="16"
                    fill="transparent"
                    className="point-hit-target"
                    onMouseEnter={() => setHoveredIndex(pt.index)}
                    onMouseLeave={() => setHoveredIndex(null)}
                  />
                  {/* Visible point circle */}
                  <circle
                    cx={pt.x}
                    cy={pt.y}
                    r={isHovered ? 7 : 4.5}
                    fill={isHovered ? '#7c3aed' : '#ffffff'}
                    stroke="#7c3aed"
                    strokeWidth={isHovered ? 3 : 2}
                    className="point-circle"
                  />
                </g>
              );
            })}

            {/* X-Axis Baseline */}
            <line
              x1={padLeft}
              y1={padTop + plotHeight}
              x2={svgWidth - padRight}
              y2={padTop + plotHeight}
              stroke="#cbd5e1"
              strokeWidth="1.5"
            />

            {/* X-Axis Date Labels */}
            {chartData.points.map((pt, i) => {
              // Only display date labels on spaced intervals if many points exist
              const total = chartData.points.length;
              const shouldShow =
                total <= 6 ||
                i === 0 ||
                i === total - 1 ||
                i === Math.floor(total / 2) ||
                i === Math.floor(total / 4) ||
                i === Math.floor((3 * total) / 4);

              if (!shouldShow) return null;

              return (
                <text
                  key={pt.snapshot.id}
                  x={pt.x}
                  y={padTop + plotHeight + 20}
                  textAnchor="middle"
                  fontSize="11"
                  fill="#64748b"
                  fontFamily="inherit"
                >
                  {pt.snapshot.snapshot_date.slice(5)}
                </text>
              );
            })}
          </svg>
        </div>
      ) : null}
    </div>
  );
}
