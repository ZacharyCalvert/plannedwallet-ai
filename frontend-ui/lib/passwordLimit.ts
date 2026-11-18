export const MAX_PASSWORD_BYTES = 72;

const encoder = new TextEncoder();

export function utf8ByteLength(value: string): number {
  return encoder.encode(value).length;
}

export function clampPasswordToMaxBytes(
  value: string,
  maxBytes = MAX_PASSWORD_BYTES
): string {
  if (utf8ByteLength(value) <= maxBytes) {
    return value;
  }
  const codePoints = Array.from(value);
  let totalBytes = 0;
  let keptCount = 0;
  for (const cp of codePoints) {
    const cpBytes = encoder.encode(cp).length;
    if (totalBytes + cpBytes > maxBytes) {
      break;
    }
    totalBytes += cpBytes;
    keptCount += 1;
  }
  return codePoints.slice(0, keptCount).join('');
}
