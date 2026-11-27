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
                "show_categories": True,
                "calculate_average_daily_spend": True,
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

    # show_categories enabled and average_return_yoy removed
    assert view["settings"]["show_categories"] is True
    assert "average_return_yoy" not in view

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
            "show_categories": False,
            "calculate_average_daily_spend": False,
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

    # 2b. Create and link a recurring bill to the perspective, then verify & unlink
    bill_res = api_context.post(
        "/api/v1/recurring-transactions",
        data={
            "amount": -75.0,
            "frequency": "monthly",
            "entity": "Internet Service",
            "description": "Monthly internet",
        },
        headers=headers,
    )
    assert bill_res.status == 201, f"Create bill failed: {bill_res.text()}"
    bill_id = bill_res.json()["id"]

    link_rt = api_context.post(
        f"/api/v1/perspectives/{p_id}/recurring-transactions",
        data={"recurring_transaction_id": bill_id},
        headers=headers,
    )
    assert link_rt.status == 201, f"Link recurring tx failed: {link_rt.text()}"
    list_rt = api_context.get(f"/api/v1/perspectives/{p_id}/recurring-transactions", headers=headers)
    assert list_rt.status == 200
    assert any(r["id"] == bill_id for r in list_rt.json())

    unlink_rt = api_context.delete(
        f"/api/v1/perspectives/{p_id}/recurring-transactions/{bill_id}",
        headers=headers,
    )
    assert unlink_rt.status == 204

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

    # 5. Relink account 1 and verify both accounts are linked again
    relink_res = api_context.post(
        f"/api/v1/perspectives/{p_id}/accounts",
        data={"account_id": acc1_id},
        headers=headers,
    )
    assert relink_res.status == 201, f"Relink account 1 failed: {relink_res.text()}"
    list_relinked = api_context.get(f"/api/v1/perspectives/{p_id}/accounts", headers=headers)
    assert list_relinked.status == 200
    relinked_ids = [a["id"] for a in list_relinked.json()]
    assert acc1_id in relinked_ids and acc2_id in relinked_ids

    # 6. Update settings again (toggle show_categories / track_ruined_date / custom_daily_spend) and verify via GET /settings
    put_settings_2 = api_context.put(
        f"/api/v1/perspectives/{p_id}/settings",
        data={
            "show_financial_trend": True,
            "show_categories": True,
            "track_ruined_date": False,
            "custom_daily_spend": 55.25,
        },
        headers=headers,
    )
    assert put_settings_2.status == 200
    get_settings = api_context.get(f"/api/v1/perspectives/{p_id}/settings", headers=headers)
    assert get_settings.status == 200
    settings_2 = get_settings.json()
    assert settings_2["show_categories"] is True
    assert settings_2["track_ruined_date"] is False
    assert settings_2["custom_daily_spend"] == 55.25

    # 7. Delete perspective and verify it is removed while linked accounts remain intact
    del_res = api_context.delete(f"/api/v1/perspectives/{p_id}", headers=headers)
    assert del_res.status == 204, f"Delete perspective failed: {del_res.text()}"
    get_deleted = api_context.get(f"/api/v1/perspectives/{p_id}", headers=headers)
    assert get_deleted.status == 404
    list_perspectives = api_context.get("/api/v1/perspectives", headers=headers)
    assert list_perspectives.status == 200
    assert all(p["id"] != p_id for p in (list_perspectives.json() or []))
    assert api_context.get(f"/api/v1/accounts/{acc1_id}", headers=headers).status == 200
    assert api_context.get(f"/api/v1/accounts/{acc2_id}", headers=headers).status == 200


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


