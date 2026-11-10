'use client';

import { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { loginUser } from '../lib/api';

interface QRCodeDisplayProps {
  totpUrl: string;
  totpSecret: string;
}

function QRCodeDisplay({ totpUrl, totpSecret }: QRCodeDisplayProps) {
  return (
    <div className="totp-qr-container">
      <p className="totp-instructions">
        Scan this QR code with your authenticator app (e.g. Google Authenticator):
      </p>
      <div className="totp-qr-wrapper">
        <QRCodeSVG value={totpUrl} size={180} />
      </div>
      <div className="totp-secret-wrapper">
        <span className="totp-secret-label">Manual key:</span>
        <code className="totp-secret-key">{totpSecret}</code>
      </div>
    </div>
  );
}

interface VerificationInputProps {
  code: string;
  onChange: (v: string) => void;
  loading: boolean;
  onCancel: () => void;
}

function VerificationInput({ code, onChange, loading, onCancel }: VerificationInputProps) {
  return (
    <>
      <div className="form-group">
        <label htmlFor="verify-totp">Authenticator Code</label>
        <input
          id="verify-totp"
          type="text"
          inputMode="numeric"
          pattern="[0-9]*"
          maxLength={6}
          required
          value={code}
          onChange={(e) => onChange(e.target.value)}
          placeholder="6-digit code"
        />
      </div>
      <div className="form-actions">
        <button type="button" className="btn-secondary" onClick={onCancel} disabled={loading}>
          Back
        </button>
        <button type="submit" className="btn-primary" disabled={loading}>
          {loading ? 'Verifying...' : 'Verify & Continue'}
        </button>
      </div>
    </>
  );
}

interface TOTPSetupProps {
  username: string;
  password: string;
  totpUrl: string;
  totpSecret: string;
  onSuccess: (username: string, token: string) => void;
  onCancel: () => void;
}

export function TOTPSetup(props: TOTPSetupProps) {
  const [totpCode, setTotpCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const token = await loginUser(props.username, props.password, totpCode.trim());
      props.onSuccess(props.username, token);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={handleVerify}>
      <h3 className="totp-title">Two-Factor Authentication Setup</h3>
      {error && <div className="alert-error">{error}</div>}
      <QRCodeDisplay totpUrl={props.totpUrl} totpSecret={props.totpSecret} />
      <VerificationInput
        code={totpCode}
        onChange={setTotpCode}
        loading={loading}
        onCancel={props.onCancel}
      />
    </form>
  );
}
