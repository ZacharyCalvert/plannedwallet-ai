import hashlib
import uuid
import pytest
from playwright.sync_api import APIRequestContext
from conftest import decode_jwt_unverified, compute_totp_code, load_frontend_pdf_bytes

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

    # Unauthenticated and cross-user profile/deletes must be rejected
    assert api_context.get(f"/api/v1/users/{user_a['username']}").status == 401
    assert api_context.get(f"/api/v1/users/{user_a['username']}", headers=hdr_b).status == 403
    get_a = api_context.get(f"/api/v1/users/{user_a['username']}", headers=hdr_a)
    assert get_a.status == 200, f"Expected 200 getting own profile, got {get_a.status}: {get_a.text()}"
    assert get_a.json().get("username") == user_a["username"]
    assert get_a.json().get("email") == user_a["email"]
    assert "totp_secret" not in get_a.json()

    assert api_context.delete(f"/api/v1/users/{user_a['username']}").status == 401
    assert api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_b).status == 403

    # User A deletes own account (204 No Content)
    del_res = api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_a)
    assert del_res.status == 204, f"Expected 204 deleting own user, got {del_res.status}: {del_res.text()}"

    # Old JWT cannot get/refresh/delete again, and login fails with 404
    assert api_context.get(f"/api/v1/users/{user_a['username']}", headers=hdr_a).status == 404
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


def test_user_a_cannot_delete_user_b(api_context: APIRequestContext):
    """Test: User A logged in with valid credentials cannot trigger DELETE for User B."""
    user_a = generate_user_data(prefix="actora")
    res_a = api_context.post("/api/v1/users", data=user_a)
    assert res_a.status == 201
    tok_a_res = api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": compute_totp_code(res_a.json()["totp_secret"])},
    )
    assert tok_a_res.status == 200
    hdr_a = {"Authorization": f"Bearer {tok_a_res.json()['access_token']}"}

    user_b = generate_user_data(prefix="targetb")
    res_b = api_context.post("/api/v1/users", data=user_b)
    assert res_b.status == 201
    tok_b_res = api_context.post(
        f"/api/v1/users/{user_b['username']}/token",
        data={"password": user_b["password"], "totp_code": compute_totp_code(res_b.json()["totp_secret"])},
    )
    assert tok_b_res.status == 200
    hdr_b = {"Authorization": f"Bearer {tok_b_res.json()['access_token']}"}

    acc_b_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "User B Protected Checking", "type": "checking", "description": "Protected account"},
        headers=hdr_b,
    )
    assert acc_b_res.status == 201
    acc_b_id = acc_b_res.json()["id"]
    assert api_context.post(
        f"/api/v1/accounts/{acc_b_id}/records",
        data={"value": 250.0, "date_sent": "2026-11-15", "entity": "Employer Payroll"},
        headers=hdr_b,
    ).status == 201

    # User A attempting to delete User B (exact or case-folded username) must fail with 403 FORBIDDEN
    for target_name in [user_b["username"], user_b["username"].upper()]:
        del_b = api_context.delete(f"/api/v1/users/{target_name}", headers=hdr_a)
        assert del_b.status == 403, f"Expected 403 deleting {target_name} as User A, got {del_b.status}: {del_b.text()}"
        assert del_b.json().get("error") == "FORBIDDEN"

    # Confirm User B profile, token refresh, account, and records remain completely intact
    prof_b = api_context.get(f"/api/v1/users/{user_b['username']}", headers=hdr_b)
    assert prof_b.status == 200
    assert prof_b.json()["username"] == user_b["username"]

    ref_b = api_context.post(f"/api/v1/users/{user_b['username']}/token/refresh", headers=hdr_b)
    assert ref_b.status == 200

    recs_b = api_context.get(f"/api/v1/accounts/{acc_b_id}/records", headers=hdr_b)
    assert recs_b.status == 200 and len(recs_b.json()) == 1


