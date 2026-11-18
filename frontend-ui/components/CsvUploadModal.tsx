'use client';

import React, { useState, useRef, useEffect, useMemo } from 'react';
import {
  parseAndValidateCsv,
  ParsedCsvResult,
  ParsedCsvRow,
  MAX_BULK_UPLOAD_RECORDS,
  MIN_RECORD_YEAR,
  detectRowCollisions,
} from '../lib/csvParser';
import {
  createAccountRecord,
  fetchAccountRecords,
  fetchCategories,
  createCategory,
} from '../lib/api';
import {
  resolveAndCreateMissingCategories,
  normalizeCategoryKey,
} from '../lib/accountCategories';
import { AccountRecord, Category } from '../lib/types';

interface CsvUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  accountId: string;
  accountName: string;
  accountType: string;
  existingRecords?: AccountRecord[];
  existingCategories?: Category[];
  onSuccess: () => void;
}

type ModalStage = 'select' | 'review' | 'importing' | 'complete';

async function uploadRecordRow(
  token: string,
  accountId: string,
  row: ParsedCsvRow,
  effectiveAmount: number,
  categoryId: string | null
): Promise<void> {
  await createAccountRecord(token, accountId, {
    value: effectiveAmount,
    date_sent: row.dateSent,
    date_cleared: row.dateCleared,
    category_id: categoryId,
    entity: row.recipient,
    description: row.description || undefined,
  });
}

