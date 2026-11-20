# Agent Policy & PDF Regeneration Guidelines

This guide instructs AI agents and engineers on how to edit, style, regenerate, and verify the two legal/compliance PDFs shipped with the Planned Wallet static frontend.

---

## 1. File Locations & Overwrite Policy ("Step On Existing Files")

Regenerated PDFs **must directly overwrite (step on)** the existing tracked files in `frontend-ui/public/` using the exact same filenames:

| Document | Exact Target Path (Overwrite In Place) | Served URL | Structure & Pagination |
| :--- | :--- | :--- | :--- |
| **Privacy & Security Policy** | `frontend-ui/public/privacy-and-security-policy.pdf` | `/privacy-and-security-policy.pdf` | **Multi-page** (Part 1: Privacy Policy, Part 2: Security Policy starting on a new page; grows naturally with content) |
| **Terms of Use** | `frontend-ui/public/terms-of-use.pdf` | `/terms-of-use.pdf` | **Multi-page** (grows naturally to accommodate full legal coverage) |

### Strict Asset Hygiene Rules
- **Do NOT rename, version-suffix, or move** the files (e.g., never create `terms-of-use-v2.pdf`); UI components (`frontend-ui/components/Footer.tsx`, `frontend-ui/components/PolicyAgreementPrompt.tsx`), client hashing helpers (`frontend-ui/lib/policyAgreement.ts`), and integration tests (`itests/test_features.py`) reference `/terms-of-use.pdf` and `/privacy-and-security-policy.pdf`.
- **Do NOT leave source HTML, Markdown, CSS, or generator scripts inside `frontend-ui/public/`**. Next.js static export (`npm run build` -> `frontend-ui/out/`) copies every file under `frontend-ui/public/` straight to S3/CloudFront. Keep PDF generation scripts in `scripts/pdf-generation/` (e.g. `scripts/pdf-generation/generate_pdfs.py`).
- **No Third-Party Trademarks / Registered Financial Brands**: Keep all examples generic and refer strictly to **Planned Wallet** and **Events and APIs LLC** per `AGENTS.md`.
- **Public PDF Security Hygiene (No Internal Details)**: Do **NOT** include explicit internal HTTP header names (`X-Origin-Verify`, `X-Client-ID`, `traceparent`, etc.), URL paths (`/token`, `/api/v1/users`, etc.), environment variable names (`EXPECTED_ORIGIN_SECRET`, `PW_READONLY`, `PW_TERMS`, etc.), database names/flags (`rds.force_ssl`), or internal table names (`user_recovery_keys`, `policy_agreements`, etc.) in the generated public PDFs. Describe security, session, and auditing controls in clear, non-identifying prose.

---

## 2. Mandatory Corporate, Eligibility & Platform Requirements

Every regeneration of either PDF **must** preserve and reflect these non-negotiable details:

1. **Legal Entity / Operator**:
   - **Events and APIs LLC** (copyright footer: `© 2026 Events and APIs LLC. All rights reserved.`).
2. **Jurisdiction & US-Only Eligibility**:
   - Planned Wallet is designed, hosted exclusively in **United States (US) cloud regions**, and operated strictly for **United States customers and consumers only**.
   - Terms of Use must explicitly state: `Registration requires affirmative confirmation that you are at least 18 years of age and a customer located in the United States.`
   - Individuals outside the United States must be instructed not to register or submit personal or financial data.
3. **Core Privacy & AI Commitments**:
   - Events and APIs LLC **never sells, rents, leases, trades, or brokers** personal or financial data to third parties.
   - Customer data is **never fed into any AI service, machine learning pipeline, or Large Language Model (LLM)**; all balances, reconciliations, schedules, and trend charts are computed deterministically.
4. **Core Security & Session Behavior (Must Match Implemented Code; Describe Without Internal Identifiers)**:
   - **Authentication & Recovery**: Username, email, salted `bcrypt` password hashing (enforced 72-byte UTF-8 boundary), mandatory **RFC 6238 TOTP MFA**, visual CAPTCHA on registration and recovery, and single-use rotating **12-word recovery phrases** stored as salted **Argon2id** hashes.
   - **15-Minute Inactivity Auto-Logout**: Access JWTs expire after **15 minutes (900 seconds)** (`HMAC-SHA256` with unique token identifiers) and refresh only during active, focused user clicks or keystrokes inside the UI.
   - **Policy Ledger**: Immutable audit ledger storing user acceptance, active `terms-of-use.pdf` SHA-256 digest, `US` consumer confirmation, timestamps, and client IP address.
   - **Zero Caching**: Strict no-store and no-cache directives on gateway/static/API responses.
   - **RBAC & Anti-Enumeration**: `read` / `edit` / `admin` account sharing roles; unauthorized access returns `404 Not Found`.
   - **Read-Only Maintenance Mode & Deletion**: Configurable read-only maintenance safeguard halts writes (`503 Service Unavailable`); signed-in users can permanently delete records or their entire user profile/accounts from User Details, while operational security/audit logs (request correlation identifiers, IP addresses, timestamps) are retained.
