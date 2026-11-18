import { describe, it } from 'node:test';
import assert from 'node:assert';
import type { RecurringTransaction } from './types.ts';
import {
  formatRecurringIntervalLabel,
  getBillDaysUntilDue,
  getBillDisplayName,
  isValidBillIntervalInput,
  normalizeNegativeBillAmountInput,
  normalizePositiveBillIntervalInput,
  normalizePositiveIncomeAmountInput,
  shouldShowBillReminder,
  formatBillScheduleLabel,
  parseBillAmount,
} from './bills.ts';

function makeRecurring(overrides: Partial<RecurringTransaction> = {}): RecurringTransaction {
  return {
    id: 'rt-1',
    user_id: 'u-1',
    account_id: 'a-1',
    category_id: null,
    amount: -125.0,
    last_date_applied: '2026-10-17',
    frequency: 'monthly',
    interval: 1,
    schedule: null,
    name: 'Electric',
    entity: 'City Electric Utility',
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
      shouldShowBillReminder(makeRecurring({ reminder: false, next_due_date: '2026-11-01' }), today),
      false
    );

    // Any overdue item with reminder=true is shown
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'weekly', next_due_date: '2026-11-16' }), today),
      true
    );

    // Weekly: <= 2 days (48h) shown, 3 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'weekly', next_due_date: '2026-11-19' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'weekly', next_due_date: '2026-11-20' }), today),
      false
    );

    // Bi-monthly: <= 3 days shown, 4 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'bi-monthly', next_due_date: '2026-11-20' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'bi-monthly', next_due_date: '2026-11-21' }), today),
      false
    );

    // Monthly: <= 7 days (1 week) shown, 8 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'monthly', next_due_date: '2026-11-24' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'monthly', next_due_date: '2026-11-25' }), today),
      false
    );

    // Yearly: <= 30 days shown, 31 days not shown
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'yearly', next_due_date: '2026-12-17' }), today),
      true
    );
    assert.strictEqual(
      shouldShowBillReminder(makeRecurring({ frequency: 'yearly', next_due_date: '2026-12-18' }), today),
      false
    );
  });

  it('formats human-readable schedule descriptions and parses currency amounts', () => {
    assert.strictEqual(
      formatBillScheduleLabel(makeRecurring({ frequency: 'monthly', interval: 6, schedule: null })),
      'Every 6 months (from last paid)'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeRecurring({
          frequency: 'monthly',
          schedule: { type: 'nth_weekday', scope: 'month', nth: -1, weekday: 'monday' },
        })
      ),
      'Last Monday of the month'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeRecurring({
          frequency: 'yearly',
          schedule: { type: 'month_day', scope: 'year', month: 4, day: 14 },
        })
      ),
      'April 14th every year'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeRecurring({
          frequency: 'bi-monthly',
          schedule: { type: 'first_fifteenth', scope: 'bi-monthly' },
        })
      ),
      '1st and 15th of the month'
    );
    assert.strictEqual(
      formatBillScheduleLabel(
        makeRecurring({
          frequency: 'bi-monthly',
          schedule: { type: 'fifteenth_last', scope: 'bi-monthly' },
        })
      ),
      '15th and last day of the month'
    );
    assert.strictEqual(parseBillAmount('$(1,250.50)'), -1250.5);
    assert.strictEqual(parseBillAmount('$95.00'), 95);
    assert.strictEqual(getBillDisplayName(makeRecurring({ name: 'Electric', entity: 'Regional Utility' })), 'Electric');
    assert.strictEqual(getBillDisplayName(makeRecurring({ name: '', entity: 'Regional Utility' })), 'Regional Utility');
    assert.strictEqual(normalizeNegativeBillAmountInput('125.50'), '-125.50');
    assert.strictEqual(normalizeNegativeBillAmountInput('1,250.00'), '-1,250.00');
    assert.strictEqual(normalizeNegativeBillAmountInput('-88.25'), '-88.25');
    assert.strictEqual(normalizeNegativeBillAmountInput('0.00'), '0.00');
    assert.strictEqual(normalizeNegativeBillAmountInput(''), '');
    assert.strictEqual(normalizePositiveIncomeAmountInput('-2500.00'), '2500.00');
    assert.strictEqual(normalizePositiveIncomeAmountInput('2500.00'), '2500.00');
    assert.strictEqual(normalizePositiveIncomeAmountInput('0.00'), '0.00');
    assert.strictEqual(getBillDisplayName({ name: 'Paycheck', entity: 'Employer Payroll' }), 'Paycheck');
    assert.strictEqual(getBillDisplayName({ name: '', entity: 'Employer Payroll' }), 'Employer Payroll');
    assert.strictEqual(isValidBillIntervalInput('6'), true);
    assert.strictEqual(isValidBillIntervalInput(''), true);
    assert.strictEqual(isValidBillIntervalInput('0'), false);
    assert.strictEqual(isValidBillIntervalInput('-10'), false);
    assert.strictEqual(normalizePositiveBillIntervalInput(''), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('-10'), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('0'), '1');
    assert.strictEqual(normalizePositiveBillIntervalInput('6'), '6');
    assert.strictEqual(formatRecurringIntervalLabel('monthly', '1'), 'Every 1 month');
    assert.strictEqual(formatRecurringIntervalLabel('monthly', '6'), 'Every 6 months');
    assert.strictEqual(formatRecurringIntervalLabel('yearly', '1'), 'Every 1 year');
    assert.strictEqual(formatRecurringIntervalLabel('yearly', '2'), 'Every 2 years');
    assert.strictEqual(formatRecurringIntervalLabel('weekly', '1'), 'Every 1 week');
    assert.strictEqual(formatRecurringIntervalLabel('weekly', '2'), 'Every 2 weeks');
    assert.strictEqual(formatRecurringIntervalLabel('bi-monthly', '1'), 'Twice a month');
  });
});
