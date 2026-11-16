'use client';

import { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate, useParams, useMatch } from 'react-router-dom';
import { Account, UserSession } from '../lib/types';
import { fetchAccounts } from '../lib/api';
import { AccountsPane } from './AccountsPane';
import { NewAccountForm } from './NewAccountForm';
import { AccountDetail } from './AccountDetail';
import { SnapshotAccountDetail } from './SnapshotAccountDetail';
import { AssetAccountDetail } from './AssetAccountDetail';

interface DashboardProps {
  session: UserSession;
}

interface AccountRouteViewProps {
  accounts: Account[];
  accountsLoaded: boolean;
  token: string;
  onAccountDeleted: (deletedId: string) => void;
}

function AccountRouteView({
  accounts,
  accountsLoaded,
  token,
  onAccountDeleted,
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
      onAccountDeleted={onAccountDeleted}
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
  const navigate = useNavigate();
  const accountMatch = useMatch('/accounts/:accountId/*');
  const matchedId = accountMatch?.params.accountId;
  const selectedId = matchedId && matchedId !== 'new' ? matchedId : null;

  useEffect(() => {
    let cancelled = false;
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
    return () => {
      cancelled = true;
    };
  }, [session.token]);

  const handleCreated = (newAcc: Account) => {
    setAccounts((prev) => [newAcc, ...prev]);
    navigate(`/accounts/${newAcc.id}`);
  };

  const handleAccountDeleted = (deletedId: string) => {
    const remaining = accounts.filter((a) => a.id !== deletedId);
    setAccounts(remaining);
    navigate(remaining.length > 0 ? `/accounts/${remaining[0].id}` : '/', { replace: true });
  };

  const handleCancelCreate = () => {
    navigate(accounts.length > 0 ? `/accounts/${accounts[0].id}` : '/');
  };

  return (
    <div className="dashboard-layout">
      <AccountsPane
        accounts={accounts}
        selectedId={selectedId}
        onSelectAccount={(acc) => navigate(`/accounts/${acc.id}`)}
        onNewAccountClick={() => navigate('/accounts/new')}
      />
      <main className="dashboard-content">
        <Routes>
          <Route path="/" element={<DashboardIndex accounts={accounts} accountsLoaded={accountsLoaded} />} />
          <Route
            path="/accounts/new"
            element={<NewAccountForm token={session.token} onSuccess={handleCreated} onCancel={handleCancelCreate} />}
          />
          <Route
            path="/accounts/:accountId"
            element={
              <AccountRouteView
                accounts={accounts}
                accountsLoaded={accountsLoaded}
                token={session.token}
                onAccountDeleted={handleAccountDeleted}
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
              />
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}
