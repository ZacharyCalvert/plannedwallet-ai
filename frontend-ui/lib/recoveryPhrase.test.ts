import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildRecoveryKitContent,
  pickSpotCheckIndices,
  splitRecoveryPhrase,
  verifySpotCheckAnswers,
} from './recoveryPhrase.ts';

describe('Recovery Phrase & Anti-Skip Spot Check Helpers', () => {
  const samplePhrase =
    'aback beacon canyon domain emblem harbor ivory lantern meadow quartz summit timber';

  it('splits and normalizes 12-word mnemonic phrases', () => {
    const words = splitRecoveryPhrase(`  ABACK  beacon \n canyon   domain emblem harbor ivory lantern meadow quartz summit TIMBER `);
    assert.equal(words.length, 12);
    assert.equal(words[0], 'aback');
    assert.equal(words[11], 'timber');
  });

  it('picks two distinct ascending 1-based spot-check indices', () => {
    for (let i = 0; i < 25; i += 1) {
      const [a, b] = pickSpotCheckIndices(12);
      assert.ok(a >= 1 && a <= 12);
      assert.ok(b >= 1 && b <= 12);
      assert.ok(a < b);
    }
  });

  it('verifies spot-check answers case-insensitively and rejects wrong or out-of-range inputs', () => {
    const words = splitRecoveryPhrase(samplePhrase);
    assert.equal(verifySpotCheckAnswers(words, [2, 9], ['  BEACON ', 'meadow']), true);
    assert.equal(verifySpotCheckAnswers(words, [2, 9], ['beacon', 'wrong']), false);
    assert.equal(verifySpotCheckAnswers(words, [3, 3], ['canyon', 'canyon']), false);
    assert.equal(verifySpotCheckAnswers(words, [0, 2], ['aback', 'beacon']), false);
  });

  it('formats printable recovery kit text with numbered words and username', () => {
    const kit = buildRecoveryKitContent('alice', samplePhrase, '2026-11-18T14:00:00Z');
    assert.ok(kit.includes('Username: alice'));
    assert.ok(kit.includes(' 1. aback'));
    assert.ok(kit.includes('12. timber'));
  });
});
