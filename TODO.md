Top features:
- [x] X-Client-ID
- [x] trace-context (W3C Trace Context across backend & frontend) 
- [x] CORS support
- [x] API review / plan
- [x] Database review / plan
- [x] add / update / reconcile checking records
- [x] Add credit card account types
- [x] snapshot account inputs - this account is worth x at this time
- [x] CAPTCHA for account setup
- [x] asset account -> like X bitcoin / total worth of the account
- [x] Enforce uniqueness of email as well as user
- [x] Update user agent to enforce 12 factor behaviors

Top priority:
- [x] Infrastructure + auto deploy to staging
- [x] enforce REQUIRE_CAPTCHA set to true as environment variable
- [x] git statCreate JWT secret and mount as required field for JWT issuance / signing

Medium priority:
- [x] Password changes
- itest account enumeration (401 vs 403 vs 404)
- Account lock / require reset using recovery codes after X number of failed password attempts
- [x] Add dependabot monitoring of npm, docker images, golang dependencies
- [x] branch protections, code owners, etc.
- [x] Captcha on account creation - creation only
- [x] Infrastructure + auto deploy to production
- [x] Disaster recovery / backups / automation reload of data
- [x] Data retention
- [x] Protect collision on imported records
- [x] Add routing for deep linking to asset and asset tab
- [x] Add categories (expense categories)
- [x] Reconcile calculations - locked records for edits + snapshot of net worth?
- [x] rollup data (monthly)
- [x] Review labels on credit card such that outgoing (negative) implies spend and positive implies you paid off CC
- [x] Bill reminders
- [x] unit and integration tests from pull request
- [x] validation including UI tests, unit tests for UI loading, docker standup, local itests
- [x] Add income / schedule income
- [x] safeguards against "too new" dependencies from dependabot
- [x] embed privacy and terms policy version in jwt.  Enforce
- [x] Account recovery - password reset if you know totp, totp reset if you know password
- [x] Warn about using account numbers (UI warning modal)
- [x] add "download all data" json and CSV zip exporter
- [x] Tag line / branding
- [x] favicon, banner for LinkedIn, banner for YouTube, etc
- [x] bulk upload support (large 1000 record upload)
- [x] UI text search (find record by keyword)
- [x] UI amount search
- [x] Bulk fix categories
- [x] perspectives
- [x] net worth
- [x] budgeting and expense pies
- [x] Total net worth calculation / trending
- [x] Add support of looping updates (daily)
earmark support (monthly?)
- [x] Support collapse of accounts, perspectives, bills, income
Enable retention of sorted views of accounts, perspectives, bills, income
Enable retention of starting window
Allow trending against an asset (CPI, price of gas, etc)
one time use bill (future schedule, 1 time use)
Allow sorting of accounts (user settings)
- [x] add income category
create batch job for data grabs (indexes/prices, captcha cleanup)
- [x] Monthly bill additions / automation
- [x] cleanup UI
- [x] full reconcile of records (monthly capture?)
- [x] WAF geospacial lock-down
- [x] Aggregate grouping, such as credit card debt, retirement funds, liquid assets, crypto
Auto ingestion / scraping of data from remote services (gas trends, silver, gold, etc)
- [x] postgres auto vaccuuming - schedule, auto trigger
logging - log ingestion, formatting, context

fun money / daily money based on income expectations (payroll)
Payroll / payday features -> button to add?
- [x] Cookie to ignore TOTP challenge ? 

Add perspectives - graphical trending of net worth

Production ready:
- [x] support email + support form -> "support@plannedwallet.com"
- [x] legal disclaimer
- [x] Add aget of consent confirmation: "I confirm that I am at least 18 years of age, am a resident of the United States, and agree to the Terms of Use and Privacy & Security Policy."
Compare some example/external terms of use and privacy policy
- [x] Establish means to observe activity / record counts
- [x] Calculate space, size, cost (confirm limits for usage planning)

Premium features:
Add save settings for ordering (accounts and daily updates)
Add support for show perspective balance at top of viewing window
Day reminder (X days ahead of due bill)
Add Money-in / Money-out features (trending) for asset and snapshot accounts to track growth/change
Snapshot extraction of gold / silver / stocks / gas
earmark (capture bill, mark it against an account over time)
Add financial records over 365 days old
Bulk data upload (json/csv)
over 5 accounts
over 5 credit cards
over 10k records on a single account
Share account with other member (read/write sharing)
More than 10 snapshot assets
Bill notifications (email / phone alerts)
support / intake requests
move / merge accounts
Earmark / planning (5k for property taxes, earmark it for savings based on month)
payment plans (like buying an iPhone split up by 12 months)
Loan principal management
money transfers between accounts

Low priority:
- [x] Means to halt/disable signups (new accounts)
- [x] Allow to rename account
- [x] Add sql injection test cases
Help text / features on hover
Cookie retention removing need for TOTP
Phone app - input service
daily/weekly/monthly spend averages
Broke date calculation
email confirmation
SRE dashboards
Weekly reports of sign ups / total records
Full account recovery (seed phrase?)
Login attempt throttling (1 attempt per minute after 10 failed attempts)
cpi / dow jones / sp500 / nasdaq indexes
gold / silver
bitcoin / etherum
Reconcile mode (let me reconcile a month)
STRICT reconcile mode (force me to enter a full value and select the one that isn't available for reconcile yet)
Reconcile month records (aggregate view of these were reconciled on X date from starting value X, ending Y)
account deletion
record deletion
asset deletion
support sale / revenue tracking on snapshot accounts and asset accounts (win/loss)

Follow up:
scale postgres / review native eventual consistency models


Completed:
- [x] W3C Trace Context (traceparent, tracestate) across backend, frontend, gateway, and logs
- [x] X-Client-ID header

