export interface UserSession {
  username: string;
  token: string;
}

export interface UserProfile {
  id: string;
  username: string;
  email: string;
  created_at: string;
}

export interface PolicyAgreementRequest {
  terms_approved: boolean;
  privacy_approved: boolean;
  us_customer: boolean;
  terms_sha256: string;
}

export interface PolicyAgreement {
  id: string;
  user_id: string;
  terms_sha256: string;
  terms_approved: boolean;
  privacy_approved: boolean;
  us_customer: boolean;
  active: boolean;
  client_ip: string;
  agreed_at: string;
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

export interface Category {
  id: string;
  user_id?: string | null;
  name: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CategoryExpenseSlice {
  category_id?: string | null;
  category_name: string;
  total_amount: number;
  record_count: number;
}

export interface AccountCategorySummary {
  account_id: string;
  total_expenses: number;
  categories: CategoryExpenseSlice[];
}

export interface AccountRecord {
  id: string;
  account_id: string;
  category_id?: string | null;
  reconcile_report_id?: string | null;
  value: number;
  date_sent: string;
  date_cleared: string | null;
  entity: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface ReconcileReport {
  id: string;
  account_id: string;
  name: string;
  report_date: string;
  starting_balance: number;
  ending_balance: number;
  record_count: number;
  record_ids: string[];
  records: AccountRecord[];
  created_at: string;
  updated_at: string;
}

export interface CreateReconcileReportRequest {
  name?: string;
  report_date: string;
  starting_balance: number;
  ending_balance: number;
  record_ids: string[];
}

export interface CreateRecordRequest {
  value: number;
  date_sent: string;
  date_cleared?: string | null;
  category_id?: string | null;
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

export interface FeaturesResponse {
  features: string[];
}

export type RecurringFrequency = 'weekly' | 'bi-monthly' | 'monthly' | 'yearly';
export type RecurringScheduleType =
  | 'weekday'
  | 'day_of_period'
  | 'nth_weekday'
  | 'month_day'
  | 'first_fifteenth'
  | 'fifteenth_last';
export type RecurringScheduleScope = 'week' | 'bi-monthly' | 'month' | 'year';
export type RecurringDueStatus = 'overdue' | 'upcoming' | 'scheduled' | 'unscheduled';

export type BillFrequency = RecurringFrequency;
export type BillScheduleType = RecurringScheduleType;
export type BillScheduleScope = RecurringScheduleScope;
export type BillDueStatus = RecurringDueStatus;

export interface RecurringSchedule {
  type: RecurringScheduleType;
  scope: RecurringScheduleScope;
  nth?: number | null;
  weekday?: string;
  day?: number | null;
  month?: number | null;
}

export type BillSchedule = RecurringSchedule;

export interface RecurringTransaction {
  id: string;
  user_id: string;
  account_id: string | null;
  category_id: string | null;
  amount: number;
  last_date_applied: string | null;
  frequency: RecurringFrequency;
  interval: number;
  schedule: RecurringSchedule | null;
  name: string;
  entity: string;
  description: string;
  reminder: boolean;
  next_due_date: string | null;
  due_status: RecurringDueStatus;
  days_until_due: number | null;
  created_at: string;
  updated_at: string;
}

export interface RecurringTransactionRequest {
  account_id?: string | null;
  category_id?: string | null;
  amount: number;
  last_date_applied?: string | null;
  frequency: RecurringFrequency;
  interval?: number;
  schedule?: RecurringSchedule | null;
  name?: string;
  entity: string;
  description?: string;
  reminder?: boolean;
}

export interface RecurringTransactionApplyRequest {
  account_id?: string | null;
  date?: string | null;
  date_cleared?: string | null;
}

export interface RecurringTransactionApplyResponse {
  recurring_transaction: RecurringTransaction;
  record: AccountRecord;
}
