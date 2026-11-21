import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { countDigits, hasAccountDigitsWarning } from './accountValidation.ts';

describe('Account Validation Helpers', () => {
  it('counts decimal digits accurately across text and punctuation', () => {
    assert.strictEqual(countDigits('Checking'), 0);
    assert.strictEqual(countDigits('Checking 123'), 3);
    assert.strictEqual(countDigits('Card 1234'), 4);
    assert.strictEqual(countDigits('Acct 12-345'), 5);
    assert.strictEqual(countDigits('4111-2222-3333-4444'), 16);
  });

  it('triggers warning banner at 4 or more digits', () => {
    assert.strictEqual(hasAccountDigitsWarning('Primary Checking'), false);
    assert.strictEqual(hasAccountDigitsWarning('Checking 123'), false);
    assert.strictEqual(hasAccountDigitsWarning('Checking 1234'), true);
    assert.strictEqual(hasAccountDigitsWarning('Routing 12345'), true);
  });
});
