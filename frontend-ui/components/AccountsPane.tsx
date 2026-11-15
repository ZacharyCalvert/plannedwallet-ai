'use client';

import { useState, useRef, useEffect } from 'react';
import { Account, Perspective } from '../lib/types';

interface AccountsPaneProps {
  accounts: Account[];
  selectedAccountId: string | null;
  onSelectAccount: (account: Account) => void;
  perspectives: Perspective[];
  selectedPerspectiveId: string | null;
  onSelectPerspective: (perspective: Perspective) => void;
  onNewAccountClick: () => void;
  onNewPerspectiveClick: () => void;
}

export function AccountsPane({
  accounts,
  selectedAccountId,
  onSelectAccount,
  perspectives,
  selectedPerspectiveId,
  onSelectPerspective,
  onNewAccountClick,
  onNewPerspectiveClick,
}: AccountsPaneProps) {
  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowMenu(false);
      }
    };
    if (showMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showMenu]);

  return (
    <aside className="accounts-pane">
      <div className="accounts-pane-header">
        <h2 className="pane-title">Planned Wallet</h2>
        <div className="plus-menu-container" ref={menuRef}>
          <button
            type="button"
            className="btn-plus"
            onClick={() => setShowMenu((prev) => !prev)}
            title="Add Account or Perspective"
            aria-label="Add New"
          >
            +
          </button>
          {showMenu && (
            <div className="plus-dropdown-menu">
              <button
                type="button"
                className="dropdown-item"
                onClick={() => {
                  setShowMenu(false);
                  onNewAccountClick();
                }}
              >
                <span className="dropdown-icon">🏦</span>
                <span>Add Account</span>
              </button>
              <button
                type="button"
                className="dropdown-item"
                onClick={() => {
                  setShowMenu(false);
                  onNewPerspectiveClick();
                }}
              >
                <span className="dropdown-icon">🔮</span>
                <span>Add Perspective</span>
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="accounts-list">
        {/* Perspectives Section */}
        {perspectives.length > 0 && (
          <div className="pane-section">
            <div className="pane-section-header">
              <span className="pane-section-title">Perspectives</span>
              <span className="pane-section-count">{perspectives.length}</span>
            </div>
            {perspectives.map((p) => {
              const isSelected = selectedPerspectiveId === p.id;
              return (
                <button
                  key={p.id}
                  type="button"
                  className={`account-card account-card-perspective ${isSelected ? 'selected' : ''}`}
                  onClick={() => onSelectPerspective(p)}
                >
                  <div className="account-card-header">
                    <span className="account-card-name">{p.name}</span>
                    <span className="badge-account-type badge-perspective">Perspective</span>
                  </div>
                  <p className="account-card-desc">
                    {p.description || `${p.account_ids?.length || 0} accounts aggregated`}
                  </p>
                </button>
              );
            })}
          </div>
        )}

        {/* Accounts Section */}
        <div className="pane-section">
          {perspectives.length > 0 && (
            <div className="pane-section-header">
              <span className="pane-section-title">Accounts</span>
              <span className="pane-section-count">{accounts.length}</span>
            </div>
          )}
          {accounts.length === 0 && perspectives.length === 0 ? (
            <p className="accounts-empty">No accounts or perspectives yet. Click + to add one.</p>
          ) : (
            accounts.map((acc) => {
              const typeSlug = (acc.type || '').toLowerCase().replace(/\s+/g, '-');
              const isSelected = selectedAccountId === acc.id;
              return (
                <button
                  key={acc.id}
                  type="button"
                  className={`account-card account-card-${typeSlug} ${isSelected ? 'selected' : ''}`}
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
      </div>
    </aside>
  );
}
