import base64
import uuid
import pytest
from playwright.sync_api import APIRequestContext

PNG_HEADER = b"\x89PNG\r\n\x1a\n"

def test_get_captcha(api_context: APIRequestContext):
    """Verify that GET /api/v1/captcha generates a valid challenge and data URI image."""
    res = api_context.get("/api/v1/captcha")
    assert res.status == 200, f"Expected 200, got {res.status}: {res.text()}"

    data = res.json()
    assert "id" in data and len(data["id"]) > 0, "Expected non-empty captcha ID"
    assert "image" in data, "Expected image field in response"

    image_uri = data["image"]
    assert image_uri.startswith("data:image/png;base64,"), "Expected base64 PNG data URI prefix"

    raw_b64 = image_uri.split("data:image/png;base64,")[1]
    decoded_bytes = base64.b64decode(raw_b64)
    assert decoded_bytes.startswith(PNG_HEADER), "Decoded bytes do not match PNG file signature"

def test_get_captcha_image_direct(api_context: APIRequestContext):
    """Verify direct image streaming via GET /api/v1/captcha/{captchaId}.png."""
    # 1. Generate captcha
    gen_res = api_context.get("/api/v1/captcha")
    assert gen_res.status == 200
    captcha_id = gen_res.json()["id"]

    # 2. Fetch direct image
    img_res = api_context.get(f"/api/v1/captcha/{captcha_id}.png")
    assert img_res.status == 200, f"Expected 200 for direct image, got {img_res.status}"
    assert "image/png" in img_res.headers.get("content-type", "")
    assert img_res.body().startswith(PNG_HEADER), "Expected PNG bytes in response body"

    # 3. Nonexistent captcha returns 404
    err_res = api_context.get("/api/v1/captcha/nonexistent_challenge_id.png")
    assert err_res.status == 404, f"Expected 404 for invalid captcha ID, got {err_res.status}"
    assert err_res.json().get("error") == "CAPTCHA_NOT_FOUND"

def test_register_with_invalid_captcha(api_context: APIRequestContext):
    """Verify that submitting an invalid CAPTCHA solution returns 400 INVALID_CAPTCHA."""
    gen_res = api_context.get("/api/v1/captcha")
    assert gen_res.status == 200
    captcha_id = gen_res.json()["id"]

    uid = uuid.uuid4().hex[:8]
    payload = {
        "username": f"bot_{uid}",
        "email": f"bot_{uid}@example.com",
        "password": "ValidPassword123!",
        "captcha_id": captcha_id,
        "captcha_solution": "000000",
    }
    reg_res = api_context.post("/api/v1/users", data=payload)
    assert reg_res.status == 400, f"Expected 400 on invalid captcha, got {reg_res.status}: {reg_res.text()}"
    err_data = reg_res.json()
    assert err_data.get("error") == "INVALID_CAPTCHA", f"Expected INVALID_CAPTCHA, got: {err_data}"
