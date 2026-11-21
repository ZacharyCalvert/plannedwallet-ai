export const ACCOUNT_DIGITS_WARNING_THRESHOLD = 4;

export const ACCOUNT_DIGITS_WARNING_TEXT =
  'Planned Wallet prohibits saving account numbers, including routing and credit card numbers (maximum 4 digits allowed).';

/**
 * Counts ASCII decimal digits (0-9) in a string.
 */
export function countDigits(value: string): number {
  let count = 0;
  for (let i = 0; i < value.length; i += 1) {
    const code = value.charCodeAt(i);
    if (code >= 48 && code <= 57) {
      count += 1;
    }
  }
  return count;
}

/**
 * Returns true once 4 or more digits are entered into an account name or description.
 */
export function hasAccountDigitsWarning(value: string): boolean {
  return countDigits(value) >= ACCOUNT_DIGITS_WARNING_THRESHOLD;
}
