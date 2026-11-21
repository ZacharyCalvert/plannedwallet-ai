import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Planned Wallet — Every Account. Zero Bank Logins.',
  description: 'Every Account. Zero Bank Logins. Track checking, savings, credit cards, retirement funds, and assets without exposing bank credentials.',
  icons: {
    icon: [
      { url: '/favicon.ico', sizes: 'any' },
      { url: '/planned-wallet-icon.svg', type: 'image/svg+xml' },
    ],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