5. **Cloud Hosting Accuracy Note (Update Legacy Azure Text to AWS)**:
   - The previous PDF revision contains legacy references to Microsoft Azure, Azure Front Door, and Azure PostgreSQL Flexible Server.
   - Because infrastructure has migrated to **AWS US regions**, update infrastructure wording on regeneration to reflect **AWS**:
     - Static frontend on **Amazon S3 (Private + Origin Access Control)** behind **Amazon CloudFront** + **AWS WAF v2** (TLS encryption, DDoS protection, US geographic alignment).
     - Backend API on **AWS ECS Fargate** behind **API Gateway HTTP API + Private VPC Link**, validating cryptographic origin-verification secrets from the edge gateway via constant-time comparison and rejecting bypass traffic with `403 Forbidden`.
     - Persistence on **Amazon RDS for PostgreSQL 16** in private subnets with storage encryption at rest, enforced TLS connections, automated backups, and parameterized SQL queries.
6. **Mandatory Terms of Use Legal Disclaimers & Clauses**:
   - **Informational & Organizational Tool Only**: Explicit rules stating that Planned Wallet does NOT offer financial, tax, investment, or legal advice. It is strictly a self-service organizational tool.
   - **Not an RIA, Broker-Dealer, CPA, or Fiduciary**: Events and APIs LLC is not a Registered Investment Advisor (RIA), broker-dealer, Certified Public Accountant (CPA), or fiduciary. Charts, projections, net-worth metrics, and wealth-tracking features do NOT constitute financial recommendations or investment advice.
   - **Manual Entry & Typo Disclaimer**: The platform relies on manual user entry; all outputs, calculations, ledger totals, and budget projections are entirely dependent on user-supplied data. Events and APIs LLC disclaims all liability for user typos, data-entry errors, miscalculations, omitted transactions, or outdated asset valuations.
   - **"As-Is" and "As-Available" Warranty**: The service is provided strictly "as-is" and "as-available" without warranties of any kind, specifically disclaiming warranties of uninterrupted uptime or error-free calculations.
   - **Not a System of Record**: Events and APIs LLC is not a bank, depository institution, or official system of record. Events and APIs LLC is not responsible or liable for lost, corrupted, or deleted financial entries. Users remain solely responsible for maintaining their own official financial statements and are encouraged to periodically export and maintain their own local backups of their financial data.
   - **Third-Party Data & Valuation Disclaimer**: Disclaims accuracy, timeliness, or completeness of third-party market data or valuation feeds (including cryptocurrencies, precious metals, stocks, playing cards, or other physical/digital assets).
   - **Account Termination & Acceptable Use**: Explicitly reserves the right to suspend or terminate accounts that attempt to scrape, crawl, reverse-engineer, decompile, abuse APIs or backend infrastructure, probe security, or use the platform for unlawful purposes.
   - **Binding Arbitration & Class-Action Waiver**: Disputes must be resolved individually through binding arbitration rather than in court or a class-action lawsuit, governed by the laws of the State of Texas, with exclusive venue in Tarrant County, Texas.
   - **Direct Damages Cap**: Limitation of Liability must include a hard aggregate direct damages cap equal to the greater of (A) the total amount paid by the user to Events and APIs LLC in the twelve (12) months preceding the event giving rise to liability, or (B) one hundred U.S. dollars ($100.00).
   - **Update Notification Mechanism**: We will notify users of material changes via in-app banner 30 days prior to changes taking place.
