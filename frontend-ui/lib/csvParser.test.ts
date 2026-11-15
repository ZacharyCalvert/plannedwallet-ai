import { describe, it } from 'node:test';
import assert from 'node:assert';
import { parseAndValidateCsv, parseDate, parseAmount, matchHeaders } from './csvParser.ts';

describe('CSV Parser Unit Tests', () => {
  it('correctly processes demonstration file yielding 2 successes and 2 record issues', () => {
    const demonstrationFile = `Balance,AMount,Date Sent,Date Cleared,VENDOR,Notes,
-$300.00,-$300.00,11/20/2022,12/1/2022,Tarrant Food Bank,,
"-$3,399.17",-$10.77,3/2/0204,3/1/2024,Starbucks,,
-$430.39,-$130.39,11/20/2022,12/1/2022,Mi Cocina,,
"-$3,982.93",-$54.12,9/22/0203,10/1/2023,Kuckus Nest,Octoberfest,
`;

    const result = parseAndValidateCsv(demonstrationFile);

    // Header & Overall assertions
    assert.strictEqual(result.success, true);
    assert.strictEqual(result.validCount, 2, 'Expected exactly 2 valid records');
    assert.strictEqual(result.invalidCount, 2, 'Expected exactly 2 invalid records');
    assert.strictEqual(result.emptyCount, 0, 'Expected 0 empty records');
    assert.strictEqual(result.rows.length, 4, 'Expected 4 total records in review');

    // Invalid records must be sorted to the top
    const [rowA, rowB, rowC, rowD] = result.rows;

    // Both top rows must be invalid
    assert.strictEqual(rowA.isValid, false, 'First row in review must be invalid');
    assert.strictEqual(rowB.isValid, false, 'Second row in review must be invalid');

    // Bottom rows must be valid
    assert.strictEqual(rowC.isValid, true, 'Third row in review must be valid');
    assert.strictEqual(rowD.isValid, true, 'Fourth row in review must be valid');

    // Validate the specific invalid rows (Row 3 and Row 5 with bad year formatting)
    const invalidRows = [rowA, rowB];
    const row3 = invalidRows.find((r) => r.rowNumber === 3);
    assert.ok(row3, 'Row 3 should be marked invalid');
    assert.ok(
      row3.errors.some((e) => e.includes('Date Sent')),
      'Row 3 must fail Date Sent validation due to 0204 year format'
    );
    assert.strictEqual(row3.amount, -10.77);

    const row5 = invalidRows.find((r) => r.rowNumber === 5);
    assert.ok(row5, 'Row 5 should be marked invalid');
    assert.ok(
      row5.errors.some((e) => e.includes('Date Sent')),
      'Row 5 must fail Date Sent validation due to 0203 year format'
    );
    assert.strictEqual(row5.amount, -54.12);

    // Validate the successful rows (Row 2 and Row 4)
    const validRows = [rowC, rowD];
    const row2 = validRows.find((r) => r.rowNumber === 2);
    assert.ok(row2, 'Row 2 should be valid');
    assert.strictEqual(row2.dateSent, '2022-11-20');
    assert.strictEqual(row2.dateCleared, '2022-12-01');
    assert.strictEqual(row2.amount, -300.0);
    assert.strictEqual(row2.recipient, 'Tarrant Food Bank');

    const row4 = validRows.find((r) => r.rowNumber === 4);
    assert.ok(row4, 'Row 4 should be valid');
    assert.strictEqual(row4.dateSent, '2022-11-20');
    assert.strictEqual(row4.dateCleared, '2022-12-01');
    assert.strictEqual(row4.amount, -130.39);
    assert.strictEqual(row4.recipient, 'Mi Cocina');
  });

  it('rejects 3-digit and invalid years in date parser', () => {
    assert.strictEqual(parseDate('11/15/202').dateStr, null);
    assert.strictEqual(parseDate('05/10/024').dateStr, null);
    assert.strictEqual(parseDate('3/2/0204').dateStr, null);
    assert.strictEqual(parseDate('9/22/0203').dateStr, null);
    assert.strictEqual(parseDate('11/15/2026').dateStr, '2026-11-15');
    assert.strictEqual(parseDate('11/15/26').dateStr, '2026-11-15');
  });

  it('validates optional cleared dates and descriptions', () => {
    const csvWithBadOptional = `Date Sent,Amount,Recipient,Date Cleared,Description
11/20/2022,-50.00,Supermarket,02/29/2023,Normal grocery
11/20/2022,-20.00,Corner Store,11/21/2022,${'A'.repeat(501)}
11/20/2022,-30.00,Pharmacy,,Valid pending
`;
    const res = parseAndValidateCsv(csvWithBadOptional);
    assert.strictEqual(res.validCount, 1);
    assert.strictEqual(res.invalidCount, 2);

    const badDateCleared = res.rows.find((r) => r.rowNumber === 2);
    assert.ok(badDateCleared && !badDateCleared.isValid);
    assert.ok(badDateCleared.errors.some((e) => e.includes('Date Cleared')));

    const badDesc = res.rows.find((r) => r.rowNumber === 3);
    assert.ok(badDesc && !badDesc.isValid);
    assert.ok(badDesc.errors.some((e) => e.includes('Description')));

    const validRow = res.rows.find((r) => r.rowNumber === 4);
    assert.ok(validRow && validRow.isValid);
    assert.strictEqual(validRow.dateCleared, null);
  });

  it('rejects financial records older than the year 2000', () => {
    // Dates before 2000 are rejected
    const older1 = parseDate('12/31/1999');
    assert.strictEqual(older1.dateStr, null);
    assert.ok(older1.error?.includes('older than year 2000'));

    const older2 = parseDate('1999-12-31');
    assert.strictEqual(older2.dateStr, null);
    assert.ok(older2.error?.includes('older than year 2000'));

    const older3 = parseDate('05/10/1985');
    assert.strictEqual(older3.dateStr, null);
    assert.ok(older3.error?.includes('older than year 2000'));

    // 2-digit 99 maps to 1999 -> rejected
    const older4 = parseDate('11/20/99');
    assert.strictEqual(older4.dateStr, null);
    assert.ok(older4.error?.includes('older than year 2000'));

    // Year 2000 onwards is accepted
    const y2k = parseDate('01/01/2000');
    assert.strictEqual(y2k.dateStr, '2000-01-01');

    const y2kIso = parseDate('2000-01-01');
    assert.strictEqual(y2kIso.dateStr, '2000-01-01');

    const recent = parseDate('11/20/2026');
    assert.strictEqual(recent.dateStr, '2026-11-20');

    // Entire CSV row with pre-2000 record is marked invalid
    const pre2000Csv = `Date Sent,Amount,Recipient
12/31/1999,-25.00,Bookstore
01/01/2000,-15.00,Pharmacy
`;
    const res = parseAndValidateCsv(pre2000Csv);
    assert.strictEqual(res.validCount, 1);
    assert.strictEqual(res.invalidCount, 1);
    const invalidRow = res.rows.find((r) => r.rowNumber === 2);
    assert.ok(invalidRow && !invalidRow.isValid);
    assert.ok(invalidRow.errors.some((e) => e.includes('older than year 2000')));
  });

  it('enforces maximum limit of 10,000 records per bulk upload', () => {
    // 10,001 rows should be rejected
    const rows: string[] = ['Date Sent,Amount,Recipient'];
    for (let i = 0; i < 10001; i++) {
      rows.push('11/20/2026,-10.00,Corner Grocery');
    }
    const overflowCsv = rows.join('\n');
    const overflowRes = parseAndValidateCsv(overflowCsv);
    assert.strictEqual(overflowRes.success, false);
    assert.ok(overflowRes.error?.includes('10,000'));
    assert.ok(overflowRes.error?.includes('exceeding the maximum limit'));

    // 5 rows should pass without limit issues
    const smallRows: string[] = ['Date Sent,Amount,Recipient'];
    for (let i = 0; i < 5; i++) {
      smallRows.push('11/20/2026,-10.00,Corner Grocery');
    }
    const smallRes = parseAndValidateCsv(smallRows.join('\n'));
    assert.strictEqual(smallRes.success, true);
    assert.strictEqual(smallRes.validCount, 5);
  });

  it('supports spreadsheet currency formatting with $ and parentheses', () => {
    // Specific user requested examples:
    // $(105.10) is equivalent to -105.10
    // $105.10 is equivalent to 105.10
    assert.strictEqual(parseAmount('$(105.10)').value, -105.1);
    assert.strictEqual(parseAmount('$105.10').value, 105.1);

    // Additional common spreadsheet accounting variations
    assert.strictEqual(parseAmount('($105.10)').value, -105.1);
    assert.strictEqual(parseAmount('(105.10)').value, -105.1);
    assert.strictEqual(parseAmount('-$105.10').value, -105.1);
    assert.strictEqual(parseAmount('$-105.10').value, -105.1);
    assert.strictEqual(parseAmount('-$3,399.17').value, -3399.17);
    assert.strictEqual(parseAmount('$ (105.10)').value, -105.1);
    assert.strictEqual(parseAmount('($ 105.10)').value, -105.1);
    assert.strictEqual(parseAmount('+$105.10').value, 105.1);
    assert.strictEqual(parseAmount('$0.00').value, 0);
    assert.strictEqual(parseAmount('$(0.00)').value, 0);

    // Full CSV test with spreadsheet money formats
    const moneyCsv = `Date Sent,Recipient,Amount
11/20/2026,Corner Grocery,$(105.10)
11/20/2026,Employer Payroll,$105.10
11/20/2026,Coffee Shop,($15.25)
11/20/2026,Bookstore,"-$3,399.17"
`;
    const res = parseAndValidateCsv(moneyCsv);
    assert.strictEqual(res.success, true);
    assert.strictEqual(res.validCount, 4);
    assert.strictEqual(res.rows.find((r) => r.rowNumber === 2)?.amount, -105.1);
    assert.strictEqual(res.rows.find((r) => r.rowNumber === 3)?.amount, 105.1);
    assert.strictEqual(res.rows.find((r) => r.rowNumber === 4)?.amount, -15.25);
    assert.strictEqual(res.rows.find((r) => r.rowNumber === 5)?.amount, -3399.17);
  });
});
