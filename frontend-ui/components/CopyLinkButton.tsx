'use client';

import { useState, useEffect } from 'react';
import { copyTextToClipboard } from '../lib/clipboard';

interface CopyLinkButtonProps {
  className?: string;
}

export function CopyLinkButton({ className = '' }: CopyLinkButtonProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);

  const handleCopy = async () => {
    if (await copyTextToClipboard(window.location.href)) {
      setCopied(true);
    }
  };

  return (
    <button
      type="button"
      className={`btn-secondary btn-sm btn-copy-link ${className}`.trim()}
      onClick={handleCopy}
      title="Copy deep link URL"
      aria-label="Copy link to this account view"
    >
      {copied ? 'Copied!' : 'Copy Link'}
    </button>
  );
}
