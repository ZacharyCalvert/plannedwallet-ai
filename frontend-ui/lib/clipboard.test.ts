import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { copyTextToClipboard } from './clipboard.ts';

function stubGlobals(opts: {
  clipboard?: { writeText: (t: string) => Promise<void> };
  execResult?: boolean;
}): { copiedViaExec: string[]; restore: () => void } {
  const origNav = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  const origDoc = Object.getOwnPropertyDescriptor(globalThis, 'document');
  const copiedViaExec: string[] = [];
  const fakeTextarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute() {},
    focus() {},
    select() {},
    setSelectionRange() {},
  };

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: opts.clipboard ? { clipboard: opts.clipboard } : {},
  });
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      body: {
        appendChild() {},
        removeChild() {},
      },
      createElement: () => fakeTextarea,
      execCommand: () => {
        copiedViaExec.push(fakeTextarea.value);
        return opts.execResult ?? true;
      },
    },
  });

  return {
    copiedViaExec,
    restore: () => {
      if (origNav) Object.defineProperty(globalThis, 'navigator', origNav);
      if (origDoc) Object.defineProperty(globalThis, 'document', origDoc);
    },
  };
}

describe('Clipboard Copy Helpers', () => {
  it('uses navigator.clipboard.writeText when available', async () => {
    let written = '';
    const { restore } = stubGlobals({
      clipboard: {
        writeText: async (text) => {
          written = text;
        },
      },
    });
    try {
      const ok = await copyTextToClipboard('twelve secret words');
      assert.equal(ok, true);
      assert.equal(written, 'twelve secret words');
    } finally {
      restore();
    }
  });

  it('falls back to document.execCommand copy when navigator.clipboard is unavailable', async () => {
    const { copiedViaExec, restore } = stubGlobals({ execResult: true });
    try {
      const ok = await copyTextToClipboard('fallback recovery phrase');
      assert.equal(ok, true);
      assert.deepEqual(copiedViaExec, ['fallback recovery phrase']);
    } finally {
      restore();
    }
  });

  it('falls back to document.execCommand copy when clipboard.writeText rejects', async () => {
    const { copiedViaExec, restore } = stubGlobals({
      clipboard: {
        writeText: async () => {
          throw new Error('NotAllowedError');
        },
      },
      execResult: true,
    });
    try {
      const ok = await copyTextToClipboard('rejected clipboard phrase');
      assert.equal(ok, true);
      assert.deepEqual(copiedViaExec, ['rejected clipboard phrase']);
    } finally {
      restore();
    }
  });
});
