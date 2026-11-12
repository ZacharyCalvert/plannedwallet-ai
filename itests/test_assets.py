import uuid
import pytest
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

def test_asset_lifecycle_and_multiday_values(api_context: APIRequestContext):
    """
    Test creation of asset, uniqueness constraint on (user_id, label),
    multi-day valuation records for Nov 9, 10, 11,
    and uniqueness constraint on (asset_id, value_date).
    """
    _, token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}

    # 1. Create asset
    asset_label = f"btc_{uuid.uuid4().hex[:4]}"
    create_asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": asset_label, "description": "Bitcoin cryptocurrency"},
        headers=headers,
    )
    assert create_asset_res.status == 201, f"Asset create failed: {create_asset_res.text()}"
    asset = create_asset_res.json()
    assert asset["label"] == asset_label
    assert asset["description"] == "Bitcoin cryptocurrency"

    # 2. Test unique constraint on (user_id, label)
    dup_asset_res = api_context.post(
        "/api/v1/assets",
        data={"label": asset_label.upper(), "description": "Duplicate Bitcoin"},
        headers=headers,
    )
    assert dup_asset_res.status == 409, f"Expected 409 conflict on duplicate asset label, got {dup_asset_res.status}"

    dup_exact_res = api_context.post(
        "/api/v1/assets",
        data={"label": asset_label, "description": "Duplicate Bitcoin exact"},
        headers=headers,
    )
    assert dup_exact_res.status == 409, f"Expected 409 conflict on exact duplicate asset label, got {dup_exact_res.status}"

    # 3. Add values for Nov 9th, Nov 10th, and Nov 11th
    dates_and_values = [
        ("2026-11-09", 91000.50),
        ("2026-11-10", 92100.00),
        ("2026-11-11", 93450.75),
    ]
    for val_date, val_num in dates_and_values:
        v_res = api_context.post(
            f"/api/v1/assets/{asset['id']}/values",
            data={"value": val_num, "value_date": val_date},
            headers=headers,
        )
        assert v_res.status == 201, f"Failed to record value for {val_date}: {v_res.text()}"
        val_data = v_res.json()
        assert val_data["asset_id"] == asset["id"]
        assert val_data["value_date"] == val_date
        assert val_data["value"] == val_num

    # 4. Enforce unique constraint on (asset_id, value_date)
    dup_val_res = api_context.post(
        f"/api/v1/assets/{asset['id']}/values",
        data={"value": 95000.00, "value_date": "2026-11-10"},
        headers=headers,
    )
    assert dup_val_res.status == 409, f"Expected 409 conflict on duplicate date for asset, got {dup_val_res.status}"

    # 5. List asset values and verify all 3 recorded
    list_vals_res = api_context.get(
        f"/api/v1/assets/{asset['id']}/values",
        headers=headers,
    )
    assert list_vals_res.status == 200, f"List asset values failed: {list_vals_res.text()}"
    vals = list_vals_res.json()
    assert len(vals) == 3
    dates = [v["value_date"] for v in vals]
    assert "2026-11-09" in dates
    assert "2026-11-10" in dates
    assert "2026-11-11" in dates
