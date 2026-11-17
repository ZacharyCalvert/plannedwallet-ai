'use client';

import { Account, Bill } from '../lib/types';
import {
  getBillDaysUntilDue,
  getBillDisplayName,
  getLocalTodayIso,
  shouldShowBillReminder,
} from '../lib/bills';
import { isoToDisplayDate } from './DateInput';

interface BillReminderBannersProps {
  bills: Bill[];
  accounts: Account[];
  hiddenIds: Record<string, boolean>;
  applyingId: string | null;
  onApply: (bill: Bill) => void;
  onHide: (billId: string) => void;
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
  bills,
  accounts,
  hiddenIds,
  applyingId,
  onApply,
  onHide,
}: BillReminderBannersProps) {
  const todayIso = getLocalTodayIso();
  const visible = bills.filter((b) => !hiddenIds[b.id] && shouldShowBillReminder(b, todayIso));
  if (visible.length === 0) {
    return null;
  }

  const accountNames = new Map<string, string>();
  for (const acc of accounts) {
    accountNames.set(acc.id, acc.name);
  }

  return (
    <div className="bill-reminders-stack" role="region" aria-label="Bill Reminders">
      {visible.map((bill) => {
        const days = getBillDaysUntilDue(bill.next_due_date, todayIso) ?? 0;
        const isOverdue = days < 0;
        const isApplying = applyingId === bill.id;
        const linkedName = bill.account_id ? accountNames.get(bill.account_id) : null;
        return (
          <div
            key={bill.id}
            className={`bill-reminder-banner ${isOverdue ? 'bill-reminder-overdue' : 'bill-reminder-upcoming'}`}
            role="status"
          >
            <div className="bill-reminder-info">
              <span className={`bill-reminder-pill ${isOverdue ? 'overdue' : 'upcoming'}`}>
                {getDueBadgeText(days)}
              </span>
              <strong>{getBillDisplayName(bill)}</strong>
              {bill.name && bill.name.trim() !== '' && bill.name.trim() !== bill.recipient && (
                <span className="bill-reminder-date">({bill.recipient})</span>
              )}
              <span className="bill-reminder-amount">{formatCurrency(bill.amount)}</span>
              {bill.next_due_date && (
                <span className="bill-reminder-date">
                  Due {isoToDisplayDate(bill.next_due_date)}
                </span>
              )}
              {linkedName && (
                <span className="bill-reminder-account">&middot; Account: {linkedName}</span>
              )}
            </div>
            <div className="bill-reminder-actions">
              {bill.account_id && (
                <button
                  type="button"
                  className="btn-primary btn-sm"
                  disabled={isApplying}
                  onClick={() => onApply(bill)}
                >
                  {isApplying ? 'Applying...' : 'Apply'}
                </button>
              )}
              <button
                type="button"
                className="btn-secondary btn-sm"
                disabled={isApplying}
                onClick={() => onHide(bill.id)}
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
