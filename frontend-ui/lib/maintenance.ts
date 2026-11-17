export const READ_ONLY_FEATURE = 'read_only';
export const READ_ONLY_ERROR_CODE = 'READ_ONLY_MODE';
export const MAINTENANCE_BANNER_TEXT = 'Planned Wallet is under maintenance.';
export const MAINTENANCE_EDIT_WARNING =
  'Edits are not working while the system is undergoing maintenance.';

export interface ApiErrorBody {
  error?: string;
  message?: string;
}

export function isReadOnlyFeatureEnabled(features: string[]): boolean {
  return Array.isArray(features) && features.includes(READ_ONLY_FEATURE);
}

export function isReadOnlyErrorResponse(status: number, body?: ApiErrorBody | null): boolean {
  return status === 503 || body?.error === READ_ONLY_ERROR_CODE;
}

type MaintenanceListener = (message: string) => void;
const listeners = new Set<MaintenanceListener>();

export function subscribeMaintenanceWarning(listener: MaintenanceListener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function notifyMaintenanceWarning(message = MAINTENANCE_EDIT_WARNING): void {
  for (const listener of listeners) {
    listener(message);
  }
}
