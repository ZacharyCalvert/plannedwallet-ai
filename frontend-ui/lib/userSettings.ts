import type { PinnedBalanceRef } from './types.ts';

export function applyAccountOrder<T extends { id: string }>(
  accounts: T[],
  accountOrder: string[]
): T[] {
  if (!accountOrder || accountOrder.length === 0) {
    return accounts;
  }
  const rank = new Map<string, number>();
  accountOrder.forEach((id, idx) => {
    if (!rank.has(id)) {
      rank.set(id, idx);
    }
  });
  return [...accounts].sort((a, b) => {
    const rankA = rank.get(a.id);
    const rankB = rank.get(b.id);
    if (rankA !== undefined && rankB !== undefined) {
      return rankA - rankB;
    }
    if (rankA !== undefined) return -1;
    if (rankB !== undefined) return 1;
    return 0;
  });
}

export function reorderItems<T>(items: T[], fromIndex: number, toIndex: number): T[] {
  if (
    fromIndex < 0 ||
    toIndex < 0 ||
    fromIndex >= items.length ||
    toIndex >= items.length ||
    fromIndex === toIndex
  ) {
    return items;
  }
  const next = [...items];
  const [moved] = next.splice(fromIndex, 1);
  next.splice(toIndex, 0, moved);
  return next;
}

export function isPinnedBalance(
  pinned: PinnedBalanceRef[],
  kind: 'account' | 'perspective',
  id: string
): boolean {
  return pinned.some((ref) => ref.kind === kind && ref.id === id);
}

export function togglePinnedBalance(
  pinned: PinnedBalanceRef[],
  ref: PinnedBalanceRef
): PinnedBalanceRef[] {
  if (isPinnedBalance(pinned, ref.kind, ref.id)) {
    return pinned.filter((item) => !(item.kind === ref.kind && item.id === ref.id));
  }
  return [...pinned, ref];
}
