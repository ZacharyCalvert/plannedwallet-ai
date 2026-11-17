from datetime import date, timedelta
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user


def test_bill_schedule_frequency_validation(api_context: APIRequestContext):
    """Validate valid and invalid combinations of frequency and schedule rules."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # Valid schedules across weekly, monthly, yearly, plus null schedule
    valid_payloads = [
        {"amount": -65.00, "frequency": "monthly", "last_date_paid": "2026-10-01", "recipient": "Internet Utility"},
        {
            "amount": -40.00,
            "frequency": "weekly",
            "schedule": {"type": "weekday", "scope": "week", "weekday": "friday"},
            "recipient": "Weekly Trash Pickup",
        },
        {
            "amount": -150.00,
            "frequency": "monthly",
            "schedule": {"type": "nth_weekday", "scope": "month", "nth": 3, "weekday": "thursday"},
            "recipient": "Third Thursday Club",
        },
        {
            "amount": -1200.00,
            "frequency": "monthly",
            "schedule": {"type": "nth_weekday", "scope": "month", "nth": 1, "weekday": "monday"},
            "recipient": "First Monday Rent",
        },
        {
            "amount": -95.00,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
            "recipient": "First Of Month Water",
        },
        {
            "amount": -500.00,
            "frequency": "yearly",
            "schedule": {"type": "day_of_period", "scope": "year", "day": 1},
            "recipient": "Annual Hosting First Of Year",
        },
        {
            "amount": -320.00,
            "frequency": "yearly",
            "schedule": {"type": "nth_weekday", "scope": "year", "nth": 1, "weekday": "sunday"},
            "recipient": "First Sunday Annual Dues",
        },
        {
            "amount": -450.00,
            "frequency": "monthly",
            "interval": 6,
            "last_date_paid": "2026-11-16",
            "recipient": "Semi-Annual Insurance",
        },
    ]

    for payload in valid_payloads:
        res = api_context.post("/api/v1/bills", data=payload, headers=headers)
        assert res.status == 201, f"Expected 201 for {payload}: {res.text()}"
        if payload.get("interval") == 6:
            body = res.json()
            assert body["interval"] == 6
            assert body["next_due_date"] == "2027-05-16"

    # 1st Sunday of the month on yearly frequency must be rejected; scope mismatch on monthly too
    invalid_payloads = [
        {
            "amount": -100.00,
            "frequency": "monthly",
            "interval": 0,
            "recipient": "Invalid Zero Interval",
        },
        {
            "amount": -100.00,
            "frequency": "yearly",
            "schedule": {"type": "nth_weekday", "scope": "month", "nth": 1, "weekday": "sunday"},
            "recipient": "Invalid Yearly Scope",
        },
        {
            "amount": -100.00,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "year", "day": 1},
            "recipient": "Invalid Monthly Scope",
        },
        {
            "amount": -100.00,
            "frequency": "weekly",
            "schedule": {"type": "nth_weekday", "scope": "week", "nth": 2, "weekday": "monday"},
            "recipient": "Invalid Weekly Nth",
        },
    ]

    for payload in invalid_payloads:
        res = api_context.post("/api/v1/bills", data=payload, headers=headers)
        assert res.status == 400, f"Expected 400 for invalid schedule {payload}: {res.status} {res.text()}"


def test_bill_account_category_apply_and_cascades(api_context: APIRequestContext):
    """Test creating a bill, linking to account & custom/global categories, applying to account, and cascade nullification."""
    _, owner_token = create_authenticated_user(api_context)
    _, other_token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {owner_token}"}
    other_headers = {"Authorization": f"Bearer {other_token}"}

    # Create checking account + snapshot account + custom category
    chk_acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Primary Checking", "type": "checking", "description": "Bill payment checking"},
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

    # Create unlinked bill first
    overdue_paid = (date.today() - timedelta(days=45)).isoformat()
    bill_res = api_context.post(
        "/api/v1/bills",
        data={
            "amount": -142.75,
            "last_date_paid": overdue_paid,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
            "recipient": "City Electric Utility",
            "description": "Monthly electric service",
            "reminder": True,
        },
        headers=headers,
    )
    assert bill_res.status == 201, f"Bill creation failed: {bill_res.text()}"
    bill = bill_res.json()
    assert bill["account_id"] is None
    assert bill["category_id"] is None
    assert bill["due_status"] == "overdue"

    # Reminder endpoint surfaces overdue reminder bill
    rem_res = api_context.get("/api/v1/bills/reminders", headers=headers)
    assert rem_res.status == 200
    assert any(b["id"] == bill["id"] for b in rem_res.json())

    # Bills cannot be shared: other user sees 404
    assert api_context.get(f"/api/v1/bills/{bill['id']}", headers=other_headers).status == 404

    # Cannot assign category without linked account, nor link to snapshot account
    no_acc_cat = api_context.put(
        f"/api/v1/bills/{bill['id']}",
        data={
            "category_id": custom_cat["id"],
            "amount": -142.75,
            "last_date_paid": overdue_paid,
            "frequency": "monthly",
            "recipient": "City Electric Utility",
        },
        headers=headers,
    )
    assert no_acc_cat.status == 400

    snap_link = api_context.put(
        f"/api/v1/bills/{bill['id']}",
        data={
            "account_id": snap_acc["id"],
            "amount": -142.75,
            "last_date_paid": overdue_paid,
            "frequency": "monthly",
            "recipient": "City Electric Utility",
        },
        headers=headers,
    )
    assert snap_link.status == 400

    # Link bill to checking account + custom category
    link_res = api_context.put(
        f"/api/v1/bills/{bill['id']}",
        data={
            "account_id": chk_acc["id"],
            "category_id": custom_cat["id"],
            "amount": -142.75,
            "last_date_paid": overdue_paid,
            "frequency": "monthly",
            "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
            "recipient": "City Electric Utility",
            "description": "Monthly electric service",
            "reminder": True,
        },
        headers=headers,
    )
    assert link_res.status == 200, f"Linking bill failed: {link_res.text()}"
    linked = link_res.json()
    assert linked["account_id"] == chk_acc["id"]
    assert linked["category_id"] == custom_cat["id"]

    # Apply to account creates checking record and updates last_date_paid
    apply_date = date.today().isoformat()
    apply_res = api_context.post(
        f"/api/v1/bills/{bill['id']}/apply",
        data={"date": apply_date},
        headers=headers,
    )
    assert apply_res.status == 201, f"Apply bill failed: {apply_res.text()}"
    applied = apply_res.json()
    assert applied["bill"]["last_date_paid"] == apply_date
    assert applied["record"]["account_id"] == chk_acc["id"]
    assert applied["record"]["category_id"] == custom_cat["id"]
    assert applied["record"]["value"] == -142.75
    assert applied["record"]["entity"] == "City Electric Utility"

    # Deleting custom category clears bill category_id while keeping bill and account_id valid
    del_cat = api_context.delete(f"/api/v1/categories/{custom_cat['id']}", headers=headers)
    assert del_cat.status == 204
    after_cat = api_context.get(f"/api/v1/bills/{bill['id']}", headers=headers).json()
    assert after_cat["category_id"] is None
    assert after_cat["account_id"] == chk_acc["id"]

    # Re-link using a global default category (Utilities) and confirm it works
    cats = api_context.get("/api/v1/categories", headers=headers).json()
    utilities = next(c for c in cats if c["name"] == "Utilities")
    relink = api_context.put(
        f"/api/v1/bills/{bill['id']}",
        data={
            "account_id": chk_acc["id"],
            "category_id": utilities["id"],
            "amount": -142.75,
            "last_date_paid": apply_date,
            "frequency": "monthly",
            "recipient": "City Electric Utility",
            "description": "Monthly electric service",
            "reminder": True,
        },
        headers=headers,
    )
    assert relink.status == 200
    assert relink.json()["category_id"] == utilities["id"]

    # Deleting linked account preserves bill while setting BOTH account_id and category_id to null
    del_acc = api_context.delete(f"/api/v1/accounts/{chk_acc['id']}", headers=headers)
    assert del_acc.status == 204
    after_acc = api_context.get(f"/api/v1/bills/{bill['id']}", headers=headers).json()
    assert after_acc["account_id"] is None
    assert after_acc["category_id"] is None
    assert after_acc["recipient"] == "City Electric Utility"


def test_bill_schedule_patterns_and_next_due_dates(api_context: APIRequestContext):
    """Validate next_due_date computation for common monthly, interval, and yearly schedule patterns."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    cases = [
        (
            "1st of the month",
            {
                "amount": -100.00,
                "frequency": "monthly",
                "last_date_paid": "2026-10-01",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 1},
                "recipient": "Monthly Rent",
            },
            "2026-11-01",
        ),
        (
            "15th of the month",
            {
                "amount": -85.00,
                "frequency": "monthly",
                "last_date_paid": "2026-11-15",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 15},
                "recipient": "Mid-Month Loan",
            },
            "2026-12-15",
        ),
        (
            "last of the month (Feb clamping)",
            {
                "amount": -60.00,
                "frequency": "monthly",
                "last_date_paid": "2026-01-31",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 31},
                "recipient": "Month-End Trash Service",
            },
            "2026-02-28",
        ),
        (
            "last of the month (Apr 30 clamping)",
            {
                "amount": -60.00,
                "frequency": "monthly",
                "last_date_paid": "2026-03-31",
                "schedule": {"type": "day_of_period", "scope": "month", "day": 31},
                "recipient": "Month-End Sewer Service",
            },
            "2026-04-30",
        ),
        (
            "every 6 months",
            {
                "amount": -450.00,
                "frequency": "monthly",
                "interval": 6,
                "last_date_paid": "2026-11-16",
                "recipient": "Semi-Annual Property Policy",
            },
            "2027-05-16",
        ),
        (
            "every 1st of the year",
            {
                "amount": -250.00,
                "frequency": "yearly",
                "last_date_paid": "2026-01-01",
                "schedule": {"type": "day_of_period", "scope": "year", "day": 1},
                "recipient": "New Year Association Dues",
            },
            "2027-01-01",
        ),
        (
            "April 14th every year",
            {
                "amount": -300.00,
                "frequency": "yearly",
                "last_date_paid": "2026-04-14",
                "schedule": {"type": "month_day", "scope": "year", "month": 4, "day": 14},
                "recipient": "Spring Tax Prep Fee",
            },
            "2027-04-14",
        ),
        (
            "1st Monday of the month",
            {
                "amount": -125.00,
                "frequency": "monthly",
                "last_date_paid": "2026-11-02",
                "schedule": {"type": "nth_weekday", "scope": "month", "nth": 1, "weekday": "monday"},
                "recipient": "First Monday Grounds Care",
            },
            "2026-12-07",
        ),
        (
            "Last Monday of the month",
            {
                "amount": -140.00,
                "frequency": "monthly",
                "last_date_paid": "2026-10-26",
                "schedule": {"type": "nth_weekday", "scope": "month", "nth": -1, "weekday": "monday"},
                "recipient": "Last Monday Maintenance",
            },
            "2026-11-30",
        ),
        (
            "November 1st every year",
            {
                "amount": -520.00,
                "frequency": "yearly",
                "last_date_paid": "2026-11-01",
                "schedule": {"type": "month_day", "scope": "year", "month": 11, "day": 1},
                "recipient": "Annual Homeowner Dues",
            },
            "2027-11-01",
        ),
    ]

    for label, payload, expected_due in cases:
        res = api_context.post("/api/v1/bills", data=payload, headers=headers)
        assert res.status == 201, f"{label} create failed: {res.status} {res.text()}"
        created = res.json()
        assert created["next_due_date"] == expected_due, (
            f"{label}: expected next_due_date={expected_due}, got {created['next_due_date']}"
        )

        fetched = api_context.get(f"/api/v1/bills/{created['id']}", headers=headers)
        assert fetched.status == 200
        assert fetched.json()["next_due_date"] == expected_due

