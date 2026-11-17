import { describe, it } from 'node:test';
import assert from 'node:assert';
import {
  READ_ONLY_FEATURE,
  MAINTENANCE_EDIT_WARNING,
  isReadOnlyFeatureEnabled,
  isReadOnlyErrorResponse,
  subscribeMaintenanceWarning,
  notifyMaintenanceWarning,
} from './maintenance.ts';

describe('Maintenance & Feature Flag Helpers', () => {
  it('detects read_only feature flag cleanly', () => {
    assert.strictEqual(isReadOnlyFeatureEnabled([READ_ONLY_FEATURE]), true);
    assert.strictEqual(isReadOnlyFeatureEnabled([]), false);
    assert.strictEqual(isReadOnlyFeatureEnabled(['other_flag']), false);
  });

  it('detects read-only maintenance error responses cleanly', () => {
    assert.strictEqual(isReadOnlyErrorResponse(503, {}), true);
    assert.strictEqual(isReadOnlyErrorResponse(409, { error: 'READ_ONLY_MODE' }), true);
    assert.strictEqual(isReadOnlyErrorResponse(409, { error: 'RECORD_RECONCILED' }), false);
    assert.strictEqual(isReadOnlyErrorResponse(400, { error: 'BAD_REQUEST' }), false);
  });

  it('publishes and unsubscribes maintenance warnings', () => {
    const received: string[] = [];
    const unsubscribe = subscribeMaintenanceWarning((msg) => {
      received.push(msg);
    });

    notifyMaintenanceWarning();
    assert.deepStrictEqual(received, [MAINTENANCE_EDIT_WARNING]);

    unsubscribe();
    notifyMaintenanceWarning('Should not be received');
    assert.deepStrictEqual(received, [MAINTENANCE_EDIT_WARNING]);
  });
});
