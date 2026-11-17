'use client';

import { useState, useEffect } from 'react';
import { BrowserRouter, useNavigate } from 'react-router-dom';
import { UserSession } from '../lib/types';
import { fetchFeatures } from '../lib/api';
import {
  isReadOnlyFeatureEnabled,
  subscribeMaintenanceWarning,
} from '../lib/maintenance';
import { Banner } from '../components/Banner';
import { AuthCenter } from '../components/AuthCenter';
import { Dashboard } from '../components/Dashboard';
import { MaintenanceWarningModal } from '../components/MaintenanceWarningModal';

function AppShell() {
  const [session, setSession] = useState<UserSession | null>(null);
  const [isReadOnly, setIsReadOnly] = useState(false);
  const [maintenanceWarning, setMaintenanceWarning] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;
    fetchFeatures().then((features) => {
      if (!cancelled) {
        setIsReadOnly(isReadOnlyFeatureEnabled(features));
      }
    });
    const unsubscribe = subscribeMaintenanceWarning((msg) => {
      setIsReadOnly(true);
      setMaintenanceWarning(msg);
    });
    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, []);

  const handleAuthSuccess = (username: string, token: string) => {
    setSession({ username, token });
  };

  const handleSignOut = () => {
    setSession(null);
    navigate('/', { replace: true });
  };

  return (
    <div className="app-root">
      <Banner username={session?.username} onSignOut={handleSignOut} readOnly={isReadOnly} />
      <div className="app-content">
        {!session ? (
          <AuthCenter onSuccess={handleAuthSuccess} readOnly={isReadOnly} />
        ) : (
          <Dashboard session={session} />
        )}
      </div>
      {maintenanceWarning && (
        <MaintenanceWarningModal
          message={maintenanceWarning}
          onClose={() => setMaintenanceWarning(null)}
        />
      )}
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
