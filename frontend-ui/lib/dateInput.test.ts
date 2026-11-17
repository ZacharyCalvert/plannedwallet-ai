import { describe, it } from 'node:test';
import assert from 'node:assert';
import {
  isoToDisplayDate,
  displayDateToIso,
  applyDateAutoSlash,
  computeSlashKeyAction,
} from './dateInput.ts';

describe('DateInput Helpers', () => {
  it('auto-expands 2-digit years other than "20" while keeping "20" as 4-digit prefix', () => {
    assert.strictEqual(applyDateAutoSlash('11/01/20', '11/01/2'), '11/01/20');
    assert.strictEqual(applyDateAutoSlash('11/01/26', '11/01/2'), '11/01/2026');
    assert.strictEqual(applyDateAutoSlash('11/01/19', '11/01/1'), '11/01/2019');
    // Does not re-expand when backspacing
    assert.strictEqual(applyDateAutoSlash('11/01/26', '11/01/260'), '11/01/26');

    assert.strictEqual(displayDateToIso('10/01/20'), '10/01/20');
    assert.strictEqual(displayDateToIso('11/01/26'), '2026-11-01');
    assert.strictEqual(displayDateToIso('10/01/202'), '10/01/202');
    assert.strictEqual(displayDateToIso('10/01/2026'), '2026-10-01');
    assert.strictEqual(isoToDisplayDate('2026-10-01'), '10/01/2026');
    assert.strictEqual(isoToDisplayDate('2026-10-01T12:00:00Z'), '10/01/2026');
  });

  it('auto-slashes after 2-digit month and day without breaking single slash transition', () => {
    const afterTwoDigits = applyDateAutoSlash('11', '1');
    assert.strictEqual(afterTwoDigits, '11/');

    // Pressing slash immediately after "11/" stays in DD instead of inserting "01/"
    assert.deepStrictEqual(computeSlashKeyAction('11/', 3, 3), { action: 'ignore' });

    // Typing day and then slash transitions to YYYY
    assert.deepStrictEqual(computeSlashKeyAction('11/5', 4, 4), {
      action: 'update',
      value: '11/05/',
      cursorPos: 6,
    });

    // Pressing slash immediately after auto-slashed "11/05/" stays in YYYY instead of jumping fields
    const afterDay = applyDateAutoSlash('11/05', '11/0');
    assert.strictEqual(afterDay, '11/05/');
    assert.deepStrictEqual(computeSlashKeyAction('11/05/', 6, 6), { action: 'ignore' });

    // Typing single-digit month followed by slash pads and transitions to DD
    assert.deepStrictEqual(computeSlashKeyAction('1', 1, 1), {
      action: 'update',
      value: '01/',
      cursorPos: 3,
    });

    // Pressing slash after entering the year focuses next field
    assert.deepStrictEqual(computeSlashKeyAction('11/05/2026', 10, 10), {
      action: 'focusNext',
    });
  });
});
