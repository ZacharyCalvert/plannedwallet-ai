'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  Account,
  Category,
  PatchUserSettingsRequest,
  Perspective,
  PinnedBalanceRef,
  RecurringTransaction,
  RecurringTransactionRequest,
  UserSession,
  UserSettingsPayload,
} from '../lib/types';
import {
  applyRecurringTransactionToAccount,
  createRecurringTransaction,
  deleteRecurringTransaction,
  fetchAccounts,
  fetchCategories,
  fetchPerspectives,
  fetchRecurringTransactions,
  fetchUserSettings,
  patchUserSettings,
  updateRecurringTransaction,
} from '../lib/api';
import { getLocalTodayIso } from '../lib/bills';
import { DailyUpdateItem, sortDailyUpdateItems } from '../lib/dailyUpdates';
import { loadPendingDailyUpdates } from '../lib/dailyUpdatesApi';
import { isJwtSubscriber } from '../lib/tokenRefresh';
import { applyAccountOrder, isPinnedBalance, togglePinnedBalance } from '../lib/userSettings';
import { getMobileHeaderTitle, MOBILE_MEDIA_QUERY } from '../lib/mobileLayout';
import { AccountsPane } from './AccountsPane';
import { DailyUpdatesModal } from './DailyUpdatesModal';
import { PinnedBalancesBar } from './PinnedBalancesBar';
import { NewAccountForm } from './NewAccountForm';
import { NewPerspectiveForm } from './NewPerspectiveForm';
import { PerspectiveDetail } from './PerspectiveDetail';
import { AccountDetail } from './AccountDetail';
import { SnapshotAccountDetail } from './SnapshotAccountDetail';
import { AssetAccountDetail } from './AssetAccountDetail';
import { BillsPane } from './BillsPane';
import { IncomesPane } from './IncomesPane';
import { BillReminderBanners } from './BillReminderBanners';
import { UserDetailsPane } from './UserDetailsPane';

type DashboardView =
  | { kind: 'home' }
  | { kind: 'user' }
  | { kind: 'new-account' }
  | { kind: 'new-perspective' }
  | { kind: 'perspective'; perspectiveId: string }
  | { kind: 'bills' }
  | { kind: 'incomes' }
  | { kind: 'account'; accountId: string; tab?: string };

interface DashboardProps {
  session: UserSession;
  onUserDeleted: () => void;
  openUserTrigger?: number;
}

interface AccountRouteViewProps {
  accountId: string;
  tab?: string;
  accounts: Account[];
  accountsLoaded: boolean;
  token: string;
  onAccountUpdated: (updated: Account) => void;
  onAccountDeleted: (deletedId: string) => void;
  onTabChange: (accountId: string, nextTab: string) => void;
  onGoHome: () => void;
  accountRefreshVersion: number;
  showPinToggle?: boolean;
  isPinned?: boolean;
  onTogglePin?: () => void;
}

function AccountRouteView({
  accountId,
  tab,
  accounts,
  accountsLoaded,
  token,
  onAccountUpdated,
  onAccountDeleted,
  onTabChange,
  onGoHome,
  accountRefreshVersion,
  showPinToggle = false,
  isPinned = false,
  onTogglePin,
}: AccountRouteViewProps) {
  const account = accounts.find((a) => a.id === accountId);

  if (!accountsLoaded) {
    return <div className="card empty-prompt"><p>Loading account...</p></div>;
  }

  if (!account) {
    return (
      <div className="card empty-prompt">
        <p>Account not found or you do not have access.</p>
        <button type="button" className="btn-secondary btn-sm" onClick={onGoHome}>
          Go to Accounts
        </button>
      </div>
    );
  }

  const handleTabChange = (nextTab: string) => {
    onTabChange(account.id, nextTab);
  };

  if (account.type === 'snapshot') {
    return (
      <SnapshotAccountDetail
        key={`${account.id}-${accountRefreshVersion}`}
        account={account}
        token={token}
        urlTab={tab}
        onTabChange={handleTabChange}
        onAccountUpdated={onAccountUpdated}
        onAccountDeleted={onAccountDeleted}
        showPinToggle={showPinToggle}
        isPinned={isPinned}
        onTogglePin={onTogglePin}
      />
    );
  }

  if (account.type === 'asset' || account.type === 'assets') {
    return (
      <AssetAccountDetail
        key={`${account.id}-${accountRefreshVersion}`}
        account={account}
        token={token}
        urlTab={tab}
        onTabChange={handleTabChange}
        onAccountUpdated={onAccountUpdated}
        onAccountDeleted={onAccountDeleted}
        showPinToggle={showPinToggle}
        isPinned={isPinned}
        onTogglePin={onTogglePin}
      />
    );
  }

  return (
    <AccountDetail
      key={account.id}
      account={account}
      token={token}
      urlTab={tab}
      onTabChange={handleTabChange}
      onAccountUpdated={onAccountUpdated}
      onAccountDeleted={onAccountDeleted}
      refreshVersion={accountRefreshVersion}
      showPinToggle={showPinToggle}
      isPinned={isPinned}
      onTogglePin={onTogglePin}
    />
  );
}

