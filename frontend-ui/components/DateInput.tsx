'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  isoToDisplayDate,
  displayDateToIso,
  applyDateAutoSlash,
  computeSlashKeyAction,
} from '../lib/dateInput';

export { isoToDisplayDate, displayDateToIso };

interface DateInputProps {
  id: string;
  value: string; // ISO format "YYYY-MM-DD" or ""
  onChange: (val: string) => void;
  required?: boolean;
  nextFieldId?: string;
  disabled?: boolean;
  defaultToday?: boolean;
}

export function DateInput({
  id,
  value,
  onChange,
  required = false,
  nextFieldId,
  disabled = false,
  defaultToday = false,
}: DateInputProps) {
  const [displayValue, setDisplayValue] = useState(() => isoToDisplayDate(value));
  const nativeDateRef = useRef<HTMLInputElement>(null);

  // Sync with external value prop changes (e.g. form reset, selecting record, or calendar picker)
  // without clobbering in-progress edits like "11/1/2026" before typing the second digit.
  useEffect(() => {
    if (displayDateToIso(displayValue) !== value) {
      setDisplayValue(isoToDisplayDate(value));
    }
  }, [value]);

  const focusNextField = () => {
    if (!nextFieldId) return;
    const nextEl = document.getElementById(nextFieldId);
    if (nextEl) {
      nextEl.focus();
      if ('select' in nextEl && typeof nextEl.select === 'function') {
        nextEl.select();
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== '/') return;
    const input = e.currentTarget;
    const val = input.value;
    const start = input.selectionStart ?? val.length;
    const end = input.selectionEnd ?? val.length;
    const result = computeSlashKeyAction(val, start, end);

    e.preventDefault();
    if (result.action === 'ignore') return;
    if (result.action === 'focusNext') {
      focusNextField();
      return;
    }

    setDisplayValue(result.value);
    const iso = displayDateToIso(result.value);
    if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
      onChange(iso);
    }
    setTimeout(() => {
      input.setSelectionRange(result.cursorPos, result.cursorPos);
    }, 0);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = applyDateAutoSlash(e.target.value, displayValue);
    if (val.length > 10) return;

    setDisplayValue(val);
    const iso = displayDateToIso(val);
    if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
      onChange(iso);
    } else if (!val.trim()) {
      onChange('');
    }
  };

  const handleBlur = () => {
    if (!displayValue.trim()) {
      onChange('');
      return;
    }
    const iso = displayDateToIso(displayValue);
    if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
      setDisplayValue(isoToDisplayDate(iso));
      onChange(iso);
    } else if (required && defaultToday) {
      const today = new Date().toISOString().slice(0, 10);
      setDisplayValue(isoToDisplayDate(today));
      onChange(today);
    }
  };

  const openCalendar = () => {
    try {
      nativeDateRef.current?.showPicker?.();
    } catch {
      nativeDateRef.current?.focus();
    }
  };

  const handleNativeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const iso = e.target.value;
    if (iso) {
      setDisplayValue(isoToDisplayDate(iso));
      onChange(iso);
    }
  };

  return (
    <div className="date-input-wrapper">
      <input
        id={id}
        type="text"
        inputMode="numeric"
        placeholder="MM/DD/YYYY"
        value={displayValue}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onBlur={handleBlur}
        onFocus={(e) => e.target.select()}
        className="date-text-input"
        required={required}
        disabled={disabled}
        autoComplete="off"
      />
      <button
        type="button"
        className="date-calendar-btn"
        onClick={openCalendar}
        onMouseDown={(e) => e.preventDefault()}
        tabIndex={-1}
        title="Open calendar"
        aria-label="Open calendar"
      >
        <svg
          width="15"
          height="15"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
        </svg>
      </button>
      <input
        ref={nativeDateRef}
        type="date"
        tabIndex={-1}
        className="hidden-native-date"
        value={displayDateToIso(displayValue)}
        onChange={handleNativeChange}
        aria-hidden="true"
      />
    </div>
  );
}
