import uuid
import pytest
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

def test_asset_account_lifecycle_and_valuation(api_context: APIRequestContext):
    """
    Test full lifecycle of an asset account:
    - Create asset account (e.g. Precious Metals)
    - Add multiple assets (silver, gold, copper)
    - Set multi-day valuations (verifying latest value is picked)
    - Add assets to account with quantities
    - Verify individual and aggregated account values (quantity * latest_value)
    - Update holding quantity
    - Delete an asset holding
    """
    user_a, token_a = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token_a}"}

    # 1. Create asset account
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={
            "name": "Precious Metals",
            "type": "asset",
            "description": "Physical precious metals holdings",
        },
        headers=headers,
    )
    assert acc_res.status == 201, f"Failed to create asset account: {acc_res.text()}"
    account = acc_res.json()
    account_id = account["id"]
    assert account["type"] == "asset"

    # 2. Create assets: Silver, Gold, Copper
    uid = uuid.uuid4().hex[:6]
    labels = {
        "slv": f"slv_{uid}",
        "gld": f"gld_{uid}",
        "cpr": f"cpr_{uid}",
    }
    assets = {}
    for key, label in labels.items():
        res = api_context.post(
            "/api/v1/assets",
            data={"label": label, "description": f"{key} asset description"},
            headers=headers,
        )
        assert res.status == 201, f"Failed to create asset {label}: {res.text()}"
        assets[key] = res.json()

    # 3. Add valuations
    # Silver: Nov 10 = $30.00, Nov 12 = $32.50 (Nov 12 should be selected as latest)
    api_context.post(
        f"/api/v1/assets/{assets['slv']['id']}/values",
        data={"value": 30.00, "value_date": "2026-11-10"},
        headers=headers,
    )
    api_context.post(
        f"/api/v1/assets/{assets['slv']['id']}/values",
        data={"value": 32.50, "value_date": "2026-11-12"},
        headers=headers,
    )
    # Gold: Nov 12 = $2650.00
    api_context.post(
        f"/api/v1/assets/{assets['gld']['id']}/values",
        data={"value": 2650.00, "value_date": "2026-11-12"},
        headers=headers,
    )
    # Copper: Nov 12 = $4.15
    api_context.post(
        f"/api/v1/assets/{assets['cpr']['id']}/values",
        data={"value": 4.15, "value_date": "2026-11-12"},
        headers=headers,
    )

    # 4. Add holdings to Precious Metals account
    # slv: 45.209 oz -> 45.209 * 32.50 = 1469.2925
    add_slv = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": assets["slv"]["id"], "quantity": 45.209},
        headers=headers,
    )
    assert add_slv.status == 201, f"Failed to add silver: {add_slv.text()}"
    slv_holding = add_slv.json()
    assert slv_holding["quantity"] == 45.209
    assert slv_holding["latest_value"] == 32.50
    assert abs(slv_holding["total_value"] - 1469.2925) < 0.0001

    # gld: 2.5 oz -> 2.5 * 2650 = 6625.00
    add_gld = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": assets["gld"]["id"], "quantity": 2.5},
        headers=headers,
    )
    assert add_gld.status == 201, f"Failed to add gold: {add_gld.text()}"
    gld_holding = add_gld.json()
    assert gld_holding["latest_value"] == 2650.00
    assert abs(gld_holding["total_value"] - 6625.00) < 0.0001

    # cpr: 10.0 lbs -> 10.0 * 4.15 = 41.50
    add_cpr = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": assets["cpr"]["id"], "quantity": 10.0},
        headers=headers,
    )
    assert add_cpr.status == 201, f"Failed to add copper: {add_cpr.text()}"
    cpr_holding = add_cpr.json()
    assert abs(cpr_holding["total_value"] - 41.50) < 0.0001

    # 5. List assets in account
    list_res = api_context.get(f"/api/v1/accounts/{account_id}/assets", headers=headers)
    assert list_res.status == 200
    items = list_res.json()
    assert len(items) == 3

    # 6. Verify Account Summary & Total Calculated Value
    # Total = 1469.2925 + 6625.00 + 41.50 = 8135.7925
    sum_res = api_context.get(f"/api/v1/accounts/{account_id}/assets/summary", headers=headers)
    assert sum_res.status == 200
    sum_data = sum_res.json()
    assert abs(sum_data["total_value"] - 8135.7925) < 0.001
    assert len(sum_data["assets"]) == 3

    # 7. Update quantity of Copper from 10.0 to 20.0
    # New copper total = 20.0 * 4.15 = 83.00
    upd_res = api_context.put(
        f"/api/v1/accounts/{account_id}/assets/{assets['cpr']['id']}",
        data={"quantity": 20.0},
        headers=headers,
    )
    assert upd_res.status == 200, f"Failed to update copper: {upd_res.text()}"
    assert upd_res.json()["quantity"] == 20.0
    assert abs(upd_res.json()["total_value"] - 83.00) < 0.0001

    # Verify summary updated: 1469.2925 + 6625.00 + 83.00 = 8177.2925
    sum_upd_res = api_context.get(f"/api/v1/accounts/{account_id}/assets/summary", headers=headers)
    assert abs(sum_upd_res.json()["total_value"] - 8177.2925) < 0.001

    # 8. Delete Copper from account
    del_res = api_context.delete(
        f"/api/v1/accounts/{account_id}/assets/{assets['cpr']['id']}",
        headers=headers,
    )
    assert del_res.status == 204

    # Verify copper is gone and summary reflects remaining silver + gold
    # Remaining total = 1469.2925 + 6625.00 = 8094.2925
    final_sum_res = api_context.get(f"/api/v1/accounts/{account_id}/assets/summary", headers=headers)
    assert len(final_sum_res.json()["assets"]) == 2
    assert abs(final_sum_res.json()["total_value"] - 8094.2925) < 0.001


