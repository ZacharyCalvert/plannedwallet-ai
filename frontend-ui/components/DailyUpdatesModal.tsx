'use client';

import { useEffect, useState } from 'react';
import { DailyUpdateItem } from '../lib/dailyUpdates';
import { submitDailyUpdateItem } from '../lib/dailyUpdatesApi';
import { isoToDisplayDate } from '../lib/dateInput';

interface DailyUpdatesModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  todayIso: string;
  items: DailyUpdateItem[];
  onSaved: (updatedIds: string[]) => Promise<void> | void;
}

function itemKey(item: DailyUpdateItem): string {
  return `${item.kind}:${item.id}`;
}

function formatReferenceValue(val: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: val > 0 && val < 1 ? 6 : 2,
  }).format(val);
}

function parseAmountInput(raw: string): number | null {
  const clean = raw.replace(/[$,\s]/g, '').trim();
  if (!clean) return null;
  const num = Number(clean);
  if (!Number.isFinite(num)) return NaN;
  return num;
}

export function DailyUpdatesModal({
  isOpen,
  onClose,
  token,
  todayIso,
  items,
  onSaved,
}: DailyUpdatesModalProps) {
  const [values, setValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen) return;
    setError('');
    setValues((prev) => {
      const next: Record<string, string> = {};
      for (const item of items) {
        const key = itemKey(item);
        next[key] = prev[key] ?? '';
      }
      return next;
    });
  }, [isOpen, items]);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, saving, onClose]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const toSubmit: Array<{ item: DailyUpdateItem; amount: number }> = [];
    for (const item of items) {
      const raw = values[itemKey(item)] ?? '';
      const parsed = parseAmountInput(raw);
      if (parsed === null) continue;
      if (Number.isNaN(parsed)) {
        setError(`Enter a valid number for "${item.name}".`);
        return;
      }
      if (item.kind === 'asset' && parsed < 0) {
        setError(`Asset unit value for "${item.name}" cannot be negative.`);
        return;
      }
      toSubmit.push({ item, amount: parsed });
    }

    if (toSubmit.length === 0) {
      setError('Enter at least one value to save.');
      return;
    }

    setSaving(true);
    const succeededKeys: string[] = [];
    const failedMessages: string[] = [];

    for (const entry of toSubmit) {
      try {
        await submitDailyUpdateItem(token, entry.item, entry.amount, todayIso);
        succeededKeys.push(itemKey(entry.item));
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Failed to save update';
        failedMessages.push(`${entry.item.name}: ${msg}`);
      }
    }

    if (succeededKeys.length > 0) {
      setValues((prev) => {
        const next = { ...prev };
        for (const k of succeededKeys) {
          delete next[k];
        }
        return next;
      });
      await onSaved(succeededKeys);
    }

    setSaving(false);
    if (failedMessages.length > 0) {
      setError(failedMessages.join(' | '));
      return;
    }

    if (succeededKeys.length === items.length) {
      onClose();
    }
  };

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="daily-updates-modal-title"
    >
      <div className="modal-content daily-updates-modal">
        <div className="modal-header">
          <div>
            <h3 id="daily-updates-modal-title">Today&apos;s Numbers</h3>
            <p className="text-muted font-sm">
              Update values missing a record for {isoToDisplayDate(todayIso)}.
            </p>
          </div>
          <button
            type="button"
            className="btn-close"
            onClick={onClose}
            disabled={saving}
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body daily-updates-body">
            {items.length === 0 ? (
              <p className="text-muted">All snapshot accounts and asset values are up to date for today.</p>
            ) : (
              <div className="daily-updates-list">
                {items.map((item) => {
                  const key = itemKey(item);
                  const inputId = `daily-update-${item.kind}-${item.id}`;
                  return (
                    <div key={key} className="daily-update-row">
                      <div className="daily-update-meta">
                        <div className="daily-update-title-line">
                          <span
                            className={`badge-account-type ${
                              item.kind === 'snapshot' ? 'badge-snapshot' : 'badge-asset'
                            }`}
                          >
                            {item.kind === 'snapshot' ? 'snapshot' : 'asset'}
                          </span>
                          <label htmlFor={inputId} className="daily-update-name">
                            {item.name}
                          </label>
                        </div>
                        <span className="daily-update-subtext">
                          {item.lastValue !== null && item.lastDate
                            ? `Last: ${formatReferenceValue(item.lastValue)} on ${isoToDisplayDate(item.lastDate)}`
                            : item.subtitle}
                        </span>
                      </div>
                      <div className="daily-update-input-wrap">
                        <span className="daily-update-currency" aria-hidden="true">
                          $
                        </span>
                        <input
                          id={inputId}
                          type="text"
                          inputMode="decimal"
                          className="daily-update-input"
                          placeholder="0.00"
                          value={values[key] ?? ''}
                          disabled={saving}
                          onChange={(e) =>
                            setValues((prev) => ({
                              ...prev,
                              [key]: e.target.value,
                            }))
                          }
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {error && <p className="text-danger font-sm mt-3">{error}</p>}
          </div>

          <div className="modal-actions mt-4">
            <button
              type="button"
              className="btn-secondary"
              onClick={onClose}
              disabled={saving}
            >
              Cancel
            </button>
            {items.length > 0 && (
              <button type="submit" className="btn-primary" disabled={saving}>
                {saving ? 'Saving...' : 'Save Updates'}
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}
