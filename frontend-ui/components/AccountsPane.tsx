import { Account } from '../lib/types';

interface AccountsPaneProps {
  accounts: Account[];
  selectedId: string | null;
  onSelectAccount: (account: Account) => void;
  onNewAccountClick: () => void;
}

export function AccountsPane({
  accounts,
  selectedId,
  onSelectAccount,
  onNewAccountClick,
}: AccountsPaneProps) {
  return (
    <aside className="accounts-pane">
      <div className="accounts-pane-header">
        <h2 className="pane-title">Accounts</h2>
        <button
          type="button"
          className="btn-plus"
          onClick={onNewAccountClick}
          title="Create New Account"
          aria-label="Create New Account"
        >
          +
        </button>
      </div>
      <div className="accounts-list">
        {accounts.length === 0 ? (
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
      </div>
    </aside>
  );
}
