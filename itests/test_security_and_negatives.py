import uuid
import pytest
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

def test_user_isolation_default_denial(api_context: APIRequestContext):
    """
    Confirm User A cannot access User B's accounts, records, or account assets by default.
    Both accounts and records must return 404 (preventing enumeration/information disclosure).
    """
    user_a, token_a = create_authenticated_user(api_context)
    user_b, token_b = create_authenticated_user(api_context)
    headers_a = {"Authorization": f"Bearer {token_a}"}
    headers_b = {"Authorization": f"Bearer {token_b}"}

    # 1. User A creates a checking account and checking record
    chk_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "User A Checking", "type": "checking", "description": "Confidential account"},
        headers=headers_a,
    )
    assert chk_res.status == 201
    chk_a_id = chk_res.json()["id"]

    rec_res = api_context.post(
        f"/api/v1/accounts/{chk_a_id}/records",
        data={"value": 1500.00, "date_sent": "2026-11-10", "entity": "Employer", "description": "Salary"},
        headers=headers_a,
    )
    assert rec_res.status == 201
    rec_a_id = rec_res.json()["id"]

    # 2. User A creates an asset account, asset, valuation, and holding
    ast_acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "User A Vault", "type": "asset", "description": "Secret precious metals"},
        headers=headers_a,
    )
    assert ast_acc_res.status == 201
    ast_a_id = ast_acc_res.json()["id"]

    asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": f"slv_{uuid.uuid4().hex[:4]}", "description": "Silver bullion"},
        headers=headers_a,
    )
    assert asset_res.status == 201
    asset_a_id = asset_res.json()["id"]

    api_context.post(
        f"/api/v1/assets/{asset_a_id}/values",
        data={"value": 35.00, "value_date": "2026-11-12"},
        headers=headers_a,
    )

    holding_res = api_context.post(
        f"/api/v1/accounts/{ast_a_id}/assets",
        data={"asset_id": asset_a_id, "quantity": 100.0},
        headers=headers_a,
    )
    assert holding_res.status == 201
    holding_a_id = holding_res.json()["id"]

    # 3. User B creates their own account
    b_chk = api_context.post(
        "/api/v1/accounts",
        data={"name": "User B Checking", "type": "checking", "description": "User B personal"},
        headers=headers_b,
    )
    assert b_chk.status == 201
    chk_b_id = b_chk.json()["id"]

    # --- VERIFY USER B CANNOT ACCESS USER A'S CHECKING ACCOUNT ---
    # List accounts: User B should only see their own account
    list_res = api_context.get("/api/v1/accounts", headers=headers_b)
    assert list_res.status == 200
    b_accounts = list_res.json()
    b_account_ids = [a["id"] for a in b_accounts]
    assert chk_b_id in b_account_ids
    assert chk_a_id not in b_account_ids
    assert ast_a_id not in b_account_ids

    # Direct access to User A's checking account: 404
    assert api_context.get(f"/api/v1/accounts/{chk_a_id}", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{chk_a_id}/records", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{chk_a_id}/records/{rec_a_id}", headers=headers_b).status == 404

    # Mutations on User A's checking account by User B: 404
    assert api_context.post(
        f"/api/v1/accounts/{chk_a_id}/records",
        data={"value": -50.0, "date_sent": "2026-11-10", "entity": "Hacker"},
        headers=headers_b,
    ).status == 404

    assert api_context.put(
        f"/api/v1/accounts/{chk_a_id}/records/{rec_a_id}",
        data={"value": -999.0, "date_sent": "2026-11-10", "entity": "Tampered"},
        headers=headers_b,
    ).status == 404

    assert api_context.delete(f"/api/v1/accounts/{chk_a_id}/records/{rec_a_id}", headers=headers_b).status == 404

    # --- VERIFY USER B CANNOT ACCESS USER A'S ASSET ACCOUNT OR ASSETS ---
    assert api_context.get(f"/api/v1/accounts/{ast_a_id}", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{ast_a_id}/assets", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{ast_a_id}/assets/summary", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{ast_a_id}/assets/{asset_a_id}", headers=headers_b).status == 404
    assert api_context.get(f"/api/v1/accounts/{ast_a_id}/assets/{holding_a_id}", headers=headers_b).status == 404

    # Mutations on User A's asset account by User B: 404
    assert api_context.post(
        f"/api/v1/accounts/{ast_a_id}/assets",
        data={"asset_id": asset_a_id, "quantity": 50.0},
        headers=headers_b,
    ).status == 404

    assert api_context.put(
        f"/api/v1/accounts/{ast_a_id}/assets/{asset_a_id}",
        data={"quantity": 1.0},
        headers=headers_b,
    ).status == 404

    assert api_context.delete(f"/api/v1/accounts/{ast_a_id}/assets/{asset_a_id}", headers=headers_b).status == 404

    # User B attempting to grant or revoke access to User A's account: denied
    assert api_context.post(
        f"/api/v1/accounts/{ast_a_id}/access",
        data={"user_id": user_b["id"], "access_level": "admin"},
        headers=headers_b,
    ).status in [403, 404]

    assert api_context.delete(
        f"/api/v1/accounts/{ast_a_id}/access/{user_b['id']}",
        headers=headers_b,
    ).status in [403, 404]


def test_sql_injection_attempts_have_no_impact(api_context: APIRequestContext):
    """
    Confirm SQL injection attempts across path parameters and request payloads
    do not compromise tables, alter query structures, or affect existing data.
    """
    user, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Seed legitimate data
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Integrity Check Account", "type": "checking", "description": "Baseline account"},
        headers=headers,
    )
    assert acc_res.status == 201
    account_id = acc_res.json()["id"]

    rec_res = api_context.post(
        f"/api/v1/accounts/{account_id}/records",
        data={"value": 100.00, "date_sent": "2026-11-10", "entity": "Safe Entity", "description": "Safe Desc"},
        headers=headers,
    )
    assert rec_res.status == 201
    record_id = rec_res.json()["id"]

    asset_label = f"safe_{uuid.uuid4().hex[:4]}"
    ast_res = api_context.post(
        "/api/v1/assets",
        data={"label": asset_label, "description": "Baseline asset"},
        headers=headers,
    )
    assert ast_res.status == 201
    asset_id = ast_res.json()["id"]

    # 2. Path parameter injection attacks
    malicious_ids = [
        "123' OR '1'='1' --",
        "'; DROP TABLE accounts; --",
        "'; DROP TABLE users; --",
        "'; DROP TABLE checking_records; --",
        "'; DROP TABLE assets; --",
        "'; DROP TABLE asset_values; --",
        "'; DROP TABLE account_assets; --",
        "1' UNION SELECT * FROM accounts--",
    ]

    for attack_str in malicious_ids:
        # Checking account path
        res1 = api_context.get(f"/api/v1/accounts/{attack_str}", headers=headers)
        assert res1.status in [400, 404, 500], f"Expected error response on SQL injection in accountId: {res1.status}"

        # Checking record path
        res2 = api_context.get(f"/api/v1/accounts/{account_id}/records/{attack_str}", headers=headers)
        assert res2.status in [400, 404, 500]

        # Asset path
        res3 = api_context.get(f"/api/v1/assets/{attack_str}", headers=headers)
        assert res3.status in [400, 404, 500]

    # 3. Payload injection attacks (verifying string parameters are safely escaped)
    payload_attacks = [
        "Robert'); DROP TABLE accounts;--",
        "'; DROP TABLE users; --",
        "' OR '1'='1",
        "admin'--",
    ]

    for attack_str in payload_attacks:
        # In account creation
        api_context.post(
            "/api/v1/accounts",
            data={"name": f"Account {attack_str[:20]}", "type": "checking", "description": attack_str},
            headers=headers,
        )

        # In checking record creation
        api_context.post(
            f"/api/v1/accounts/{account_id}/records",
            data={"value": -1.0, "date_sent": "2026-11-10", "entity": attack_str[:50], "description": attack_str},
            headers=headers,
        )

    # 4. Verify system and database integrity
    # Table 'accounts' exists and baseline data intact
    get_acc = api_context.get(f"/api/v1/accounts/{account_id}", headers=headers)
    assert get_acc.status == 200, f"Account was corrupted or dropped: {get_acc.text()}"
    assert get_acc.json()["name"] == "Integrity Check Account"

    # Table 'checking_records' exists and baseline data intact
    get_rec = api_context.get(f"/api/v1/accounts/{account_id}/records/{record_id}", headers=headers)
    assert get_rec.status == 200, f"Checking records table was corrupted: {get_rec.text()}"
    assert get_rec.json()["value"] == 100.00

    # Table 'assets' exists and baseline data intact
    get_ast = api_context.get(f"/api/v1/assets/{asset_id}", headers=headers)
    assert get_ast.status == 200, f"Assets table was corrupted: {get_ast.text()}"
    assert get_ast.json()["label"] == asset_label

    # Table 'users' exists and new user can still be registered and authenticated
    new_user, new_token = create_authenticated_user(api_context)
    assert new_token is not None and len(new_token) > 0


