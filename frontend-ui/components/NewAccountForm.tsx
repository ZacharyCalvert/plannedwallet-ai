'use client';

import { useState } from 'react';
import { createCheckingAccount } from '../lib/api';
import { Account } from '../lib/types';

interface NewAccountFormProps {
  token: string;
  onSuccess: (newAccount: Account) => void;
  onCancel: () => void;
}

export function NewAccountForm({ token, onSuccess, onCancel }: NewAccountFormProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const created = await createCheckingAccount(token, name.trim(), description.trim());
      onSuccess(created);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create account');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card new-account-card">
      <div className="new-account-header">
        <h3>Create New Account</h3>
        <span className="badge-checking">Checking Only</span>
      </div>
      {error && <div className="alert-error">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="account-name">Account Name</label>
          <input
            id="account-name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Primary Everyday Checking"
          />
        </div>
        <div className="form-group">
          <label htmlFor="account-type">Account Type</label>
          <input id="account-type" type="text" value="checking" disabled readOnly />
          <p className="form-hint">Currently, only checking accounts are supported.</p>
        </div>
        <div className="form-group">
          <label htmlFor="account-desc">Description</label>
          <textarea
            id="account-desc"
            required
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="e.g. Household bills and everyday purchases"
          />
        </div>
        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={onCancel}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? 'Creating...' : 'Create Account'}
          </button>
        </div>
      </form>
    </div>
  );
}
