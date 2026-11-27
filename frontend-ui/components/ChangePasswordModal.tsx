'use client';

import { useCallback, useEffect, useState } from 'react';
import { changePassword, fetchCaptcha } from '../lib/api';
import { clampPasswordToMaxBytes, MAX_PASSWORD_BYTES, utf8ByteLength } from '../lib/passwordLimit';

interface ChangePasswordModalProps {
  token: string;
  username: string;
  onClose: () => void;
  onSuccess: (message: string) => void;
}

export function ChangePasswordModal({
  token,
  username,
  onClose,
  onSuccess,
}: ChangePasswordModalProps) {
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaImage, setCaptchaImage] = useState('');
  const [captchaSolution, setCaptchaSolution] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const loadCaptcha = useCallback(async () => {
    try {
      const c = await fetchCaptcha();
      setCaptchaId(c.id);
      setCaptchaImage(c.image);
      setCaptchaSolution('');
    } catch {
      // Ignore initial CAPTCHA load error when server does not enforce it
    }
  }, []);

  useEffect(() => {
    void loadCaptcha();
  }, [loadCaptcha]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setError('New password and confirmation do not match.');
      return;
    }

    setError('');
    setSubmitting(true);
    try {
      const trimmedCaptcha = captchaSolution.trim();
      const res = await changePassword(token, username, {
        old_password: oldPassword,
        new_password: newPassword,
        new_password_confirm: confirmPassword,
        totp_code: totpCode.trim(),
        ...(trimmedCaptcha ? { captcha_id: captchaId, captcha_solution: trimmedCaptcha } : {}),
      });
      onSuccess(res.message || 'Password updated successfully.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to change password');
      void loadCaptcha();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="change-password-modal-title"
    >
      <div className="modal-content">
        <div className="modal-header">
          <h3 id="change-password-modal-title">Change Password</h3>
          <button
            type="button"
            className="btn-close"
            onClick={onClose}
            disabled={submitting}
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="modal-body">
          <form className="auth-form" onSubmit={handleSubmit}>
            <p className="text-sm text-muted">
              Enter your current password, choose a new password, and verify with your 6-digit MFA authenticator code and CAPTCHA.
            </p>
            {error && <div className="alert-error" role="alert">{error}</div>}

            <div className="form-group">
              <label htmlFor="change-old-password">Current Password</label>
              <input
                id="change-old-password"
                type="password"
                required
                maxLength={MAX_PASSWORD_BYTES}
                value={oldPassword}
                onChange={(e) => setOldPassword(clampPasswordToMaxBytes(e.target.value))}
                placeholder="Enter your current password"
              />
            </div>

            <div className="form-group">
              <label htmlFor="change-new-password">New Password</label>
              <input
                id="change-new-password"
                type="password"
                required
                maxLength={MAX_PASSWORD_BYTES}
                value={newPassword}
                onChange={(e) => setNewPassword(clampPasswordToMaxBytes(e.target.value))}
                placeholder="10-72 bytes: upper, lower, digit, symbol"
              />
              <p className="form-hint">
                10–{MAX_PASSWORD_BYTES} bytes ({utf8ByteLength(newPassword)}/{MAX_PASSWORD_BYTES}) with uppercase, lowercase, digit &amp; symbol.
              </p>
            </div>

            <div className="form-group">
              <label htmlFor="change-confirm-password">Confirm New Password</label>
              <input
                id="change-confirm-password"
                type="password"
                required
                maxLength={MAX_PASSWORD_BYTES}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(clampPasswordToMaxBytes(e.target.value))}
                placeholder="Re-enter your new password"
              />
            </div>

            <div className="form-group">
              <label htmlFor="change-totp">MFA Authenticator Code (TOTP)</label>
              <input
                id="change-totp"
                type="text"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                required
                value={totpCode}
                onChange={(e) => setTotpCode(e.target.value)}
                placeholder="6-digit code"
              />
            </div>

            <div className="form-group captcha-group">
              <label htmlFor="change-captcha">Security Verification (CAPTCHA)</label>
              <div className="captcha-display-row">
                {captchaImage ? (
                  <img src={captchaImage} alt="CAPTCHA challenge code" className="captcha-img" />
                ) : (
                  <div className="captcha-placeholder">Loading challenge...</div>
                )}
                <button
                  type="button"
                  className="btn-secondary btn-refresh-captcha"
                  onClick={() => void loadCaptcha()}
                  disabled={submitting}
                >
                  ↻ Refresh
                </button>
              </div>
              <input
                id="change-captcha"
                type="text"
                autoComplete="off"
                value={captchaSolution}
                onChange={(e) => setCaptchaSolution(e.target.value)}
                placeholder="Enter the 6 characters shown above"
                className="captcha-input"
              />
            </div>

            <div className="form-actions">
              <button
                type="button"
                className="btn-secondary"
                onClick={onClose}
                disabled={submitting}
              >
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={submitting}>
                {submitting ? 'Updating...' : 'Update Password'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
