'use client';

import { useEffect, useState } from 'react';
import {
  fetchAccountRecords,
  fetchAccountSnapshots,
  fetchAssetAccountSummary,
  fetchPerspectiveView,
} from '../lib/api';
import { Account, Perspective, PinnedBalanceRef } from '../lib/types';

interface PinnedBalancesBarProps {
  token: string;
  pinnedBalances: PinnedBalanceRef[];
  accounts: Account[];
  perspectives: Perspective[];
  refreshVersion: number;
  onSelectAccount: (accountId: string) => void;
  onSelectPerspective: (perspectiveId: string) => void;
  onUnpin?: (ref: PinnedBalanceRef) => void;
}

interface ResolvedPinnedBalance {
  ref: PinnedBalanceRef;
  name: string;
  typeLabel: string;
  balance: number | null;
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
  return isNegative ? `-${formatted}` : formatted;
}

async function resolveAccountExpectedBalance(token: string, acc: Account): Promise<number> {
  const t = (acc.type || '').toLowerCase();
  if (t === 'snapshot') {
    const snaps = await fetchAccountSnapshots(token, acc.id).catch(() => []);
    if (snaps.length === 0) return 0;
    const latest = [...snaps].sort((a, b) => {
      const d = b.snapshot_date.slice(0, 10).localeCompare(a.snapshot_date.slice(0, 10));
      return d !== 0 ? d : b.created_at.localeCompare(a.created_at);
    })[0];
    return latest.value;
  }
  if (t === 'asset' || t === 'assets') {
    const summary = await fetchAssetAccountSummary(token, acc.id).catch(() => null);
    return summary ? summary.total_value : 0;
  }
  const records = await fetchAccountRecords(token, acc.id).catch(() => []);
  return records.reduce((sum, r) => sum + r.value, 0);
}

export function PinnedBalancesBar({
  token,
  pinnedBalances,
  accounts,
  perspectives,
  refreshVersion,
  onSelectAccount,
  onSelectPerspective,
  onUnpin,
}: PinnedBalancesBarProps) {
  const [items, setItems] = useState<ResolvedPinnedBalance[]>([]);

  useEffect(() => {
    if (pinnedBalances.length === 0) {
      setItems([]);
      return;
    }
    let cancelled = false;
    Promise.all(
      pinnedBalances.map(async (ref): Promise<ResolvedPinnedBalance | null> => {
        if (ref.kind === 'account') {
          const acc = accounts.find((a) => a.id === ref.id);
          if (!acc) return null;
          const balance = await resolveAccountExpectedBalance(token, acc);
          return { ref, name: acc.name, typeLabel: acc.type, balance };
        }
        const p = perspectives.find((item) => item.id === ref.id);
        if (!p) return null;
        const view = await fetchPerspectiveView(token, p.id).catch(() => null);
        return {
          ref,
          name: p.name,
          typeLabel: 'perspective',
          balance: view ? view.current_total_value : null,
        };
      })
    ).then((resolved) => {
      if (!cancelled) {
        setItems(resolved.filter((item): item is ResolvedPinnedBalance => item !== null));
      }
    });
    return () => {
      cancelled = true;
    };
  }, [token, pinnedBalances, accounts, perspectives, refreshVersion]);

  if (items.length === 0) {
    return null;
  }

  return (
    <div className="pinned-balances-bar" aria-label="Pinned Expected Balances">
      {items.map((item) => {
        const key = `${item.ref.kind}:${item.ref.id}`;
        const bal = item.balance;
        const colorClass =
          bal === null ? 'text-muted' : bal < 0 ? 'text-danger' : bal > 0 ? 'text-success' : '';
        return (
          <div
            key={key}
            className="pinned-balance-chip"
            role="button"
            tabIndex={0}
            onClick={() =>
              item.ref.kind === 'account'
                ? onSelectAccount(item.ref.id)
                : onSelectPerspective(item.ref.id)
            }
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                if (item.ref.kind === 'account') {
                  onSelectAccount(item.ref.id);
                } else {
                  onSelectPerspective(item.ref.id);
                }
              }
            }}
          >
            <div className="pinned-balance-chip-meta">
              <span className="pinned-balance-name">{item.name}</span>
              <span className="pinned-balance-type">{item.typeLabel}</span>
            </div>
            <span className={`pinned-balance-value ${colorClass}`}>
              {bal === null ? '—' : formatCurrency(bal)}
            </span>
            {onUnpin && (
              <button
                type="button"
                className="pinned-balance-unpin"
                aria-label={`Unpin ${item.name}`}
                title={`Unpin ${item.name}`}
                onClick={(e) => {
                  e.stopPropagation();
                  onUnpin(item.ref);
                }}
              >
                ×
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
