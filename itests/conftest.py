import os
import base64
import hashlib
import hmac
import json
import struct
import time
import pytest
from playwright.sync_api import sync_playwright, APIRequestContext

@pytest.fixture(scope="session")
def target_server() -> str:
    """Return the base URL of the target server from TARGET_SERVER or default."""
    server = os.getenv("TARGET_SERVER", "http://localhost:8080").rstrip("/")
    return server

@pytest.fixture(scope="session")
def playwright_instance():
    """Provide a session-scoped Playwright instance."""
    with sync_playwright() as playwright:
        yield playwright

@pytest.fixture(scope="session")
def api_context(playwright_instance, target_server: str):
    """Provide a Playwright APIRequestContext bound to the target server."""
    request_context = playwright_instance.request.new_context(
        base_url=target_server,
        extra_http_headers={"X-Client-ID": "plawrite-itest"},
    )
    yield request_context
    request_context.dispose()

@pytest.fixture
def raw_context(playwright_instance, target_server: str):
    """Provide a fresh Playwright APIRequestContext without pre-configured headers."""
    ctx = playwright_instance.request.new_context(base_url=target_server)
    yield ctx
    ctx.dispose()

def decode_jwt_unverified(token: str) -> dict:
    """Decode JWT payload without verifying signature for assertion checks."""
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("Invalid JWT format: expected 3 dot-separated segments")
    payload_b64 = parts[1]
    padding = len(payload_b64) % 4
    if padding:
        payload_b64 += "=" * (4 - padding)
    decoded_bytes = base64.urlsafe_b64decode(payload_b64)
    return json.loads(decoded_bytes.decode("utf-8"))

def compute_totp_code(secret: str) -> str:
    """Compute current RFC 6238 TOTP 6-digit code from base32 secret."""
    clean_secret = secret.strip().upper()
    padding = len(clean_secret) % 8
    if padding:
        clean_secret += "=" * (8 - padding)
    key = base64.b32decode(clean_secret)
    counter = int(time.time()) // 30
    msg = struct.pack(">Q", counter)
    h = hmac.new(key, msg, hashlib.sha1).digest()
    offset = h[-1] & 0x0F
    code = struct.unpack(">I", h[offset : offset + 4])[0] & 0x7FFFFFFF
    return f"{code % 1000000:06d}"