7. **Mandatory Privacy Policy Data Governance & Disclosures**:
   - **Clear Separation of Data Categories**:
     - *Account Data*: Name, email address, salted/hashed password (`bcrypt`), subscription billing status.
     - *User-Entered Financial Data*: Expense logs, transaction entries, budget categories, manually entered asset/liability values, and notes.
     - *Technical / Usage Data*: IP addresses, device/browser types, session logs, request correlation identifiers, and analytics/session cookies.
   - **Explicit Non-Collection Statement**: Explicitly state that Planned Wallet and Events and APIs LLC **never collect** bank credentials (logins/passwords/PINs), bank account numbers, routing numbers, Social Security Numbers (SSNs), or government-issued IDs.
   - **AWS Cloud Hosting & Sub-processors**: Detail that hosting and data persistence are powered by **Amazon Web Services (AWS)** in US regions (compute, storage, edge routing, managed RDS PostgreSQL databases) and list third-party sub-processor categories (cloud infrastructure/hosting, edge delivery/DDoS protection, customer support/transactional messaging, and subscription billing).
   - **Explicit Prohibition on Selling User Financial Data**: Clearly reaffirm that Events and APIs LLC explicitly does not sell, rent, lease, trade, or monetize user financial data or personal records.
   - **Baseline Security Controls & Realistic Disclaimer**: Describe baseline security controls (TLS/HTTPS encryption in transit, AES-256 encryption at rest, MFA/session expiration) without creating an impossible standard, expressly noting that no internet transmission or electronic storage method is 100% secure.
   - **Data Retention & Account Deletion Timelines**: Retained while the account is active; upon user deletion, permanently removed from active production databases within 30 days, and purged from rolling backup snapshots within a defined window (30–90 days).
   - **Children's Privacy (COPPA)**: Explicitly state that the service is strictly intended for individuals 18 years of age or older, and that we do not knowingly collect personal or financial data from individuals under 18 (or under 13 under COPPA).
   - **Privacy Rights & Contact Information**: Dedicated section detailing user access, correction, export (portability), and deletion rights (both self-service in the application / User Details and by email) with contact email `support@plannedwallet.com`.

---

## 3. Branding Guidelines (Colors, Fonts & Page Layout)

PDFs are rendered from HTML/CSS via **Headless Chromium (`Skia/PDF`)** so they share the exact design system defined in `frontend-ui/app/styles/variables.css`.

### 3.1 Color Palette

| Token / Usage | Hex Value | Visual Role |
| :--- | :--- | :--- |
| **Header Banner Dark Gradient** | `#0f172a` -> `#1e293b` | Top rounded banner background (deep slate/navy matching `.banner`) |
| **Header Title Text** | `#ffffff` | Bold document title (`Planned Wallet — Privacy Policy`, etc.) |
| **Header Subtitle & Right Meta** | `#cbd5e1` / `#94a3b8` | Company attribution, effective date, US-only badge text |
| **Primary Heading & Bold Lead Text** | `#0f172a` | `--foreground`: Section headings (`h2`), bullet lead-ins |
| **Body Prose** | `#334155` | Dark slate readable body copy |
| **Muted Footer & Captions** | `#64748b` | `--muted`: Footer copyright and `Page X of Y` text |
| **Primary Accent / Callout Left Border** | `#2563eb` (`#1d4ed8` lead label) | `--primary` / `--primary-hover`: `4px` left border on top summary callout |
| **Callout Box Fill & Border** | `#eff6ff` (border `#bfdbfe`) | `--primary-light`: Soft blue callout background |
| **Side-by-Side Feature Card Fill** | `#f8fafc` | Soft neutral card background (used on Security Policy 2-column grid) |
| **Hairline Dividers & Card Borders** | `#e2e8f0` | `--border`: Horizontal rule under section headers & above footer |
| **Inline Code Chip Background** | `#f1f5f9` (text `#0f172a`) | `--background`: Monospace pill for headers/flags (`bcrypt`, `X-Client-ID`, etc.) |

### 3.2 Typography

- **Primary Sans-Serif Stack**:
  `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`
- **Monospace Stack** (renders inline code chips cleanly, e.g. `Menlo-Regular` in macOS Chromium):
  `"Menlo", "SFMono-Regular", Consolas, "Liberation Mono", monospace`
- **Recommended Type Scale**:
  - Banner title: `18pt` (`font-weight: 700`, `letter-spacing: -0.02em`)
  - Banner subtitle & top-right meta: `9pt`–`9.5pt`
  - Callout box text: `9.5pt` (`line-height: 1.45`)
  - Section headings (`1. ...`, `2. ...`): `11.5pt` (`font-weight: 700`, bottom border `1px solid #e2e8f0`, `padding-bottom: 3pt`)
  - Body text & bullets: `9.5pt` (`line-height: 1.42`)
  - Two-column card titles: `9.5pt` bold; card body: `8.8pt` (`line-height: 1.4`)
  - Inline `<code>`: `8.5pt` monospace on `#f1f5f9`, `padding: 1px 4px`, `border-radius: 3px`
  - Footer text: `8.2pt` (`color: #64748b`)
  - **Comfortable Pagination**: Prioritize clear, legible typography (`9pt`–`9.5pt` body text, `1.42` line-height) over cramming content onto a fixed page count; allow both PDFs to expand naturally across multiple pages as required by full legal and policy coverage.

