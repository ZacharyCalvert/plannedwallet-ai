export const STAGE_HOSTNAME = 'stage.plannedwallet.com';

export const STAGE_BANNER_TEXT =
  'Development / Staging Environment — This is not a production service. Data entered here is for testing only and may be reset.';

/**
 * Returns true when the provided hostname (case-insensitive, optional port stripped)
 * matches the staging host (`stage.plannedwallet.com`), regardless of http or https.
 */
export function isStagingHost(hostname?: string | null): boolean {
  if (!hostname) {
    return false;
  }
  const normalized = hostname.trim().toLowerCase().split(':')[0];
  return normalized === STAGE_HOSTNAME;
}

/**
 * Safely checks `window.location.hostname` in browser environments.
 */
export function isStagingBrowser(): boolean {
  if (typeof window === 'undefined' || !window.location) {
    return false;
  }
  return isStagingHost(window.location.hostname);
}
