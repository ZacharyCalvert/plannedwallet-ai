import uuid
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user


def test_reconcile_report_positive_lifecycle(api_context: APIRequestContext):
    """Create reconcile report on credit card, verify cleared date default, report listing, and allowed description/category edits."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    acc_res = api_context.post(
        "/api/v1/accounts",
        data={"name": "Monthly Credit Card", "type": "credit card", "description": "Card for reconcile tests"},
        headers=headers,
    )
    assert acc_res.status == 201, f"Account creation failed: {acc_res.text()}"
    acc = acc_res.json()

    # Record 1: no cleared date; Record 2: already has cleared date
    r1 = api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -30.00, "date_sent": "2026-11-05", "entity": "Local Bistro"},
        headers=headers,
    ).json()
    r2 = api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -70.00, "date_sent": "2026-11-08", "date_cleared": "2026-11-09", "entity": "Corner Grocery"},
        headers=headers,
    ).json()

    rep_res = api_context.post(
        f"/api/v1/accounts/{acc['id']}/reconcile-reports",
        data={
            "name": "Nov2026",
            "report_date": "2026-11-16",
            "starting_balance": 0.00,
            "ending_balance": -100.00,
            "record_ids": [r1["id"], r2["id"]],
        },
        headers=headers,
    )
    assert rep_res.status == 201, f"Reconcile report creation failed: {rep_res.text()}"
    report = rep_res.json()
    assert report["name"] == "Nov2026"
    assert report["record_count"] == 2

    # Uncleared record receives report_date as date_cleared; cleared record keeps original date_cleared
    r1_after = api_context.get(f"/api/v1/accounts/{acc['id']}/records/{r1['id']}", headers=headers).json()
    r2_after = api_context.get(f"/api/v1/accounts/{acc['id']}/records/{r2['id']}", headers=headers).json()
    assert r1_after["reconcile_report_id"] == report["id"]
    assert r1_after["date_cleared"] == "2026-11-16"
    assert r2_after["date_cleared"] == "2026-11-09"

    # List and Get reconcile reports
    list_res = api_context.get(f"/api/v1/accounts/{acc['id']}/reconcile-reports", headers=headers)
    assert list_res.status == 200
    assert len(list_res.json()) == 1

    get_res = api_context.get(f"/api/v1/accounts/{acc['id']}/reconcile-reports/{report['id']}", headers=headers)
    assert get_res.status == 200
    assert len(get_res.json()["records"]) == 2

    # Description and category may still be modified on reconciled records
    cats = api_context.get("/api/v1/categories", headers=headers).json()
    groceries = next(c for c in cats if c["name"] == "Groceries")
    patch_ok = api_context.patch(
        f"/api/v1/accounts/{acc['id']}/records/{r1['id']}",
        data={"description": "Dinner including tip", "category_id": groceries["id"]},
        headers=headers,
    )
    assert patch_ok.status == 200, f"Expected description/category patch to succeed: {patch_ok.text()}"
    assert patch_ok.json()["description"] == "Dinner including tip"
    assert patch_ok.json()["category_id"] == groceries["id"]


def test_reconcile_report_negative_cases(api_context: APIRequestContext):
    """Validate 400 balance mismatch, 409 duplicate reconciliation, 409 immutable fields, 403 read-only, and 400 snapshot rejection."""
    _, owner_token = create_authenticated_user(api_context)
    reader, reader_token = create_authenticated_user(api_context)
    owner_headers = {"Authorization": f"Bearer {owner_token}"}
    reader_headers = {"Authorization": f"Bearer {reader_token}"}

    sav = api_context.post(
        "/api/v1/accounts",
        data={"name": f"Savings_{uuid.uuid4().hex[:4]}", "type": "savings", "description": "Reserve savings"},
        headers=owner_headers,
    ).json()
    snap = api_context.post(
        "/api/v1/accounts",
        data={"name": f"Snap_{uuid.uuid4().hex[:4]}", "type": "snapshot", "description": "Snapshot account"},
        headers=owner_headers,
    ).json()

    rec = api_context.post(
        f"/api/v1/accounts/{sav['id']}/records",
        data={"value": 500.00, "date_sent": "2026-11-01", "entity": "Employer Payroll"},
        headers=owner_headers,
    ).json()

    # Balance mismatch -> 400
    mismatch_res = api_context.post(
        f"/api/v1/accounts/{sav['id']}/reconcile-reports",
        data={
            "report_date": "2026-11-16",
            "starting_balance": 0.00,
            "ending_balance": 450.00,
            "record_ids": [rec["id"]],
        },
        headers=owner_headers,
    )
    assert mismatch_res.status == 400

    # Valid reconcile report -> 201
    ok_res = api_context.post(
        f"/api/v1/accounts/{sav['id']}/reconcile-reports",
        data={
            "name": "Year-End",
            "report_date": "2026-11-16",
            "starting_balance": 100.00,
            "ending_balance": 600.00,
            "record_ids": [rec["id"]],
        },
        headers=owner_headers,
    )
    assert ok_res.status == 201

    # Cannot attach record to a second reconcile report -> 409
    dup_res = api_context.post(
        f"/api/v1/accounts/{sav['id']}/reconcile-reports",
        data={
            "report_date": "2026-11-20",
            "starting_balance": 600.00,
            "ending_balance": 1100.00,
            "record_ids": [rec["id"]],
        },
        headers=owner_headers,
    )
    assert dup_res.status == 409

    # Editing amount, sent date, cleared date, or entity on reconciled record -> 409
    for payload in [
        {"value": 550.00},
        {"date_sent": "2026-11-02"},
        {"date_cleared": "2026-11-17"},
        {"entity": "Other Employer"},
    ]:
        conflict_res = api_context.patch(
            f"/api/v1/accounts/{sav['id']}/records/{rec['id']}",
            data=payload,
            headers=owner_headers,
        )
        assert conflict_res.status == 409, f"Expected 409 for {payload}, got {conflict_res.status}"

    # Read-only shared user cannot create reconcile report -> 403, can list -> 200
    api_context.post(
        f"/api/v1/accounts/{sav['id']}/access",
        data={"user_id": reader["id"], "access_level": "read"},
        headers=owner_headers,
    )
    forbidden_res = api_context.post(
        f"/api/v1/accounts/{sav['id']}/reconcile-reports",
        data={"report_date": "2026-11-20", "starting_balance": 600.00, "ending_balance": 600.00, "record_ids": []},
        headers=reader_headers,
    )
    assert forbidden_res.status == 403
    read_list = api_context.get(f"/api/v1/accounts/{sav['id']}/reconcile-reports", headers=reader_headers)
    assert read_list.status == 200

    # Snapshot account rejects reconcile reports -> 400
    snap_res = api_context.post(
        f"/api/v1/accounts/{snap['id']}/reconcile-reports",
        data={"report_date": "2026-11-16", "starting_balance": 0.00, "ending_balance": 0.00, "record_ids": []},
        headers=owner_headers,
    )
    assert snap_res.status == 400


def test_reconcile_report_deletion_unlinks_and_unlocks_record(api_context: APIRequestContext):
    """Create a record, attach to a reconcile report, verify update fails with 409, delete report, then verify record is preserved and updatable again."""
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    acc = api_context.post(
        "/api/v1/accounts",
        data={"name": f"Checking_{uuid.uuid4().hex[:4]}", "type": "checking", "description": "Delete reconcile test"},
        headers=headers,
    ).json()

    rec = api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -40.00, "date_sent": "2026-11-05", "entity": "Local Diner"},
        headers=headers,
    ).json()

    rep_res = api_context.post(
        f"/api/v1/accounts/{acc['id']}/reconcile-reports",
        data={
            "name": "MidNov2026",
            "report_date": "2026-11-16",
            "starting_balance": 100.00,
            "ending_balance": 60.00,
            "record_ids": [rec["id"]],
        },
        headers=headers,
    )
    assert rep_res.status == 201, f"Failed to create reconcile report: {rep_res.text()}"
    report = rep_res.json()

    # Updating locked fields while reconciled fails with 409
    locked_res = api_context.patch(
        f"/api/v1/accounts/{acc['id']}/records/{rec['id']}",
        data={"value": -45.00},
        headers=headers,
    )
    assert locked_res.status == 409, f"Expected 409 on reconciled record update, got {locked_res.status}"

    # Delete reconcile report -> 204
    del_res = api_context.delete(
        f"/api/v1/accounts/{acc['id']}/reconcile-reports/{report['id']}",
        headers=headers,
    )
    assert del_res.status == 204, f"Expected 204 deleting reconcile report, got {del_res.status}"

    # Report is gone (404)
    get_rep = api_context.get(
        f"/api/v1/accounts/{acc['id']}/reconcile-reports/{report['id']}",
        headers=headers,
    )
    assert get_rep.status == 404

    # Record still exists, is unlinked, and can now be updated again
    rec_after_del = api_context.get(
        f"/api/v1/accounts/{acc['id']}/records/{rec['id']}",
        headers=headers,
    )
    assert rec_after_del.status == 200
    assert rec_after_del.json()["reconcile_report_id"] is None

    unlock_res = api_context.patch(
        f"/api/v1/accounts/{acc['id']}/records/{rec['id']}",
        data={"value": -45.00, "date_sent": "2026-11-06", "entity": "Local Bistro"},
        headers=headers,
    )
    assert unlock_res.status == 200, f"Expected 200 updating record after report deletion, got {unlock_res.text()}"
    updated = unlock_res.json()
    assert updated["value"] == -45.00
    assert updated["date_sent"] == "2026-11-06"
    assert updated["entity"] == "Local Bistro"