### 3.3 Page Geometry & Shared Structural Chrome

- **Page Size**: US Letter (`8.5in × 11in`), portrait, `print_background: true`.
- **Margins**: Top/Bottom `0.55in` (or `0.65in` bottom when using Playwright `footer_template`), Left/Right `0.60in`.
- **Top Header Banner** (`border-radius: 8px`, `padding: 14px 18px`, dark slate gradient):
  - Left block: Title (`Planned Wallet — <Section Title>`) + Subtitle (`Published by Events and APIs LLC` or `Events and APIs LLC • <Focus>`).
  - Right block: `Effective Date: <Month Day, Year>` + `US Customers & Consumers Only` (or `AWS Cloud (US Regions)` on the Security Policy banner).
- **Callout Summary Box** (immediately below banner on Privacy Policy and Terms of Use):
  - `#eff6ff` background, `4px solid #2563eb` left border, `1px solid #bfdbfe` remaining borders, `border-radius: 6px`.
- **Footer Bar** (rendered cleanly on every page via Playwright `display_header_footer=True` with `footer_template`, or `@page` CSS running footer, `border-top: 1px solid #e2e8f0`):
  - Left: `© 2026 Events and APIs LLC. All rights reserved.`
  - Right: `Planned Wallet • <Document Name> (Page <pageNumber> of <totalPages>)`

---

## 4. Outline of Existing Document Contents

Preserve the logical structure below when making edits or policy tweaks.

### 4.1 `privacy-and-security-policy.pdf` — Part 1: Privacy Policy

- **Banner**:
  - Title: `Planned Wallet — Privacy Policy`
  - Subtitle: `Published by Events and APIs LLC`
  - Right Meta: `Effective Date: <Date>` / `US Customers & Consumers Only`
- **Top Callout (`Core Privacy Commitments`)**:
  - Reaffirms essential commitments: operated strictly for US customers and consumers, zero data sales (we explicitly never sell user financial data), zero AI/LLM ingestion, no collection of banking credentials or government IDs, baseline encryption (TLS & AES-256), 15-minute auto-logout, and COPPA compliance (not intended for under 18).
- **Section 1 — Introduction, US Eligibility & Children's Privacy (COPPA)**:
  - Identifies **Events and APIs LLC** ("Company", "we", "us", "our") as the operator of Planned Wallet.
  - States **Intended for US Customers & Consumers Only**: designed, hosted in US AWS Cloud regions (`us-east-2`), intended strictly for US residents/consumers under US federal and state law; non-US individuals must not register or submit data.
  - **Children's Privacy (COPPA)**: The service is strictly intended for individuals 18 years of age or older. We do not knowingly collect, solicit, or maintain personal or financial data from anyone under 18 (or under 13 under the Children's Online Privacy Protection Act). If we learn that an individual under 18 has submitted personal information, we will promptly delete that data.
- **Section 2 — Categories of Data Collected**:
  - **Account Data**: Email address, salted and hashed password (`bcrypt`), and subscription billing status.
  - **User-Entered Financial Data**: Expense logs, transaction entries, budget categories, manually entered asset and liability values, notes, reconcile histories, and recurring schedules.
  - **Technical / Usage Data**: IP addresses, device/browser types, operating systems, session logs, request correlation identifiers, and essential analytics or session cookies.
- **Section 3 — What We Explicitly Do Not Collect**:
  - Explicit affirmative statement that Planned Wallet and Events and APIs LLC **never** collect, request, or store:
    - Bank account logins, credentials, passwords, PINs, or direct bank connection tokens (no Plaid, Finicity, or bank screen-scraping).
    - Full bank account numbers or routing numbers.
    - Social Security Numbers (SSNs), Taxpayer Identification Numbers (TINs), or national identification numbers.
    - Driver's licenses, passports, or other government-issued IDs.
