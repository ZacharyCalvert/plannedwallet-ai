'use client';

import { useState, useMemo } from 'react';
import { AssetAccountTrendPoint } from '../lib/types';

interface AssetTrendGraphProps {
  points: AssetAccountTrendPoint[];
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

function formatDateDisplay(isoDate: string): string {
  if (!isoDate) return '';
  const parts = isoDate.split('-');
  if (parts.length !== 3) return isoDate;
  const [y, m, d] = parts;
  const dateObj = new Date(parseInt(y, 10), parseInt(m, 10) - 1, parseInt(d, 10));
  return dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function AssetTrendGraph({ points }: AssetTrendGraphProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [rangeFilter, setRangeFilter] = useState<'all' | '30d' | '90d' | '1y'>('all');

  // Sort points chronologically
  const sorted = useMemo(() => {
    const list = Array.isArray(points) ? points : [];
    return [...list].sort((a, b) => a.date.localeCompare(b.date));
  }, [points]);

  // Apply range filter
  const filtered = useMemo(() => {
    if (rangeFilter === 'all' || sorted.length === 0) return sorted;
    const latestDate = new Date(sorted[sorted.length - 1].date);
    let daysToSubtract = 30;
    if (rangeFilter === '90d') daysToSubtract = 90;
    if (rangeFilter === '1y') daysToSubtract = 365;

    const cutoff = new Date(latestDate);
    cutoff.setDate(cutoff.getDate() - daysToSubtract);
    const cutoffStr = cutoff.toISOString().split('T')[0];

    const result = sorted.filter((s) => s.date >= cutoffStr);
    return result.length > 0 ? result : sorted;
  }, [sorted, rangeFilter]);

  // Metrics
  const metrics = useMemo(() => {
    if (filtered.length === 0) return null;
    const first = filtered[0];
    const latest = filtered[filtered.length - 1];
    const change = latest.total_value - first.total_value;
    const pctChange = first.total_value !== 0 ? (change / Math.abs(first.total_value)) * 100 : 0;

    let minVal = filtered[0].total_value;
    let maxVal = filtered[0].total_value;
    for (const s of filtered) {
      if (s.total_value < minVal) minVal = s.total_value;
      if (s.total_value > maxVal) maxVal = s.total_value;
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

    let minVal = filtered[0].total_value;
    let maxVal = filtered[0].total_value;
    for (const s of filtered) {
      if (s.total_value < minVal) minVal = s.total_value;
      if (s.total_value > maxVal) maxVal = s.total_value;
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

    const firstTime = new Date(filtered[0].date).getTime();
    const lastTime = new Date(filtered[filtered.length - 1].date).getTime();
    const timeSpan = lastTime - firstTime;

    const pts = filtered.map((s, idx) => {
      let x = padLeft;
      if (filtered.length === 1) {
        x = padLeft + plotWidth / 2;
      } else if (timeSpan > 0) {
        const t = new Date(s.date).getTime();
        x = padLeft + ((t - firstTime) / timeSpan) * plotWidth;
      } else {
        x = padLeft + (idx / (filtered.length - 1)) * plotWidth;
      }

      const yNorm = (s.total_value - yMin) / (yRange || 1);
      const y = padTop + plotHeight - yNorm * plotHeight;

      return { x, y, point: s, index: idx };
    });

    // Build SVG path
    const linePath = pts.reduce((acc, pt, i) => {
      return `${acc} ${i === 0 ? 'M' : 'L'} ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`;
    }, '');

    // Build SVG area fill path
    const firstPt = pts[0];
    const lastPt = pts[pts.length - 1];
    const baselineY = padTop + plotHeight;
    const areaPath = `${linePath} L ${lastPt.x.toFixed(1)} ${baselineY} L ${firstPt.x.toFixed(1)} ${baselineY} Z`;

    // 4 Y grid lines
    const gridLines = [0, 0.333, 0.666, 1].map((ratio) => {
      const val = yMin + ratio * yRange;
      const y = padTop + plotHeight - ratio * plotHeight;
      return { y, val };
    });

    return { points: pts, linePath, areaPath, gridLines, yMin, yMax };
  }, [filtered, padLeft, padRight, padTop, padBottom, plotWidth, plotHeight]);

  const activePoint = hoveredIndex !== null && chartData ? chartData.points[hoveredIndex] : null;

  return (
    <div className="trend-graph-section">
      {/* Header and Filter Buttons */}
      <div className="graph-controls-row">
        <div>
          <h3 className="graph-title">Portfolio Valuation Trend</h3>
          <p className="graph-subtitle">
            Timeline of account worth calculated from asset quantities and valuations over time
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
            <span className="metric-label">Starting Portfolio Value</span>
            <span className="metric-val">{formatCurrency(metrics.first.total_value)}</span>
            <span className="metric-sub">{metrics.first.date}</span>
          </div>

          <div className="metric-tile">
            <span className="metric-label">Current Portfolio Value</span>
            <span className="metric-val">{formatCurrency(metrics.latest.total_value)}</span>
            <span className="metric-sub">{metrics.latest.date}</span>
          </div>

          <div className="metric-tile">
            <span className="metric-label">Net Value Change</span>
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
            <span className="metric-label">Valuation Range</span>
            <span className="metric-val">
              {formatCurrency(metrics.minVal)} - {formatCurrency(metrics.maxVal)}
            </span>
            <span className="metric-sub">{filtered.length} timeline points</span>
          </div>
        </div>
      )}

      {/* Graph Area */}
      {sorted.length === 0 ? (
        <div className="graph-empty-card">
          <div className="graph-empty-icon">📈</div>
          <h4>No Valuation History Recorded</h4>
          <p>
            Add asset holdings and asset valuations across dates to generate a continuous valuation trend.
          </p>
        </div>
      ) : sorted.length === 1 ? (
        <div className="graph-single-card">
          <div className="graph-single-callout">
            <span className="single-val">{formatCurrency(sorted[0].total_value)}</span>
            <span className="single-date">Recorded on {formatDateDisplay(sorted[0].date)}</span>
          </div>
          <p className="single-subtext">
            Record holdings or valuations across multiple dates to view your continuous trend line.
          </p>
        </div>
      ) : chartData ? (
        <div className="graph-canvas-wrapper">
          {/* Floating Tooltip */}
          {activePoint && (
            <div
              className={`graph-floating-tooltip ${activePoint.y < 95 ? 'tooltip-below' : ''}`}
              style={{
                left: `${(activePoint.x / svgWidth) * 100}%`,
                top: `${(activePoint.y / svgHeight) * 100}%`,
              }}
            >
              <div className="tooltip-date">
                {formatDateDisplay(activePoint.point.date)}
              </div>
              <div className="tooltip-value">
                {formatCurrency(activePoint.point.total_value)}
              </div>
              {activePoint.index > 0 && (
                <div className="tooltip-diff">
                  {(() => {
                    const prev = chartData.points[activePoint.index - 1].point;
                    const diff = activePoint.point.total_value - prev.total_value;
                    const sign = diff >= 0 ? '+' : '';
                    return `${sign}${formatCurrency(diff)} from ${prev.date}`;
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
              <linearGradient id="assetTrendGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#0284c7" stopOpacity="0.28" />
                <stop offset="100%" stopColor="#0284c7" stopOpacity="0.01" />
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
            <path d={chartData.areaPath} fill="url(#assetTrendGradient)" />

            {/* Main Trend Polyline */}
            <path
              d={chartData.linePath}
              fill="none"
              stroke="#0284c7"
              strokeWidth="3.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />

            {/* Interactive Data Points */}
            {chartData.points.map((pt) => {
              const isHovered = hoveredIndex === pt.index;
              return (
                <g
                  key={pt.point.date + '-' + pt.index}
                  className="point-interactive-group"
                  onMouseEnter={() => setHoveredIndex(pt.index)}
                  onMouseLeave={() => setHoveredIndex(null)}
                >
                  {/* Invisible larger target for smooth hovering */}
                  <circle
                    cx={pt.x}
                    cy={pt.y}
                    r="18"
                    fill="#000000"
                    opacity="0"
                    pointerEvents="all"
                    className="point-hit-target"
                  />
                  {/* Visible point circle */}
                  <circle
                    cx={pt.x}
                    cy={pt.y}
                    r={isHovered ? 7 : 4.5}
                    fill={isHovered ? '#0284c7' : '#ffffff'}
                    stroke="#0284c7"
                    strokeWidth={isHovered ? 3 : 2}
                    pointerEvents="none"
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
                  key={'lbl-' + pt.point.date + '-' + i}
                  x={pt.x}
                  y={padTop + plotHeight + 20}
                  textAnchor="middle"
                  fontSize="11"
                  fill="#64748b"
                  fontFamily="inherit"
                >
                  {pt.point.date.slice(5)}
                </text>
              );
            })}
          </svg>
        </div>
      ) : null}
    </div>
  );
}
