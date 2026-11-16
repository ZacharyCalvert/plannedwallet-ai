import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { downsampleTrendPoints } from './trendDownsampler.ts';
import type { PerspectiveTrendPoint } from './types.ts';

describe('trendDownsampler Unit Tests', () => {
  it('enforces no more than 1 dot per 10 pixels on the X axis with large datasets', () => {
    const chartWidth = 700;
    const padLeft = 70;
    const minPixelDistance = 10;

    // Generate 1000 daily points
    const points: PerspectiveTrendPoint[] = [];
    for (let i = 0; i < 1000; i++) {
      const dayStr = String(i).padStart(4, '0');
      points.push({
        date: `2024-01-01T${dayStr}`,
        total_value: 10000 + i * 10,
      });
    }

    const downsampled = downsampleTrendPoints(points, chartWidth, padLeft, minPixelDistance);

    // Max dots across 700px with min 10px spacing is 70
    assert.ok(downsampled.length <= Math.floor(chartWidth / minPixelDistance));
    assert.ok(downsampled.length > 0);

    // Verify distance between consecutive dots is at least minPixelDistance (10px)
    for (let i = 1; i < downsampled.length; i++) {
      const diff = downsampled[i].x - downsampled[i - 1].x;
      assert.ok(
        diff >= minPixelDistance,
        `Distance between dot ${i - 1} and ${i} was ${diff}px, expected >= ${minPixelDistance}px`
      );
    }
  });

  it('correctly averages values during the downsampled time windows', () => {
    const chartWidth = 100; // 10 buckets of 10px each
    const padLeft = 0;
    const minPixelDistance = 10;

    // 5 points that will fall into the first 10px bucket
    // (indices 0 to 4 in a 50-point array will have ratio <= 0.08 -> x <= 8px -> bucket 0)
    const points: PerspectiveTrendPoint[] = [
      { date: '2024-01-01', total_value: 10.0, account_values: { acc1: 10.0 } },
      { date: '2024-01-02', total_value: 20.0, account_values: { acc1: 20.0 } },
      { date: '2024-01-03', total_value: 30.0, account_values: { acc1: 30.0 } },
      { date: '2024-01-04', total_value: 40.0, account_values: { acc1: 40.0 } },
      { date: '2024-01-05', total_value: 50.0, account_values: { acc1: 50.0 } },
    ];

    // If chart width is 10px, all 5 points will be in the single bucket [0, 10px)
    const downsampled = downsampleTrendPoints(points, 10, padLeft, minPixelDistance);

    assert.equal(downsampled.length, 1);
    assert.equal(downsampled[0].total_value, 30.0); // (10 + 20 + 30 + 40 + 50) / 5 = 30
    assert.equal(downsampled[0].account_values?.acc1, 30.0);
    assert.equal(downsampled[0].count, 5);
    assert.equal(downsampled[0].startDate, '2024-01-01');
    assert.equal(downsampled[0].endDate, '2024-01-05');
  });

  it('handles empty and single-point lists gracefully', () => {
    assert.deepEqual(downsampleTrendPoints([], 700, 70, 10), []);

    const single = [{ date: '2024-01-01', total_value: 500.0 }];
    const res = downsampleTrendPoints(single, 700, 70, 10);
    assert.equal(res.length, 1);
    assert.equal(res[0].total_value, 500.0);
    assert.equal(res[0].x, 70 + 350);
  });
});
