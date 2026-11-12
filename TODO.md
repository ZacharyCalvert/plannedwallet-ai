Top features:
- [x] X-Client-ID
- [x] trace-context (W3C Trace Context across backend & frontend) 
- [x] CORS support
- [x] API review / plan
- [x] Database review / plan
- [x] add / update / reconcile checking records
- [x] Add credit card account types
snapshot account inputs - this account is worth x at this time
CAPTCHA for account setup
asset account -> like X bitcoin / total worth of the account
Add categories (expense categories)

Top priority:
Infrastructure + auto deploy to staging

Medium priority:
Add dependabot monitoring of npm, docker images, golang dependencies
branch protections, code owners, etc.
Captcha on account creation - creation only
Infrastructure + auto deploy to production
Disaster recovery / backups / automation reload of data
Data retention
graphical trending of net worth
rollup data (monthly)
Reconcile calculations - locked records for edits + snapshot of net worth?
account settings: reconcile account on month, or daily
Bill reminders
Monthly bill additions / automation
cleanup UI
full reconcile of records (monthly capture?)
WAF geospacial lock-down
Aggregate grouping, such as credit card debt, retirement funds, liquid assets, crypto
money transfers between accounts
Total net worth calculation / trending
Auto ingestion / scraping of data from remote services (gas trends, silver, gold, etc)
postgres auto vaccuuming - schedule, auto trigger
logging - log ingestion, formatting, context
text search (find record by keyword) -> text search in postgres?
fun money / daily money based on income expectations (payroll)
Payroll / payday features -> button to add?
Cookie to ignore TOTP challenge ? 
Snapshot extraction of gold / silver / stocks / gas

Premium features:
Bulk data upload (json/csv)
over 5 accounts
over 5 credit cards
over 100k records on a single account
Share account with other member (read/write sharing)
Creating your own snapshot trends?
Bill notifications (email / phone alerts)
support / intake requests
move / merge accounts
Earmark / planning (5k for property taxes, earmark it for savings based on month)
payment plans (like buying an iPhone split up by 12 months)

Low priority:
Allow to rename account
Add sql injection test cases
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
- W3C Trace Context (traceparent, tracestate) across backend, frontend, gateway, and logs
- X-Client-ID header

