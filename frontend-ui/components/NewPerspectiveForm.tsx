'use client';

import { useState } from 'react';
import { Account, Perspective, RecurringTransaction } from '../lib/types';
import { createPerspective } from '../lib/api';

interface NewPerspectiveFormProps {
  token: string;
  accounts: Account[];
  recurringTransactions: RecurringTransaction[];
  onSuccess: (created: Perspective) => void;
  onCancel: () => void;
}

export function NewPerspectiveForm({
  token,
  accounts,
  recurringTransactions,
  onSuccess,
  onCancel,
}: NewPerspectiveFormProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedAccounts, setSelectedAccounts] = useState<Record<string, boolean>>({});
  const [selectedRecurring, setSelectedRecurring] = useState<Record<string, boolean>>({});
  const [showCategories, setShowCategories] = useState(false);
  const [trackRuinedDate, setTrackRuinedDate] = useState(false);
  const [dailySpendStr, setDailySpendStr] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const toggleAccount = (id: string) => {
    setSelectedAccounts((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const toggleRecurring = (id: string) => {
    setSelectedRecurring((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError('Perspective name is required.');
      return;
    }

    let customDailySpend: number | null = null;
    if (trackRuinedDate && dailySpendStr.trim()) {
      const parsed = parseFloat(dailySpendStr.replace(/[$,\s]/g, ''));
      if (!Number.isFinite(parsed) || parsed < 0) {
        setError('Daily spend must be a valid non-negative number.');
        return;
      }
      customDailySpend = parsed;
    }

    const accountIds = Object.keys(selectedAccounts).filter((id) => selectedAccounts[id]);
    const recurringIds = Object.keys(selectedRecurring).filter((id) => selectedRecurring[id]);

    setSubmitting(true);
    try {
      const created = await createPerspective(token, {
        name: trimmedName,
        description: description.trim(),
        account_ids: accountIds,
        recurring_transaction_ids: recurringIds,
        settings: {
          show_financial_trend: true,
          show_categories: showCategories,
          track_ruined_date: trackRuinedDate,
          custom_daily_spend: customDailySpend,
        },
      });
      onSuccess(created);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create perspective.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="card form-card">
      <div className="form-card-header">
        <h2>New Perspective</h2>
        <p className="text-muted">
          Aggregate multiple accounts, bills, and income into a unified viewing window.
        </p>
      </div>

      {error && <div className="alert-error">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="perspective-name">Perspective Name</label>
          <input
            id="perspective-name"
            type="text"
            value={name}
            maxLength={100}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Retirement Portfolio, Credit Card Spend"
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="perspective-desc">Description</label>
          <input
            id="perspective-desc"
            type="text"
            value={description}
            maxLength={255}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional description of what this perspective tracks"
          />
        </div>

        <div className="form-group">
          <label>Linked Accounts</label>
          {accounts.length === 0 ? (
            <p className="text-muted text-sm">No accounts created yet.</p>
          ) : (
            <div className="perspective-checkbox-grid">
              {accounts.map((acc) => (
                <label key={acc.id} className="perspective-checkbox-item">
                  <input
                    type="checkbox"
                    checked={Boolean(selectedAccounts[acc.id])}
                    onChange={() => toggleAccount(acc.id)}
                  />
                  <span>{acc.name}</span>
                  <span className="text-muted text-sm">({acc.type})</span>
                </label>
              ))}
            </div>
          )}
        </div>

        <div className="form-group">
          <label>Linked Bills &amp; Income</label>
          {recurringTransactions.length === 0 ? (
            <p className="text-muted text-sm">No recurring bills or income configured yet.</p>
          ) : (
            <div className="perspective-checkbox-grid">
              {recurringTransactions.map((tx) => (
                <label key={tx.id} className="perspective-checkbox-item">
                  <input
                    type="checkbox"
                    checked={Boolean(selectedRecurring[tx.id])}
                    onChange={() => toggleRecurring(tx.id)}
                  />
                  <span>{tx.entity}</span>
                  <span className="text-muted text-sm">
                    ({tx.amount < 0 ? 'Bill' : 'Income'} ${Math.abs(tx.amount).toFixed(2)} / {tx.frequency})
                  </span>
                </label>
              ))}
            </div>
          )}
        </div>

        <div className="form-group">
          <label>Viewing Window Settings</label>
          <div className="perspective-settings-checks">
            <label className="perspective-checkbox-item">
              <input
                type="checkbox"
                checked={showCategories}
                onChange={(e) => setShowCategories(e.target.checked)}
              />
              <span>Show Categories Pie Chart (accounts, bills, and income)</span>
            </label>
            <label className="perspective-checkbox-item">
              <input
                type="checkbox"
                checked={trackRuinedDate}
                onChange={(e) => setTrackRuinedDate(e.target.checked)}
              />
              <span>Track Projected $0 Runway Date</span>
            </label>
          </div>
        </div>

        {trackRuinedDate && (
          <div className="form-group">
            <label htmlFor="perspective-daily-spend">Custom Daily Spend ($/day)</label>
            <input
              id="perspective-daily-spend"
              type="text"
              value={dailySpendStr}
              onChange={(e) => setDailySpendStr(e.target.value)}
              placeholder="e.g. 75.00"
            />
          </div>
        )}

        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={onCancel} disabled={submitting}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={submitting}>
            {submitting ? 'Creating...' : 'Create Perspective'}
          </button>
        </div>
      </form>
    </div>
  );
}
