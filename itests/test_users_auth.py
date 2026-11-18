import uuid
import pytest
from playwright.sync_api import APIRequestContext
from conftest import decode_jwt_unverified, compute_totp_code

TEST_PASSWORD = "ComplexPassword123!"

def generate_user_data(prefix: str = "itest"):
    """Helper to generate unique user registration data."""
    unique_id = uuid.uuid4().hex[:8]
    return {
        "username": f"{prefix}_{unique_id}",
        "email": f"{prefix}_{unique_id}@example.com",
        "password": TEST_PASSWORD,
    }

def test_hello_endpoint(api_context: APIRequestContext):
    """Test health check / smoke test endpoint at /api/v1/hello."""
    response = api_context.get("/api/v1/hello")
    assert response.status == 200, f"Expected 200, got {response.status}: {response.text()}"
    assert response.text() == "world"

def test_health_endpoint(api_context: APIRequestContext):
    """Test database-confirmed health check endpoint at /api/health."""
    response = api_context.get("/api/health")
    assert response.status == 200, f"Expected 200, got {response.status}: {response.text()}"
    data = response.json()
    assert data.get("status") == "up"

def test_create_user(api_context: APIRequestContext):
    """Test 1: Create a user with username, email, and receive TOTP secret."""
    user_data = generate_user_data()
    response = api_context.post("/api/v1/users", data=user_data)

    assert response.status == 201, f"Expected 201, got {response.status}: {response.text()}"
    data = response.json()
    assert "id" in data and len(data["id"]) > 0
    assert data["username"] == user_data["username"]
    assert data["email"] == user_data["email"]
    assert "totp_secret" in data and len(data["totp_secret"]) > 0
    assert "totp_url" in data and data["totp_url"].startswith("otpauth://totp/")
    assert "created_at" in data

def test_login_and_validate_jwt(api_context: APIRequestContext):
    """Test 2: Login as user with password and valid TOTP code."""
    user_data = generate_user_data()
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201
    create_data = create_res.json()
    user_id = create_data["id"]
    totp_secret = create_data["totp_secret"]

    totp_code = compute_totp_code(totp_secret)
    login_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"], "totp_code": totp_code},
    )
    assert login_res.status == 200, f"Login failed: {login_res.text()}"
    token_data = login_res.json()

    assert "access_token" in token_data and len(token_data["access_token"]) > 0
    assert token_data.get("token_type") == "Bearer"
    assert token_data.get("expires_in") == 900

    payload = decode_jwt_unverified(token_data["access_token"])
    assert payload.get("sub") == user_id
    assert payload.get("username") == user_data["username"]
    assert payload.get("subscriber") is False
    assert payload.get("exp", 0) - payload.get("iat", 0) == 900


def test_refresh_valid_jwt(api_context: APIRequestContext):
    """Test: Exchange a valid 15-minute JWT for a newly issued JWT without re-entering credentials."""
    user_data = generate_user_data(prefix="refresh")
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201
    create_data = create_res.json()

    totp_code = compute_totp_code(create_data["totp_secret"])
    login_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"], "totp_code": totp_code},
    )
    assert login_res.status == 200
    old_token = login_res.json()["access_token"]
    old_payload = decode_jwt_unverified(old_token)

    refresh_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token/refresh",
        headers={"Authorization": f"Bearer {old_token}"},
    )
    assert refresh_res.status == 200, f"Refresh failed: {refresh_res.text()}"
    ref_data = refresh_res.json()
    assert ref_data.get("token_type") == "Bearer"
    assert ref_data.get("expires_in") == 900
    new_token = ref_data["access_token"]
    assert new_token != old_token

    new_payload = decode_jwt_unverified(new_token)
    assert new_payload.get("sub") == create_data["id"]
    assert new_payload.get("username") == user_data["username"]
    assert new_payload.get("exp", 0) >= old_payload.get("exp", 0)
    assert new_payload.get("exp", 0) - new_payload.get("iat", 0) == 900

    # Refreshed JWT accesses authenticated endpoints
    acc_res = api_context.get("/api/v1/accounts", headers={"Authorization": f"Bearer {new_token}"})
    assert acc_res.status == 200

    # Invalid / missing JWT is rejected (401)
    assert api_context.post(f"/api/v1/users/{user_data['username']}/token/refresh").status == 401
    assert api_context.post(
        f"/api/v1/users/{user_data['username']}/token/refresh",
        headers={"Authorization": "Bearer invalid.jwt.token"},
    ).status == 401


