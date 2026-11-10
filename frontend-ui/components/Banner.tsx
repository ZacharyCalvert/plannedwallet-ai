interface BannerProps {
  username?: string;
  onSignOut?: () => void;
}

export function Banner({ username, onSignOut }: BannerProps) {
  return (
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
  );
}
