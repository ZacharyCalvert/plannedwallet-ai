import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { BULK_UPLOAD_BATCH_SIZE, chunkItemsForBulkUpload } from './bulkUpload.ts';

describe('Bulk Upload Helpers', () => {
  it('chunks 4890 records into 5 batches of 1000, 1000, 1000, 1000, and 890', () => {
    assert.equal(BULK_UPLOAD_BATCH_SIZE, 1000);
    const items = Array.from({ length: 4890 }, (_, i) => i + 1);
    const batches = chunkItemsForBulkUpload(items);

    assert.equal(batches.length, 5);
    assert.deepEqual(
      batches.map((b) => b.length),
      [1000, 1000, 1000, 1000, 890]
    );
    assert.equal(batches[0][0], 1);
    assert.equal(batches[4][889], 4890);
  });

  it('handles empty arrays and exact multiples of 1000 cleanly', () => {
    assert.deepEqual(chunkItemsForBulkUpload([]), []);
    const exact = Array.from({ length: 2000 }, (_, i) => i);
    const batches = chunkItemsForBulkUpload(exact);
    assert.equal(batches.length, 2);
    assert.deepEqual(
      batches.map((b) => b.length),
      [1000, 1000]
    );
  });
});
