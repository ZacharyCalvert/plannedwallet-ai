'use client';

import { useState } from 'react';
import { Account } from '../lib/types';
import { deleteAccount } from '../lib/api';

interface DeleteAccountModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  account: Account;
  onSuccess: (deletedId: string) => void;
}

export function DeleteAccountModal({
  isOpen,
  onClose,
  token,
  account,
  onSuccess,
}: DeleteAccountModalProps) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const handleDelete = async () => {
    setDeleting(true);
    setError('');
    try {
      await deleteAccount(token, account.id);
      onSuccess(account.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete account');
      setDeleting(false);
    }
  };

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-account-modal-title"
    >
      <div className="modal-content modal-danger">
        <div className="modal-header">
          <h3 id="delete-account-modal-title">Delete Account</h3>
          <button
            type="button"
            className="btn-close"
            onClick={onClose}
            disabled={deleting}
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="modal-body">
          <div className="warning-banner">
            <span className="warning-icon" aria-hidden="true">⚠️</span>
            <div>
              <strong>Warning: This action is permanent and cannot be undone!</strong>
            </div>
          </div>

          <p className="mt-3">
            Are you sure you want to delete <strong>{account.name}</strong>?
          </p>

          <p className="text-muted font-sm mt-2">
            Deleting this account will permanently remove all associated transaction records,
            holdings, snapshots, and perspective links.
          </p>

          {error && <p className="text-danger font-sm mt-3">{error}</p>}
        </div>

        <div className="modal-actions mt-4">
          <button
            type="button"
            className="btn-secondary"
            onClick={onClose}
            disabled={deleting}
          >
            Cancel
          </button>
          <button
            type="button"
            className="btn-danger"
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? 'Deleting...' : 'Yes, Delete Account'}
          </button>
        </div>
      </div>
    </div>
  );
}