- **Section 4 — How Data is Used & Shared (Sub-processors & AWS Cloud Hosting)**:
  - **Purpose of Use**: Processing deterministic balance and trend calculations, managing authentication and session security, preventing abuse and fraudulent activity, and maintaining platform uptime and reliability.
  - **Cloud Infrastructure & AWS Sub-processor**: Planned Wallet is hosted exclusively in **Amazon Web Services (AWS)** in US regions (e.g., `us-east-2`). AWS processes and persists data strictly on our behalf to provide secure compute (ECS Fargate), edge routing and DDoS mitigation (CloudFront, WAF v2, API Gateway), private static delivery (S3), and managed relational persistence (Amazon RDS for PostgreSQL).
  - **Categories of Third-Party Service Providers**: We share data only with necessary third-party sub-processors bound by strict confidentiality and data protection obligations:
    - *Cloud Infrastructure & Hosting Providers* (e.g., AWS compute, networking, database persistence).
    - *Edge Delivery & DDoS / Security Providers* (e.g., AWS CloudFront, AWS WAF).
    - *Customer Support & Transactional Email Services* (e.g., email delivery for account verification and password reset).
    - *Payment & Subscription Billing Processors* (e.g., payment gateway for managing subscription billing status without Planned Wallet ever seeing or storing full credit card numbers).
- **Section 5 — We Explicitly Do Not Sell User Financial Data & Zero AI Ingestion**:
  - **No Sale of Personal or Financial Data**: Events and APIs LLC **explicitly does not sell, rent, lease, monetize, trade, or broker** user financial data, transaction histories, personal details, or usage patterns to third parties, advertisers, or data brokers.
  - **Zero AI / LLM Training**: Customer financial records, memos, and categories are never fed into generative AI services, machine learning models, or LLM training pipelines.
- **Section 6 — Data Security & Encryption Practices**:
  - Baseline security controls described realistically without creating an impossible standard:
    - **Encryption in Transit**: All data transmitted between clients and our platform is encrypted in transit using modern Transport Layer Security (TLS / HTTPS).
    - **Encryption at Rest**: Databases, persistent volumes, and storage buckets are encrypted at rest using industry-standard AES-256 encryption.
    - **Access & Authentication Safeguards**: Mandatory RFC 6238 TOTP MFA, visual CAPTCHA verification, Argon2id recovery phrases, and 15-minute inactivity auto-logout.
    - **Realistic Standard / Security Disclaimer**: While we implement robust administrative, technical, and physical safeguards designed to protect personal and financial data, no method of transmission over the internet or electronic storage method is 100% secure, and absolute security cannot be guaranteed.
- **Section 7 — Data Retention & Account Deletion Rights**:
  - **Active Account Retention**: Personal and user-entered financial data is retained only for the duration of an active account to provide services.
  - **User-Controlled Account Deletion**: Users have the right to permanently delete specific records, financial accounts, or their entire user account at any time directly through the User Details screen.
  - **Deletion & Purge Timeline**: When a user initiates account deletion, their personal and financial records are permanently deleted from active production databases within 30 days (or immediately upon confirmation). Data preserved in automated encrypted backup archives is purged within a defined backup lifecycle window (e.g., 30 to 90 days as backup snapshots rotate and expire), after which it is completely unrecoverable. Anonymized operational telemetry (timestamps, trace IDs, security audit logs) may be retained for security auditing and legal compliance.
- **Section 8 — Your Privacy Rights & Contact Information**:
  - **Access, Correction, Export (Portability) & Deletion Rights**: Users have the right to access, correct, export, or permanently delete their account and financial entries. Self-service export and deletion are available directly in the application and **User Details**, or users may submit a verified deletion or privacy inquiry via email.
  - **Privacy Contact**: Direct privacy questions, deletion requests, or compliance inquiries to `support@plannedwallet.com` (operated by **Events and APIs LLC**).
- **Footer**: `© 2026 Events and APIs LLC. All rights reserved.` | `Planned Wallet • Privacy & Security Policy (Page X of Y)`

### 4.2 `privacy-and-security-policy.pdf` — Part 2: Security Policy (Starts on New Page)

- **Banner**:
  - Title: `Planned Wallet — Security Policy`
  - Subtitle: `Events and APIs LLC • AWS Cloud Hosting & Security Controls`
  - Right Meta: `Effective Date: <Date>` / `AWS Cloud (US Regions)`
- **Section 1 — AWS Cloud Infrastructure, Edge Security & RDS PostgreSQL**:
  - Intro paragraph summarizing defense-in-depth across US AWS edge ingress, containerized compute, and managed persistence.
  - **2-Column Grid Cards (`#f8fafc` / `1px solid #e2e8f0`)**:
    - **Left Card — CloudFront Edge, WAF v2 & API Gateway**:
      - TLS encryption in transit, managed DDoS mitigation, AWS WAF v2 inspection & rate limiting, US-aligned consumer access controls, Private S3 (OAC) static delivery, and Private VPC Link to ECS Fargate.
      - Production containers validate cryptographic origin-verification secrets from the edge gateway using constant-time comparison and reject unauthorized direct origin traffic (`403 Forbidden`).
    - **Right Card — Amazon RDS for PostgreSQL 16**:
      - Private-subnet Amazon RDS PostgreSQL 16 database with storage encryption at rest, enforced TLS connections, automated backups/point-in-time recovery, and parameterized SQL queries preventing SQL injection.
