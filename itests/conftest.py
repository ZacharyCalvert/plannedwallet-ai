import os
import base64
import json
import pytest
from playwright.sync_api import sync_playwright, APIRequestContext

@pytest.fixture(scope="session")
def target_server() -> str:
    """Return the base URL of the target server from TARGET_SERVER or default."""
    server = os.getenv("TARGET_SERVER", "http://localhost:8080").rstrip("/")
    return server

@pytest.fixture(scope="session")
def api_context(target_server: str):
    """Provide a Playwright APIRequestContext bound to the target server."""
    with sync_playwright() as playwright:
        request_context = playwright.request.new_context(base_url=target_server)
        yield request_context
        request_context.dispose()

def decode_jwt_unverified(token: str) -> dict:
    """Decode JWT payload without verifying signature for assertion checks."""
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("Invalid JWT format: expected 3 dot-separated segments")
    payload_b64 = parts[1]
    # Handle base64 padding
    padding = len(payload_b64) % 4
    if padding:
        payload_b64 += "=" * (4 - padding)
    decoded_bytes = base64.urlsafe_b64decode(payload_b64)
    return json.loads(decoded_bytes.decode("utf-8"))
