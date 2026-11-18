'use client';

import { useState, useEffect } from 'react';
import { UserProfile, UserSession } from '../lib/types';
import { deleteUserAccount, fetchUserProfile } from '../lib/api';

interface UserDetailsPaneProps {
  session: UserSession;
  onUserDeleted: () => void;
  onBackToAccounts: () => void;
}

export function UserDetailsPane({
  session,
  onUserDeleted,
  onBackToAccounts,
}: UserDetailsPaneProps) {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    fetchUserProfile(session.token, session.username)
      .then((data) => {
        if (!cancelled) {
          setProfile(data);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load user details');
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [session.username]);

  const handleConfirmDelete = async () => {
    setDeleting(true);
    setDeleteError('');
    try {
      await deleteUserAccount(session.token, session.username);
      onUserDeleted();
    } catch (err: unknown) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete user account');
      setDeleting(false);
    }
  };

  return (
    <div className="card account-detail-card">
      <div className="account-detail-header">
        <div className="account-detail-info">
          <div className="account-title-row">
            <h2>User Details</h2>
          </div>
          <p className="account-desc-text">
            Manage your profile information and permanent user account deletion.
          </p>
        </div>
        <div className="user-details-header-actions">
          <button type="button" className="btn-secondary btn-sm" onClick={onBackToAccounts}>
            Back to Accounts
          </button>
        </div>
      </div>

      {error && <div className="alert-error mt-3">{error}</div>}

      {loading ? (
        <p className="text-muted mt-3">Loading user details...</p>
      ) : (
        <div className="user-details-section">
          <dl className="user-details-grid">
            <div className="user-details-item">
              <dt>Username</dt>
              <dd>{profile?.username || session.username}</dd>
            </div>
            <div className="user-details-item">
              <dt>Email Address</dt>
              <dd>{profile?.email || 'Unavailable'}</dd>
            </div>
          </dl>

          <div className="user-danger-zone">
            <div>
              <h3>Delete User</h3>
              <p className="text-muted font-sm">
                Permanently remove your user profile and all associated accounts and records.
              </p>
            </div>
            <button
              type="button"
              className="btn-danger-outline btn-sm"
              onClick={() => {
                setDeleteError('');
                setConfirmOpen(true);
              }}
            >
              Delete User Account
            </button>
          </div>
        </div>
      )}

      {confirmOpen && (
        <div
          className="modal-backdrop"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-user-modal-title"
        >
          <div className="modal-content modal-danger">
            <div className="modal-header">
              <h3 id="delete-user-modal-title">Delete User Account</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setConfirmOpen(false)}
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
                  <strong>This action is permanent and cannot be undone.</strong>
                </div>
              </div>

              <p className="mt-3">
                Are you sure you want to delete user <strong>{profile?.username || session.username}</strong>?
              </p>

              <p className="text-muted font-sm mt-2">
                Deleting your user removes all account data, records, and username details.
                Activity logs, including login times and IP information, cannot be removed.
              </p>

              {deleteError && <p className="text-danger font-sm mt-3">{deleteError}</p>}
            </div>

            <div className="modal-actions mt-4">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setConfirmOpen(false)}
                disabled={deleting}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn-danger"
                onClick={handleConfirmDelete}
                disabled={deleting}
              >
                {deleting ? 'Deleting...' : 'Yes, Delete User'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
