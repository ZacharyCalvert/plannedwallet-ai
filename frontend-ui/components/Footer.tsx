'use client';

export function Footer() {
  return (
    <footer className="app-footer">
      <div className="app-footer-content">
        <span className="app-footer-copy">
          &copy; {new Date().getFullYear()} Events and APIs LLC. All rights reserved. Intended for US customers and consumers only.
        </span>
        <nav className="app-footer-links" aria-label="Legal and security policies">
          <a
            href="/privacy-and-security-policy.pdf"
            target="_blank"
            rel="noopener noreferrer"
            className="app-footer-link"
          >
            Privacy Policy
          </a>
          <span className="app-footer-sep" aria-hidden="true">|</span>
          <a
            href="/privacy-and-security-policy.pdf#page=2"
            target="_blank"
            rel="noopener noreferrer"
            className="app-footer-link"
          >
            Security Policy
          </a>
        </nav>
      </div>
    </footer>
  );
}