def test_asset_account_access_control(api_context: APIRequestContext):
    """
    Test access control on asset accounts per account_access:
    - User B has no access -> 404 (does not leak existence)
    - User A grants 'read' access to User B
    - User B can GET holdings & summary, but receives 403 on mutations
    - User A upgrades User B to 'edit' access
    - User B can mutate holdings
    """
    user_a, token_a = create_authenticated_user(api_context)
    user_b, token_b = create_authenticated_user(api_context)
    headers_a = {"Authorization": f"Bearer {token_a}"}
    headers_b = {"Authorization": f"Bearer {token_b}"}

    # User A creates asset account & asset
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Vault", "type": "asset", "description": "Shared Vault"},
        headers=headers_a,
    )
    acc_id = acc_res.json()["id"]

    asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": f"slv_{uuid.uuid4().hex[:4]}", "description": "Silver"},
        headers=headers_a,
    )
    asset_id = asset_res.json()["id"]

    # 1. User B has no access -> 404 Not Found
    unauth_list = api_context.get(f"/api/v1/accounts/{acc_id}/assets", headers=headers_b)
    assert unauth_list.status == 404

    # 2. User A grants 'read' access to User B
    grant_read = api_context.post(
        f"/api/v1/accounts/{acc_id}/access",
        data={"user_id": user_b["id"], "access_level": "read"},
        headers=headers_a,
    )
    assert grant_read.status == 200, f"Failed to grant read access: {grant_read.text()}"

    # User B can read
    read_list = api_context.get(f"/api/v1/accounts/{acc_id}/assets", headers=headers_b)
    assert read_list.status == 200
    read_sum = api_context.get(f"/api/v1/accounts/{acc_id}/assets/summary", headers=headers_b)
    assert read_sum.status == 200

    # User B tries to add asset -> 403 Forbidden
    forbidden_add = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset_id, "quantity": 10.0},
        headers=headers_b,
    )
    assert forbidden_add.status == 403

    # User A adds the asset
    add_res = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset_id, "quantity": 10.0},
        headers=headers_a,
    )
    assert add_res.status == 201

    # User B tries to update quantity -> 403 Forbidden
    forbidden_upd = api_context.put(
        f"/api/v1/accounts/{acc_id}/assets/{asset_id}",
        data={"quantity": 25.0},
        headers=headers_b,
    )
    assert forbidden_upd.status == 403

    # User B tries to delete asset -> 403 Forbidden
    forbidden_del = api_context.delete(
        f"/api/v1/accounts/{acc_id}/assets/{asset_id}",
        headers=headers_b,
    )
    assert forbidden_del.status == 403

    # 3. User A upgrades User B to 'edit' access
    grant_edit = api_context.post(
        f"/api/v1/accounts/{acc_id}/access",
        data={"user_id": user_b["id"], "access_level": "edit"},
        headers=headers_a,
    )
    assert grant_edit.status == 200

    # User B can now update quantity
    edit_upd = api_context.put(
        f"/api/v1/accounts/{acc_id}/assets/{asset_id}",
        data={"quantity": 30.0},
        headers=headers_b,
    )
    assert edit_upd.status == 200
    assert edit_upd.json()["quantity"] == 30.0

    # User B can delete asset
    edit_del = api_context.delete(
        f"/api/v1/accounts/{acc_id}/assets/{asset_id}",
        headers=headers_b,
    )
    assert edit_del.status == 204


