import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  clampPasswordToMaxBytes,
  MAX_PASSWORD_BYTES,
  utf8ByteLength,
} from './passwordLimit.ts';

describe('Password UTF-8 Byte Limit Helpers', () => {
  it('preserves passwords at or under 72 UTF-8 bytes', () => {
    const exactAscii = 'A1!' + 'a'.repeat(69);
    assert.equal(utf8ByteLength(exactAscii), MAX_PASSWORD_BYTES);
    assert.equal(clampPasswordToMaxBytes(exactAscii), exactAscii);
  });

  it('clamps ASCII passwords exceeding 72 bytes to 72 bytes', () => {
    const tooLong = 'A1!' + 'b'.repeat(80);
    const clamped = clampPasswordToMaxBytes(tooLong);
    assert.equal(utf8ByteLength(clamped), MAX_PASSWORD_BYTES);
    assert.equal(clamped, 'A1!' + 'b'.repeat(69));
  });

  it('clamps multi-byte UTF-8 characters without splitting surrogate pairs or exceeding 72 bytes', () => {
    // 40 x 2-byte characters = 80 UTF-8 bytes -> clamps cleanly to 36 chars (72 bytes)
    const accented = 'é'.repeat(40);
    const clampedAccented = clampPasswordToMaxBytes(accented);
    assert.equal(utf8ByteLength(clampedAccented), 72);
    assert.equal(Array.from(clampedAccented).length, 36);

    // 19 x 4-byte emoji = 76 UTF-8 bytes -> clamps cleanly to 18 emoji (72 bytes)
    const emojis = '🔒'.repeat(19);
    const clampedEmoji = clampPasswordToMaxBytes(emojis);
    assert.equal(utf8ByteLength(clampedEmoji), 72);
    assert.equal(Array.from(clampedEmoji).length, 18);
  });
});