- **Section 2 — Application & Identity Security Practices**:
  - **Mandatory Multi-Factor Authentication (TOTP MFA) & Recovery**: Visual CAPTCHA verification, salted `bcrypt` password verification (72-byte limit enforced cleanly), RFC 6238 TOTP login, and single-use rotating 12-word recovery phrases hashed with Argon2id.
  - **15-Minute Token Lifetime & Inactivity Auto-Logout**: `HMAC-SHA256` JWTs (`900s` TTL, unique token identifiers), refreshed only on focused user interaction.
  - **Zero Client/Proxy Caching**: Strict no-store and no-cache directives on gateway/frontend/sensitive API responses.
  - **Role-Based Account Access & Anti-Enumeration**: `read`, `edit`, and `admin` sharing controls; unauthorized private resource callers receive `404 Not Found`.
- **Section 3 — AI Isolation, Data Governance & Operational Readiness**:
  - **Zero AI / LLM Integration**: No outbound pipeline or integration feeding customer data to AI/LLMs.
  - **No Sale of Customer Data**: Strict prohibition by Events and APIs LLC.
  - **Traceability & Read-Only Maintenance Safeguard**: End-to-end distributed tracing and request correlation logging, plus a configurable read-only maintenance toggle halting mutations (`503 Service Unavailable`) while allowing reads and login.
  - **Jurisdiction**: Operated by Events and APIs LLC exclusively for US customers and consumers.
- **Footer**: `© 2026 Events and APIs LLC. All rights reserved.` | `Planned Wallet • Privacy & Security Policy (Page X of Y)`

### 4.3 `terms-of-use.pdf` — Terms of Use

- **Banner**:
  - Title: `Planned Wallet — Terms of Use`
  - Subtitle: `Events and APIs LLC • Consumer Agreement`
  - Right Meta: `Effective Date: <Date>` / `US Customers & Consumers Only`
- **Top Callout (`Summary of Key Terms`)**:
  - Highlights essential terms: US customers only, informational & self-service organizational tool only (no financial, tax, investment, or legal advice; no RIA, broker-dealer, CPA, or fiduciary relationship), strict reliance on manual user entry (disclaiming liability for typos, miscalculations, or outdated valuations), "as-is" and "as-available" without warranties of uninterrupted uptime or error-free calculations, not an official system of record (no liability for lost or corrupted entries), third-party asset valuation disclaimer, 15-minute inactivity auto-logout, zero data selling or AI training, and mandatory individual binding arbitration with a class-action waiver.
- **Section 1 — Acceptance of Terms & US Consumer Eligibility**:
  - Agreement between the user and **Events and APIs LLC** governing access to Planned Wallet web application, APIs, and documentation.
  - **United States Only**: Explicit requirement that registration requires affirmative confirmation that the user is at least 18 years of age and a customer located in the United States. Non-US residents and individuals under 18 are prohibited from registering or transmitting data.
- **Section 2 — Informational Tool Only, Non-Fiduciary Status & Disclaimers**:
  - **Informational & Organizational Tool Only**: Explicit rules stating that Planned Wallet does NOT offer financial, tax, investment, or legal advice. It is strictly a self-service personal budgeting and organizational tool.
  - **No RIA, Broker-Dealer, CPA, or Fiduciary Relationship**: Events and APIs LLC is not a Registered Investment Advisor (RIA), broker-dealer, Certified Public Accountant (CPA), financial planner, or fiduciary. Graphs, charts, amortization schedules, projections, and wealth-tracking/net-worth perspectives are purely mathematical representations of user inputs and do NOT constitute financial recommendations, endorsements, or investment advice.
  - **Reliance on Manual User Entry & Typo Disclaimer**: The platform relies entirely on manual user inputs. All outputs, balances, net-worth calculations, and budget projections are strictly dependent on user-supplied data. Events and APIs LLC disclaims all responsibility and liability for user typos, data-entry errors, miscalculations, omitted entries, or outdated valuations.
  - **Not a System of Record**: Events and APIs LLC is not a bank, depository institution, or official system of record. Events and APIs LLC is not responsible or liable for lost, corrupted, delayed, or deleted financial entries. Users remain solely responsible for retaining and verifying their own official financial statements and records, and are strongly encouraged to periodically export and maintain their own local backups of their financial data.
  - **Third-Party Data & Valuation Disclaimer**: Third-party reference data, pricing feeds, and asset valuations (including cryptocurrencies, precious metals, stocks, collectible playing cards, or other digital and physical assets) are provided for reference only without guarantees of timeliness, accuracy, or completeness. Events and APIs LLC disclaims all liability for outdated, delayed, or erroneous third-party market data.
