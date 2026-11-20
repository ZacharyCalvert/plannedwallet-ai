from pathlib import Path
from playwright.sync_api import sync_playwright

EFFECTIVE_DATE = "November 19, 2026"

SHARED_CSS = """
  * {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    color: #334155;
    font-size: 9.3pt;
    line-height: 1.42;
    background: #ffffff;
  }
  .page-break {
    page-break-before: always;
    break-before: page;
  }
  .banner {
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
    color: #ffffff;
    border-radius: 8px;
    padding: 14px 18px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    break-inside: avoid;
  }
  .banner-left h1 {
    font-size: 17.5pt;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: #ffffff;
    margin-bottom: 2px;
  }
  .banner-left .subtitle {
    font-size: 9.2pt;
    color: #cbd5e1;
  }
  .banner-right {
    text-align: right;
    font-size: 8.8pt;
    color: #cbd5e1;
    line-height: 1.35;
  }
  .banner-right .badge {
    color: #94a3b8;
    font-weight: 600;
  }
  .callout {
    background: #eff6ff;
    border: 1px solid #bfdbfe;
    border-left: 4px solid #2563eb;
    border-radius: 6px;
    padding: 10px 14px;
    margin-bottom: 13px;
    font-size: 9.2pt;
    line-height: 1.44;
    break-inside: avoid;
  }
  .callout-title {
    color: #1d4ed8;
    font-weight: 700;
    margin-bottom: 3px;
    font-size: 9.5pt;
  }
  h2 {
    color: #0f172a;
    font-size: 11.2pt;
    font-weight: 700;
    border-bottom: 1px solid #e2e8f0;
    padding-bottom: 3pt;
    margin-top: 11px;
    margin-bottom: 6px;
    break-after: avoid;
    page-break-after: avoid;
  }
  p {
    margin-bottom: 6px;
  }
  ul {
    margin-left: 16px;
    margin-bottom: 7px;
  }
  li {
    margin-bottom: 4px;
  }
  strong {
    color: #0f172a;
    font-weight: 600;
  }
  code {
    font-family: "Menlo", "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    font-size: 8.3pt;
    background: #f1f5f9;
    color: #0f172a;
    padding: 1px 4px;
    border-radius: 3px;
  }
  .grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-top: 6px;
    margin-bottom: 10px;
    break-inside: avoid;
  }
  .card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 10px 12px;
    font-size: 8.8pt;
    line-height: 1.4;
  }
  .card-title {
    color: #0f172a;
    font-size: 9.5pt;
    font-weight: 700;
    margin-bottom: 5px;
  }
  .card ul {
    margin-left: 14px;
    margin-bottom: 0;
  }
  .card li {
    margin-bottom: 3.5px;
  }
  .caps {
    font-size: 8.9pt;
    line-height: 1.38;
  }
  .section-block {
    margin-bottom: 4px;
  }
"""

def build_footer_html(doc_label: str) -> str:
    return f"""
    <div style="width: 100%; margin: 0 0.60in; padding-top: 6px; border-top: 1px solid #e2e8f0;
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
                font-size: 8.2pt; color: #64748b; display: flex; justify-content: space-between; align-items: center;">
      <span>&copy; 2026 Events and APIs LLC. All rights reserved.</span>
      <span>Planned Wallet &bull; {doc_label} (Page <span class="pageNumber"></span> of <span class="totalPages"></span>)</span>
    </div>
    """