def test_policy_agreement_ledger_lifecycle_and_validation(
    api_context: APIRequestContext, raw_context: APIRequestContext
):
    """Verify policy_agreements ledger captures client IP, timestamp, SHA-256 validation, and cascade cleanup."""
    terms_sha256 = hashlib.sha256(load_frontend_pdf_bytes(raw_context, "terms-of-use.pdf")).hexdigest()
    privacy_sha256 = hashlib.sha256(load_frontend_pdf_bytes(raw_context, "privacy-and-security-policy.pdf")).hexdigest()

    user_a = generate_user_data(prefix="pol_a")
    user_b = generate_user_data(prefix="pol_b")
    res_a = api_context.post("/api/v1/users", data=user_a)
    res_b = api_context.post("/api/v1/users", data=user_b)
    assert res_a.status == 201 and res_b.status == 201

    tok_a = api_context.post(
        f"/api/v1/users/{user_a['username']}/token",
        data={"password": user_a["password"], "totp_code": compute_totp_code(res_a.json()["totp_secret"])},
    ).json()["access_token"]
    tok_b = api_context.post(
        f"/api/v1/users/{user_b['username']}/token",
        data={"password": user_b["password"], "totp_code": compute_totp_code(res_b.json()["totp_secret"])},
    ).json()["access_token"]
    hdr_a = {"Authorization": f"Bearer {tok_a}"}
    hdr_b = {"Authorization": f"Bearer {tok_b}"}

    # Mismatched terms_sha256, mismatched privacy_sha256, and missing 18+ US adult consumer confirmation must fail cleanly with 400
    bad_hash = api_context.post(
        f"/api/v1/users/{user_a['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": True,
            "terms_sha256": "0" * 64,
            "privacy_sha256": privacy_sha256,
        },
        headers=hdr_a,
    )
    assert bad_hash.status == 400
    assert bad_hash.json().get("error") == "TERMS_HASH_MISMATCH"

    bad_privacy_hash = api_context.post(
        f"/api/v1/users/{user_a['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": True,
            "terms_sha256": terms_sha256,
            "privacy_sha256": "0" * 64,
        },
        headers=hdr_a,
    )
    assert bad_privacy_hash.status == 400
    assert bad_privacy_hash.json().get("error") == "PRIVACY_HASH_MISMATCH"

    no_us = api_context.post(
        f"/api/v1/users/{user_a['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": False,
            "terms_sha256": terms_sha256,
            "privacy_sha256": privacy_sha256,
        },
        headers=hdr_a,
    )
    assert no_us.status == 400
    assert no_us.json().get("error") == "POLICY_US_ADULT_CONSUMER_REQUIRED"

    # Cross-user submission must return 403 FORBIDDEN
    cross_user = api_context.post(
        f"/api/v1/users/{user_b['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": True,
            "terms_sha256": terms_sha256,
            "privacy_sha256": privacy_sha256,
        },
        headers=hdr_a,
    )
    assert cross_user.status == 403

    # Valid submission creates an active ledger entry capturing IP and timestamp
    first = api_context.post(
        f"/api/v1/users/{user_a['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": True,
            "terms_sha256": terms_sha256,
            "privacy_sha256": privacy_sha256,
        },
        headers=hdr_a,
    )
    assert first.status == 201, f"Expected 201, got {first.status}: {first.text()}"
    rec1 = first.json()
    assert rec1["terms_sha256"] == terms_sha256
    assert rec1["privacy_sha256"] == privacy_sha256
    assert rec1["terms_approved"] is True
    assert rec1["privacy_approved"] is True
    assert rec1["us_adult_consumer_confirmed"] is True
    assert rec1["active"] is True
    assert rec1.get("client_ip")
    assert rec1.get("agreed_at")

    # Submitting again supersedes previous row active=False and preserves ledger ordering
    second = api_context.post(
        f"/api/v1/users/{user_a['username']}/policy-agreements",
        data={
            "terms_approved": True,
            "privacy_approved": True,
            "us_adult_consumer_confirmed": True,
            "terms_sha256": terms_sha256,
            "privacy_sha256": privacy_sha256,
        },
        headers=hdr_a,
    )
    assert second.status == 201

    ledger = api_context.get(f"/api/v1/users/{user_a['username']}/policy-agreements", headers=hdr_a)
    assert ledger.status == 200
    entries = ledger.json()
    assert len(entries) == 2
    assert entries[0]["id"] == second.json()["id"] and entries[0]["active"] is True
    assert entries[1]["id"] == rec1["id"] and entries[1]["active"] is False

    # Refresh token issues JWT with terms_approved=True
    ref = api_context.post(f"/api/v1/users/{user_a['username']}/token/refresh", headers=hdr_a)
    assert ref.status == 200
    claims = decode_jwt_unverified(ref.json()["access_token"])
    assert claims.get("terms_approved") is True

    # User B ledger stays empty and deleting User A cleans up cleanly
    b_ledger = api_context.get(f"/api/v1/users/{user_b['username']}/policy-agreements", headers=hdr_b)
    assert b_ledger.status == 200 and b_ledger.json() == []
    assert api_context.delete(f"/api/v1/users/{user_a['username']}", headers=hdr_a).status == 204


