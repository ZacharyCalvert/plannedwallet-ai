import {
  Account,
  AccountAsset,
  AccountCategorySummary,
  AccountRecord,
  AccountRecoveryRequest,
  AccountRecoveryResponse,
  AccountSnapshot,
  AddAccountAssetRequest,
  Asset,
  AssetAccountSummary,
  AssetAccountTrendPoint,
  AssetStore,
  AssetStoreRecord,
  AssetStoreTrendPoint,
  AssetValue,
  CaptchaResponse,
  Category,
  CreateAssetRequest,
  CreateAssetStoreRequest,
  CreateAssetValueRequest,
  CreateReconcileReportRequest,
  CreateRecordRequest,
  CreateSnapshotRequest,
  CreateStoreRecordRequest,
  PolicyAgreement,
  PolicyAgreementRequest,
  ReconcileReport,
  RecurringTransaction,
  RecurringTransactionApplyRequest,
  RecurringTransactionApplyResponse,
  RecurringTransactionRequest,
  RotateRecoveryKeyResponse,
  UpdateAccountAssetRequest,
  UpdateAssetStoreRequest,
  UpdateSnapshotRequest,
  UpdateStoreRecordRequest,
  UserProfile,
  UserRegistrationResponse,
} from './types';
import { createTraceparent, logTraceError, logTraceInfo } from './trace';
import {
  MAINTENANCE_EDIT_WARNING,
  isReadOnlyErrorResponse,
  notifyMaintenanceWarning,
} from './maintenance';
import {
  isPolicyApprovalRequiredResponse,
  notifyPolicyRequired,
} from './policyAgreement';

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
      const errBody = await res.clone().json().catch(() => ({}));
      if (isPolicyApprovalRequiredResponse(res.status, errBody)) {
        notifyPolicyRequired();
      }
      const method = (init.method || 'GET').toUpperCase();
      if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
        if (isReadOnlyErrorResponse(res.status, errBody)) {
          notifyMaintenanceWarning(MAINTENANCE_EDIT_WARNING);
        }
      }
    } else {
      logTraceInfo(resTraceparent, `Response success ${res.status} for ${url}`);
    }
    return res;
  } catch (err) {
    logTraceError(traceparent, `Network failure for ${url}:`, err);
    throw err;
  }
}

export async function fetchFeatures(): Promise<string[]> {
  try {
    const res = await tracedFetch('/api/v1/features');
    if (!res.ok) {
      return [];
    }
    const data = await res.json().catch(() => ({}));
    return Array.isArray(data?.features) ? data.features : [];
  } catch {
    return [];
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

export async function refreshToken(
  username: string,
  token: string
): Promise<string> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/token/refresh`;
  const res = await tracedFetch(url, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to refresh access token');
  }
  const data = await res.json();
  return data.access_token;
}

export async function fetchUserProfile(
  token: string,
  username: string
): Promise<UserProfile> {
  const url = `/api/v1/users/${encodeURIComponent(username)}`;
  const res = await tracedFetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to load user details');
  }
  return res.json();
}

export async function deleteUserAccount(
  token: string,
  username: string
): Promise<void> {
  const url = `/api/v1/users/${encodeURIComponent(username)}`;
  const res = await tracedFetch(url, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete user account');
  }
}

export async function submitPolicyAgreement(
  token: string,
  username: string,
  payload: PolicyAgreementRequest
): Promise<PolicyAgreement> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/policy-agreements`;
  const res = await tracedFetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to record policy agreement');
  }
  return res.json();
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

export async function recoverAccount(
  username: string,
  payload: AccountRecoveryRequest
): Promise<AccountRecoveryResponse> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/recover`;
  const res = await tracedFetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Account recovery failed');
  }
  return res.json();
}

export async function rotateRecoveryKey(
  token: string,
  username: string,
  password: string,
  totpCode: string
): Promise<RotateRecoveryKeyResponse> {
  const url = `/api/v1/users/${encodeURIComponent(username)}/recovery-key/rotate`;
  const res = await tracedFetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ password, totp_code: totpCode }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to rotate recovery phrase');
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

export async function deleteAccount(
  token: string,
  accountId: string
): Promise<void> {
  const res = await tracedFetch(`/api/v1/accounts/${encodeURIComponent(accountId)}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete account');
  }
}

export async function patchAccount(
  token: string,
  accountId: string,
  payload: { name?: string; description?: string }
): Promise<Account> {
  const res = await tracedFetch(`/api/v1/accounts/${encodeURIComponent(accountId)}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to update account');
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

export async function createAccountRecordsBulk(
  token: string,
  accountId: string,
  records: CreateRecordRequest[]
): Promise<{ created_count: number; records: AccountRecord[] }> {
  const res = await tracedFetch(`/api/v1/accounts/${encodeURIComponent(accountId)}/records/bulk`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ records }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to bulk create records');
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
  data: { value?: number; date_cleared?: string | null; category_id?: string | null; description?: string }
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

export async function deleteAccountRecord(
  token: string,
  accountId: string,
  recordId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/records/${encodeURIComponent(recordId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete record');
  }
}

export async function fetchReconcileReports(
  token: string,
  accountId: string
): Promise<ReconcileReport[]> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/reconcile-reports`,
    { headers: { Authorization: `Bearer ${token}` } }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to load reconcile reports');
  }
  return res.json();
}

