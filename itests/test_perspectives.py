import pytest
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

def test_perspective_creation_and_view(api_context: APIRequestContext):
    """
    Test creating a financial perspective aggregating 3 snapshot accounts
    and a checking account, and verify the viewing window metrics:
    - Current total value
    - Aggregated financial trend
    - Average daily spend
    - Year-over-year return rate
    - Ruined date forecasting
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Create 3 snapshot accounts (generic naming per rules)
    snap1_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Retirement Horizon", "type": "snapshot", "description": "Primary Retirement"},
        headers=headers,
    )
    assert snap1_res.status == 201
    snap1_id = snap1_res.json()["id"]

    snap2_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Secondary Reserve", "type": "snapshot", "description": "Secondary Portfolio"},
        headers=headers,
    )
    assert snap2_res.status == 201
    snap2_id = snap2_res.json()["id"]

    snap3_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Employer Plan", "type": "snapshot", "description": "Workplace Retirement Plan"},
        headers=headers,
    )
    assert snap3_res.status == 201
    snap3_id = snap3_res.json()["id"]

    # 2. Add valuations to snapshot accounts across dates
    # snap1: 2025-01-01 = 10000, 2026-01-01 = 11000
    api_context.post(f"/api/v1/accounts/{snap1_id}/snapshots", data={"value": 10000.0, "snapshot_date": "2025-01-01"}, headers=headers)
    api_context.post(f"/api/v1/accounts/{snap1_id}/snapshots", data={"value": 11000.0, "snapshot_date": "2026-01-01"}, headers=headers)

    # snap2: 2025-01-01 = 5000, 2026-01-01 = 6000
    api_context.post(f"/api/v1/accounts/{snap2_id}/snapshots", data={"value": 5000.0, "snapshot_date": "2025-01-01"}, headers=headers)
    api_context.post(f"/api/v1/accounts/{snap2_id}/snapshots", data={"value": 6000.0, "snapshot_date": "2026-01-01"}, headers=headers)

    # snap3: 2025-01-01 = 20000, 2026-01-01 = 23000
    api_context.post(f"/api/v1/accounts/{snap3_id}/snapshots", data={"value": 20000.0, "snapshot_date": "2025-01-01"}, headers=headers)
    api_context.post(f"/api/v1/accounts/{snap3_id}/snapshots", data={"value": 23000.0, "snapshot_date": "2026-01-01"}, headers=headers)

    # 3. Create transactional checking account with debits/expenses
    chk_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Primary Everyday Checking", "type": "checking", "description": "Household checking"},
        headers=headers,
    )
    assert chk_res.status == 201
    chk_id = chk_res.json()["id"]

    # Add transactions: deposit 5000 on 2026-01-01, spend 300 total across 10 days
    api_context.post(f"/api/v1/accounts/{chk_id}/records", data={"value": 5000.0, "date_sent": "2026-01-01", "entity": "Employer Payroll"}, headers=headers)
    api_context.post(f"/api/v1/accounts/{chk_id}/records", data={"value": -150.0, "date_sent": "2026-01-02", "entity": "Supermarket"}, headers=headers)
    api_context.post(f"/api/v1/accounts/{chk_id}/records", data={"value": -150.0, "date_sent": "2026-01-11", "entity": "Corner Grocery"}, headers=headers)

    # 4. Create perspective aggregating these 4 accounts with all analytics enabled
    persp_res = api_context.post(
        "/api/v1/perspectives",
        data={
            "name": "Combined Wealth & Retirement",
            "description": "Consolidated view of retirement and liquid savings",
            "account_ids": [snap1_id, snap2_id, snap3_id, chk_id],
            "settings": {
                "show_financial_trend": True,
                "calculate_average_daily_spend": True,
                "calculate_average_return_yoy": True,
                "track_ruined_date": True,
            },
        },
        headers=headers,
    )
    assert persp_res.status == 201, f"Failed to create perspective: {persp_res.text()}"
    perspective = persp_res.json()
    persp_id = perspective["id"]
    assert perspective["name"] == "Combined Wealth & Retirement"
    assert len(perspective["account_ids"]) == 4

    # 5. Retrieve perspective viewing window
    view_res = api_context.get(f"/api/v1/perspectives/{persp_id}/view", headers=headers)
    assert view_res.status == 200, f"Failed to get view: {view_res.text()}"
    view = view_res.json()

    # Total = 11000 + 6000 + 23000 (snapshots) + (5000 - 300 = 4700 checking) = 44700.0
    assert view["current_total_value"] == 44700.0
    assert len(view["accounts"]) == 4

    # Trend verification
    assert "trend" in view
    assert len(view["trend"]) >= 2
    # First point on 2025-01-01: 10000 + 5000 + 20000 = 35000.0
    assert view["trend"][0]["date"] == "2025-01-01"
    assert view["trend"][0]["total_value"] == 35000.0

    # Average daily spend verification (300 spent over 10 days from Jan 2 to Jan 11 = 30/day)
    assert view["average_daily_spend"] is not None
    assert round(view["average_daily_spend"], 1) == 30.0

    # YoY return verification (Growth from 35000 on 2025-01-01 to ~44700 on 2026-01-11)
    assert view["average_return_yoy"] is not None
    assert view["average_return_yoy"] > 0.0

    # Ruined date verification
    assert view["ruined_date"] is not None
    assert view["days_until_ruined"] is not None
    # 44700 / 30 = 1490 days
    assert view["days_until_ruined"] == 1490


def test_perspective_settings_and_account_linking(api_context: APIRequestContext):
    """
    Test updating settings, custom daily spend simulation, and linking/unlinking accounts.
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # Create account
    acc1_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Savings Snapshot", "type": "snapshot", "description": "Valuations"},
        headers=headers,
    )
    acc1_id = acc1_res.json()["id"]

    acc2_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Secondary Savings Snapshot", "type": "snapshot", "description": "Valuations"},
        headers=headers,
    )
    acc2_id = acc2_res.json()["id"]

    # Create perspective without initial accounts
    p_res = api_context.post(
        "/api/v1/perspectives",
        data={"name": "Dynamic Perspective", "description": "Test dynamic settings"},
        headers=headers,
    )
    assert p_res.status == 201
    p_id = p_res.json()["id"]

    # 1. Update settings with a custom daily spend override ($100/day for job-loss forecasting)
    put_settings = api_context.put(
        f"/api/v1/perspectives/{p_id}/settings",
        data={
            "show_financial_trend": True,
            "calculate_average_daily_spend": False,
            "calculate_average_return_yoy": False,
            "track_ruined_date": True,
            "custom_daily_spend": 100.0,
        },
        headers=headers,
    )
    assert put_settings.status == 200
    settings = put_settings.json()
    assert settings["custom_daily_spend"] == 100.0
    assert settings["track_ruined_date"] is True

    # 2. Link account 1
    link_res = api_context.post(
        f"/api/v1/perspectives/{p_id}/accounts",
        data={"account_id": acc1_id},
        headers=headers,
    )
    assert link_res.status == 201

    # Duplicate link returns 409
    dup_res = api_context.post(
        f"/api/v1/perspectives/{p_id}/accounts",
        data={"account_id": acc1_id},
        headers=headers,
    )
    assert dup_res.status == 409

    # Link account 2
    api_context.post(
        f"/api/v1/perspectives/{p_id}/accounts",
        data={"account_id": acc2_id},
        headers=headers,
    )

    # 3. List linked accounts
    list_accs = api_context.get(f"/api/v1/perspectives/{p_id}/accounts", headers=headers)
    assert list_accs.status == 200
    accs = list_accs.json()
    assert len(accs) == 2
    linked_ids = [a["id"] for a in accs]
    assert acc1_id in linked_ids and acc2_id in linked_ids

    # 4. Unlink account 1
    unlink_res = api_context.delete(f"/api/v1/perspectives/{p_id}/accounts/{acc1_id}", headers=headers)
    assert unlink_res.status == 204

    # Verify list has only account 2
    list_after = api_context.get(f"/api/v1/perspectives/{p_id}/accounts", headers=headers)
    assert list_after.status == 200
    assert len(list_after.json()) == 1
    assert list_after.json()[0]["id"] == acc2_id


