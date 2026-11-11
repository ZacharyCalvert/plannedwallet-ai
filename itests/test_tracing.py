import re
import pytest
from playwright.sync_api import APIRequestContext

TRACEPARENT_REGEX = re.compile(r"^([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$")

def test_w3c_trace_context_generated_when_absent(api_context: APIRequestContext):
    """Verify backend generates valid W3C traceparent header when client provides none."""
    res = api_context.get("/api/v1/hello")
    assert res.status == 200
    traceparent = res.headers.get("traceparent")
    assert traceparent is not None, "traceparent header missing from response"
    m = TRACEPARENT_REGEX.match(traceparent)
    assert m is not None, f"invalid traceparent format: {traceparent}"
    version, trace_id, span_id, flags = m.groups()
    assert version == "00"
    assert trace_id != "0" * 32
    assert span_id != "0" * 16

def test_w3c_trace_context_propagated_when_provided(api_context: APIRequestContext):
    """Verify backend preserves trace_id and tracestate from incoming traceparent."""
    custom_trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
    incoming_parent = f"00-{custom_trace_id}-00f067aa0ba902b7-01"
    headers = {
        "traceparent": incoming_parent,
        "tracestate": "congo=t61rcWkgMzE",
    }
    res = api_context.get("/api/v1/hello", headers=headers)
    assert res.status == 200
    traceparent = res.headers.get("traceparent")
    assert traceparent is not None
    m = TRACEPARENT_REGEX.match(traceparent)
    assert m is not None
    assert m.group(2) == custom_trace_id, f"trace_id mismatch: expected {custom_trace_id}, got {m.group(2)}"
    assert res.headers.get("tracestate") == "congo=t61rcWkgMzE"

def test_w3c_trace_context_on_error_response(api_context: APIRequestContext):
    """Verify error responses preserve trace context and return correlated traceparent."""
    custom_trace_id = "a1b2c3d4e5f60718293a4b5c6d7e8f90"
    incoming_parent = f"00-{custom_trace_id}-1122334455667788-01"
    res = api_context.post(
        "/api/v1/users/nonexistent_user/token",
        data={"password": "any", "totp_code": "000000"},
        headers={"traceparent": incoming_parent},
    )
    assert res.status == 404 or res.status == 401
    traceparent = res.headers.get("traceparent")
    assert traceparent is not None
    m = TRACEPARENT_REGEX.match(traceparent)
    assert m is not None
    assert m.group(2) == custom_trace_id

def test_missing_x_client_id_rejected(raw_context: APIRequestContext):
    """Verify backend rejects requests without X-Client-ID header with 400 Bad Request."""
    res = raw_context.get("/api/v1/hello")
    assert res.status == 400
    data = res.json()
    assert data.get("error") == "MISSING_CLIENT_ID"
    assert res.headers.get("traceparent") is not None
