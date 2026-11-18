'use client';

import { useState } from 'react';
import { SignInForm } from './SignInForm';
import { CreateUserForm } from './CreateUserForm';
import { RecoverAccountForm } from './RecoverAccountForm';

interface AuthCenterProps {
  onSuccess: (username: string, token: string) => void;
  readOnly?: boolean;
  sessionNotice?: string;
}

export function AuthCenter({ onSuccess, readOnly = false, sessionNotice }: AuthCenterProps) {
  const [tab, setTab] = useState<'signin' | 'create'>('signin');
  const [recovering, setRecovering] = useState(false);
  const activeTab = readOnly ? 'signin' : tab;

  return (
    <div className="auth-center-container">
      <div className={`auth-card ${(activeTab === 'create' || recovering) ? 'recovery-card-wide' : ''}`}>
        <div className="auth-tabs">
          <button
            type="button"
            className={`auth-tab ${activeTab === 'signin' ? 'active' : ''}`}
            onClick={() => {
              setTab('signin');
              setRecovering(false);
            }}
          >
            Sign In
          </button>
          {!readOnly && (
            <button
              type="button"
              className={`auth-tab ${activeTab === 'create' ? 'active' : ''}`}
              onClick={() => {
                setTab('create');
                setRecovering(false);
              }}
            >
              Create Account
            </button>
          )}
        </div>
        <div className="auth-body">
          {sessionNotice && (
            <div className="alert-error mb-3" role="status">
              {sessionNotice}
            </div>
          )}
          {activeTab === 'signin' ? (
            recovering ? (
              <RecoverAccountForm
                onSuccess={onSuccess}
                onCancel={() => setRecovering(false)}
              />
            ) : (
              <SignInForm
                onSuccess={onSuccess}
                onForgotCredentials={readOnly ? undefined : () => setRecovering(true)}
              />
            )
          ) : (
            <CreateUserForm onSuccess={onSuccess} />
          )}
        </div>
      </div>
    </div>
  );
}
