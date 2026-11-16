import uuid
from playwright.sync_api import APIRequestContext
from test_checking import create_authenticated_user

DEFAULT_CATEGORY_NAMES = {
    "Housing",
    "Transportation",
    "Groceries",
    "Utilities",
    "Healthcare",
    "Insurance",
    "Debt",
}


def test_default_categories_and_crud_lifecycle(api_context: APIRequestContext):
    """Verify 7 default categories exist, cannot be deleted, and custom categories support CRUD + nullification."""
    _, token = create_authenticated_user(api_context)
    _, other_token = create_authenticated_user(api_context)
    headers = {"Authorization": f"Bearer {token}"}
    other_headers = {"Authorization": f"Bearer {other_token}"}

    list_res = api_context.get("/api/v1/categories", headers=headers)
    assert list_res.status == 200, f"List categories failed: {list_res.text()}"
    cats = list_res.json()
    names = {c["name"] for c in cats}
    assert DEFAULT_CATEGORY_NAMES.issubset(names), f"Missing default categories: {DEFAULT_CATEGORY_NAMES - names}"

    # Default categories cannot be deleted
    housing = next(c for c in cats if c["name"] == "Housing")
    assert housing["is_default"] is True
    del_default = api_context.delete(f"/api/v1/categories/{housing['id']}", headers=headers)
    assert del_default.status == 403

    # Create user-scoped category
    custom_name = f"Dining_{uuid.uuid4().hex[:6]}"
    create_res = api_context.post("/api/v1/categories", data={"name": custom_name}, headers=headers)
    assert create_res.status == 201, f"Create category failed: {create_res.text()}"
    custom = create_res.json()
    assert custom["is_default"] is False
    assert custom["name"] == custom_name

    # Duplicate name for same user returns 409
    dup_res = api_context.post("/api/v1/categories", data={"name": custom_name.lower()}, headers=headers)
    assert dup_res.status == 409

    # Other user does not see private category
    other_cats = api_context.get("/api/v1/categories", headers=other_headers).json()
    assert all(c["id"] != custom["id"] for c in other_cats)

    # Create checking account and assign custom category
    acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Everyday Checking", "type": "checking", "description": "Category test"},
        headers=headers,
    ).json()
    rec = api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -32.50, "date_sent": "2026-11-10", "entity": "Local Cafe", "category_id": custom["id"]},
        headers=headers,
    ).json()
    assert rec["category_id"] == custom["id"]

    # Delete category -> record becomes uncategorized (category_id == None)
    del_res = api_context.delete(f"/api/v1/categories/{custom['id']}", headers=headers)
    assert del_res.status == 204
    rec_after = api_context.get(f"/api/v1/accounts/{acc['id']}/records/{rec['id']}", headers=headers).json()
    assert rec_after["category_id"] is None


def test_shared_account_category_visibility_and_expense_summary(api_context: APIRequestContext):
    """Verify shared account exposes used categories and expense summary aggregates expenses by category."""
    _, owner_token = create_authenticated_user(api_context)
    shared_user, shared_token = create_authenticated_user(api_context)
    owner_headers = {"Authorization": f"Bearer {owner_token}"}
    shared_headers = {"Authorization": f"Bearer {shared_token}"}

    # Owner creates custom category and credit card account
    custom = api_context.post(
        "/api/v1/categories",
        data={"name": f"Travel_{uuid.uuid4().hex[:6]}"},
        headers=owner_headers,
    ).json()
    acc = api_context.post(
        "/api/v1/accounts",
        data={"name": "Shared Credit Card", "type": "credit card", "description": "Shared card"},
        headers=owner_headers,
    ).json()

    cats = api_context.get("/api/v1/categories", headers=owner_headers).json()
    groceries = next(c for c in cats if c["name"] == "Groceries")

    # Add categorized expenses, uncategorized expense, and income (ignored in expense pie chart)
    api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -120.00, "date_sent": "2026-11-01", "entity": "Corner Grocery", "category_id": groceries["id"]},
        headers=owner_headers,
    )
    api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -350.00, "date_sent": "2026-11-02", "entity": "Regional Rail", "category_id": custom["id"]},
        headers=owner_headers,
    )
    api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": -30.00, "date_sent": "2026-11-03", "entity": "Misc Shop"},
        headers=owner_headers,
    )
    api_context.post(
        f"/api/v1/accounts/{acc['id']}/records",
        data={"value": 500.00, "date_sent": "2026-11-04", "entity": "Card Payment"},
        headers=owner_headers,
    )

    # Share account with shared_user
    grant_res = api_context.post(
        f"/api/v1/accounts/{acc['id']}/access",
        data={"user_id": shared_user["id"], "access_level": "edit"},
        headers=owner_headers,
    )
    assert grant_res.status in (200, 201), f"Grant access failed: {grant_res.text()}"

    # Shared user now sees owner's custom category because it is used in the shared account
    shared_cats = api_context.get("/api/v1/categories", headers=shared_headers).json()
    assert any(c["id"] == custom["id"] for c in shared_cats)

    # Shared user cannot delete owner's custom category
    del_forbidden = api_context.delete(f"/api/v1/categories/{custom['id']}", headers=shared_headers)
    assert del_forbidden.status == 403

    # Expense summary for pie chart
    summary_res = api_context.get(f"/api/v1/accounts/{acc['id']}/records/categories", headers=shared_headers)
    assert summary_res.status == 200, f"Summary failed: {summary_res.text()}"
    summary = summary_res.json()
    assert summary["account_id"] == acc["id"]
    assert summary["total_expenses"] == 500.00
    assert len(summary["categories"]) == 3
    assert summary["categories"][0]["category_name"] == custom["name"]
    assert summary["categories"][0]["total_amount"] == 350.00
    assert summary["categories"][1]["category_name"] == "Groceries"
    assert summary["categories"][1]["total_amount"] == 120.00
    assert summary["categories"][2]["category_name"] == "Uncategorized"
    assert summary["categories"][2]["total_amount"] == 30.00
