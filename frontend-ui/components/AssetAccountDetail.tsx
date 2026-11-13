'use client';

import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Account,
  AccountAsset,
  Asset,
  AssetAccountSummary,
  AssetAccountTrendPoint,
  AssetValue,
} from '../lib/types';
import { DateInput, displayDateToIso, isoToDisplayDate } from './DateInput';
import { AssetTrendGraph } from './AssetTrendGraph';
import {
  fetchAccountAssets,
  fetchAssetAccountSummary,
  fetchAssetAccountTrend,
  fetchAssets,
  createAsset,
  updateAsset,
  fetchAssetValues,
  createAssetValue,
  updateAssetValue,
  deleteAssetValue,
  addAccountAsset,
  updateAccountAsset,
  deleteAccountAsset,
} from '../lib/api';

interface AssetAccountDetailProps {
  account: Account;
  token: string;
}

function getTodayDateString(): string {
  const d = new Date();
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const abs = Math.abs(val);
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(abs);
  return isNegative ? `-${formatted}` : formatted;
}

function formatUnitValue(val: number): string {
  if (val === 0) return '$0.00';
  if (val >= 1) {
    return formatCurrency(val);
  }
  return `$${val.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })}`;
}

function formatQuantity(qty: number): string {
  return Number.isInteger(qty) ? qty.toString() : qty.toLocaleString('en-US', { maximumFractionDigits: 8 });
}

function parseAmount(amtStr: string): number {
  if (!amtStr) return NaN;
  const clean = amtStr.replace(/[$€£,]/g, '').trim();
  if (!/^-?\d*(\.\d+)?$/.test(clean) || clean === '' || clean === '-' || clean === '.') {
    return NaN;
  }
  return parseFloat(clean);
}

