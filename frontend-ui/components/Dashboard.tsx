'use client';

import { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate, useParams, useMatch } from 'react-router-dom';
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

interface DashboardProps {
  session: UserSession;
  onUserDeleted: () => void;
}

interface AccountRouteViewProps {
  accounts: Account[];
  accountsLoaded: boolean;
  token: string;
  onAccountUpdated: (updated: Account) => void;
  onAccountDeleted: (deletedId: string) => void;
  accountRefreshVersion: number;
}

function AccountRouteView({
  accounts,
  accountsLoaded,
  token,
  onAccountUpdated,
  onAccountDeleted,
  accountRefreshVersion,
}: AccountRouteViewProps) {
  const { accountId, tab } = useParams<{ accountId: string; tab?: string }>();
  const navigate = useNavigate();
  const account = accounts.find((a) => a.id === accountId);

  if (!accountsLoaded) {
    return <div className="card empty-prompt"><p>Loading account...</p></div>;
  }

  if (!account) {
    return (
      <div className="card empty-prompt">
        <p>Account not found or you do not have access.</p>
        <button type="button" className="btn-secondary btn-sm" onClick={() => navigate('/')}>
          Go to Accounts
        </button>
      </div>
    );
  }

  const handleTabChange = (nextTab: string, replace = false) => {
    navigate(`/accounts/${account.id}/${nextTab}`, { replace });
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

function DashboardIndex({ accounts, accountsLoaded }: { accounts: Account[]; accountsLoaded: boolean }) {
  if (!accountsLoaded) {
    return <div className="card empty-prompt"><p>Loading accounts...</p></div>;
  }
  if (accounts.length > 0) {
    return <Navigate to={`/accounts/${accounts[0].id}`} replace />;
  }
  return (
    <div className="card empty-prompt">
      <p>Select an account from the left pane or click <strong>+</strong> to create an account.</p>
    </div>
  );
}

export function Dashboard({ session, onUserDeleted }: DashboardProps) {
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
  const navigate = useNavigate();
  const accountMatch = useMatch('/accounts/:accountId/*');
  const perspectiveMatch = useMatch('/perspectives/:perspectiveId');
  const billsMatch = useMatch('/bills');
  const incomesMatch = useMatch('/incomes');
  const matchedId = accountMatch?.params.accountId;
  const selectedId = matchedId && matchedId !== 'new' ? matchedId : null;
  const matchedPerspectiveId = perspectiveMatch?.params.perspectiveId;
  const selectedPerspectiveId =
    matchedPerspectiveId && matchedPerspectiveId !== 'new' ? matchedPerspectiveId : null;
  const bills = recurringTransactions.filter((tx) => tx.amount < 0);
  const incomes = recurringTransactions.filter((tx) => tx.amount > 0);
  const showBills = bills.length > 0 || billsActivated || Boolean(billsMatch);
  const showIncomes = incomes.length > 0 || incomesActivated || Boolean(incomesMatch);
  const isSubscriber = isJwtSubscriber(session.token);

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
    navigate(`/accounts/${newAcc.id}`);
  };

  const handleActivateBills = () => {
    setBillsActivated(true);
    navigate('/bills');
  };

  const handleActivateIncomes = () => {
    setIncomesActivated(true);
    navigate('/incomes');
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
    navigate(remaining.length > 0 ? `/accounts/${remaining[0].id}` : '/', { replace: true });
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
      navigate(`/accounts/${accounts[0].id}`);
    } else if (showBills) {
      navigate('/bills');
    } else if (showIncomes) {
      navigate('/incomes');
    } else {
      navigate('/');
    }
  };

  return (
    <div className="dashboard-layout">
      <AccountsPane
        accounts={accounts}
        selectedId={selectedId}
        onSelectAccount={(acc) => navigate(`/accounts/${acc.id}`)}
        onNewAccountClick={() => navigate('/accounts/new')}
        perspectives={perspectives}
        selectedPerspectiveId={selectedPerspectiveId}
        onSelectPerspective={(p) => navigate(`/perspectives/${p.id}`)}
        onNewPerspectiveClick={() => navigate('/perspectives/new')}
        showBills={showBills}
        billsSelected={Boolean(billsMatch)}
        billsCount={bills.length}
        onSelectBills={() => navigate('/bills')}
        onAddBillsClick={handleActivateBills}
        showIncomes={showIncomes}
        incomesSelected={Boolean(incomesMatch)}
        incomesCount={incomes.length}
        onSelectIncomes={() => navigate('/incomes')}
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
        <Routes>
          <Route path="/" element={<DashboardIndex accounts={accounts} accountsLoaded={accountsLoaded} />} />
          <Route
            path="/user"
            element={
              <UserDetailsPane
                session={session}
                accounts={accounts}
                onUserDeleted={onUserDeleted}
                onBackToAccounts={handleCancelCreate}
              />
            }
          />
          <Route
            path="/accounts/new"
            element={
              <NewAccountForm
                token={session.token}
                onSuccess={handleCreated}
                onCancel={handleCancelCreate}
              />
            }
          />
          <Route
            path="/perspectives/new"
            element={
              <NewPerspectiveForm
                token={session.token}
                accounts={accounts}
                recurringTransactions={recurringTransactions}
                onSuccess={(created) => {
                  setPerspectives((prev) => [created, ...prev]);
                  navigate(`/perspectives/${created.id}`);
                }}
                onCancel={handleCancelCreate}
              />
            }
          />
          <Route
            path="/perspectives/:perspectiveId"
            element={
              selectedPerspectiveId ? (
                <PerspectiveDetail
                  perspectiveId={selectedPerspectiveId}
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
                      navigate(`/perspectives/${remaining[0].id}`, { replace: true });
                    } else {
                      handleCancelCreate();
                    }
                  }}
                />
              ) : (
                <Navigate to="/" replace />
              )
            }
          />
          <Route
            path="/bills"
            element={
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
            }
          />
          <Route
            path="/incomes"
            element={
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
            }
          />
          <Route
            path="/accounts/:accountId"
            element={
              <AccountRouteView
                accounts={accounts}
                accountsLoaded={accountsLoaded}
                token={session.token}
                onAccountUpdated={handleAccountUpdated}
                onAccountDeleted={handleAccountDeleted}
                accountRefreshVersion={(matchedId ? accountRefreshVersions[matchedId] || 0 : 0) + dailyUpdatesRefreshVersion}
              />
            }
          />
          <Route
            path="/accounts/:accountId/:tab"
            element={
              <AccountRouteView
                accounts={accounts}
                accountsLoaded={accountsLoaded}
                token={session.token}
                onAccountUpdated={handleAccountUpdated}
                onAccountDeleted={handleAccountDeleted}
                accountRefreshVersion={(matchedId ? accountRefreshVersions[matchedId] || 0 : 0) + dailyUpdatesRefreshVersion}
              />
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
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
