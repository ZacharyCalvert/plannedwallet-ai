export interface ColumnMapping {
  dateSentIndex: number;
  dateClearedIndex: number;
  recipientIndex: number;
  descriptionIndex: number;
  amountIndex: number;
}

export interface ParsedCsvRow {
  rowNumber: number;
  dateSent: string;
  dateCleared: string | null;
  recipient: string;
  description: string;
  amount: number;
  isValid: boolean;
  errors: string[];
}

export interface ParsedCsvResult {
  success: boolean;
  error?: string;
  headers: string[];
  mapping?: ColumnMapping;
  rows: ParsedCsvRow[];
  validCount: number;
  invalidCount: number;
  emptyCount: number;
}

export const MIN_RECORD_YEAR = 2000;
export const MAX_RECORD_YEAR = 2100;
export const MAX_BULK_UPLOAD_RECORDS = 10000;

// Supported aliases for header columns (case-insensitive)
const DATE_SENT_ALIASES = [
  'date sent',
  'date',
  'sent',
  'spent on',
  'transaction date',
  'trans date',
  'posting date',
  'post date',
  'effective date',
  'trade date',
];
const DATE_CLEARED_ALIASES = [
  'date cleared',
  'cleared date',
  'cleared',
  'cleared on',
  'reconciled',
  'reconciled on',
  'reconcile date',
  'reconciliation date',
  'settlement date',
  'settled',
  'settled on',
  'value date',
  'posting date',
  'post date',
];
const RECIPIENT_ALIASES = ['recipient', 'entity', 'payee', 'merchant', 'vendor', 'to', 'name'];
const DESCRIPTION_ALIASES = ['description', 'notes', 'details', 'memo', 'comment', 'narrative'];
const AMOUNT_ALIASES = ['amount', 'value', 'total', 'sum', 'cost', 'charge'];

/**
 * Parses a single CSV line according to RFC-4180 (handling quoted values and escaped quotes)
 */
export function parseCsvLine(line: string): string[] {
  const fields: string[] = [];
  let current = '';
  let inQuotes = false;
  let i = 0;

  while (i < line.length) {
    const char = line[i];

    if (inQuotes) {
      if (char === '"') {
        if (i + 1 < line.length && line[i + 1] === '"') {
          current += '"';
          i += 2;
        } else {
          inQuotes = false;
          i++;
        }
      } else {
        current += char;
        i++;
      }
    } else {
      if (char === '"') {
        inQuotes = true;
        i++;
      } else if (char === ',') {
        fields.push(current.trim());
        current = '';
        i++;
      } else {
        current += char;
        i++;
      }
    }
  }

  fields.push(current.trim());
  return fields;
}

/**
 * Splits CSV text into an array of lines, properly handling multi-line quoted fields
 * and stripping UTF-8 BOM if present.
 */
export function splitCsvLines(text: string): string[] {
  const cleanText = text.replace(/^\uFEFF/, '');
  const rawLines = cleanText.split(/\r\n|\n|\r/);
  const result: string[] = [];
  let buffer = '';
  let insideQuotes = false;

  for (const line of rawLines) {
    if (buffer) {
      buffer += '\n' + line;
    } else {
      buffer = line;
    }

    // Count quotes that are not preceded by an escape
    let quoteCount = 0;
    for (let i = 0; i < buffer.length; i++) {
      if (buffer[i] === '"') quoteCount++;
    }

    insideQuotes = quoteCount % 2 !== 0;

    if (!insideQuotes) {
      if (buffer.trim()) {
        result.push(buffer);
      }
      buffer = '';
    }
  }

  if (buffer.trim()) {
    result.push(buffer);
  }

  return result;
}

/**
 * Matches detected CSV headers with supported column aliases without collisions
 */