PRIVACY_AND_SECURITY_HTML = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<style>{SHARED_CSS}</style>
</head>
<body>

  <!-- PART 1: PRIVACY POLICY -->
  <div class="banner">
    <div class="banner-left">
      <h1>Planned Wallet &mdash; Privacy Policy</h1>
      <div class="subtitle">Published by Events and APIs LLC</div>
    </div>
    <div class="banner-right">
      <div>Effective Date: {EFFECTIVE_DATE}</div>
      <div class="badge">US Customers &amp; Consumers Only</div>
    </div>
  </div>

  <div class="callout">
    <div class="callout-title">Core Privacy Commitments</div>
    Events and APIs LLC operates Planned Wallet strictly for <strong>United States customers and consumers</strong>. We <strong>explicitly do not sell user financial data</strong>, never feed customer data into any AI service or LLM, never collect bank credentials or government IDs, enforce baseline encryption (TLS in transit and AES-256 at rest), enforce a strict 15-minute inactivity auto-logout, and restrict access to individuals 18 years of age or older.
  </div>

  <div class="section-block">
    <h2>1. Introduction, US Eligibility &amp; Children's Privacy (COPPA)</h2>
    <p>This Privacy Policy explains how <strong>Events and APIs LLC</strong> (&ldquo;Company,&rdquo; &ldquo;we,&rdquo; &ldquo;us,&rdquo; or &ldquo;our&rdquo;) collects, uses, safeguards, and deletes information in connection with the Planned Wallet web application and APIs.</p>
    <ul>
      <li><strong>Intended for US Customers &amp; Consumers Only:</strong> Planned Wallet is designed, hosted exclusively in United States Amazon Web Services (AWS) cloud regions, and operated strictly for United States residents and consumers under applicable US federal and state laws. Individuals outside the United States must not register or submit personal or financial data.</li>
      <li><strong>Children's Privacy (COPPA):</strong> Planned Wallet is strictly intended for individuals <strong>18 years of age or older</strong>. We do not knowingly collect, solicit, or maintain personal or financial data from anyone under 18 (or under 13 under the Children's Online Privacy Protection Act). If we learn that an individual under 18 has registered or submitted personal information, we will promptly delete that data.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>2. Categories of Data Collected</h2>
    <p>We collect only the information necessary to provide self-service financial organization and secure account access:</p>
    <ul>
      <li><strong>Account Data:</strong> Email address, username, salted and hashed password (<code>bcrypt</code>), RFC 6238 TOTP MFA secret, single-use Argon2id-hashed 12-word recovery phrase metadata, and subscription billing status.</li>
      <li><strong>User-Entered Financial Data:</strong> Expense logs, transaction entries, account labels and types (<code>checking</code>, <code>savings</code>, <code>credit card</code>, <code>asset</code>, <code>snapshot</code>), budget categories, manually entered asset and liability values, reconcile histories, recurring schedules, and notes.</li>
      <li><strong>Technical / Usage Data:</strong> IP addresses, device and browser types, operating system metadata, session logs, request correlation identifiers, immutable policy acceptance records, and essential analytics or session cookies.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>3. What We Explicitly Do Not Collect</h2>
    <p>Planned Wallet is a manual-entry organizational platform. Events and APIs LLC <strong>never</strong> collects, requests, or stores:</p>
    <ul>
      <li>Bank account logins, credentials, passwords, PINs, or direct bank aggregation tokens (we do not connect to or screen-scrape bank accounts).</li>
      <li>Full bank account numbers or routing numbers.</li>
      <li>Social Security Numbers (SSNs), Taxpayer Identification Numbers (TINs), or national identification numbers.</li>
      <li>Driver's licenses, passports, or other government-issued identification documents.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>4. How Data is Used &amp; Shared (Sub-processors &amp; AWS Cloud Hosting)</h2>
    <p>We use collected data strictly to calculate deterministic balances and trend projections, authenticate sessions, prevent fraud and API abuse, and maintain service reliability.</p>
    <ul>
      <li><strong>AWS Cloud Hosting Sub-processor:</strong> Planned Wallet is hosted exclusively on <strong>Amazon Web Services (AWS)</strong> in US regions. AWS processes and persists data solely on our behalf to provide containerized compute (ECS Fargate), edge delivery and DDoS/WAF protection (CloudFront, WAF v2, API Gateway), private static object storage (S3), and managed relational persistence (Amazon RDS for PostgreSQL).</li>
      <li><strong>Categories of Third-Party Service Providers:</strong> We disclose data only to essential sub-processors bound by strict confidentiality and data protection obligations: (a) <em>Cloud Infrastructure &amp; Hosting Providers</em> (AWS compute, networking, and database storage); (b) <em>Edge Delivery &amp; Security Providers</em> (CloudFront CDN and WAF filtering); (c) <em>Customer Support &amp; Transactional Email Services</em> (verification and recovery notifications); and (d) <em>Payment &amp; Subscription Billing Processors</em> (managing subscription billing status without Planned Wallet ever viewing or storing full payment card numbers).</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>5. We Explicitly Do Not Sell User Financial Data &amp; Zero AI Ingestion</h2>
    <ul>
      <li><strong>No Sale of Personal or Financial Data:</strong> Events and APIs LLC <strong>explicitly does not sell, rent, lease, monetize, trade, or broker</strong> user financial data, transaction histories, personal details, or usage patterns to advertisers, data brokers, or any third party.</li>
      <li><strong>Customer Data Is Not Fed Into Any AI Service or LLM:</strong> Financial records, memos, categories, and balances are never sent to generative AI APIs, machine learning training pipelines, or Large Language Model (LLM) prompts. All financial calculations run deterministically in code.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>6. Data Security &amp; Encryption Practices</h2>
    <p>We maintain baseline administrative, technical, and physical safeguards designed to protect your information:</p>
    <ul>
      <li><strong>Encryption in Transit &amp; at Rest:</strong> All communications between your browser and our platform are encrypted in transit using modern TLS/HTTPS. Databases, persistent volumes, and storage buckets are encrypted at rest using industry-standard AES-256 encryption.</li>
      <li><strong>Access &amp; Session Controls:</strong> Accounts are protected by mandatory TOTP MFA, visual CAPTCHA checks, Argon2id recovery hashes, and a strict <strong>15-minute (900-second) inactivity auto-logout</strong>.</li>
      <li><strong>Security Disclaimer:</strong> While we implement strong baseline security practices, no method of transmission over the internet or electronic storage system is 100% secure, and absolute security cannot be guaranteed.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>7. Data Retention &amp; Account Deletion Rights</h2>
    <ul>
      <li><strong>Active Account Retention:</strong> Personal and user-entered financial data is retained only for as long as your account remains active and necessary to provide the service.</li>
      <li><strong>User-Controlled Deletion &amp; Purge Timelines:</strong> You may delete individual transactions, accounts, or your entire user profile at any time from <strong>User Details</strong>. Upon account deletion, your personal and financial records are permanently deleted from active production databases within <strong>30 days</strong>. Automated encrypted backup snapshots are purged within a defined <strong>30- to 90-day</strong> rolling lifecycle window as backup cycles rotate and expire. Anonymized operational security telemetry (timestamps, IP addresses, trace identifiers) may be retained for security auditing and legal compliance.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>8. Your Privacy Rights &amp; Contact Information</h2>
    <ul>
      <li><strong>Access, Correction, Export (Portability) &amp; Deletion Rights:</strong> As a United States consumer, you have the right to access, correct, export, or permanently delete your personal information and user-entered financial records. You can exercise your export and deletion rights directly at any time using the self-service controls in the application and <strong>User Details</strong>, or by submitting a verified request to our privacy team.</li>
      <li><strong>Contact Us:</strong> To submit a data deletion request, exercise your privacy rights, or ask questions about this Privacy Policy, please contact <strong>Events and APIs LLC</strong> at <code>support@plannedwallet.com</code>.</li>
    </ul>
  </div>

  <!-- PART 2: SECURITY POLICY -->
  <div class="page-break"></div>

  <div class="banner">
    <div class="banner-left">
      <h1>Planned Wallet &mdash; Security Policy</h1>
      <div class="subtitle">Events and APIs LLC &bull; AWS Cloud Hosting &amp; Security Controls</div>
    </div>
    <div class="banner-right">
      <div>Effective Date: {EFFECTIVE_DATE}</div>
      <div class="badge">AWS Cloud (US Regions)</div>
    </div>
  </div>

  <div class="section-block">
    <h2>1. AWS Cloud Infrastructure, Edge Security &amp; RDS PostgreSQL</h2>
    <p>Planned Wallet employs defense-in-depth across US-based AWS edge ingress, isolated container compute, and encrypted managed database persistence.</p>
    <div class="grid-2">
      <div class="card">
        <div class="card-title">CloudFront Edge, WAF v2 &amp; API Gateway</div>
        <ul>
          <li>Enforced HTTPS/TLS encryption in transit, managed DDoS mitigation, AWS WAF v2 inspection and rate limiting, and US-aligned consumer access controls.</li>
          <li>Static frontend assets served from private Amazon S3 via Origin Access Control (OAC); API traffic routed through API Gateway HTTP API and Private VPC Link to AWS ECS Fargate.</li>
          <li>Production backend containers validate cryptographic origin-verification secrets from the edge gateway using constant-time comparison and reject direct bypass requests with <code>403 Forbidden</code>.</li>
        </ul>
      </div>
      <div class="card">
        <div class="card-title">Amazon RDS for PostgreSQL 16</div>
        <ul>
          <li>Deployed in private VPC subnets with no public internet exposure, AES-256 storage encryption at rest, and enforced TLS connections.</li>
          <li>Automated encrypted backups and point-in-time recovery (PITR) protect against hardware failure while honoring rolling deletion lifecycles.</li>
          <li>All data access uses strictly parameterized SQL queries, eliminating SQL injection vectors across account and financial endpoints.</li>
        </ul>
      </div>
    </div>
  </div>

  <div class="section-block">
    <h2>2. Application &amp; Identity Security Practices</h2>
    <ul>
      <li><strong>Mandatory Multi-Factor Authentication (TOTP MFA) &amp; Recovery:</strong> Registration and recovery require visual CAPTCHA verification. Passwords are verified using salted <code>bcrypt</code> (72-byte UTF-8 boundary enforced cleanly), login requires RFC 6238 TOTP MFA, and single-use rotating 12-word recovery phrases are stored as salted Argon2id hashes.</li>
      <li><strong>15-Minute Token Lifetime &amp; Inactivity Auto-Logout:</strong> Access tokens are <code>HMAC-SHA256</code> JWTs (<code>900s</code> TTL with unique token identifiers), refreshed solely during active, focused user interaction in the UI.</li>
      <li><strong>Zero Client/Proxy Caching:</strong> Gateway, frontend, and sensitive API responses enforce strict no-store and no-cache directives to prevent client or proxy caching of financial data.</li>
      <li><strong>Role-Based Account Access &amp; Anti-Enumeration:</strong> Shared accounts enforce <code>read</code>, <code>edit</code>, and <code>admin</code> permission boundaries; unauthorized callers requesting private resources receive <code>404 Not Found</code> to prevent resource enumeration.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>3. AI Isolation, Data Governance &amp; Operational Readiness</h2>
    <ul>
      <li><strong>Zero AI / LLM Integration:</strong> No outbound pipeline, webhook, or model integration sends customer financial records, memos, or identities to external AI services or LLMs.</li>
      <li><strong>No Sale of Customer Data:</strong> Events and APIs LLC strictly prohibits selling, renting, or brokering customer personal or financial data.</li>
      <li><strong>Traceability &amp; Read-Only Maintenance Safeguard:</strong> Requests carry standardized distributed tracing and correlation identifiers for structured audit logging. A configurable read-only maintenance safeguard halts write mutations (<code>503 Service Unavailable</code>) while preserving read access and authentication.</li>
      <li><strong>Jurisdiction:</strong> Operated by Events and APIs LLC exclusively within US cloud infrastructure for United States customers and consumers.</li>
    </ul>
  </div>

