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

export interface CaptchaResponse {
  id: string;
  image: string;
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

export interface Asset {
  id: string;
  user_id?: string | null;
  label: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface AssetValue {
  id: string;
  asset_id: string;
  value: number;
  value_date: string;
  created_at: string;
  updated_at: string;
}

export interface AccountAsset {
  id: string;
  account_id: string;
  asset_id: string;
  label: string;
  description: string;
  quantity: number;
  asset_date: string;
  latest_value: number;
  latest_value_date?: string | null;
  total_value: number;
  created_at: string;
  updated_at: string;
}

export interface AssetAccountSummary {
  account_id: string;
  total_value: number;
  assets: AccountAsset[];
  stores?: AssetStore[];
}

export interface AssetStore {
  id: string;
  account_id: string;
  asset_id: string;
  label: string;
  description: string;
  asset_label: string;
  asset_description?: string;
  latest_quantity: number;
  latest_record_date?: string | null;
  latest_value: number;
  latest_value_date?: string | null;
  total_value: number;
  created_at: string;
  updated_at: string;
}

export interface AssetStoreRecord {
  id: string;
  asset_store_id: string;
  quantity: number;
  record_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAssetStoreRequest {
  asset_id: string;
  label: string;
  description?: string;
  quantity?: number;
  record_date?: string;
}

export interface UpdateAssetStoreRequest {
  label: string;
  description?: string;
  asset_id?: string;
}

export interface CreateStoreRecordRequest {
  quantity: number;
  record_date: string;
}

export interface UpdateStoreRecordRequest {
  quantity?: number;
  record_date?: string;
}

export interface AssetStoreTrendPoint {
  date: string;
  quantity: number;
  unit_value: number;
  total_value: number;
}

export interface AssetAccountTrendPoint {
  date: string;
  total_value: number;
}

export interface CreateAssetRequest {
  label: string;
  description?: string;
}

export interface CreateAssetValueRequest {
  value: number;
  value_date: string;
}

export interface AddAccountAssetRequest {
  asset_id: string;
  quantity: number;
  asset_date?: string;
}

export interface UpdateAccountAssetRequest {
  quantity?: number;
  asset_date?: string;
}

export interface PerspectiveSettings {
  id: string;
  perspective_id: string;
  show_financial_trend: boolean;
  calculate_average_daily_spend: boolean;
  calculate_average_return_yoy: boolean;
  track_ruined_date: boolean;
  custom_daily_spend?: number | null;
  created_at: string;
  updated_at: string;
}

export interface PerspectiveSettingsInput {
  show_financial_trend?: boolean;
  calculate_average_daily_spend?: boolean;
  calculate_average_return_yoy?: boolean;
  track_ruined_date?: boolean;
  custom_daily_spend?: number | null;
}

export interface Perspective {
  id: string;
  user_id: string;
  name: string;
  description: string;
  access_level: 'read' | 'edit' | 'admin';
  settings?: PerspectiveSettings;
  account_ids?: string[];
  created_at: string;
  updated_at: string;
}

export interface CreatePerspectiveRequest {
  name: string;
  description?: string;
  account_ids?: string[];
  settings?: PerspectiveSettingsInput;
}

export interface PerspectiveTrendPoint {
  date: string;
  total_value: number;
  account_values?: Record<string, number>;
}

export interface PerspectiveViewResponse {
  perspective: Perspective;
  settings: PerspectiveSettings;
  accounts: Account[];
  current_total_value: number;
  trend?: PerspectiveTrendPoint[];
  average_daily_spend?: number | null;
  average_return_yoy?: number | null;
  ruined_date?: string | null;
  days_until_ruined?: number | null;
  daily_spend_30d?: number | null;
  days_until_ruined_30d?: number | null;
  ruined_date_30d?: string | null;
  daily_spend_90d?: number | null;
  days_until_ruined_90d?: number | null;
  ruined_date_90d?: string | null;
  daily_spend_365d?: number | null;
  days_until_ruined_365d?: number | null;
  ruined_date_365d?: string | null;
}

export interface PerspectiveAccess {
  id: string;
  perspective_id: string;
  user_id: string;
  access_level: 'read' | 'edit' | 'admin';
  created_at: string;
  updated_at: string;
}

