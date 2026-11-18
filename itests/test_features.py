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


def test_privacy_and_security_policy_pdf_served(raw_context: APIRequestContext):
    """Verify the static Privacy & Security Policy PDF is served at /privacy-and-security-policy.pdf."""
    res = raw_context.get("/privacy-and-security-policy.pdf")
    assert res.status == 200, f"Expected 200 for policy PDF, got {res.status}"
    content_type = res.headers.get("content-type", "")
    assert "application/pdf" in content_type.lower(), f"Expected application/pdf, got {content_type}"
    body = res.body()
    assert body.startswith(b"%PDF-"), "Expected valid PDF signature"