- **Section 3 — Account Security, 15-Minute Auto-Logout & Policy Ledger**:
  - **Credentials, TOTP & Recovery Phrase Protection**: User is solely responsible for safeguarding username, password, RFC 6238 TOTP authenticator secret, and single-use 12-word recovery phrase.
  - **15-Minute Inactivity Auto-Logout**: Access JWTs expire after 15 minutes (900s) of inactivity and refresh solely during focused user interaction.
  - **Immutable Policy Agreement Ledger**: Policy updates require re-acceptance. Acceptance is recorded in our tamper-evident policy agreement ledger storing the cryptographic SHA-256 digest of the Terms of Use, timestamp, US customer confirmation, and client IP address.
- **Section 4 — Account Termination & Acceptable Use**:
  - **Acceptable Use & Conduct Prohibitions**: Users agree not to scrape, crawl, spider, or systematically extract platform data; reverse-engineer, decompile, or disassemble application code or API schemas; abuse or overload APIs; probe or bypass edge origin verification or authentication controls; inject malicious payloads; or use the platform for unlawful, fraudulent, or prohibited purposes.
  - **Right of Account Suspension & Termination**: Events and APIs LLC expressly reserves the absolute right to suspend, terminate, or block accounts and IP addresses immediately, with or without notice, for any violation of acceptable use, security compromise, API abuse, scraping, or unlawful conduct.
  - **Zero AI / LLM Usage & No Data Sale**: Reaffirms that customer data is never sold, traded, or brokered, and is never fed to AI models or LLM training pipelines.
  - **User Data Deletion & Retained Audit Logs**: Users may permanently delete financial accounts, records, or their entire profile via User Details. Operational security audit logs (timestamps, IP addresses, and request correlation identifiers) are retained for compliance and security auditing.
- **Section 5 — "As-Is" Warranties, Limitation of Liability, Arbitration & Governing Law**:
  - **"AS IS" and "AS AVAILABLE" Warranties**: The service is provided strictly "AS IS" and "AS AVAILABLE" without warranties of any kind, whether express, statutory, or implied (including warranties of merchantability, fitness for a particular purpose, and non-infringement). Events and APIs LLC specifically disclaims warranties of uninterrupted uptime, availability, or error-free calculations.
  - **Limitation of Liability**: To the maximum extent permitted by applicable law, in no event shall Events and APIs LLC, its owners, members, officers, employees, or agents be liable for any indirect, incidental, special, consequential, or punitive damages, or loss of profits, data, goodwill, or business opportunity. Must also include an explicit aggregate direct damages cap: `"IN NO EVENT SHALL EVENTS AND APIS LLC'S TOTAL AGGREGATE LIABILITY FOR ALL CLAIMS ARISING OUT OF OR RELATING TO THESE TERMS OR THE SERVICE EXCEED THE GREATER OF (A) THE TOTAL AMOUNT PAID BY YOU TO EVENTS AND APIS LLC IN THE TWELVE (12) MONTHS PRECEDING THE EVENT GIVING RISE TO LIABILITY, OR (B) ONE HUNDRED U.S. DOLLARS ($100.00)."`
  - **Binding Individual Arbitration**: Any dispute, controversy, or claim arising out of or relating to these Terms, the Privacy & Security Policy, or access to Planned Wallet shall be resolved exclusively through final and binding individual arbitration administered under the rules of the American Arbitration Association (AAA) or equivalent recognized arbitration provider, rather than in court, except that either party may assert qualifying individual claims in small claims court.
  - **Class-Action Waiver**: ALL DISPUTES MUST BE RESOLVED ON AN INDIVIDUAL BASIS. USERS EXPRESSLY WAIVE ANY RIGHT TO COMMENCE, JOIN, OR PARTICIPATE AS A PLAINTIFF OR CLASS MEMBER IN ANY CLASS ACTION, COLLECTIVE ACTION, PRIVATE ATTORNEY GENERAL ACTION, OR REPRESENTATIVE PROCEEDING AGAINST EVENTS AND APIS LLC.
  - **Governing Law & Venue**: Governed by the laws of the United States and the laws of the State of Texas (without regard to conflict-of-law principles). Exclusive venue for any non-arbitrable proceeding lies in the state or federal courts located in Tarrant County, Texas.
