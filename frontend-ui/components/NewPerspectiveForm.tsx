'use client';

import { useState } from 'react';
import { createPerspective } from '../lib/api';
import { Account, Perspective } from '../lib/types';

interface NewPerspectiveFormProps {
  token: string;
  allAccounts: Account[];
  onSuccess: (newPerspective: Perspective) => void;
  onCancel: () => void;
}

export function NewPerspectiveForm({
  token,
  allAccounts,
  onSuccess,
  onCancel,
}: NewPerspectiveFormProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedAccountIds, setSelectedAccountIds] = useState<string[]>([]);
  const [showTrend, setShowTrend] = useState(true);
  const [calcDailySpend, setCalcDailySpend] = useState(false);
  const [calcReturnYoY, setCalcReturnYoY] = useState(false);
  const [trackRuined, setTrackRuined] = useState(false);
  const [customSpend, setCustomSpend] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const toggleAccountSelection = (accId: string) => {
    setSelectedAccountIds((prev) =>
      prev.includes(accId) ? prev.filter((id) => id !== accId) : [...prev, accId]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Perspective name is required');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const parsedCustomSpend =
        customSpend.trim() !== '' ? parseFloat(customSpend) : null;

      const created = await createPerspective(token, {
        name: name.trim(),
        description: description.trim(),
        account_ids: selectedAccountIds,
        settings: {
          show_financial_trend: showTrend,
          calculate_average_daily_spend: calcDailySpend,
          calculate_average_return_yoy: calcReturnYoY,
          track_ruined_date: trackRuined,
          custom_daily_spend: parsedCustomSpend,
        },
      });
      onSuccess(created);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create perspective');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card new-account-card new-perspective-card">
      <div className="new-account-header">
        <h3>Create Financial Perspective</h3>
        <p className="section-help-text">
          A viewing window combining multiple accounts into unified financial trends and analytical forecasting.
        </p>
      </div>

      {error && <div className="alert-error mb-3">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="perspective-name">Perspective Name</label>
          <input
            id="perspective-name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Combined Retirement & Wealth, Total Net Worth"
          />
        </div>

        <div className="form-group">
          <label htmlFor="perspective-desc">Description (optional)</label>
          <textarea
            id="perspective-desc"
            rows={2}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="e.g. Aggregated viewing window across retirement and everyday checking"
          />
        </div>

        {/* Account Selection */}
        <div className="form-group mt-3">
          <label>Link Accounts (select to aggregate)</label>
          {allAccounts.length === 0 ? (
            <p className="text-muted text-sm">No existing accounts available to link.</p>
          ) : (
            <div className="account-selection-list">
              {allAccounts.map((acc) => {
                const isSelected = selectedAccountIds.includes(acc.id);
                const typeSlug = (acc.type || '').toLowerCase().replace(/\s+/g, '-');
                return (
                  <label
                    key={acc.id}
                    className={`account-checkbox-item ${isSelected ? 'selected' : ''}`}
                  >
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={() => toggleAccountSelection(acc.id)}
                    />
                    <div className="account-checkbox-info">
                      <span className="acc-item-name">{acc.name}</span>
                      <span className={`badge-account-type badge-${typeSlug}`}>{acc.type}</span>
                    </div>
                  </label>
                );
              })}
            </div>
          )}
        </div>

        {/* Perspective Configuration Toggles */}
        <div className="form-group mt-3">
          <label>Perspective Analytics & Preferences</label>
          <div className="checkbox-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={showTrend}
                onChange={(e) => setShowTrend(e.target.checked)}
              />
              <span>Show financial trend (graph) when opening perspective</span>
            </label>

            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={calcDailySpend}
                onChange={(e) => setCalcDailySpend(e.target.checked)}
              />
              <span>Calculate average daily spend</span>
            </label>

            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={calcReturnYoY}
                onChange={(e) => setCalcReturnYoY(e.target.checked)}
              />
              <span>Calculate average return year over year</span>
            </label>

            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={trackRuined}
                onChange={(e) => setTrackRuined(e.target.checked)}
              />
              <span>Loss of income trajectory (graph spend to $0)</span>
            </label>
          </div>
        </div>

        <hr className="form-divider" />

        <div className="form-group">
          <label htmlFor="new-custom-spend">Custom Daily Spend Override (optional, $/day)</label>
          <input
            id="new-custom-spend"
            type="number"
            step="0.01"
            min="0"
            value={customSpend}
            onChange={(e) => setCustomSpend(e.target.value)}
            placeholder="e.g. 150.00"
          />
          <span className="form-hint">
            Simulate a custom daily spending budget (leave empty to compute automatically from account records).
          </span>
        </div>

        <div className="form-actions mt-4">
          <button type="button" className="btn-secondary" onClick={onCancel}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? 'Creating...' : 'Create Perspective'}
          </button>
        </div>
      </form>
    </div>
  );
}
