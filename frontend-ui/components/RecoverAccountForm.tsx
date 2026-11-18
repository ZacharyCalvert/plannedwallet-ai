'use client';

import { useCallback, useEffect, useState } from 'react';
import { fetchCaptcha, recoverAccount } from '../lib/api';
import { clampPasswordToMaxBytes, MAX_PASSWORD_BYTES, utf8ByteLength } from '../lib/passwordLimit';
import { AccountRecoveryResponse } from '../lib/types';
import { RecoveryPhraseSetup } from './RecoveryPhraseSetup';
import { TOTPSetup } from './TOTPSetup';

interface RecoverAccountFormProps {
  onSuccess: (username: string, token: string) => void;
  onCancel: () => void;
}

export function RecoverAccountForm({ onSuccess, onCancel }: RecoverAccountFormProps) {
  const [username, setUsername] = useState('');
  const [recoveryPhrase, setRecoveryPhrase] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaImage, setCaptchaImage] = useState('');
  const [captchaSolution, setCaptchaSolution] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [recoveryResult, setRecoveryResult] = useState<AccountRecoveryResponse | null>(null);
  const [phraseConfirmed, setPhraseConfirmed] = useState(false);

  const loadCaptcha = useCallback(async () => {
    try {
      const c = await fetchCaptcha();
      setCaptchaId(c.id);
      setCaptchaImage(c.image);
      setCaptchaSolution('');
    } catch {
      // Ignore initial load error
    }
  }, []);

  useEffect(() => {
    loadCaptcha();
  }, [loadCaptcha]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setError('New password and confirmation do not match.');
      return;
    }

    setError('');
    setLoading(true);
    try {
      const res = await recoverAccount(username.trim(), {
        recovery_phrase: recoveryPhrase,
        new_password: newPassword,
        captcha_id: captchaId,
        captcha_solution: captchaSolution.trim(),
      });
      setPhraseConfirmed(false);
      setRecoveryResult(res);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Account recovery failed');
      loadCaptcha();
    } finally {
      setLoading(false);
    }
  };

  if (recoveryResult && !phraseConfirmed) {
    return (
      <RecoveryPhraseSetup
        username={recoveryResult.username}
        phrase={recoveryResult.recovery_phrase}
        title="Save Your New Replacement Recovery Phrase"
        subtitle="Your previous recovery phrase and authenticator device are permanently invalidated. Save this new 12-word phrase before re-enrolling your authenticator."
        continueLabel="Continue to New Authenticator Setup"
        onConfirmed={() => setPhraseConfirmed(true)}
      />
    );
  }

  if (recoveryResult && phraseConfirmed) {
    return (
      <TOTPSetup
        username={recoveryResult.username}
        password={newPassword}
        totpUrl={recoveryResult.totp_url}
        totpSecret={recoveryResult.totp_secret}
        onSuccess={onSuccess}
        onCancel={() => setPhraseConfirmed(false)}
      />
    );
  }

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      <h3 className="totp-title">Recover Account Credentials</h3>
      <p className="text-sm text-muted">
        Enter your username, your 12-word recovery phrase, and a new password to reset both your password and two-factor authenticator.
      </p>

      {error && <div className="alert-error" role="alert">{error}</div>}

      <div className="form-group">
        <label htmlFor="recover-username">Username</label>
        <input
          id="recover-username"
          type="text"
          required
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Enter your username"
        />
      </div>

      <div className="form-group">
        <label htmlFor="recover-phrase">12-Word Secret Recovery Phrase</label>
        <textarea
          id="recover-phrase"
          rows={3}
          required
          value={recoveryPhrase}
          onChange={(e) => setRecoveryPhrase(e.target.value)}
          placeholder="Enter all 12 words separated by spaces"
        />
      </div>

      <div className="form-group">
        <label htmlFor="recover-password">New Password</label>
        <input
          id="recover-password"
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
        <label htmlFor="recover-password-confirm">Confirm New Password</label>
        <input
          id="recover-password-confirm"
          type="password"
          required
          maxLength={MAX_PASSWORD_BYTES}
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(clampPasswordToMaxBytes(e.target.value))}
          placeholder="Re-enter your new password"
        />
      </div>

      <div className="form-group captcha-group">
        <label htmlFor="recover-captcha">Security Verification (CAPTCHA)</label>
        <div className="captcha-display-row">
          {captchaImage ? (
            <img src={captchaImage} alt="CAPTCHA challenge code" className="captcha-img" />
          ) : (
            <div className="captcha-placeholder">Loading challenge...</div>
          )}
          <button
            type="button"
            className="btn-secondary btn-refresh-captcha"
            onClick={loadCaptcha}
            disabled={loading}
          >
            ↻ Refresh
          </button>
        </div>
        <input
          id="recover-captcha"
          type="text"
          required
          autoComplete="off"
          value={captchaSolution}
          onChange={(e) => setCaptchaSolution(e.target.value)}
          placeholder="Enter the 6 characters shown above"
          className="captcha-input"
        />
      </div>

      <div className="form-actions">
        <button type="button" className="btn-secondary" onClick={onCancel} disabled={loading}>
          Back to Sign In
        </button>
        <button type="submit" className="btn-primary" disabled={loading}>
          {loading ? 'Recovering Account...' : 'Reset Password & MFA'}
        </button>
      </div>
    </form>
  );
}
