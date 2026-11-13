'use client';

import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Account,
  Asset,
  AssetAccountSummary,
  AssetAccountTrendPoint,
  AssetStore,
  AssetStoreRecord,
  AssetStoreTrendPoint,
  AssetValue,
} from '../lib/types';
import { DateInput, displayDateToIso, isoToDisplayDate } from './DateInput';
import { AssetTrendGraph } from './AssetTrendGraph';
import {
  fetchAssetStores,
  createAssetStore,
  updateAssetStore,
  deleteAssetStore,
  fetchAssetStoreRecords,
  createAssetStoreRecord,
  updateAssetStoreRecord,
  deleteAssetStoreRecord,
  fetchAssetStoreTrend,
  fetchAssetAccountSummary,
  fetchAssetAccountTrend,
  fetchAssets,
  createAsset,
  updateAsset,
  deleteAsset,
  fetchAssetValues,
  createAssetValue,
  updateAssetValue,
  deleteAssetValue,
} from '../lib/api';

interface AssetAccountDetailProps {
  account: Account;
  token: string;
}

type AssetTab = 'classes' | 'values' | 'storage' | 'records' | 'trend';

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
  const [stores, setStores] = useState<AssetStore[]>([]);
  const [summary, setSummary] = useState<AssetAccountSummary | null>(null);
  const [trendPoints, setTrendPoints] = useState<AssetAccountTrendPoint[]>([]);
  const [allAssets, setAllAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  // 5 Tabs: classes, values, storage, records, trend
  const [activeTab, setActiveTab] = useState<AssetTab>('classes');
  const [initialTabSet, setInitialTabSet] = useState(false);
  const [hasAnyAssetValue, setHasAnyAssetValue] = useState(false);

  // Dynamic column hiding for Storage table
  const [hideDescription, setHideDescription] = useState(false);
  const [hideActions, setHideActions] = useState(false);
  const holdingsContainerRef = useRef<HTMLDivElement>(null);
  const holdingsTableRef = useRef<HTMLTableElement>(null);
  const hideDescRef = useRef(false);
  const hideActionsRef = useRef(false);
  const descThresholdRef = useRef<number>(940);
  const actionsThresholdRef = useRef<number>(720);

  // Tab 1: Asset Classes state
  const [newAssetLabel, setNewAssetLabel] = useState('');
  const [newAssetDesc, setNewAssetDesc] = useState('');
  const [creatingAsset, setCreatingAsset] = useState(false);
  const [createAssetError, setCreateAssetError] = useState('');
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null);
  const [editAssetLabel, setEditAssetLabel] = useState('');
  const [editAssetDesc, setEditAssetDesc] = useState('');
  const [updatingAsset, setUpdatingAsset] = useState(false);
  const [editAssetError, setEditAssetError] = useState('');

  // Tab 2: Asset Values state
  const [selectedAssetForValues, setSelectedAssetForValues] = useState<string>('');
  const [assetValuationHistory, setAssetValuationHistory] = useState<AssetValue[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [historyValueStr, setHistoryValueStr] = useState('');
  const [historyDateStr, setHistoryDateStr] = useState(getTodayDateString());
  const [submittingHistoryVal, setSubmittingHistoryVal] = useState(false);
  const [historyError, setHistoryError] = useState('');
  const [editingHistoryVal, setEditingHistoryVal] = useState<AssetValue | null>(null);
  const [editHistoryValStr, setEditHistoryValStr] = useState('');
  const [editHistoryDateStr, setEditHistoryDateStr] = useState(getTodayDateString());

  // Tab 3: Asset Storage state
  const [newStoreAssetId, setNewStoreAssetId] = useState('');
  const [newStoreLabel, setNewStoreLabel] = useState('');
  const [newStoreDesc, setNewStoreDesc] = useState('');
  const [newStoreInitQty, setNewStoreInitQty] = useState('');
  const [newStoreInitDate, setNewStoreInitDate] = useState(getTodayDateString());
  const [creatingStore, setCreatingStore] = useState(false);
  const [createStoreError, setCreateStoreError] = useState('');
  const [editingStore, setEditingStore] = useState<AssetStore | null>(null);
  const [editStoreLabel, setEditStoreLabel] = useState('');
  const [editStoreDesc, setEditStoreDesc] = useState('');
  const [editStoreAssetId, setEditStoreAssetId] = useState('');
  const [updatingStore, setUpdatingStore] = useState(false);
  const [editStoreError, setEditStoreError] = useState('');

  // Tab 4: Quantity Records state
  const [selectedStoreIdForRecords, setSelectedStoreIdForRecords] = useState<string>('');
  const [storeRecords, setStoreRecords] = useState<AssetStoreRecord[]>([]);
  const [storeTrendPoints, setStoreTrendPoints] = useState<AssetStoreTrendPoint[]>([]);
  const [loadingRecords, setLoadingRecords] = useState(false);
  const [newRecordQtyStr, setNewRecordQtyStr] = useState('');
  const [newRecordDateStr, setNewRecordDateStr] = useState(getTodayDateString());
  const [addingRecord, setAddingRecord] = useState(false);
  const [addRecordError, setAddRecordError] = useState('');
  const [editingRecord, setEditingRecord] = useState<AssetStoreRecord | null>(null);
  const [editRecordQtyStr, setEditRecordQtyStr] = useState('');
  const [editRecordDateStr, setEditRecordDateStr] = useState(getTodayDateString());
  const [updatingRecord, setUpdatingRecord] = useState(false);
  const [editRecordError, setEditRecordError] = useState('');

  const editAssetInputRef = useRef<HTMLInputElement>(null);

  // Load account stores, summary, catalog, and valuation trend
  const loadData = async () => {
    setLoading(true);
    setError('');
    try {
      const [storesData, sumData, trendData, catalogData] = await Promise.all([
        fetchAssetStores(token, account.id).catch(() => []),
        fetchAssetAccountSummary(token, account.id).catch(() => null),
        fetchAssetAccountTrend(token, account.id).catch(() => []),
        fetchAssets(token).catch(() => []),
      ]);
      const safeStores = Array.isArray(storesData) ? storesData : [];
      const safeTrend = Array.isArray(trendData) ? trendData : [];
      const safeCatalog = Array.isArray(catalogData) ? catalogData : [];

      setStores(safeStores);
      setSummary(sumData);
      setTrendPoints(safeTrend);
      setAllAssets(safeCatalog);

      let anyValFound = false;
      if (safeStores.some((s) => Boolean(s.latest_value_date) || (typeof s.latest_value === 'number' && s.latest_value > 0))) {
        anyValFound = true;
      } else if (safeCatalog.length > 0) {
        const valChecks = await Promise.all(
          safeCatalog.map((a) => fetchAssetValues(token, a.id).catch(() => []))
        );
        anyValFound = valChecks.some((history) => Array.isArray(history) && history.length > 0);
      }
      setHasAnyAssetValue(anyValFound);

      if (safeStores.length > 0 && !selectedStoreIdForRecords) {
        setSelectedStoreIdForRecords(safeStores[0].id);
      }
      if (safeCatalog.length > 0 && !selectedAssetForValues) {
        setSelectedAssetForValues(safeCatalog[0].id);
      }
      if (safeCatalog.length > 0 && !newStoreAssetId) {
        setNewStoreAssetId(safeCatalog[0].id);
      }

      if (!initialTabSet) {
        if (safeStores.length > 0) {
          setActiveTab('storage');
        } else if (safeCatalog.length > 0) {
          setActiveTab('storage');
        } else {
          setActiveTab('classes');
        }
        setInitialTabSet(true);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to load asset account details.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [account.id, token]);

  // Load valuations when selected asset for Tab 2 changes
  useEffect(() => {
    if (!selectedAssetForValues) {
      setAssetValuationHistory([]);
      return;
    }
    setLoadingHistory(true);
    setHistoryError('');
    fetchAssetValues(token, selectedAssetForValues)
      .then((history) => {
        setAssetValuationHistory(Array.isArray(history) ? history : []);
      })
      .catch((err) => {
        setHistoryError(err instanceof Error ? err.message : 'Failed to load valuations.');
      })
      .finally(() => {
        setLoadingHistory(false);
      });
  }, [selectedAssetForValues, token]);

  // Load records and trend when selected store for Tab 4 changes
  useEffect(() => {
    if (!selectedStoreIdForRecords) {
      setStoreRecords([]);
      setStoreTrendPoints([]);
      return;
    }
    setLoadingRecords(true);
    setAddRecordError('');
    Promise.all([
      fetchAssetStoreRecords(token, account.id, selectedStoreIdForRecords).catch(() => []),
      fetchAssetStoreTrend(token, account.id, selectedStoreIdForRecords).catch(() => []),
    ])
      .then(([recs, tr]) => {
        setStoreRecords(Array.isArray(recs) ? recs : []);
        setStoreTrendPoints(Array.isArray(tr) ? tr : []);
      })
      .catch((err) => {
        setAddRecordError(err instanceof Error ? err.message : 'Failed to load store records.');
      })
      .finally(() => {
        setLoadingRecords(false);
      });
  }, [selectedStoreIdForRecords, account.id, token]);

  // Sync refs for dynamic overflow handling
  useEffect(() => {
    hideDescRef.current = hideDescription;
  }, [hideDescription]);

  useEffect(() => {
    hideActionsRef.current = hideActions;
  }, [hideActions]);

  // Dynamic auto-hiding of Description and Actions columns when storage table does not fit
  useEffect(() => {
    if (activeTab !== 'storage' || !Array.isArray(stores) || stores.length === 0) return;

    const container = holdingsContainerRef.current;
    const table = holdingsTableRef.current;
    if (!container || !table) return;

    descThresholdRef.current = 940;
    actionsThresholdRef.current = 720;

    const checkFit = () => {
      const containerWidth = container.clientWidth;
      if (containerWidth === 0) return;

      const tableWidth = table.scrollWidth;

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
  }, [activeTab, stores]);

  const totalCalculatedWorth = useMemo(() => {
    if (summary && typeof summary.total_value === 'number') {
      return summary.total_value;
    }
    const list = Array.isArray(stores) ? stores : [];
    return list.reduce((sum, s) => sum + (s.total_value || 0), 0);
  }, [summary, stores]);

  // Selected store object for Tab 4
  const currentSelectedStore = useMemo(() => {
    const list = Array.isArray(stores) ? stores : [];
    return list.find((s) => s.id === selectedStoreIdForRecords) || null;
  }, [stores, selectedStoreIdForRecords]);

  // Selected asset object for Tab 2
  const currentSelectedAsset = useMemo(() => {
    const list = Array.isArray(allAssets) ? allAssets : [];
    return list.find((a) => a.id === selectedAssetForValues) || null;
  }, [allAssets, selectedAssetForValues]);

  // Tab Visibility Rules:
  // 1. If no Asset Classes: hide Values, Storage, Quantity Records, and Trend. Only Classes is shown.
  // 2. Once Asset Classes are defined: Show Asset Values and Asset Storage.
  // 3. Once an Asset Storage record is defined (non-empty): Show Quantity Records.
  // 4. Once at least one Asset Value and one Quantity Record are defined: Show Account Value Trend.
  const hasAssetClasses = (allAssets || []).length > 0;
  const hasAssetStorage = (stores || []).length > 0;
  const hasQuantityRecord =
    (stores || []).some(
      (s) => Boolean(s.latest_record_date) || (typeof s.latest_quantity === 'number' && s.latest_quantity > 0)
    ) || (storeRecords || []).length > 0;
  const hasAssetValue =
    hasAnyAssetValue ||
    (stores || []).some(
      (s) => Boolean(s.latest_value_date) || (typeof s.latest_value === 'number' && s.latest_value > 0)
    ) ||
    (assetValuationHistory || []).length > 0;

  const showClassesTab = true;
  const showValuesTab = hasAssetClasses;
  const showStorageTab = hasAssetClasses;
  const showRecordsTab = hasAssetClasses && hasAssetStorage;
  const showTrendTab = hasAssetClasses && hasAssetStorage && hasAssetValue && hasQuantityRecord;

  // Keep activeTab on an eligible visible tab
  useEffect(() => {
    if (loading) return;
    if (!showValuesTab && activeTab === 'values') {
      setActiveTab('classes');
    } else if (!showStorageTab && activeTab === 'storage') {
      setActiveTab('classes');
    } else if (!showRecordsTab && activeTab === 'records') {
      setActiveTab(showStorageTab ? 'storage' : 'classes');
    } else if (!showTrendTab && activeTab === 'trend') {
      setActiveTab(showRecordsTab ? 'records' : showStorageTab ? 'storage' : 'classes');
    }
  }, [loading, showValuesTab, showStorageTab, showRecordsTab, showTrendTab, activeTab]);

  // Handlers for Tab 1: Asset Classes
  const handleCreateAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateAssetError('');
    const label = newAssetLabel.trim();
    if (!label) {
      setCreateAssetError('Asset label / symbol is required.');
      return;
    }
    setCreatingAsset(true);
    try {
      await createAsset(token, { label, description: newAssetDesc.trim() });
      setNewAssetLabel('');
      setNewAssetDesc('');
      setMessage(`Asset class "${label}" created successfully.`);
      await loadData();
    } catch (err: unknown) {
      setCreateAssetError(err instanceof Error ? err.message : 'Failed to create asset class.');
    } finally {
      setCreatingAsset(false);
    }
  };

  const handleUpdateAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingAsset) return;
    setEditAssetError('');
    const label = editAssetLabel.trim();
    if (!label) {
      setEditAssetError('Asset label is required.');
      return;
    }
    setUpdatingAsset(true);
    try {
      await updateAsset(token, editingAsset.id, { label, description: editAssetDesc.trim() });
      setEditingAsset(null);
      setMessage(`Asset class updated to "${label}".`);
      await loadData();
    } catch (err: unknown) {
      setEditAssetError(err instanceof Error ? err.message : 'Failed to update asset class.');
    } finally {
      setUpdatingAsset(false);
    }
  };

  const handleDeleteAsset = async (asset: Asset) => {
    if (!window.confirm(`Are you sure you want to delete asset "${asset.label}"?`)) return;
    try {
      await deleteAsset(token, asset.id);
      setMessage(`Asset class "${asset.label}" removed.`);
      await loadData();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete asset class.');
    }
  };

  // Handlers for Tab 2: Asset Values
  const handleAddValuation = async (e: React.FormEvent) => {
    e.preventDefault();
    setHistoryError('');
    if (!selectedAssetForValues) {
      setHistoryError('Please select an asset.');
      return;
    }
    const val = parseAmount(historyValueStr);
    if (isNaN(val) || val < 0) {
      setHistoryError('Please enter a valid non-negative valuation amount.');
      return;
    }
    const dateStr = displayDateToIso(historyDateStr);
    if (!dateStr) {
      setHistoryError('Please select a valid date.');
      return;
    }
    setSubmittingHistoryVal(true);
    try {
      await createAssetValue(token, selectedAssetForValues, { value: val, value_date: dateStr });
      setHistoryValueStr('');
      setHistoryDateStr(getTodayDateString());
      setMessage(`Valuation of ${formatUnitValue(val)} recorded.`);
      const updatedHistory = await fetchAssetValues(token, selectedAssetForValues);
      setAssetValuationHistory(updatedHistory);
      await loadData();
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to record valuation.');
    } finally {
      setSubmittingHistoryVal(false);
    }
  };

  const handleUpdateValuation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingHistoryVal || !selectedAssetForValues) return;
    setHistoryError('');
    const val = parseAmount(editHistoryValStr);
    if (isNaN(val) || val < 0) {
      setHistoryError('Please enter a valid non-negative valuation.');
      return;
    }
    const dateStr = displayDateToIso(editHistoryDateStr);
    try {
      await updateAssetValue(token, selectedAssetForValues, editingHistoryVal.id, { value: val, value_date: dateStr });
      setEditingHistoryVal(null);
      setMessage('Valuation updated.');
      const updatedHistory = await fetchAssetValues(token, selectedAssetForValues);
      setAssetValuationHistory(updatedHistory);
      await loadData();
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to update valuation.');
    }
  };

  const handleDeleteValuation = async (valId: string) => {
    if (!selectedAssetForValues || !window.confirm('Delete this valuation record?')) return;
    try {
      await deleteAssetValue(token, selectedAssetForValues, valId);
      setMessage('Valuation deleted.');
      const updatedHistory = await fetchAssetValues(token, selectedAssetForValues);
      setAssetValuationHistory(updatedHistory);
      await loadData();
    } catch (err: unknown) {
      setHistoryError(err instanceof Error ? err.message : 'Failed to delete valuation.');
    }
  };

  // Handlers for Tab 3: Asset Storage
  const handleCreateStore = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateStoreError('');
    if (!newStoreAssetId) {
      setCreateStoreError('Please select an asset class.');
      return;
    }
    const label = newStoreLabel.trim();
    if (!label) {
      setCreateStoreError('Storage label is required (e.g., "red", "attic box").');
      return;
    }
    let qty: number | undefined;
    if (newStoreInitQty.trim()) {
      qty = parseAmount(newStoreInitQty);
      if (isNaN(qty) || qty <= 0) {
        setCreateStoreError('Initial quantity must be greater than zero.');
        return;
      }
    }
    const dateStr = newStoreInitDate ? displayDateToIso(newStoreInitDate) : undefined;
    setCreatingStore(true);
    try {
      const created = await createAssetStore(token, account.id, {
        asset_id: newStoreAssetId,
        label,
        description: newStoreDesc.trim(),
        quantity: qty,
        record_date: dateStr,
      });
      setNewStoreLabel('');
      setNewStoreDesc('');
      setNewStoreInitQty('');
      setMessage(`Asset storage "${label}" created successfully.`);
      setSelectedStoreIdForRecords(created.id);
      await loadData();
    } catch (err: unknown) {
      setCreateStoreError(err instanceof Error ? err.message : 'Failed to create storage location.');
    } finally {
      setCreatingStore(false);
    }
  };

  const handleUpdateStore = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingStore) return;
    setEditStoreError('');
    const label = editStoreLabel.trim();
    if (!label) {
      setEditStoreError('Storage label is required.');
      return;
    }
    setUpdatingStore(true);
    try {
      await updateAssetStore(token, account.id, editingStore.id, {
        label,
        description: editStoreDesc.trim(),
        asset_id: editStoreAssetId || undefined,
      });
      setEditingStore(null);
      setMessage(`Storage updated.`);
      await loadData();
    } catch (err: unknown) {
      setEditStoreError(err instanceof Error ? err.message : 'Failed to update storage.');
    } finally {
      setUpdatingStore(false);
    }
  };

  const handleDeleteStore = async (s: AssetStore) => {
    if (!window.confirm(`Delete storage "${s.label}" and all its records?`)) return;
    try {
      await deleteAssetStore(token, account.id, s.id);
      setMessage(`Storage "${s.label}" removed.`);
      await loadData();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to delete storage.');
    }
  };

  // Handlers for Tab 4: Quantity Records
  const handleAddStoreRecord = async (e: React.FormEvent) => {
    e.preventDefault();
    setAddRecordError('');
    if (!selectedStoreIdForRecords) {
      setAddRecordError('Please select a storage location.');
      return;
    }
    const qty = parseAmount(newRecordQtyStr);
    if (isNaN(qty) || qty <= 0) {
      setAddRecordError('Quantity must be greater than zero.');
      return;
    }
    const dateStr = displayDateToIso(newRecordDateStr);
    if (!dateStr) {
      setAddRecordError('Please select a valid date.');
      return;
    }
    setAddingRecord(true);
    try {
      await createAssetStoreRecord(token, account.id, selectedStoreIdForRecords, {
        quantity: qty,
        record_date: dateStr,
      });
      setNewRecordQtyStr('');
      setMessage(`Recorded quantity of ${formatQuantity(qty)} on ${isoToDisplayDate(dateStr)}.`);
      const [updatedRecs, updatedTrend] = await Promise.all([
        fetchAssetStoreRecords(token, account.id, selectedStoreIdForRecords),
        fetchAssetStoreTrend(token, account.id, selectedStoreIdForRecords),
      ]);
      setStoreRecords(updatedRecs);
      setStoreTrendPoints(updatedTrend);
      await loadData();
    } catch (err: unknown) {
      setAddRecordError(err instanceof Error ? err.message : 'Failed to record quantity.');
    } finally {
      setAddingRecord(false);
    }
  };

  const handleUpdateStoreRecord = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingRecord || !selectedStoreIdForRecords) return;
    setEditRecordError('');
    const qty = parseAmount(editRecordQtyStr);
    if (isNaN(qty) || qty <= 0) {
      setEditRecordError('Quantity must be greater than zero.');
      return;
    }
    const dateStr = displayDateToIso(editRecordDateStr);
    setUpdatingRecord(true);
    try {
      await updateAssetStoreRecord(token, account.id, selectedStoreIdForRecords, editingRecord.id, {
        quantity: qty,
        record_date: dateStr,
      });
      setEditingRecord(null);
      setMessage('Quantity record updated.');
      const [updatedRecs, updatedTrend] = await Promise.all([
        fetchAssetStoreRecords(token, account.id, selectedStoreIdForRecords),
        fetchAssetStoreTrend(token, account.id, selectedStoreIdForRecords),
      ]);
      setStoreRecords(updatedRecs);
      setStoreTrendPoints(updatedTrend);
      await loadData();
    } catch (err: unknown) {
      setEditRecordError(err instanceof Error ? err.message : 'Failed to update record.');
    } finally {
      setUpdatingRecord(false);
    }
  };

  const handleDeleteStoreRecord = async (recId: string) => {
    if (!selectedStoreIdForRecords || !window.confirm('Delete this quantity record?')) return;
    try {
      await deleteAssetStoreRecord(token, account.id, selectedStoreIdForRecords, recId);
      setMessage('Quantity record deleted.');
      const [updatedRecs, updatedTrend] = await Promise.all([
        fetchAssetStoreRecords(token, account.id, selectedStoreIdForRecords),
        fetchAssetStoreTrend(token, account.id, selectedStoreIdForRecords),
      ]);
      setStoreRecords(updatedRecs);
      setStoreTrendPoints(updatedTrend);
      await loadData();
    } catch (err: unknown) {
      setAddRecordError(err instanceof Error ? err.message : 'Failed to delete record.');
    }
  };

  return (
    <div className="asset-account-detail">
      {/* Top Portfolio Summary Header */}
      <div className="asset-summary-grid">
        <div className="asset-summary-card">
          <span className="summary-label">Total Account Value</span>
          <span className="summary-value-large text-success">
            {formatCurrency(totalCalculatedWorth)}
          </span>
          <span className="summary-subtext">Cumulative worth across all storage locations</span>
        </div>

        <div className="asset-summary-card">
          <span className="summary-label">Storage Locations</span>
          <span className="summary-value-medium">{(stores || []).length}</span>
          <span className="summary-subtext">Active holding locations / keys</span>
        </div>

        <div className="asset-summary-card">
          <span className="summary-label">Catalog Classes</span>
          <span className="summary-value-medium">{(allAssets || []).length}</span>
          <span className="summary-subtext">Configured asset classes</span>
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="snapshot-tabs-container">
        <div className="snapshot-tabs-bar asset-tabs-bar">
          <button
            type="button"
            className={`snapshot-tab-button ${activeTab === 'classes' ? 'active' : ''}`}
            onClick={() => { setActiveTab('classes'); setMessage(''); setError(''); }}
          >
            Asset Classes
          </button>
          {showValuesTab && (
            <button
              type="button"
              className={`snapshot-tab-button ${activeTab === 'values' ? 'active' : ''}`}
              onClick={() => { setActiveTab('values'); setMessage(''); setError(''); }}
            >
              Asset Values
            </button>
          )}
          {showStorageTab && (
            <button
              type="button"
              className={`snapshot-tab-button ${activeTab === 'storage' ? 'active' : ''}`}
              onClick={() => { setActiveTab('storage'); setMessage(''); setError(''); }}
            >
              Asset Storage ({(stores || []).length})
            </button>
          )}
          {showRecordsTab && (
            <button
              type="button"
              className={`snapshot-tab-button ${activeTab === 'records' ? 'active' : ''}`}
              onClick={() => { setActiveTab('records'); setMessage(''); setError(''); }}
            >
              Quantity Records
            </button>
          )}
          {showTrendTab && (
            <button
              type="button"
              className={`snapshot-tab-button ${activeTab === 'trend' ? 'active' : ''}`}
              onClick={() => { setActiveTab('trend'); setMessage(''); setError(''); }}
            >
              Account Value Trend
            </button>
          )}
        </div>
      </div>

      {error && <div className="alert-box alert-box-error">{error}</div>}
      {message && <div className="alert-box alert-box-success">{message}</div>}

      {loading ? (
        <div className="loading-state">Loading portfolio data...</div>
      ) : (
        <>
          {/* ========================================================================= */}
          {/* TAB 1: Asset Classes */}
          {/* ========================================================================= */}
          {activeTab === 'classes' && (
            <div className="asset-tab-content">
              <div className="card">
                <div className="card-header-flex">
                  <div>
                    <h3>Asset Classes Catalog</h3>
                    <p className="accounts-empty" style={{ textAlign: 'left', padding: '0.2rem 0' }}>
                      Define and manage the asset classes / definitions (e.g. Silver, Gold, Precious Metals, Digital Tokens) tracked across your portfolios.
                    </p>
                  </div>
                </div>

                <form onSubmit={handleCreateAsset} className="form-grid mb-4">
                  <div className="form-row-2">
                    <div className="form-group">
                      <label htmlFor="new-asset-label">Symbol / Class Label *</label>
                      <input
                        id="new-asset-label"
                        type="text"
                        placeholder="e.g. SLV-oz, Gold-oz, Token-Alpha"
                        value={newAssetLabel}
                        onChange={(e) => setNewAssetLabel(e.target.value)}
                        required
                      />
                    </div>
                    <div className="form-group">
                      <label htmlFor="new-asset-desc">Description</label>
                      <input
                        id="new-asset-desc"
                        type="text"
                        placeholder="e.g. Decentralized digital currency"
                        value={newAssetDesc}
                        onChange={(e) => setNewAssetDesc(e.target.value)}
                      />
                    </div>
                  </div>

                  {createAssetError && <p className="text-danger font-sm">{createAssetError}</p>}

                  <div className="form-actions" style={{ marginTop: '0.75rem' }}>
                    <button type="submit" className="btn-primary" disabled={creatingAsset}>
                      {creatingAsset ? 'Defining Class...' : 'Define Asset Class'}
                    </button>
                  </div>
                </form>

                {(allAssets || []).length === 0 ? (
                  <p className="accounts-empty">No asset classes in catalog yet.</p>
                ) : (
                  <div className="table-responsive">
                    <table className="holdings-table">
                      <thead>
                        <tr>
                          <th>Class / Symbol</th>
                          <th>Description</th>
                          <th>Type</th>
                          <th className="text-center">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(allAssets || []).map((asset) => (
                          <tr key={asset.id}>
                            <td>
                              <span className="asset-tag">{asset.label}</span>
                            </td>
                            <td>{asset.description || '—'}</td>
                            <td>
                              {asset.user_id ? (
                                <span className="badge-user-asset">Custom</span>
                              ) : (
                                <span className="badge-system-asset">System</span>
                              )}
                            </td>
                            <td className="text-center">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  onClick={() => {
                                    setEditingAsset(asset);
                                    setEditAssetLabel(asset.label);
                                    setEditAssetDesc(asset.description);
                                    setEditAssetError('');
                                  }}
                                >
                                  Rename / Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  onClick={() => {
                                    setSelectedAssetForValues(asset.id);
                                    setActiveTab('values');
                                  }}
                                >
                                  Manage Values
                                </button>
                                {asset.user_id && (
                                  <button
                                    type="button"
                                    className="btn-table-action btn-table-action-danger"
                                    onClick={() => handleDeleteAsset(asset)}
                                  >
                                    Delete
                                  </button>
                                )}
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

          {/* ========================================================================= */}
          {/* TAB 2: Asset Values */}
          {/* ========================================================================= */}
          {showValuesTab && activeTab === 'values' && (
            <div className="asset-tab-content">
              <div className="card">
                <div className="card-header-flex">
                  <div>
                    <h3>Asset Values Over Time</h3>
                    <p className="accounts-empty" style={{ textAlign: 'left', padding: '0.2rem 0' }}>
                      Record and track date-keyed unit valuations for each asset class.
                    </p>
                  </div>
                  <div className="asset-picker-box">
                    <label htmlFor="select-asset-for-vals"><strong>Asset Class:</strong></label>
                    <select
                      id="select-asset-for-vals"
                      value={selectedAssetForValues}
                      onChange={(e) => setSelectedAssetForValues(e.target.value)}
                    >
                      {(allAssets || []).map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.label} {a.description ? `(${a.description})` : ''}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {currentSelectedAsset && (
                  <form onSubmit={handleAddValuation} className="history-valuation-form mb-4">
                    <h4>Record Unit Price for {currentSelectedAsset.label}</h4>
                    <div className="form-row-2">
                      <div className="form-group">
                        <label htmlFor="history-val-amt">Unit Price (USD) *</label>
                        <div className="amount-input-wrapper">
                          <span className="currency-prefix">$</span>
                          <input
                            id="history-val-amt"
                            type="text"
                            className="amount-input"
                            placeholder="e.g. 65,000.00"
                            value={historyValueStr}
                            onChange={(e) => setHistoryValueStr(e.target.value)}
                            required
                          />
                        </div>
                      </div>
                      <div className="form-group">
                        <label htmlFor="history-val-date">Valuation Date *</label>
                        <DateInput
                          id="history-val-date"
                          value={historyDateStr}
                          onChange={setHistoryDateStr}
                        />
                      </div>
                    </div>

                    {historyError && <p className="text-danger font-sm">{historyError}</p>}

                    <div className="form-actions" style={{ marginTop: '0.75rem' }}>
                      <button type="submit" className="btn-primary" disabled={submittingHistoryVal}>
                        {submittingHistoryVal ? 'Recording...' : 'Record Valuation'}
                      </button>
                    </div>
                  </form>
                )}

                {loadingHistory ? (
                  <div className="loading-state">Loading valuations...</div>
                ) : (assetValuationHistory || []).length === 0 ? (
                  <p className="accounts-empty">No valuation history recorded for this asset class yet.</p>
                ) : (
                  <div className="table-responsive">
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
                        {(assetValuationHistory || []).map((v) => (
                          <tr key={v.id}>
                            <td>
                              <span className="snapshot-date-badge font-mono">
                                {isoToDisplayDate(v.value_date)}
                              </span>
                            </td>
                            <td className="text-right font-mono font-bold">
                              {formatUnitValue(v.value)}
                            </td>
                            <td className="text-muted font-sm">
                              {new Date(v.created_at).toLocaleString()}
                            </td>
                            <td className="text-center">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  onClick={() => {
                                    setEditingHistoryVal(v);
                                    setEditHistoryValStr(v.value.toString());
                                    setEditHistoryDateStr(v.value_date);
                                  }}
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action btn-table-action-danger"
                                  onClick={() => handleDeleteValuation(v.id)}
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

          {/* ========================================================================= */}
          {/* TAB 3: Asset Storage (Holdings) */}
          {/* ========================================================================= */}
          {showStorageTab && activeTab === 'storage' && (
            <div className="asset-tab-content">
              {/* Add Storage Location Form */}
              <div className="card new-holding-card">
                <h3>Add Asset Storage / Holding</h3>
                <p className="accounts-empty" style={{ textAlign: 'left', padding: '0.2rem 0' }}>
                  Create a labeled storage location, hardware key, or vault in this account (e.g. &quot;red&quot;, &quot;blue&quot;, &quot;attic box&quot;, &quot;silver mattress&quot;).
                </p>

                <form onSubmit={handleCreateStore} className="form-grid">
                  <div className="form-row-2">
                    <div className="form-group">
                      <label htmlFor="new-store-asset">Asset Class *</label>
                      <select
                        id="new-store-asset"
                        value={newStoreAssetId}
                        onChange={(e) => setNewStoreAssetId(e.target.value)}
                        required
                      >
                        {(allAssets || []).map((a) => (
                          <option key={a.id} value={a.id}>
                            {a.label} {a.description ? `(${a.description})` : ''}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div className="form-group">
                      <label htmlFor="new-store-label">Storage Label *</label>
                      <input
                        id="new-store-label"
                        type="text"
                        placeholder="e.g. red, blue, attic box, silver mattress"
                        value={newStoreLabel}
                        onChange={(e) => setNewStoreLabel(e.target.value)}
                        required
                      />
                    </div>
                  </div>

                  <div className="form-row-2">
                    <div className="form-group">
                      <label htmlFor="new-store-desc">Description (Optional)</label>
                      <input
                        id="new-store-desc"
                        type="text"
                        placeholder="e.g. Cold storage private key in office safe"
                        value={newStoreDesc}
                        onChange={(e) => setNewStoreDesc(e.target.value)}
                      />
                    </div>
                    <div className="form-row-2">
                      <div className="form-group">
                        <label htmlFor="new-store-init-qty">Initial Quantity (Optional)</label>
                        <input
                          id="new-store-init-qty"
                          type="text"
                          placeholder="e.g. 0.2"
                          value={newStoreInitQty}
                          onChange={(e) => setNewStoreInitQty(e.target.value)}
                        />
                      </div>
                      <div className="form-group">
                        <label htmlFor="new-store-init-date">Initial Date</label>
                        <DateInput
                          id="new-store-init-date"
                          value={newStoreInitDate}
                          onChange={setNewStoreInitDate}
                        />
                      </div>
                    </div>
                  </div>

                  {createStoreError && <p className="text-danger font-sm">{createStoreError}</p>}

                  <div className="form-actions" style={{ marginTop: '0.75rem' }}>
                    <button type="submit" className="btn-primary" disabled={creatingStore}>
                      {creatingStore ? 'Creating Storage...' : 'Create Storage Location'}
                    </button>
                  </div>
                </form>
              </div>

              {/* Storages Table */}
              <div className="card">
                <h3>Asset Storage Locations</h3>
                {(stores || []).length === 0 ? (
                  <div className="empty-holdings-prompt">
                    <p>No storage locations in this account yet.</p>
                  </div>
                ) : (
                  <div className="table-responsive holdings-table-container" ref={holdingsContainerRef}>
                    <table
                      className={`holdings-table current-holdings-table ${hideDescription ? 'hide-col-desc' : ''} ${hideActions ? 'hide-col-actions' : ''}`}
                      ref={holdingsTableRef}
                    >
                      <thead>
                        <tr>
                          <th>Storage Label</th>
                          <th>Asset Class</th>
                          <th className="col-description">Description</th>
                          <th className="text-right">Latest Quantity</th>
                          <th>Last Record Date</th>
                          <th className="text-right">Unit Value</th>
                          <th className="text-right">Total Worth</th>
                          <th className="text-center col-actions">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(stores || []).map((s) => (
                          <tr key={s.id}>
                            <td>
                              <span className="asset-tag" style={{ backgroundColor: '#e0f2fe', color: '#0369a1', borderColor: '#bae6fd' }}>
                                {s.label}
                              </span>
                            </td>
                            <td>
                              <span className="asset-tag">{s.asset_label}</span>
                            </td>
                            <td className="col-description" title={s.description || ''}>
                              {s.description || '—'}
                            </td>
                            <td className="text-right font-mono font-bold">
                              {formatQuantity(s.latest_quantity)}
                            </td>
                            <td>
                              {s.latest_record_date ? (
                                <span className="snapshot-date-badge font-mono">
                                  {isoToDisplayDate(s.latest_record_date)}
                                </span>
                              ) : (
                                <span className="text-muted">No records</span>
                              )}
                            </td>
                            <td className="text-right font-mono">
                              {s.latest_value > 0 ? formatUnitValue(s.latest_value) : <span className="text-muted">Unvalued</span>}
                            </td>
                            <td className="text-right font-mono font-bold text-success">
                              {formatCurrency(s.total_value)}
                            </td>
                            <td className="text-center col-actions">
                              <div className="table-actions-row">
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  title="Manage Quantity Records by Date"
                                  onClick={() => {
                                    setSelectedStoreIdForRecords(s.id);
                                    setActiveTab('records');
                                  }}
                                >
                                  Records
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action"
                                  onClick={() => {
                                    setEditingStore(s);
                                    setEditStoreLabel(s.label);
                                    setEditStoreDesc(s.description);
                                    setEditStoreAssetId(s.asset_id);
                                    setEditStoreError('');
                                  }}
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn-table-action btn-table-action-danger"
                                  onClick={() => handleDeleteStore(s)}
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

          {/* ========================================================================= */}
          {/* TAB 4: Quantity Records */}
          {/* ========================================================================= */}
          {showRecordsTab && activeTab === 'records' && (
            <div className="asset-tab-content">
              <div className="card">
                <div className="card-header-flex">
                  <div>
                    <h3>Storage Quantity Records by Date</h3>
                    <p className="accounts-empty" style={{ textAlign: 'left', padding: '0.2rem 0' }}>
                      Track quantities over time for each storage location (e.g. buying or selling amounts across dates).
                    </p>
                  </div>
                  {(stores || []).length > 0 && (
                    <div className="asset-picker-box">
                      <label htmlFor="select-store-for-recs"><strong>Storage Location:</strong></label>
                      <select
                        id="select-store-for-recs"
                        value={selectedStoreIdForRecords}
                        onChange={(e) => setSelectedStoreIdForRecords(e.target.value)}
                      >
                        {(stores || []).map((s) => (
                          <option key={s.id} value={s.id}>
                            {s.label} ({s.asset_label})
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>

                {(stores || []).length === 0 ? (
                  <div className="empty-holdings-prompt">
                    <p>No storage locations available. Please create a storage location in the Asset Storage tab first.</p>
                    <button
                      type="button"
                      className="btn-primary"
                      onClick={() => setActiveTab('storage')}
                      style={{ marginTop: '1rem' }}
                    >
                      Go to Asset Storage
                    </button>
                  </div>
                ) : currentSelectedStore ? (
                  <>
                    {/* Add Quantity Record Form */}
                    <form onSubmit={handleAddStoreRecord} className="history-valuation-form mb-4">
                      <h4>
                        Record Quantity for {currentSelectedStore.label} ({currentSelectedStore.asset_label})
                      </h4>
                      <div className="form-row-2">
                        <div className="form-group">
                          <label htmlFor="new-rec-qty">Held Quantity *</label>
                          <input
                            id="new-rec-qty"
                            type="text"
                            placeholder="e.g. 0.35, 12.5"
                            value={newRecordQtyStr}
                            onChange={(e) => setNewRecordQtyStr(e.target.value)}
                            required
                          />
                        </div>
                        <div className="form-group">
                          <label htmlFor="new-rec-date">Record Date *</label>
                          <DateInput
                            id="new-rec-date"
                            value={newRecordDateStr}
                            onChange={setNewRecordDateStr}
                          />
                        </div>
                      </div>

                      {addRecordError && <p className="text-danger font-sm">{addRecordError}</p>}

                      <div className="form-actions" style={{ marginTop: '0.75rem' }}>
                        <button type="submit" className="btn-primary" disabled={addingRecord}>
                          {addingRecord ? 'Recording...' : 'Save Quantity Record'}
                        </button>
                      </div>
                    </form>

                    {/* Table of Records */}
                    {loadingRecords ? (
                      <div className="loading-state">Loading quantity records...</div>
                    ) : (storeRecords || []).length === 0 ? (
                      <p className="accounts-empty">No quantity records for this storage location yet.</p>
                    ) : (
                      <div className="table-responsive">
                        <table className="holdings-table">
                          <thead>
                            <tr>
                              <th>Record Date</th>
                              <th className="text-right">Quantity</th>
                              <th>Timestamp</th>
                              <th className="text-center">Actions</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(storeRecords || []).map((r) => (
                              <tr key={r.id}>
                                <td>
                                  <span className="snapshot-date-badge font-mono">
                                    {isoToDisplayDate(r.record_date)}
                                  </span>
                                </td>
                                <td className="text-right font-mono font-bold">
                                  {formatQuantity(r.quantity)}
                                </td>
                                <td className="text-muted font-sm">
                                  {new Date(r.created_at).toLocaleString()}
                                </td>
                                <td className="text-center">
                                  <div className="table-actions-row">
                                    <button
                                      type="button"
                                      className="btn-table-action"
                                      onClick={() => {
                                        setEditingRecord(r);
                                        setEditRecordQtyStr(r.quantity.toString());
                                        setEditRecordDateStr(r.record_date);
                                        setEditRecordError('');
                                      }}
                                    >
                                      Edit
                                    </button>
                                    <button
                                      type="button"
                                      className="btn-table-action btn-table-action-danger"
                                      onClick={() => handleDeleteStoreRecord(r.id)}
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
                  </>
                ) : null}
              </div>
            </div>
          )}

          {/* ========================================================================= */}
          {/* TAB 5: Total Account Value Trend */}
          {/* ========================================================================= */}
          {showTrendTab && activeTab === 'trend' && (
            <div className="asset-tab-content">
              <div className="card">
                <AssetTrendGraph points={trendPoints} />
              </div>
            </div>
          )}
        </>
      )}

      {/* MODAL: Rename / Edit Asset Class */}
      {editingAsset && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <div className="modal-header">
              <h3>Rename / Edit Asset Class</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingAsset(null)}
              >
                ×
              </button>
            </div>
            <form onSubmit={handleUpdateAsset}>
              <div className="form-group mb-3">
                <label htmlFor="edit-asset-label">Class Symbol / Label *</label>
                <input
                  id="edit-asset-label"
                  ref={editAssetInputRef}
                  type="text"
                  value={editAssetLabel}
                  onChange={(e) => setEditAssetLabel(e.target.value)}
                  required
                />
              </div>
              <div className="form-group mb-3">
                <label htmlFor="edit-asset-desc">Description</label>
                <input
                  id="edit-asset-desc"
                  type="text"
                  value={editAssetDesc}
                  onChange={(e) => setEditAssetDesc(e.target.value)}
                />
              </div>

              {editAssetError && <p className="text-danger font-sm mb-3">{editAssetError}</p>}

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingAsset(null)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary" disabled={updatingAsset}>
                  {updatingAsset ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Edit Storage Location */}
      {editingStore && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <div className="modal-header">
              <h3>Edit Storage Location</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingStore(null)}
              >
                ×
              </button>
            </div>
            <form onSubmit={handleUpdateStore}>
              <div className="form-group mb-3">
                <label htmlFor="edit-store-label">Storage Label *</label>
                <input
                  id="edit-store-label"
                  type="text"
                  value={editStoreLabel}
                  onChange={(e) => setEditStoreLabel(e.target.value)}
                  required
                />
              </div>
              <div className="form-group mb-3">
                <label htmlFor="edit-store-asset">Asset Class</label>
                <select
                  id="edit-store-asset"
                  value={editStoreAssetId}
                  onChange={(e) => setEditStoreAssetId(e.target.value)}
                >
                  {allAssets.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.label} {a.description ? `(${a.description})` : ''}
                    </option>
                  ))}
                </select>
              </div>
              <div className="form-group mb-3">
                <label htmlFor="edit-store-desc">Description</label>
                <input
                  id="edit-store-desc"
                  type="text"
                  value={editStoreDesc}
                  onChange={(e) => setEditStoreDesc(e.target.value)}
                />
              </div>

              {editStoreError && <p className="text-danger font-sm mb-3">{editStoreError}</p>}

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingStore(null)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary" disabled={updatingStore}>
                  {updatingStore ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Edit Quantity Record */}
      {editingRecord && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <div className="modal-header">
              <h3>Edit Quantity Record</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingRecord(null)}
              >
                ×
              </button>
            </div>
            <form onSubmit={handleUpdateStoreRecord}>
              <div className="form-group mb-3">
                <label htmlFor="edit-rec-qty">Held Quantity *</label>
                <input
                  id="edit-rec-qty"
                  type="text"
                  value={editRecordQtyStr}
                  onChange={(e) => setEditRecordQtyStr(e.target.value)}
                  required
                />
              </div>
              <div className="form-group mb-3">
                <label htmlFor="edit-rec-date">Record Date *</label>
                <DateInput
                  id="edit-rec-date"
                  value={editRecordDateStr}
                  onChange={setEditRecordDateStr}
                />
              </div>

              {editRecordError && <p className="text-danger font-sm mb-3">{editRecordError}</p>}

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingRecord(null)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary" disabled={updatingRecord}>
                  {updatingRecord ? 'Saving...' : 'Save Record'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Edit Valuation */}
      {editingHistoryVal && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <div className="modal-header">
              <h3>Edit Valuation</h3>
              <button
                type="button"
                className="btn-close"
                onClick={() => setEditingHistoryVal(null)}
              >
                ×
              </button>
            </div>
            <form onSubmit={handleUpdateValuation}>
              <div className="form-group mb-3">
                <label htmlFor="edit-history-val-amt">Unit Price (USD) *</label>
                <div className="amount-input-wrapper">
                  <span className="currency-prefix">$</span>
                  <input
                    id="edit-history-val-amt"
                    type="text"
                    className="amount-input"
                    value={editHistoryValStr}
                    onChange={(e) => setEditHistoryValStr(e.target.value)}
                    required
                  />
                </div>
              </div>
              <div className="form-group mb-3">
                <label htmlFor="edit-history-val-date">Valuation Date *</label>
                <DateInput
                  id="edit-history-val-date"
                  value={editHistoryDateStr}
                  onChange={setEditHistoryDateStr}
                />
              </div>

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setEditingHistoryVal(null)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Save Valuation
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
