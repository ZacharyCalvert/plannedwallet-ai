'use client';

import { useState, useEffect } from 'react';
import { Account, RotateRecoveryKeyResponse, UserProfile, UserSession } from '../lib/types';
import { deleteUserAccount, fetchUserProfile, rotateRecoveryKey } from '../lib/api';
import { clampPasswordToMaxBytes, MAX_PASSWORD_BYTES } from '../lib/passwordLimit';
import { downloadAllAccountsZip } from '../lib/exportData';
import { RecoveryPhraseSetup } from './RecoveryPhraseSetup';

interface UserDetailsPaneProps {
  session: UserSession;
  accounts: Account[];
  onUserDeleted: () => void;
  onBackToAccounts: () => void;
}

export function UserDetailsPane({
  session,
  accounts,
  onUserDeleted,
  onBackToAccounts,
}: UserDetailsPaneProps) {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState('');
  const [rotateOpen, setRotateOpen] = useState(false);
  const [rotatePassword, setRotatePassword] = useState('');
  const [rotateTotp, setRotateTotp] = useState('');
  const [rotating, setRotating] = useState(false);
  const [rotateError, setRotateError] = useState('');
  const [rotatedResult, setRotatedResult] = useState<RotateRecoveryKeyResponse | null>(null);
  const [downloadingAll, setDownloadingAll] = useState(false);
  const [downloadAllError, setDownloadAllError] = useState('');

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

  const handleDownloadAll = async () => {
    if (downloadingAll || accounts.length === 0) return;
    setDownloadAllError('');
    setDownloadingAll(true);
    try {
      await downloadAllAccountsZip(session.token, accounts);
    } catch (err: unknown) {
      setDownloadAllError(err instanceof Error ? err.message : 'Failed to export all account data.');
    } finally {
      setDownloadingAll(false);
    }
  };

  const handleOpenRotate = () => {
    setRotatePassword('');
    setRotateTotp('');
    setRotateError('');
    setRotatedResult(null);
    setRotateOpen(true);
  };

  const handleConfirmRotate = async (e: React.FormEvent) => {
    e.preventDefault();
    setRotating(true);
    setRotateError('');
    try {
      const res = await rotateRecoveryKey(session.token, session.username, rotatePassword, rotateTotp.trim());
      setRotatedResult(res);
      setProfile((prev) => (prev ? { ...prev, recovery_key_hint: res.recovery_key_hint } : prev));
    } catch (err: unknown) {
      setRotateError(err instanceof Error ? err.message : 'Failed to rotate recovery phrase');
    } finally {
      setRotating(false);
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
            <div className="user-details-item">
              <dt>Active Recovery Phrase</dt>
              <dd>{profile?.recovery_key_hint ? `12-word phrase (${profile.recovery_key_hint})` : 'Active'}</dd>
            </div>
          </dl>

          <div className="user-security-zone">
            <div>
              <h3>Recovery Phrase</h3>
              <p className="text-muted font-sm">
                Generate a fresh 12-word recovery phrase and retire your current recovery phrase.
              </p>
            </div>
            <button
              type="button"
              className="btn-secondary btn-sm"
              onClick={handleOpenRotate}
            >
              Rotate Recovery Phrase
            </button>
          </div>

          <div className="user-security-zone">
            <div>
              <h3>Export Account Data</h3>
              <p className="text-muted font-sm">
                Download a ZIP archive containing one CSV file per account with all entered data.
              </p>
              {downloadAllError && <p className="text-danger font-sm mt-2">{downloadAllError}</p>}
            </div>
            <button
              type="button"
              className="btn-secondary btn-sm"
              onClick={() => void handleDownloadAll()}
              disabled={downloadingAll || accounts.length === 0}
              title={
                accounts.length === 0
                  ? 'No accounts available to export'
                  : 'Download a ZIP archive with one CSV file per account'
              }
            >
              {downloadingAll ? 'Preparing Zip...' : 'Download All Data'}
            </button>
          </div>

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

      {rotateOpen && (
        <div
          className="modal-backdrop"
          role="dialog"
          aria-modal="true"
          aria-labelledby="rotate-recovery-modal-title"
        >
          <div className="modal-content">
            <div className="modal-header">
              <h3 id="rotate-recovery-modal-title">Rotate Recovery Phrase</h3>
              {!rotatedResult && (
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => setRotateOpen(false)}
                  disabled={rotating}
                  aria-label="Close"
                >
                  ×
                </button>
              )}
            </div>

            <div className="modal-body">
              {rotatedResult ? (
                <RecoveryPhraseSetup
                  username={profile?.username || session.username}
                  phrase={rotatedResult.recovery_phrase}
                  title="New 12-Word Recovery Phrase"
                  subtitle="Your previous recovery phrase is now permanently deactivated. Store and verify this new phrase before closing."
                  continueLabel="Done"
                  onConfirmed={() => {
                    setRotateOpen(false);
                    setRotatedResult(null);
                  }}
                />
              ) : (
                <form className="auth-form" onSubmit={handleConfirmRotate}>
                  <p className="text-sm text-muted">
                    Confirm your current password and 6-digit authenticator code to invalidate your old recovery phrase and generate a new one.
                  </p>
                  {rotateError && <div className="alert-error" role="alert">{rotateError}</div>}
                  <div className="form-group">
                    <label htmlFor="rotate-password">Current Password</label>
                    <input
                      id="rotate-password"
                      type="password"
                      required
                      maxLength={MAX_PASSWORD_BYTES}
                      value={rotatePassword}
                      onChange={(e) => setRotatePassword(clampPasswordToMaxBytes(e.target.value))}
                      placeholder="Enter your password"
                    />
                  </div>
                  <div className="form-group">
                    <label htmlFor="rotate-totp">Authenticator Code (TOTP)</label>
                    <input
                      id="rotate-totp"
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      maxLength={6}
                      required
                      value={rotateTotp}
                      onChange={(e) => setRotateTotp(e.target.value)}
                      placeholder="6-digit code"
                    />
                  </div>
                  <div className="form-actions">
                    <button
                      type="button"
                      className="btn-secondary"
                      onClick={() => setRotateOpen(false)}
                      disabled={rotating}
                    >
                      Cancel
                    </button>
                    <button type="submit" className="btn-primary" disabled={rotating}>
                      {rotating ? 'Rotating...' : 'Generate New Phrase'}
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
