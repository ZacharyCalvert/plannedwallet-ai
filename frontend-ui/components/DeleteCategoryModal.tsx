'use client';

import { useState, useEffect } from 'react';
import { AccountRecord, Category } from '../lib/types';
import {
  deleteCategory,
  fetchAccounts,
  fetchAccountRecords,
  patchAccountRecord,
} from '../lib/api';
import { transitionAndDeleteCategory } from '../lib/accountCategories';

interface DeleteCategoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  category: Category | null;
  allCategories: Category[];
  associatedRecords: AccountRecord[];
  onDeleted: (deletedCategoryId: string, transitionToCategoryId: string | null) => void;
}

function formatCurrency(val: number): string {
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(abs);
  return val < 0 ? `-${formatted}` : formatted;
}

export function DeleteCategoryModal({
  isOpen,
  onClose,
  token,
  category,
  allCategories,
  associatedRecords,
  onDeleted,
}: DeleteCategoryModalProps) {
  const [transitionTargetId, setTransitionTargetId] = useState<string>('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen) {
      setTransitionTargetId('');
      setError('');
    }
  }, [isOpen, category?.id]);

  if (!isOpen || !category) return null;

  const defaultOptions = allCategories.filter((c) => c.is_default && c.id !== category.id);
  const customOptions = allCategories.filter((c) => !c.is_default && c.id !== category.id);

  const collectRecordsUsingCategory = async (): Promise<AccountRecord[]> => {
    const byId = new Map<string, AccountRecord>();
    for (const rec of associatedRecords) {
      byId.set(rec.id, rec);
    }
    const accounts = await fetchAccounts(token).catch(() => []);
    const recordAccountTypes = new Set(['checking', 'savings', 'credit card', 'credit_card']);
    const otherAccounts = accounts.filter(
      (a) =>
        recordAccountTypes.has((a.type || '').toLowerCase()) &&
        associatedRecords.every((r) => r.account_id !== a.id)
    );
    for (const acc of otherAccounts) {
      const recs = await fetchAccountRecords(token, acc.id).catch(() => []);
      for (const r of recs) {
        if (r.category_id === category.id) {
          byId.set(r.id, r);
        }
      }
    }
    return Array.from(byId.values());
  };

  const handleConfirm = async () => {
    setSubmitting(true);
    setError('');
    try {
      const target = transitionTargetId || null;
      const recordsToMove = target ? await collectRecordsUsingCategory() : [];
      await transitionAndDeleteCategory(
        recordsToMove,
        category.id,
        target,
        async (accId, recId, nextCatId) => {
          await patchAccountRecord(token, accId, recId, { category_id: nextCatId });
        },
        async (catId) => {
          await deleteCategory(token, catId);
        }
      );
      onDeleted(category.id, target);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete category');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-backdrop">
      <div className="modal-content delete-category-modal">
        <div className="modal-header">
          <div>
            <h2 className="modal-title">Delete Category &ldquo;{category.name}&rdquo;</h2>
            <p className="modal-subtitle">
              {associatedRecords.length > 0
                ? 'Confirm category removal and record transition'
                : 'Confirm category removal'}
            </p>
          </div>
          <button type="button" className="btn-close" onClick={onClose} disabled={submitting} aria-label="Close">
            &times;
          </button>
        </div>

        <div className="modal-body-stack">
          <div className="notice-banner">
            <strong>Records will be preserved:</strong> Deleting this category will <strong>NOT</strong> delete any existing transaction records associated with it.
          </div>

          {associatedRecords.length > 0 && (
            <div className="field-group">
              <label htmlFor="transition-category-select">Transition associated records to</label>
              <select
                id="transition-category-select"
                value={transitionTargetId}
                onChange={(e) => setTransitionTargetId(e.target.value)}
                disabled={submitting}
              >
                <option value="">Uncategorized (Unassigned)</option>
                {defaultOptions.length > 0 && (
                  <optgroup label="Default Categories">
                    {defaultOptions.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </optgroup>
                )}
                {customOptions.length > 0 && (
                  <optgroup label="Custom Categories">
                    {customOptions.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </optgroup>
                )}
              </select>
            </div>
          )}

          <div>
            <h4 className="associated-records-heading">
              Associated Records in This Account ({associatedRecords.length})
            </h4>
            {associatedRecords.length === 0 ? (
              <p className="text-muted text-sm">No records in this account currently use this category.</p>
            ) : (
              <div className="associated-records-list">
                <table className="records-table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Recipient</th>
                      <th>Description</th>
                      <th className="text-right">Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {associatedRecords.map((rec) => (
                      <tr key={rec.id}>
                        <td>{rec.date_sent}</td>
                        <td className="font-semibold">{rec.entity}</td>
                        <td className="text-muted">{rec.description || '—'}</td>
                        <td className="text-right font-semibold">{formatCurrency(rec.value)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {error && <div className="alert-error">{error}</div>}
        </div>

        <div className="modal-footer">
          <button type="button" className="btn-secondary" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button type="button" className="btn-danger" onClick={handleConfirm} disabled={submitting}>
            {submitting ? 'Deleting...' : 'Delete Category'}
          </button>
        </div>
      </div>
    </div>
  );
}
