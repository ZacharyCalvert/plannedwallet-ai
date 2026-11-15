'use client';

import { useState, useMemo } from 'react';
import { Account, PerspectiveTrendPoint } from '../lib/types';

interface PerspectiveTrendGraphProps {
  trend: PerspectiveTrendPoint[];
  accounts: Account[];
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

export function PerspectiveTrendGraph({ trend, accounts }: PerspectiveTrendGraphProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [rangeFilter, setRangeFilter] = useState<'all' | '30d' | '90d' | '1y'>('all');

  const accountMap = useMemo(() => {
    const map = new Map<string, Account>();
    for (const a of accounts) {
      map.set(a.id, a);
    }
    return map;
  }, [accounts]);

  const sorted = useMemo(() => {
    return [...trend].sort((a, b) => a.date.localeCompare(b.date));
  }, [trend]);

  const filtered = useMemo(() => {
    if (rangeFilter === 'all' || sorted.length === 0) return sorted;
    const latestDate = new Date(sorted[sorted.length - 1].date);
    let daysToSubtract = 30;
    if (rangeFilter === '90d') daysToSubtract = 90;
    if (rangeFilter === '1y') daysToSubtract = 365;

    const cutoff = new Date(latestDate);
    cutoff.setDate(cutoff.getDate() - daysToSubtract);
    const cutoffStr = cutoff.toISOString().split('T')[0];

    const result = sorted.filter((p) => p.date >= cutoffStr);
    return result.length > 0 ? result : sorted;
  }, [sorted, rangeFilter]);

  const metrics = useMemo(() => {
    if (filtered.length === 0) return null;
    const first = filtered[0];
    const latest = filtered[filtered.length - 1];
    const change = latest.total_value - first.total_value;
    const pctChange = first.total_value !== 0 ? (change / Math.abs(first.total_value)) * 100 : 0;

    let minVal = filtered[0].total_value;
    let maxVal = filtered[0].total_value;
    for (const p of filtered) {
      if (p.total_value < minVal) minVal = p.total_value;
      if (p.total_value > maxVal) maxVal = p.total_value;
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

  if (filtered.length === 0) {
    return (
      <div className="trend-graph-empty card">
        <p>No historical trend points recorded across linked accounts.</p>
      </div>
    );
  }

  const width = 800;
  const height = 320;
  const padLeft = 70;
  const padRight = 30;
  const padTop = 30;
  const padBottom = 40;

  const chartWidth = width - padLeft - padRight;
  const chartHeight = height - padTop - padBottom;

  const minVal = metrics ? Math.min(metrics.minVal, 0) : 0;
  const maxVal = metrics ? Math.max(metrics.maxVal, 100) : 100;
  const valRange = maxVal - minVal === 0 ? 1 : maxVal - minVal;

  const getX = (idx: number) => {
    if (filtered.length === 1) return padLeft + chartWidth / 2;
    return padLeft + (idx / (filtered.length - 1)) * chartWidth;
  };

  const getY = (val: number) => {
    return padTop + chartHeight - ((val - minVal) / valRange) * chartHeight;
  };

  const pointsString = filtered.map((p, idx) => `${getX(idx)},${getY(p.total_value)}`).join(' ');
  const areaPoints = `${getX(0)},${padTop + chartHeight} ${pointsString} ${getX(filtered.length - 1)},${padTop + chartHeight}`;

  const hoveredPoint = hoveredIndex !== null ? filtered[hoveredIndex] : null;

  return (
    <div className="perspective-trend-container">
      {/* Metric Cards Row */}
      {metrics && (
        <div className="trend-metrics-row">
          <div className="trend-metric-card">
            <span className="metric-label">Start Valuation</span>
            <span className="metric-value">{formatCurrency(metrics.first.total_value)}</span>
            <span className="metric-subtext">{formatDateDisplay(metrics.first.date)}</span>
          </div>

          <div className="trend-metric-card">
            <span className="metric-label">Latest Valuation</span>
            <span className="metric-value">{formatCurrency(metrics.latest.total_value)}</span>
            <span className="metric-subtext">{formatDateDisplay(metrics.latest.date)}</span>
          </div>

          <div className="trend-metric-card">
            <span className="metric-label">Net Growth</span>
            <span className={`metric-value ${metrics.change >= 0 ? 'amount-text-positive' : 'amount-text-negative'}`}>
              {metrics.change >= 0 ? '+' : ''}
              {formatCurrency(metrics.change)}
            </span>
            <span className={`metric-subtext ${metrics.pctChange >= 0 ? 'text-positive' : 'text-negative'}`}>
              {metrics.pctChange >= 0 ? '▲' : '▼'} {metrics.pctChange.toFixed(2)}%
            </span>
          </div>

          <div className="trend-metric-card">
            <span className="metric-label">Min / Max Range</span>
            <span className="metric-value">{formatCurrency(metrics.maxVal)}</span>
            <span className="metric-subtext">Low: {formatCurrency(metrics.minVal)}</span>
          </div>
        </div>
      )}

      {/* Range Filter Controls */}
      <div className="trend-controls-bar">
        <span className="trend-chart-title">Aggregate Net Worth Over Time</span>
        <div className="btn-group-range">
          <button
            type="button"
            className={`btn-range ${rangeFilter === 'all' ? 'active' : ''}`}
            onClick={() => setRangeFilter('all')}
          >
            All
          </button>
          <button
            type="button"
            className={`btn-range ${rangeFilter === '1y' ? 'active' : ''}`}
            onClick={() => setRangeFilter('1y')}
          >
            1Y
          </button>
          <button
            type="button"
            className={`btn-range ${rangeFilter === '90d' ? 'active' : ''}`}
            onClick={() => setRangeFilter('90d')}
          >
            90D
          </button>
          <button
            type="button"
            className={`btn-range ${rangeFilter === '30d' ? 'active' : ''}`}
            onClick={() => setRangeFilter('30d')}
          >
            30D
          </button>
        </div>
      </div>

      {/* Interactive SVG Chart */}
      <div className="trend-chart-wrapper">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="trend-chart-svg"
          onMouseLeave={() => setHoveredIndex(null)}
        >
          <defs>
            <linearGradient id="perspGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stopColor="#8b5cf6" stopOpacity="0.3" />
              <stop offset="100%" stopColor="#8b5cf6" stopOpacity="0.0" />
            </linearGradient>
          </defs>

          {/* Grid lines */}
          {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
            const yVal = minVal + ratio * valRange;
            const yPos = getY(yVal);
            return (
              <g key={ratio}>
                <line
                  x1={padLeft}
                  y1={yPos}
                  x2={width - padRight}
                  y2={yPos}
                  stroke="#334155"
                  strokeDasharray="4 4"
                  strokeWidth="1"
                />
                <text
                  x={padLeft - 8}
                  y={yPos + 4}
                  fill="#94a3b8"
                  fontSize="11"
                  textAnchor="end"
                >
                  {formatCurrency(yVal)}
                </text>
              </g>
            );
          })}

          {/* Fill area */}
          <polygon points={areaPoints} fill="url(#perspGradient)" />

          {/* Line */}
          <polyline
            points={pointsString}
            fill="none"
            stroke="#8b5cf6"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Interactive Points */}
          {filtered.map((p, idx) => {
            const x = getX(idx);
            const y = getY(p.total_value);
            const isHovered = hoveredIndex === idx;
            return (
              <g
                key={p.date + idx}
                onMouseEnter={() => setHoveredIndex(idx)}
                style={{ cursor: 'pointer' }}
              >
                {/* Large transparent hover target */}
                <circle cx={x} cy={y} r="14" fill="transparent" />
                {/* Visible dot */}
                <circle
                  cx={x}
                  cy={y}
                  r={isHovered ? 6 : 3.5}
                  fill={isHovered ? '#c4b5fd' : '#8b5cf6'}
                  stroke="#1e293b"
                  strokeWidth="2"
                />
              </g>
            );
          })}

          {/* Hover Crosshair Line */}
          {hoveredIndex !== null && (
            <line
              x1={getX(hoveredIndex)}
              y1={padTop}
              x2={getX(hoveredIndex)}
              y2={padTop + chartHeight}
              stroke="#a78bfa"
              strokeDasharray="2 2"
              strokeWidth="1.5"
              pointerEvents="none"
            />
          )}

          {/* X Axis Labels */}
          <text x={padLeft} y={height - 10} fill="#94a3b8" fontSize="11" textAnchor="start">
            {formatDateDisplay(filtered[0].date)}
          </text>
          {filtered.length > 1 && (
            <text x={width - padRight} y={height - 10} fill="#94a3b8" fontSize="11" textAnchor="end">
              {formatDateDisplay(filtered[filtered.length - 1].date)}
            </text>
          )}
        </svg>

        {/* Hover Tooltip Box */}
        {hoveredPoint && hoveredIndex !== null && (
          <div
            className="trend-tooltip card"
            style={{
              left: `${Math.min(Math.max((getX(hoveredIndex) / width) * 100, 15), 85)}%`,
              top: '12%',
            }}
          >
            <div className="tooltip-date">{formatDateDisplay(hoveredPoint.date)}</div>
            <div className="tooltip-total-row">
              <span className="tooltip-label">Combined Total:</span>
              <span className="tooltip-value">{formatCurrency(hoveredPoint.total_value)}</span>
            </div>
            {hoveredPoint.account_values && Object.keys(hoveredPoint.account_values).length > 0 && (
              <div className="tooltip-breakdown">
                <span className="breakdown-title">Account Breakdown:</span>
                {Object.entries(hoveredPoint.account_values).map(([accId, val]) => {
                  const acc = accountMap.get(accId);
                  const name = acc ? acc.name : `Account (${accId.slice(0, 6)}...)`;
                  return (
                    <div key={accId} className="tooltip-breakdown-row">
                      <span className="breakdown-name">{name}:</span>
                      <span className="breakdown-val">{formatCurrency(val)}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
