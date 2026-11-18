'use client';

import { useEffect, useMemo, useState } from 'react';
import { copyTextToClipboard } from '../lib/clipboard';
import {
  buildRecoveryKitContent,
  pickSpotCheckIndices,
  splitRecoveryPhrase,
  verifySpotCheckAnswers,
} from '../lib/recoveryPhrase';

interface RecoveryPhraseSetupProps {
  username: string;
  phrase: string;
  title?: string;
  subtitle?: string;
  continueLabel?: string;
  onConfirmed: () => void;
  onCancel?: () => void;
}

export function RecoveryPhraseSetup({
  username,
  phrase,
  title = 'Save Your 12-Word Recovery Phrase',
  subtitle = 'Shown only once. If you lose both your password and authenticator device, this phrase is the only way to recover your account.',
  continueLabel = 'Continue',
  onConfirmed,
  onCancel,
}: RecoveryPhraseSetupProps) {
  const words = useMemo(() => splitRecoveryPhrase(phrase), [phrase]);
  const [indices, setIndices] = useState<[number, number]>([1, 2]);
  const [wordOneAnswer, setWordOneAnswer] = useState('');
  const [wordTwoAnswer, setWordTwoAnswer] = useState('');
  const [savedConfirmed, setSavedConfirmed] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setIndices(pickSpotCheckIndices(words.length));
    setWordOneAnswer('');
    setWordTwoAnswer('');
    setSavedConfirmed(false);
    setCopied(false);
    setError('');
  }, [phrase, words.length]);

  const spotMatched = verifySpotCheckAnswers(words, indices, [wordOneAnswer, wordTwoAnswer]);
  const canProceed = savedConfirmed && spotMatched;

  const handleCopy = async () => {
    const ok = await copyTextToClipboard(phrase);
    if (ok) {
      setError('');
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2500);
      return;
    }
    setError('Clipboard copy unavailable; please download the recovery kit or write down the 12 words.');
  };

  const handleDownload = () => {
    const content = buildRecoveryKitContent(username, phrase);
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${username}-recovery-kit.txt`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!spotMatched) {
      setError(`Please enter exact Word #${indices[0]} and Word #${indices[1]} to confirm you saved your phrase.`);
      return;
    }
    if (!savedConfirmed) {
      setError('Please confirm that you stored your recovery phrase safely.');
      return;
    }
    setError('');
    onConfirmed();
  };

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      <h3 className="totp-title">{title}</h3>
      <div className="recovery-warning-box" role="note">
        <strong>Shown Only Once:</strong> {subtitle}
      </div>

      <div className="recovery-word-grid" aria-label="12-word recovery phrase">
        {words.map((word, idx) => (
          <div key={`${idx + 1}-${word}`} className="recovery-word-item">
            <span className="recovery-word-index">{idx + 1}.</span>
            <span className="recovery-word-text">{word}</span>
          </div>
        ))}
      </div>

      <div className="recovery-actions-row">
        <button type="button" className="btn-secondary btn-sm" onClick={handleCopy}>
          {copied ? 'Copied Phrase!' : 'Copy Phrase'}
        </button>
        <button type="button" className="btn-secondary btn-sm" onClick={handleDownload}>
          Download Recovery Kit (.txt)
        </button>
      </div>

      <div className="recovery-spot-check">
        <p className="text-sm font-semibold">
          Verify Backup: Enter Word #{indices[0]} and Word #{indices[1]}
        </p>
        <div className="recovery-spot-inputs">
          <div className="form-group">
            <label htmlFor="spot-word-a">Word #{indices[0]}</label>
            <input
              id="spot-word-a"
              type="text"
              autoComplete="off"
              required
              value={wordOneAnswer}
              onChange={(e) => setWordOneAnswer(e.target.value)}
              placeholder={`Enter word #${indices[0]}`}
            />
          </div>
          <div className="form-group">
            <label htmlFor="spot-word-b">Word #{indices[1]}</label>
            <input
              id="spot-word-b"
              type="text"
              autoComplete="off"
              required
              value={wordTwoAnswer}
              onChange={(e) => setWordTwoAnswer(e.target.value)}
              placeholder={`Enter word #${indices[1]}`}
            />
          </div>
        </div>

        <label className="policy-checkbox-row">
          <input
            type="checkbox"
            checked={savedConfirmed}
            onChange={(e) => setSavedConfirmed(e.target.checked)}
          />
          <span>
            I have saved my 12-word recovery phrase securely and understand it cannot be retrieved later.
          </span>
        </label>
      </div>

      {error && <div className="alert-error" role="alert">{error}</div>}

      <div className="form-actions">
        {onCancel && (
          <button type="button" className="btn-secondary" onClick={onCancel}>
            Back
          </button>
        )}
        <button type="submit" className="btn-primary" disabled={!canProceed}>
          {continueLabel}
        </button>
      </div>
    </form>
  );
}
