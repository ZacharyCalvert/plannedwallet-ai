import uuid
import pytest
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

def test_snapshot_account_lifecycle_and_trending(api_context: APIRequestContext):
    """
    Test full lifecycle of a snapshot account:
    - Create snapshot account (e.g. Trading Account)
    - Add yesterday's valuation (1200.00) and today's valuation (1210.00)
    - Add an earlier valuation (1180.50)
    - Verify snapshots list is ordered chronologically by snapshot_date ASC
    - Test date filtering with start_date and end_date
    - Retrieve single snapshot by ID
    - Update a snapshot valuation
    - Delete a snapshot
    """
    user_a, token_a = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token_a}"}

    # 1. Create snapshot account
    acc_res = api_context.post(
        "/api/v1/accounts",
        data={
            "name": "Trading Account",
            "type": "snapshot",
            "description": "Manual valuations",
        },
        headers=headers,
    )
    assert acc_res.status == 201, f"Failed to create snapshot account: {acc_res.text()}"
    account = acc_res.json()
    account_id = account["id"]
    assert account["type"] == "snapshot"

    # 2. Add snapshots out of chronological order to verify sorting
    # Nov 10: 1200.00 (yesterday)
    res_nov10 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 1200.00, "snapshot_date": "2026-11-10"},
        headers=headers,
    )
    assert res_nov10.status == 201, f"Failed to add snapshot: {res_nov10.text()}"
    snap_nov10 = res_nov10.json()
    assert snap_nov10["value"] == 1200.00
    assert snap_nov10["snapshot_date"] == "2026-11-10"

    # Nov 11: 1210.00 (today)
    res_nov11 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 1210.00, "snapshot_date": "2026-11-11"},
        headers=headers,
    )
    assert res_nov11.status == 201
    snap_nov11 = res_nov11.json()
    assert snap_nov11["value"] == 1210.00

    # Nov 08: 1180.50 (earlier)
    res_nov08 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 1180.50, "snapshot_date": "2026-11-08"},
        headers=headers,
    )
    assert res_nov08.status == 201
    snap_nov08 = res_nov08.json()

    # 3. List snapshots - verify chronological ordering (Nov 08 -> Nov 10 -> Nov 11)
    list_res = api_context.get(f"/api/v1/accounts/{account_id}/snapshots", headers=headers)
    assert list_res.status == 200
    snapshots = list_res.json()
    assert len(snapshots) == 3
    assert [s["snapshot_date"] for s in snapshots] == ["2026-11-08", "2026-11-10", "2026-11-11"]
    assert [s["value"] for s in snapshots] == [1180.50, 1200.00, 1210.00]

    # 4. Filter snapshots by start_date and end_date
    filter_res = api_context.get(
        f"/api/v1/accounts/{account_id}/snapshots?start_date=2026-11-10&end_date=2026-11-11",
        headers=headers,
    )
    assert filter_res.status == 200
    filtered = filter_res.json()
    assert len(filtered) == 2
    assert [s["snapshot_date"] for s in filtered] == ["2026-11-10", "2026-11-11"]

    # 5. Get single snapshot by ID
    get_res = api_context.get(
        f"/api/v1/accounts/{account_id}/snapshots/{snap_nov10['id']}",
        headers=headers,
    )
    assert get_res.status == 200
    assert get_res.json()["value"] == 1200.00
    assert get_res.json()["snapshot_date"] == "2026-11-10"

    # 6. Update snapshot valuation
    put_res = api_context.put(
        f"/api/v1/accounts/{account_id}/snapshots/{snap_nov10['id']}",
        data={"value": 1205.50, "snapshot_date": "2026-11-10"},
        headers=headers,
    )
    assert put_res.status == 200
    assert put_res.json()["value"] == 1205.50

    # 7. Delete snapshot
    del_res = api_context.delete(
        f"/api/v1/accounts/{account_id}/snapshots/{snap_nov08['id']}",
        headers=headers,
    )
    assert del_res.status == 204

    # Verify count decreased to 2
    list_after = api_context.get(f"/api/v1/accounts/{account_id}/snapshots", headers=headers).json()
    assert len(list_after) == 2
    assert [s["snapshot_date"] for s in list_after] == ["2026-11-10", "2026-11-11"]


