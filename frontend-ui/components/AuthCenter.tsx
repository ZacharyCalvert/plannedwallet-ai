'use client';

import { useState } from 'react';
import { SignInForm } from './SignInForm';
import { CreateUserForm } from './CreateUserForm';

interface AuthCenterProps {
  onSuccess: (username: string, token: string) => void;
  readOnly?: boolean;
  sessionNotice?: string;
}

export function AuthCenter({ onSuccess, readOnly = false, sessionNotice }: AuthCenterProps) {
  const [tab, setTab] = useState<'signin' | 'create'>('signin');
  const activeTab = readOnly ? 'signin' : tab;

  return (
    <div className="auth-center-container">
      <div className="auth-card">
        <div className="auth-tabs">
          <button
            type="button"
            className={`auth-tab ${activeTab === 'signin' ? 'active' : ''}`}
            onClick={() => setTab('signin')}
          >
            Sign In
          </button>
          {!readOnly && (
            <button
              type="button"
              className={`auth-tab ${activeTab === 'create' ? 'active' : ''}`}
              onClick={() => setTab('create')}
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
            <SignInForm onSuccess={onSuccess} />
          ) : (
            <CreateUserForm onSuccess={onSuccess} />
          )}
        </div>
      </div>
    </div>
  );
}
