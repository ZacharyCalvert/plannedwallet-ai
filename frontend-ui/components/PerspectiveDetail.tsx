'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  Account,
  Perspective,
  PerspectiveSettings,
  PerspectiveViewResponse,
} from '../lib/types';
import {
  fetchPerspectiveView,
  updatePerspectiveSettings,
  linkPerspectiveAccount,
  unlinkPerspectiveAccount,
  deletePerspective,
} from '../lib/api';
import { PerspectiveTrendGraph } from './PerspectiveTrendGraph';
import { RuinedTrajectoryChart } from './RuinedTrajectoryChart';

interface PerspectiveDetailProps {
  perspective: Perspective;
  token: string;
  allAccounts: Account[];
  onPerspectiveUpdated: (updated: Perspective) => void;
  onPerspectiveDeleted: (id: string) => void;
}

type TabKey = 'trend' | 'spend' | 'return' | 'ruined' | 'settings';

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

function formatDateDisplay(isoDate: string): string {
  if (!isoDate) return '';
  const [y, m, d] = isoDate.split('-');
  const dateObj = new Date(parseInt(y, 10), parseInt(m, 10) - 1, parseInt(d, 10));
  return dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function PerspectiveDetail({
  perspective,
  token,
  allAccounts,
  onPerspectiveUpdated,
  onPerspectiveDeleted,
}: PerspectiveDetailProps) {
  const [view, setView] = useState<PerspectiveViewResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  // Helper to resolve the leftmost available tab from perspective settings
  const getLeftmostTab = (s?: PerspectiveSettings): TabKey => {
    if (!s || s.show_financial_trend) return 'trend';
    if (s.calculate_average_daily_spend) return 'spend';
    if (s.calculate_average_return_yoy) return 'return';
    if (s.track_ruined_date) return 'ruined';
    return 'settings';
  };

  const [activeTab, setActiveTab] = useState<TabKey>(() => getLeftmostTab(perspective?.settings));

  // Settings form state
  const [settingsForm, setSettingsForm] = useState({
    show_financial_trend: true,
    calculate_average_daily_spend: false,
    calculate_average_return_yoy: false,
    track_ruined_date: false,
    custom_daily_spend: '',
  });
  const [savingSettings, setSavingSettings] = useState(false);
  const [settingsSuccess, setSettingsSuccess] = useState('');

  // Account linking state
  const [selectedAccountToAdd, setSelectedAccountToAdd] = useState('');
  const [linkingAccount, setLinkingAccount] = useState(false);

  const loadView = async () => {
    try {
      setLoading(true);
      setError('');
      const data = await fetchPerspectiveView(token, perspective.id);
      setView(data);
      if (data.settings) {
        setActiveTab(getLeftmostTab(data.settings));
        setSettingsForm({
          show_financial_trend: data.settings.show_financial_trend,
          calculate_average_daily_spend: data.settings.calculate_average_daily_spend,
          calculate_average_return_yoy: data.settings.calculate_average_return_yoy,
          track_ruined_date: data.settings.track_ruined_date,
          custom_daily_spend:
            data.settings.custom_daily_spend != null
              ? String(data.settings.custom_daily_spend)
              : '',
        });
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load perspective view');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setActiveTab(getLeftmostTab(perspective?.settings));
    loadView();
  }, [perspective.id, token]);

  // Compute tabs dynamically based on perspective settings
  const dynamicTabs = useMemo(() => {
    const s = view?.settings || perspective?.settings;
    const tabs: { key: TabKey; label: string; icon: string }[] = [];
    if (!s || s.show_financial_trend) {
      tabs.push({ key: 'trend', label: 'Financial Trend', icon: '📈' });
    }
    if (s?.calculate_average_daily_spend) {
      tabs.push({ key: 'spend', label: 'Daily Spend', icon: '💳' });
    }
    if (s?.calculate_average_return_yoy) {
      tabs.push({ key: 'return', label: 'YoY Return', icon: '🚀' });
    }
    if (s?.track_ruined_date) {
      tabs.push({ key: 'ruined', label: 'Loss of Income Trajectory', icon: '⏳' });
    }
    tabs.push({ key: 'settings', label: 'Settings', icon: '⚙️' });
    return tabs;
  }, [view?.settings, perspective?.settings]);

  // Ensure active tab is valid
  useEffect(() => {
    if (dynamicTabs.length > 0) {
      const valid = dynamicTabs.some((t) => t.key === activeTab);
      if (!valid) {
        setActiveTab(dynamicTabs[0].key);
      }
    }
  }, [dynamicTabs, activeTab]);

  const unlinkedAccounts = useMemo(() => {
    const linkedIds = new Set(view?.accounts.map((a) => a.id) || []);
    return allAccounts.filter((a) => !linkedIds.has(a.id));
  }, [allAccounts, view?.accounts]);

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingSettings(true);
    setSettingsSuccess('');
    setError('');
    try {
      const customSpend =
        settingsForm.custom_daily_spend.trim() !== ''
          ? parseFloat(settingsForm.custom_daily_spend)
          : null;

      const updatedSettings = await updatePerspectiveSettings(token, perspective.id, {
        show_financial_trend: settingsForm.show_financial_trend,
        calculate_average_daily_spend: settingsForm.calculate_average_daily_spend,
        calculate_average_return_yoy: settingsForm.calculate_average_return_yoy,
        track_ruined_date: settingsForm.track_ruined_date,
        custom_daily_spend: customSpend,
      });

      setSettingsSuccess('Perspective settings updated successfully!');
      onPerspectiveUpdated({
        ...perspective,
        settings: updatedSettings,
      });
      await loadView();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to update settings');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleLinkAccount = async () => {
    if (!selectedAccountToAdd) return;
    setLinkingAccount(true);
    setError('');
    try {
      await linkPerspectiveAccount(token, perspective.id, selectedAccountToAdd);
      setSelectedAccountToAdd('');
      await loadView();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to link account');
    } finally {
      setLinkingAccount(false);
    }
  };

  const handleUnlinkAccount = async (accId: string) => {
    if (!confirm('Are you sure you want to remove this account from the perspective?')) {
      return;
    }
    try {
      await unlinkPerspectiveAccount(token, perspective.id, accId);
      await loadView();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to unlink account');
    }
  };

  const handleDeletePerspective = async () => {
    if (!confirm(`Are you sure you want to delete perspective "${perspective.name}"?`)) return;
    try {
      await deletePerspective(token, perspective.id);
      onPerspectiveDeleted(perspective.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete perspective');
    }
  };

  if (loading && !view) {
    return (
      <div className="card loading-card">
        <p>Loading perspective...</p>
      </div>
    );
  }

  const currentTotal = view?.current_total_value ?? 0;

  return (
    <div className="card perspective-detail-card">
      {/* Header Banner */}
      <div className="account-detail-header perspective-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>{perspective.name}</h2>
            <span className="badge-account-type badge-perspective">Perspective</span>
          </div>
          <p className="account-desc-text">{perspective.description || 'Aggregated Financial Perspective'}</p>
          <p className="account-id-label">Perspective ID: {perspective.id}</p>
        </div>

        <div className="account-balances">
          <div className="balance-box current-balance-box">
            <span className="balance-label">Combined Net Worth</span>
            <span className="balance-value balance-positive">{formatCurrency(currentTotal)}</span>
            <span className="balance-subtext">Across {view?.accounts.length ?? 0} linked accounts</span>
          </div>

          {view?.average_daily_spend != null && (
            <div className="balance-box expected-balance-box">
              <span className="balance-label">Avg Daily Spend</span>
              <span className="balance-value amount-text-negative">
                {formatCurrency(view.average_daily_spend)}
              </span>
              <span className="balance-subtext">per day</span>
            </div>
          )}

          {view?.days_until_ruined != null && (
            <div className="balance-box ruined-balance-box">
              <span className="balance-label">Runway</span>
              <span className="balance-value">{view.days_until_ruined} days</span>
              <span className="balance-subtext">{formatDateDisplay(view.ruined_date || '')}</span>
            </div>
          )}
        </div>
      </div>

      {error && <div className="alert-error mb-4">{error}</div>}
      {settingsSuccess && <div className="alert-success mb-4">{settingsSuccess}</div>}

      {/* Dynamic Tab Bar based on Perspective Settings */}
      <div className="snapshot-tabs-bar perspective-tabs-bar">
        {dynamicTabs.map((t) => (
          <button
            key={t.key}
            type="button"
            className={`snapshot-tab-button ${activeTab === t.key ? 'active' : ''}`}
            onClick={() => setActiveTab(t.key)}
          >
            <span className="tab-icon">{t.icon}</span>
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab 1: Financial Trend */}
      {activeTab === 'trend' && view?.settings.show_financial_trend && (
        <div className="perspective-tab-content">
          <PerspectiveTrendGraph trend={view.trend || []} accounts={view.accounts || []} />
        </div>
      )}

      {/* Tab 2: Daily Spend */}
      {activeTab === 'spend' && view?.settings.calculate_average_daily_spend && (
        <div className="perspective-tab-content">
          <div className="metric-spotlight-card card">
            <h3>Average Daily Spend Analysis</h3>
            <div className="spotlight-metric-value amount-text-negative">
              {formatCurrency(view.average_daily_spend || 0)}
              <span className="spotlight-unit">/ day</span>
            </div>
            <p className="spotlight-description">
              Calculated based on debits and expenses tracked across linked checking, savings, and credit card accounts.
            </p>

            <div className="spend-projections-grid">
              <div className="projection-box">
                <span className="proj-label">Weekly Burn</span>
                <span className="proj-val">{formatCurrency((view.average_daily_spend || 0) * 7)}</span>
              </div>
              <div className="projection-box">
                <span className="proj-label">Monthly Burn (30d)</span>
                <span className="proj-val">{formatCurrency((view.average_daily_spend || 0) * 30)}</span>
              </div>
              <div className="projection-box">
                <span className="proj-label">Annual Burn (365d)</span>
                <span className="proj-val">{formatCurrency((view.average_daily_spend || 0) * 365)}</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tab 3: YoY Return */}
      {activeTab === 'return' && view?.settings.calculate_average_return_yoy && (
        <div className="perspective-tab-content">
          <div className="metric-spotlight-card card">
            <h3>Year-Over-Year Annualized Return</h3>
            <div className="spotlight-metric-value amount-text-positive">
              {view.average_return_yoy != null ? (
                <>
                  {view.average_return_yoy >= 0 ? '+' : ''}
                  {(view.average_return_yoy * 100).toFixed(2)}%
                  <span className="spotlight-unit">YoY</span>
                </>
              ) : (
                <span className="text-muted">Insufficient historical data</span>
              )}
            </div>
            <p className="spotlight-description">
              Compound Annual Growth Rate (CAGR) measuring how your aggregated accounts and portfolios are growing over time.
            </p>
          </div>
        </div>
      )}

      {/* Tab 4: Loss of Income Trajectory (Spend to $0) */}
      {activeTab === 'ruined' && view?.settings.track_ruined_date && (
        <div className="perspective-tab-content">
          <div className="metric-spotlight-card card ruined-spotlight-card">
            <div className="ruined-spotlight-header">
              <div>
                <h3 className="ruined-spotlight-title">Loss of Income Trajectory (Spend to $0)</h3>
                <p className="ruined-spotlight-desc">
                  Simulation modeling how long your accumulated savings will last if all incoming revenue ceases today.
                </p>
              </div>
              <div className="ruined-date-pill">
                Current Savings: <strong>{formatCurrency(currentTotal)}</strong>
              </div>
            </div>

            {/* 30, 90, 365 Days Spending & Runway Comparison */}
            <div className="runway-windows-grid mt-2">
              {/* 30 Day Card */}
              <div className="runway-window-card">
                <span className="window-label">Last 30 Days Pace</span>
                <div className="window-spend-amount">
                  {formatCurrency(view.daily_spend_30d ?? 0)}
                  <span className="window-unit">/ day</span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Runway:</span>
                  <span className="stat-val">
                    {view.days_until_ruined_30d != null ? `${view.days_until_ruined_30d} days` : 'Infinite'}
                  </span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Zero-Balance:</span>
                  <span className="stat-val font-semibold">
                    {view.ruined_date_30d ? formatDateDisplay(view.ruined_date_30d) : 'Never'}
                  </span>
                </div>
              </div>

              {/* 90 Day Card */}
              <div className="runway-window-card">
                <span className="window-label">Last 90 Days Pace</span>
                <div className="window-spend-amount">
                  {formatCurrency(view.daily_spend_90d ?? 0)}
                  <span className="window-unit">/ day</span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Runway:</span>
                  <span className="stat-val">
                    {view.days_until_ruined_90d != null ? `${view.days_until_ruined_90d} days` : 'Infinite'}
                  </span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Zero-Balance:</span>
                  <span className="stat-val font-semibold">
                    {view.ruined_date_90d ? formatDateDisplay(view.ruined_date_90d) : 'Never'}
                  </span>
                </div>
              </div>

              {/* 365 Day Card */}
              <div className="runway-window-card">
                <span className="window-label">Last 365 Days Pace</span>
                <div className="window-spend-amount">
                  {formatCurrency(view.daily_spend_365d ?? 0)}
                  <span className="window-unit">/ day</span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Runway:</span>
                  <span className="stat-val">
                    {view.days_until_ruined_365d != null ? `${view.days_until_ruined_365d} days` : 'Infinite'}
                  </span>
                </div>
                <div className="window-runway-stat">
                  <span className="stat-label">Zero-Balance:</span>
                  <span className="stat-val font-semibold">
                    {view.ruined_date_365d ? formatDateDisplay(view.ruined_date_365d) : 'Never'}
                  </span>
                </div>
              </div>
            </div>

            {/* Custom simulation pace if configured */}
            {view.settings.custom_daily_spend != null && view.settings.custom_daily_spend > 0 && (
              <div className="custom-pace-callout mt-2">
                <span>Custom Daily Spend Override: <strong>{formatCurrency(view.settings.custom_daily_spend)}/day</strong></span>
                <span> → Projected Runway: <strong>{view.days_until_ruined} days</strong> ({formatDateDisplay(view.ruined_date || '')})</span>
              </div>
            )}

            {/* Transfer & CC payment deduction notice */}
            <div className="transfer-filter-notice mt-2">
              <span className="notice-icon">💳</span>
              <span className="notice-text">
                Transfers between checking/savings and credit cards within a 35-day window are automatically recognized and removed from spend calculations to avoid double-counting.
              </span>
            </div>
          </div>

          {/* Visual Trajectory Chart */}
          <RuinedTrajectoryChart
            currentSavings={currentTotal}
            dailySpend30d={view.daily_spend_30d ?? 0}
            daysUntilRuined30d={view.days_until_ruined_30d ?? null}
            ruinedDate30d={view.ruined_date_30d ?? null}
            dailySpend90d={view.daily_spend_90d ?? 0}
            daysUntilRuined90d={view.days_until_ruined_90d ?? null}
            ruinedDate90d={view.ruined_date_90d ?? null}
            dailySpend365d={view.daily_spend_365d ?? 0}
            daysUntilRuined365d={view.days_until_ruined_365d ?? null}
            ruinedDate365d={view.ruined_date_365d ?? null}
          />
        </div>
      )}

      {/* Tab 5: Settings (Final tab) */}
      {activeTab === 'settings' && (
        <div className="perspective-tab-content settings-tab-content">
          {/* Section 1: Perspective Configuration */}
          <div className="settings-section card">
            <h3>Perspective Preferences</h3>
            <p className="section-help-text">
              Configure which analytics and viewing tabs are displayed in this perspective.
            </p>

            <form onSubmit={handleSaveSettings}>
              <div className="checkbox-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={settingsForm.show_financial_trend}
                    onChange={(e) =>
                      setSettingsForm((prev) => ({ ...prev, show_financial_trend: e.target.checked }))
                    }
                  />
                  <span>Show financial trend (graph) when viewing this perspective</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={settingsForm.calculate_average_daily_spend}
                    onChange={(e) =>
                      setSettingsForm((prev) => ({
                        ...prev,
                        calculate_average_daily_spend: e.target.checked,
                      }))
                    }
                  />
                  <span>Calculate average daily spend</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={settingsForm.calculate_average_return_yoy}
                    onChange={(e) =>
                      setSettingsForm((prev) => ({
                        ...prev,
                        calculate_average_return_yoy: e.target.checked,
                      }))
                    }
                  />
                  <span>Calculate average return year over year</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={settingsForm.track_ruined_date}
                    onChange={(e) =>
                      setSettingsForm((prev) => ({ ...prev, track_ruined_date: e.target.checked }))
                    }
                  />
                  <span>Loss of income trajectory (graph spend to $0)</span>
                </label>
              </div>

              <hr className="form-divider" />

              <div className="form-group">
                <label htmlFor="custom-daily-spend">Custom Daily Spend Override (optional, $/day)</label>
                <input
                  id="custom-daily-spend"
                  type="number"
                  step="0.01"
                  min="0"
                  value={settingsForm.custom_daily_spend}
                  onChange={(e) =>
                    setSettingsForm((prev) => ({ ...prev, custom_daily_spend: e.target.value }))
                  }
                  placeholder="e.g. 100.00 (leave blank to calculate from transactions)"
                />
                <span className="field-hint">
                  Allows simulating custom budgets (e.g. what happens if you spend $100/day during job loss).
                </span>
              </div>

              <div className="form-actions mt-4">
                <button type="submit" className="btn-primary" disabled={savingSettings}>
                  {savingSettings ? 'Saving...' : 'Save Settings'}
                </button>
              </div>
            </form>
          </div>

          {/* Section 2: Manage Linked Accounts */}
          <div className="settings-section card mt-4">
            <h3>Linked Accounts</h3>
            <p className="section-help-text">
              Accounts aggregated into this perspective. Transactions and valuations from these accounts form the combined trends.
            </p>

            <div className="link-account-row">
              <select
                value={selectedAccountToAdd}
                onChange={(e) => setSelectedAccountToAdd(e.target.value)}
                disabled={unlinkedAccounts.length === 0}
              >
                <option value="">
                  {unlinkedAccounts.length === 0 ? 'All accessible accounts already linked' : '-- Select account to link --'}
                </option>
                {unlinkedAccounts.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name} ({a.type})
                  </option>
                ))}
              </select>
              <button
                type="button"
                className="btn-primary"
                onClick={handleLinkAccount}
                disabled={!selectedAccountToAdd || linkingAccount}
              >
                {linkingAccount ? 'Linking...' : 'Add Account'}
              </button>
            </div>

            <div className="linked-accounts-table-wrapper mt-3">
              <table className="table-records">
                <thead>
                  <tr>
                    <th>Account Name</th>
                    <th>Type</th>
                    <th>Description</th>
                    <th className="text-right">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {view?.accounts.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="text-muted text-center py-3">
                        No accounts linked yet. Use the dropdown above to add accounts.
                      </td>
                    </tr>
                  ) : (
                    view?.accounts.map((acc) => {
                      const typeSlug = (acc.type || '').toLowerCase().replace(/\s+/g, '-');
                      return (
                        <tr key={acc.id}>
                          <td><strong>{acc.name}</strong></td>
                          <td>
                            <span className={`badge-account-type badge-${typeSlug}`}>{acc.type}</span>
                          </td>
                          <td className="text-muted">{acc.description}</td>
                          <td className="text-right">
                            <button
                              type="button"
                              className="btn-delete"
                              onClick={() => handleUnlinkAccount(acc.id)}
                            >
                              Remove
                            </button>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Section 3: Danger Zone */}
          <div className="settings-section card card-danger mt-4">
            <h3>Delete Perspective</h3>
            <p className="section-help-text">
              Deleting this perspective removes the combined viewing window and settings. The underlying accounts and transactions are preserved.
            </p>
            <button type="button" className="btn-delete-perspective" onClick={handleDeletePerspective}>
              Delete Perspective
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