export function matchHeaders(headers: string[]): {
  mapping: ColumnMapping;
  missingRequired: string[];
} {
  const normalized = headers.map((h) => h.toLowerCase().trim().replace(/^["']|["']$/g, ''));

  const findIndex = (aliases: string[], excludeIndices: number[] = []): number => {
    return normalized.findIndex((h, idx) => !excludeIndices.includes(idx) && aliases.includes(h));
  };

  const dateSentIndex = findIndex(DATE_SENT_ALIASES);
  const dateClearedIndex = findIndex(DATE_CLEARED_ALIASES, [dateSentIndex]);
  const recipientIndex = findIndex(RECIPIENT_ALIASES, [dateSentIndex, dateClearedIndex]);
  const descriptionIndex = findIndex(DESCRIPTION_ALIASES, [dateSentIndex, dateClearedIndex, recipientIndex]);
  const amountIndex = findIndex(AMOUNT_ALIASES, [dateSentIndex, dateClearedIndex, recipientIndex, descriptionIndex]);

  const missingRequired: string[] = [];
  if (dateSentIndex === -1) missingRequired.push('Date Sent (supported: Date Sent, Date, Sent, Spent On, Transaction Date)');
  if (recipientIndex === -1) missingRequired.push('Recipient (supported: Recipient, Entity, Payee, Merchant, Vendor, To)');
  if (amountIndex === -1) missingRequired.push('Amount (supported: Amount, Value, Total, Sum)');

  return {
    mapping: {
      dateSentIndex,
      dateClearedIndex,
      recipientIndex,
      descriptionIndex,
      amountIndex,
    },
    missingRequired,
  };
}

/**
 * Formats a number with leading zeroes
 */
function pad2(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

function pad4(n: number): string {
  return String(n).padStart(4, '0');
}

/**
 * Validates calendar date constraints including leap years
 */
function isValidDate(year: number, month: number, day: number): boolean {
  if (month < 1 || month > 12) return false;
  if (day < 1 || day > 31) return false;
  const daysInMonth = new Date(year, month, 0).getDate();
  return day <= daysInMonth;
}

/**
 * Parses and normalizes dates in MM/dd/yy, mm/dd/yyyy, or yyyy-MM-dd formats into YYYY-MM-DD.
 * Strictly rejects 1-digit, 3-digit, or >4-digit years, and validates calendar validity (leap years, month days).
 */
export function parseDate(val: string): { dateStr: string | null; error?: string } {
  let clean = val.trim();
  if (!clean) return { dateStr: null };

  // Strip enclosing quotes if present
  clean = clean.replace(/^["']|["']$/g, '').trim();
  if (!clean) return { dateStr: null };

  // Strip trailing time portion if present (e.g. "11/15/2026 12:00:00 AM", "2026-11-15T00:00:00Z")
  const timeSplit = clean.split(/[T\s]/);
  const datePart = timeSplit[0].trim();

  // Format 1: ISO YYYY-MM-DD, YYYY/MM/DD, or YYYY.MM.DD
  const isoMatch = /^(\d{4})[\/\-\.](\d{1,2})[\/\-\.](\d{1,2})$/.exec(datePart);
  if (isoMatch) {
    const y = parseInt(isoMatch[1], 10);
    const m = parseInt(isoMatch[2], 10);
    const d = parseInt(isoMatch[3], 10);
    if (y < MIN_RECORD_YEAR || y > MAX_RECORD_YEAR) {
      return {
        dateStr: null,
        error:
          y < MIN_RECORD_YEAR
            ? `Date is older than year ${MIN_RECORD_YEAR} (${y}): records before ${MIN_RECORD_YEAR} are not permitted`
            : `Year exceeds maximum allowed limit (${MAX_RECORD_YEAR}): ${y}`,
      };
    }
    if (!isValidDate(y, m, d)) {
      return { dateStr: null, error: `Invalid calendar date: "${datePart}"` };
    }
    return { dateStr: `${pad4(y)}-${pad2(m)}-${pad2(d)}` };
  }

  // Format 2: MM/dd/yyyy or MM/dd/yy (strictly 2 or 4 digits for year)
  const slashMatch = /^(\d{1,2})[\/\-\.](\d{1,2})[\/\-\.](\d{2}|\d{4})$/.exec(datePart);
  if (slashMatch) {
    const m = parseInt(slashMatch[1], 10);
    const d = parseInt(slashMatch[2], 10);
    const yearStr = slashMatch[3];
    let y = parseInt(yearStr, 10);

    // 2-digit year conversion: yy >= 70 -> 19yy, else 20yy
    if (yearStr.length === 2) {
      y = y >= 70 ? 1900 + y : 2000 + y;
    }

    if (y < MIN_RECORD_YEAR || y > MAX_RECORD_YEAR) {
      return {
        dateStr: null,
        error:
          y < MIN_RECORD_YEAR
            ? `Date is older than year ${MIN_RECORD_YEAR} (${y}): records before ${MIN_RECORD_YEAR} are not permitted`
            : `Year exceeds maximum allowed limit (${MAX_RECORD_YEAR}): ${y}`,
      };
    }
    if (!isValidDate(y, m, d)) {
      return { dateStr: null, error: `Invalid calendar date: "${datePart}"` };
    }
    return { dateStr: `${pad4(y)}-${pad2(m)}-${pad2(d)}` };
  }

  return { dateStr: null, error: `Unrecognized date format: "${val}". Expected MM/dd/yy or mm/dd/yyyy` };
}

/**
 * Parses numeric monetary amounts commonly found in spreadsheets:
 * strips $ currency symbols, commas, and quotes, and treats values surrounded
 * by parentheses e.g. (105.10), $(105.10), or ($105.10) as negative numbers.
 */
export function parseAmount(val: string): { value: number | null; error?: string } {
  let clean = val
    .trim()
    .replace(/["']/g, '')
    .replace(/\$/g, '')
    .replace(/,/g, '')
    .trim();

  if (!clean) {
    return { value: null, error: 'Amount is required' };
  }

  let isNegative = false;

  // Accounting parenthesis format: (105.10), $(105.10), -(105.10) -> negative
  if (clean.startsWith('-(') && clean.endsWith(')')) {
    isNegative = true;
    clean = clean.slice(2, -1).trim();
  } else if (clean.startsWith('(') && clean.endsWith(')')) {
    isNegative = true;
    clean = clean.slice(1, -1).trim();
  } else if (clean.startsWith('-')) {
    isNegative = true;
    clean = clean.slice(1).trim();
  } else if (clean.startsWith('+')) {
    clean = clean.slice(1).trim();
  }

  // Handle redundant inner negative sign if present: (-105.10)
  if (clean.startsWith('-')) {
    isNegative = true;
    clean = clean.slice(1).trim();
  }

  const num = parseFloat(clean);
  if (isNaN(num) || !isFinite(num)) {
    return { value: null, error: `Invalid numeric amount: "${val}"` };
  }

  const finalVal = isNegative ? -Math.abs(num) : num;
  const rounded = Math.round(finalVal * 100) / 100;
  return { value: rounded === 0 ? 0 : rounded };
}

/**
 * Ingests, parses, and validates raw CSV text entirely in memory
 */
export function parseAndValidateCsv(text: string): ParsedCsvResult {
  const lines = splitCsvLines(text);
  if (lines.length === 0) {
    return {
      success: false,
      error: 'The CSV file is empty.',
      headers: [],
      rows: [],
      validCount: 0,
      invalidCount: 0,
      emptyCount: 0,
    };
  }

  const headerLine = lines[0];
  const headers = parseCsvLine(headerLine);
  if (headers.length === 0 || headers.every((h) => !h)) {
    return {
      success: false,
      error: 'No valid header row found in CSV.',
      headers: [],
      rows: [],
      validCount: 0,
      invalidCount: 0,
      emptyCount: 0,
    };
  }

  const { mapping, missingRequired } = matchHeaders(headers);
  if (missingRequired.length > 0) {
    return {
      success: false,
      error: `Missing required column(s): ${missingRequired.join('; ')}`,
      headers,
      mapping,
      rows: [],
      validCount: 0,
      invalidCount: 0,
      emptyCount: 0,
    };
  }

  const parsedRows: ParsedCsvRow[] = [];
  let validCount = 0;
  let invalidCount = 0;
  let emptyCount = 0;

  for (let i = 1; i < lines.length; i++) {
    const rawFields = parseCsvLine(lines[i]);
    const rowNumber = i + 1; // 1-indexed for user display

    // Extract raw string values for mapped columns
    const rawDateSent = (rawFields[mapping.dateSentIndex] || '').trim();
    const rawRecipient = (rawFields[mapping.recipientIndex] || '').trim();
    const rawAmount = (rawFields[mapping.amountIndex] || '').trim();
    const rawDateCleared = (
      mapping.dateClearedIndex !== -1 ? rawFields[mapping.dateClearedIndex] || '' : ''
    ).trim();
    const rawDescription = (
      mapping.descriptionIndex !== -1 ? rawFields[mapping.descriptionIndex] || '' : ''
    ).trim();

    // Skip/discard any record which has none of the required AND none of the optional fields
    const hasAnyContent = Boolean(
      rawDateSent || rawRecipient || rawAmount || rawDateCleared || rawDescription
    );

    if (!hasAnyContent) {
      emptyCount++;
      continue; // Skip / discard completely empty row from view and processing
    }

    const errors: string[] = [];

    // 1. Date Sent (Required: valid calendar date formatted as YYYY-MM-DD)
    const dateSentRes = parseDate(rawDateSent);
    if (!rawDateSent) {
      errors.push('Date Sent is required');
    } else if (!dateSentRes.dateStr) {
      errors.push(
        dateSentRes.error
          ? `Date Sent: ${dateSentRes.error}`
          : `Date Sent: Invalid date format "${rawDateSent}". Expected MM/dd/yy or mm/dd/yyyy`
      );
    }

    // 2. Date Cleared (Optional: if provided, must be a valid calendar date)
    let dateClearedStr: string | null = null;
    const cleanDateCleared = rawDateCleared.trim();
    if (cleanDateCleared) {
      const dateClearedRes = parseDate(cleanDateCleared);
      if (dateClearedRes.dateStr) {
        dateClearedStr = dateClearedRes.dateStr;
      } else {
        errors.push(
          dateClearedRes.error
            ? `Date Cleared: ${dateClearedRes.error}`
            : `Date Cleared: Invalid date format "${rawDateCleared}". Expected MM/dd/yy or mm/dd/yyyy`
        );
      }
    }

    // 3. Recipient / Entity (Required: 1-255 characters)
    if (!rawRecipient) {
      errors.push('Recipient cannot be blank');
    } else if (rawRecipient.length > 255) {
      errors.push(`Recipient cannot exceed 255 characters (length: ${rawRecipient.length})`);
    }

    // 4. Description (Optional: max 500 characters)
    const description = rawDescription;
    if (rawDescription.length > 500) {
      errors.push(`Description cannot exceed 500 characters (length: ${rawDescription.length})`);
    }

    // 5. Amount (Required: finite numeric value)
    const amountRes = parseAmount(rawAmount);
    if (amountRes.value === null) {
      errors.push(amountRes.error || 'Amount is required');
    }

    const isValid = errors.length === 0;
    if (isValid) {
      validCount++;
    } else {
      invalidCount++;
    }

    parsedRows.push({
      rowNumber,
      dateSent: dateSentRes.dateStr || rawDateSent,
      dateCleared: dateClearedStr || (cleanDateCleared ? rawDateCleared : null),
      recipient: rawRecipient,
      description,
      amount: amountRes.value ?? 0,
      isValid,
      errors,
    });
  }

  // Bring any validation failed records to the top of the review pane
  parsedRows.sort((a, b) => {
    if (!a.isValid && b.isValid) return -1;
    if (a.isValid && !b.isValid) return 1;
    return a.rowNumber - b.rowNumber;
  });

  if (parsedRows.length === 0) {
    return {
      success: false,
      error:
        emptyCount > 0
          ? `The CSV file contains only empty records (${emptyCount} empty record${emptyCount === 1 ? '' : 's'} found).`
          : 'The CSV file contains a header but no data rows.',
      headers,
      mapping,
      rows: [],
      validCount: 0,
      invalidCount: 0,
      emptyCount,
    };
  }

  // Enforce maximum records limit of 10,000 records per bulk upload
  if (parsedRows.length > MAX_BULK_UPLOAD_RECORDS) {
    return {
      success: false,
      error: `File contains ${parsedRows.length.toLocaleString()} records, exceeding the maximum limit of ${MAX_BULK_UPLOAD_RECORDS.toLocaleString()} records per bulk upload.`,
      headers,
      mapping,
      rows: [],
      validCount: 0,
      invalidCount: 0,
      emptyCount,
    };
  }

  return {
    success: true,
    headers,
    mapping,
    rows: parsedRows,
    validCount,
    invalidCount,
    emptyCount,
  };
}
