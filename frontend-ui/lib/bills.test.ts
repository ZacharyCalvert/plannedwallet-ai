import { describe, it } from 'node:test';
import assert from 'node:assert';
import type { Bill } from './types.ts';
import {
  getBillDaysUntilDue,
  getBillDisplayName,
  isValidBillIntervalInput,
  normalizeNegativeBillAmountInput,
  normalizePositiveBillIntervalInput,
  shouldShowBillReminder,
  formatBillScheduleLabel,
  parseBillAmount,
} from './bills.ts';

function makeBill(overrides: Partial<Bill> = {}): Bill {
  return {
    id: 'b-1',
    user_id: 'u-1',
    account_id: 'a-1',
    category_id: null,
    amount: -125.0,
    last_date_paid: '2026-10-17',
    frequency: 'monthly',
    interval: 1,
    schedule: null,
    name: 'Electric',
    recipient: 'City Electric Utility',
    description: 'Monthly service',
    reminder: true,
    next_due_date: '2026-11-17',
    due_status: 'upcoming',
    days_until_due: 0,
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
    ...overrides,
  };
}

describe('Bills Helpers', () => {
  it('computes calendar day differences cleanly without DST drift', () => {
    assert.strictEqual(getBillDaysUntilDue('2026-11-17', '2026-11-17'), 0);
    assert.strictEqual(getBillDaysUntilDue('2026-11-16', '2026-11-17'), -1);
    assert.strictEqual(getBillDaysUntilDue('2026-11-19', '2026-11-17'), 2);
    assert.strictEqual(getBillDaysUntilDue(null, '2026-11-17'), null);
  });

  it('enforces reminder visibility windows for weekly (48h), monthly (1w), yearly (30d), and overdue bills', () => {
    const today = '2026-11-17';

    // Reminder flag false -> never shown, even when overdue
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ reminder: false, next_due_date: '2026-11-01' }), today),
      false
    );

    // Any overdue bill with reminder=true is shown
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'weekly', next_due_date: '2026-11-16' }), today),
      true
    );

    // Weekly: <= 2 days (48h) shown, 3 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'weekly', next_due_date: '2026-11-19' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'weekly', next_due_date: '2026-11-20' }), today),
      false
    );

    // Monthly: <= 7 days (1 week) shown, 8 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'monthly', next_due_date: '2026-11-24' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'monthly', next_due_date: '2026-11-25' }), today),
      false
    );

    // Yearly: <= 30 days shown, 31 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'yearly', next_due_date: '2026-12-17' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeBill({ frequency: 'yearly', next_due_date: '2026-12-18' }), today),
      false
    );
  });

  it('formats human-readable schedule descriptions and parses currency amounts', () => {
    assert.strictEqual(
      formatBillScheduleLabel(makeBill({ frequency: 'monthly', interval: 6, schedule: null })),
      'Every 6 months (from last paid)'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeBill({
          frequency: 'monthly',
          schedule: { type: 'nth_weekday', scope: 'month', nth: -1, weekday: 'monday' },
        })
      ),
      'Last Monday of the month'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeBill({
          frequency: 'yearly',
          schedule: { type: 'month_day', scope: 'year', month: 4, day: 14 },
        })
      ),
      'April 14th every year'
    );
    assert.strictEqual(parseBillAmount('$(1,250.50)'), -1250.5);
    assert.strictEqual(parseBillAmount('$95.00'), 95);
    assert.strictEqual(getBillDisplayName(makeBill({ name: 'Electric', recipient: 'Regional Utility' })), 'Electric');
    assert.strictEqual(getBillDisplayName(makeBill({ name: '', recipient: 'Regional Utility' })), 'Regional Utility');
    assert.strictEqual(normalizeNegativeBillAmountInput('125.50'), '-125.50');
    assert.strictEqual(normalizeNegativeBillAmountInput('1,250.00'), '-1,250.00');
    assert.strictEqual(normalizeNegativeBillAmountInput('-88.25'), '-88.25');
    assert.strictEqual(normalizeNegativeBillAmountInput('0.00'), '0.00');
    assert.strictEqual(normalizeNegativeBillAmountInput(''), '');
    assert.strictEqual(isValidBillIntervalInput('6'), true);
    assert.strictEqual(isValidBillIntervalInput(''), true);
    assert.strictEqual(isValidBillIntervalInput('0'), false);
    assert.strictEqual(isValidBillIntervalInput('-10'), false);
    assert.strictEqual(normalizePositiveBillIntervalInput(''), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('-10'), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('0'), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('6'), '6');
  });
});
