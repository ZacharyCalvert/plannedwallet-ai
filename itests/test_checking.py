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
        "name": f"Checking_{uuid.uuid4().hex[:4]}",
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

    account_name = f"Everyday_Checking_{uuid.uuid4().hex[:4]}"
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


def test_account_patch_name_and_description(api_context: APIRequestContext):
    """Validate PATCH /api/v1/accounts/{accountId} updates name and/or description and enforces access/validation."""
    _, owner_token = create_authenticated_user(api_context)
    reader, reader_token = create_authenticated_user(api_context)
    editor, editor_token = create_authenticated_user(api_context)
    _, outsider_token = create_authenticated_user(api_context)

    owner_hdr = {"Authorization": f"Bearer {owner_token}"}
    reader_hdr = {"Authorization": f"Bearer {reader_token}"}
    editor_hdr = {"Authorization": f"Bearer {editor_token}"}
    outsider_hdr = {"Authorization": f"Bearer {outsider_token}"}

    # 1. Create account and seed a checking record
    orig_name = f"Primary_Checking_{uuid.uuid4().hex[:4]}"
    create_res = api_context.post(
        "/api/v1/accounts",
        data={"name": orig_name, "type": "checking", "description": "Original checking description"},
        headers=owner_hdr,
    )
    assert create_res.status == 201, f"Failed to create account: {create_res.text()}"
    account_id = create_res.json()["id"]
    record = add_checking_record(api_context, account_id, owner_hdr)

    # 2a. Owner updates name only (description stays intact)
    renamed = f"Household_Checking_{uuid.uuid4().hex[:4]}"
    patch_res = api_context.patch(
        f"/api/v1/accounts/{account_id}",
        data={"name": f"  {renamed}  "},
        headers=owner_hdr,
    )
    assert patch_res.status == 200, f"Name patch failed: {patch_res.status} {patch_res.text()}"
    patched_acc = patch_res.json()
    assert patched_acc["name"] == renamed
    assert patched_acc["type"] == "checking"
    assert patched_acc["description"] == "Original checking description"

    # 2b. Owner updates description only (name stays intact)
    patch_desc_res = api_context.patch(
        f"/api/v1/accounts/{account_id}",
        data={"description": "  Updated checking description  "},
        headers=owner_hdr,
    )
    assert patch_desc_res.status == 200, f"Description patch failed: {patch_desc_res.status} {patch_desc_res.text()}"
    assert patch_desc_res.json()["name"] == renamed
    assert patch_desc_res.json()["description"] == "Updated checking description"

    # 2c. Owner updates both name and description together
    final_name = f"Operating_Checking_{uuid.uuid4().hex[:4]}"
    final_desc = "Combined monthly bills and household spending"
    patch_both_res = api_context.patch(
        f"/api/v1/accounts/{account_id}",
        data={"name": final_name, "description": final_desc},
        headers=owner_hdr,
    )
    assert patch_both_res.status == 200, f"Combined patch failed: {patch_both_res.status} {patch_both_res.text()}"
    assert patch_both_res.json()["name"] == final_name
    assert patch_both_res.json()["description"] == final_desc

    # Verify persistence via GET single + GET list + records still intact
    get_res = api_context.get(f"/api/v1/accounts/{account_id}", headers=owner_hdr)
    assert get_res.status == 200
    assert get_res.json()["name"] == final_name
    assert get_res.json()["description"] == final_desc

    list_res = api_context.get("/api/v1/accounts", headers=owner_hdr)
    assert list_res.status == 200
    assert any(a["id"] == account_id and a["name"] == final_name and a["description"] == final_desc for a in list_res.json())

    rec_res = api_context.get(f"/api/v1/accounts/{account_id}/records", headers=owner_hdr)
    assert rec_res.status == 200
    assert any(r["id"] == record["id"] for r in rec_res.json())

    # 3. Reject payloads with immutable/disallowed fields, missing patch fields, or invalid values
    invalid_payloads = [
        {},
        {"name": ""},
        {"name": "   "},
        {"name": "x" * 101},
        {"description": ""},
        {"description": "   "},
        {"description": "d" * 256},
        {"type": "savings"},
        {"name": "Blocked Rename", "type": "savings"},
        {"name": "Blocked Rename", "admin_id": reader["id"]},
    ]
    for bad_payload in invalid_payloads:
        bad_res = api_context.patch(f"/api/v1/accounts/{account_id}", data=bad_payload, headers=owner_hdr)
        assert bad_res.status == 400, f"Expected 400 for {bad_payload}, got {bad_res.status}: {bad_res.text()}"

    # 4. Enforce access control (unshared -> 404, read -> 403, edit -> 200)
    assert api_context.patch(f"/api/v1/accounts/{account_id}", data={"name": "Outsider"}, headers=outsider_hdr).status == 404

    assert api_context.post(
        f"/api/v1/accounts/{account_id}/access",
        data={"user_id": reader["id"], "access_level": "read"},
        headers=owner_hdr,
    ).status == 200
    assert api_context.post(
        f"/api/v1/accounts/{account_id}/access",
        data={"user_id": editor["id"], "access_level": "edit"},
        headers=owner_hdr,
    ).status == 200

    assert api_context.patch(f"/api/v1/accounts/{account_id}", data={"name": "Reader"}, headers=reader_hdr).status == 403

    editor_name = f"Editor_Renamed_{uuid.uuid4().hex[:4]}"
    editor_res = api_context.patch(
        f"/api/v1/accounts/{account_id}",
        data={"name": editor_name, "description": "Edited by shared editor"},
        headers=editor_hdr,
    )
    assert editor_res.status == 200, f"Expected 200 for editor patch, got {editor_res.status}: {editor_res.text()}"
    assert editor_res.json()["name"] == editor_name
    assert editor_res.json()["description"] == "Edited by shared editor"


