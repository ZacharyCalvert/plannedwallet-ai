// W3C Trace Context Utilities (W3C Recommendation: https://www.w3.org/TR/trace-context/)

export interface TraceContext {
  version: string;
  traceId: string;
  spanId: string;
  traceFlags: string;
}

function getRandomHex(byteCount: number): string {
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    const bytes = new Uint8Array(byteCount);
    crypto.getRandomValues(bytes);
    return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  }
  let hex = '';
  for (let i = 0; i < byteCount; i++) {
    hex += Math.floor(Math.random() * 256).toString(16).padStart(2, '0');
  }
  return hex;
}

export function generateTraceId(): string {
  let id = getRandomHex(16);
  while (id === '00000000000000000000000000000000') {
    id = getRandomHex(16);
  }
  return id;
}

export function generateSpanId(): string {
  let id = getRandomHex(8);
  while (id === '0000000000000000') {
    id = getRandomHex(8);
  }
  return id;
}

export function createTraceparent(): string {
  return `00-${generateTraceId()}-${generateSpanId()}-01`;
}

export function logTraceInfo(traceparent: string, message: string, ...args: unknown[]) {
  console.info(`[W3C Trace: ${traceparent}] ${message}`, ...args);
}

export function logTraceError(traceparent: string, message: string, ...args: unknown[]) {
  console.error(`[W3C Trace: ${traceparent}] ${message}`, ...args);
}
