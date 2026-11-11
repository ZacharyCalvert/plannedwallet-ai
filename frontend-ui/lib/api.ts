import { Account, AccountRecord, CreateRecordRequest, UserRegistrationResponse } from './types';
import { createTraceparent, logTraceError, logTraceInfo } from './trace';

async function tracedFetch(url: string, init: RequestInit = {}): Promise<Response> {
  const traceparent = createTraceparent();
  const headers = new Headers(init.headers || {});
  if (!headers.has('traceparent')) {
    headers.set('traceparent', traceparent);
  }
  if (!headers.has('X-Client-ID')) {
    headers.set('X-Client-ID', 'nextjs-static');
  }
  logTraceInfo(traceparent, `Request: ${init.method || 'GET'} ${url}`);

  try {
    const res = await fetch(url, { ...init, headers });
    const resTraceparent = res.headers.get('traceparent') || traceparent;
    if (!res.ok) {
      logTraceError(resTraceparent, `Response error ${res.status} for ${url}`);
    } else {
      logTraceInfo(resTraceparent, `Response success ${res.status} for ${url}`);
    }
    return res;
  } catch (err) {
    logTraceError(traceparent, `Network failure for ${url}:`, err);
    throw err;
  }
}

export async function loginUser(
  username: string,
  password: string,
  totpCode: string
): Promise<string> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/token`;
  const res = await tracedFetch(url, {
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
  const res = await tracedFetch('/api/v1/users', {
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
  const res = await tracedFetch('/api/v1/accounts', {
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
  const res = await tracedFetch('/api/v1/accounts', {
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

export async function fetchAccountRecords(
  token: string,
  accountId: string
): Promise<AccountRecord[]> {
  const res = await tracedFetch(`/api/v1/accounts/${encodeURIComponent(accountId)}/records`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error('Failed to load account records');
  }
  return res.json();
}

export async function createAccountRecord(
  token: string,
  accountId: string,
  data: CreateRecordRequest
): Promise<AccountRecord> {
  const res = await tracedFetch(`/api/v1/accounts/${encodeURIComponent(accountId)}/records`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create record');
  }
  return res.json();
}

export async function updateAccountRecord(
  token: string,
  accountId: string,
  recordId: string,
  data: CreateRecordRequest
): Promise<AccountRecord> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/records/${encodeURIComponent(recordId)}`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to update record');
  }
  return res.json();
}

export async function patchAccountRecord(
  token: string,
  accountId: string,
  recordId: string,
  data: { date_cleared?: string | null }
): Promise<AccountRecord> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/records/${encodeURIComponent(recordId)}`,
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to update record');
  }
  return res.json();
}