def test_duplicate_asset_creation_rejected(api_context: APIRequestContext):
    """
    Confirm creating duplicate assets is rejected:
    - Same label for same user -> 409 Conflict
    - Uppercase / lowercase / mixed-case label -> 409 Conflict
    - Leading / trailing whitespace variations -> 409 Conflict
    - Duplicate asset holding in the same account -> 409 Conflict
    - Duplicate valuation for the same asset on the same date -> 409 Conflict
    - Verify another user CAN create the same label independently (per-user scoping)
    """
    user1, token1 = create_authenticated_user(api_context)
    user2, token2 = create_authenticated_user(api_context)
    headers1 = {"Authorization": f"Bearer {token1}"}
    headers2 = {"Authorization": f"Bearer {token2}"}

    uid = uuid.uuid4().hex[:6]
    base_label = f"asset_{uid}"

    # 1. User 1 creates initial asset
    init_res = api_context.post(
        "/api/v1/assets",
        data={"label": base_label, "description": "Original asset definition"},
        headers=headers1,
    )
    assert init_res.status == 201, f"Failed to create initial asset: {init_res.text()}"
    asset1 = init_res.json()
    assert asset1["label"] == base_label

    # 2. Reject exact duplicate label
    dup_exact = api_context.post(
        "/api/v1/assets",
        data={"label": base_label, "description": "Exact duplicate label"},
        headers=headers1,
    )
    assert dup_exact.status == 409, f"Expected 409 on exact duplicate, got {dup_exact.status}"

    # 3. Reject uppercase variation
    dup_upper = api_context.post(
        "/api/v1/assets",
        data={"label": base_label.upper(), "description": "Uppercase duplicate label"},
        headers=headers1,
    )
    assert dup_upper.status == 409, f"Expected 409 on uppercase duplicate, got {dup_upper.status}"

    # 4. Reject mixed case variation
    dup_mixed = api_context.post(
        "/api/v1/assets",
        data={"label": base_label.capitalize(), "description": "Capitalized duplicate label"},
        headers=headers1,
    )
    assert dup_mixed.status == 409, f"Expected 409 on mixed case duplicate, got {dup_mixed.status}"

    # 5. Reject whitespace padded variation
    dup_spaces = api_context.post(
        "/api/v1/assets",
        data={"label": f"   {base_label}   ", "description": "Whitespace padded duplicate"},
        headers=headers1,
    )
    assert dup_spaces.status == 409, f"Expected 409 on padded duplicate, got {dup_spaces.status}"

    # 6. Reject duplicate asset holding within the same account
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Vault", "type": "asset", "description": "Holding account"},
        headers=headers1,
    )
    acc_id = acc_res.json()["id"]

    add_holding1 = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset1["id"], "quantity": 10.0},
        headers=headers1,
    )
    assert add_holding1.status == 201

    add_holding_dup = api_context.post(
        f"/api/v1/accounts/{acc_id}/assets",
        data={"asset_id": asset1["id"], "quantity": 5.0},
        headers=headers1,
    )
    assert add_holding_dup.status == 409, f"Expected 409 on duplicate account asset holding, got {add_holding_dup.status}"

    # 7. Reject duplicate valuation for the same asset on the same date
    val_res1 = api_context.post(
        f"/api/v1/assets/{asset1['id']}/values",
        data={"value": 100.0, "value_date": "2026-11-12"},
        headers=headers1,
    )
    assert val_res1.status == 201

    val_res_dup = api_context.post(
        f"/api/v1/assets/{asset1['id']}/values",
        data={"value": 105.0, "value_date": "2026-11-12"},
        headers=headers1,
    )
    assert val_res_dup.status == 409, f"Expected 409 on duplicate asset valuation date, got {val_res_dup.status}"

    # 8. User 2 CAN create the same label (per-user scoping)
    u2_res = api_context.post(
        "/api/v1/assets",
        data={"label": base_label, "description": "User 2 asset with same label"},
        headers=headers2,
    )
    assert u2_res.status == 201, f"Expected User 2 to successfully create asset with same label, got {u2_res.status}"

    # User 2 cannot duplicate their own asset
    u2_dup = api_context.post(
        "/api/v1/assets",
        data={"label": base_label, "description": "User 2 duplicate"},
        headers=headers2,
    )
    assert u2_dup.status == 409, f"Expected User 2 duplicate to fail with 409, got {u2_dup.status}"