def test_snapshot_uniqueness_constraint_and_conflicts(api_context: APIRequestContext):
    """
    Verify unique constraint on (account_id, snapshot_date):
    - Attempting to record two valuations on the same date produces 409 Conflict
    - Attempting to update a snapshot's date to match an existing date produces 409 Conflict
    """
    user_a, token_a = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token_a}"}

    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Crypto Snapshot", "type": "snapshot", "description": "Crypto cold storage"},
        headers=headers,
    )
    assert acc_res.status == 201
    account_id = acc_res.json()["id"]

    # First record on 2026-11-12
    res1 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 50000.00, "snapshot_date": "2026-11-12"},
        headers=headers,
    )
    assert res1.status == 201

    # Second record on the same date -> 409 Conflict
    res2 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 51000.00, "snapshot_date": "2026-11-12"},
        headers=headers,
    )
    assert res2.status == 409, f"Expected 409 on duplicate date, got {res2.status}: {res2.text()}"

    # Add record on 2026-11-13
    res3 = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 52000.00, "snapshot_date": "2026-11-13"},
        headers=headers,
    )
    assert res3.status == 201
    snap_nov13_id = res3.json()["id"]

    # Try updating 2026-11-13 snapshot to 2026-11-12 -> 409 Conflict
    put_conflict = api_context.put(
        f"/api/v1/accounts/{account_id}/snapshots/{snap_nov13_id}",
        data={"value": 53000.00, "snapshot_date": "2026-11-12"},
        headers=headers,
    )
    assert put_conflict.status == 409, f"Expected 409 on date collision, got {put_conflict.status}: {put_conflict.text()}"


def test_snapshot_account_access_control(api_context: APIRequestContext):
    """
    Verify multi-user permissions on snapshot accounts:
    - Reader can view snapshots, but is denied (403) from adding, updating, or deleting.
    - Editor can add, update, and delete snapshots.
    - Unshared user gets 404 Not Found.
    """
    owner, owner_token = create_authenticated_user(api_context)
    reader, reader_token = create_authenticated_user(api_context)
    editor, editor_token = create_authenticated_user(api_context)
    unshared, unshared_token = create_authenticated_user(api_context)

    owner_hdr = {"Authorization": f"Bearer {owner_token}"}
    reader_hdr = {"Authorization": f"Bearer {reader_token}"}
    editor_hdr = {"Authorization": f"Bearer {editor_token}"}
    unshared_hdr = {"Authorization": f"Bearer {unshared_token}"}

    # Owner creates snapshot account
    acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Shared Trading Account", "type": "snapshot", "description": "Shared"},
        headers=owner_hdr,
    ).json()
    account_id = acc["id"]

    # Grant 'read' to reader, 'edit' to editor
    api_context.post(
        f"/api/v1/accounts/{account_id}/access",
        data={"user_id": reader["id"], "access_level": "read"},
        headers=owner_hdr,
    )
    api_context.post(
        f"/api/v1/accounts/{account_id}/access",
        data={"user_id": editor["id"], "access_level": "edit"},
        headers=owner_hdr,
    )

    # 1. Editor can create a snapshot
    res_edit_post = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 10000.00, "snapshot_date": "2026-11-01"},
        headers=editor_hdr,
    )
    assert res_edit_post.status == 201
    snapshot_id = res_edit_post.json()["id"]

    # 2. Reader can view snapshots
    res_read_list = api_context.get(f"/api/v1/accounts/{account_id}/snapshots", headers=reader_hdr)
    assert res_read_list.status == 200
    assert len(res_read_list.json()) == 1

    # 3. Reader is forbidden (403) from mutations
    res_read_post = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 10500.00, "snapshot_date": "2026-11-02"},
        headers=reader_hdr,
    )
    assert res_read_post.status == 403

    res_read_put = api_context.put(
        f"/api/v1/accounts/{account_id}/snapshots/{snapshot_id}",
        data={"value": 11000.00, "snapshot_date": "2026-11-01"},
        headers=reader_hdr,
    )
    assert res_read_put.status == 403

    res_read_del = api_context.delete(
        f"/api/v1/accounts/{account_id}/snapshots/{snapshot_id}",
        headers=reader_hdr,
    )
    assert res_read_del.status == 403

    # 4. Unshared user gets 404 Not Found (no leaking)
    res_unshared_get = api_context.get(f"/api/v1/accounts/{account_id}/snapshots", headers=unshared_hdr)
    assert res_unshared_get.status == 404

    res_unshared_post = api_context.post(
        f"/api/v1/accounts/{account_id}/snapshots",
        data={"value": 9999.00, "snapshot_date": "2026-11-02"},
        headers=unshared_hdr,
    )
    assert res_unshared_post.status == 404

    # 5. Editor can update and delete
    res_edit_put = api_context.put(
        f"/api/v1/accounts/{account_id}/snapshots/{snapshot_id}",
        data={"value": 10200.00, "snapshot_date": "2026-11-01"},
        headers=editor_hdr,
    )
    assert res_edit_put.status == 200

    res_edit_del = api_context.delete(
        f"/api/v1/accounts/{account_id}/snapshots/{snapshot_id}",
        headers=editor_hdr,
    )
    assert res_edit_del.status == 204


