import { Account, Category, RecurringTransaction, RecurringTransactionRequest } from '../lib/types';
import { RecurringSchedulePane } from './RecurringSchedulePane';

interface BillsPaneProps {
  bills: RecurringTransaction[];
  accounts: Account[];
  categories: Category[];
  applyingId: string | null;
  onCreateBill: (payload: RecurringTransactionRequest) => Promise<void>;
  onUpdateBill: (id: string, payload: RecurringTransactionRequest) => Promise<void>;
  onDeleteBill: (id: string) => Promise<void>;
  onApplyBill: (bill: RecurringTransaction) => Promise<void>;
}

export function BillsPane({
  bills,
  accounts,
  categories,
  applyingId,
  onCreateBill,
  onUpdateBill,
  onDeleteBill,
  onApplyBill,
}: BillsPaneProps) {
  return (
    <RecurringSchedulePane
      kind="bill"
      items={bills}
      accounts={accounts}
      categories={categories}
      applyingId={applyingId}
      onCreate={onCreateBill}
      onUpdate={onUpdateBill}
      onDelete={onDeleteBill}
      onApply={onApplyBill}
    />
  );
}