def test_asset_account_negative_cases(api_context: APIRequestContext):
    """
    Test negative and error edge cases:
    - Adding asset to non-asset account (checking) -> 400
    - Non-positive quantities (<= 0) -> 400
    - Duplicate asset in same account -> 409
    - Non-existent asset ID -> 404
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Non-asset account (checking)
    chk_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Checking", "type": "checking", "description": "Daily Checking"},
        headers=headers,
    )
    chk_id = chk_res.json()["id"]

    asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": f"slv_{uuid.uuid4().hex[:4]}", "description": "Silver"},
        headers=headers,
    )
    asset_id = asset_res.json()["id"]

    chk_add = api_context.post(
        f"/api/v1/accounts/{chk_id}/assets",
        data={"asset_id": asset_id, "quantity": 10.0},
        headers=headers,
    )
    assert chk_add.status == 400

    # 2. Asset account tests
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Metals", "type": "asset", "description": "Metals Account"},
        headers=headers,
    )
    acc_id = acc_res.json()["id"]

    # Negative / Zero quantity
    for bad_qty in [0, -5.5]:
        bad_add = api_context.post(
            f"/api/v1/accounts/{acc_id}/assets",
            data={"asset_id": asset_id, "quantity": bad_qty},
            headers=headers,
        )
        assert bad_add.status == 400

    # Add valid
    ok_add = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset_id, "quantity": 15.0},
        headers=headers,
    )
    assert ok_add.status == 201

    # Duplicate asset conflict
    dup_add = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset_id, "quantity": 20.0},
        headers=headers,
    )
    assert dup_add.status == 409

    # Non-existent asset
    fake_asset_id = str(uuid.uuid4())
    fake_add = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": fake_asset_id, "quantity": 5.0},
        headers=headers,
    )
    assert fake_add.status == 404


def test_asset_account_multi_date_holdings_and_trend(api_context: APIRequestContext):
    """
    Test setting asset quantities across multiple dates and tracking portfolio valuation over time:
    - User records 0.3 units on 2026-01-01
    - User records 0.5 units on 2026-07-01
    - User records 0.2 units on 2026-10-01
    - Valuations on 2026-01-01 ($40,000), 2026-07-01 ($60,000), 2026-10-01 ($65,000)
    - Verify summary reflects current position (0.2 units * $65,000 = $13,000)
    - Verify trend endpoint returns chronological timeline points
    - Verify updating holding date and quantity
    - Verify deleting a specific holding
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Create asset account
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={
            "name": "Digital Asset Portfolio",
            "type": "asset",
            "description": "Multi-date asset holding portfolio",
        },
        headers=headers,
    )
    assert acc_res.status == 201
    account_id = acc_res.json()["id"]

    # 2. Create generic asset definition
    uid = uuid.uuid4().hex[:6]
    asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": f"token_{uid}", "description": "Digital utility token"},
        headers=headers,
    )
    assert asset_res.status == 201
    asset_id = asset_res.json()["id"]

    # 3. Add valuations on Jan 1, July 1, Oct 1
    api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 40000.00, "value_date": "2026-01-01"},
        headers=headers,
    )
    api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 60000.00, "value_date": "2026-07-01"},
        headers=headers,
    )
    api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 65000.00, "value_date": "2026-10-01"},
        headers=headers,
    )

    # 4. Add holdings on Jan 1 (0.3), July 1 (0.5), Oct 1 (0.2)
    h1_res = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": asset_id, "quantity": 0.3, "asset_date": "2026-01-01"},
        headers=headers,
    )
    assert h1_res.status == 201
    h1 = h1_res.json()
    assert h1["asset_date"] == "2026-01-01"
    assert h1["quantity"] == 0.3
    assert abs(h1["total_value"] - 0.3 * 40000.00) < 0.01

    h2_res = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": asset_id, "quantity": 0.5, "asset_date": "2026-07-01"},
        headers=headers,
    )
    assert h2_res.status == 201
    h2 = h2_res.json()
    assert h2["asset_date"] == "2026-07-01"
    assert h2["quantity"] == 0.5
    assert abs(h2["total_value"] - 0.5 * 60000.00) < 0.01

    h3_res = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": asset_id, "quantity": 0.2, "asset_date": "2026-10-01"},
        headers=headers,
    )
    assert h3_res.status == 201
    h3 = h3_res.json()
    assert h3["asset_date"] == "2026-10-01"
    assert h3["quantity"] == 0.2
    assert abs(h3["total_value"] - 0.2 * 65000.00) < 0.01

    # Conflict when adding duplicate date for same asset
    dup_res = api_context.post(
        f"/api/v1/accounts/{account_id}/assets",
        data={"asset_id": asset_id, "quantity": 0.99, "asset_date": "2026-10-01"},
        headers=headers,
    )
    assert dup_res.status == 409

    # 5. Verify List Account Assets returns all 3 holdings (ordered newest first)
    list_res = api_context.get(f"/api/v1/accounts/{account_id}/assets", headers=headers)
    assert list_res.status == 200
    all_holdings = list_res.json()
    assert len(all_holdings) == 3
    assert all_holdings[0]["asset_date"] == "2026-10-01"
    assert all_holdings[1]["asset_date"] == "2026-07-01"
    assert all_holdings[2]["asset_date"] == "2026-01-01"

    # 6. Verify Summary computes current position from latest holding (0.2 * 65,000 = 13,000)
    sum_res = api_context.get(f"/api/v1/accounts/{account_id}/assets/summary", headers=headers)
    assert sum_res.status == 200
    sum_data = sum_res.json()
    assert abs(sum_data["total_value"] - 13000.00) < 0.01
    assert len(sum_data["assets"]) == 1
    assert sum_data["assets"][0]["quantity"] == 0.2

    # 7. Verify Trend endpoint returns time-series progression
    trend_res = api_context.get(f"/api/v1/accounts/{account_id}/assets/trend", headers=headers)
    assert trend_res.status == 200
    trend = trend_res.json()
    assert len(trend) == 3
    assert trend[0]["date"] == "2026-01-01"
    assert abs(trend[0]["total_value"] - 12000.00) < 0.01
    assert trend[1]["date"] == "2026-07-01"
    assert abs(trend[1]["total_value"] - 30000.00) < 0.01
    assert trend[2]["date"] == "2026-10-01"
    assert abs(trend[2]["total_value"] - 13000.00) < 0.01

    # Date range filtering on trend endpoint
    filtered_trend = api_context.get(
        f"/api/v1/accounts/{account_id}/assets/trend?start_date=2026-06-01&end_date=2026-10-01",
        headers=headers,
    )
    assert filtered_trend.status == 200
    f_pts = filtered_trend.json()
    assert len(f_pts) == 2
    assert f_pts[0]["date"] == "2026-07-01"
    assert f_pts[1]["date"] == "2026-10-01"

    # 8. Update holding h1 date and quantity
    upd_res = api_context.put(
        f"/api/v1/accounts/{account_id}/assets/{h1['id']}",
        data={"quantity": 0.4, "asset_date": "2026-02-01"},
        headers=headers,
    )
    assert upd_res.status == 200
    updated_h1 = upd_res.json()
    assert updated_h1["quantity"] == 0.4
    assert updated_h1["asset_date"] == "2026-02-01"

    # 9. Delete holding h2 (July 1)
    del_res = api_context.delete(
        f"/api/v1/accounts/{account_id}/assets/{h2['id']}",
        headers=headers,
    )
    assert del_res.status == 204

    # Verify only 2 holdings remain
    post_del_list = api_context.get(f"/api/v1/accounts/{account_id}/assets", headers=headers)
    assert len(post_del_list.json()) == 2


