'use client';

import { useState } from 'react';
import { loginUser } from '../lib/api';

interface SignInFormProps {
  onSuccess: (username: string, token: string) => void;
}

interface InputFieldProps {
  id: string;
  label: string;
  type: string;
  value: string;
  placeholder: string;
  maxLength?: number;
  inputMode?: 'text' | 'numeric';
  pattern?: string;
  onChange: (v: string) => void;
}

function InputField(p: InputFieldProps) {
  return (
    <div className="form-group">
      <label htmlFor={p.id}>{p.label}</label>
      <input
        id={p.id}
        type={p.type}
        required
        value={p.value}
        onChange={(e) => p.onChange(e.target.value)}
        placeholder={p.placeholder}
        maxLength={p.maxLength}
        inputMode={p.inputMode}
        pattern={p.pattern}
      />
    </div>
  );
}

interface SignInFieldsProps {
  username: string;
  password: string;
  totpCode: string;
  onChangeUsername: (val: string) => void;
  onChangePassword: (val: string) => void;
  onChangeTotpCode: (val: string) => void;
}

function SignInFields(props: SignInFieldsProps) {
  return (
    <>
      <InputField
        id="signin-username"
        label="Username"
        type="text"
        value={props.username}
        placeholder="Enter your username"
        onChange={props.onChangeUsername}
      />
      <InputField
        id="signin-password"
        label="Password"
        type="password"
        value={props.password}
        placeholder="Enter your password"
        onChange={props.onChangePassword}
      />
      <InputField
        id="signin-totp"
        label="Authenticator Code (TOTP)"
        type="text"
        inputMode="numeric"
        pattern="[0-9]*"
        maxLength={6}
        value={props.totpCode}
        placeholder="6-digit code"
        onChange={props.onChangeTotpCode}
      />
    </>
  );
}

export function SignInForm({ onSuccess }: SignInFormProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const token = await loginUser(username.trim(), password, totpCode.trim());
      onSuccess(username.trim(), token);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Sign in failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      {error && <div className="alert-error">{error}</div>}
      <SignInFields
        username={username}
        password={password}
        totpCode={totpCode}
        onChangeUsername={setUsername}
        onChangePassword={setPassword}
        onChangeTotpCode={setTotpCode}
      />
      <button type="submit" className="btn-primary btn-block" disabled={loading}>
        {loading ? 'Signing In...' : 'Sign In'}
      </button>
    </form>
  );
}
