export function isoToDisplayDate(isoStr: string): string {
  if (!isoStr) return '';
  const clean = isoStr.trim();
  const isoMatch = /^(\d{4})-(\d{2})-(\d{2})(?:$|T)/.exec(clean);
  if (isoMatch) {
    const [, y, m, d] = isoMatch;
    return `${m}/${d}/${y}`;
  }
  return clean;
}

export function displayDateToIso(displayStr: string): string {
  const clean = displayStr.trim();
  if (!clean) return '';

  if (/^\d{4}-\d{2}-\d{2}$/.test(clean)) {
    return clean;
  }

  const parts = clean.split('/');
  if (parts.length === 3) {
    const [rawM, rawD, rawY] = parts;
    const m = rawM.padStart(2, '0');
    const d = rawD.padStart(2, '0');
    const y = /^\d{2}$/.test(rawY) && rawY !== '20' ? `20${rawY}` : rawY;
    const numM = parseInt(m, 10);
    const numD = parseInt(d, 10);
    if (y.length === 4 && numM >= 1 && numM <= 12 && numD >= 1 && numD <= 31) {
      return `${y}-${m}-${d}`;
    }
  }

  if (/^\d{8}$/.test(clean)) {
    return `${clean.slice(4, 8)}-${clean.slice(0, 2)}-${clean.slice(2, 4)}`;
  }

  return clean;
}

export function applyDateAutoSlash(nextRaw: string, prevDisplay: string): string {
  const val = nextRaw.replace(/[^\d/]/g, '');
  const grew = val.length > prevDisplay.length;
  if (grew && !prevDisplay.endsWith('/') && (/^\d{2}$/.test(val) || /^\d{2}\/\d{2}$/.test(val))) {
    return `${val}/`;
  }
  const shortYearMatch = /^(\d{1,2}\/\d{1,2})\/(\d{2})$/.exec(val);
  if (grew && shortYearMatch && shortYearMatch[2] !== '20') {
    return `${shortYearMatch[1]}/20${shortYearMatch[2]}`;
  }
  return val;
}

function clampTwoDigitSegment(raw: string, max: number): string {
  const trimmed = raw.trim();
  const padded = !trimmed || trimmed === '0' || trimmed === '00' ? '01' : trimmed.padStart(2, '0');
  const num = parseInt(padded, 10);
  if (num > max) return String(max);
  if (num < 1) return '01';
  return padded;
}

export type SlashKeyAction =
  | { action: 'ignore' }
  | { action: 'focusNext' }
  | { action: 'update'; value: string; cursorPos: number };

export function computeSlashKeyAction(val: string, start: number, end: number): SlashKeyAction {
  const textBefore = val.slice(0, start);
  const textAfter = val.slice(end).replace(/^\//, '');
  const parts = textBefore.split('/');
  const partIdx = parts.length - 1;
  const currentPart = parts[partIdx];

  if (partIdx > 0 && currentPart === '') {
    return { action: 'ignore' };
  }
  if (partIdx === 0) {
    const m = clampTwoDigitSegment(currentPart, 12);
    return { action: 'update', value: `${m}/${textAfter}`, cursorPos: m.length + 1 };
  }
  if (partIdx === 1) {
    const d = clampTwoDigitSegment(currentPart, 31);
    const monthPart = parts[0];
    return {
      action: 'update',
      value: `${monthPart}/${d}/${textAfter}`,
      cursorPos: monthPart.length + 1 + d.length + 1,
    };
  }
  return { action: 'focusNext' };
}
