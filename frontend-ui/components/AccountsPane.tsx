'use client';

import { useState, useEffect, useRef } from 'react';
import { Account } from '../lib/types';

interface AccountsPaneProps {
  accounts: Account[];
  selectedId: string | null;
  onSelectAccount: (account: Account) => void;
  onNewAccountClick: () => void;
  showBills?: boolean;
  billsSelected?: boolean;
  billsCount?: number;
  onSelectBills?: () => void;
  onAddBillsClick?: () => void;
}

export function AccountsPane({
  accounts,
  selectedId,
  onSelectAccount,
  onNewAccountClick,
  showBills = false,
  billsSelected = false,
  billsCount = 0,
  onSelectBills,
  onAddBillsClick,
}: AccountsPaneProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [addMenuOpen, setAddMenuOpen] = useState(false);
  const expandedMenuRef = useRef<HTMLDivElement>(null);
  const railMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (showBills) {
      setAddMenuOpen(false);
    }
  }, [showBills]);

  useEffect(() => {
    if (!addMenuOpen) return;
    const handlePointerDown = (e: MouseEvent) => {
      const target = e.target as Node;
      const inExpanded = expandedMenuRef.current?.contains(target);
      const inRail = railMenuRef.current?.contains(target);
      if (!inExpanded && !inRail) {
        setAddMenuOpen(false);
      }
    };
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setAddMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [addMenuOpen]);

  const handlePlusClick = () => {
    if (!showBills && onAddBillsClick) {
      setAddMenuOpen((prev) => !prev);
      return;
    }
    setAddMenuOpen(false);
    onNewAccountClick();
  };

  const handleChooseAccount = () => {
    setAddMenuOpen(false);
    onNewAccountClick();
  };

  const handleChooseBills = () => {
    setAddMenuOpen(false);
    onAddBillsClick?.();
  };

  const plusTitle = !showBills && onAddBillsClick ? 'Add Account or Bills' : 'Create New Account';

  const renderAddPopup = (rail = false) =>
    addMenuOpen && !showBills && onAddBillsClick ? (
      <div
        className={`add-choice-popup ${rail ? 'add-choice-popup-rail' : ''}`}
        role="menu"
        aria-label="Add item"
      >
        <button
          type="button"
          className="add-choice-item"
          role="menuitem"
          onClick={handleChooseAccount}
        >
          Account
        </button>
        <button
          type="button"
          className="add-choice-item"
          role="menuitem"
          onClick={handleChooseBills}
        >
          Bills
        </button>
      </div>
    ) : null;

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
            <div className="plus-menu-anchor" ref={expandedMenuRef}>
              <button
                type="button"
                className="btn-plus"
                onClick={handlePlusClick}
                title={plusTitle}
                aria-label={plusTitle}
                aria-haspopup={!showBills && onAddBillsClick ? 'menu' : undefined}
                aria-expanded={!showBills && onAddBillsClick ? addMenuOpen : undefined}
              >
                +
              </button>
              {renderAddPopup(false)}
            </div>
          )}
        </div>
      </div>

      {collapsed ? (
        <div className="accounts-collapsed-rail">
          <div className="plus-menu-anchor" ref={railMenuRef}>
            <button
              type="button"
              className="btn-plus"
              onClick={handlePlusClick}
              title={plusTitle}
              aria-label={plusTitle}
              aria-haspopup={!showBills && onAddBillsClick ? 'menu' : undefined}
              aria-expanded={!showBills && onAddBillsClick ? addMenuOpen : undefined}
            >
              +
            </button>
            {renderAddPopup(true)}
          </div>
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
          {accounts.length === 0 && !showBills ? (
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
          {showBills && onSelectBills && (
            <button
              type="button"
              className={`account-card account-card-bills ${billsSelected ? 'selected' : ''}`}
              onClick={onSelectBills}
            >
              <div className="account-card-header">
                <span className="account-card-name">Bills</span>
                <span className="badge-account-type badge-bills">
                  {billsCount} {billsCount === 1 ? 'bill' : 'bills'}
                </span>
              </div>
              <p className="account-card-desc">Recurring bills &amp; reminders</p>
            </button>
          )}
        </div>
      )}
    </aside>
  );
}
