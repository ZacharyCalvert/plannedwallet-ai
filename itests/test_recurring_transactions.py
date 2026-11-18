from datetime import date, timedelta
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user


def test_recurring_schedule_frequency_validation(api_context: APIRequestContext):
    """Validate valid and invalid combinations of frequency and schedule rules."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    valid_payloads = [
        {"amount": -65.00, "frequency": "monthly", "last_date_applied": "2026-10-01", "entity": "Internet Utility"},
        {
            "amount": 1800.00,
            "frequency": "weekly",
            "schedule": {"type": "weekday", "scope": "week", "weekday": "friday"},
            "name": "Paycheck",
            "entity": "Employer Payroll",
        },
        {
            "amount": -450.00,
            "frequency": "monthly",
            "interval": 6,
            "last_date_applied": "2026-11-16",
            "entity": "Semi-Annual Insurance",
        },
    ]

    for payload in valid_payloads:
        res = api_context.post("/api/v1/recurring-transactions", data=payload, headers=headers)
        assert res.status == 201, f"Expected 201 for {payload}: {res.text()}"
        if payload.get("interval") == 6:
            body = res.json()
            assert body["interval"] == 6
            assert body["next_due_date"] == "2027-05-16"

    invalid_payloads = [
        {"amount": 0, "frequency": "monthly", "entity": "Zero Amount"},
        {"amount": -100.00, "frequency": "monthly", "interval": 0, "entity": "Invalid Zero Interval"},
        {
            "amount": -100.00,
            "frequency": "yearly",
            "schedule": {"type": "nth_weekday", "scope": "month", "nth": 1, "weekday": "sunday"},
            "entity": "Invalid Yearly Scope",
        },
        {
            "amount": -100.00,
            "frequency": "weekly",
            "schedule": {"type": "nth_weekday", "scope": "week", "nth": 2, "weekday": "monday"},
            "entity": "Invalid Weekly Nth",
        },
    ]

    for payload in invalid_payloads:
        res = api_context.post("/api/v1/recurring-transactions", data=payload, headers=headers)
        assert res.status == 400, f"Expected 400 for invalid payload {payload}: {res.status} {res.text()}"


def test_recurring_bills_incomes_apply_names_and_cascades(api_context: APIRequestContext):
    """Test negative (bill) and positive (income) recurring transactions, short names, reminders, apply, and cascades."""
    _, owner_token = create_authenticated_user(api_context)
    _, other_token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {owner_token}"}
    other_headers = {"Authorization": f"Bearer {other_token}"}

    chk_acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Primary Checking", "type": "checking", "description": "Recurring checking"},
        headers=headers,
    ).json()
    snap_acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Retirement Snapshot", "type": "snapshot", "description": "Non-record account"},
        headers=headers,
    ).json()
    custom_cat = api_context.post(
        "/api/v1/categories",
        data={"name": "Municipal Utilities"},
        headers=headers,
    ).json()

    # Short name >= 20 chars rejected
    too_long = api_context.post(
        "/api/v1/recurring-transactions",
        data={
            "amount": -125.00,
            "frequency": "monthly",
            "name": "12345678901234567890",
            "entity": "Regional Electric Utility",
        },
        headers=headers,
    )
    assert too_long.status == 400

    # Create overdue bill (< 0) and overdue income (> 0)
    overdue_applied = (date.today() - timedelta(days=45)).isoformat()
    bill_res = api_context.post(
        "/api/v1/recurring-transactions",
        data={
            "amount": -142.75,
            "last_date_applied": overdue_applied,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
            "name": "Electric",
            "entity": "City Electric Utility",
            "description": "Monthly electric service",
            "reminder": True,
        },
        headers=headers,
    )
    assert bill_res.status == 201, bill_res.text()
    bill = bill_res.json()
    assert bill["name"] == "Electric"
    assert bill["due_status"] == "overdue"

    income_res = api_context.post(
        "/api/v1/recurring-transactions",
        data={
            "account_id": chk_acc["id"],
            "amount": 2500.00,
            "last_date_applied": overdue_applied,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 15},
            "name": "Paycheck",
            "entity": "Employer Payroll",
            "description": "Monthly direct deposit",
            "reminder": True,
        },
        headers=headers,
    )
    assert income_res.status == 201, income_res.text()
    income = income_res.json()

    # Private isolation + reminders endpoint
    assert api_context.get(f"/api/v1/recurring-transactions/{bill['id']}", headers=other_headers).status == 404
    rem_ids = {item["id"] for item in api_context.get("/api/v1/recurring-transactions/reminders", headers=headers).json()}
    assert bill["id"] in rem_ids and income["id"] in rem_ids

    # Reject category without account & snapshot link
    assert api_context.put(
        f"/api/v1/recurring-transactions/{bill['id']}",
        data={
            "category_id": custom_cat["id"],
            "amount": -142.75,
            "frequency": "monthly",
            "name": "Electric",
            "entity": "City Electric Utility",
        },
        headers=headers,
    ).status == 400
    assert api_context.put(
        f"/api/v1/recurring-transactions/{bill['id']}",
        data={
            "account_id": snap_acc["id"],
            "amount": -142.75,
            "frequency": "monthly",
            "name": "Electric",
            "entity": "City Electric Utility",
        },
        headers=headers,
    ).status == 400

    # Link bill and apply both bill & income
    link_res = api_context.put(
        f"/api/v1/recurring-transactions/{bill['id']}",
        data={
            "account_id": chk_acc["id"],
            "category_id": custom_cat["id"],
            "amount": -142.75,
            "last_date_applied": overdue_applied,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
            "name": "Electric",
            "entity": "City Electric Utility",
            "description": "Monthly electric service",
            "reminder": True,
        },
        headers=headers,
    )
    assert link_res.status == 200

    apply_date = date.today().isoformat()
    applied_bill = api_context.post(
        f"/api/v1/recurring-transactions/{bill['id']}/apply",
        data={"date": apply_date},
        headers=headers,
    ).json()
    assert applied_bill["record"]["value"] == -142.75
    assert applied_bill["record"]["entity"] == "City Electric Utility"
    assert applied_bill["recurring_transaction"]["last_date_applied"] == apply_date

    applied_inc = api_context.post(
        f"/api/v1/recurring-transactions/{income['id']}/apply",
        data={"date": apply_date},
        headers=headers,
    ).json()
    assert applied_inc["record"]["value"] == 2500.00
    assert applied_inc["record"]["entity"] == "Employer Payroll"
    assert applied_inc["recurring_transaction"]["last_date_applied"] == apply_date

    # Category and account deletion cascades
    assert api_context.delete(f"/api/v1/categories/{custom_cat['id']}", headers=headers).status == 204
    after_cat = api_context.get(f"/api/v1/recurring-transactions/{bill['id']}", headers=headers).json()
    assert after_cat["category_id"] is None and after_cat["account_id"] == chk_acc["id"]

    assert api_context.delete(f"/api/v1/accounts/{chk_acc['id']}", headers=headers).status == 204
    after_acc = api_context.get(f"/api/v1/recurring-transactions/{bill['id']}", headers=headers).json()
    assert after_acc["account_id"] is None and after_acc["category_id"] is None


def test_recurring_schedule_patterns_and_next_due_dates(api_context: APIRequestContext):
    """Validate next_due_date computation for common monthly, interval, and yearly schedule patterns."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    cases = [
        (
            "1st of the month",
            {
                "amount": -100.00,
                "frequency": "monthly",
                "last_date_applied": "2026-10-01",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
                "entity": "Monthly Rent",
            },
            "2026-11-01",
        ),
        (
            "15th of the month",
            {
                "amount": -85.00,
                "frequency": "monthly",
                "last_date_applied": "2026-11-15",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 15},
                "entity": "Mid-Month Loan",
            },
            "2026-12-15",
        ),
        (
            "last of the month (Feb clamping)",
            {
                "amount": -60.00,
                "frequency": "monthly",
                "last_date_applied": "2026-01-31",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 31},
                "entity": "Month-End Trash Service",
            },
            "2026-02-28",
        ),
        (
            "last of the month (Apr 30 clamping)",
            {
                "amount": -60.00,
                "frequency": "monthly",
                "last_date_applied": "2026-03-31",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 31},
                "entity": "Month-End Sewer Service",
            },
            "2026-04-30",
        ),
        (
            "every 6 months",
            {
                "amount": -450.00,
                "frequency": "monthly",
                "interval": 6,
                "last_date_applied": "2026-11-16",
                "entity": "Semi-Annual Property Policy",
            },
            "2027-05-16",
        ),
        (
            "every 1st of the year",
            {
                "amount": -250.00,
                "frequency": "yearly",
                "last_date_applied": "2026-01-01",
                "schedule": {"type": "day_of_period", "scope": "year", "day": 1},
                "entity": "New Year Association Dues",
            },
            "2027-01-01",
        ),
        (
            "April 14th every year",
            {
                "amount": -300.00,
                "frequency": "yearly",
                "last_date_applied": "2026-04-14",
                "schedule": {"type": "month_day", "scope": "year", "month": 4, "day": 14},
                "entity": "Spring Tax Prep Fee",
            },
            "2027-04-14",
        ),
        (
            "1st Monday of the month",
            {
                "amount": -125.00,
                "frequency": "monthly",
                "last_date_applied": "2026-11-02",
                "schedule": {"type": "nth_weekday", "scope": "month", "nth": 1, "weekday": "monday"},
                "entity": "First Monday Grounds Care",
            },
            "2026-12-07",
        ),
        (
            "Last Monday of the month",
            {
                "amount": -140.00,
                "frequency": "monthly",
                "last_date_applied": "2026-10-26",
                "schedule": {"type": "nth_weekday", "scope": "month", "nth": -1, "weekday": "monday"},
                "entity": "Last Monday Maintenance",
            },
            "2026-11-30",
        ),
        (
            "November 1st every year",
            {
                "amount": -520.00,
                "frequency": "yearly",
                "last_date_applied": "2026-11-01",
                "schedule": {"type": "month_day", "scope": "year", "month": 11, "day": 1},
                "entity": "Annual Homeowner Dues",
            },
            "2027-11-01",
        ),
    ]

    for label, payload, expected_due in cases:
        res = api_context.post("/api/v1/recurring-transactions", data=payload, headers=headers)
        assert res.status == 201, f"{label} create failed: {res.status} {res.text()}"
        created = res.json()
        assert created["next_due_date"] == expected_due, (
            f"{label}: expected next_due_date={expected_due}, got {created['next_due_date']}"
        )
