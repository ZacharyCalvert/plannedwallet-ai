export function splitRecoveryPhrase(phrase: string): string[] {
  return phrase
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
}

function randomInt(maxExclusive: number): number {
  if (maxExclusive <= 1) return 0;
  if (typeof globalThis.crypto?.getRandomValues === 'function') {
    const buf = new Uint32Array(1);
    globalThis.crypto.getRandomValues(buf);
    return buf[0] % maxExclusive;
  }
  return Math.floor(Math.random() * maxExclusive);
}

export function pickSpotCheckIndices(wordCount: number): [number, number] {
  if (wordCount < 2) {
    return [1, 1];
  }
  const firstIdx = randomInt(wordCount);
  let secondIdx = randomInt(wordCount - 1);
  if (secondIdx >= firstIdx) {
    secondIdx += 1;
  }
  const firstPos = firstIdx + 1;
  const secondPos = secondIdx + 1;
  return firstPos < secondPos ? [firstPos, secondPos] : [secondPos, firstPos];
}

export function verifySpotCheckAnswers(
  words: string[],
  indices: [number, number],
  answers: [string, string]
): boolean {
  const [posA, posB] = indices;
  if (posA < 1 || posB < 1 || posA > words.length || posB > words.length || posA === posB) {
    return false;
  }
  const expectedA = words[posA - 1].toLowerCase();
  const expectedB = words[posB - 1].toLowerCase();
  const actualA = answers[0].trim().toLowerCase();
  const actualB = answers[1].trim().toLowerCase();
  return actualA === expectedA && actualB === expectedB;
}

export function buildRecoveryKitContent(
  username: string,
  phrase: string,
  createdAt = new Date().toISOString()
): string {
  const words = splitRecoveryPhrase(phrase);
  const numbered = words.map((w, idx) => `${String(idx + 1).padStart(2, ' ')}. ${w}`).join('\n');
  return [
    'Planned Wallet - Emergency Recovery Kit',
    `Username: ${username}`,
    `Generated: ${createdAt}`,
    '',
    'Store this 12-word recovery phrase offline in a secure location.',
    'This phrase can reset both your password and authenticator (MFA) token once.',
    'Using this phrase permanently retires it and issues a replacement phrase.',
    '',
    numbered,
    '',
  ].join('\n');
}
