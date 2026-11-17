import { MAINTENANCE_BANNER_TEXT } from '../lib/maintenance';

interface BannerProps {
  username?: string;
  onSignOut?: () => void;
  readOnly?: boolean;
}

export function Banner({ username, onSignOut, readOnly = false }: BannerProps) {
  return (
    <>
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
            <span className="banner-username">Signed in as: <strong>{username}</strong></span>
            <button type="button" className="btn-secondary btn-sm" onClick={onSignOut}>
              Sign Out
            </button>
          </div>
        )}
      </header>
    </>
  );
}
