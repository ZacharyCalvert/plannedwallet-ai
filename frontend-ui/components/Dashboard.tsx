'use client';

import { useState, useEffect } from 'react';
import { Account, Perspective, UserSession } from '../lib/types';
import { fetchAccounts, fetchPerspectives } from '../lib/api';
import { AccountsPane } from './AccountsPane';
import { NewAccountForm } from './NewAccountForm';
import { NewPerspectiveForm } from './NewPerspectiveForm';
import { AccountDetail } from './AccountDetail';
import { SnapshotAccountDetail } from './SnapshotAccountDetail';
import { AssetAccountDetail } from './AssetAccountDetail';
import { PerspectiveDetail } from './PerspectiveDetail';

interface DashboardProps {
  session: UserSession;
}

export function Dashboard({ session }: DashboardProps) {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [perspectives, setPerspectives] = useState<Perspective[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<Account | null>(null);
  const [selectedPerspective, setSelectedPerspective] = useState<Perspective | null>(null);
  const [activeCreation, setActiveCreation] = useState<'account' | 'perspective' | null>(null);

  const loadData = async () => {
    try {
      const [accList, perspList] = await Promise.all([
        fetchAccounts(session.token).catch(() => []),
        fetchPerspectives(session.token).catch(() => []),
      ]);
      setAccounts(accList);
      setPerspectives(perspList);

      // Default selection on first load if nothing is selected
      if (!selectedAccount && !selectedPerspective) {
        if (perspList.length > 0) {
          setSelectedPerspective(perspList[0]);
        } else if (accList.length > 0) {
          setSelectedAccount(accList[0]);
        }
      }
    } catch {
      // Graceful error handling
    }
  };

  useEffect(() => {
    loadData();
  }, [session.token]);

  const handleAccountCreated = (newAcc: Account) => {
    setAccounts((prev) => [newAcc, ...prev]);
    setSelectedAccount(newAcc);
    setSelectedPerspective(null);
    setActiveCreation(null);
  };

  const handlePerspectiveCreated = (newPersp: Perspective) => {
    setPerspectives((prev) => [newPersp, ...prev]);
    setSelectedPerspective(newPersp);
    setSelectedAccount(null);
    setActiveCreation(null);
  };

  const handlePerspectiveUpdated = (updated: Perspective) => {
    setPerspectives((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
    if (selectedPerspective?.id === updated.id) {
      setSelectedPerspective(updated);
    }
  };

  const handlePerspectiveDeleted = (deletedId: string) => {
    setPerspectives((prev) => prev.filter((p) => p.id !== deletedId));
    if (selectedPerspective?.id === deletedId) {
      setSelectedPerspective(null);
      if (accounts.length > 0) {
        setSelectedAccount(accounts[0]);
      }
    }
  };

  return (
    <div className="dashboard-layout">
      <AccountsPane
        accounts={accounts}
        selectedAccountId={selectedAccount?.id ?? null}
        onSelectAccount={(acc) => {
          setSelectedAccount(acc);
          setSelectedPerspective(null);
          setActiveCreation(null);
        }}
        perspectives={perspectives}
        selectedPerspectiveId={selectedPerspective?.id ?? null}
        onSelectPerspective={(p) => {
          setSelectedPerspective(p);
          setSelectedAccount(null);
          setActiveCreation(null);
        }}
        onNewAccountClick={() => {
          setActiveCreation('account');
          setSelectedAccount(null);
          setSelectedPerspective(null);
        }}
        onNewPerspectiveClick={() => {
          setActiveCreation('perspective');
          setSelectedAccount(null);
          setSelectedPerspective(null);
        }}
      />
      <main className="dashboard-content">
        {activeCreation === 'account' ? (
          <NewAccountForm
            token={session.token}
            onSuccess={handleAccountCreated}
            onCancel={() => setActiveCreation(null)}
          />
        ) : activeCreation === 'perspective' ? (
          <NewPerspectiveForm
            token={session.token}
            allAccounts={accounts}
            onSuccess={handlePerspectiveCreated}
            onCancel={() => setActiveCreation(null)}
          />
        ) : selectedPerspective ? (
          <PerspectiveDetail
            perspective={selectedPerspective}
            token={session.token}
            allAccounts={accounts}
            onPerspectiveUpdated={handlePerspectiveUpdated}
            onPerspectiveDeleted={handlePerspectiveDeleted}
          />
        ) : selectedAccount ? (
          selectedAccount.type === 'snapshot' ? (
            <SnapshotAccountDetail account={selectedAccount} token={session.token} />
          ) : selectedAccount.type === 'asset' || selectedAccount.type === 'assets' ? (
            <AssetAccountDetail account={selectedAccount} token={session.token} />
          ) : (
            <AccountDetail account={selectedAccount} token={session.token} />
          )
        ) : (
          <div className="card empty-prompt">
            <p>
              Select an account or perspective from the left pane or click <strong>+</strong> to add one.
            </p>
          </div>
        )}
      </main>
    </div>
  );
}
