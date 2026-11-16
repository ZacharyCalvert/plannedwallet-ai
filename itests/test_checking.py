import uuid
import pytest
from playwright.sync_api import APIRequestContext
from conftest import compute_totp_code

TEST_PASSWORD = "ComplexPassword123!"

def create_authenticated_user(api_context: APIRequestContext) -> tuple[dict, str]:
    """Helper to create and authenticate a user, returning user data and JWT token."""
    uid = uuid.uuid4().hex[:8]
    user_data = {
        "username": f"user_{uid}",
        "email": f"user_{uid}@example.com",
        "password": TEST_PASSWORD,
    }
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201, f"User creation failed: {create_res.text()}"
    user_data["id"] = create_res.json()["id"]
    totp_secret = create_res.json()["totp_secret"]
    totp_code = compute_totp_code(totp_secret)

    token_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"], "totp_code": totp_code},
    )
    assert token_res.status == 200, f"Login failed: {token_res.text()}"
    return user_data, token_res.json()["access_token"]

def create_checking_account(api_context: APIRequestContext, headers: dict) -> dict:
    """Helper to create a checking account."""
    payload = {
        "name": f"Checking_{uuid.uuid4().hex[:6]}",
        "type": "checking",
        "description": "Primary checking account for integration test",
    }
    res = api_context.post("/api/v1/accounts", data=payload, headers=headers)
    assert res.status == 201, f"Failed to create account: {res.text()}"
    return res.json()

def add_checking_record(api_context: APIRequestContext, account_id: str, headers: dict) -> dict:
    """Helper to add a checking record to an account."""
    payload = {
        "value": -42.50,
        "date_sent": "2026-11-10",
        "entity": "Corner Grocery",
        "description": "Weekly grocery run",
    }
    res = api_context.post(
        f"/api/v1/accounts/{account_id}/records",
        data=payload,
        headers=headers,
    )
    assert res.status == 201, f"Add record failed: {res.text()}"
    return res.json()

def test_create_checking_account_and_records(api_context: APIRequestContext):
    """Create checking account, confirm via get accounts, add record, confirm via get records."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Create checking account and verify via get accounts API
    account = create_checking_account(api_context, headers)
    acc_res = api_context.get("/api/v1/accounts", headers=headers)
    assert acc_res.status == 200, f"Get accounts failed: {acc_res.text()}"
    accounts = acc_res.json()
    assert any(acc["id"] == account["id"] for acc in accounts)

    # 2. Add record to account and verify via get records API
    record = add_checking_record(api_context, account["id"], headers)
    rec_res = api_context.get(
        f"/api/v1/accounts/{account['id']}/records",
        headers=headers,
    )
    assert rec_res.status == 200, f"Get records failed: {rec_res.text()}"
    records = rec_res.json()
    assert any(r["id"] == record["id"] and r["entity"] == "Corner Grocery" for r in records)


def test_account_deletion_and_recreation(api_context: APIRequestContext):
    """Validate account deletion (admin only) and recreation with the same name."""
    # 1. Create User A and an account
    _, token_a = create_authenticated_user(api_context)
    headers_a = {"Authorization": f"Bearer {token_a}"}

    account_name = f"Everyday_Checking_{uuid.uuid4().hex[:6]}"
    create_payload = {
        "name": account_name,
        "type": "checking",
        "description": "Primary daily operating checking account",
    }
    create_res = api_context.post("/api/v1/accounts", data=create_payload, headers=headers_a)
    assert create_res.status == 201, f"Failed to create account: {create_res.text()}"
    account = create_res.json()
    account_id = account["id"]

    # 2. Add record to account
    record = add_checking_record(api_context, account_id, headers_a)
    assert record["id"]

    # 3. Create User B and verify User B cannot delete User A's account (403 Forbidden)
    _, token_b = create_authenticated_user(api_context)
    headers_b = {"Authorization": f"Bearer {token_b}"}
    del_forbidden_res = api_context.delete(f"/api/v1/accounts/{account_id}", headers=headers_b)
    assert del_forbidden_res.status == 403, (
        f"Expected 403 when non-creator deletes account, got {del_forbidden_res.status}: {del_forbidden_res.text()}"
    )

    # 4. User A deletes account (204 No Content)
    del_res = api_context.delete(f"/api/v1/accounts/{account_id}", headers=headers_a)
    assert del_res.status == 204, f"Delete account failed: {del_res.status} {del_res.text()}"

    # 5. Verify account is gone (404 Not Found)
    get_res = api_context.get(f"/api/v1/accounts/{account_id}", headers=headers_a)
    assert get_res.status == 404, f"Expected 404 for deleted account, got {get_res.status}"

    # 6. Re-create a similar account with the exact same name (201 Created)
    recreate_res = api_context.post("/api/v1/accounts", data=create_payload, headers=headers_a)
    assert recreate_res.status == 201, (
        f"Re-creating account with same name failed: {recreate_res.status} {recreate_res.text()}"
    )
    new_account = recreate_res.json()
    assert new_account["id"] != account_id
    assert new_account["name"] == account_name

