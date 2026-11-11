import pytest
from playwright.sync_api import sync_playwright

def test_cors_preflight_allowed_host(target_server: str):
    """Verify backend returns 204 No Content and CORS headers for allowed client host."""
    with sync_playwright() as playwright:
        ctx = playwright.request.new_context(base_url=target_server)
        res = ctx.fetch(
            "/api/v1/hello",
            method="OPTIONS",
            headers={
                "Host": "www.plannedwallet.com",
                "Access-Control-Request-Method": "GET",
            },
        )
        assert res.status == 204
        methods = res.headers.get("access-control-allow-methods", "")
        assert "GET" in methods
        assert res.headers.get("access-control-allow-origin") is not None
        ctx.dispose()

def test_cors_preflight_disallowed_host(target_server: str):
    """Verify backend rejects pre-flight challenge for disallowed host with 403 Forbidden."""
    with sync_playwright() as playwright:
        ctx = playwright.request.new_context(base_url=target_server)
        res = ctx.fetch(
            "/api/v1/hello",
            method="OPTIONS",
            headers={
                "Host": "evil.com",
                "Access-Control-Request-Method": "POST",
            },
        )
        assert res.status == 403
        data = res.json()
        assert data.get("error") == "CORS_FORBIDDEN"
        ctx.dispose()

def test_cors_preflight_allowed_origin(target_server: str):
    """Verify backend accepts pre-flight with allowed Origin header and reflects it."""
    with sync_playwright() as playwright:
        ctx = playwright.request.new_context(base_url=target_server)
        res = ctx.fetch(
            "/api/v1/hello",
            method="OPTIONS",
            headers={
                "Host": "local.plannedwallet.com",
                "Origin": "http://local.plannedwallet.com:3000",
                "Access-Control-Request-Method": "GET",
            },
        )
        assert res.status == 204
        assert res.headers.get("access-control-allow-origin") == "http://local.plannedwallet.com:3000"
        ctx.dispose()
