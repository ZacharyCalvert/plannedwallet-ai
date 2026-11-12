export interface UserSession {
  username: string;
  token: string;
}

export interface UserRegistrationResponse {
  id: string;
  username: string;
  email: string;
  totp_secret: string;
  totp_url: string;
  created_at: string;
}

export interface Account {
  id: string;
  admin_id: string;
  name: string;
  type: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface AccountRecord {
  id: string;
  account_id: string;
  value: number;
  date_sent: string;
  date_cleared: string | null;
  entity: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRecordRequest {
  value: number;
  date_sent: string;
  date_cleared?: string | null;
  entity: string;
  description?: string;
}

export interface AccountSnapshot {
  id: string;
  account_id: string;
  value: number;
  snapshot_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSnapshotRequest {
  value: number;
  snapshot_date: string;
}

export interface UpdateSnapshotRequest {
  value: number;
  snapshot_date?: string;
}
