import hashlib

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


def test_terms_of_use_pdf_served_and_matches_pw_terms(raw_context: APIRequestContext):
    """Verify /terms-of-use.pdf is served as a valid PDF matching docker-compose PW_TERMS SHA-256."""
    res = raw_context.get("/terms-of-use.pdf")
    assert res.status == 200, f"Expected 200 for Terms PDF, got {res.status}"
    content_type = res.headers.get("content-type", "")
    assert "application/pdf" in content_type.lower(), f"Expected application/pdf, got {content_type}"
    body = res.body()
    assert body.startswith(b"%PDF-"), "Expected valid PDF signature"
    digest = hashlib.sha256(body).hexdigest()
    assert digest == "b08f3d8b1cd44544888a78ed13316c0910725944bc46808ab542e36d9b2fd84c"
