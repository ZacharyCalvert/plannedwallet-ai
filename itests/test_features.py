import hashlib
import os

from conftest import load_frontend_pdf_bytes
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
    body = load_frontend_pdf_bytes(raw_context, "privacy-and-security-policy.pdf")
    assert body.startswith(b"%PDF-"), "Expected valid PDF signature"


def test_terms_of_use_pdf_served_and_matches_pw_terms(raw_context: APIRequestContext):
    """Verify /terms-of-use.pdf is served as a valid PDF matching docker-compose PW_TERMS SHA-256."""
    body = load_frontend_pdf_bytes(raw_context, "terms-of-use.pdf")
    assert body.startswith(b"%PDF-"), "Expected valid PDF signature"
    digest = hashlib.sha256(body).hexdigest()
    assert digest == "4906a573260b73ad9c9c5b95c7492cca669ce4354e91a7ba4383a955d85bfb80"


def test_privacy_policy_pdf_served_and_matches_pw_privacy(raw_context: APIRequestContext):
    """Verify /privacy-and-security-policy.pdf is served as a valid PDF matching docker-compose PW_PRIVACY SHA-256."""
    body = load_frontend_pdf_bytes(raw_context, "privacy-and-security-policy.pdf")
    assert body.startswith(b"%PDF-"), "Expected valid PDF signature"
    digest = hashlib.sha256(body).hexdigest()
    assert digest == "a731e26873a4229a02f875965c59a4f7b5de6c029b7e24de6907ed5b9aa6a1a7"


def test_get_version_endpoint(raw_context: APIRequestContext):
    """Verify GET /api/v1/version returns stamped build provenance metadata without caching."""
    res = raw_context.get(
        "/api/v1/version",
        headers={"X-Client-ID": "itest-version"},
    )
    assert res.status == 200, f"Unexpected status: {res.status} {res.text()}"
    assert res.headers.get("cache-control") == "no-store"
    body = res.json()
    for field in ("github_run", "build_date", "git_sha"):
        assert isinstance(body.get(field), str) and body[field].strip() != "", f"Missing {field} in {body}"
    expected_sha = os.getenv("GITHUB_SHA", "").strip()
    if expected_sha:
        assert body["git_sha"] == expected_sha
