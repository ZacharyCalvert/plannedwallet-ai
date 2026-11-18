import { Account, Category, RecurringTransaction, RecurringTransactionRequest } from '../lib/types';
import { RecurringSchedulePane } from './RecurringSchedulePane';

interface IncomesPaneProps {
  incomes: RecurringTransaction[];
  accounts: Account[];
  categories: Category[];
  applyingId: string | null;
  onCreateIncome: (payload: RecurringTransactionRequest) => Promise<void>;
  onUpdateIncome: (id: string, payload: RecurringTransactionRequest) => Promise<void>;
  onDeleteIncome: (id: string) => Promise<void>;
  onApplyIncome: (income: RecurringTransaction) => Promise<void>;
}

export function IncomesPane({
  incomes,
  accounts,
  categories,
  applyingId,
  onCreateIncome,
  onUpdateIncome,
  onDeleteIncome,
  onApplyIncome,
}: IncomesPaneProps) {
  return (
    <RecurringSchedulePane
      kind="income"
      items={incomes}
      accounts={accounts}
      categories={categories}
      applyingId={applyingId}
      onCreate={onCreateIncome}
      onUpdate={onUpdateIncome}
      onDelete={onDeleteIncome}
      onApply={onApplyIncome}
    />
  );
}
