import {
  Account,
  AccountAsset,
  AccountRecord,
  AccountSnapshot,
  AddAccountAssetRequest,
  Asset,
  AssetAccountSummary,
  AssetAccountTrendPoint,
  AssetValue,
  CaptchaResponse,
  CreateAssetRequest,
  CreateAssetValueRequest,
  CreateRecordRequest,
  CreateSnapshotRequest,
  UpdateAccountAssetRequest,
  UpdateSnapshotRequest,
  UserRegistrationResponse,
} from './types';
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

export async function fetchCaptcha(): Promise<CaptchaResponse> {
  const res = await tracedFetch('/api/v1/captcha');
  if (!res.ok) {
    throw new Error('Failed to load CAPTCHA challenge');
  }
  return res.json();
}

export async function registerUser(
  username: string,
  email: string,
  password: string,
  captchaId?: string,
  captchaSolution?: string
): Promise<UserRegistrationResponse> {
  const payload: Record<string, string> = { username, email, password };
  if (captchaId) payload.captcha_id = captchaId;
  if (captchaSolution) payload.captcha_solution = captchaSolution;

  const res = await tracedFetch('/api/v1/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
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

export async function createAccount(
  token: string,
  name: string,
  type: string,
  description: string
): Promise<Account> {
  const res = await tracedFetch('/api/v1/accounts', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name, type, description }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create account');
  }
  return res.json();
}

export async function createCheckingAccount(
  token: string,
  name: string,
  description: string
): Promise<Account> {
  return createAccount(token, name, 'checking', description);
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

export async function fetchAccountSnapshots(
  token: string,
  accountId: string,
  startDate?: string,
  endDate?: string
): Promise<AccountSnapshot[]> {
  const query = new URLSearchParams();
  if (startDate) query.set('start_date', startDate);
  if (endDate) query.set('end_date', endDate);
  const qs = query.toString() ? `?${query.toString()}` : '';

  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/snapshots${qs}`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to load snapshots');
  }
  return res.json();
}

export async function createAccountSnapshot(
  token: string,
  accountId: string,
  data: CreateSnapshotRequest
): Promise<AccountSnapshot> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/snapshots`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to record snapshot');
  }
  return res.json();
}

export async function updateAccountSnapshot(
  token: string,
  accountId: string,
  snapshotId: string,
  data: UpdateSnapshotRequest
): Promise<AccountSnapshot> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/snapshots/${encodeURIComponent(snapshotId)}`,
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
    throw new Error(err.message || 'Failed to update snapshot');
  }
  return res.json();
}

export async function deleteAccountSnapshot(
  token: string,
  accountId: string,
  snapshotId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/snapshots/${encodeURIComponent(snapshotId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete snapshot');
  }
}

export async function fetchAssets(token: string, label?: string): Promise<Asset[]> {
  const query = label ? `?label=${encodeURIComponent(label)}` : '';
  const res = await tracedFetch(`/api/v1/assets${query}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error('Failed to load assets');
  }
  return res.json();
}

export async function createAsset(token: string, data: CreateAssetRequest): Promise<Asset> {
  const res = await tracedFetch('/api/v1/assets', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create asset');
  }
  return res.json();
}

export async function updateAsset(
  token: string,
  assetId: string,
  data: CreateAssetRequest
): Promise<Asset> {
  const res = await tracedFetch(`/api/v1/assets/${encodeURIComponent(assetId)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to update asset definition');
  }
  return res.json();
}

export async function fetchAssetValues(
  token: string,
  assetId: string,
  startDate?: string,
  endDate?: string
): Promise<AssetValue[]> {
  const query = new URLSearchParams();
  if (startDate) query.set('start_date', startDate);
  if (endDate) query.set('end_date', endDate);
  const qs = query.toString() ? `?${query.toString()}` : '';

  const res = await tracedFetch(
    `/api/v1/assets/${encodeURIComponent(assetId)}/values${qs}`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load asset valuation history');
  }
  return res.json();
}

export async function createAssetValue(
  token: string,
  assetId: string,
  data: CreateAssetValueRequest
): Promise<AssetValue> {
  const res = await tracedFetch(
    `/api/v1/assets/${encodeURIComponent(assetId)}/values`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to record asset valuation');
  }
  return res.json();
}

export async function updateAssetValue(
  token: string,
  assetId: string,
  valueId: string,
  data: CreateAssetValueRequest
): Promise<AssetValue> {
  const res = await tracedFetch(
    `/api/v1/assets/${encodeURIComponent(assetId)}/values/${encodeURIComponent(valueId)}`,
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
    throw new Error(err.message || 'Failed to update asset valuation');
  }
  return res.json();
}

export async function deleteAssetValue(
  token: string,
  assetId: string,
  valueId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/assets/${encodeURIComponent(assetId)}/values/${encodeURIComponent(valueId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete asset valuation');
  }
}

export async function fetchAccountAssets(
  token: string,
  accountId: string
): Promise<AccountAsset[]> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load account assets');
  }
  return res.json();
}

export async function fetchAssetAccountSummary(
  token: string,
  accountId: string
): Promise<AssetAccountSummary> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets/summary`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load asset account summary');
  }
  return res.json();
}

export async function fetchAssetAccountTrend(
  token: string,
  accountId: string,
  startDate?: string,
  endDate?: string
): Promise<AssetAccountTrendPoint[]> {
  const params = new URLSearchParams();
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const q = params.toString() ? `?${params.toString()}` : '';
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets/trend${q}`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load asset account trend');
  }
  return res.json();
}

export async function addAccountAsset(
  token: string,
  accountId: string,
  data: AddAccountAssetRequest
): Promise<AccountAsset> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to add asset to account');
  }
  return res.json();
}

export async function updateAccountAsset(
  token: string,
  accountId: string,
  assetId: string,
  data: UpdateAccountAssetRequest
): Promise<AccountAsset> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets/${encodeURIComponent(assetId)}`,
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
    throw new Error(err.message || 'Failed to update asset quantity');
  }
  return res.json();
}

export async function deleteAccountAsset(
  token: string,
  accountId: string,
  assetId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/assets/${encodeURIComponent(assetId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to remove asset from account');
  }
}
