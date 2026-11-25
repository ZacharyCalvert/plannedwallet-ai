'use client';

import { Account, RecurringTransaction } from '../lib/types';
import {
  getBillDaysUntilDue,
  getBillDisplayName,
  getLocalTodayIso,
  shouldShowBillReminder,
} from '../lib/bills';
import { isoToDisplayDate } from './DateInput';

interface BillReminderBannersProps {
  transactions: RecurringTransaction[];
  accounts: Account[];
  hiddenIds: Record<string, boolean>;
  applyingId: string | null;
  onApply: (tx: RecurringTransaction) => void;
  onMarkPaid?: (tx: RecurringTransaction) => void;
  onHide: (id: string) => void;
}

function formatCurrency(val: number): string {
  const isNegative = val < 0;
  const formatted = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(Math.abs(val));
  return isNegative ? `-${formatted}` : formatted;
}

function getDueBadgeText(days: number): string {
  if (days < 0) {
    const overdueDays = Math.abs(days);
    return `Overdue by ${overdueDays} day${overdueDays === 1 ? '' : 's'}`;
  }
  if (days === 0) return 'Due Today';
  return `Due in ${days} day${days === 1 ? '' : 's'}`;
}

export function BillReminderBanners({
  transactions,
  accounts,
  hiddenIds,
  applyingId,
  onApply,
  onMarkPaid,
  onHide,
}: BillReminderBannersProps) {
  const todayIso = getLocalTodayIso();
  const visible = transactions.filter(
    (tx) => !hiddenIds[tx.id] && shouldShowBillReminder(tx, todayIso)
  );
  if (visible.length === 0) {
    return null;
  }

  const accountNames = new Map<string, string>();
  for (const acc of accounts) {
    accountNames.set(acc.id, acc.name);
  }

  return (
    <div className="bill-reminders-stack" role="region" aria-label="Recurring Reminders">
      {visible.map((tx) => {
        const days = getBillDaysUntilDue(tx.next_due_date, todayIso) ?? 0;
        const isOverdue = days < 0;
        const isIncome = tx.amount > 0;
        const isApplying = applyingId === tx.id;
        const linkedName = tx.account_id ? accountNames.get(tx.account_id) : null;
        const displayName = getBillDisplayName(tx);
        return (
          <div
            key={tx.id}
            className={`bill-reminder-banner ${
              isOverdue
                ? 'bill-reminder-overdue'
                : isIncome
                ? 'bill-reminder-income-upcoming'
                : 'bill-reminder-upcoming'
            }`}
            role="status"
          >
            <div className="bill-reminder-content">
              <span className={`bill-reminder-pill ${isOverdue ? 'overdue' : 'upcoming'}`}>
                {getDueBadgeText(days)}
              </span>
              <strong>{displayName}</strong>
              {displayName !== tx.entity && (
                <span className="bill-reminder-date">({tx.entity})</span>
              )}
              <span className="bill-reminder-amount">{formatCurrency(tx.amount)}</span>
              {tx.next_due_date && (
                <span className="bill-reminder-date">
                  Due {isoToDisplayDate(tx.next_due_date)}
                </span>
              )}
              {linkedName && (
                <span className="bill-reminder-account">&middot; Account: {linkedName}</span>
              )}
            </div>
            <div className="bill-reminder-actions">
              {tx.account_id ? (
                <button
                  type="button"
                  className="btn-primary btn-sm btn-reminder-action"
                  disabled={isApplying}
                  onClick={() => onApply(tx)}
                >
                  {isApplying ? 'Applying...' : 'Apply'}
                </button>
              ) : (
                onMarkPaid && (
                  <button
                    type="button"
                    className="btn-paid btn-sm btn-reminder-action"
                    disabled={isApplying}
                    onClick={() => onMarkPaid(tx)}
                  >
                    {isApplying ? 'Saving...' : 'Paid'}
                  </button>
                )
              )}
              <button
                type="button"
                className="btn-secondary btn-sm"
                disabled={isApplying}
                onClick={() => onHide(tx.id)}
              >
                Hide
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
