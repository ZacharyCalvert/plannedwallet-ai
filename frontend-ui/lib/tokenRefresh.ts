export const TOKEN_REFRESH_THRESHOLD_MS = 5 * 60 * 1000;
export const TOKEN_REFRESH_FAILURE_COOLDOWN_MS = 15 * 1000;

function decodeBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
  if (typeof globalThis.atob === 'function') {
    const binary = globalThis.atob(padded);
    const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  }
  return Buffer.from(padded, 'base64').toString('utf8');
}

export function parseJwtClaims(token: string): Record<string, unknown> | null {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3 || !parts[1]) {
    return null;
  }
  try {
    const parsed = JSON.parse(decodeBase64Url(parts[1]));
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}

export function isJwtTermsApproved(token: string): boolean {
  const claims = parseJwtClaims(token);
  if (!claims) {
    return true;
  }
  return claims.terms_approved !== false;
}

export function isJwtSubscriber(token: string): boolean {
  const claims = parseJwtClaims(token);
  if (!claims) {
    return false;
  }
  return claims.subscriber === true;
}

export function parseJwtExpirationMs(token: string): number | null {
  const parsed = parseJwtClaims(token);
  if (!parsed || typeof parsed.exp !== 'number' || !Number.isFinite(parsed.exp)) {
    return null;
  }
  return parsed.exp * 1000;
}

export interface RefreshEvaluationInput {
  token: string;
  isAppFocused: boolean;
  inFlight: boolean;
  nowMs?: number;
  lastFailureAtMs?: number | null;
  thresholdMs?: number;
  failureCooldownMs?: number;
}

export function shouldRefreshTokenOnActivity({
  token,
  isAppFocused,
  inFlight,
  nowMs = Date.now(),
  lastFailureAtMs = null,
  thresholdMs = TOKEN_REFRESH_THRESHOLD_MS,
  failureCooldownMs = TOKEN_REFRESH_FAILURE_COOLDOWN_MS,
}: RefreshEvaluationInput): boolean {
  if (!token || !isAppFocused || inFlight) {
    return false;
  }
  if (lastFailureAtMs !== null && nowMs - lastFailureAtMs < failureCooldownMs) {
    return false;
  }
  const expMs = parseJwtExpirationMs(token);
  if (expMs === null) {
    return false;
  }
  const remainingMs = expMs - nowMs;
  return remainingMs > 0 && remainingMs < thresholdMs;
}

export function getRemainingSessionMs(token: string, nowMs = Date.now()): number | null {
  const expMs = parseJwtExpirationMs(token);
  if (expMs === null) {
    return null;
  }
  return Math.max(0, expMs - nowMs);
}
