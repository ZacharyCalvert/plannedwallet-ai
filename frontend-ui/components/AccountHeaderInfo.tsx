'use client';

import { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { Account } from '../lib/types';
import { patchAccount } from '../lib/api';
import {
  ACCOUNT_DIGITS_WARNING_TEXT,
  hasAccountDigitsWarning,
} from '../lib/accountValidation';
import { downloadAccountCsv } from '../lib/exportData';

type EditableField = 'name' | 'description';

interface AccountHeaderInfoProps {
  account: Account;
  token: string;
  badgeClassName?: string;
  onAccountUpdated?: (updated: Account) => void;
  onDeleteClick: () => void;
}

export function AccountHeaderInfo({
  account,
  token,
  badgeClassName,
  onAccountUpdated,
  onDeleteClick,
}: AccountHeaderInfoProps) {
  const [editingField, setEditingField] = useState<EditableField | null>(null);
  const [draftValue, setDraftValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [editError, setEditError] = useState('');

  const inputRef = useRef<HTMLInputElement>(null);
  const cancellingRef = useRef(false);
  const committingRef = useRef(false);

  useEffect(() => {
    if (editingField && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editingField]);

  const startEditing = (field: EditableField) => {
    if (saving) return;
    cancellingRef.current = false;
    committingRef.current = false;
    setEditError('');
    setDraftValue(field === 'name' ? account.name : account.description);
    setEditingField(field);
  };

  const commitEdit = async (fromKeyDown = false) => {
    if (!editingField || committingRef.current || cancellingRef.current) {
      return;
    }

    const field = editingField;
    const trimmed = draftValue.trim();
    const current = (field === 'name' ? account.name : account.description).trim();

    if (!trimmed) {
      setEditError(
        field === 'name' ? 'Account name cannot be empty.' : 'Account description cannot be empty.'
      );
      if (!fromKeyDown) {
        setEditingField(null);
      }
      return;
    }

    if (trimmed === current) {
      setEditingField(null);
      setEditError('');
      return;
    }

    committingRef.current = true;
    setSaving(true);
    setEditError('');

    try {
      const payload = field === 'name' ? { name: trimmed } : { description: trimmed };
      const updated = await patchAccount(token, account.id, payload);
      setEditingField(null);
      onAccountUpdated?.(updated);
    } catch (err: unknown) {
      setEditError(err instanceof Error ? err.message : 'Failed to update account.');
      setEditingField(null);
    } finally {
      setSaving(false);
      committingRef.current = false;
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      void commitEdit(true);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancellingRef.current = true;
      setEditError('');
      setEditingField(null);
    }
  };

  const handleDownloadClick = async () => {
    if (downloading) return;
    setEditError('');
    setDownloading(true);
    try {
      await downloadAccountCsv(token, account);
    } catch (err: unknown) {
      setEditError(err instanceof Error ? err.message : 'Failed to export account data.');
    } finally {
      setDownloading(false);
    }
  };

  const resolvedBadgeClass =
    badgeClassName || `badge-${(account.type || '').toLowerCase().replace(/\s+/g, '-')}`;

  return (
    <div className="account-detail-info">
      <div className="account-title-row">
        {editingField === 'name' ? (
          <input
            ref={inputRef}
            type="text"
            className="account-inline-input account-inline-name-input"
            aria-label="Edit account name"
            maxLength={100}
            value={draftValue}
            disabled={saving}
            onChange={(e) => setDraftValue(e.target.value)}
            onBlur={() => void commitEdit(false)}
            onKeyDown={handleKeyDown}
          />
        ) : (
          <h2
            className="account-editable-text"
            title="Double-click to edit account name"
            onDoubleClick={() => startEditing('name')}
          >
            {account.name}
          </h2>
        )}
        <span className={`badge-account-type ${resolvedBadgeClass}`}>{account.type}</span>
        <button
          type="button"
          className="btn-secondary btn-sm ml-2"
          onClick={() => void handleDownloadClick()}
          disabled={downloading}
          title="Download all entered data for this account as CSV"
          aria-label="Download Data"
        >
          {downloading ? 'Exporting...' : 'Download Data'}
        </button>
        <button
          type="button"
          className="btn-danger-outline btn-sm ml-2"
          onClick={onDeleteClick}
          title="Delete this account"
          aria-label="Delete this account"
        >
          Delete Account
        </button>
      </div>

      {editingField === 'description' ? (
        <input
          ref={inputRef}
          type="text"
          className="account-inline-input account-inline-desc-input"
          aria-label="Edit account description"
          maxLength={255}
          value={draftValue}
          disabled={saving}
          onChange={(e) => setDraftValue(e.target.value)}
          onBlur={() => void commitEdit(false)}
          onKeyDown={handleKeyDown}
        />
      ) : (
        <p
          className="account-desc-text account-editable-text"
          title="Double-click to edit account description"
          onDoubleClick={() => startEditing('description')}
        >
          {account.description}
        </p>
      )}

      {editingField !== null && hasAccountDigitsWarning(draftValue) && (
        <div className="warning-banner mt-2" role="alert">
          <span className="warning-icon" aria-hidden="true">⚠️</span>
          <span>{ACCOUNT_DIGITS_WARNING_TEXT}</span>
        </div>
      )}

      {editError && (
        <p className="account-inline-error" role="alert">
          {editError}
        </p>
      )}
    </div>
  );
}