- **Footer**: `© 2026 Events and APIs LLC. All rights reserved.` | `Planned Wallet • Terms of Use (Page X of Y)`

---

## 5. Recommended Regeneration Workflow (Chromium / Playwright)

Run the top-level Makefile target to regenerate both PDFs into `frontend-ui/public/` and print the updated `PW_TERMS` SHA-256 checksum:

```bash
make pdfs
```

Under the hood, [generate_pdfs.py](scripts/pdf-generation/generate_pdfs.py) renders clean HTML templates using headless Chromium via the existing Python Playwright environment (`itests/.venv`) to produce compact, searchable PDFs matching the existing Chromium Skia/PDF typography:

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.launch()
    page = browser.new_page()
    page.set_content(html_string, wait_until="load")
    page.pdf(
        path=output_pdf_path,
        format="Letter",
        print_background=True,
        display_header_footer=True,
        header_template="<div></div>",
        footer_template=footer_html,  # uses .pageNumber and .totalPages
        margin={"top": "0.55in", "bottom": "0.65in", "left": "0.60in", "right": "0.60in"},
    )
    browser.close()
```

### Pagination & Visual Inspection Check
After generating the PDFs, inspect every rendered page to ensure:
1. `frontend-ui/public/privacy-and-security-policy.pdf` cleanly separates **Part 1 (Privacy Policy)** and **Part 2 (Security Policy)** onto a new page via CSS `page-break-before: always` / `break-before: page`, expanding naturally across as many pages as needed without orphan headings or trailing blank pages.
2. `frontend-ui/public/terms-of-use.pdf` expands naturally across as many pages as needed for comfortable readability and complete legal coverage, using `break-inside: avoid` on callouts or compact blocks so sections do not split awkwardly.
3. Every page displays a clean footer (`© 2026 Events and APIs LLC. All rights reserved.` and `Page X of Y`), with no clipped borders, overflowing `<code>` pills, overlapping text, or broken 2-column grid alignment.

---

## 6. Critical Post-Update Checksum Sync (`PW_TERMS` & `PW_PRIVACY`)

Because `backend-api` enforces policy acceptance against the exact SHA-256 checksums of `frontend-ui/public/terms-of-use.pdf` (`PW_TERMS`) and `frontend-ui/public/privacy-and-security-policy.pdf` (`PW_PRIVACY`), **regenerating either PDF changes its SHA-256 hash** and requires updating every `PW_TERMS` / `PW_PRIVACY` reference before merging:

1. **Compute the new 64-character lowercase SHA-256 hex digests**:
   ```bash
   shasum -a 256 frontend-ui/public/terms-of-use.pdf frontend-ui/public/privacy-and-security-policy.pdf
   ```
2. **Update every repository reference holding the old hashes**:
   - `docker-compose.yml` (`PW_TERMS` and `PW_PRIVACY` defaults on `backend-api`)
   - `.github/workflows/backend-api.yml` (`-e PW_TERMS=...` and `-e PW_PRIVACY=...` in `test-and-build`)
   - `itests/test_features.py` (`assert digest == "..."` in `test_terms_of_use_pdf_served_and_matches_pw_terms` and `test_privacy_policy_pdf_served_and_matches_pw_privacy`)
   - *(Optional / informational)* `.github/scripts/test_verify_ecs_taskdef_env.py`, `backend-api/schema.yml` examples, and handler unit test sample hashes if you wish to keep fixture hashes aligned.
3. **Update Live AWS Environments (IaC / SSM / ECS Task Definition)**:
   - Update `PW_TERMS` and `PW_PRIVACY` in `stage` and `prod` (Terraform / SSM Parameter Store) simultaneously with deploying the updated frontend so users are cleanly prompted to re-accept the updated policies (`403 POLICY_APPROVAL_REQUIRED` -> modal prompt -> `policy_agreements` new active row).
4. **Verify End-to-End Before Committing**:
   ```bash
   make -C frontend-ui test     # verifies policyAgreement.test.ts SHA-256 computation
   make -C frontend-ui build    # exports static out/ containing both PDFs
   make itest                   # verifies /terms-of-use.pdf matches PW_TERMS in fresh Docker stack
   ```
