'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { AccountRecord, Category } from '../lib/types';
import { patchAccountRecord, updateAccountRecord } from '../lib/api';
import { parseNumericSearchQuery, searchAccountRecords } from '../lib/recordSearch';
import { getAmountColorClass } from '../lib/bills';

interface RecordSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  accountId: string;
  accountType: string;
  records: AccountRecord[];
  categories: Category[];
  onRecordUpdated: (updated: AccountRecord) => void;
}

interface InlineEditDraft {
  value: string;
  dateSent: string;
  dateCleared: string;
  entity: string;
  categoryId: string;
  description: string;
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
  return isNegative ? `-${formatted}` : formatted;
}

export function RecordSearchModal({
  isOpen,
  onClose,
  token,
  accountId,
  accountType,
  records,
  categories,
  onRecordUpdated,
}: RecordSearchModalProps) {
  const [query, setQuery] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<InlineEditDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    setEditingId(null);
    setDraft(null);
    setError('');
    const id = setTimeout(() => inputRef.current?.focus(), 0);
    return () => clearTimeout(id);
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        if (editingId) {
          setEditingId(null);
          setDraft(null);
          setError('');
        } else {
          onClose();
        }
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isOpen, editingId, onClose]);

  const numericTarget = useMemo(() => parseNumericSearchQuery(query), [query]);
  const categoryNameById = useMemo(
    () => new Map(categories.map((cat) => [cat.id, cat.name])),
    [categories]
  );

  const matchedRecords = useMemo(
    () => searchAccountRecords(records, query, categories),
    [records, query, categories]
  );

  if (!isOpen) return null;

  const isCreditCard = accountType.toLowerCase().includes('credit');

  const startInlineEdit = (rec: AccountRecord) => {
    setError('');
    setEditingId(rec.id);
    setDraft({
      value: rec.value.toFixed(2),
      dateSent: rec.date_sent,
      dateCleared: rec.date_cleared || '',
      entity: rec.entity,
      categoryId: rec.category_id || '',
      description: rec.description || '',
    });
  };

  const cancelInlineEdit = () => {
    setEditingId(null);
    setDraft(null);
    setError('');
  };

  const handleSaveInlineEdit = async (rec: AccountRecord) => {
    if (!draft) return;
    const isReconciled = Boolean(rec.reconcile_report_id);
    setError('');

    let updated: AccountRecord;
    setSaving(true);
    try {
      if (isReconciled) {
        updated = await patchAccountRecord(token, accountId, rec.id, {
          category_id: draft.categoryId || null,
          description: draft.description.trim(),
        });
      } else {
        const parsedVal = parseFloat(draft.value.replace(/[$,\s]/g, ''));
        if (!Number.isFinite(parsedVal)) {
          setError('Please enter a valid numeric amount.');
          setSaving(false);
          return;
        }
        if (!draft.entity.trim()) {
          setError('Recipient cannot be empty.');
          setSaving(false);
          return;
        }
        if (!/^\d{4}-\d{2}-\d{2}$/.test(draft.dateSent.trim())) {
          setError('Date Sent must be formatted as YYYY-MM-DD.');
          setSaving(false);
          return;
        }
        if (draft.dateCleared.trim() && !/^\d{4}-\d{2}-\d{2}$/.test(draft.dateCleared.trim())) {
          setError('Date Cleared must be formatted as YYYY-MM-DD or left blank.');
          setSaving(false);
          return;
        }
        updated = await updateAccountRecord(token, accountId, rec.id, {
          value: parsedVal,
          date_sent: draft.dateSent.trim(),
          date_cleared: draft.dateCleared.trim() ? draft.dateCleared.trim() : null,
          entity: draft.entity.trim(),
          category_id: draft.categoryId || null,
          description: draft.description.trim(),
        });
      }
      onRecordUpdated(updated);
      setEditingId(null);
      setDraft(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to update record');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-content record-search-modal"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label="Search and edit account records"
      >
        <div className="modal-header">
          <div>
            <h2 className="modal-title">Search Records</h2>
            <p className="modal-subtitle">
              Enter a number for exact absolute amount (e.g. 5.55 matches -5.55 and 5.55), or text to filter recipients, descriptions, categories, and dates.
            </p>
          </div>
          <button type="button" className="btn-close" onClick={onClose} aria-label="Close search">
            &times;
          </button>
        </div>

        <div className="record-search-bar">
          <input
            ref={inputRef}
            type="text"
            className="record-search-input"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search amount (e.g. 5.55) or text (e.g. QT)..."
            aria-label="Search records"
          />
          <span className="record-search-mode-badge">
            {!query.trim()
              ? `All (${matchedRecords.length})`
              : numericTarget !== null
                ? `Amount |${Math.abs(numericTarget).toFixed(2)}| (${matchedRecords.length})`
                : `Text "${query.trim()}" (${matchedRecords.length})`}
          </span>
        </div>

        {error && <div className="form-error mt-2">{error}</div>}

        <div className="record-search-results">
          {matchedRecords.length === 0 ? (
            <div className="empty-state">No records match &ldquo;{query}&rdquo;.</div>
          ) : (
            <table className="records-table">
              <thead>
                <tr>
                  <th>Sent</th>
                  <th>Cleared</th>
                  <th>Recipient</th>
                  <th>Category</th>
                  <th>Description</th>
                  <th className="text-right">Amount</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {matchedRecords.map((rec) => {
                  const isEditing = editingId === rec.id && draft !== null;
                  const isReconciled = Boolean(rec.reconcile_report_id);
                  return (
                    <tr key={rec.id} className={isEditing ? 'selected-row' : ''}>
                      <td>
                        {isEditing && !isReconciled ? (
                          <input
                            type="text"
                            className="record-search-inline-input"
                            value={draft.dateSent}
                            onChange={(e) => setDraft({ ...draft, dateSent: e.target.value })}
                            aria-label="Edit date sent"
                          />
                        ) : (
                          rec.date_sent
                        )}
                      </td>
                      <td>
                        {isEditing && !isReconciled ? (
                          <input
                            type="text"
                            className="record-search-inline-input"
                            value={draft.dateCleared}
                            placeholder="Pending"
                            onChange={(e) => setDraft({ ...draft, dateCleared: e.target.value })}
                            aria-label="Edit date cleared"
                          />
                        ) : (
                          rec.date_cleared || <span className="text-muted">Pending</span>
                        )}
                      </td>
                      <td>
                        {isEditing && !isReconciled ? (
                          <input
                            type="text"
                            className="record-search-inline-input"
                            value={draft.entity}
                            onChange={(e) => setDraft({ ...draft, entity: e.target.value })}
                            aria-label="Edit recipient"
                          />
                        ) : (
                          <>
                            {rec.entity}
                            {isReconciled && (
                              <span className="reconciled-pill ml-2" title="Reconciled record">
                                Reconciled
                              </span>
                            )}
                          </>
                        )}
                      </td>
                      <td>
                        {isEditing ? (
                          <select
                            className="record-search-inline-input"
                            value={draft.categoryId}
                            onChange={(e) => setDraft({ ...draft, categoryId: e.target.value })}
                            aria-label="Edit category"
                          >
                            <option value="">Uncategorized</option>
                            {categories.map((cat) => (
                              <option key={cat.id} value={cat.id}>
                                {cat.name}
                              </option>
                            ))}
                          </select>
                        ) : rec.category_id ? (
                          categoryNameById.get(rec.category_id) || 'Uncategorized'
                        ) : (
                          <span className="text-muted">Uncategorized</span>
                        )}
                      </td>
                      <td>
                        {isEditing ? (
                          <input
                            type="text"
                            className="record-search-inline-input"
                            value={draft.description}
                            onChange={(e) => setDraft({ ...draft, description: e.target.value })}
                            aria-label="Edit description"
                          />
                        ) : (
                          rec.description || <span className="text-muted">—</span>
                        )}
                      </td>
                      <td className={`text-right ${getAmountColorClass(String(rec.value))}`}>
                        {isEditing && !isReconciled ? (
                          <input
                            type="text"
                            className="record-search-inline-input text-right"
                            value={draft.value}
                            onChange={(e) => setDraft({ ...draft, value: e.target.value })}
                            aria-label="Edit amount"
                          />
                        ) : (
                          formatCurrency(rec.value)
                        )}
                      </td>
                      <td>
                        {isEditing ? (
                          <div className="record-search-actions">
                            <button
                              type="button"
                              className="btn-primary btn-sm"
                              onClick={() => handleSaveInlineEdit(rec)}
                              disabled={saving}
                            >
                              {saving ? 'Saving...' : 'Save'}
                            </button>
                            <button
                              type="button"
                              className="btn-secondary btn-sm"
                              onClick={cancelInlineEdit}
                              disabled={saving}
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            type="button"
                            className="btn-secondary btn-sm"
                            onClick={() => startInlineEdit(rec)}
                          >
                            Edit
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
