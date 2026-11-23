'use client';

import { useState, useEffect } from 'react';
import {
  Account,
  Category,
  Perspective,
  RecurringTransaction,
  RecurringTransactionRequest,
  UserSession,
} from '../lib/types';
import {
  applyRecurringTransactionToAccount,
  createRecurringTransaction,
  deleteRecurringTransaction,
  fetchAccounts,
  fetchCategories,
  fetchPerspectives,
  fetchRecurringTransactions,
  updateRecurringTransaction,
} from '../lib/api';
import { getLocalTodayIso } from '../lib/bills';
import { DailyUpdateItem } from '../lib/dailyUpdates';
import { loadPendingDailyUpdates } from '../lib/dailyUpdatesApi';
import { isJwtSubscriber } from '../lib/tokenRefresh';
import { AccountsPane } from './AccountsPane';
import { DailyUpdatesModal } from './DailyUpdatesModal';
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
  const [view, setView] = useState<DashboardView>({ kind: 'home' });

  const selectedId = view.kind === 'account' ? view.accountId : null;
  const selectedPerspectiveId = view.kind === 'perspective' ? view.perspectiveId : null;
  const billsMatch = view.kind === 'bills';
  const incomesMatch = view.kind === 'incomes';
  const bills = recurringTransactions.filter((tx) => tx.amount < 0);
  const incomes = recurringTransactions.filter((tx) => tx.amount > 0);
  const showBills = bills.length > 0 || billsActivated || billsMatch;
  const showIncomes = incomes.length > 0 || incomesActivated || incomesMatch;
  const isSubscriber = isJwtSubscriber(session.token);

  useEffect(() => {
    if (openUserTrigger > 0) {
      setView({ kind: 'user' });
    }
  }, [openUserTrigger]);

  useEffect(() => {
    if (view.kind === 'home' && accountsLoaded && accounts.length > 0) {
      setView({ kind: 'account', accountId: accounts[0].id });
    }
  }, [view.kind, accountsLoaded, accounts]);

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

  return (
    <div className="dashboard-layout">
      <AccountsPane
        accounts={accounts}
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
        dailyUpdatesCount={isSubscriber ? dailyUpdateItems.length : 0}
        onDailyUpdatesClick={() => setDailyUpdatesOpen(true)}
      />
      <main className="dashboard-content">
        {reminderError && <div className="alert-error">{reminderError}</div>}
        <BillReminderBanners
          transactions={recurringTransactions}
          accounts={accounts}
          hiddenIds={hiddenReminderIds}
          applyingId={applyingRecurringId}
          onApply={handleApplyRecurring}
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
          />
        )}
      </main>
      <DailyUpdatesModal
        isOpen={isSubscriber && dailyUpdatesOpen}
        onClose={() => setDailyUpdatesOpen(false)}
        token={session.token}
        todayIso={getLocalTodayIso()}
        items={dailyUpdateItems}
        onSaved={() => {
          setDailyUpdatesRefreshVersion((v) => v + 1);
        }}
      />
    </div>
  );
}
