'use client';

import { useEffect } from 'react';
import { MAINTENANCE_EDIT_WARNING } from '../lib/maintenance';

interface MaintenanceWarningModalProps {
  message: string;
  onClose: () => void;
}

export function MaintenanceWarningModal({ message, onClose }: MaintenanceWarningModalProps) {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  return (
    <div
      className="modal-backdrop maintenance-warning-backdrop"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="maintenance-warning-title"
      aria-describedby="maintenance-warning-desc"
      onClick={onClose}
    >
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3 id="maintenance-warning-title">Maintenance Mode Active</h3>
          <button
            type="button"
            className="btn-close"
            onClick={onClose}
            aria-label="Close maintenance warning"
          >
            &times;
          </button>
        </div>
        <div className="modal-body-stack">
          <p id="maintenance-warning-desc" className="maintenance-warning-text">
            {message || MAINTENANCE_EDIT_WARNING}
          </p>
          <p className="text-muted text-sm">
            Planned Wallet is currently in read-only maintenance mode. Viewing existing data remains available.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <button type="button" className="btn-primary btn-sm" onClick={onClose} autoFocus>
              OK
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
