'use client';

import React, { useState, useEffect, useRef } from 'react';

interface DateInputProps {
  id: string;
  value: string; // ISO format "YYYY-MM-DD" or ""
  onChange: (val: string) => void;
  required?: boolean;
  nextFieldId?: string;
  disabled?: boolean;
  defaultToday?: boolean;
}

export function isoToDisplayDate(isoStr: string): string {
  if (!isoStr) return '';
  const clean = isoStr.trim();
  if (/^\d{4}-\d{2}-\d{2}$/.test(clean)) {
    const [y, m, d] = clean.split('-');
    return `${m}/${d}/${y}`;
  }
  return clean;
}

export function displayDateToIso(displayStr: string): string {
  const clean = displayStr.trim();
  if (!clean) return '';

  if (/^\d{4}-\d{2}-\d{2}$/.test(clean)) {
    return clean;
  }

  const parts = clean.split('/');
  if (parts.length === 3) {
    let [m, d, y] = parts;
    m = m.padStart(2, '0');
    d = d.padStart(2, '0');
    if (y.length === 2) {
      y = `20${y}`;
    }
    const numM = parseInt(m, 10);
    const numD = parseInt(d, 10);
    if (
      y.length === 4 &&
      numM >= 1 &&
      numM <= 12 &&
      numD >= 1 &&
      numD <= 31
    ) {
      return `${y}-${m}-${d}`;
    }
  }

  // Handle 8 digits MMDDYYYY e.g. 10152026
  if (/^\d{8}$/.test(clean)) {
    const m = clean.slice(0, 2);
    const d = clean.slice(2, 4);
    const y = clean.slice(4, 8);
    return `${y}-${m}-${d}`;
  }

  return clean;
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

  // Sync with value prop changes (e.g. form reset or selecting record to edit)
  useEffect(() => {
    setDisplayValue(isoToDisplayDate(value));
  }, [value]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === '/') {
      const input = e.currentTarget;
      const val = input.value;
      const start = input.selectionStart ?? val.length;
      const end = input.selectionEnd ?? val.length;

      const textBefore = val.slice(0, start);
      const textAfter = val.slice(end);

      const parts = textBefore.split('/');
      const currentPartIndex = parts.length - 1;
      const currentPart = parts[currentPartIndex];

      if (currentPartIndex === 0) {
        // Month segment: assume 01 if "1", pad single digits
        e.preventDefault();
        let m = currentPart.trim();
        if (!m || m === '0' || m === '00') {
          m = '01';
        } else {
          m = m.padStart(2, '0');
        }
        const num = parseInt(m, 10);
        if (num > 12) m = '12';
        if (num < 1) m = '01';

        const newVal = `${m}/${textAfter.replace(/^\//, '')}`;
        setDisplayValue(newVal);
        const iso = displayDateToIso(newVal);
        if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
          onChange(iso);
        }
        setTimeout(() => {
          const nextPos = m.length + 1;
          input.setSelectionRange(nextPos, nextPos);
        }, 0);
      } else if (currentPartIndex === 1) {
        // Day segment: pad single digits (e.g. 1 -> 01, 5 -> 05)
        e.preventDefault();
        let d = currentPart.trim();
        if (!d || d === '0' || d === '00') {
          d = '01';
        } else {
          d = d.padStart(2, '0');
        }
        const num = parseInt(d, 10);
        if (num > 31) d = '31';
        if (num < 1) d = '01';

        const monthPart = parts[0];
        const newVal = `${monthPart}/${d}/${textAfter.replace(/^\//, '')}`;
        setDisplayValue(newVal);
        const iso = displayDateToIso(newVal);
        if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
          onChange(iso);
        }
        setTimeout(() => {
          const nextPos = monthPart.length + 1 + d.length + 1;
          input.setSelectionRange(nextPos, nextPos);
        }, 0);
      } else if (currentPartIndex >= 2) {
        // Year segment: pressing '/' moves to nextFieldId
        e.preventDefault();
        if (nextFieldId) {
          const nextEl = document.getElementById(nextFieldId);
          if (nextEl) {
            nextEl.focus();
            if ('select' in nextEl && typeof nextEl.select === 'function') {
              nextEl.select();
            }
          }
        }
      }
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let val = e.target.value.replace(/[^\d/]/g, '');

    // Auto-advance with slash if user entered 2 digits without slash
    if (/^\d{2}$/.test(val) && !displayValue.endsWith('/') && val.length > displayValue.length) {
      val = `${val}/`;
    } else if (/^\d{2}\/\d{2}$/.test(val) && !displayValue.endsWith('/') && val.length > displayValue.length) {
      val = `${val}/`;
    }

    if (val.length <= 10) {
      setDisplayValue(val);
      const iso = displayDateToIso(val);
      if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
        onChange(iso);
      } else if (!val.trim()) {
        onChange('');
      }
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
