'use client';

import { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate, useParams, useMatch } from 'react-router-dom';
import { Account, Bill, BillCreateRequest, Category, UserSession } from '../lib/types';
import {
  applyBillToAccount,
  createBill,
  deleteBill,
  fetchAccounts,
  fetchBills,
  fetchCategories,
  updateBill,
} from '../lib/api';
import { getLocalTodayIso } from '../lib/bills';
import { AccountsPane } from './AccountsPane';
import { NewAccountForm } from './NewAccountForm';
import { AccountDetail } from './AccountDetail';
import { SnapshotAccountDetail } from './SnapshotAccountDetail';
import { AssetAccountDetail } from './AssetAccountDetail';
import { BillsPane } from './BillsPane';
import { BillReminderBanners } from './BillReminderBanners';

interface DashboardProps {
  session: UserSession;
}

interface AccountRouteViewProps {
  accounts: Account[];
  accountsLoaded: boolean;
  token: string;
  onAccountDeleted: (deletedId: string) => void;
  accountRefreshVersion: number;
}

function AccountRouteView({
  accounts,
  accountsLoaded,
  token,
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
        key={account.id}
        account={account}
        token={token}
        urlTab={tab}
        onTabChange={handleTabChange}
        onAccountDeleted={onAccountDeleted}
      />
    );
  }

  if (account.type === 'asset' || account.type === 'assets') {
    return (
      <AssetAccountDetail
        key={account.id}
        account={account}
        token={token}
        urlTab={tab}
        onTabChange={handleTabChange}
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

export function Dashboard({ session }: DashboardProps) {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [accountsLoaded, setAccountsLoaded] = useState(false);
  const [bills, setBills] = useState<Bill[]>([]);
  const [billsActivated, setBillsActivated] = useState(false);
  const [categories, setCategories] = useState<Category[]>([]);
  const [hiddenReminderIds, setHiddenReminderIds] = useState<Record<string, boolean>>({});
  const [applyingBillId, setApplyingBillId] = useState<string | null>(null);
  const [reminderError, setReminderError] = useState('');
  const [accountRefreshVersions, setAccountRefreshVersions] = useState<Record<string, number>>({});
  const navigate = useNavigate();
  const accountMatch = useMatch('/accounts/:accountId/*');
  const billsMatch = useMatch('/bills');
  const matchedId = accountMatch?.params.accountId;
  const selectedId = matchedId && matchedId !== 'new' ? matchedId : null;
  const showBills = bills.length > 0 || billsActivated || Boolean(billsMatch);

  useEffect(() => {
    let cancelled = false;
    setBillsActivated(false);
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

    fetchBills(session.token)
      .then((list) => {
        if (!cancelled) setBills(list);
      })
      .catch(() => {
        // Bills load failure handled gracefully
      });

    fetchCategories(session.token)
      .then((list) => {
        if (!cancelled) setCategories(list);
      })
      .catch(() => {
        // Categories load failure handled gracefully
      });

    return () => {
      cancelled = true;
    };
  }, [session.token]);

  useEffect(() => {
    if (!billsMatch) return;
    setBillsActivated(true);
    let cancelled = false;
    fetchCategories(session.token)
      .then((list) => {
        if (!cancelled) setCategories(list);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [billsMatch, session.token]);

  const handleCreated = (newAcc: Account) => {
    setAccounts((prev) => [newAcc, ...prev]);
    navigate(`/accounts/${newAcc.id}`);
  };

  const handleActivateBills = () => {
    setBillsActivated(true);
    navigate('/bills');
  };

  const handleAccountDeleted = (deletedId: string) => {
    const remaining = accounts.filter((a) => a.id !== deletedId);
    setAccounts(remaining);
    setBills((prev) =>
      prev.map((b) =>
        b.account_id === deletedId ? { ...b, account_id: null, category_id: null } : b
      )
    );
    navigate(remaining.length > 0 ? `/accounts/${remaining[0].id}` : '/', { replace: true });
  };

  const handleCreateBill = async (payload: BillCreateRequest) => {
    const created = await createBill(session.token, payload);
    setBills((prev) => [...prev, created]);
    setBillsActivated(true);
  };

  const handleUpdateBill = async (billId: string, payload: BillCreateRequest) => {
    const updated = await updateBill(session.token, billId, payload);
    setBills((prev) => prev.map((b) => (b.id === billId ? updated : b)));
  };

  const handleDeleteBill = async (billId: string) => {
    await deleteBill(session.token, billId);
    setBills((prev) => prev.filter((b) => b.id !== billId));
  };

  const handleApplyBill = async (bill: Bill) => {
    setReminderError('');
    setApplyingBillId(bill.id);
    try {
      const targetDate = bill.next_due_date || getLocalTodayIso();
      const res = await applyBillToAccount(session.token, bill.id, { date: targetDate });
      setBills((prev) => prev.map((b) => (b.id === bill.id ? res.bill : b)));
      setAccountRefreshVersions((prev) => ({
        ...prev,
        [res.record.account_id]: (prev[res.record.account_id] || 0) + 1,
      }));
    } catch (err: unknown) {
      setReminderError(err instanceof Error ? err.message : 'Failed to apply bill to account');
    } finally {
      setApplyingBillId(null);
    }
  };

  const handleHideReminder = (billId: string) => {
    setHiddenReminderIds((prev) => ({ ...prev, [billId]: true }));
  };

  const handleCancelCreate = () => {
    if (accounts.length > 0) {
      navigate(`/accounts/${accounts[0].id}`);
    } else if (showBills) {
      navigate('/bills');
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
        showBills={showBills}
        billsSelected={Boolean(billsMatch)}
        billsCount={bills.length}
        onSelectBills={() => navigate('/bills')}
        onAddBillsClick={handleActivateBills}
      />
      <main className="dashboard-content">
        {reminderError && <div className="alert-error">{reminderError}</div>}
        <BillReminderBanners
          bills={bills}
          accounts={accounts}
          hiddenIds={hiddenReminderIds}
          applyingId={applyingBillId}
          onApply={handleApplyBill}
          onHide={handleHideReminder}
        />
        <Routes>
          <Route path="/" element={<DashboardIndex accounts={accounts} accountsLoaded={accountsLoaded} />} />
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
            path="/bills"
            element={
              <BillsPane
                bills={bills}
                accounts={accounts}
                categories={categories}
                applyingId={applyingBillId}
                onCreateBill={handleCreateBill}
                onUpdateBill={handleUpdateBill}
                onDeleteBill={handleDeleteBill}
                onApplyBill={handleApplyBill}
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
                onAccountDeleted={handleAccountDeleted}
                accountRefreshVersion={matchedId ? accountRefreshVersions[matchedId] || 0 : 0}
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
                onAccountDeleted={handleAccountDeleted}
                accountRefreshVersion={matchedId ? accountRefreshVersions[matchedId] || 0 : 0}
              />
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}
