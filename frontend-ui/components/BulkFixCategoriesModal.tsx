'use client';

import { useEffect, useMemo, useState } from 'react';
import { AccountRecord, Category } from '../lib/types';
import { patchAccountRecord } from '../lib/api';
import {
  bulkAssignCategoryToRecords,
  getUncategorizedRecordsForVendor,
  summarizeUncategorizedVendors,
} from '../lib/accountCategories';

interface BulkFixCategoriesModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  accountId: string;
  records: AccountRecord[];
  categories: Category[];
  onRecordsUpdated: (updatedRecords: AccountRecord[]) => void;
}

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
}

export function BulkFixCategoriesModal({
  isOpen,
  onClose,
  token,
  accountId,
  records,
  categories,
  onRecordsUpdated,
}: BulkFixCategoriesModalProps) {
  const [vendorName, setVendorName] = useState('');
  const [targetCategoryId, setTargetCategoryId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [progress, setProgress] = useState<{ done: number; total: number }>({ done: 0, total: 0 });
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  useEffect(() => {
    if (isOpen) {
      setVendorName('');
      setTargetCategoryId('');
      setError('');
      setSuccessMessage('');
      setProgress({ done: 0, total: 0 });
    }
  }, [isOpen]);

  const vendorSummaries = useMemo(
    () => summarizeUncategorizedVendors(records),
    [records]
  );

  const filteredVendors = useMemo(() => {
    const q = vendorName.trim().toLowerCase();
    if (!q) return vendorSummaries;
    return vendorSummaries.filter((v) => v.displayName.toLowerCase().includes(q));
  }, [vendorSummaries, vendorName]);

  const matchingRecords = useMemo(
    () => getUncategorizedRecordsForVendor(records, vendorName),
    [records, vendorName]
  );

  const defaultCategories = useMemo(
    () => categories.filter((c) => c.is_default),
    [categories]
  );

  const customCategories = useMemo(
    () => categories.filter((c) => !c.is_default),
    [categories]
  );

  if (!isOpen) return null;

  const handleApply = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMessage('');

    if (matchingRecords.length === 0) {
      setError('Select or enter an exact vendor/payer with uncategorized records.');
      return;
    }
    if (!targetCategoryId) {
      setError('Please select a target category.');
      return;
    }

    const selectedCategory = categories.find((c) => c.id === targetCategoryId);
    const appliedVendor = matchingRecords[0].entity.trim();
    setSubmitting(true);
    setProgress({ done: 0, total: matchingRecords.length });

    try {
      const updated = await bulkAssignCategoryToRecords(
        matchingRecords,
        targetCategoryId,
        async (_accId, recId, catId) =>
          patchAccountRecord(token, accountId, recId, { category_id: catId }),
        (done, total) => setProgress({ done, total })
      );
      onRecordsUpdated(updated);
      setVendorName('');
      setTargetCategoryId('');
      setSuccessMessage(
        `Updated ${updated.length} uncategorized ${updated.length === 1 ? 'record' : 'records'} for "${appliedVendor}" to "${selectedCategory?.name || 'selected category'}".`
      );
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to bulk update categories.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={() => !submitting && onClose()}>
      <div
        className="modal-content bulk-fix-categories-modal"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label="Bulk Fix Categories by Vendor or Recipient"
      >
        <div className="modal-header">
          <div>
            <h2 className="modal-title">Bulk Fix Categories</h2>
            <p className="modal-subtitle">
              Assign a category to all uncategorized transactions (money out or money in) for a specific vendor or payer.
            </p>
          </div>
          {!submitting && (
            <button type="button" className="btn-close" onClick={onClose} aria-label="Close">
              &times;
            </button>
          )}
        </div>

        <div className="modal-body">
          {error && <div className="alert-error mb-3">{error}</div>}
          {successMessage && <div className="alert-success mb-3">{successMessage}</div>}

          {vendorSummaries.length === 0 ? (
            <div className="empty-state">
              All records in this account already have a category assigned.
            </div>
          ) : (
            <form onSubmit={handleApply}>
              <div className="form-group">
                <label htmlFor="bulk-fix-vendor-input">Vendor / Payer Name</label>
                <input
                  id="bulk-fix-vendor-input"
                  type="text"
                  value={vendorName}
                  onChange={(e) => {
                    setVendorName(e.target.value);
                    setError('');
                  }}
                  placeholder="Type or click a vendor or payer below (e.g. Corner Grocery)..."
                  disabled={submitting}
                />
              </div>

              <div className="bulk-fix-vendor-list" aria-label="Uncategorized vendors and payers">
                {filteredVendors.length === 0 ? (
                  <div className="text-muted text-sm p-2">
                    No uncategorized vendors or payers match &ldquo;{vendorName}&rdquo;.
                  </div>
                ) : (
                  filteredVendors.map((v) => {
                    const isSelected =
                      v.key === vendorName.trim().toLowerCase();
                    return (
                      <button
                        key={v.key}
                        type="button"
                        className={`bulk-fix-vendor-row ${isSelected ? 'selected' : ''}`}
                        onClick={() => {
                          setVendorName(v.displayName);
                          setError('');
                        }}
                        disabled={submitting}
                      >
                        <span className="font-semibold">{v.displayName}</span>
                        <span className="text-muted text-sm">
                          {v.count} {v.count === 1 ? 'record' : 'records'} &bull;{' '}
                          {v.netAmount >= 0 ? '+' : '-'}
                          {formatCurrency(v.netAmount)}
                        </span>
                      </button>
                    );
                  })
                )}
              </div>

              {matchingRecords.length > 0 && (
                <div className="bulk-fix-preview mt-3">
                  <div className="text-sm font-semibold mb-1">
                    Matching uncategorized records ({matchingRecords.length}):
                  </div>
                  <ul className="bulk-fix-preview-list">
                    {matchingRecords.slice(0, 5).map((r) => (
                      <li key={r.id}>
                        <span>{r.date_sent}</span>
                        <span>{r.entity}</span>
                        <span>{r.description || '—'}</span>
                        <span className={r.value >= 0 ? 'text-success' : 'text-danger'}>
                          {r.value >= 0 ? '+' : '-'}
                          {formatCurrency(r.value)}
                        </span>
                      </li>
                    ))}
                  </ul>
                  {matchingRecords.length > 5 && (
                    <div className="text-muted text-sm mt-1">
                      ...and {matchingRecords.length - 5} more
                    </div>
                  )}
                </div>
              )}

              <div className="form-group mt-3">
                <label htmlFor="bulk-fix-category-select">Assign Category</label>
                <select
                  id="bulk-fix-category-select"
                  value={targetCategoryId}
                  onChange={(e) => setTargetCategoryId(e.target.value)}
                  disabled={submitting}
                >
                  <option value="">-- Select category --</option>
                  {defaultCategories.length > 0 && (
                    <optgroup label="Default Categories">
                      {defaultCategories.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </optgroup>
                  )}
                  {customCategories.length > 0 && (
                    <optgroup label="Custom Categories">
                      {customCategories.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </optgroup>
                  )}
                </select>
              </div>

              {submitting && progress.total > 0 && (
                <p className="text-muted text-sm mt-2">
                  Updating {progress.done} of {progress.total} records...
                </p>
              )}

              <div className="modal-actions mt-4">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={onClose}
                  disabled={submitting}
                >
                  Close
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={submitting || matchingRecords.length === 0 || !targetCategoryId}
                >
                  {submitting
                    ? `Updating (${progress.done}/${progress.total})...`
                    : `Apply to ${matchingRecords.length} ${matchingRecords.length === 1 ? 'Record' : 'Records'}`}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
