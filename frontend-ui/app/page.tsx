'use client';

import { useState } from 'react';
import { UserSession } from '../lib/types';
import { Banner } from '../components/Banner';
import { AuthCenter } from '../components/AuthCenter';
import { Dashboard } from '../components/Dashboard';

export default function HomePage() {
  const [session, setSession] = useState<UserSession | null>(null);

  const handleAuthSuccess = (username: string, token: string) => {
    setSession({ username, token });
  };

  const handleSignOut = () => {
    setSession(null);
  };

  return (
    <div className="app-root">
      <Banner username={session?.username} onSignOut={handleSignOut} />
      <div className="app-content">
        {!session ? (
          <AuthCenter onSuccess={handleAuthSuccess} />
        ) : (
          <Dashboard session={session} />
        )}
      </div>
    </div>
  );
}
