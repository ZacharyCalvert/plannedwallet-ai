import { Account, UserRegistrationResponse } from './types';

export async function loginUser(
  username: string,
  password: string,
  totpCode: string
): Promise<string> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/token`;
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password, totp_code: totpCode }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Invalid username, password, or TOTP code');
  }
  const data = await res.json();
  return data.access_token;
}

export async function registerUser(
  username: string,
  email: string,
  password: string
): Promise<UserRegistrationResponse> {
  const res = await fetch('/api/v1/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, email, password }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Registration failed');
  }
  return res.json();
}

export async function fetchAccounts(token: string): Promise<Account[]> {
  const res = await fetch('/api/v1/accounts', {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error('Failed to load accounts');
  }
  return res.json();
}

export async function createCheckingAccount(
  token: string,
  name: string,
  description: string
): Promise<Account> {
  const res = await fetch('/api/v1/accounts', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name, type: 'checking', description }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create account');
  }
  return res.json();
}
