import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import {
  computeBufferSha256Hex,
  fetchPrivacyPdfSha256,
  fetchTermsPdfSha256,
  isPolicyApprovalRequiredResponse,
  notifyPolicyRequired,
  POLICY_APPROVAL_ERROR_CODE,
  PRIVACY_PDF_URL,
  sha256HexFallback,
  subscribePolicyRequired,
  TERMS_PDF_URL,
} from './policyAgreement.ts';

describe('Policy Agreement Helpers', () => {
  it('computes lowercase SHA-256 hex matching the static terms-of-use.pdf bytes', async () => {
    const here = path.dirname(fileURLToPath(import.meta.url));
    const pdfPath = path.resolve(here, '../public/terms-of-use.pdf');
    const pdfBytes = await readFile(pdfPath);
    const expectedHash = createHash('sha256').update(pdfBytes).digest('hex');

    const webCryptoHash = await computeBufferSha256Hex(pdfBytes);
    assert.equal(webCryptoHash, expectedHash);
    assert.equal(webCryptoHash.length, 64);

    assert.equal(sha256HexFallback(pdfBytes), expectedHash);
    assert.equal(
      sha256HexFallback(new Uint8Array(0)),
      'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
    );
    assert.equal(
      sha256HexFallback(new TextEncoder().encode('abc')),
      'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad'
    );

    let capturedUrl = '';
    let capturedCache = '';
    const fetchedHash = await fetchTermsPdfSha256(async (input, init) => {
      capturedUrl = String(input);
      capturedCache = init?.cache ?? '';
      return new Response(pdfBytes, { status: 200 });
    });

    assert.equal(capturedUrl, TERMS_PDF_URL);
    assert.equal(capturedCache, 'no-store');
    assert.equal(fetchedHash, expectedHash);
  });

  it('computes lowercase SHA-256 hex matching the static privacy-and-security-policy.pdf bytes', async () => {
    const here = path.dirname(fileURLToPath(import.meta.url));
    const pdfPath = path.resolve(here, '../public/privacy-and-security-policy.pdf');
    const pdfBytes = await readFile(pdfPath);
    const expectedHash = createHash('sha256').update(pdfBytes).digest('hex');

    let capturedUrl = '';
    let capturedCache = '';
    const fetchedHash = await fetchPrivacyPdfSha256(async (input, init) => {
      capturedUrl = String(input);
      capturedCache = init?.cache ?? '';
      return new Response(pdfBytes, { status: 200 });
    });

    assert.equal(capturedUrl, PRIVACY_PDF_URL);
    assert.equal(capturedCache, 'no-store');
    assert.equal(fetchedHash, expectedHash);
  });

  it('throws if downloading terms-of-use.pdf or privacy-and-security-policy.pdf fails', async () => {
    await assert.rejects(
      () => fetchTermsPdfSha256(async () => new Response('missing', { status: 404 })),
      /Unable to download Terms of Use PDF/
    );
    await assert.rejects(
      () => fetchPrivacyPdfSha256(async () => new Response('missing', { status: 404 })),
      /Unable to download Privacy & Security Policy PDF/
    );
  });

  it('detects 403 POLICY_APPROVAL_REQUIRED responses and notifies subscribers', () => {
    assert.equal(
      isPolicyApprovalRequiredResponse(403, { error: POLICY_APPROVAL_ERROR_CODE }),
      true
    );
    assert.equal(
      isPolicyApprovalRequiredResponse(403, { error: 'FORBIDDEN' }),
      false
    );
    assert.equal(
      isPolicyApprovalRequiredResponse(401, { error: POLICY_APPROVAL_ERROR_CODE }),
      false
    );

    let calls = 0;
    const unsubscribe = subscribePolicyRequired(() => {
      calls += 1;
    });
    notifyPolicyRequired();
    assert.equal(calls, 1);
    unsubscribe();
    notifyPolicyRequired();
    assert.equal(calls, 1);
  });
});
