'use client';

import { useState } from 'react';
import { Account } from '../lib/types';

interface AccountsPaneProps {
  accounts: Account[];
  selectedId: string | null;
  onSelectAccount: (account: Account) => void;
  onNewAccountClick: () => void;
}

export function AccountsPane({
  accounts,
  selectedId,
  onSelectAccount,
  onNewAccountClick,
}: AccountsPaneProps) {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside className={`accounts-pane ${collapsed ? 'accounts-pane-collapsed' : ''}`}>
      <div className="accounts-pane-header">
        {!collapsed && <h2 className="pane-title">Accounts</h2>}
        <div className="accounts-pane-actions">
          <button
            type="button"
            className="btn-pane-toggle"
            onClick={() => setCollapsed((prev) => !prev)}
            title={collapsed ? 'Expand accounts list' : 'Collapse accounts list'}
            aria-label={collapsed ? 'Expand accounts list' : 'Collapse accounts list'}
            aria-expanded={!collapsed}
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              {collapsed ? (
                <polyline points="13 17 18 12 13 7" />
              ) : (
                <polyline points="11 17 6 12 11 7" />
              )}
              {collapsed ? (
                <polyline points="6 17 11 12 6 7" />
              ) : (
                <polyline points="18 17 13 12 18 7" />
              )}
            </svg>
          </button>
          {!collapsed && (
            <button
              type="button"
              className="btn-plus"
              onClick={onNewAccountClick}
              title="Create New Account"
              aria-label="Create New Account"
            >
              +
            </button>
          )}
        </div>
      </div>

      {collapsed ? (
        <div className="accounts-collapsed-rail">
          <button
            type="button"
            className="btn-plus"
            onClick={onNewAccountClick}
            title="Create New Account"
            aria-label="Create New Account"
          >
            +
          </button>
          <button
            type="button"
            className="accounts-collapsed-expand-label"
            onClick={() => setCollapsed(false)}
            title="Show accounts list"
          >
            Accounts ({accounts.length})
          </button>
        </div>
      ) : (
        <div className="accounts-list">
          {accounts.length === 0 ? (
            <p className="accounts-empty">No accounts yet. Click + to add one.</p>
          ) : (
            accounts.map((acc) => {
              const typeSlug = (acc.type || '').toLowerCase().replace(/\s+/g, '-');
              return (
                <button
                  key={acc.id}
                  type="button"
                  className={`account-card account-card-${typeSlug} ${selectedId === acc.id ? 'selected' : ''}`}
                  onClick={() => onSelectAccount(acc)}
                >
                  <div className="account-card-header">
                    <span className="account-card-name">{acc.name}</span>
                    <span className={`badge-account-type badge-${typeSlug}`}>{acc.type}</span>
                  </div>
                  <p className="account-card-desc">{acc.description}</p>
                </button>
              );
            })
          )}
        </div>
      )}
    </aside>
  );
}
