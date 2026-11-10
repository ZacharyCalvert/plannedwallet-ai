# Integration Tests (`itests`)

Integration tests for Planned Wallet API built with Python, Playwright, and Pytest.

## Overview
These tests validate end-to-end user workflows against a running instance of the backend API service (local dev or CI/CD clean launch).

Current test coverage:
1. **User Registration**: Creates a user with unique credentials and validates response envelope.
2. **User Authentication & JWT Validation**: Performs token exchange via `/api/v1/users/{username}/token`, validates status 200, JWT format, and unverified claims (`sub`, `username`, expiration).
3. **Username Conflict Enforcement**: Verifies that registering an existing username with a different email fails with HTTP 409 (`USER_CONFLICT`).

## Requirements
- Python 3.9+
- Running instance of `backend-api`

## Running Tests

### 1. Default (http://localhost:8080)
```bash
make test
```

### 2. Custom Target Server
```bash
make test TARGET_SERVER=http://localhost:8080
# or via environment variable
TARGET_SERVER=http://127.0.0.1:8080 make test
```

### 3. Clean environment
```bash
make clean
```
