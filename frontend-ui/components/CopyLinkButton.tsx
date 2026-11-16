'use client';

import { useState, useEffect } from 'react';

interface CopyLinkButtonProps {
  className?: string;
}

function fallbackCopyText(text: string): boolean {
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  const ok = document.execCommand('copy');
  document.body.removeChild(textarea);
  return ok;
}

export function CopyLinkButton({ className = '' }: CopyLinkButtonProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);

  const handleCopy = async () => {
    const url = window.location.href;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(url);
      } else {
        fallbackCopyText(url);
      }
      setCopied(true);
    } catch {
      if (fallbackCopyText(url)) {
        setCopied(true);
      }
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