export async function createReconcileReport(
  token: string,
  accountId: string,
  data: CreateReconcileReportRequest
): Promise<ReconcileReport> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/reconcile-reports`,
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
    throw new Error(err.message || 'Failed to create reconcile report');
  }
  return res.json();
}

export async function deleteReconcileReport(
  token: string,
  accountId: string,
  reportId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/reconcile-reports/${encodeURIComponent(reportId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete reconcile report');
  }
}

export async function fetchCategories(token: string): Promise<Category[]> {
  const res = await tracedFetch('/api/v1/categories', {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to load categories');
  }
  return res.json();
}

export async function createCategory(token: string, name: string): Promise<Category> {
  const res = await tracedFetch('/api/v1/categories', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create category');
  }
  return res.json();
}

export async function deleteCategory(token: string, categoryId: string): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/categories/${encodeURIComponent(categoryId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete category');
  }
}

export async function fetchAccountCategorySummary(
  token: string,
  accountId: string,
  params?: { cleared?: boolean; start_date?: string; end_date?: string }
): Promise<AccountCategorySummary> {
  const query = new URLSearchParams();
  if (params?.cleared !== undefined) query.set('cleared', String(params.cleared));
  if (params?.start_date) query.set('start_date', params.start_date);
  if (params?.end_date) query.set('end_date', params.end_date);
  const qs = query.toString() ? `?${query.toString()}` : '';

  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/records/categories${qs}`,
    { headers: { Authorization: `Bearer ${token}` } }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to load category summary');
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
  const data = await res.json();
  return Array.isArray(data) ? data : [];
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

export async function deleteAsset(
  token: string,
  assetId: string
): Promise<void> {
  const res = await tracedFetch(`/api/v1/assets/${encodeURIComponent(assetId)}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete asset definition');
  }
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
  const data = await res.json();
  return Array.isArray(data) ? data : [];
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
  const data = await res.json();
  return Array.isArray(data) ? data : [];
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
  const data = await res.json();
  return Array.isArray(data) ? data : [];
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

export async function fetchAssetStores(
  token: string,
  accountId: string
): Promise<AssetStore[]> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load asset stores');
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function createAssetStore(
  token: string,
  accountId: string,
  data: CreateAssetStoreRequest
): Promise<AssetStore> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores`,
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
    throw new Error(err.message || 'Failed to create asset store');
  }
  return res.json();
}

export async function updateAssetStore(
  token: string,
  accountId: string,
  storeId: string,
  data: UpdateAssetStoreRequest
): Promise<AssetStore> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}`,
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
    throw new Error(err.message || 'Failed to update asset store');
  }
  return res.json();
}

export async function deleteAssetStore(
  token: string,
  accountId: string,
  storeId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete asset store');
  }
}

export async function fetchAssetStoreRecords(
  token: string,
  accountId: string,
  storeId: string
): Promise<AssetStoreRecord[]> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}/records`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load store records');
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function createAssetStoreRecord(
  token: string,
  accountId: string,
  storeId: string,
  data: CreateStoreRecordRequest
): Promise<AssetStoreRecord> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}/records`,
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
    throw new Error(err.message || 'Failed to add store record');
  }
  return res.json();
}

export async function updateAssetStoreRecord(
  token: string,
  accountId: string,
  storeId: string,
  recordId: string,
  data: UpdateStoreRecordRequest
): Promise<AssetStoreRecord> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}/records/${encodeURIComponent(recordId)}`,
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
    throw new Error(err.message || 'Failed to update store record');
  }
  return res.json();
}

export async function deleteAssetStoreRecord(
  token: string,
  accountId: string,
  storeId: string,
  recordId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}/records/${encodeURIComponent(recordId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete store record');
  }
}

export async function fetchAssetStoreTrend(
  token: string,
  accountId: string,
  storeId: string
): Promise<AssetStoreTrendPoint[]> {
  const res = await tracedFetch(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/stores/${encodeURIComponent(storeId)}/trend`,
    {
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    throw new Error('Failed to load store trend');
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

// ---------- Recurring Transactions API ----------

export async function fetchRecurringTransactions(
  token: string
): Promise<RecurringTransaction[]> {
  const res = await tracedFetch('/api/v1/recurring-transactions', {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error('Failed to load recurring transactions');
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function createRecurringTransaction(
  token: string,
  data: RecurringTransactionRequest
): Promise<RecurringTransaction> {
  const res = await tracedFetch('/api/v1/recurring-transactions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to create recurring transaction');
  }
  return res.json();
}

export async function updateRecurringTransaction(
  token: string,
  recurringId: string,
  data: RecurringTransactionRequest
): Promise<RecurringTransaction> {
  const res = await tracedFetch(
    `/api/v1/recurring-transactions/${encodeURIComponent(recurringId)}`,
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
    throw new Error(err.message || 'Failed to update recurring transaction');
  }
  return res.json();
}

export async function deleteRecurringTransaction(
  token: string,
  recurringId: string
): Promise<void> {
  const res = await tracedFetch(
    `/api/v1/recurring-transactions/${encodeURIComponent(recurringId)}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    }
  );
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.message || 'Failed to delete recurring transaction');
  }
}

export async function applyRecurringTransactionToAccount(
  token: string,
  recurringId: string,
  data: RecurringTransactionApplyRequest = {}
): Promise<RecurringTransactionApplyResponse> {
  const res = await tracedFetch(
    `/api/v1/recurring-transactions/${encodeURIComponent(recurringId)}/apply`,
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
    throw new Error(err.message || 'Failed to apply recurring transaction to account');
  }
  return res.json();
}
