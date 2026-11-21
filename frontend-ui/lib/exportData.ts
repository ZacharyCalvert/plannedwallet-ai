import {
  fetchAccountRecords,
  fetchAccountSnapshots,
  fetchAssetStoreRecords,
  fetchAssetStores,
  fetchCategories,
} from './api';
import {
  buildAccountCsvFilename,
  buildCategoryNameMap,
  formatAssetStoresCsv,
  formatCheckingRecordsCsv,
  formatSnapshotsCsv,
} from './exportFormatters';
import type { Account, AssetStoreRecord, Category } from './types';
import { createZipArchive, ZipEntry } from './zip';

/**
 * Fetches all entered data for a single account and formats it as a CSV string.
 */
export async function exportAccountCsvContent(
  token: string,
  account: Account,
  preloadedCategories?: Category[]
): Promise<string> {
  const normalizedType = (account.type || '').toLowerCase();
  if (normalizedType === 'snapshot') {
    const snapshots = await fetchAccountSnapshots(token, account.id);
    return formatSnapshotsCsv(snapshots);
  }
  if (normalizedType === 'asset' || normalizedType === 'assets') {
    const stores = await fetchAssetStores(token, account.id);
    const recordEntries = await Promise.all(
      stores.map(async (store) => ({
        id: store.id,
        records: await fetchAssetStoreRecords(token, account.id, store.id),
      }))
    );
    const recordsByStore: Record<string, AssetStoreRecord[]> = {};
    for (const entry of recordEntries) {
      recordsByStore[entry.id] = entry.records;
    }
    return formatAssetStoresCsv(stores, recordsByStore);
  }

  const [records, categories] = await Promise.all([
    fetchAccountRecords(token, account.id),
    preloadedCategories ? Promise.resolve(preloadedCategories) : fetchCategories(token),
  ]);
  return formatCheckingRecordsCsv(records, buildCategoryNameMap(categories));
}

function triggerBrowserDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Downloads a single account's entered data as a CSV file.
 */
export async function downloadAccountCsv(token: string, account: Account): Promise<void> {
  const csv = await exportAccountCsvContent(token, account);
  const filename = buildAccountCsvFilename(account);
  triggerBrowserDownload(new Blob([csv], { type: 'text/csv;charset=utf-8' }), filename);
}

/**
 * Downloads a ZIP archive containing one CSV file per account.
 */
export async function downloadAllAccountsZip(
  token: string,
  accounts: Account[]
): Promise<void> {
  const categories = await fetchCategories(token);
  const usedNames = new Set<string>();
  const entries: ZipEntry[] = await Promise.all(
    accounts.map(async (account) => {
      const csv = await exportAccountCsvContent(token, account, categories);
      const filename = buildAccountCsvFilename(account, usedNames);
      return { filename, data: csv };
    })
  );
  const archive = createZipArchive(entries);
  const dateSuffix = new Date().toISOString().slice(0, 10);
  const zipBytes = new Uint8Array(archive.byteLength);
  zipBytes.set(archive);
  triggerBrowserDownload(
    new Blob([zipBytes], { type: 'application/zip' }),
    `planned-wallet-export-${dateSuffix}.zip`
  );
}
