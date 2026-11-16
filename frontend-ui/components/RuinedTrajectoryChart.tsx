'use client';

import { useState, useMemo } from 'react';

interface RuinedTrajectoryChartProps {
  currentSavings: number;
  dailySpend30d: number;
  daysUntilRuined30d: number | null;
  ruinedDate30d: string | null;
  dailySpend90d: number;
  daysUntilRuined90d: number | null;
  ruinedDate90d: string | null;
  dailySpend365d: number;
  daysUntilRuined365d: number | null;
  ruinedDate365d: string | null;
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

export function RuinedTrajectoryChart({
  currentSavings,
  dailySpend30d,
  daysUntilRuined30d,
  ruinedDate30d,
  dailySpend90d,
  daysUntilRuined90d,
  ruinedDate90d,
  dailySpend365d,
  daysUntilRuined365d,
  ruinedDate365d,
}: RuinedTrajectoryChartProps) {
  const [hoveredPace, setHoveredPace] = useState<string | null>(null);

  const width = 800;
  const height = 260;
  const padLeft = 70;
  const padRight = 30;
  const padTop = 20;
  const padBottom = 35;

  const chartWidth = width - padLeft - padRight;
  const chartHeight = height - padTop - padBottom;

  const maxDays = useMemo(() => {
    const list = [
      daysUntilRuined30d || 0,
      daysUntilRuined90d || 0,
      daysUntilRuined365d || 0,
    ];
    const mx = Math.max(...list);
    return mx > 0 ? Math.ceil(mx * 1.1) : 365;
  }, [daysUntilRuined30d, daysUntilRuined90d, daysUntilRuined365d]);

  const getY = (val: number) => {
    if (currentSavings <= 0) return padTop + chartHeight;
    const ratio = Math.max(0, Math.min(1, val / currentSavings));
    return padTop + chartHeight - ratio * chartHeight;
  };

  const getX = (days: number) => {
    const ratio = Math.max(0, Math.min(1, days / maxDays));
    return padLeft + ratio * chartWidth;
  };

  // Trajectory series definition
  const series = useMemo(() => {
    const list = [];
    if (daysUntilRuined30d && daysUntilRuined30d > 0) {
      list.push({
        id: '30d',
        label: '30-Day Spending Pace',
        color: '#ef4444', // Red
        days: daysUntilRuined30d,
        date: ruinedDate30d,
        daily: dailySpend30d,
      });
    }
    if (daysUntilRuined90d && daysUntilRuined90d > 0) {
      list.push({
        id: '90d',
        label: '90-Day Spending Pace',
        color: '#f59e0b', // Amber
        days: daysUntilRuined90d,
        date: ruinedDate90d,
        daily: dailySpend90d,
      });
    }
    if (daysUntilRuined365d && daysUntilRuined365d > 0) {
      list.push({
        id: '365d',
        label: '365-Day Spending Pace',
        color: '#8b5cf6', // Violet
        days: daysUntilRuined365d,
        date: ruinedDate365d,
        daily: dailySpend365d,
      });
    }
    return list;
  }, [
    daysUntilRuined30d,
    dailySpend30d,
    ruinedDate30d,
    daysUntilRuined90d,
    dailySpend90d,
    ruinedDate90d,
    daysUntilRuined365d,
    dailySpend365d,
    ruinedDate365d,
  ]);

  if (currentSavings <= 0 || series.length === 0) {
    return null;
  }

  const startX = getX(0);
  const startY = getY(currentSavings);

  return (
    <div className="perspective-graph-container mt-4">
      <div className="graph-toolbar">
        <span className="trend-chart-title">Loss of Income Trajectory: Spend Down to $0</span>
        <div className="graph-summary-pills">
          {series.map((s) => (
            <div
              key={s.id}
              className="summary-pill"
              style={{
                cursor: 'pointer',
                borderLeft: `4px solid ${s.color}`,
                backgroundColor: hoveredPace === s.id ? '#f1f5f9' : '#f8fafc',
              }}
              onMouseEnter={() => setHoveredPace(s.id)}
              onMouseLeave={() => setHoveredPace(null)}
            >
              <span className="summary-pill-label">{s.label}</span>
              <span className="summary-pill-value">
                {s.days} days ({formatCurrency(s.daily)}/day)
              </span>
            </div>
          ))}
        </div>
      </div>

      <div className="trend-chart-wrapper">
        <svg viewBox={`0 0 ${width} ${height}`} className="trend-chart-svg">
          {/* Horizontal Grid lines */}
          {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
            const val = currentSavings * ratio;
            const yPos = getY(val);
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
                  {formatCurrency(val)}
                </text>
              </g>
            );
          })}

          {/* Zero baseline */}
          <line
            x1={padLeft}
            y1={getY(0)}
            x2={width - padRight}
            y2={getY(0)}
            stroke="#64748b"
            strokeWidth="1.5"
          />

          {/* Trajectory projection lines */}
          {series.map((s) => {
            const endX = getX(s.days);
            const endY = getY(0);
            const isHovered = hoveredPace === s.id;
            const opacity = hoveredPace === null || isHovered ? 1.0 : 0.25;
            const strokeWidth = isHovered ? 3.5 : 2;

            // Intermediate points sampled at least 10px apart (enforcing <= 1 dot per 10 pixels on X axis)
            const pixelDist = Math.abs(endX - startX);
            const maxAllowedDots = Math.max(1, Math.floor(pixelDist / 10));
            const stepDays = Math.max(1, Math.ceil(s.days / maxAllowedDots));

            const dots = [];
            for (let d = 0; d <= s.days; d += stepDays) {
              const ptX = getX(d);
              const val = Math.max(0, currentSavings - d * s.daily);
              const ptY = getY(val);
              dots.push({ d, x: ptX, y: ptY, val });
            }
            // Ensure end dot at $0 is present
            if (dots[dots.length - 1].d !== s.days) {
              dots.push({ d: s.days, x: endX, y: endY, val: 0 });
            }

            return (
              <g key={s.id} opacity={opacity}>
                <line
                  x1={startX}
                  y1={startY}
                  x2={endX}
                  y2={endY}
                  stroke={s.color}
                  strokeWidth={strokeWidth}
                  strokeLinecap="round"
                />
                {/* Dots along line (spaced >= 10px on X axis) */}
                {dots.map((pt, i) => (
                  <circle
                    key={i}
                    cx={pt.x}
                    cy={pt.y}
                    r={i === 0 || i === dots.length - 1 ? 4 : 2}
                    fill={s.color}
                  />
                ))}
                {/* Runway end label */}
                <text
                  x={endX}
                  y={endY - 8}
                  fill={s.color}
                  fontSize="11"
                  fontWeight="600"
                  textAnchor={endX > width - 100 ? 'end' : 'middle'}
                >
                  {s.days}d
                </text>
              </g>
            );
          })}

          {/* Starting dot at Today / Current Savings */}
          <circle cx={startX} cy={startY} r="5" fill="#3b82f6" stroke="#1e293b" strokeWidth="2" />
          <text x={startX} y={startY - 8} fill="#3b82f6" fontSize="11" fontWeight="700">
            Today ({formatCurrency(currentSavings)})
          </text>

          {/* X Axis Day Labels */}
          <text x={padLeft} y={height - 10} fill="#94a3b8" fontSize="11" textAnchor="start">
            Day 0 (Today)
          </text>
          <text x={width - padRight} y={height - 10} fill="#94a3b8" fontSize="11" textAnchor="end">
            Day {maxDays}
          </text>
        </svg>
      </div>
    </div>
  );
}