export function AssetAccountDetail({ account, token }: AssetAccountDetailProps) {
  const [holdings, setHoldings] = useState<AccountAsset[]>([]);
  const [summary, setSummary] = useState<AssetAccountSummary | null>(null);
  const [trendPoints, setTrendPoints] = useState<AssetAccountTrendPoint[]>([]);
  const [allAssets, setAllAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  const [activeTab, setActiveTab] = useState<'holdings' | 'trend' | 'settings' | 'definitions'>('holdings');

  // Dynamic column hiding for Current Holdings table when content does not fit
  const [hideDescription, setHideDescription] = useState(false);
  const [hideActions, setHideActions] = useState(false);
  const holdingsContainerRef = useRef<HTMLDivElement>(null);
  const holdingsTableRef = useRef<HTMLTableElement>(null);

  const hideDescRef = useRef(false);
  const hideActionsRef = useRef(false);
  const descThresholdRef = useRef<number>(940);
  const actionsThresholdRef = useRef<number>(720);

  // Add Holding form state
  const [selectedAssetIdForAdd, setSelectedAssetIdForAdd] = useState('');
  const [addQuantityStr, setAddQuantityStr] = useState('');
  const [addHoldingDate, setAddHoldingDate] = useState(getTodayDateString());
  const [addingHolding, setAddingHolding] = useState(false);
  const [addHoldingError, setAddHoldingError] = useState('');

  // Edit Holding Quantity modal / state
  const [editingHolding, setEditingHolding] = useState<AccountAsset | null>(null);
  const [editQuantityStr, setEditQuantityStr] = useState('');
  const [editHoldingDate, setEditHoldingDate] = useState(getTodayDateString());
  const [updatingQuantity, setUpdatingQuantity] = useState(false);
  const [editQuantityError, setEditQuantityError] = useState('');

  // Quick Valuation modal / state
  const [valuingAsset, setValuingAsset] = useState<{ id: string; label: string; description: string } | null>(null);
  const [valuationValueStr, setValuationValueStr] = useState('');
  const [valuationDate, setValuationDate] = useState(getTodayDateString());
  const [savingValuation, setSavingValuation] = useState(false);
  const [valuationError, setValuationError] = useState('');

  // Asset Definitions tab state
  const [newAssetLabel, setNewAssetLabel] = useState('');
  const [newAssetDesc, setNewAssetDesc] = useState('');
  const [newAssetInitialVal, setNewAssetInitialVal] = useState('');
  const [newAssetInitialDate, setNewAssetInitialDate] = useState(getTodayDateString());
  const [creatingAsset, setCreatingAsset] = useState(false);
  const [createAssetError, setCreateAssetError] = useState('');

  // Asset Valuations Manager state (Tab 3)
  const [selectedAssetForHistory, setSelectedAssetForHistory] = useState<string>('');
  const [assetValuationHistory, setAssetValuationHistory] = useState<AssetValue[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [historyValueStr, setHistoryValueStr] = useState('');
  const [historyDateStr, setHistoryDateStr] = useState(getTodayDateString());
  const [submittingHistoryVal, setSubmittingHistoryVal] = useState(false);
  const [historyError, setHistoryError] = useState('');
  const [editingHistoryValId, setEditingHistoryValId] = useState<string | null>(null);

  // Edit Asset Definition modal / state
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null);
  const [editAssetLabel, setEditAssetLabel] = useState('');
  const [editAssetDesc, setEditAssetDesc] = useState('');
  const [updatingAsset, setUpdatingAsset] = useState(false);
  const [editAssetError, setEditAssetError] = useState('');

  const editQtyInputRef = useRef<HTMLInputElement>(null);
  const valuationInputRef = useRef<HTMLInputElement>(null);
  const editAssetInputRef = useRef<HTMLInputElement>(null);

  // Load account assets, summary, catalog, and valuation trend
  const loadData = async () => {
    setLoading(true);
    setError('');
    try {
      const [holdingsData, summaryData, assetsData, trendData] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssets(token),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(holdingsData);
      setSummary(summaryData);
      setAllAssets(assetsData);
      setTrendPoints(trendData || []);

      if (assetsData.length > 0 && !selectedAssetIdForAdd) {
        setSelectedAssetIdForAdd(assetsData[0].id);
      }
      if (assetsData.length > 0 && !selectedAssetForHistory) {
        setSelectedAssetForHistory(assetsData[0].id);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load asset account details');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [account.id, token]);

  // Load valuation history when selected asset changes in Definitions tab
  const loadValuationHistory = async (assetId: string) => {
    if (!assetId) {
      setAssetValuationHistory([]);
      return;
    }
    setLoadingHistory(true);
    setHistoryError('');
    try {
      const values = await fetchAssetValues(token, assetId);
      // Sort newest to oldest
      const sorted = [...values].sort((a, b) => b.value_date.localeCompare(a.value_date));
      setAssetValuationHistory(sorted);
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to load valuations');
    } finally {
      setLoadingHistory(false);
    }
  };

  useEffect(() => {
    if (selectedAssetForHistory) {
      loadValuationHistory(selectedAssetForHistory);
      setEditingHistoryValId(null);
      setHistoryValueStr('');
      setHistoryDateStr(getTodayDateString());
    }
  }, [selectedAssetForHistory, token]);

  // Focus input when editing quantity modal opens
  useEffect(() => {
    if (editingHolding && editQtyInputRef.current) {
      editQtyInputRef.current.focus();
      editQtyInputRef.current.select();
    }
  }, [editingHolding]);

  // Focus input when quick valuation modal opens
  useEffect(() => {
    if (valuingAsset && valuationInputRef.current) {
      valuationInputRef.current.focus();
      valuationInputRef.current.select();
    }
  }, [valuingAsset]);

  // Focus input when editing asset definition modal opens
  useEffect(() => {
    if (editingAsset && editAssetInputRef.current) {
      editAssetInputRef.current.focus();
      editAssetInputRef.current.select();
    }
  }, [editingAsset]);

  // All catalog assets available to add
  const availableAssetsToAdd = useMemo(() => allAssets, [allAssets]);

  // Update selected asset for add when available list updates
  useEffect(() => {
    if (availableAssetsToAdd.length > 0) {
      const exists = availableAssetsToAdd.some((a) => a.id === selectedAssetIdForAdd);
      if (!exists) {
        setSelectedAssetIdForAdd(availableAssetsToAdd[0].id);
      }
    } else {
      setSelectedAssetIdForAdd('');
    }
  }, [availableAssetsToAdd]);

  const totalCalculatedWorth = useMemo(() => {
    if (summary && typeof summary.total_value === 'number') {
      return summary.total_value;
    }
    return holdings.reduce((sum, h) => sum + (h.total_value || 0), 0);
  }, [summary, holdings]);

  useEffect(() => {
    hideDescRef.current = hideDescription;
  }, [hideDescription]);

  useEffect(() => {
    hideActionsRef.current = hideActions;
  }, [hideActions]);

  // Dynamic auto-hiding of Description and Actions columns when holdings table does not fit
  useEffect(() => {
    if (activeTab !== 'holdings' || holdings.length === 0) return;

    const container = holdingsContainerRef.current;
    const table = holdingsTableRef.current;
    if (!container || !table) return;

    // Reset baseline thresholds whenever holdings data changes
    descThresholdRef.current = 940;
    actionsThresholdRef.current = 720;

    const checkFit = () => {
      const containerWidth = container.clientWidth;
      if (containerWidth === 0) return;

      const tableWidth = table.scrollWidth;

      // 1. If table overflows container, increase threshold to require more room
      if (tableWidth > containerWidth + 2) {
        if (!hideDescRef.current) {
          descThresholdRef.current = Math.max(descThresholdRef.current, tableWidth);
          hideDescRef.current = true;
          setHideDescription(true);

          if (tableWidth - containerWidth > 150 && !hideActionsRef.current) {
            actionsThresholdRef.current = Math.max(actionsThresholdRef.current, containerWidth + 40);
            hideActionsRef.current = true;
            setHideActions(true);
          }
          return;
        } else if (!hideActionsRef.current) {
          actionsThresholdRef.current = Math.max(actionsThresholdRef.current, tableWidth);
          hideActionsRef.current = true;
          setHideActions(true);
          return;
        }
      }

      // 2. Responsive checks: HIDE when too small, RETURN when large enough
      // Actions column (baseline 720px, returns at 740px)
      if (containerWidth < actionsThresholdRef.current) {
        if (!hideActionsRef.current) {
          hideActionsRef.current = true;
          setHideActions(true);
        }
      } else if (containerWidth >= actionsThresholdRef.current + 20) {
        if (hideActionsRef.current) {
          hideActionsRef.current = false;
          setHideActions(false);
        }
      }

      // Description column (baseline 940px, returns at 960px)
      if (containerWidth < descThresholdRef.current) {
        if (!hideDescRef.current) {
          hideDescRef.current = true;
          setHideDescription(true);
        }
      } else if (containerWidth >= descThresholdRef.current + 20) {
        if (hideDescRef.current && containerWidth >= actionsThresholdRef.current + 20) {
          hideDescRef.current = false;
          setHideDescription(false);
        }
      }
    };

    checkFit();

    let resizeObserver: ResizeObserver | null = null;
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        checkFit();
      });
      resizeObserver.observe(container);
    }

    window.addEventListener('resize', checkFit);
    return () => {
      if (resizeObserver) resizeObserver.disconnect();
      window.removeEventListener('resize', checkFit);
    };
  }, [activeTab, holdings]);

  // Handle Add Asset Holding to Account
  const handleAddHolding = async (e: React.FormEvent) => {
    e.preventDefault();
    setAddHoldingError('');

    if (!selectedAssetIdForAdd) {
      setAddHoldingError('Please select an asset to add.');
      return;
    }

    const qty = parseAmount(addQuantityStr);
    if (isNaN(qty) || qty <= 0) {
      setAddHoldingError('Please enter a positive numeric quantity (greater than 0).');
      return;
    }

    let dateToUse = addHoldingDate.trim();
    if (dateToUse && !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      dateToUse = displayDateToIso(dateToUse);
    }
    if (!dateToUse || !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      setAddHoldingError('Please enter a valid holding date (MM/DD/YYYY or YYYY-MM-DD).');
      return;
    }

    setAddingHolding(true);
    try {
      await addAccountAsset(token, account.id, {
        asset_id: selectedAssetIdForAdd,
        quantity: qty,
        asset_date: dateToUse,
      });
      const [updatedHoldings, updatedSummary, updatedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(updatedHoldings);
      if (updatedSummary) setSummary(updatedSummary);
      if (updatedTrend) setTrendPoints(updatedTrend);

      setAddQuantityStr('');
      setMessage(`Successfully added holding with quantity ${qty} on ${isoToDisplayDate(dateToUse)}.`);
      setTimeout(() => setMessage(''), 3000);
    } catch (err: unknown) {
      setAddHoldingError(err instanceof Error ? err.message : 'Failed to add asset to account');
    } finally {
      setAddingHolding(false);
    }
  };

  // Handle Update Holding Quantity and Date
  const handleUpdateQuantity = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingHolding) return;
    setEditQuantityError('');

    const qty = parseAmount(editQuantityStr);
    if (isNaN(qty) || qty <= 0) {
      setEditQuantityError('Please enter a positive numeric quantity (greater than 0).');
      return;
    }

    let dateToUse = editHoldingDate.trim();
    if (dateToUse && !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      dateToUse = displayDateToIso(dateToUse);
    }

    setUpdatingQuantity(true);
    try {
      await updateAccountAsset(token, account.id, editingHolding.id, {
        quantity: qty,
        asset_date: dateToUse || undefined,
      });
      const [updatedHoldings, updatedSummary, updatedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(updatedHoldings);
      if (updatedSummary) setSummary(updatedSummary);
      if (updatedTrend) setTrendPoints(updatedTrend);

      setEditingHolding(null);
      setMessage(`Updated ${editingHolding.label} holding.`);
      setTimeout(() => setMessage(''), 3000);
    } catch (err: unknown) {
      setEditQuantityError(err instanceof Error ? err.message : 'Failed to update asset holding');
    } finally {
      setUpdatingQuantity(false);
    }
  };

  // Handle Remove Holding from Account
  const handleRemoveHolding = async (holding: AccountAsset) => {
    if (!confirm(`Are you sure you want to remove ${holding.label} holding from this account?`)) {
      return;
    }
    try {
      await deleteAccountAsset(token, account.id, holding.id);
      const [updatedHoldings, updatedSummary, updatedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(updatedHoldings);
      if (updatedSummary) setSummary(updatedSummary);
      if (updatedTrend) setTrendPoints(updatedTrend);

      setMessage(`Removed ${holding.label} holding from this account.`);
      setTimeout(() => setMessage(''), 3000);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to remove asset');
    }
  };

  // Handle Quick Record Valuation (modal)
  const handleSaveQuickValuation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valuingAsset) return;
    setValuationError('');

    const val = parseAmount(valuationValueStr);
    if (isNaN(val) || val < 0) {
      setValuationError('Please enter a valid non-negative valuation amount.');
      return;
    }

    let dateToUse = valuationDate.trim();
    if (dateToUse && !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      dateToUse = displayDateToIso(dateToUse);
    }
    if (!dateToUse || !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      setValuationError('Please enter a valid valuation date (YYYY-MM-DD).');
      return;
    }

    setSavingValuation(true);
    try {
      await createAssetValue(token, valuingAsset.id, {
        value: val,
        value_date: dateToUse,
      });

      // Reload holdings, summary, and trend to reflect new valuation
      const [refreshedHoldings, refreshedSummary, refreshedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(refreshedHoldings);
      if (refreshedSummary) setSummary(refreshedSummary);
      if (refreshedTrend) setTrendPoints(refreshedTrend);

      if (selectedAssetForHistory === valuingAsset.id) {
        await loadValuationHistory(valuingAsset.id);
      }

      setValuingAsset(null);
      setMessage(`Recorded ${valuingAsset.label} valuation of ${formatCurrency(val)} for ${isoToDisplayDate(dateToUse)}.`);
      setTimeout(() => setMessage(''), 4000);
    } catch (err: unknown) {
      setValuationError(err instanceof Error ? err.message : 'Failed to record valuation');
    } finally {
      setSavingValuation(false);
    }
  };

  // Handle Define New Asset (Tab 3)
  const handleCreateAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateAssetError('');

    const label = newAssetLabel.trim();
    if (!label) {
      setCreateAssetError('Asset symbol/label is required (e.g. BTC, ETH, SOL).');
      return;
    }

    setCreatingAsset(true);
    try {
      const created = await createAsset(token, {
        label,
        description: newAssetDesc.trim(),
      });

      // If initial valuation provided, record it
      if (newAssetInitialVal.trim()) {
        const initVal = parseAmount(newAssetInitialVal);
        let initDate = newAssetInitialDate.trim();
        if (initDate && !/^\d{4}-\d{2}-\d{2}$/.test(initDate)) {
          initDate = displayDateToIso(initDate);
        }
        if (!isNaN(initVal) && initVal >= 0 && /^\d{4}-\d{2}-\d{2}$/.test(initDate)) {
          await createAssetValue(token, created.id, {
            value: initVal,
            value_date: initDate,
          }).catch(() => null);
        }
      }

      setAllAssets((prev) => [...prev, created]);
      setSelectedAssetForHistory(created.id);
      setSelectedAssetIdForAdd(created.id);

      setNewAssetLabel('');
      setNewAssetDesc('');
      setNewAssetInitialVal('');
      setNewAssetInitialDate(getTodayDateString());

      setMessage(`Asset definition "${created.label}" created successfully.`);
      setTimeout(() => setMessage(''), 4000);
    } catch (err: unknown) {
      setCreateAssetError(err instanceof Error ? err.message : 'Failed to create asset definition');
    } finally {
      setCreatingAsset(false);
    }
  };

  // Handle Record or Update Historical Valuation (Tab 3)
  const handleSaveHistoryValuation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAssetForHistory) return;
    setHistoryError('');

    const val = parseAmount(historyValueStr);
    if (isNaN(val) || val < 0) {
      setHistoryError('Please enter a valid non-negative valuation amount.');
      return;
    }

    let dateToUse = historyDateStr.trim();
    if (dateToUse && !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      dateToUse = displayDateToIso(dateToUse);
    }
    if (!dateToUse || !/^\d{4}-\d{2}-\d{2}$/.test(dateToUse)) {
      setHistoryError('Please enter a valid valuation date (YYYY-MM-DD).');
      return;
    }

    setSubmittingHistoryVal(true);
    try {
      if (editingHistoryValId) {
        await updateAssetValue(token, selectedAssetForHistory, editingHistoryValId, {
          value: val,
          value_date: dateToUse,
        });
        setMessage('Valuation updated successfully.');
      } else {
        await createAssetValue(token, selectedAssetForHistory, {
          value: val,
          value_date: dateToUse,
        });
        setMessage(`Valuation of ${formatCurrency(val)} on ${isoToDisplayDate(dateToUse)} recorded successfully.`);
      }

      await loadValuationHistory(selectedAssetForHistory);

      // Refresh account holdings, summary, and trend if this asset is held
      const [refreshedHoldings, refreshedSummary, refreshedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(refreshedHoldings);
      if (refreshedSummary) setSummary(refreshedSummary);
      if (refreshedTrend) setTrendPoints(refreshedTrend);

      setEditingHistoryValId(null);
      setHistoryValueStr('');
      setHistoryDateStr(getTodayDateString());
      setTimeout(() => setMessage(''), 4000);
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to save valuation');
    } finally {
      setSubmittingHistoryVal(false);
    }
  };

  // Delete valuation record from history
  const handleDeleteHistoryValuation = async (valId: string) => {
    if (!confirm('Are you sure you want to delete this historical valuation point?')) return;
    try {
      await deleteAssetValue(token, selectedAssetForHistory, valId);
      await loadValuationHistory(selectedAssetForHistory);

      const [refreshedHoldings, refreshedSummary, refreshedTrend] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
      ]);
      setHoldings(refreshedHoldings);
      if (refreshedSummary) setSummary(refreshedSummary);
      if (refreshedTrend) setTrendPoints(refreshedTrend);

      setMessage('Historical valuation deleted.');
      setTimeout(() => setMessage(''), 3000);
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to delete valuation');
    }
  };

  // Open Edit / Rename Asset Modal
  const handleOpenEditAsset = (assetId: string) => {
    const asset = allAssets.find((a) => a.id === assetId);
    if (!asset) return;
    setEditingAsset(asset);
    setEditAssetLabel(asset.label);
    setEditAssetDesc(asset.description || '');
    setEditAssetError('');
  };

  // Save Asset Edit / Rename
  const handleSaveAssetEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingAsset) return;
    setEditAssetError('');

    const label = editAssetLabel.trim();
    if (!label) {
      setEditAssetError('Asset symbol/label cannot be empty.');
      return;
    }

    setUpdatingAsset(true);
    try {
      const updated = await updateAsset(token, editingAsset.id, {
        label,
        description: editAssetDesc.trim(),
      });

      // Update allAssets in state
      setAllAssets((prev) => prev.map((a) => (a.id === updated.id ? updated : a)));

      // Reload holdings and summary to reflect new label across all account views
      const [refreshedHoldings, refreshedSummary] = await Promise.all([
        fetchAccountAssets(token, account.id),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
      ]);
      setHoldings(refreshedHoldings);
      if (refreshedSummary) setSummary(refreshedSummary);

      setEditingAsset(null);
      setMessage(`Asset definition updated to "${updated.label}".`);
      setTimeout(() => setMessage(''), 4000);
    } catch (err: unknown) {
      setEditAssetError(err instanceof Error ? err.message : 'Failed to update asset definition');
    } finally {
      setUpdatingAsset(false);
    }
  };

  return (
    <div className="account-detail-container">
      {/* Account Header */}
      <div className="account-detail-header">
        <div>
          <div className="account-header-title-row">
            <h2 className="account-title">{account.name}</h2>
            <span className="badge-account-type badge-asset">{account.type}</span>
          </div>
          <p className="account-description">{account.description}</p>
        </div>
      </div>

      {/* Global Alerts */}
      {error && <div className="alert-error">{error}</div>}
      {message && <div className="alert-success">{message}</div>}

      {/* Summary Cards */}
      <div className="asset-summary-grid">
        <div className="asset-summary-card">
          <span className="summary-label">Total Account Worth</span>
          <span className="summary-value-large">{formatCurrency(totalCalculatedWorth)}</span>
          <span className="summary-subtext">Sum of (Quantity × Latest Valuation)</span>
        </div>
        <div className="asset-summary-card">
          <span className="summary-label">Active Holdings</span>
          <span className="summary-value-medium">{holdings.length} {holdings.length === 1 ? 'Asset' : 'Assets'}</span>
          <span className="summary-subtext">Held in this portfolio</span>
        </div>
        <div className="asset-summary-card">
          <span className="summary-label">Catalog Definitions</span>
          <span className="summary-value-medium">{allAssets.length} Defined</span>
          <span className="summary-subtext">Available for accounts</span>
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="snapshot-tabs-bar asset-tabs-bar">
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'holdings' ? 'active' : ''}`}
          onClick={() => setActiveTab('holdings')}
        >
          Holdings & Portfolio
        </button>
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'trend' ? 'active' : ''}`}
          onClick={() => setActiveTab('trend')}
        >
          Valuation Trend
        </button>
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => setActiveTab('settings')}
        >
          Account Settings & Holdings
        </button>
        <button
          type="button"
          className={`snapshot-tab-button ${activeTab === 'definitions' ? 'active' : ''}`}
          onClick={() => setActiveTab('definitions')}
        >
          Asset Definitions & Valuations
        </button>
      </div>

      {loading ? (
        <div className="loading-state">Loading portfolio data...</div>
      ) : (
        <>
          {/* TAB 1: Holdings & Portfolio */}
          {activeTab === 'holdings' && (
            <div className="asset-tab-content">
              <div className="card holdings-card">
                <div className="card-header-flex">
                  <div>
                    <h3>Current Holdings</h3>
                    <p className="muted-text">Assets held in this account across dates with their valuations</p>
                  </div>
                  {availableAssetsToAdd.length > 0 && (
                    <button
                      type="button"
                      className="btn-primary btn-sm"
                      onClick={() => setActiveTab('settings')}
                    >
                      + Add Asset Holding
                    </button>
                  )}
                </div>

                {holdings.length === 0 ? (
                  <div className="empty-holdings-prompt">
                    <p>No asset holdings in this account yet.</p>
                    <div className="empty-actions-row">
                      <button
                        type="button"
                        className="btn-primary"
                        onClick={() => setActiveTab('settings')}
                      >
                        Add Existing Asset
                      </button>
                      <button
                        type="button"
                        className="btn-secondary"
                        onClick={() => setActiveTab('definitions')}
                      >
                        Define New Asset
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="table-responsive holdings-table-container" ref={holdingsContainerRef}>
                    <table
                      className={`holdings-table current-holdings-table ${hideDescription ? 'hide-col-desc' : ''} ${hideActions ? 'hide-col-actions' : ''}`}
                      ref={holdingsTableRef}
                    >
                      <thead>
                        <tr>
                          <th>Asset</th>
                          <th className="col-description">Description</th>
                          <th className="text-right">Held Quantity</th>
                          <th>Holding Date</th>
                          <th className="text-right">Latest Valuation</th>
                          <th>Valuation Date</th>
                          <th className="text-right">Total Worth</th>
                          <th className="text-center col-actions">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {holdings.map((h) => (
                          <tr key={h.id}>
                            <td>
                              <button
                                type="button"
                                className="asset-tag-btn"
                                title="Click to rename or edit asset definition"
                                onClick={() => handleOpenEditAsset(h.asset_id)}
                              >
                                {h.label} <span className="edit-icon-hint">✎</span>
                              </button>
                            </td>
                            <td className="col-description" title={h.description || ''}>{h.description || '—'}</td>
                            <td className="text-right font-mono font-bold">
                              {formatQuantity(h.quantity)}
                            </td>
                            <td>
                              <span className="snapshot-date-badge font-mono">
                                {isoToDisplayDate(h.asset_date)}
                              </span>
                            </td>
                            <td className="text-right font-mono">
                              {h.latest_value > 0 ? formatUnitValue(h.latest_value) : <span className="text-muted">Unvalued</span>}
                            </td>
                            <td>
                              {h.latest_value_date ? (
                                <span className="snapshot-date-badge">
                                  {isoToDisplayDate(h.latest_value_date)}
                                </span>
                              ) : (
                                <span className="text-muted">—</span>
                              )}
                            </td>
                            <td className="text-right font-mono font-bold text-success">
                              {formatCurrency(h.total_value)}
                            </td>
                            <td className="text-center col-actions">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="Rename / Edit Asset Definition"
                                  onClick={() => handleOpenEditAsset(h.asset_id)}
                                >
                                  Rename
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="Update Held Quantity or Date"
                                  onClick={() => {
                                    setEditingHolding(h);
                                    setEditQuantityStr(h.quantity.toString());
                                    setEditHoldingDate(h.asset_date || getTodayDateString());
                                    setEditQuantityError('');
                                  }}
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="Record or Update Valuation"
                                  onClick={() => {
                                    setValuingAsset({
                                      id: h.asset_id,
                                      label: h.label,
                                      description: h.description,
                                    });
                                    setValuationValueStr(h.latest_value > 0 ? h.latest_value.toString() : '');
                                    setValuationDate(getTodayDateString());
                                    setValuationError('');
                                  }}
                                >
                                  Valuate
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action btn-table-action-danger"
                                  title="Remove from Account"
                                  onClick={() => handleRemoveHolding(h)}
                                >
                                  Remove
                                </button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* TAB 2: Valuation Trend */}
          {activeTab === 'trend' && (
            <div className="asset-tab-content">
              <div className="card">
                <AssetTrendGraph points={trendPoints} />
              </div>
            </div>
          )}

          {/* TAB 2: Account Settings & Holdings */}
          {activeTab === 'settings' && (
            <div className="asset-tab-content">
              {/* Add Holding Form */}
              <div className="card new-holding-card">
                <h3>Add Asset to Account</h3>
                <p className="muted-text">
                  Select an asset from the catalog and set your held quantity in this account.
                </p>

                {addHoldingError && <div className="alert-error">{addHoldingError}</div>}

                {availableAssetsToAdd.length === 0 ? (
                  <div className="info-box">
                    <p>All currently defined assets are already added to this account, or no assets have been defined yet.</p>
                    <button
                      type="button"
                      className="btn-primary btn-sm"
                      style={{ marginTop: '0.75rem' }}
                      onClick={() => setActiveTab('definitions')}
                    >
                      Define a New Asset in Catalog
                    </button>
                  </div>
                ) : (
                  <form onSubmit={handleAddHolding} className="add-holding-form">
                    <div className="form-row-3">
                      <div className="form-group">
                        <label htmlFor="add-asset-select">Asset</label>
                        <select
                          id="add-asset-select"
                          value={selectedAssetIdForAdd}
                          onChange={(e) => setSelectedAssetIdForAdd(e.target.value)}
                        >
                          {availableAssetsToAdd.map((a) => (
                            <option key={a.id} value={a.id}>
                              {a.label} {a.description ? `(${a.description})` : ''}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="form-group">
                        <label htmlFor="add-holding-quantity">Held Quantity</label>
                        <input
                          id="add-holding-quantity"
                          type="text"
                          inputMode="decimal"
                          required
                          value={addQuantityStr}
                          onChange={(e) => setAddQuantityStr(e.target.value)}
                          placeholder="e.g. 0.2"
                        />
                      </div>
                      <div className="form-group">
                        <label htmlFor="add-holding-date">Holding Date</label>
                        <DateInput
                          id="add-holding-date"
                          value={addHoldingDate}
                          onChange={(val) => setAddHoldingDate(val)}
                          required
                        />
                      </div>
                    </div>
                    <div className="form-actions">
                      <button
                        type="submit"
                        className="btn-primary"
                        disabled={addingHolding || !selectedAssetIdForAdd}
                      >
                        {addingHolding ? 'Adding...' : 'Add Holding to Account'}
                      </button>
                      <button
                        type="button"
                        className="btn-secondary"
                        onClick={() => setActiveTab('definitions')}
                      >
                        + Define New Asset
                      </button>
                    </div>
                  </form>
                )}
              </div>

              {/* Manage Existing Holdings Section */}
              <div className="card manage-holdings-card">
                <h3>Manage Held Quantities & Valuations</h3>
                <p className="muted-text">
                  Directly adjust quantities or update valuations for the assets in this account.
                </p>

                {holdings.length === 0 ? (
                  <p className="accounts-empty">No holdings in this account yet.</p>
                ) : (
                  <div className="holdings-settings-list">
                    {holdings.map((h) => (
                      <div key={h.id} className="holding-setting-item">
                        <div className="holding-setting-info">
                          <div className="holding-title-row">
                            <span className="asset-tag">{h.label}</span>
                            <span className="holding-desc-text">{h.description}</span>
                          </div>
                          <div className="holding-stats-row">
                            <span>Held Quantity: <strong>{formatQuantity(h.quantity)}</strong></span>
                            <span>Holding Date: <strong>{isoToDisplayDate(h.asset_date)}</strong></span>
                            <span>Valuation: <strong>{h.latest_value > 0 ? formatUnitValue(h.latest_value) : 'Unvalued'}</strong></span>
                            <span>Total Worth: <strong className="text-success">{formatCurrency(h.total_value)}</strong></span>
                          </div>
                        </div>

                        <div className="holding-setting-actions">
                          <button
                            type="button"
                            className="btn-secondary btn-sm"
                            title="Rename or edit this asset definition"
                            onClick={() => handleOpenEditAsset(h.asset_id)}
                          >
                            Rename Asset
                          </button>
                          <button
                            type="button"
                            className="btn-secondary btn-sm"
                            onClick={() => {
                              setEditingHolding(h);
                              setEditQuantityStr(h.quantity.toString());
                              setEditHoldingDate(h.asset_date || getTodayDateString());
                              setEditQuantityError('');
                            }}
                          >
                            Update Holding
                          </button>
                          <button
                            type="button"
                            className="btn-secondary btn-sm"
                            onClick={() => {
                              setValuingAsset({
                                id: h.asset_id,
                                label: h.label,
                                description: h.description,
                              });
                              setValuationValueStr(h.latest_value > 0 ? h.latest_value.toString() : '');
                              setValuationDate(getTodayDateString());
                              setValuationError('');
                            }}
                          >
                            Set Valuation
                          </button>
                          <button
                            type="button"
                            className="btn-danger btn-sm"
                            onClick={() => handleRemoveHolding(h)}
                          >
                            Remove
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Account Metadata */}
              <div className="card account-metadata-card">
                <h3>Account Information</h3>
                <div className="metadata-grid">
                  <div>
                    <span className="metadata-label">Account Name</span>
                    <span className="metadata-val">{account.name}</span>
                  </div>
                  <div>
                    <span className="metadata-label">Account Type</span>
                    <span className="metadata-val">{account.type}</span>
                  </div>
                  <div>
                    <span className="metadata-label">Account ID</span>
                    <span className="metadata-val font-mono">{account.id}</span>
                  </div>
                  <div>
                    <span className="metadata-label">Description</span>
                    <span className="metadata-val">{account.description || '—'}</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* TAB 3: Asset Definitions & Valuations */}
          {activeTab === 'definitions' && (
            <div className="asset-tab-content">
              {/* Define New Asset Card */}
              <div className="card create-asset-card">
                <h3>Define New Asset</h3>
                <p className="muted-text">
                  Add definitions for assets such as Bitcoin, Ethereum, or Solana to the system catalog.
                </p>

                {createAssetError && <div className="alert-error">{createAssetError}</div>}

                <form onSubmit={handleCreateAsset} className="create-asset-form">
                  <div className="form-row-2">
                    <div className="form-group">
                      <label htmlFor="new-asset-label">Asset Symbol / Ticker</label>
                      <input
                        id="new-asset-label"
                        type="text"
                        required
                        value={newAssetLabel}
                        onChange={(e) => setNewAssetLabel(e.target.value)}
                        placeholder="e.g. BTC, ETH, SOL"
                      />
                    </div>
                    <div className="form-group">
                      <label htmlFor="new-asset-desc">Description / Full Name</label>
                      <input
                        id="new-asset-desc"
                        type="text"
                        value={newAssetDesc}
                        onChange={(e) => setNewAssetDesc(e.target.value)}
                        placeholder="e.g. Bitcoin, Ethereum, Solana"
                      />
                    </div>
                  </div>

                  <div className="form-row-2">
                    <div className="form-group">
                      <label htmlFor="new-asset-initial-val">
                        Initial Valuation (Optional USD / unit)
                      </label>
                      <input
                        id="new-asset-initial-val"
                        type="text"
                        inputMode="decimal"
                        value={newAssetInitialVal}
                        onChange={(e) => setNewAssetInitialVal(e.target.value)}
                        placeholder="e.g. 12,000.00"
                      />
                    </div>
                    <div className="form-group">
                      <label htmlFor="new-asset-initial-date">Initial Valuation Date</label>
                      <DateInput
                        id="new-asset-initial-date"
                        value={newAssetInitialDate}
                        onChange={(val) => setNewAssetInitialDate(val)}
                      />
                    </div>
                  </div>

                  <div className="form-actions">
                    <button
                      type="submit"
                      className="btn-primary"
                      disabled={creatingAsset || !newAssetLabel.trim()}
                    >
                      {creatingAsset ? 'Creating...' : 'Create Asset Definition'}
                    </button>
                  </div>
                </form>
              </div>

              {/* Defined Assets Catalog Card */}
              <div className="card catalog-list-card">
                <div className="card-header-flex">
                  <div>
                    <h3>Defined Assets Catalog</h3>
                    <p className="muted-text">
                      Manage and rename asset definitions (e.g. rename "Silver" to "Slv-oz")
                    </p>
                  </div>
                </div>

                {allAssets.length === 0 ? (
                  <p className="accounts-empty">No asset definitions in catalog yet.</p>
                ) : (
                  <div className="table-responsive">
                    <table className="holdings-table">
                      <thead>
                        <tr>
                          <th>Symbol / Ticker</th>
                          <th>Description</th>
                          <th>Type</th>
                          <th className="text-center">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {allAssets.map((asset) => (
                          <tr key={asset.id}>
                            <td>
                              <span className="asset-tag">{asset.label}</span>
                            </td>
                            <td>{asset.description || '—'}</td>
                            <td>
                              {asset.user_id ? (
                                <span className="badge-user-asset">Custom Asset</span>
                              ) : (
                                <span className="badge-system-asset">System Global</span>
                              )}
                            </td>
                            <td className="text-center">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="Rename / Edit Asset Definition"
                                  onClick={() => handleOpenEditAsset(asset.id)}
                                >
                                  Edit / Rename
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="View Valuations"
                                  onClick={() => setSelectedAssetForHistory(asset.id)}
                                >
                                  Valuations
                                </button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>

              {/* Valuation Manager Card */}
              <div className="card valuation-manager-card">
                <div className="card-header-flex">
                  <div>
                    <h3>Asset Valuations History</h3>
                    <p className="muted-text">
                      Record and track valuations over time (e.g. set the value today vs a month ago).
                    </p>
                  </div>
                  {allAssets.length > 0 && (
                    <div className="asset-picker-box">
                      <label htmlFor="asset-history-select">Asset:</label>
                      <select
                        id="asset-history-select"
                        value={selectedAssetForHistory}
                        onChange={(e) => setSelectedAssetForHistory(e.target.value)}
                      >
                        {allAssets.map((a) => (
                          <option key={a.id} value={a.id}>
                            {a.label} {a.description ? `(${a.description})` : ''}
                          </option>
                        ))}
                      </select>
                      <button
                        type="button"
                        className="btn-secondary btn-sm"
                        title="Rename or edit this asset definition"
                        onClick={() => handleOpenEditAsset(selectedAssetForHistory)}
                        disabled={!selectedAssetForHistory}
                      >
                        Rename / Edit Asset
                      </button>
                    </div>
                  )}
                </div>

                {historyError && <div className="alert-error">{historyError}</div>}

                {/* Form to record/edit valuation for selected asset */}
                {selectedAssetForHistory && (
                  <form onSubmit={handleSaveHistoryValuation} className="history-valuation-form">
                    <h4>{editingHistoryValId ? 'Edit Valuation Point' : 'Record Valuation Point'}</h4>
                    <div className="form-row-2">
                      <div className="form-group">
                        <label htmlFor="history-val-input">Valuation (USD per unit)</label>
                        <input
                          id="history-val-input"
                          type="text"
                          inputMode="decimal"
                          required
                          value={historyValueStr}
                          onChange={(e) => setHistoryValueStr(e.target.value)}
                          placeholder="e.g. 12,000.00"
                        />
                      </div>
                      <div className="form-group">
                        <label htmlFor="history-date-input">Valuation Date</label>
                        <DateInput
                          id="history-date-input"
                          value={historyDateStr}
                          onChange={(val) => setHistoryDateStr(val)}
                          required
                        />
                      </div>
                    </div>

                    <div className="form-actions">
                      {editingHistoryValId && (
                        <button
                          type="button"
                          className="btn-secondary"
                          onClick={() => {
                            setEditingHistoryValId(null);
                            setHistoryValueStr('');
                            setHistoryDateStr(getTodayDateString());
                          }}
                        >
                          Cancel Edit
                        </button>
                      )}
                      <button
                        type="submit"
                        className="btn-primary"
                        disabled={submittingHistoryVal || !historyValueStr.trim()}
                      >
                        {submittingHistoryVal
                          ? 'Saving...'
                          : editingHistoryValId
                          ? 'Update Valuation'
                          : 'Record Valuation'}
                      </button>
                    </div>
                  </form>
                )}

                {/* History Table */}
                {loadingHistory ? (
                  <div className="loading-state">Loading valuations...</div>
                ) : assetValuationHistory.length === 0 ? (
                  <p className="accounts-empty">No valuation history recorded for this asset yet.</p>
                ) : (
                  <div className="table-responsive" style={{ marginTop: '1.25rem' }}>
                    <table className="holdings-table">
                      <thead>
                        <tr>
                          <th>Valuation Date</th>
                          <th className="text-right">Unit Price (USD)</th>
                          <th>Recorded Timestamp</th>
                          <th className="text-center">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {assetValuationHistory.map((v) => (
                          <tr key={v.id}>
                            <td>
                              <span className="snapshot-date-badge">
                                {isoToDisplayDate(v.value_date)}
                              </span>
                            </td>
                            <td className="text-right font-mono font-bold">
                              {formatUnitValue(v.value)}
                            </td>
                            <td className="snapshot-timestamp text-muted">
                              {new Date(v.created_at).toLocaleString()}
                            </td>
                            <td className="text-center">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  onClick={() => {
                                    setEditingHistoryValId(v.id);
                                    setHistoryValueStr(v.value.toString());
                                    setHistoryDateStr(v.value_date);
                                  }}
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action btn-table-action-danger"
                                  onClick={() => handleDeleteHistoryValuation(v.id)}
                                >
                                  Delete
                                </button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}

      {/* MODAL: Edit Holding Quantity & Date */}
      {editingHolding && (
        <div className="modal-backdrop" onClick={() => setEditingHolding(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Update Holding</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingHolding(null)}
              >
                ×
              </button>
            </div>
            {editQuantityError && <div className="alert-error">{editQuantityError}</div>}
            <form onSubmit={handleUpdateQuantity}>
              <div className="form-group">
                <label>Asset</label>
                <div className="static-field-value">
                  <span className="asset-tag">{editingHolding.label}</span>
                  <span style={{ marginLeft: '0.5rem', color: 'var(--muted)' }}>
                    {editingHolding.description}
                  </span>
                </div>
              </div>
              <div className="form-group">
                <label htmlFor="edit-holding-quantity">Held Quantity</label>
                <input
                  id="edit-holding-quantity"
                  ref={editQtyInputRef}
                  type="text"
                  inputMode="decimal"
                  required
                  value={editQuantityStr}
                  onChange={(e) => setEditQuantityStr(e.target.value)}
                  placeholder="e.g. 0.3"
                />
              </div>
              <div className="form-group">
                <label htmlFor="edit-holding-date">Holding Date</label>
                <DateInput
                  id="edit-holding-date"
                  value={editHoldingDate}
                  onChange={(val) => setEditHoldingDate(val)}
                  required
                />
              </div>
              <div className="form-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingHolding(null)}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={updatingQuantity}
                >
                  {updatingQuantity ? 'Saving...' : 'Save Holding'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Quick Record Valuation */}
      {valuingAsset && (
        <div className="modal-backdrop" onClick={() => setValuingAsset(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Record Asset Valuation</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setValuingAsset(null)}
              >
                ×
              </button>
            </div>
            {valuationError && <div className="alert-error">{valuationError}</div>}
            <form onSubmit={handleSaveQuickValuation}>
              <div className="form-group">
                <label>Asset</label>
                <div className="static-field-value">
                  <span className="asset-tag">{valuingAsset.label}</span>
                  <span style={{ marginLeft: '0.5rem', color: 'var(--muted)' }}>
                    {valuingAsset.description}
                  </span>
                </div>
              </div>
              <div className="form-group">
                <label htmlFor="quick-valuation-val">Per-Unit Price (USD)</label>
                <input
                  id="quick-valuation-val"
                  ref={valuationInputRef}
                  type="text"
                  inputMode="decimal"
                  required
                  value={valuationValueStr}
                  onChange={(e) => setValuationValueStr(e.target.value)}
                  placeholder="e.g. 12,000.00"
                />
              </div>
              <div className="form-group">
                <label htmlFor="quick-valuation-date">Valuation Date</label>
                <DateInput
                  id="quick-valuation-date"
                  value={valuationDate}
                  onChange={(val) => setValuationDate(val)}
                  required
                />
              </div>
              <div className="form-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setValuingAsset(null)}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={savingValuation}
                >
                  {savingValuation ? 'Saving...' : 'Record Valuation'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      {/* MODAL: Edit Asset Definition */}
      {editingAsset && (
        <div className="modal-backdrop" onClick={() => setEditingAsset(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Rename / Edit Asset Definition</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingAsset(null)}
              >
                ×
              </button>
            </div>
            {editAssetError && <div className="alert-error">{editAssetError}</div>}
            <form onSubmit={handleSaveAssetEdit}>
              <div className="form-group">
                <label htmlFor="edit-asset-label">Asset Symbol / Ticker</label>
                <input
                  id="edit-asset-label"
                  ref={editAssetInputRef}
                  type="text"
                  required
                  value={editAssetLabel}
                  onChange={(e) => setEditAssetLabel(e.target.value)}
                  placeholder="e.g. Slv-oz, BTC, ETH"
                />
              </div>
              <div className="form-group">
                <label htmlFor="edit-asset-desc">Description / Full Name</label>
                <input
                  id="edit-asset-desc"
                  type="text"
                  value={editAssetDesc}
                  onChange={(e) => setEditAssetDesc(e.target.value)}
                  placeholder="e.g. Silver 1 Troy Ounce"
                />
              </div>
              <div className="form-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingAsset(null)}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={updatingAsset || !editAssetLabel.trim()}
                >
                  {updatingAsset ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