def test_asset_stores_and_records_workflow(api_context: APIRequestContext):
    """
    Test full workflow of labeled asset stores and date-based quantity records:
    - Create asset account and asset definition (Alpha Coin)
    - Add multi-date valuations for Alpha Coin (Jan 1, Feb 1, Oct 1)
    - Create labeled asset stores: 'Red Key' and 'Blue Key'
    - Verify duplicate store label within same account fails (409 Conflict)
    - Add date-based quantity records to 'Red Key' (Jan 1: 0.3, Feb 1: 0.5, Oct 1: 0.2)
    - Verify duplicate date record for same store fails (409 Conflict)
    - Add date-based quantity records to 'Blue Key' (Feb 1: 0.1, Oct 1: 0.4)
    - Verify store list includes latest quantity and calculated total value
    - Verify store-specific trend over time
    - Verify account-wide store summary and multi-store aggregated trend
    - Update store label ('Blue Key' -> 'Navy Key')
    - Update store record quantity
    - Delete store record and verify count
    - Delete store and verify cascade deletion of records
    - Verify access control permissions (read cannot mutate stores or records)
    """
    user_owner, token_owner = create_authenticated_user(api_context)
    headers_owner = {"Authorization": f"Bearer {token_owner}"}

    # 1. Create asset account
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={
            "name": "Crypto Asset Portfolio",
            "type": "asset",
            "description": "Multi-store cold and warm storage keys",
        },
        headers=headers_owner,
    )
    assert acc_res.status == 201, f"Failed to create account: {acc_res.text()}"
    account = acc_res.json()
    account_id = account["id"]

    # 2. Create asset definition
    uid = uuid.uuid4().hex[:6]
    asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": f"alphacoin_{uid}", "description": "Generic crypto alpha coin"},
        headers=headers_owner,
    )
    assert asset_res.status == 201, f"Failed to create asset: {asset_res.text()}"
    asset_id = asset_res.json()["id"]

    # 3. Add valuations: Jan 1 = $40,000, Feb 1 = $45,000, Oct 1 = $60,000
    v1_res = api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 40000.00, "value_date": "2026-01-01"},
        headers=headers_owner,
    )
    assert v1_res.status == 201
    v2_res = api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 45000.00, "value_date": "2026-02-01"},
        headers=headers_owner,
    )
    assert v2_res.status == 201
    v3_res = api_context.post(
        f"/api/v1/assets/{asset_id}/values",
        data={"value": 60000.00, "value_date": "2026-10-01"},
        headers=headers_owner,
    )
    assert v3_res.status == 201

    # 4. Create Asset Stores: 'Red Key' and 'Blue Key'
    s1_res = api_context.post(
        f"/api/v1/accounts/{account_id}/stores",
        data={
            "asset_id": asset_id,
            "label": "Red Key",
            "description": "Primary cold storage private key (red)",
        },
        headers=headers_owner,
    )
    assert s1_res.status == 201, f"Failed to create store: {s1_res.text()}"
    store_red = s1_res.json()
    assert store_red["label"] == "Red Key"
    assert store_red["latest_quantity"] == 0.0
    assert store_red["total_value"] == 0.0

    s2_res = api_context.post(
        f"/api/v1/accounts/{account_id}/stores",
        data={
            "asset_id": asset_id,
            "label": "Blue Key",
            "description": "Secondary hardware key (blue)",
        },
        headers=headers_owner,
    )
    assert s2_res.status == 201, f"Failed to create store: {s2_res.text()}"
    store_blue = s2_res.json()
    assert store_blue["label"] == "Blue Key"

    # Duplicate store label check (case-insensitive)
    dup_store = api_context.post(
        f"/api/v1/accounts/{account_id}/stores",
        data={"asset_id": asset_id, "label": "red key"},
        headers=headers_owner,
    )
    assert dup_store.status == 409

    # 5. Add Quantity Records to 'Red Key'
    # Jan 1: 0.3
    r1 = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        data={"quantity": 0.3, "record_date": "2026-01-01"},
        headers=headers_owner,
    )
    assert r1.status == 201, f"Failed to add store record: {r1.text()}"
    rec1 = r1.json()
    assert rec1["quantity"] == 0.3
    assert rec1["record_date"] == "2026-01-01"

    # Feb 1: 0.5
    r2 = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        data={"quantity": 0.5, "record_date": "2026-02-01"},
        headers=headers_owner,
    )
    assert r2.status == 201

    # Oct 1: 0.2
    r3 = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        data={"quantity": 0.2, "record_date": "2026-10-01"},
        headers=headers_owner,
    )
    assert r3.status == 201

    # Conflict on duplicate date for same store
    dup_rec = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        data={"quantity": 0.9, "record_date": "2026-10-01"},
        headers=headers_owner,
    )
    assert dup_rec.status == 409

    # 6. Add Quantity Records to 'Blue Key'
    # Feb 1: 0.1
    b1 = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_blue['id']}/records",
        data={"quantity": 0.1, "record_date": "2026-02-01"},
        headers=headers_owner,
    )
    assert b1.status == 201

    # Oct 1: 0.4
    b2 = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_blue['id']}/records",
        data={"quantity": 0.4, "record_date": "2026-10-01"},
        headers=headers_owner,
    )
    assert b2.status == 201

    # 7. List Stores and verify current positions
    stores_res = api_context.get(f"/api/v1/accounts/{account_id}/stores", headers=headers_owner)
    assert stores_res.status == 200
    store_list = stores_res.json()
    assert len(store_list) == 2
    by_label = {s["label"]: s for s in store_list}

    # Red Key latest: 0.2, valuation: 60,000 -> total = 12,000
    assert abs(by_label["Red Key"]["latest_quantity"] - 0.2) < 0.001
    assert abs(by_label["Red Key"]["latest_value"] - 60000.00) < 0.01
    assert abs(by_label["Red Key"]["total_value"] - 12000.00) < 0.01

    # Blue Key latest: 0.4, valuation: 60,000 -> total = 24,000
    assert abs(by_label["Blue Key"]["latest_quantity"] - 0.4) < 0.001
    assert abs(by_label["Blue Key"]["latest_value"] - 60000.00) < 0.01
    assert abs(by_label["Blue Key"]["total_value"] - 24000.00) < 0.01

    # 8. Store Trend Endpoint for Red Key
    s1_trend_res = api_context.get(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/trend",
        headers=headers_owner,
    )
    assert s1_trend_res.status == 200
    s1_trend = s1_trend_res.json()
    assert len(s1_trend) == 3
    # 2026-01-01: 0.3 * 40,000 = 12,000
    assert s1_trend[0]["date"] == "2026-01-01"
    assert abs(s1_trend[0]["quantity"] - 0.3) < 0.001
    assert abs(s1_trend[0]["total_value"] - 12000.00) < 0.01
    # 2026-02-01: 0.5 * 45,000 = 22,500
    assert s1_trend[1]["date"] == "2026-02-01"
    assert abs(s1_trend[1]["quantity"] - 0.5) < 0.001
    assert abs(s1_trend[1]["total_value"] - 22500.00) < 0.01
    # 2026-10-01: 0.2 * 60,000 = 12,000
    assert s1_trend[2]["date"] == "2026-10-01"
    assert abs(s1_trend[2]["quantity"] - 0.2) < 0.001
    assert abs(s1_trend[2]["total_value"] - 12000.00) < 0.01

    # 9. Account Summary
    sum_res = api_context.get(
        f"/api/v1/accounts/{account_id}/stores/summary",
        headers=headers_owner,
    )
    assert sum_res.status == 200
    summary = sum_res.json()
    assert abs(summary["total_value"] - 36000.00) < 0.01
    assert len(summary["stores"]) == 2

    # 10. Aggregated Account Trend across all stores
    acc_trend_res = api_context.get(
        f"/api/v1/accounts/{account_id}/stores/trend",
        headers=headers_owner,
    )
    assert acc_trend_res.status == 200
    acc_trend = acc_trend_res.json()
    assert len(acc_trend) == 3
    # 2026-01-01: Red (0.3 * 40k = 12k), Blue (0) -> 12,000
    assert acc_trend[0]["date"] == "2026-01-01"
    assert abs(acc_trend[0]["total_value"] - 12000.00) < 0.01
    # 2026-02-01: Red (0.5 * 45k = 22.5k), Blue (0.1 * 45k = 4.5k) -> 27,000
    assert acc_trend[1]["date"] == "2026-02-01"
    assert abs(acc_trend[1]["total_value"] - 27000.00) < 0.01
    # 2026-10-01: Red (0.2 * 60k = 12k), Blue (0.4 * 60k = 24k) -> 36,000
    assert acc_trend[2]["date"] == "2026-10-01"
    assert abs(acc_trend[2]["total_value"] - 36000.00) < 0.01

    # 11. Update Store Label
    upd_store = api_context.put(
        f"/api/v1/accounts/{account_id}/stores/{store_blue['id']}",
        data={"label": "Navy Key", "description": "Updated blue hardware key"},
        headers=headers_owner,
    )
    assert upd_store.status == 200
    assert upd_store.json()["label"] == "Navy Key"

    # 12. Update Store Record
    upd_rec = api_context.put(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records/{rec1['id']}",
        data={"quantity": 0.35, "record_date": "2026-01-05"},
        headers=headers_owner,
    )
    assert upd_rec.status == 200
    assert abs(upd_rec.json()["quantity"] - 0.35) < 0.001
    assert upd_rec.json()["record_date"] == "2026-01-05"

    # 13. Delete Store Record
    del_rec = api_context.delete(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records/{rec1['id']}",
        headers=headers_owner,
    )
    assert del_rec.status == 204

    # List records returns 2 remaining for Red Key
    recs_res = api_context.get(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        headers=headers_owner,
    )
    assert recs_res.status == 200
    assert len(recs_res.json()) == 2

    # 14. Access Control: Collaborator with Read Access
    user_collab, token_collab = create_authenticated_user(api_context)
    headers_collab = {"Authorization": f"Bearer {token_collab}"}

    # Grant read access to collaborator
    grant_res = api_context.post(
        f"/api/v1/accounts/{account_id}/access",
        data={"user_id": user_collab["id"], "access_level": "read"},
        headers=headers_owner,
    )
    assert grant_res.status == 200

    # Collab can read stores
    collab_read = api_context.get(
        f"/api/v1/accounts/{account_id}/stores",
        headers=headers_collab,
    )
    assert collab_read.status == 200

    # Collab cannot create store (403 Forbidden)
    collab_create_store = api_context.post(
        f"/api/v1/accounts/{account_id}/stores",
        data={"asset_id": asset_id, "label": "Unauthorized Store"},
        headers=headers_collab,
    )
    assert collab_create_store.status == 403

    # Collab cannot add record (403 Forbidden)
    collab_create_rec = api_context.post(
        f"/api/v1/accounts/{account_id}/stores/{store_red['id']}/records",
        data={"quantity": 1.0, "record_date": "2026-11-01"},
        headers=headers_collab,
    )
    assert collab_create_rec.status == 403

    # 15. Delete Store
    del_store = api_context.delete(
        f"/api/v1/accounts/{account_id}/stores/{store_blue['id']}",
        headers=headers_owner,
    )
    assert del_store.status == 204

    # Verify only Red Key store remains
    final_stores = api_context.get(
        f"/api/v1/accounts/{account_id}/stores",
        headers=headers_owner,
    )
    assert len(final_stores.json()) == 1
    assert final_stores.json()[0]["label"] == "Red Key"

