'use client';

import { useEffect, useState } from 'react';
import { STAGE_BANNER_TEXT, isStagingBrowser } from '../lib/environment';
import { MAINTENANCE_BANNER_TEXT } from '../lib/maintenance';

interface BannerProps {
  username?: string;
  onUserClick?: () => void;
  onSignOut?: () => void;
  readOnly?: boolean;
}

export function Banner({ username, onUserClick, onSignOut, readOnly = false }: BannerProps) {
  const [isStaging, setIsStaging] = useState(false);

  useEffect(() => {
    setIsStaging(isStagingBrowser());
  }, []);

  return (
    <>
      {isStaging && (
        <div className="staging-banner" role="alert">
          {STAGE_BANNER_TEXT}
        </div>
      )}
      {readOnly && (
        <div className="maintenance-banner" role="alert">
          {MAINTENANCE_BANNER_TEXT}
        </div>
      )}
      <header className="banner">
        <div className="banner-logo">
          <span className="banner-icon">💼</span>
          <span className="banner-title">Planned Wallet</span>
        </div>
        {username && onSignOut && (
          <div className="banner-user">
            <span className="banner-username">
              Signed in as:{' '}
              {onUserClick ? (
                <button
                  type="button"
                  className="banner-user-button"
                  onClick={onUserClick}
                  title="View user details"
                  aria-label={`View user details for ${username}`}
                >
                  <strong>{username}</strong>
                </button>
              ) : (
                <strong>{username}</strong>
              )}
            </span>
            <button type="button" className="btn-secondary btn-sm" onClick={onSignOut}>
              Sign Out
            </button>
          </div>
        )}
      </header>
    </>
  );
}
