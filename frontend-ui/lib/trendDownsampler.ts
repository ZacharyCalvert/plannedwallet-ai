import type { PerspectiveTrendPoint } from './types.ts';

export interface DownsampledPoint {
  date: string;
  startDate: string;
  endDate: string;
  total_value: number;
  account_values?: { [key: string]: number };
  x: number;
  count: number;
}

/**
 * Downsamples trend points so that no more than 1 dot is rendered per minPixelDistance (default 10px) on the X axis.
 * All points that fall within each pixel window are averaged together.
 */
export function downsampleTrendPoints(
  points: PerspectiveTrendPoint[],
  chartWidth: number,
  padLeft: number = 0,
  minPixelDistance: number = 10
): DownsampledPoint[] {
  if (!points || points.length === 0) {
    return [];
  }
  if (points.length === 1) {
    return [
      {
        date: points[0].date,
        startDate: points[0].date,
        endDate: points[0].date,
        total_value: points[0].total_value,
        account_values: points[0].account_values,
        x: padLeft + chartWidth / 2,
        count: 1,
      },
    ];
  }

  const numBuckets = Math.max(1, Math.floor(chartWidth / minPixelDistance));
  const buckets: { [bucketIdx: number]: PerspectiveTrendPoint[] } = {};

  for (let i = 0; i < points.length; i++) {
    const rawRatio = i / (points.length - 1);
    const rawX = rawRatio * chartWidth;
    const bucketIdx = Math.min(numBuckets - 1, Math.floor(rawX / minPixelDistance));
    if (!buckets[bucketIdx]) {
      buckets[bucketIdx] = [];
    }
    buckets[bucketIdx].push(points[i]);
  }

  const result: DownsampledPoint[] = [];
  const bucketIndices = Object.keys(buckets)
    .map(Number)
    .sort((a, b) => a - b);

  for (const bIdx of bucketIndices) {
    const group = buckets[bIdx];
    const count = group.length;
    const sumTotal = group.reduce((acc, p) => acc + p.total_value, 0);
    const avgTotal = sumTotal / count;

    const accSums: { [accId: string]: number } = {};
    const accCounts: { [accId: string]: number } = {};
    for (const p of group) {
      if (p.account_values) {
        for (const [k, v] of Object.entries(p.account_values)) {
          accSums[k] = (accSums[k] || 0) + v;
          accCounts[k] = (accCounts[k] || 0) + 1;
        }
      }
    }
    const avgAccValues: { [accId: string]: number } = {};
    for (const [k, s] of Object.entries(accSums)) {
      avgAccValues[k] = s / (accCounts[k] || 1);
    }

    const startDate = group[0].date;
    const endDate = group[group.length - 1].date;
    const date = startDate === endDate ? startDate : `${startDate} – ${endDate}`;
    const x = padLeft + bIdx * minPixelDistance + minPixelDistance / 2;

    result.push({
      date,
      startDate,
      endDate,
      total_value: avgTotal,
      account_values: Object.keys(avgAccValues).length > 0 ? avgAccValues : undefined,
      x,
      count,
    });
  }

  return result;
}