def test_perspective_trend_includes_asset_account_values(api_context: APIRequestContext):
    """Ensure asset account store valuations are included in perspective trend points and reconcile with current_total_value."""
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    chk_id = api_context.post(
        "/api/v1/accounts",
        data={"name": "Household Checking", "type": "checking", "description": "Liquid cash"},
        headers=headers,
    ).json()["id"]
    snp_id = api_context.post(
        "/api/v1/accounts",
        data={"name": "Retirement Account", "type": "snapshot", "description": "Long-term plan"},
        headers=headers,
    ).json()["id"]
    ast_id = api_context.post(
        "/api/v1/accounts",
        data={"name": "Commodity Reserve", "type": "asset", "description": "Asset holdings"},
        headers=headers,
    ).json()["id"]

    # Checking: +2000 on 2026-01-01, -500 on 2026-01-15 -> 1500
    assert api_context.post(
        f"/api/v1/accounts/{chk_id}/records",
        data={"value": 2000.0, "date_sent": "2026-01-01", "entity": "Employer Payroll"},
        headers=headers,
    ).status == 201
    assert api_context.post(
        f"/api/v1/accounts/{chk_id}/records",
        data={"value": -500.0, "date_sent": "2026-01-15", "entity": "Corner Grocery"},
        headers=headers,
    ).status == 201

    # Snapshot: 5000 on 2026-01-05 -> 5000
    assert api_context.post(
        f"/api/v1/accounts/{snp_id}/snapshots",
        data={"value": 5000.0, "snapshot_date": "2026-01-05"},
        headers=headers,
    ).status == 201

    # Asset store: qty 10 @ 100 on 2026-01-01 -> 1000; qty 12 on 2026-01-10 -> 1200; price 125 on 2026-01-20 -> 1500
    asset_def_id = api_context.post(
        "/api/v1/assets",
        data={"label": f"commodity_{user['username'][:6]}", "description": "Index Basket"},
        headers=headers,
    ).json()["id"]
    assert api_context.post(
        f"/api/v1/assets/{asset_def_id}/values",
        data={"value": 100.0, "value_date": "2026-01-01"},
        headers=headers,
    ).status == 201
    assert api_context.post(
        f"/api/v1/assets/{asset_def_id}/values",
        data={"value": 125.0, "value_date": "2026-01-20"},
        headers=headers,
    ).status == 201
    store_id = api_context.post(
        f"/api/v1/accounts/{ast_id}/stores",
        data={"asset_id": asset_def_id, "label": "Primary Vault", "description": "Stored holdings"},
        headers=headers,
    ).json()["id"]
    assert api_context.post(
        f"/api/v1/accounts/{ast_id}/stores/{store_id}/records",
        data={"quantity": 10.0, "record_date": "2026-01-01"},
        headers=headers,
    ).status == 201
    assert api_context.post(
        f"/api/v1/accounts/{ast_id}/stores/{store_id}/records",
        data={"quantity": 12.0, "record_date": "2026-01-10"},
        headers=headers,
    ).status == 201

    persp_res = api_context.post(
        "/api/v1/perspectives",
        data={
            "name": "Total Net Worth",
            "description": "Combined checking, snapshot, and asset portfolio",
            "account_ids": [chk_id, snp_id, ast_id],
            "settings": {"show_financial_trend": True},
        },
        headers=headers,
    )
    assert persp_res.status == 201
    persp_id = persp_res.json()["id"]

    view_res = api_context.get(f"/api/v1/perspectives/{persp_id}/view", headers=headers)
    assert view_res.status == 200
    view = view_res.json()

    # Aggregate = 1500 checking + 5000 snapshot + 1500 asset = 8000.0
    assert view["current_total_value"] == 8000.0
    trend = view["trend"]
    assert [pt["date"] for pt in trend] == [
        "2026-01-01",
        "2026-01-05",
        "2026-01-10",
        "2026-01-15",
        "2026-01-20",
    ]
    assert [pt["total_value"] for pt in trend] == [3000.0, 8000.0, 8200.0, 7700.0, 8000.0]
    assert trend[-1]["total_value"] == view["current_total_value"]
    assert api_context.delete(f"/api/v1/users/{user['username']}", headers=headers).status == 204