def test_user_a_cannot_exchange_token_for_user_b(api_context: APIRequestContext):
    """Test: Logging in as User A and attempting to exchange/refresh a token for User B must fail."""
    user_a = generate_user_data(prefix="usera")
    create_a = api_context.post("/api/v1/users", data=user_a)
    assert create_a.status == 201
    secret_a = create_a.json()["totp_secret"]

    user_b = generate_user_data(prefix="userb")
    create_b = api_context.post("/api/v1/users", data=user_b)
    assert create_b.status == 201

    totp_a = compute_totp_code(secret_a)
    login_a = api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": totp_a},
    )
    assert login_a.status == 200
    token_a = login_a.json()["access_token"]

    # 1. User A attempting to exchange User A's valid JWT for User B's JWT fails with 403
    refresh_for_b = api_context.post(
        f"/api/v1/users/{user_b['username']}/token/refresh",
        headers={"Authorization": f"Bearer {token_a}"},
    )
    assert refresh_for_b.status == 403, f"Expected 403, got {refresh_for_b.status}: {refresh_for_b.text()}"
    assert refresh_for_b.json().get("error") == "FORBIDDEN"
    assert "access_token" not in refresh_for_b.json()

    # 2. User A attempting password/TOTP login against User B with User A's credentials fails with 401
    creds_for_b = api_context.post(
        f"/api/v1/users/{user_b['username']}/token",
        data={"password": user_a["password"], "totp_code": totp_a},
    )
    assert creds_for_b.status == 401, f"Expected 401, got {creds_for_b.status}: {creds_for_b.text()}"
    assert "access_token" not in creds_for_b.json()

    # 3. User A attempting /users/{userB}/token using Bearer token only (no password/TOTP) fails with 400
    bearer_only_for_b = api_context.post(
        f"/api/v1/users/{user_b['username']}/token",
        headers={"Authorization": f"Bearer {token_a}"},
    )
    assert bearer_only_for_b.status == 400, f"Expected 400, got {bearer_only_for_b.status}: {bearer_only_for_b.text()}"
    assert "access_token" not in bearer_only_for_b.json()

def test_new_account_jwt_subscriber_is_false(api_context: APIRequestContext):
    """Test: Confirm the issued JWT for a new account has subscriber value of false."""
    user_data = generate_user_data(prefix="subtest")
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201
    create_data = create_res.json()
    user_id = create_data["id"]
    totp_secret = create_data["totp_secret"]

    totp_code = compute_totp_code(totp_secret)
    login_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"], "totp_code": totp_code},
    )
    assert login_res.status == 200, f"Login failed: {login_res.text()}"
    token_data = login_res.json()
    payload = decode_jwt_unverified(token_data["access_token"])
    assert "subscriber" in payload
    assert payload["subscriber"] is False
    assert payload.get("sub") == user_id
    assert payload.get("username") == user_data["username"]

def test_login_invalid_totp_fails(api_context: APIRequestContext):
    """Test: Attempting login with invalid TOTP code returns 401."""
    user_data = generate_user_data()
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201

    login_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"], "totp_code": "000000"},
    )
    assert login_res.status == 401
    assert login_res.json().get("error") == "INVALID_TOTP"

def test_create_duplicate_username_fails(api_context: APIRequestContext):
    """Test 3: Attempt to create same username with different email and verify 409 conflict."""
    user_data = generate_user_data()
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201

    diff_email = f"diff_{uuid.uuid4().hex[:8]}@example.com"
    duplicate_user_data = {
        "username": user_data["username"],
        "email": diff_email,
        "password": TEST_PASSWORD,
    }

    dup_res = api_context.post("/api/v1/users", data=duplicate_user_data)
    assert dup_res.status == 409, f"Expected 409 conflict, got {dup_res.status}: {dup_res.text()}"
    err_data = dup_res.json()
    assert err_data.get("error") == "USER_CONFLICT"