</body>
</html>
"""

TERMS_OF_USE_HTML = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<style>{SHARED_CSS}</style>
</head>
<body>

  <div class="banner">
    <div class="banner-left">
      <h1>Planned Wallet &mdash; Terms of Use</h1>
      <div class="subtitle">Events and APIs LLC &bull; Consumer Agreement</div>
    </div>
    <div class="banner-right">
      <div>Effective Date: {EFFECTIVE_DATE}</div>
      <div class="badge">US Customers &amp; Consumers Only</div>
    </div>
  </div>

  <div class="callout">
    <div class="callout-title">Summary of Key Terms</div>
    By accessing Planned Wallet, you agree to these Terms of Use and our Privacy &amp; Security Policy, affirm that you are a customer in the United States (18+), and acknowledge that Planned Wallet is a self-service informational tool only (no financial, tax, investment, or legal advice; no RIA or fiduciary relationship), relies strictly on manual user entry (disclaiming typos and outdated valuations), is provided &ldquo;as-is&rdquo; without uptime or error-free calculation warranties, is not a system of record, enforces a 15-minute auto-logout, and requires <strong>binding individual arbitration and a class-action waiver</strong>.
  </div>

  <div class="section-block">
    <h2>1. Acceptance of Terms &amp; US Consumer Eligibility</h2>
    <ul>
      <li><strong>Operator &amp; Agreement:</strong> These Terms of Use govern your access to and use of Planned Wallet, operated by <strong>Events and APIs LLC</strong> (&ldquo;Company,&rdquo; &ldquo;we,&rdquo; &ldquo;us,&rdquo; or &ldquo;our&rdquo;).</li>
      <li><strong>United States Customers &amp; Consumers Only:</strong> Planned Wallet is intended exclusively for customers and consumers located in the United States who are at least 18 years of age. Registration requires affirmative confirmation that you are at least 18 years of age and a customer located in the United States. Individuals outside the United States or under 18 years of age are prohibited from registering or submitting data.</li>
      <li><strong>Material Updates:</strong> We will notify users of material changes to these Terms via an in-app banner at least 30 days prior to changes taking effect.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>2. Informational Tool Only, Non-Fiduciary Status &amp; Disclaimers</h2>
    <ul>
      <li><strong>No Financial, Tax, Investment, or Legal Advice:</strong> Planned Wallet is strictly a self-service informational and personal organizational tool. It does <strong>not</strong> offer financial, tax, investment, accounting, or legal advice, and provides no bank custody or money-movement capabilities.</li>
      <li><strong>No RIA, Broker-Dealer, CPA, or Fiduciary Status:</strong> Events and APIs LLC is <strong>not</strong> a Registered Investment Advisor (RIA), broker-dealer, Certified Public Accountant (CPA), financial planner, or fiduciary. Charts, projections, amortization views, and wealth-tracking or net-worth features are purely mathematical summaries of user inputs and do not constitute financial recommendations or investment advice.</li>
      <li><strong>Manual User Entry &amp; Typo Disclaimer:</strong> Because Planned Wallet relies on manual entry, all outputs, balances, net-worth calculations, and budget projections are entirely dependent on user-supplied data. Events and APIs LLC disclaims all liability for user typos, transposition errors, miscalculations, omitted entries, or outdated asset valuations.</li>
      <li><strong>Not a System of Record:</strong> Events and APIs LLC is not a bank, depository institution, or official system of record, and is not responsible or liable for lost, corrupted, or deleted financial entries. You remain solely responsible for maintaining and verifying your own official financial statements and records. We strongly encourage you to periodically export and maintain your own local backups of your financial data.</li>
      <li><strong>Third-Party Data &amp; Valuation Disclaimer:</strong> Any third-party data, reference feeds, or asset values (including cryptocurrencies, precious metals, stocks, collectible playing cards, or other digital and physical assets) are provided for reference only. We make no guarantees regarding their accuracy, timeliness, or completeness and disclaim liability for delayed or inaccurate valuations.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>3. Account Security, 15-Minute Auto-Logout &amp; Policy Ledger</h2>
    <ul>
      <li><strong>Credentials, TOTP &amp; Recovery Phrase Protection:</strong> You are solely responsible for safeguarding your password, RFC 6238 TOTP authenticator secret, and single-use 12-word recovery phrase.</li>
      <li><strong>15-Minute Inactivity Auto-Logout:</strong> Access tokens expire after 15 minutes (900 seconds) of inactivity and refresh only during active, focused user engagement inside the UI.</li>
      <li><strong>Immutable Policy Agreement Ledger:</strong> Acceptance is recorded in our tamper-evident policy agreement ledger storing the cryptographic SHA-256 digest of the Terms of Use, timestamps, 18+ age and US customer confirmation, and client IP address. Material updates require re-acceptance.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>4. Account Termination &amp; Acceptable Use</h2>
    <ul>
      <li><strong>Acceptable Use &amp; Conduct Prohibitions:</strong> You agree not to scrape, crawl, spider, or systematically extract data; reverse-engineer, decompile, or disassemble application code or APIs; abuse or overload APIs; probe or bypass edge origin verification or authentication controls; transmit malicious payloads; or use the platform for unlawful or fraudulent purposes.</li>
      <li><strong>Right of Account Suspension &amp; Termination:</strong> Events and APIs LLC expressly reserves the right to suspend, terminate, or block accounts and IP addresses immediately, with or without prior notice, if we determine you have attempted to scrape, reverse-engineer, abuse APIs, violate these Terms, or use the platform for unlawful purposes.</li>
      <li><strong>Zero AI / LLM Usage &amp; No Data Sale:</strong> Your data is never sold, rented, or brokered to third parties, and is never fed into AI services or LLM training pipelines.</li>
      <li><strong>User Deletion &amp; Retained Audit Logs:</strong> You may permanently delete financial accounts, records, or your entire user profile at any time via User Details. Operational security audit logs (timestamps, IP addresses, and request correlation identifiers) are retained for security auditing and compliance.</li>
    </ul>
  </div>

  <div class="section-block">
    <h2>5. &ldquo;As-Is&rdquo; Warranties, Limitation of Liability, Arbitration &amp; Governing Law</h2>
    <ul>
      <li><strong>&ldquo;AS IS&rdquo; and &ldquo;AS AVAILABLE&rdquo; Warranties:</strong> <span class="caps">THE SERVICE IS PROVIDED STRICTLY &ldquo;AS-IS&rdquo; AND &ldquo;AS-AVAILABLE&rdquo; WITHOUT WARRANTIES OF ANY KIND, EXPRESS, STATUTORY, OR IMPLIED, INCLUDING WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, OR NON-INFRINGEMENT. EVENTS AND APIS LLC DISCLAIMS ALL WARRANTIES OF UNINTERRUPTED UPTIME, AVAILABILITY, OR ERROR-FREE CALCULATIONS.</span></li>
      <li><strong>Limitation of Liability:</strong> <span class="caps">TO THE MAXIMUM EXTENT PERMITTED BY LAW, EVENTS AND APIS LLC AND ITS MEMBERS, OFFICERS, EMPLOYEES, AND CONTRACTORS SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, OR ANY LOSS OF PROFITS, DATA, OR GOODWILL ARISING FROM YOUR USE OF THE SERVICE. IN NO EVENT SHALL EVENTS AND APIS LLC'S TOTAL AGGREGATE LIABILITY FOR ALL CLAIMS ARISING OUT OF OR RELATING TO THESE TERMS OR THE SERVICE EXCEED THE GREATER OF (A) THE TOTAL AMOUNT PAID BY YOU TO EVENTS AND APIS LLC IN THE TWELVE (12) MONTHS PRECEDING THE EVENT GIVING RISE TO LIABILITY, OR (B) ONE HUNDRED U.S. DOLLARS ($100.00).</span></li>
      <li><strong>Binding Individual Arbitration:</strong> Any dispute, claim, or controversy arising out of or relating to these Terms, the Privacy &amp; Security Policy, or Planned Wallet shall be resolved exclusively through final and binding individual arbitration administered under the rules of the American Arbitration Association (AAA) or an equivalent recognized arbitration provider, rather than in court (except qualifying individual claims in small claims court).</li>
      <li><strong>Class-Action Waiver:</strong> <span class="caps">ALL DISPUTES MUST BE RESOLVED ON AN INDIVIDUAL BASIS. YOU EXPRESSLY WAIVE ANY RIGHT TO BRING, JOIN, OR PARTICIPATE AS A PLAINTIFF OR CLASS MEMBER IN ANY CLASS ACTION, COLLECTIVE ACTION, PRIVATE ATTORNEY GENERAL ACTION, OR REPRESENTATIVE PROCEEDING AGAINST EVENTS AND APIS LLC.</span></li>
      <li><strong>Governing Law &amp; Venue:</strong> These Terms are governed by applicable United States federal law and the laws of the State of Texas, without regard to conflict-of-law principles. Exclusive venue for any non-arbitrable proceeding lies in the state or federal courts located in Tarrant County, Texas.</li>
    </ul>
  </div>

</body>
</html>
"""

