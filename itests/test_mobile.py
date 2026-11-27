import hashlib
import uuid
from playwright.sync_api import APIRequestContext
from conftest import compute_totp_code, decode_jwt_unverified
from test_checking import create_authenticated_user, TEST_PASSWORD


def test_mobile_widget_token_isolation_device_binding_and_record_entry(api_context: APIRequestContext):
    """Verify /api/v1/mobile/* 6-month device-bound token flow, strict scope isolation, and revocation."""
    uid = uuid.uuid4().hex[:8]
    user = {
        "username": f"mob_{uid}",
        "email": f"mob_{uid}@example.com",
        "password": TEST_PASSWORD,
    }
    reg_res = api_context.post("/api/v1/users", data=user)
    assert reg_res.status == 201
    reg_body = reg_res.json()
    user["id"] = reg_body["id"]
    user["totp_secret"] = reg_body["totp_secret"]
    tok_res = api_context.post(
        f"/api/v1/users/{user['username']}/token",
        data={"password": user["password"], "totp_code": compute_totp_code(user["totp_secret"])},
    )
    assert tok_res.status == 200
    std_token = tok_res.json()["access_token"]

    other_user, other_token = create_authenticated_user(api_context)
    std_hdr = {"Authorization": f"Bearer {std_token}"}
    other_hdr = {"Authorization": f"Bearer {other_token}"}

    # Create checking, savings, credit card, snapshot, asset, and shared accounts
    chk_id = api_context.post(
        "/api/v1/accounts",
        headers=std_hdr,
        data={"name": "Primary Checking", "type": "checking", "description": "Checking account"},
    ).json()["id"]
    sav_id = api_context.post(
        "/api/v1/accounts",
        headers=std_hdr,
        data={"name": "Emergency Savings", "type": "savings", "description": "Savings account"},
    ).json()["id"]
    cc_id = api_context.post(
        "/api/v1/accounts",
        headers=std_hdr,
        data={"name": "Everyday Credit Card", "type": "credit card", "description": "Credit account"},
    ).json()["id"]
    snap_id = api_context.post(
        "/api/v1/accounts",
        headers=std_hdr,
        data={"name": "Brokerage Snapshot", "type": "snapshot", "description": "Snapshot account"},
    ).json()["id"]
    asset_id = api_context.post(
        "/api/v1/accounts",
        headers=std_hdr,
        data={"name": "Precious Metals", "type": "asset", "description": "Asset account"},
    ).json()["id"]

    other_ro_id = api_context.post(
        "/api/v1/accounts",
        headers=other_hdr,
        data={"name": "Other Read Only", "type": "checking", "description": "Shared read"},
    ).json()["id"]
    other_unshared_id = api_context.post(
        "/api/v1/accounts",
        headers=other_hdr,
        data={"name": "Other Private", "type": "checking", "description": "Unshared"},
    ).json()["id"]
    assert api_context.post(
        f"/api/v1/accounts/{other_ro_id}/access",
        headers=other_hdr,
        data={"user_id": user["id"], "access_level": "read"},
    ).status == 200

    # Issuing mobile token without X-Mobile-Identifier fails with 400
    no_hdr_issue = api_context.post("/api/v1/mobile/token", headers=std_hdr)
    assert no_hdr_issue.status == 400

    device_id = f"mobile-install-{uuid.uuid4()}"
    expected_device_hash = hashlib.sha256(device_id.encode("utf-8")).hexdigest()

    issue_res = api_context.post(
        "/api/v1/mobile/token",
        headers={**std_hdr, "X-Mobile-Identifier": device_id},
    )
    assert issue_res.status == 201, f"Failed to issue mobile token: {issue_res.text()}"
    mob_data = issue_res.json()
    assert mob_data["limitation"] == "mobile"
    assert mob_data["expires_in"] == 180 * 24 * 3600
    assert mob_data["token_type"] == "Bearer"

    claims = decode_jwt_unverified(mob_data["access_token"])
    assert claims.get("limitation") == "mobile"
    assert claims.get("device_hash") == expected_device_hash
    assert int(claims["exp"]) - int(claims["iat"]) == 180 * 24 * 3600

    mob_hdr = {
        "Authorization": f"Bearer {mob_data['access_token']}",
        "X-Mobile-Identifier": device_id,
    }

    # Mobile-limited token MUST NOT access standard endpoints or mint another mobile token
    for method, url in (
        ("GET", "/api/v1/accounts"),
        ("GET", f"/api/v1/accounts/{chk_id}/records"),
        ("GET", f"/api/v1/users/{user['username']}"),
        ("POST", "/api/v1/mobile/token"),
    ):
        r = api_context.get(url, headers=mob_hdr) if method == "GET" else api_context.post(url, headers=mob_hdr)
        assert r.status == 403, f"Expected 403 for mobile token on {method} {url}, got {r.status}"
        assert r.json().get("error") == "TOKEN_SCOPE_LIMITED"

    # Standard 15m token MUST NOT access mobile endpoints
    assert api_context.get(
        "/api/v1/mobile/accounts",
        headers={**std_hdr, "X-Mobile-Identifier": device_id},
    ).status == 403

    # Wrong or missing X-Mobile-Identifier MUST fail with 401
    for bad_hdr in (
        {"Authorization": f"Bearer {mob_data['access_token']}"},
        {"Authorization": f"Bearer {mob_data['access_token']}", "X-Mobile-Identifier": "different-device-id"},
    ):
        r = api_context.get("/api/v1/mobile/accounts", headers=bad_hdr)
        assert r.status == 401
        assert r.json().get("error") == "INVALID_MOBILE_IDENTIFIER"

    # List mobile accounts + categories exposes only editable checking/savings/credit card IDs/names/types
    targets_res = api_context.get("/api/v1/mobile/accounts", headers=mob_hdr)
    assert targets_res.status == 200, f"Expected 200 listing mobile targets: {targets_res.text()}"
    targets = targets_res.json()
    acc_ids = {a["id"] for a in targets["accounts"]}
    assert acc_ids == {chk_id, sav_id, cc_id}
    for acc_item in targets["accounts"]:
        assert set(acc_item.keys()) == {"id", "name", "type"}
    assert len(targets["categories"]) > 0
    cat_id = targets["categories"][0]["id"]

    # Adding records works for checking, savings, and credit card with minimal response
    for target_acc, label in ((chk_id, "Corner Grocery"), (sav_id, "Reserve Transfer"), (cc_id, "Coffee Shop")):
        create_rec = api_context.post(
            f"/api/v1/mobile/accounts/{target_acc}/records",
            headers=mob_hdr,
            data={
                "value": -19.25,
                "date_sent": "2026-11-26",
                "entity": label,
                "category_id": cat_id,
            },
        )
        assert create_rec.status == 201, f"Failed creating mobile record on {target_acc}: {create_rec.text()}"
        body = create_rec.json()
        assert set(body.keys()) == {"id", "account_id", "created_at"}
        assert body["account_id"] == target_acc

    # Verify created record appears via standard account records API
    recs = api_context.get(f"/api/v1/accounts/{cc_id}/records", headers=std_hdr).json()
    assert len(recs) == 1
    assert recs[0]["entity"] == "Coffee Shop"
    assert recs[0]["value"] == -19.25
    assert recs[0]["category_id"] == cat_id

    # Snapshot, asset, read-only shared, and unshared accounts rejected appropriately
    payload = {"value": -5.0, "date_sent": "2026-11-26", "entity": "Invalid Target"}
    assert api_context.post(f"/api/v1/mobile/accounts/{snap_id}/records", headers=mob_hdr, data=payload).status == 400
    assert api_context.post(f"/api/v1/mobile/accounts/{asset_id}/records", headers=mob_hdr, data=payload).status == 400
    assert api_context.post(f"/api/v1/mobile/accounts/{other_ro_id}/records", headers=mob_hdr, data=payload).status == 403
    assert api_context.post(f"/api/v1/mobile/accounts/{other_unshared_id}/records", headers=mob_hdr, data=payload).status == 404

    # Extra unknown field rejected by bindStrictJSON
    assert api_context.post(
        f"/api/v1/mobile/accounts/{chk_id}/records",
        headers=mob_hdr,
        data={**payload, "description": "Unexpected"},
    ).status == 400

    # Password change revokes active mobile tokens immediately
    pw_res = api_context.post(
        f"/api/v1/users/{user['username']}/password",
        headers=std_hdr,
        data={
            "old_password": user["password"],
            "new_password": "UpdatedComplexPassword999!",
            "new_password_confirm": "UpdatedComplexPassword999!",
            "totp_code": compute_totp_code(user["totp_secret"]),
        },
    )
    assert pw_res.status == 200
    revoked_res = api_context.get("/api/v1/mobile/accounts", headers=mob_hdr)
    assert revoked_res.status == 401
    assert revoked_res.json().get("error") == "TOKEN_REVOKED"

    # Cleanup
    assert api_context.delete(f"/api/v1/users/{user['username']}", headers=std_hdr).status == 204
    assert api_context.delete(f"/api/v1/users/{other_user['username']}", headers=other_hdr).status == 204
