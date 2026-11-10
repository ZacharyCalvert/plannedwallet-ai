'use client';

import { useState } from 'react';
import { loginUser } from '../lib/api';

interface SignInFormProps {
  onSuccess: (username: string, token: string) => void;
}

export function SignInForm({ onSuccess }: SignInFormProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const token = await loginUser(username.trim(), password);
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
      <div className="form-group">
        <label htmlFor="signin-username">Username</label>
        <input
          id="signin-username"
          type="text"
          required
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Enter your username"
        />
      </div>
      <div className="form-group">
        <label htmlFor="signin-password">Password</label>
        <input
          id="signin-password"
          type="password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Enter your password"
        />
      </div>
      <button type="submit" className="btn-primary btn-block" disabled={loading}>
        {loading ? 'Signing In...' : 'Sign In'}
      </button>
    </form>
  );
}
