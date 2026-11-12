'use client';

import { useCallback, useEffect, useState } from 'react';
import { fetchCaptcha, registerUser } from '../lib/api';
import { UserRegistrationResponse } from '../lib/types';
import { TOTPSetup } from './TOTPSetup';

interface CreateUserFormProps {
  onSuccess: (username: string, token: string) => void;
}

interface UserFieldsProps {
  username: string;
  email: string;
  password: string;
  onChangeUsername: (v: string) => void;
  onChangeEmail: (v: string) => void;
  onChangePassword: (v: string) => void;
}

function UserFields(p: UserFieldsProps) {
  return (
    <>
      <div className="form-group">
        <label htmlFor="signup-username">Username</label>
        <input
          id="signup-username"
          type="text"
          required
          value={p.username}
          onChange={(e) => p.onChangeUsername(e.target.value)}
          placeholder="Choose a username"
        />
      </div>
      <div className="form-group">
        <label htmlFor="signup-email">Email Address</label>
        <input
          id="signup-email"
          type="email"
          required
          value={p.email}
          onChange={(e) => p.onChangeEmail(e.target.value)}
          placeholder="name@example.com"
        />
      </div>
      <div className="form-group">
        <label htmlFor="signup-password">Password</label>
        <input
          id="signup-password"
          type="password"
          required
          value={p.password}
          onChange={(e) => p.onChangePassword(e.target.value)}
          placeholder="Min 10 chars, uppercase, lowercase, digit, symbol"
        />
        <p className="form-hint">At least 10 chars with uppercase, lowercase, digit & symbol.</p>
      </div>
    </>
  );
}

interface CaptchaFieldProps {
  captchaImage: string;
  solution: string;
  onChangeSolution: (v: string) => void;
  onRefreshCaptcha: () => void;
  loading: boolean;
}

function CaptchaField(p: CaptchaFieldProps) {
  return (
    <div className="form-group captcha-group">
      <label htmlFor="signup-captcha">Security Verification (CAPTCHA)</label>
      <div className="captcha-display-row">
        {p.captchaImage ? (
          <img
            src={p.captchaImage}
            alt="CAPTCHA challenge code"
            className="captcha-img"
          />
        ) : (
          <div className="captcha-placeholder">Loading challenge...</div>
        )}
        <button
          type="button"
          className="btn-secondary btn-refresh-captcha"
          onClick={p.onRefreshCaptcha}
          disabled={p.loading}
          title="Reload CAPTCHA challenge"
        >
          ↻ Refresh
        </button>
      </div>
      <input
        id="signup-captcha"
        type="text"
        required
        autoComplete="off"
        value={p.solution}
        onChange={(e) => p.onChangeSolution(e.target.value)}
        placeholder="Enter the 6 characters shown above"
        className="captcha-input"
      />
      <p className="form-hint">Enter the characters displayed in the image to verify you are human.</p>
    </div>
  );
}

interface RegisterFormViewProps {
  error: string;
  loading: boolean;
  fields: UserFieldsProps;
  captcha: CaptchaFieldProps;
  onSubmit: (e: React.FormEvent) => void;
}

function RegisterFormView({ error, loading, fields, captcha, onSubmit }: RegisterFormViewProps) {
  return (
    <form className="auth-form" onSubmit={onSubmit}>
      {error && <div className="alert-error">{error}</div>}
      <UserFields {...fields} />
      <CaptchaField {...captcha} />
      <button type="submit" className="btn-primary btn-block" disabled={loading}>
        {loading ? 'Creating Account...' : 'Continue to Two-Factor Setup'}
      </button>
    </form>
  );
}

function useRegistration() {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaImage, setCaptchaImage] = useState('');
  const [captchaSolution, setCaptchaSolution] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [regData, setRegData] = useState<UserRegistrationResponse | null>(null);

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

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await registerUser(
        username.trim(),
        email.trim(),
        password,
        captchaId,
        captchaSolution.trim()
      );
      setRegData(res);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed');
      loadCaptcha();
    } finally {
      setLoading(false);
    }
  };

  return {
    username, setUsername, email, setEmail, password, setPassword,
    captchaId, captchaImage, captchaSolution, setCaptchaSolution, loadCaptcha,
    error, loading, regData, setRegData, submit,
  };
}

export function CreateUserForm({ onSuccess }: CreateUserFormProps) {
  const r = useRegistration();

  if (r.regData) {
    return (
      <TOTPSetup
        username={r.regData.username}
        password={r.password}
        totpUrl={r.regData.totp_url}
        totpSecret={r.regData.totp_secret}
        onSuccess={onSuccess}
        onCancel={() => r.setRegData(null)}
      />
    );
  }

  const fields = {
    username: r.username, email: r.email, password: r.password,
    onChangeUsername: r.setUsername, onChangeEmail: r.setEmail, onChangePassword: r.setPassword,
  };
  const captcha = {
    captchaImage: r.captchaImage,
    solution: r.captchaSolution,
    onChangeSolution: r.setCaptchaSolution,
    onRefreshCaptcha: r.loadCaptcha,
    loading: r.loading,
  };
  return <RegisterFormView error={r.error} loading={r.loading} fields={fields} captcha={captcha} onSubmit={r.submit} />;
}
