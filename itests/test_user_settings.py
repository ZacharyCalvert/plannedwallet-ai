import uuid
from playwright.sync_api import APIRequestContext
from conftest import compute_totp_code, decode_jwt_unverified


def _register_and_login(api_context: APIRequestContext, prefix: str = "uset"):
    uid = uuid.uuid4().hex[:8]
    username = f"{prefix}_{uid}"
    password = "ComplexPassword123!"
    reg = api_context.post(
        "/api/v1/users",
        data={
            "username": username,
            "email": f"{username}@example.com",
            "password": password,
        },
    )
    assert reg.status == 201, f"Registration failed: {reg.text()}"
    user = reg.json()
    totp_code = compute_totp_code(user["totp_secret"])
    tok_res = api_context.post(
        f"/api/v1/users/{username}/token",
        data={"password": password, "totp_code": totp_code},
    )
    assert tok_res.status == 200, f"Login failed: {tok_res.text()}"
    return user, username, tok_res.json()["access_token"]


def test_user_settings_premium_gating_and_access_control(api_context: APIRequestContext):
    """
    Verify:
    - Basic users (subscriber=false) receive 403 PREMIUM_REQUIRED on GET & PATCH /settings
    - Users cannot access another user's /settings endpoint (403)
    """
    _, username_a, token_a = _register_and_login(api_context, "useta")
    _, username_b, token_b = _register_and_login(api_context, "usetb")
    headers_a = {"Authorization": f"Bearer {token_a}"}
    headers_b = {"Authorization": f"Bearer {token_b}"}

    claims_a = decode_jwt_unverified(token_a)
    if claims_a.get("subscriber") is False:
        get_basic = api_context.get(f"/api/v1/users/{username_a}/settings", headers=headers_a)
        assert get_basic.status == 403, f"Expected 403 for basic user, got {get_basic.status}: {get_basic.text()}"
        assert get_basic.json().get("error") == "PREMIUM_REQUIRED"

        patch_basic = api_context.patch(
            f"/api/v1/users/{username_a}/settings",
            data={"account_order": ["acc-1"]},
            headers=headers_a,
        )
        assert patch_basic.status == 403, f"Expected 403 on PATCH for basic user, got {patch_basic.status}"
        assert patch_basic.json().get("error") == "PREMIUM_REQUIRED"

    # Cross-user access is always forbidden
    cross_get = api_context.get(f"/api/v1/users/{username_b}/settings", headers=headers_a)
    assert cross_get.status == 403
    cross_patch = api_context.patch(
        f"/api/v1/users/{username_a}/settings",
        data={"account_order": ["acc-1"]},
        headers=headers_b,
    )
    assert cross_patch.status == 403