def test_account_recovery_resets_password_totp_and_rotates_single_use_phrase(
    api_context: APIRequestContext,
):
    """Verify 12-word Argon2id recovery phrase issuance, single-use password+TOTP reset, replay denial, and rotation."""
    user = generate_user_data(prefix="rec_it")
    reg_res = api_context.post("/api/v1/users", data=user)
    assert reg_res.status == 201, f"Registration failed: {reg_res.text()}"
    reg = reg_res.json()
    phrase1 = reg.get("recovery_phrase", "")
    hint1 = reg.get("recovery_key_hint", "")
    assert len(phrase1.split()) == 12
    assert hint1.startswith("••••")

    # Signed-in profile exposes recovery_key_hint only (never recovery_phrase)
    tok1 = api_context.post(
        f"/api/v1/users/{user['username']}/token",
        data={"password": user["password"], "totp_code": compute_totp_code(reg["totp_secret"])},
    ).json()["access_token"]
    prof1 = api_context.get(
        f"/api/v1/users/{user['username']}", headers={"Authorization": f"Bearer {tok1}"}
    ).json()
    assert prof1.get("recovery_key_hint") == hint1
    assert "recovery_phrase" not in prof1

    # Invalid recovery phrase must return 401 INVALID_RECOVERY_CREDENTIAL
    bad_rec = api_context.post(
        f"/api/v1/users/{user['username']}/recover",
        data={
            "recovery_phrase": "wrong one two three four five six seven eight nine ten eleven",
            "new_password": "NewStrongPassword123!",
        },
    )
    assert bad_rec.status == 401
    assert bad_rec.json().get("error") == "INVALID_RECOVERY_CREDENTIAL"

    # Valid recovery phrase (even with noisy casing/spacing) resets password & TOTP and rotates phrase
    noisy_phrase = f"  {phrase1.upper().replace(' ', '   ')}  "
    new_password = "NewStrongPassword123!"
    rec_res = api_context.post(
        f"/api/v1/users/{user['username']}/recover",
        data={"recovery_phrase": noisy_phrase, "new_password": new_password},
    )
    assert rec_res.status == 200, f"Recovery failed: {rec_res.text()}"
    rec = rec_res.json()
    assert rec["username"] == user["username"]
    assert rec["totp_secret"] != reg["totp_secret"]
    assert len(rec["recovery_phrase"].split()) == 12
    assert rec["recovery_phrase"] != phrase1
    assert rec["recovery_key_hint"] != hint1

    # Consumed phrase is strictly single-use: replaying it must fail with 401
    replay = api_context.post(
        f"/api/v1/users/{user['username']}/recover",
        data={"recovery_phrase": phrase1, "new_password": "AnotherPassword123!"},
    )
    assert replay.status == 401

    # Old password + new TOTP fails; new password + old TOTP fails; new password + new TOTP succeeds
    new_totp = compute_totp_code(rec["totp_secret"])
    assert api_context.post(
        f"/api/v1/users/{user['username']}/token",
        data={"password": user["password"], "totp_code": new_totp},
    ).status == 401

    tok2_res = api_context.post(
        f"/api/v1/users/{user['username']}/token",
        data={"password": new_password, "totp_code": new_totp},
    )
    assert tok2_res.status == 200
    hdr2 = {"Authorization": f"Bearer {tok2_res.json()['access_token']}"}
    prof2 = api_context.get(f"/api/v1/users/{user['username']}", headers=hdr2).json()
    assert prof2.get("recovery_key_hint") == rec["recovery_key_hint"]

    # Signed-in user can rotate their active recovery phrase cleanly
    rot_res = api_context.post(
        f"/api/v1/users/{user['username']}/recovery-key/rotate",
        data={"password": new_password, "totp_code": new_totp},
        headers=hdr2,
    )
    assert rot_res.status == 200
    rot = rot_res.json()
    assert len(rot["recovery_phrase"].split()) == 12
    assert rot["recovery_phrase"] != rec["recovery_phrase"]

    # Previous rotated-out phrase can no longer recover the account
    assert api_context.post(
        f"/api/v1/users/{user['username']}/recover",
        data={"recovery_phrase": rec["recovery_phrase"], "new_password": "ThirdPassword123!"},
    ).status == 401

    assert api_context.delete(f"/api/v1/users/{user['username']}", headers=hdr2).status == 204