def main():
    repo_root = Path(__file__).resolve().parent.parent.parent
    out_dir = repo_root / "frontend-ui" / "public"
    privacy_pdf = out_dir / "privacy-and-security-policy.pdf"
    terms_pdf = out_dir / "terms-of-use.pdf"

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()

        # Generate Privacy & Security Policy
        page.set_content(PRIVACY_AND_SECURITY_HTML, wait_until="load")
        page.pdf(
            path=str(privacy_pdf),
            format="Letter",
            print_background=True,
            display_header_footer=True,
            header_template="<div></div>",
            footer_template=build_footer_html("Privacy &amp; Security Policy"),
            margin={"top": "0.55in", "bottom": "0.65in", "left": "0.60in", "right": "0.60in"},
        )

        # Generate Terms of Use
        page.set_content(TERMS_OF_USE_HTML, wait_until="load")
        page.pdf(
            path=str(terms_pdf),
            format="Letter",
            print_background=True,
            display_header_footer=True,
            header_template="<div></div>",
            footer_template=build_footer_html("Terms of Use"),
            margin={"top": "0.55in", "bottom": "0.65in", "left": "0.60in", "right": "0.60in"},
        )

        browser.close()
    print(f"Generated:\n  {privacy_pdf}\n  {terms_pdf}")

if __name__ == "__main__":
    main()
