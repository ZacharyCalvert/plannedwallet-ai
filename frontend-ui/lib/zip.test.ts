import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { crc32, createZipArchive } from './zip.ts';

describe('ZIP Archive Helpers', () => {
  it('computes standard IEEE 802.3 CRC-32', () => {
    const bytes = new TextEncoder().encode('123456789');
    assert.strictEqual(crc32(bytes), 0xcbf43926);
  });

  it('builds a valid PKZIP archive with local headers, central directory, and EOCD', () => {
    const archive = createZipArchive(
      [
        { filename: 'checking.csv', data: 'date_sent,value\n2026-11-01,-10.00\n' },
        { filename: 'savings.csv', data: 'date_sent,value\n2026-11-02,500.00\n' },
      ],
      new Date('2026-11-21T12:00:00Z')
    );

    const view = new DataView(archive.buffer, archive.byteOffset, archive.byteLength);
    // Local file header signature at byte 0
    assert.strictEqual(view.getUint32(0, true), 0x04034b50);

    // End of central directory signature at the tail (22 bytes from end)
    const eocdOffset = archive.length - 22;
    assert.strictEqual(view.getUint32(eocdOffset, true), 0x06054b50);
    assert.strictEqual(view.getUint16(eocdOffset + 8, true), 2);
    assert.strictEqual(view.getUint16(eocdOffset + 10, true), 2);

    const text = new TextDecoder().decode(archive);
    assert.ok(text.includes('checking.csv'));
    assert.ok(text.includes('savings.csv'));
    assert.ok(text.includes('2026-11-01,-10.00'));
  });
});