def test_delete_signed_in_user_cascades_all_records(api_context: APIRequestContext):
    """Test: Signed-in user can delete their account, cascading all owned accounts and records."""
    user_a = generate_user_data(prefix="deluser")
    res_a = api_context.post("/api/v1/users", data=user_a)
    assert res_a.status == 201
    secret_a = res_a.json()["totp_secret"]
    totp_a = compute_totp_code(secret_a)
    tok_a = api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": totp_a},
    ).json()["access_token"]
    hdr_a = {"Authorization": f"Bearer {tok_a}"}

    user_b = generate_user_data(prefix="keepuser")
    res_b = api_context.post("/api/v1/users", data=user_b)
    assert res_b.status == 201
    secret_b = res_b.json()["totp_secret"]
    tok_b = api_context.post(
        f"/api/v1/users/{user_b['username']}/token",
        data={"password": user_b["password"], "totp_code": compute_totp_code(secret_b)},
    ).json()["access_token"]
    hdr_b = {"Authorization": f"Bearer {tok_b}"}

    # Populate User A owned data across accounts, records, categories, reconcile reports, snapshots, perspectives, recurring
    acc_chk = api_context.post(
        "/api/v1/accounts",
        data={"name": "Primary Everyday Checking", "type": "checking", "description": "Checking account"},
        headers=hdr_a,
    )
    assert acc_chk.status == 201
    chk_id = acc_chk.json()["id"]

    cat_res = api_context.post("/api/v1/categories", data={"name": f"Cat_{uuid.uuid4().hex[:4]}"}, headers=hdr_a)
    assert cat_res.status == 201
    cat_id = cat_res.json()["id"]

    rec_res = api_context.post(
        f"/api/v1/accounts/{chk_id}/records",
        data={"value": -35.0, "date_sent": "2026-11-10", "date_cleared": "2026-11-11", "category_id": cat_id, "entity": "Corner Grocery"},
        headers=hdr_a,
    )
    assert rec_res.status == 201
    rec_id = rec_res.json()["id"]

    rep_res = api_context.post(
        f"/api/v1/accounts/{chk_id}/reconcile-reports",
        data={"name": "Nov Check", "report_date": "2026-11-12", "starting_balance": 0.0, "ending_balance": -35.0, "record_ids": [rec_id]},
        headers=hdr_a,
    )
    assert rep_res.status == 201

    acc_snap = api_context.post(
        "/api/v1/accounts",
        data={"name": "Brokerage Snapshot", "type": "snapshot", "description": "Portfolio"},
        headers=hdr_a,
    )
    assert acc_snap.status == 201
    snap_acc_id = acc_snap.json()["id"]
    assert api_context.post(
        f"/api/v1/accounts/{snap_acc_id}/snapshots",
        data={"value": 5000.0, "snapshot_date": "2026-11-12"},
        headers=hdr_a,
    ).status == 201

    persp_res = api_context.post(
        "/api/v1/perspectives",
        data={"name": "Net Worth", "description": "Combined", "account_ids": [chk_id, snap_acc_id]},
        headers=hdr_a,
    )
    assert persp_res.status == 201

    rt_res = api_context.post(
        "/api/v1/recurring-transactions",
        data={"amount": -65.0, "frequency": "monthly", "interval": 1, "entity": "Internet Service", "account_id": chk_id},
        headers=hdr_a,
    )
    assert rt_res.status == 201

    # User B has an account with a record that stays intact
    acc_b = api_context.post(
        "/api/v1/accounts",
        data={"name": "User B Checking", "type": "checking", "description": "Keep intact"},
        headers=hdr_b,
    ).json()["id"]
    assert api_context.post(
        f"/api/v1/accounts/{acc_b}/records",
        data={"value": 100.0, "date_sent": "2026-11-10", "entity": "Employer Payroll"},
        headers=hdr_b,
    ).status == 201

    # Unauthenticated and cross-user deletes must be rejected
    assert api_context.delete(f"/api/v1/users/{user_a['username']}").status == 401
    assert api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_b).status == 403

    # User A deletes own account (204 No Content)
    del_res = api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_a)
    assert del_res.status == 204, f"Expected 204 deleting own user, got {del_res.status}: {del_res.text()}"

    # Old JWT cannot refresh or recreate/delete again, and login fails with 404
    assert api_context.post(f"/api/v1/users/{user_a['username']}/token/refresh", headers=hdr_a).status == 404
    assert api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_a).status == 404
    assert api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": compute_totp_code(secret_a)},
    ).status == 404

    # User B's account & records remain intact
    b_recs = api_context.get(f"/api/v1/accounts/{acc_b}/records", headers=hdr_b)
    assert b_recs.status == 200 and len(b_recs.json()) == 1

    # Re-registering with same username & email succeeds cleanly with empty accounts/recurring
    recreate_res = api_context.post("/api/v1/users", data=user_a)
    assert recreate_res.status == 201, f"Re-registration after deletion failed: {recreate_res.text()}"
    new_tok = api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": compute_totp_code(recreate_res.json()["totp_secret"])},
    ).json()["access_token"]
    new_hdr = {"Authorization": f"Bearer {new_tok}"}
    assert api_context.get("/api/v1/accounts", headers=new_hdr).json() == []
    assert api_context.get("/api/v1/recurring-transactions", headers=new_hdr).json() == []
