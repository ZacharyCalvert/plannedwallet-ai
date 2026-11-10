'use client';

import { useState } from 'react';
import { registerUser } from '../lib/api';

interface CreateUserFormProps {
  onSuccess: (username: string, token: string) => void;
}

export function CreateUserForm({ onSuccess }: CreateUserFormProps) {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const token = await registerUser(username.trim(), email.trim(), password);
      onSuccess(username.trim(), token);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      {error && <div className="alert-error">{error}</div>}
      <div className="form-group">
        <label htmlFor="signup-username">Username</label>
        <input
          id="signup-username"
          type="text"
          required
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Choose a username"
        />
      </div>
      <div className="form-group">
        <label htmlFor="signup-email">Email Address</label>
        <input
          id="signup-email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="name@example.com"
        />
      </div>
      <div className="form-group">
        <label htmlFor="signup-password">Password</label>
        <input
          id="signup-password"
          type="password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Min 10 chars, uppercase, lowercase, number, symbol"
        />
        <p className="form-hint">At least 10 chars with 1 uppercase, 1 lowercase, 1 digit & 1 symbol.</p>
      </div>
      <button type="submit" className="btn-primary btn-block" disabled={loading}>
        {loading ? 'Creating Account...' : 'Create Account'}
      </button>
    </form>
  );
}