def test_account_rejects_five_or_more_digits(api_context: APIRequestContext):
    """Ensure account creation and edits allow up to 4 digits but reject 5+ digits."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 4 digits in name and 4 digits in description must succeed
    ok_res = api_context.post(
        "/api/v1/accounts",
        data={
            "name": "Rewards Card 1234",
            "type": "credit card",
            "description": "Ending 9876",
        },
        headers=headers,
    )
    assert ok_res.status == 201, f"Expected 201 for 4-digit fields, got {ok_res.status}: {ok_res.text()}"
    account_id = ok_res.json()["id"]

    # 5+ digits in name or description must be rejected on POST
    bad_creates = [
        {"name": "Checking 12345", "type": "checking", "description": "Everyday checking"},
        {"name": "Card 4111-2222-3333-4444", "type": "credit card", "description": "Primary card"},
        {"name": "Everyday Checking", "type": "checking", "description": "Routing 012345678"},
        {"name": "Personal Savings", "type": "savings", "description": "SSN 123-45-6789"},
    ]
    for payload in bad_creates:
        res = api_context.post("/api/v1/accounts", data=payload, headers=headers)
        assert res.status == 400, f"Expected 400 for {payload}, got {res.status}: {res.text()}"
        assert res.json().get("error") == "PROHIBITED_SENSITIVE_DIGITS"

    # 4 digits allowed on PATCH; 5+ digits rejected on PATCH
    patch_ok = api_context.patch(
        f"/api/v1/accounts/{account_id}",
        data={"name": "Rewards 4321"},
        headers=headers,
    )
    assert patch_ok.status == 200, f"Expected 200 for 4-digit PATCH, got {patch_ok.status}: {patch_ok.text()}"

    for bad_patch in ({"name": "Card 12345"}, {"description": "Acct 98-765"}):
        patch_bad = api_context.patch(f"/api/v1/accounts/{account_id}", data=bad_patch, headers=headers)
        assert patch_bad.status == 400, f"Expected 400 for {bad_patch}, got {patch_bad.status}: {patch_bad.text()}"
        assert patch_bad.json().get("error") == "PROHIBITED_SENSITIVE_DIGITS"


