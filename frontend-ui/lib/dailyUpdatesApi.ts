import {
  fetchAccountSnapshots,
  createAccountSnapshot,
  fetchAssetAccountSummary,
  fetchAssetValues,
  createAssetValue,
} from './api';
import { getLocalTodayIso } from './bills';
import {
  DailyUpdateItem,
  buildPendingAssetItem,
  buildPendingSnapshotItem,
  sortDailyUpdateItems,
} from './dailyUpdates';
import { Account } from './types';

export async function loadPendingDailyUpdates(
  token: string,
  accounts: Account[],
  todayIso: string = getLocalTodayIso()
): Promise<DailyUpdateItem[]> {
  const snapshotAccounts = accounts.filter(
    (acc) => (acc.type || '').toLowerCase() === 'snapshot'
  );
  const assetAccounts = accounts.filter((acc) => {
    const t = (acc.type || '').toLowerCase();
    return t === 'asset' || t === 'assets';
  });

  const [snapshotResults, assetSummaries] = await Promise.all([
    Promise.all(
      snapshotAccounts.map(async (acc) => {
        const snapshots = await fetchAccountSnapshots(token, acc.id).catch(() => []);
        return buildPendingSnapshotItem(acc, snapshots, todayIso);
      })
    ),
    Promise.all(
      assetAccounts.map(async (acc) => {
        const summary = await fetchAssetAccountSummary(token, acc.id).catch(() => null);
        return { account: acc, summary };
      })
    ),
  ]);

  const assetMap = new Map<string, { label: string; accountNames: Set<string> }>();
  for (const { account, summary } of assetSummaries) {
    if (!summary) continue;
    for (const store of summary.stores || []) {
      const existing = assetMap.get(store.asset_id);
      if (existing) {
        existing.accountNames.add(account.name);
      } else {
        assetMap.set(store.asset_id, {
          label: store.asset_label,
          accountNames: new Set([account.name]),
        });
      }
    }
    for (const asset of summary.assets || []) {
      const existing = assetMap.get(asset.asset_id);
      if (existing) {
        existing.accountNames.add(account.name);
      } else {
        assetMap.set(asset.asset_id, {
          label: asset.label,
          accountNames: new Set([account.name]),
        });
      }
    }
  }

  const assetResults = await Promise.all(
    Array.from(assetMap.entries()).map(async ([assetId, info]) => {
      const values = await fetchAssetValues(token, assetId).catch(() => []);
      return buildPendingAssetItem(
        assetId,
        info.label,
        Array.from(info.accountNames),
        values,
        todayIso
      );
    })
  );

  const combined = [...snapshotResults, ...assetResults].filter(
    (item): item is DailyUpdateItem => item !== null
  );
  return sortDailyUpdateItems(combined);
}

export async function submitDailyUpdateItem(
  token: string,
  item: DailyUpdateItem,
  value: number,
  todayIso: string
): Promise<void> {
  if (item.kind === 'snapshot') {
    await createAccountSnapshot(token, item.id, {
      value,
      snapshot_date: todayIso,
    });
    return;
  }
  await createAssetValue(token, item.id, {
    value,
    value_date: todayIso,
  });
}
