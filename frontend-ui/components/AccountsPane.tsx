'use client';

import { useState, useEffect, useRef } from 'react';
import { Account, Perspective } from '../lib/types';

interface AccountsPaneProps {
  accounts: Account[];
  selectedId: string | null;
  onSelectAccount: (account: Account) => void;
  onNewAccountClick: () => void;
  perspectives?: Perspective[];
  selectedPerspectiveId?: string | null;
  onSelectPerspective?: (perspective: Perspective) => void;
  onNewPerspectiveClick?: () => void;
  showBills?: boolean;
  billsSelected?: boolean;
  billsCount?: number;
  onSelectBills?: () => void;
  onAddBillsClick?: () => void;
  showIncomes?: boolean;
  incomesSelected?: boolean;
  incomesCount?: number;
  onSelectIncomes?: () => void;
  onAddIncomesClick?: () => void;
  dailyUpdatesCount?: number;
  onDailyUpdatesClick?: () => void;
}

export function AccountsPane({
  accounts,
  selectedId,
  onSelectAccount,
  onNewAccountClick,
  perspectives = [],
  selectedPerspectiveId = null,
  onSelectPerspective,
  onNewPerspectiveClick,
  showBills = false,
  billsSelected = false,
  billsCount = 0,
  onSelectBills,
  onAddBillsClick,
  showIncomes = false,
  incomesSelected = false,
  incomesCount = 0,
  onSelectIncomes,
  onAddIncomesClick,
  dailyUpdatesCount = 0,
  onDailyUpdatesClick,
}: AccountsPaneProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [accountsCollapsed, setAccountsCollapsed] = useState(false);
  const [recurringCollapsed, setRecurringCollapsed] = useState(false);
  const [perspectivesCollapsed, setPerspectivesCollapsed] = useState(false);
  const [addMenuOpen, setAddMenuOpen] = useState(false);
  const expandedMenuRef = useRef<HTMLDivElement>(null);
  const railMenuRef = useRef<HTMLDivElement>(null);

  const canOfferBills = !showBills && Boolean(onAddBillsClick);
  const canOfferIncomes = !showIncomes && Boolean(onAddIncomesClick);
  const canOfferPerspective = Boolean(onNewPerspectiveClick);
  const hasMenuChoices = canOfferBills || canOfferIncomes || canOfferPerspective;

  useEffect(() => {
    if (!hasMenuChoices) {
      setAddMenuOpen(false);
    }
  }, [hasMenuChoices]);

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
    if (hasMenuChoices) {
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

  const handleChooseIncomes = () => {
    setAddMenuOpen(false);
    onAddIncomesClick?.();
  };

  const handleChoosePerspective = () => {
    setAddMenuOpen(false);
    onNewPerspectiveClick?.();
  };

  const plusTitle = hasMenuChoices ? 'Add Account, Perspective, Bills, or Income' : 'Create New Account';

  const renderAddPopup = (rail = false) =>
    addMenuOpen && hasMenuChoices ? (
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
        {canOfferPerspective && (
          <button
            type="button"
            className="add-choice-item"
            role="menuitem"
            onClick={handleChoosePerspective}
          >
            Perspective
          </button>
        )}
        {canOfferBills && (
          <button
            type="button"
            className="add-choice-item"
            role="menuitem"
            onClick={handleChooseBills}
          >
            Bills
          </button>
        )}
        {canOfferIncomes && (
          <button
            type="button"
            className="add-choice-item"
            role="menuitem"
            onClick={handleChooseIncomes}
          >
            Income
          </button>
        )}
      </div>
    ) : null;

  return (
    <aside className={`accounts-pane ${collapsed ? 'accounts-pane-collapsed' : ''}`}>
      <div className="accounts-pane-header">
        {!collapsed && <h2 className="pane-title">Finances</h2>}
        <div className="accounts-pane-actions">
          <button
            type="button"
            className="btn-pane-toggle"
            onClick={() => setCollapsed((prev) => !prev)}
            title={collapsed ? 'Expand finances list' : 'Collapse finances list'}
            aria-label={collapsed ? 'Expand finances list' : 'Collapse finances list'}
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
                aria-haspopup={hasMenuChoices ? 'menu' : undefined}
                aria-expanded={hasMenuChoices ? addMenuOpen : undefined}
              >
                +
              </button>
              {renderAddPopup(false)}
            </div>
          )}
        </div>
      </div>

      {!collapsed && dailyUpdatesCount > 0 && onDailyUpdatesClick && (
        <div className="daily-updates-banner">
          <button
            type="button"
            className="btn-daily-updates"
            onClick={onDailyUpdatesClick}
          >
            <span>Today&apos;s Numbers</span>
            <span className="daily-updates-count">{dailyUpdatesCount}</span>
          </button>
        </div>
      )}

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
            title="Show finances list"
          >
            Finances ({accounts.length})
          </button>
        </div>
      ) : (
        <div className="accounts-list">
          {accounts.length === 0 && !showBills && !showIncomes && perspectives.length === 0 ? (
            <p className="accounts-empty">No finances yet. Click + to add one.</p>
          ) : (
            <>
              {accounts.length > 0 && (
                <>
                  <button
                    type="button"
                    className="accounts-section-toggle"
                    onClick={() => setAccountsCollapsed((prev) => !prev)}
                    aria-expanded={!accountsCollapsed}
                  >
                    <span>Accounts ({accounts.length})</span>
                    <span className="accounts-section-chevron" aria-hidden="true">
                      {accountsCollapsed ? '▸' : '▾'}
                    </span>
                  </button>
                  {!accountsCollapsed &&
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
                    })}
                </>
              )}
            </>
          )}
          {((showBills && onSelectBills) || (showIncomes && onSelectIncomes)) && (
            <>
              <button
                type="button"
                className="accounts-section-toggle"
                onClick={() => setRecurringCollapsed((prev) => !prev)}
                aria-expanded={!recurringCollapsed}
              >
                <span>Recurring Transactions</span>
                <span className="accounts-section-chevron" aria-hidden="true">
                  {recurringCollapsed ? '▸' : '▾'}
                </span>
              </button>
              {!recurringCollapsed && (
                <>
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
                  {showIncomes && onSelectIncomes && (
                    <button
                      type="button"
                      className={`account-card account-card-incomes ${incomesSelected ? 'selected' : ''}`}
                      onClick={onSelectIncomes}
                    >
                      <div className="account-card-header">
                        <span className="account-card-name">Income</span>
                        <span className="badge-account-type badge-incomes">
                          {incomesCount} {incomesCount === 1 ? 'entry' : 'entries'}
                        </span>
                      </div>
                      <p className="account-card-desc">Recurring income &amp; reminders</p>
                    </button>
                  )}
                </>
              )}
            </>
          )}
          {perspectives.length > 0 && onSelectPerspective && (
            <>
              <button
                type="button"
                className="accounts-section-toggle"
                onClick={() => setPerspectivesCollapsed((prev) => !prev)}
                aria-expanded={!perspectivesCollapsed}
              >
                <span>Perspectives ({perspectives.length})</span>
                <span className="accounts-section-chevron" aria-hidden="true">
                  {perspectivesCollapsed ? '▸' : '▾'}
                </span>
              </button>
              {!perspectivesCollapsed &&
                perspectives.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className={`account-card account-card-perspective ${selectedPerspectiveId === p.id ? 'selected' : ''}`}
                    onClick={() => onSelectPerspective(p)}
                  >
                    <div className="account-card-header">
                      <span className="account-card-name">{p.name}</span>
                      <span className="badge-account-type badge-perspective">perspective</span>
                    </div>
                    <p className="account-card-desc">{p.description || 'Aggregated viewing window'}</p>
                  </button>
                ))}
            </>
          )}
        </div>
      )}
    </aside>
  );
}