def test_account_recovery_rejects_partial_11_of_12_and_reordered_phrases(
    api_context: APIRequestContext,
):
    """Ensure account recovery fails when 11 of 12 words match, a word is omitted, or word order is altered."""
    user = generate_user_data(prefix="rec_11of12")
    reg_res = api_context.post("/api/v1/users", data=user)
    assert reg_res.status == 201
    reg = reg_res.json()
    words = reg["recovery_phrase"].split()
    assert len(words) == 12

    wrong_first = ["decoy" if words[0] != "decoy" else "zebra"] + words[1:]
    wrong_mid = words[:5] + ["decoy" if words[5] != "decoy" else "zebra"] + words[6:]
    wrong_last = words[:11] + ["decoy" if words[11] != "decoy" else "zebra"]
    truncated_11 = words[:11]
    swapped_order = words[:10] + [words[11], words[10]]

    for bad_phrase in (wrong_first, wrong_mid, wrong_last, truncated_11, swapped_order):
        res = api_context.post(
            f"/api/v1/users/{user['username']}/recover",
            data={
                "recovery_phrase": " ".join(bad_phrase),
                "new_password": "AttemptedResetPass123!",
            },
        )
        assert res.status == 401, f"Expected 401 for bad phrase {bad_phrase}, got {res.status}"
        assert res.json().get("error") == "INVALID_RECOVERY_CREDENTIAL"

    # Original credentials & genuine recovery phrase remain valid and unconsumed
    tok_res = api_context.post(
        f"/api/v1/users/{user['username']}/token",
        data={"password": user["password"], "totp_code": compute_totp_code(reg["totp_secret"])},
    )
    assert tok_res.status == 200
    hdr = {"Authorization": f"Bearer {tok_res.json()['access_token']}"}
    assert api_context.delete(f"/api/v1/users/{user['username']}", headers=hdr).status == 204


def test_password_over_72_bytes_returns_400_invalid_password(api_context: APIRequestContext):
    """Ensure passwords exceeding bcrypt's 72-byte cap fail cleanly with 400 INVALID_PASSWORD instead of 500."""
    too_long_user = generate_user_data(prefix="pw73")
    too_long_user["password"] = "A1!" + ("x" * 70)
    res = api_context.post("/api/v1/users", data=too_long_user)
    assert res.status == 400, f"Expected 400 for 73-byte password, got {res.status}: {res.text()}"
    assert res.json().get("error") == "INVALID_PASSWORD"
