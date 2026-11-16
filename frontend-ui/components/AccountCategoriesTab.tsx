'use client';

import { useState, useMemo, useRef, useEffect } from 'react';
import { AccountRecord, Category } from '../lib/types';
import { createCategory } from '../lib/api';
import { findCategoryByNameCaseInsensitive, getRecordsForCategory } from '../lib/accountCategories';
import { DeleteCategoryModal } from './DeleteCategoryModal';

interface AccountCategoriesTabProps {
  token: string;
  records: AccountRecord[];
  categories: Category[];
  onCategoryCreated: (cat: Category) => void;
  onCategoryDeleted: (deletedId: string, transitionToCategoryId: string | null) => void;
}

export function AccountCategoriesTab({
  token,
  records,
  categories,
  onCategoryCreated,
  onCategoryDeleted,
}: AccountCategoriesTabProps) {
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');
  const [deletingCategory, setDeletingCategory] = useState<Category | null>(null);
  const categoryInputRef = useRef<HTMLInputElement>(null);
  const [focusCounter, setFocusCounter] = useState(0);

  const focusCategoryInput = () => {
    categoryInputRef.current?.focus();
  };

  useEffect(() => {
    if (focusCounter > 0 && !creating) {
      focusCategoryInput();
      const raf = requestAnimationFrame(focusCategoryInput);
      const timer = setTimeout(focusCategoryInput, 50);
      return () => {
        cancelAnimationFrame(raf);
        clearTimeout(timer);
      };
    }
  }, [focusCounter, creating]);

  const countByCategory = useMemo(() => {
    const counts = new Map<string, number>();
    for (const r of records) {
      if (r.category_id) {
        counts.set(r.category_id, (counts.get(r.category_id) || 0) + 1);
      }
    }
    return counts;
  }, [records]);

  const defaultCategories = useMemo(
    () => categories.filter((c) => c.is_default),
    [categories]
  );

  const customCategories = useMemo(
    () => categories.filter((c) => !c.is_default),
    [categories]
  );

  const associatedRecords = useMemo(
    () => (deletingCategory ? getRecordsForCategory(records, deletingCategory.id) : []),
    [deletingCategory, records]
  );

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateError('');
    const trimmed = newName.trim();
    if (!trimmed) {
      setCreateError('Category name is required.');
      setFocusCounter((c) => c + 1);
      return;
    }
    const existing = findCategoryByNameCaseInsensitive(categories, trimmed);
    if (existing) {
      setCreateError(`Category "${existing.name}" already exists.`);
      setFocusCounter((c) => c + 1);
      return;
    }

    setCreating(true);
    try {
      const created = await createCategory(token, trimmed);
      onCategoryCreated(created);
      setNewName('');
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create category');
    } finally {
      setCreating(false);
      setFocusCounter((c) => c + 1);
    }
  };

  return (
    <div className="categories-tab-section">
      <section className="new-record-section">
        <div className="section-title-row">
          <h3>Create Custom Category</h3>
          <span className="section-subtitle">Add a custom transaction category for your accounts</span>
        </div>
        {createError && <div className="alert-error mb-3">{createError}</div>}
        <form onSubmit={handleCreate} className="inline-category-create-form">
          <input
            ref={categoryInputRef}
            type="text"
            placeholder="e.g. Dining Out, Subscriptions, Home Repair"
            value={newName}
            maxLength={100}
            onChange={(e) => setNewName(e.target.value)}
            disabled={creating}
          />
          <button
            type="submit"
            className="btn-primary btn-sm"
            disabled={creating}
            onMouseDown={(e) => e.preventDefault()}
          >
            {creating ? 'Adding...' : 'Add Category'}
          </button>
        </form>
      </section>

      <div className="categories-lists-grid">
        <section className="categories-group-card">
          <div className="section-title-row">
            <h3>Default Categories</h3>
            <span className="records-count-badge">{defaultCategories.length}</span>
          </div>
          <div className="categories-table-list">
            {defaultCategories.map((cat) => {
              const count = countByCategory.get(cat.id) || 0;
              return (
                <div key={cat.id} className="category-list-row">
                  <div className="category-list-info">
                    <span className="font-semibold">{cat.name}</span>
                    <span className="category-badge-default">Default</span>
                  </div>
                  <span className="text-muted text-sm">
                    {count} {count === 1 ? 'record' : 'records'}
                  </span>
                </div>
              );
            })}
          </div>
        </section>

        <section className="categories-group-card">
          <div className="section-title-row">
            <h3>Custom Categories</h3>
            <span className="records-count-badge">{customCategories.length}</span>
          </div>
          {customCategories.length === 0 ? (
            <p className="text-muted text-sm">No custom categories yet. Create one above.</p>
          ) : (
            <div className="categories-table-list">
              {customCategories.map((cat) => {
                const count = countByCategory.get(cat.id) || 0;
                return (
                  <div key={cat.id} className="category-list-row">
                    <div className="category-list-info">
                      <span className="font-semibold">{cat.name}</span>
                      <span className="text-muted text-sm">
                        ({count} {count === 1 ? 'record' : 'records'} in account)
                      </span>
                    </div>
                    <button
                      type="button"
                      className="btn-danger-outline btn-sm"
                      onClick={() => setDeletingCategory(cat)}
                    >
                      Delete
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </section>
      </div>

      <DeleteCategoryModal
        isOpen={Boolean(deletingCategory)}
        onClose={() => setDeletingCategory(null)}
        token={token}
        category={deletingCategory}
        allCategories={categories}
        associatedRecords={associatedRecords}
        onDeleted={(deletedId, transitionId) => {
          setDeletingCategory(null);
          onCategoryDeleted(deletedId, transitionId);
        }}
      />
    </div>
  );
}
