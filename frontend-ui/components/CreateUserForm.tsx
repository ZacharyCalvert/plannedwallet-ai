'use client';

import { useState } from 'react';
import { registerUser } from '../lib/api';
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

interface RegisterFormViewProps {
  error: string;
  loading: boolean;
  fields: UserFieldsProps;
  onSubmit: (e: React.FormEvent) => void;
}

function RegisterFormView({ error, loading, fields, onSubmit }: RegisterFormViewProps) {
  return (
    <form className="auth-form" onSubmit={onSubmit}>
      {error && <div className="alert-error">{error}</div>}
      <UserFields {...fields} />
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
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [regData, setRegData] = useState<UserRegistrationResponse | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      setRegData(await registerUser(username.trim(), email.trim(), password));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  return { username, setUsername, email, setEmail, password, setPassword, error, loading, regData, setRegData, submit };
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
  return <RegisterFormView error={r.error} loading={r.loading} fields={fields} onSubmit={r.submit} />;
}
