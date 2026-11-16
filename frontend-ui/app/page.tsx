'use client';

import { useState, useEffect } from 'react';
import { BrowserRouter, useNavigate } from 'react-router-dom';
import { UserSession } from '../lib/types';
import { Banner } from '../components/Banner';
import { AuthCenter } from '../components/AuthCenter';
import { Dashboard } from '../components/Dashboard';

function AppShell() {
  const [session, setSession] = useState<UserSession | null>(null);
  const navigate = useNavigate();

  const handleAuthSuccess = (username: string, token: string) => {
    setSession({ username, token });
  };

  const handleSignOut = () => {
    setSession(null);
    navigate('/', { replace: true });
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

export default function HomePage() {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) {
    return <div className="app-root" />;
  }

  return (
    <BrowserRouter>
      <AppShell />
    </BrowserRouter>
  );
}
