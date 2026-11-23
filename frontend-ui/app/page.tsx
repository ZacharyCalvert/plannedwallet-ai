'use client';

import { useState, useEffect, useRef } from 'react';
import { UserSession } from '../lib/types';
import { fetchFeatures, refreshToken } from '../lib/api';
import {
  getRemainingSessionMs,
  isJwtTermsApproved,
  shouldRefreshTokenOnActivity,
} from '../lib/tokenRefresh';
import {
  isReadOnlyFeatureEnabled,
  subscribeMaintenanceWarning,
} from '../lib/maintenance';
import { subscribePolicyRequired } from '../lib/policyAgreement';
import { Banner } from '../components/Banner';
import { AuthCenter } from '../components/AuthCenter';
import { Dashboard } from '../components/Dashboard';
import { PolicyAgreementPrompt } from '../components/PolicyAgreementPrompt';
import { Footer } from '../components/Footer';
import { MaintenanceWarningModal } from '../components/MaintenanceWarningModal';

function AppShell() {
  const [session, setSession] = useState<UserSession | null>(null);
  const [isReadOnly, setIsReadOnly] = useState(false);
  const [maintenanceWarning, setMaintenanceWarning] = useState<string | null>(null);
  const [policyRequired, setPolicyRequired] = useState(false);
  const [sessionNotice, setSessionNotice] = useState('');
  const [openUserTrigger, setOpenUserTrigger] = useState(0);
  const sessionRef = useRef<UserSession | null>(session);
  const refreshInFlightRef = useRef(false);
  const lastRefreshFailureRef = useRef<number | null>(null);
  sessionRef.current = session;

  useEffect(() => {
    if (!session) {
      refreshInFlightRef.current = false;
      lastRefreshFailureRef.current = null;
      return;
    }

    const handleUserActivity = () => {
      const current = sessionRef.current;
      if (!current) return;
      const isAppFocused = typeof document.hasFocus === 'function' ? document.hasFocus() : true;
      if (
        !shouldRefreshTokenOnActivity({
          token: current.token,
          isAppFocused,
          inFlight: refreshInFlightRef.current,
          nowMs: Date.now(),
          lastFailureAtMs: lastRefreshFailureRef.current,
        })
      ) {
        return;
      }

      const activeUsername = current.username;
      const activeToken = current.token;
      refreshInFlightRef.current = true;
      refreshToken(activeUsername, activeToken)
        .then((nextToken) => {
          lastRefreshFailureRef.current = null;
          setSession((prev) =>
            prev && prev.username === activeUsername && prev.token === activeToken
              ? { ...prev, token: nextToken }
              : prev
          );
        })
        .catch(() => {
          lastRefreshFailureRef.current = Date.now();
        })
        .finally(() => {
          refreshInFlightRef.current = false;
        });
    };

    document.addEventListener('click', handleUserActivity, true);
    document.addEventListener('keydown', handleUserActivity, true);
    return () => {
      document.removeEventListener('click', handleUserActivity, true);
      document.removeEventListener('keydown', handleUserActivity, true);
    };
  }, [Boolean(session)]);

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
    const unsubscribePolicy = subscribePolicyRequired(() => {
      setPolicyRequired(true);
    });
    return () => {
      cancelled = true;
      unsubscribe();
      unsubscribePolicy();
    };
  }, []);

  useEffect(() => {
    if (!session) return;
    const remainingMs = getRemainingSessionMs(session.token, Date.now());
    if (remainingMs === null) return;

    const expireSession = () => {
      setSession(null);
      setSessionNotice('Your session expired after 15 minutes of inactivity. Please sign in again.');
    };

    if (remainingMs === 0) {
      expireSession();
      return;
    }

    const timeoutId = window.setTimeout(expireSession, remainingMs + 50);
    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [session]);

  const handleAuthSuccess = (username: string, token: string) => {
    setSessionNotice('');
    setPolicyRequired(!isJwtTermsApproved(token));
    setSession({ username, token });
  };

  const handlePolicyApproved = (nextToken: string) => {
    setPolicyRequired(false);
    setSession((prev) => (prev ? { ...prev, token: nextToken } : prev));
  };

  const handleSignOut = () => {
    setSessionNotice('');
    setPolicyRequired(false);
    setSession(null);
  };

  const handleUserDeleted = () => {
    setSessionNotice('Your user account and all associated records have been permanently deleted.');
    setPolicyRequired(false);
    setSession(null);
  };

  const needsPolicyApproval = Boolean(
    session && (policyRequired || !isJwtTermsApproved(session.token))
  );

  return (
    <div className="app-root">
      <Banner
        username={session?.username}
        onUserClick={needsPolicyApproval ? undefined : () => setOpenUserTrigger((n) => n + 1)}
        onSignOut={handleSignOut}
        readOnly={isReadOnly}
      />
      <div className="app-content">
        {!session ? (
          <AuthCenter onSuccess={handleAuthSuccess} readOnly={isReadOnly} sessionNotice={sessionNotice} />
        ) : needsPolicyApproval ? (
          <PolicyAgreementPrompt
            session={session}
            onApproved={handlePolicyApproved}
            onSignOut={handleSignOut}
          />
        ) : (
          <Dashboard
            key={session.username}
            session={session}
            onUserDeleted={handleUserDeleted}
            openUserTrigger={openUserTrigger}
          />
        )}
      </div>
      <Footer />
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

  return <AppShell />;
}
