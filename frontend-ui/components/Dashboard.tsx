'use client';

import { useState, useEffect } from 'react';
import { Account, UserSession } from '../lib/types';
import { fetchAccounts } from '../lib/api';
import { AccountsPane } from './AccountsPane';
import { NewAccountForm } from './NewAccountForm';
import { AccountDetail } from './AccountDetail';
import { SnapshotAccountDetail } from './SnapshotAccountDetail';

interface DashboardProps {
  session: UserSession;
}

export function Dashboard({ session }: DashboardProps) {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<Account | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  const loadAccounts = async () => {
    try {
      const list = await fetchAccounts(session.token);
      setAccounts(list);
      if (list.length > 0 && !selected) {
        setSelected(list[0]);
      }
    } catch {
      // Network or empty load handled gracefully
    }
  };

  useEffect(() => {
    loadAccounts();
  }, [session.token]);

  const handleCreated = (newAcc: Account) => {
    setAccounts((prev) => [newAcc, ...prev]);
    setSelected(newAcc);
    setIsCreating(false);
  };

  return (
    <div className="dashboard-layout">
      <AccountsPane
        accounts={accounts}
        selectedId={selected?.id ?? null}
        onSelectAccount={(acc) => {
          setSelected(acc);
          setIsCreating(false);
        }}
        onNewAccountClick={() => setIsCreating(true)}
      />
      <main className="dashboard-content">
        {isCreating ? (
          <NewAccountForm
            token={session.token}
            onSuccess={handleCreated}
            onCancel={() => setIsCreating(false)}
          />
        ) : selected ? (
          selected.type === 'snapshot' ? (
            <SnapshotAccountDetail account={selected} token={session.token} />
          ) : (
            <AccountDetail account={selected} token={session.token} />
          )
        ) : (
          <div className="card empty-prompt">
            <p>Select an account from the left pane or click <strong>+</strong> to create an account.</p>
          </div>
        )}
      </main>
    </div>
  );
}