def test_perspective_sharing_and_access_control(api_context: APIRequestContext):
    """
    Test perspective multi-user sharing and RBAC (read, edit, admin).
    """
    user_a, token_a = create_authenticated_user(api_context)
    user_b, token_b = create_authenticated_user(api_context)
    headers_a = {"Authorization": f"Bearer {token_a}"}
    headers_b = {"Authorization": f"Bearer {token_b}"}

    # 1. User A creates perspective
    p_res = api_context.post(
        "/api/v1/perspectives",
        data={"name": "Family Net Worth", "description": "Shared Family Perspective"},
        headers=headers_a,
    )
    assert p_res.status == 201
    p_id = p_res.json()["id"]

    # 2. User B tries to view before sharing -> 404
    res_b_view = api_context.get(f"/api/v1/perspectives/{p_id}", headers=headers_b)
    assert res_b_view.status == 404

    # 3. User A shares perspective with User B with 'read' access
    grant_res = api_context.post(
        f"/api/v1/perspectives/{p_id}/access",
        data={"user_id": user_b["id"], "access_level": "read"},
        headers=headers_a,
    )
    assert grant_res.status == 200
    grant = grant_res.json()
    assert grant["access_level"] == "read"

    # User B can now view
    res_b_ok = api_context.get(f"/api/v1/perspectives/{p_id}", headers=headers_b)
    assert res_b_ok.status == 200
    assert res_b_ok.json()["access_level"] == "read"

    # User B tries to mutate settings -> 403 Forbidden
    res_b_mutate = api_context.put(
        f"/api/v1/perspectives/{p_id}/settings",
        data={"show_financial_trend": False},
        headers=headers_b,
    )
    assert res_b_mutate.status == 403

    # 4. User A upgrades User B to 'edit'
    api_context.post(
        f"/api/v1/perspectives/{p_id}/access",
        data={"user_id": user_b["id"], "access_level": "edit"},
        headers=headers_a,
    )

    # User B can now update settings
    res_b_edit_ok = api_context.put(
        f"/api/v1/perspectives/{p_id}/settings",
        data={"show_financial_trend": False},
        headers=headers_b,
    )
    assert res_b_edit_ok.status == 200
    assert res_b_edit_ok.json()["show_financial_trend"] is False

    # 5. User A revokes User B access
    revoke_res = api_context.delete(f"/api/v1/perspectives/{p_id}/access/{user_b['id']}", headers=headers_a)
    assert revoke_res.status == 204

    # User B gets 404 again
    res_b_after = api_context.get(f"/api/v1/perspectives/{p_id}", headers=headers_b)
    assert res_b_after.status == 404


