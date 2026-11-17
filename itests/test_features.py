from playwright.sync_api import APIRequestContext


def test_get_features_endpoint(raw_context: APIRequestContext):
    """Verify GET /api/v1/features returns active backend feature flags list."""
    res = raw_context.get(
        "/api/v1/features",
        headers={"X-Client-ID": "itest-features"},
    )
    assert res.status == 200, f"Unexpected status: {res.status} {res.text()}"
    body = res.json()
    assert isinstance(body.get("features"), list)
    # Default integration environment runs with PW_READONLY=false
    assert "read_only" not in body["features"]
