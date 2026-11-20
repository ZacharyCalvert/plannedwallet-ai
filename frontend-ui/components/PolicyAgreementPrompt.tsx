'use client';

import { FormEvent, useState } from 'react';
import { UserSession } from '../lib/types';
import { fetchPrivacyPdfSha256, fetchTermsPdfSha256 } from '../lib/policyAgreement';
import { refreshToken, submitPolicyAgreement } from '../lib/api';

interface PolicyAgreementPromptProps {
  session: UserSession;
  onApproved: (nextToken: string) => void;
  onSignOut: () => void;
}

export function PolicyAgreementPrompt({
  session,
  onApproved,
  onSignOut,
}: PolicyAgreementPromptProps) {
  const [termsApproved, setTermsApproved] = useState(false);
  const [privacyApproved, setPrivacyApproved] = useState(false);
  const [usCustomer, setUsCustomer] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const allChecked = termsApproved && privacyApproved && usCustomer;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!allChecked) {
      setError('Please confirm all three policy checkboxes to continue.');
      return;
    }

    setError('');
    setSubmitting(true);
    try {
      const [termsSha256, privacySha256] = await Promise.all([
        fetchTermsPdfSha256(),
        fetchPrivacyPdfSha256(),
      ]);
      await submitPolicyAgreement(session.token, session.username, {
        terms_approved: true,
        privacy_approved: true,
        us_adult_consumer_confirmed: true,
        terms_sha256: termsSha256,
        privacy_sha256: privacySha256,
      });
      const nextToken = await refreshToken(session.username, session.token);
      onApproved(nextToken);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to save policy agreement');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="auth-center-container">
      <div className="auth-card policy-prompt-card" role="region" aria-label="Required policy acceptance">
        <div className="auth-body">
          <h2 className="policy-prompt-title">Policy &amp; Eligibility Acceptance Required</h2>
          <p className="policy-prompt-description">
            Before accessing or modifying account data, please review and agree to the current
            Terms of Use and Privacy &amp; Security Policy, and confirm 18+ U.S. consumer eligibility.
          </p>

          <form onSubmit={handleSubmit} className="auth-form">
            <div className="policy-checkbox-list">
              <label className="policy-checkbox-row">
                <input
                  type="checkbox"
                  checked={termsApproved}
                  onChange={(e) => setTermsApproved(e.target.checked)}
                  disabled={submitting}
                />
                <span>
                  I have read and agree to the{' '}
                  <a
                    href="/terms-of-use.pdf"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="policy-link"
                  >
                    Terms of Use
                  </a>
                  .
                </span>
              </label>

              <label className="policy-checkbox-row">
                <input
                  type="checkbox"
                  checked={privacyApproved}
                  onChange={(e) => setPrivacyApproved(e.target.checked)}
                  disabled={submitting}
                />
                <span>
                  I have read and agree to the{' '}
                  <a
                    href="/privacy-and-security-policy.pdf"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="policy-link"
                  >
                    Privacy &amp; Security Policy
                  </a>
                  .
                </span>
              </label>

              <label className="policy-checkbox-row">
                <input
                  type="checkbox"
                  checked={usCustomer}
                  onChange={(e) => setUsCustomer(e.target.checked)}
                  disabled={submitting}
                />
                <span>I am at least 18 years of age and a consumer in the United States.</span>
              </label>
            </div>

            {error && (
              <div className="alert-error" role="alert">
                {error}
              </div>
            )}

            <div className="form-actions">
              <button
                type="button"
                className="btn-secondary"
                onClick={onSignOut}
                disabled={submitting}
              >
                Sign Out
              </button>
              <button
                type="submit"
                className="btn-primary"
                disabled={!allChecked || submitting}
              >
                {submitting ? 'Verifying & Saving...' : 'Agree & Continue'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