def test_perspective_credit_card_transfer_deduction_and_windows(api_context: APIRequestContext):
    """
    Test that checking/savings debits matching a credit card income within 35 days
    are recognized as credit card payment transfers and excluded from spend calculations,
    and verify 30d, 90d, 365d spend and runway fields.
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Create checking and credit card accounts
    chk_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Everyday Checking", "type": "checking", "description": "Primary checking"},
        headers=headers,
    )
    assert chk_res.status == 201
    chk_id = chk_res.json()["id"]

    cc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Travel Credit Card", "type": "credit card", "description": "Everyday card"},
        headers=headers,
    )
    assert cc_res.status == 201
    cc_id = cc_res.json()["id"]

    # 2. Add records
    # Initial deposit into checking
    api_context.post(f"/api/v1/accounts/{chk_id}/records", data={"value": 5000.0, "date_sent": "2026-01-01", "entity": "Payroll Deposit"}, headers=headers)
    # Checking spend of 521.45 on Jan 1
    api_context.post(f"/api/v1/accounts/{chk_id}/records", data={"value": -521.45, "date_sent": "2026-01-01", "entity": "Credit Card Payment"}, headers=headers)
    # Credit card income of 521.45 on Feb 2 (32 days later <= 35 days)
    api_context.post(f"/api/v1/accounts/{cc_id}/records", data={"value": 521.45, "date_sent": "2026-02-02", "entity": "Automatic Payment"}, headers=headers)
    # Credit card purchase of 100.00 on Jan 15
    api_context.post(f"/api/v1/accounts/{cc_id}/records", data={"value": -100.0, "date_sent": "2026-01-15", "entity": "Grocery Market"}, headers=headers)

    # 3. Create perspective linking checking and credit card
    p_res = api_context.post(
        "/api/v1/perspectives",
        data={
            "name": "Combined Cash & Credit",
            "description": "Checking and Card perspective",
            "account_ids": [chk_id, cc_id],
            "settings": {
                "show_financial_trend": True,
                "calculate_average_daily_spend": True,
                "track_ruined_date": True,
            },
        },
        headers=headers,
    )
    assert p_res.status == 201
    p_id = p_res.json()["id"]

    # 4. Fetch perspective view
    view_res = api_context.get(f"/api/v1/perspectives/{p_id}/view", headers=headers)
    assert view_res.status == 200
    view = view_res.json()

    # Total savings: (5000 - 521.45) + (521.45 - 100) = 4900.0
    assert view["current_total_value"] == 4900.0

    # The 521.45 payment from checking to credit card within 32 days is excluded.
    # The only remaining spend is 100.0 over 1 day = 100.0/day.
    assert view["average_daily_spend"] is not None
    assert round(view["average_daily_spend"], 1) == 100.0

    # Days until ruined: 4900 / 100 = 49 days
    assert view["days_until_ruined"] == 49
    assert view["ruined_date"] is not None

    # Multi-window spend and runway metrics
    assert "daily_spend_30d" in view
    assert "daily_spend_90d" in view
    assert "daily_spend_365d" in view
    assert "days_until_ruined_30d" in view
    assert "days_until_ruined_90d" in view
    assert "days_until_ruined_365d" in view
    assert "ruined_date_30d" in view
    assert "ruined_date_90d" in view
    assert "ruined_date_365d" in view