def test_snapshot_account_type_restrictions(api_context: APIRequestContext):
    """
    Verify cross-type validation rules:
    - Checking accounts reject snapshot operations (400 Bad Request)
    - Snapshot accounts reject checking record operations (400 Bad Request)
    - Snapshot accounts reject asset operations (400 Bad Request)
    """
    user_a, token_a = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token_a}"}

    # Create a checking account
    chk_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Checking Account", "type": "checking", "description": "Daily expenses"},
        headers=headers,
    )
    assert chk_res.status == 201
    chk_id = chk_res.json()["id"]

    # Create a snapshot account
    snap_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Snapshot Account", "type": "snapshot", "description": "Manual trend"},
        headers=headers,
    )
    assert snap_res.status == 201
    snap_id = snap_res.json()["id"]

    # 1. Checking account rejects snapshot operations
    res_chk_snap = api_context.post(
        f"/api/v1/accounts/{chk_id}/snapshots",
        data={"value": 500.00, "snapshot_date": "2026-11-12"},
        headers=headers,
    )
    assert res_chk_snap.status == 400
    assert "INVALID_ACCOUNT_TYPE" in res_chk_snap.text()

    # 2. Snapshot account rejects checking records
    res_snap_rec = api_context.post(
        f"/api/v1/accounts/{snap_id}/records",
        data={
            "value": 100.00,
            "date_sent": "2026-11-12",
            "entity": "Trader",
            "description": "Invalid for snapshot account",
        },
        headers=headers,
    )
    assert res_snap_rec.status == 400
    assert "INVALID_ACCOUNT_TYPE" in res_snap_rec.text()

    # 3. Snapshot account rejects asset holding operations
    uid = uuid.uuid4().hex[:6]
    asset = api_context.post(
        "/api/v1/assets",
        data={"label": f"tkn_{uid}", "description": "Test Token"},
        headers=headers,
    ).json()

    res_snap_asset = api_context.post(
        f"/api/v1/accounts/{snap_id}/assets",
        data={"asset_id": asset["id"], "quantity": 10.0},
        headers=headers,
    )
    assert res_snap_asset.status == 400
    assert "INVALID_ACCOUNT_TYPE" in res_snap_asset.text()
