import uuid
import pytest
from playwright.sync_api import APIRequestContext
from conftest import decode_jwt_unverified

TEST_PASSWORD = "ComplexPassword123!"

def generate_user_data(prefix: str = "itest"):
    """Helper to generate unique user registration data."""
    unique_id = uuid.uuid4().hex[:8]
    return {
        "username": f"{prefix}_{unique_id}",
        "email": f"{prefix}_{unique_id}@example.com",
        "password": TEST_PASSWORD,
    }

def test_create_user(api_context: APIRequestContext):
    """Test 1: Create a user with a username and test email."""
    user_data = generate_user_data()
    response = api_context.post("/api/v1/users", data=user_data)

    assert response.status == 201, f"Expected 201, got {response.status}: {response.text()}"
    data = response.json()
    assert "id" in data and len(data["id"]) > 0
    assert data["username"] == user_data["username"]
    assert data["email"] == user_data["email"]
    assert "created_at" in data

def test_login_and_validate_jwt(api_context: APIRequestContext):
    """Test 2: Login as the user and validate the JWT response."""
    user_data = generate_user_data()
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201
    user_id = create_res.json()["id"]

    login_res = api_context.post(
        f"/api/v1/users/{user_data['username']}/token",
        data={"password": user_data["password"]},
    )
    assert login_res.status == 200, f"Login failed: {login_res.text()}"
    token_data = login_res.json()

    assert "access_token" in token_data and len(token_data["access_token"]) > 0
    assert token_data.get("token_type") == "Bearer"
    assert token_data.get("expires_in") == 3600

    payload = decode_jwt_unverified(token_data["access_token"])
    assert payload.get("sub") == user_id
    assert payload.get("username") == user_data["username"]
    assert payload.get("exp", 0) > payload.get("iat", 0)

def test_create_duplicate_username_fails(api_context: APIRequestContext):
    """Test 3: Attempt to create same username with different email and verify 409 conflict."""
    user_data = generate_user_data()
    create_res = api_context.post("/api/v1/users", data=user_data)
    assert create_res.status == 201

    diff_email = f"diff_{uuid.uuid4().hex[:8]}@example.com"
    duplicate_user_data = {
        "username": user_data["username"],
        "email": diff_email,
        "password": TEST_PASSWORD,
    }

    dup_res = api_context.post("/api/v1/users", data=duplicate_user_data)
    assert dup_res.status == 409, f"Expected 409 conflict, got {dup_res.status}: {dup_res.text()}"
    err_data = dup_res.json()
    assert err_data.get("error") == "USER_CONFLICT"
