'use client';

import { useState } from 'react';
import { SignInForm } from './SignInForm';
import { CreateUserForm } from './CreateUserForm';

interface AuthCenterProps {
  onSuccess: (username: string, token: string) => void;
}

export function AuthCenter({ onSuccess }: AuthCenterProps) {
  const [tab, setTab] = useState<'signin' | 'create'>('signin');

  return (
    <div className="auth-center-container">
      <div className="auth-card">
        <div className="auth-tabs">
          <button
            type="button"
            className={`auth-tab ${tab === 'signin' ? 'active' : ''}`}
            onClick={() => setTab('signin')}
          >
            Sign In
          </button>
          <button
            type="button"
            className={`auth-tab ${tab === 'create' ? 'active' : ''}`}
            onClick={() => setTab('create')}
          >
            Create Account
          </button>
        </div>
        <div className="auth-body">
          {tab === 'signin' ? (
            <SignInForm onSuccess={onSuccess} />
          ) : (
            <CreateUserForm onSuccess={onSuccess} />
          )}
        </div>
      </div>
    </div>
  );
}
