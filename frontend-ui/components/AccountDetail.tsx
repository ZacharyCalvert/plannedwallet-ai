import { Account } from '../lib/types';

interface AccountDetailProps {
  account: Account;
}

export function AccountDetail({ account }: AccountDetailProps) {
  return (
    <div className="card account-detail-card">
      <div className="account-detail-header">
        <div>
          <h2>{account.name}</h2>
          <span className="badge-checking">{account.type}</span>
        </div>
        <p className="account-id-label">ID: {account.id}</p>
      </div>
      <div className="account-detail-body">
        <div className="detail-item">
          <strong>Description:</strong>
          <p>{account.description}</p>
        </div>
        <div className="detail-item">
          <strong>Created:</strong>
          <p>{new Date(account.created_at).toLocaleDateString()}</p>
        </div>
        <div className="detail-item">
          <strong>Account Type:</strong>
          <p>Standard Checking Account</p>
        </div>
      </div>
    </div>
  );
}