export default function CsvUploadModal({
  isOpen,
  onClose,
  token,
  accountId,
  accountName,
  accountType,
  existingRecords,
  existingCategories,
  onSuccess,
}: CsvUploadModalProps) {
  const [stage, setStage] = useState<ModalStage>('select');
  const [fileName, setFileName] = useState<string>('');
  const [parsedResult, setParsedResult] = useState<ParsedCsvResult | null>(null);
  const [selectedRows, setSelectedRows] = useState<Record<number, boolean>>({});
  const [signsInverted, setSignsInverted] = useState<boolean>(false);
  const [parseError, setParseError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState<boolean>(false);
  const [accountRecords, setAccountRecords] = useState<AccountRecord[]>(existingRecords || []);
  const [categories, setCategories] = useState<Category[]>(existingCategories || []);

  // Import progress
  const [progress, setProgress] = useState<{ current: number; total: number }>({ current: 0, total: 0 });
  const [importSummary, setImportSummary] = useState<{ success: number; failed: number }>({ success: 0, failed: 0 });
  const [failedRows, setFailedRows] = useState<{ row: number; reason: string }[]>([]);

  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (existingRecords) {
      setAccountRecords(existingRecords);
    }
  }, [existingRecords]);

  useEffect(() => {
    if (existingCategories) {
      setCategories(existingCategories);
    }
  }, [existingCategories]);

  useEffect(() => {
    if (isOpen && token && accountId) {
      fetchAccountRecords(token, accountId)
        .then((recs) => setAccountRecords(recs))
        .catch(() => {});
      fetchCategories(token)
        .then((cats) => setCategories(cats))
        .catch(() => {});
    }
  }, [isOpen, accountId]);

  const collisionRowNumbers = useMemo(() => {
    if (!parsedResult) return new Set<number>();
    return detectRowCollisions(parsedResult.rows, accountRecords, signsInverted);
  }, [parsedResult, accountRecords, signsInverted]);

  if (!isOpen) return null;

  const resetState = () => {
    setStage('select');
    setFileName('');
    setParsedResult(null);
    setSelectedRows({});
    setSignsInverted(false);
    setParseError(null);
    setProgress({ current: 0, total: 0 });
    setImportSummary({ success: 0, failed: 0 });
    setFailedRows([]);
  };

  const handleClose = () => {
    if (stage === 'importing') return; // Prevent closing mid-import
    resetState();
    onClose();
  };

  const handleFileProcess = async (file: File) => {
    setParseError(null);
    setFileName(file.name);

    try {
      const text = await file.text();
      const res = parseAndValidateCsv(text);

      if (!res.success) {
        setParseError(res.error || 'Failed to parse CSV file');
        return;
      }

      setParsedResult(res);

      const collisions = detectRowCollisions(res.rows, accountRecords, signsInverted);
      const initialSelected: Record<number, boolean> = {};
      res.rows.forEach((r) => {
        if (r.isValid && !collisions.has(r.rowNumber)) {
          initialSelected[r.rowNumber] = true;
        }
      });
      setSelectedRows(initialSelected);
      setStage('review');
    } catch (err: unknown) {
      setParseError(err instanceof Error ? err.message : 'Error reading file');
    }
  };

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFileProcess(file);
    }
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    const file = e.dataTransfer.files?.[0];
    if (file) {
      handleFileProcess(file);
    }
  };

  const toggleRowSelection = (rowNumber: number) => {
    setSelectedRows((prev) => ({
      ...prev,
      [rowNumber]: !prev[rowNumber],
    }));
  };

  const toggleSelectAll = () => {
    if (!parsedResult) return;
    const selectable = parsedResult.rows.filter(
      (r) => r.isValid && !collisionRowNumbers.has(r.rowNumber)
    );
    const allSelected = selectable.length > 0 && selectable.every((r) => selectedRows[r.rowNumber]);

    const updated: Record<number, boolean> = { ...selectedRows };
    selectable.forEach((r) => {
      updated[r.rowNumber] = !allSelected;
    });
    setSelectedRows(updated);
  };

  const handleToggleSigns = () => {
    const nextInverted = !signsInverted;
    setSignsInverted(nextInverted);
    if (parsedResult) {
      const nextCollisions = detectRowCollisions(parsedResult.rows, accountRecords, nextInverted);
      const updated: Record<number, boolean> = {};
      parsedResult.rows.forEach((r) => {
        if (r.isValid && !nextCollisions.has(r.rowNumber)) {
          updated[r.rowNumber] = selectedRows[r.rowNumber] ?? true;
        }
      });
      setSelectedRows(updated);
    }
  };

  const getEffectiveAmount = (row: ParsedCsvRow): number => {
    return signsInverted ? -row.amount : row.amount;
  };

  const formatCurrency = (val: number): string => {
    const absVal = Math.abs(val).toFixed(2);
    const formatted = `$${Number(absVal).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
    return val < 0 ? `-${formatted}` : formatted;
  };

  const handleStartImport = async () => {
    if (!parsedResult) return;

    const rowsToImport = parsedResult.rows.filter(
      (r) => r.isValid && !collisionRowNumbers.has(r.rowNumber) && selectedRows[r.rowNumber]
    );

    if (rowsToImport.length === 0 || rowsToImport.length > MAX_BULK_UPLOAD_RECORDS) return;

    setStage('importing');
    setProgress({ current: 0, total: rowsToImport.length });

    let successCount = 0;
    let failCount = 0;
    const fails: { row: number; reason: string }[] = [];

    let categoryIdMap = new Map<string, string>();
    try {
      const latestCats = await fetchCategories(token).catch(() => categories);
      const rawNames = rowsToImport.map((r) => r.category).filter((c) => Boolean(c.trim()));
      const resolved = await resolveAndCreateMissingCategories(
        rawNames,
        latestCats,
        (name) => createCategory(token, name)
      );
      categoryIdMap = resolved.idByLowerName;
      if (resolved.created.length > 0) {
        setCategories([...latestCats, ...resolved.created]);
      }
    } catch (err: unknown) {
      failCount = rowsToImport.length;
      fails.push({
        row: 0,
        reason: err instanceof Error ? err.message : 'Failed to create categories from CSV',
      });
      setImportSummary({ success: 0, failed: failCount });
      setFailedRows(fails);
      setStage('complete');
      return;
    }

    for (let i = 0; i < rowsToImport.length; i++) {
      const row = rowsToImport[i];
      setProgress({ current: i + 1, total: rowsToImport.length });
      const catKey = normalizeCategoryKey(row.category);
      const categoryId = catKey ? (categoryIdMap.get(catKey) ?? null) : null;

      try {
        await uploadRecordRow(token, accountId, row, getEffectiveAmount(row), categoryId);
        successCount++;
      } catch (err: unknown) {
        failCount++;
        fails.push({
          row: row.rowNumber,
          reason: err instanceof Error ? err.message : 'Unknown network or API error',
        });
      }
    }

    setImportSummary({ success: successCount, failed: failCount });
    setFailedRows(fails);
    setStage('complete');
    fetchAccountRecords(token, accountId)
      .then((recs) => setAccountRecords(recs))
      .catch(() => {});
    onSuccess();
  };

  const selectedCount = parsedResult
    ? parsedResult.rows.filter(
        (r) => r.isValid && !collisionRowNumbers.has(r.rowNumber) && selectedRows[r.rowNumber]
      ).length
    : 0;

  return (
    <div className="modal-backdrop">
      <div className="modal-content csv-upload-modal">
        {/* Header */}
        <div className="modal-header">
          <div>
            <h2 className="modal-title">Bulk Import CSV Records</h2>
            <p className="modal-subtitle">
              Target Account: <strong>{accountName}</strong> ({accountType})
            </p>
          </div>
          {stage !== 'importing' && (
            <button
              type="button"
              className="btn-close"
              onClick={handleClose}
              aria-label="Close"
            >
              &times;
            </button>
          )}
        </div>

        {/* Stage 1: File Selection */}
        {stage === 'select' && (
          <div className="csv-select-stage">
            <div
              className={`csv-drop-zone ${dragActive ? 'active' : ''}`}
              onDragEnter={handleDrag}
              onDragOver={handleDrag}
              onDragLeave={handleDrag}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
            >
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,text/csv"
                style={{ display: 'none' }}
                onChange={handleFileInputChange}
              />
              <div className="drop-zone-icon">&#128196;</div>
              <p className="drop-zone-text">
                <strong>Click to choose a CSV file</strong> or drag and drop here
              </p>
              <p className="drop-zone-subtext">
                File is inspected entirely in your browser before submission
              </p>
            </div>

            {parseError && <div className="alert-error mt-3">{parseError}</div>}

            <div className="csv-requirements-card">
              <h4>Required CSV Header Format</h4>
              <ul>
                <li>
                  <strong>Date Sent</strong> (header can be: <em>Date Sent, Date, Sent, Spent On</em>)
                </li>
                <li>
                  <strong>Recipient</strong> (header can be: <em>Recipient, Entity, Payee, Merchant, Vendor, To</em>)
                </li>
                <li>
                  <strong>Amount</strong> (header can be: <em>Amount, Value, Total, Sum</em>)
                </li>
                <li>
                  <strong>Date Cleared</strong> (optional: <em>Date Cleared, Cleared, Cleared On, Reconciled</em>)
                </li>
                <li>
                  <strong>Description</strong> (optional: <em>Description, Notes, Details</em>)
                </li>
                <li>
                  <strong>Category</strong> (optional: <em>Category, Categories, Expense Category, Transaction Category</em> — new non-standard categories are created automatically)
                </li>
              </ul>
              <p className="text-muted text-xs mt-2">
                Supported date formats: <code>MM/dd/yy</code>, <code>mm/dd/yyyy</code>, or <code>yyyy-MM-dd</code>. Financial records older than the year {MIN_RECORD_YEAR} are not permitted.
              </p>
              <p className="text-muted text-xs mt-1">
                Upload limit: Maximum {MAX_BULK_UPLOAD_RECORDS.toLocaleString()} records per bulk upload.
              </p>
            </div>
          </div>
        )}

        {/* Stage 2: Review & Approval */}
        {stage === 'review' && parsedResult && (
          <div className="csv-review-stage">
            {parsedResult.emptyCount > 0 && (
              <div className="alert-warning mb-3">
                &#9888; <strong>{parsedResult.emptyCount} empty record{parsedResult.emptyCount === 1 ? '' : 's'} found</strong> (discarded from review and import).
              </div>
            )}

            <div className="csv-review-toolbar">
              <div className="review-stats">
                <span className="stat-pill total">
                  Total Rows: <strong>{parsedResult.rows.length}</strong>
                </span>
                <span className="stat-pill valid">
                  Valid: <strong>{parsedResult.validCount}</strong>
                </span>
                {collisionRowNumbers.size > 0 && (
                  <span
                    className="stat-pill collision"
                    title={`${collisionRowNumbers.size} duplicate record(s) detected and excluded`}
                  >
                    Duplicates Excluded: <strong>{collisionRowNumbers.size}</strong>
                  </span>
                )}
                {parsedResult.invalidCount > 0 && (
                  <span className="stat-pill invalid" title={`${parsedResult.invalidCount} invalid record(s) sorted to top`}>
                    Invalid (at top): <strong>{parsedResult.invalidCount}</strong>
                  </span>
                )}
                {parsedResult.emptyCount > 0 && (
                  <span className="stat-pill empty-warning" title={`${parsedResult.emptyCount} empty record(s) discarded`}>
                    Empty: <strong>{parsedResult.emptyCount}</strong>
                  </span>
                )}
                <span className="stat-pill selected">
                  Selected to Import: <strong>{selectedCount}</strong>
                </span>
              </div>

              <div className="review-actions">
                <button
                  type="button"
                  className="btn-secondary btn-sm"
                  onClick={handleToggleSigns}
                  title="Toggle positive/negative across all amounts"
                >
                  {signsInverted ? 'Reset Signs' : 'Invert Signs (+/-)'}
                </button>
                <button
                  type="button"
                  className="btn-secondary btn-sm"
                  onClick={toggleSelectAll}
                >
                  Toggle All
                </button>
              </div>
            </div>

            {parsedResult.invalidCount > 0 && (
              <div className="notice-banner banner-invalid mb-2">
                &#9888; <strong>{parsedResult.invalidCount} broken record{parsedResult.invalidCount === 1 ? '' : 's'}</strong> moved to the top of the table for your review. Broken records cannot be imported until corrected in the CSV.
              </div>
            )}

            {collisionRowNumbers.size > 0 && (
              <div className="notice-banner banner-collision mb-2">
                &#9888; <strong>{collisionRowNumbers.size} duplicate record{collisionRowNumbers.size === 1 ? '' : 's'}</strong> detected in this account and excluded from import.
              </div>
            )}

            {signsInverted && (
              <div className="notice-banner mb-2">
                Amounts are currently inverted: Positive values are now expenses (-) and negative values are income (+).
              </div>
            )}

            {/* Scrollable Preview Table */}
            <div className="csv-table-wrapper">
              <table className="csv-preview-table">
                <thead>
                  <tr>
                    <th style={{ width: '40px' }}></th>
                    <th style={{ width: '50px' }}>#</th>
                    <th>Date Sent</th>
                    <th>Date Cleared</th>
                    <th>Recipient</th>
                    <th>Category</th>
                    <th>Description</th>
                    <th style={{ textAlign: 'right' }}>Amount</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {parsedResult.rows.map((row) => {
                    const isCollision = collisionRowNumbers.has(row.rowNumber);
                    const isSelectable = row.isValid && !isCollision;
                    const isSelected = isSelectable && !!selectedRows[row.rowNumber];
                    const effectiveAmount = getEffectiveAmount(row);
                    const hasDateSentError = row.errors.some((e) =>
                      e.toLowerCase().includes('date sent')
                    );
                    const hasDateClearedError = row.errors.some((e) =>
                      e.toLowerCase().includes('date cleared')
                    );
                    const hasRecipientError = row.errors.some((e) =>
                      e.toLowerCase().includes('recipient')
                    );
                    const hasCategoryError = row.errors.some((e) =>
                      e.toLowerCase().includes('category')
                    );
                    const hasDescriptionError = row.errors.some((e) =>
                      e.toLowerCase().includes('description')
                    );
                    const hasAmountError = row.errors.some((e) =>
                      e.toLowerCase().includes('amount')
                    );

                    return (
                      <tr
                        key={row.rowNumber}
                        className={`${!row.isValid ? 'row-invalid' : ''} ${isCollision ? 'row-collision' : ''} ${isSelected ? 'row-selected' : ''}`}
                      >
                        <td>
                          <input
                            type="checkbox"
                            checked={isSelected}
                            disabled={!isSelectable}
                            onChange={() => toggleRowSelection(row.rowNumber)}
                          />
                        </td>
                        <td className="text-muted">{row.rowNumber}</td>
                        <td className={hasDateSentError ? 'cell-invalid' : ''}>
                          {row.dateSent || <em className="text-danger">Missing</em>}
                        </td>
                        <td className={hasDateClearedError ? 'cell-invalid' : ''}>
                          {row.dateCleared ? (
                            row.dateCleared
                          ) : (
                            <span className="badge-pending">Pending</span>
                          )}
                        </td>
                        <td
                          className={`font-semibold ${hasRecipientError ? 'cell-invalid' : ''}`}
                        >
                          {row.recipient || <em className="text-danger">Missing</em>}
                        </td>
                        <td className={hasCategoryError ? 'cell-invalid' : ''}>
                          {row.category ? (
                            <span className="category-pill">{row.category}</span>
                          ) : (
                            <span className="text-muted">Uncategorized</span>
                          )}
                        </td>
                        <td className={`text-muted ${hasDescriptionError ? 'cell-invalid' : ''}`}>
                          {row.description || '—'}
                        </td>
                        <td
                          className={hasAmountError ? 'cell-invalid' : ''}
                          style={{
                            textAlign: 'right',
                            fontWeight: 600,
                            color: effectiveAmount < 0 ? 'var(--credit-card)' : 'var(--savings)',
                          }}
                        >
                          {formatCurrency(effectiveAmount)}
                        </td>
                        <td>
                          {isCollision ? (
                            <span
                              className="badge-status-collision"
                              title="Exact date, vendor, and amount match found in this account"
                            >
                              &#9888; Duplicate Excluded
                            </span>
                          ) : row.isValid ? (
                            <span className="badge-status-valid">&#10003; Valid</span>
                          ) : (
                            <div className="invalid-errors-container">
                              {row.errors.map((err, errIdx) => (
                                <span
                                  key={errIdx}
                                  className="badge-status-invalid"
                                  title={err}
                                >
                                  &#9888; {err}
                                </span>
                              ))}
                            </div>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            <div className="modal-footer">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setStage('select')}
              >
                Choose Another File
              </button>
              <button
                type="button"
                className="btn-primary"
                disabled={selectedCount === 0 || selectedCount > MAX_BULK_UPLOAD_RECORDS}
                onClick={handleStartImport}
              >
                Import {selectedCount} Record{selectedCount === 1 ? '' : 's'}
              </button>
            </div>
          </div>
        )}

        {/* Stage 3: Importing in progress */}
        {stage === 'importing' && (
          <div className="csv-progress-stage">
            <h3 className="progress-title">Importing Records to Planned Wallet...</h3>
            <p className="progress-subtitle">
              Delivering record {progress.current} of {progress.total} via API
            </p>

            <div className="progress-bar-container">
              <div
                className="progress-bar-fill"
                style={{
                  width: `${progress.total > 0 ? (progress.current / progress.total) * 100 : 0}%`,
                }}
              />
            </div>
            <p className="text-muted text-sm mt-3">
              Please keep this window open while records are being uploaded.
            </p>
          </div>
        )}

        {/* Stage 4: Complete */}
        {stage === 'complete' && (
          <div className="csv-complete-stage">
            <div className="complete-icon">&#10003;</div>
            <h3 className="complete-title">Import Complete</h3>
            <p className="complete-stats">
              Successfully imported <strong>{importSummary.success}</strong> records.
              {importSummary.failed > 0 && (
                <span className="text-danger ml-2">
                  ({importSummary.failed} failed)
                </span>
              )}
            </p>

            {failedRows.length > 0 && (
              <div className="failed-rows-report mt-3">
                <h4>Failed Records</h4>
                <ul>
                  {failedRows.map((f, i) => (
                    <li key={i}>
                      Row {f.row}: {f.reason}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="modal-footer mt-4">
              <button
                type="button"
                className="btn-primary"
                onClick={handleClose}
              >
                Done
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
