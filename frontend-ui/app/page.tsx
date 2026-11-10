'use client';

import { useState, useEffect } from 'react';

function Header() {
  return (
    <header className="header">
      <h1 className="title">Planned Wallet</h1>
      <p className="subtitle">Next.js Front-End Presentation Layer</p>
    </header>
  );
}

function ApiStatusCard() {
  const [status, setStatus] = useState<string>('Checking...');
  const [isOk, setIsOk] = useState<boolean | null>(null);

  useEffect(() => {
    fetch('/api/v1/hello')
      .then((res) => (res.ok ? res.text() : Promise.reject(res.statusText)))
      .then((data) => {
        setStatus(`Connected to API (${data})`);
        setIsOk(true);
      })
      .catch(() => {
        setStatus('API currently unreachable');
        setIsOk(false);
      });
  }, []);

  const badgeClass = isOk ? 'status-badge' : 'status-badge status-offline';
  return (
    <div className="card">
      <h2>Backend API Connectivity</h2>
      <p style={{ marginTop: '0.5rem' }}>
        Status: <span className={badgeClass}>{status}</span>
      </p>
    </div>
  );
}

function FeaturesGrid() {
  return (
    <div className="grid">
      <div className="card">
        <h3>User Management</h3>
        <p className="subtitle">Self-registration and JWT authentication.</p>
      </div>
      <div className="card">
        <h3>Financial Accounts</h3>
        <p className="subtitle">Checking accounts and access control.</p>
      </div>
      <div className="card">
        <h3>Checking Records</h3>
        <p className="subtitle">Transaction tracking, debits, and credits.</p>
      </div>
    </div>
  );
}

export default function HomePage() {
  return (
    <main className="container">
      <Header />
      <ApiStatusCard />
      <FeaturesGrid />
    </main>
  );
}