function DashboardIndex({ accountsLoaded }: { accountsLoaded: boolean }) {
  if (!accountsLoaded) {
    return <div className="card empty-prompt"><p>Loading accounts...</p></div>;
  }
  return (
    <div className="card empty-prompt">
      <p>Select an account from the left pane or click <strong>+</strong> to create an account.</p>
    </div>
  );
}

export function Dashboard({ session, onUserDeleted, openUserTrigger = 0 }: DashboardProps) {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [accountsLoaded, setAccountsLoaded] = useState(false);
  const [perspectives, setPerspectives] = useState<Perspective[]>([]);
  const [recurringTransactions, setRecurringTransactions] = useState<RecurringTransaction[]>([]);
  const [billsActivated, setBillsActivated] = useState(false);
  const [incomesActivated, setIncomesActivated] = useState(false);
  const [categories, setCategories] = useState<Category[]>([]);
  const [hiddenReminderIds, setHiddenReminderIds] = useState<Record<string, boolean>>({});
  const [applyingRecurringId, setApplyingRecurringId] = useState<string | null>(null);
  const [reminderError, setReminderError] = useState('');
  const [accountRefreshVersions, setAccountRefreshVersions] = useState<Record<string, number>>({});
  const [dailyUpdateItems, setDailyUpdateItems] = useState<DailyUpdateItem[]>([]);
  const [dailyUpdatesOpen, setDailyUpdatesOpen] = useState(false);
  const [dailyUpdatesRefreshVersion, setDailyUpdatesRefreshVersion] = useState(0);
  const [userSettings, setUserSettings] = useState<UserSettingsPayload>({
    account_order: [],
    daily_update_order: [],
    pinned_balances: [],
  });
  const [view, setView] = useState<DashboardView>({ kind: 'home' });
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);

  const selectedId = view.kind === 'account' ? view.accountId : null;
  const selectedPerspectiveId = view.kind === 'perspective' ? view.perspectiveId : null;
  const billsMatch = view.kind === 'bills';
  const incomesMatch = view.kind === 'incomes';
  const bills = recurringTransactions.filter((tx) => tx.amount < 0);
  const incomes = recurringTransactions.filter((tx) => tx.amount > 0);
  const showBills = bills.length > 0 || billsActivated || billsMatch;
  const showIncomes = incomes.length > 0 || incomesActivated || incomesMatch;
  const isSubscriber = isJwtSubscriber(session.token);

  const orderedAccounts = useMemo(
    () => (isSubscriber ? applyAccountOrder(accounts, userSettings.account_order) : accounts),
    [isSubscriber, accounts, userSettings.account_order]
  );

  const orderedDailyUpdateItems = useMemo(
    () =>
      isSubscriber
        ? sortDailyUpdateItems(dailyUpdateItems, userSettings.daily_update_order)
        : [],
    [isSubscriber, dailyUpdateItems, userSettings.daily_update_order]
  );

  useEffect(() => {
    if (!isSubscriber) {
      setUserSettings({ account_order: [], daily_update_order: [], pinned_balances: [] });
      return;
    }
    let cancelled = false;
    fetchUserSettings(session.token, session.username)
      .then((res) => {
        if (!cancelled && res?.settings) {
          setUserSettings({
            account_order: res.settings.account_order || [],
            daily_update_order: res.settings.daily_update_order || [],
            pinned_balances: res.settings.pinned_balances || [],
          });
        }
      })
      .catch(() => {
        // Settings load failure handled gracefully
      });
    return () => {
      cancelled = true;
    };
  }, [isSubscriber, session.token, session.username]);

  const handlePatchSettings = (patch: PatchUserSettingsRequest) => {
    if (!isSubscriber) return;
    setUserSettings((prev) => ({
      account_order: patch.account_order ?? prev.account_order,
      daily_update_order: patch.daily_update_order ?? prev.daily_update_order,
      pinned_balances: patch.pinned_balances ?? prev.pinned_balances,
    }));
    patchUserSettings(session.token, session.username, patch)
      .then((res) => {
        if (res?.settings) {
          setUserSettings({
            account_order: res.settings.account_order || [],
            daily_update_order: res.settings.daily_update_order || [],
            pinned_balances: res.settings.pinned_balances || [],
          });
        }
      })
      .catch(() => {
        // Keep optimistic UI state on transient error
      });
  };

  const handleTogglePin = (ref: PinnedBalanceRef) => {
    const nextPinned = togglePinnedBalance(userSettings.pinned_balances, ref);
    handlePatchSettings({ pinned_balances: nextPinned });
  };

  useEffect(() => {
    if (openUserTrigger > 0) {
      setView({ kind: 'user' });
    }
  }, [openUserTrigger]);

  useEffect(() => {
    if (view.kind === 'home' && accountsLoaded && orderedAccounts.length > 0) {
      setView({ kind: 'account', accountId: orderedAccounts[0].id });
    }
  }, [view.kind, accountsLoaded, orderedAccounts]);

  useEffect(() => {
    if (typeof window !== 'undefined' && window.scrollY > 0) {
      window.scrollTo({ top: 0, behavior: 'auto' });
    }
  }, [view.kind, selectedId, selectedPerspectiveId]);

  useEffect(() => {
    setMobileDrawerOpen(false);
  }, [view, dailyUpdatesOpen, openUserTrigger]);

  useEffect(() => {
    if (!mobileDrawerOpen) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const mq = typeof window !== 'undefined' && window.matchMedia ? window.matchMedia(MOBILE_MEDIA_QUERY) : null;
    const handleViewportChange = (e: MediaQueryListEvent) => {
      if (!e.matches) {
        setMobileDrawerOpen(false);
      }
    };
    mq?.addEventListener('change', handleViewportChange);
    return () => {
      document.body.style.overflow = prevOverflow;
      mq?.removeEventListener('change', handleViewportChange);
    };
  }, [mobileDrawerOpen]);

  useEffect(() => {
    let cancelled = false;
    setBillsActivated(false);
    setIncomesActivated(false);
    setHiddenReminderIds({});
    setReminderError('');

    fetchAccounts(session.token)
      .then((list) => {
        if (!cancelled) setAccounts(list);
      })
      .catch(() => {
        // Network or empty load handled gracefully
      })
      .finally(() => {
        if (!cancelled) setAccountsLoaded(true);
      });

    fetchRecurringTransactions(session.token)
      .then((list) => {
        if (!cancelled) setRecurringTransactions(list);
      })
      .catch(() => {
        // Recurring transactions load failure handled gracefully
      });

    fetchCategories(session.token)
      .then((list) => {
        if (!cancelled) setCategories(list);
      })
      .catch(() => {
        // Categories load failure handled gracefully
      });

    fetchPerspectives(session.token)
      .then((list) => {
        if (!cancelled) setPerspectives(list);
      })
      .catch(() => {
        // Perspectives load failure handled gracefully
      });

    return () => {
      cancelled = true;
    };
  }, [session.username]);

  useEffect(() => {
    if (billsMatch) setBillsActivated(true);
    if (incomesMatch) setIncomesActivated(true);
    if (!billsMatch && !incomesMatch) return;
    let cancelled = false;
    fetchCategories(session.token)
      .then((list) => {
        if (!cancelled) setCategories(list);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [billsMatch, incomesMatch, session.username]);

  useEffect(() => {
    if (!accountsLoaded) return;
    if (!isSubscriber) {
      setDailyUpdateItems([]);
      setDailyUpdatesOpen(false);
      return;
    }
    let cancelled = false;
    loadPendingDailyUpdates(session.token, accounts, getLocalTodayIso())
      .then((items) => {
        if (!cancelled) setDailyUpdateItems(items);
      })
      .catch(() => {
        // Daily update check failure handled gracefully
      });
    return () => {
      cancelled = true;
    };
  }, [accountsLoaded, accounts, session.token, isSubscriber, dailyUpdatesRefreshVersion, selectedId]);

  const handleCreated = (newAcc: Account) => {
    setAccounts((prev) => [newAcc, ...prev]);
    setView({ kind: 'account', accountId: newAcc.id });
  };

  const handleActivateBills = () => {
    setBillsActivated(true);
    setView({ kind: 'bills' });
  };

  const handleActivateIncomes = () => {
    setIncomesActivated(true);
    setView({ kind: 'incomes' });
  };

  const handleAccountUpdated = (updated: Account) => {
    setAccounts((prev) => prev.map((a) => (a.id === updated.id ? updated : a)));
  };

  const handleAccountDeleted = (deletedId: string) => {
    const remaining = accounts.filter((a) => a.id !== deletedId);
    setAccounts(remaining);
    setRecurringTransactions((prev) =>
      prev.map((tx) =>
        tx.account_id === deletedId ? { ...tx, account_id: null, category_id: null } : tx
      )
    );
    setView(remaining.length > 0 ? { kind: 'account', accountId: remaining[0].id } : { kind: 'home' });
  };

  const handleCreateRecurring = async (payload: RecurringTransactionRequest) => {
    const created = await createRecurringTransaction(session.token, payload);
    setRecurringTransactions((prev) => [...prev, created]);
    if (created.amount < 0) {
      setBillsActivated(true);
    } else {
      setIncomesActivated(true);
    }
  };

  const handleUpdateRecurring = async (id: string, payload: RecurringTransactionRequest) => {
    const updated = await updateRecurringTransaction(session.token, id, payload);
    setRecurringTransactions((prev) => prev.map((tx) => (tx.id === id ? updated : tx)));
  };

  const handleDeleteRecurring = async (id: string) => {
    await deleteRecurringTransaction(session.token, id);
    setRecurringTransactions((prev) => prev.filter((tx) => tx.id !== id));
  };

  const handleApplyRecurring = async (tx: RecurringTransaction) => {
    setReminderError('');
    setApplyingRecurringId(tx.id);
    try {
      const targetDate = tx.next_due_date || getLocalTodayIso();
      const res = await applyRecurringTransactionToAccount(session.token, tx.id, {
        date: targetDate,
      });
      setRecurringTransactions((prev) =>
        prev.map((item) => (item.id === tx.id ? res.recurring_transaction : item))
      );
      setAccountRefreshVersions((prev) => ({
        ...prev,
        [res.record.account_id]: (prev[res.record.account_id] || 0) + 1,
      }));
    } catch (err: unknown) {
      setReminderError(
        err instanceof Error ? err.message : 'Failed to apply recurring transaction'
      );
    } finally {
      setApplyingRecurringId(null);
    }
  };

  const handleHideReminder = (id: string) => {
    setHiddenReminderIds((prev) => ({ ...prev, [id]: true }));
  };

  const handleMarkPaidRecurring = async (tx: RecurringTransaction) => {
    setReminderError('');
    setApplyingRecurringId(tx.id);
    try {
      const targetDate = tx.next_due_date || getLocalTodayIso();
      const updated = await updateRecurringTransaction(session.token, tx.id, {
        account_id: tx.account_id,
        category_id: tx.category_id,
        amount: tx.amount,
        last_date_applied: targetDate,
        frequency: tx.frequency,
        interval: tx.interval,
        schedule: tx.schedule,
        name: tx.name,
        entity: tx.entity,
        description: tx.description,
        reminder: tx.reminder,
      });
      setRecurringTransactions((prev) =>
        prev.map((item) => (item.id === tx.id ? updated : item))
      );
    } catch (err: unknown) {
      setReminderError(
        err instanceof Error ? err.message : 'Failed to mark recurring transaction as paid'
      );
    } finally {
      setApplyingRecurringId(null);
    }
  };

  const handleCancelCreate = () => {
    if (accounts.length > 0) {
      setView({ kind: 'account', accountId: accounts[0].id });
    } else if (showBills) {
      setView({ kind: 'bills' });
    } else if (showIncomes) {
      setView({ kind: 'incomes' });
    } else {
      setView({ kind: 'home' });
    }
  };

  const mobileHeaderTitle = getMobileHeaderTitle(view, accounts, perspectives);

  return (
    <div className="dashboard-layout">
      <div className="mobile-nav-bar">
        <button
          type="button"
          className="btn-mobile-menu"
          onClick={() => setMobileDrawerOpen(true)}
          aria-label="Open finances menu"
          aria-expanded={mobileDrawerOpen}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <line x1="3" y1="6" x2="21" y2="6" />
            <line x1="3" y1="12" x2="21" y2="12" />
            <line x1="3" y1="18" x2="21" y2="18" />
          </svg>
          <span>Finances</span>
          {isSubscriber && orderedDailyUpdateItems.length > 0 && (
            <span className="daily-updates-count">{orderedDailyUpdateItems.length}</span>
          )}
        </button>
        <span className="mobile-nav-title">{mobileHeaderTitle}</span>
      </div>
      <AccountsPane
        accounts={orderedAccounts}
        selectedId={selectedId}
        onSelectAccount={(acc) => setView({ kind: 'account', accountId: acc.id })}
        onNewAccountClick={() => setView({ kind: 'new-account' })}
        perspectives={perspectives}
        selectedPerspectiveId={selectedPerspectiveId}
        onSelectPerspective={(p) => setView({ kind: 'perspective', perspectiveId: p.id })}
        onNewPerspectiveClick={() => setView({ kind: 'new-perspective' })}
        showBills={showBills}
        billsSelected={billsMatch}
        billsCount={bills.length}
        onSelectBills={() => setView({ kind: 'bills' })}
        onAddBillsClick={handleActivateBills}
        showIncomes={showIncomes}
        incomesSelected={incomesMatch}
        incomesCount={incomes.length}
        onSelectIncomes={() => setView({ kind: 'incomes' })}
        onAddIncomesClick={handleActivateIncomes}
        dailyUpdatesCount={isSubscriber ? orderedDailyUpdateItems.length : 0}
        onDailyUpdatesClick={() => setDailyUpdatesOpen(true)}
        canReorderAccounts={isSubscriber}
        onReorderAccounts={(nextAccountOrder) => handlePatchSettings({ account_order: nextAccountOrder })}
        mobileOpen={mobileDrawerOpen}
        onMobileClose={() => setMobileDrawerOpen(false)}
      />
      <main className="dashboard-content">
        {isSubscriber && (
          <PinnedBalancesBar
            token={session.token}
            pinnedBalances={userSettings.pinned_balances}
            accounts={accounts}
            perspectives={perspectives}
            refreshVersion={dailyUpdatesRefreshVersion}
            onSelectAccount={(accountId) => setView({ kind: 'account', accountId })}
            onSelectPerspective={(perspectiveId) => setView({ kind: 'perspective', perspectiveId })}
            onUnpin={handleTogglePin}
          />
        )}
        {reminderError && <div className="alert-error">{reminderError}</div>}
        <BillReminderBanners
          transactions={recurringTransactions}
          accounts={accounts}
          hiddenIds={hiddenReminderIds}
          applyingId={applyingRecurringId}
          onApply={handleApplyRecurring}
          onMarkPaid={handleMarkPaidRecurring}
          onHide={handleHideReminder}
        />
        {view.kind === 'home' && <DashboardIndex accountsLoaded={accountsLoaded} />}
        {view.kind === 'user' && (
          <UserDetailsPane
            session={session}
            accounts={accounts}
            onUserDeleted={onUserDeleted}
            onBackToAccounts={handleCancelCreate}
          />
        )}
        {view.kind === 'new-account' && (
          <NewAccountForm
            token={session.token}
            onSuccess={handleCreated}
            onCancel={handleCancelCreate}
          />
        )}
        {view.kind === 'new-perspective' && (
          <NewPerspectiveForm
            token={session.token}
            accounts={accounts}
            recurringTransactions={recurringTransactions}
            onSuccess={(created) => {
              setPerspectives((prev) => [created, ...prev]);
              setView({ kind: 'perspective', perspectiveId: created.id });
            }}
            onCancel={handleCancelCreate}
          />
        )}
        {view.kind === 'perspective' && (
          <PerspectiveDetail
            perspectiveId={view.perspectiveId}
            token={session.token}
            allAccounts={accounts}
            allRecurringTransactions={recurringTransactions}
            categories={categories}
            onPerspectiveUpdated={(updated) =>
              setPerspectives((prev) => prev.map((p) => (p.id === updated.id ? updated : p)))
            }
            onPerspectiveDeleted={(deletedId) => {
              const remaining = perspectives.filter((p) => p.id !== deletedId);
              setPerspectives(remaining);
              if (remaining.length > 0) {
                setView({ kind: 'perspective', perspectiveId: remaining[0].id });
              } else {
                handleCancelCreate();
              }
            }}
            showPinToggle={isSubscriber}
            isPinned={isPinnedBalance(userSettings.pinned_balances, 'perspective', view.perspectiveId)}
            onTogglePin={() => handleTogglePin({ kind: 'perspective', id: view.perspectiveId })}
          />
        )}
        {view.kind === 'bills' && (
          <BillsPane
            bills={bills}
            accounts={accounts}
            categories={categories}
            applyingId={applyingRecurringId}
            onCreateBill={handleCreateRecurring}
            onUpdateBill={handleUpdateRecurring}
            onDeleteBill={handleDeleteRecurring}
            onApplyBill={handleApplyRecurring}
          />
        )}
        {view.kind === 'incomes' && (
          <IncomesPane
            incomes={incomes}
            accounts={accounts}
            categories={categories}
            applyingId={applyingRecurringId}
            onCreateIncome={handleCreateRecurring}
            onUpdateIncome={handleUpdateRecurring}
            onDeleteIncome={handleDeleteRecurring}
            onApplyIncome={handleApplyRecurring}
          />
        )}
        {view.kind === 'account' && (
          <AccountRouteView
            accountId={view.accountId}
            tab={view.tab}
            accounts={accounts}
            accountsLoaded={accountsLoaded}
            token={session.token}
            onAccountUpdated={handleAccountUpdated}
            onAccountDeleted={handleAccountDeleted}
            onTabChange={(accountId, nextTab) =>
              setView({ kind: 'account', accountId, tab: nextTab })
            }
            onGoHome={() => setView({ kind: 'home' })}
            accountRefreshVersion={(accountRefreshVersions[view.accountId] || 0) + dailyUpdatesRefreshVersion}
            showPinToggle={isSubscriber}
            isPinned={isPinnedBalance(userSettings.pinned_balances, 'account', view.accountId)}
            onTogglePin={() => handleTogglePin({ kind: 'account', id: view.accountId })}
          />
        )}
      </main>
      <DailyUpdatesModal
        isOpen={isSubscriber && dailyUpdatesOpen}
        onClose={() => setDailyUpdatesOpen(false)}
        token={session.token}
        todayIso={getLocalTodayIso()}
        items={orderedDailyUpdateItems}
        canReorder={isSubscriber}
        onReorder={(nextDailyOrder) => handlePatchSettings({ daily_update_order: nextDailyOrder })}
        onSaved={() => {
          setDailyUpdatesRefreshVersion((v) => v + 1);
        }}
      />
    </div>
  );
}
