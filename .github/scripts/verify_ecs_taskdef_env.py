#!/usr/bin/env python3
"""
Safeguard script to inspect an AWS ECS Fargate backend task definition JSON
(from `aws ecs describe-task-definition ... --query "taskDefinition"`) before
rendering and deploying a new ECR image revision.

Mandatory variables (in `environment` or `secrets` on container `backend`):
- REQUIRE_CAPTCHA
- GIN_MODE
- JWT_SECRET
- PRODUCTION_ENV
- POSTGRES_HOST (or DB_HOST)
- POSTGRES_DB (or DB_NAME)
- EXPECTED_ORIGIN_SECRET

SECURITY NOTE:
This repository is public. All secrets, SSM/SecretsManager ARNs, database hosts,
database names, credentials, and origin verification secrets are strictly masked
in console output to prevent credential/infrastructure leakage in public logs.
"""

import argparse
import json
import sys

SENSITIVE_VARS = {
    "JWT_SECRET",
    "POSTGRES_HOST",
    "DB_HOST",
    "POSTGRES_DB",
    "DB_NAME",
    "EXPECTED_ORIGIN_SECRET",
    "POSTGRES_PASSWORD",
    "DB_PASSWORD",
    "POSTGRES_USER",
    "DB_USER",
}

HOST_DB_ALIASES = {
    "POSTGRES_HOST": ("POSTGRES_HOST", "DB_HOST"),
    "POSTGRES_DB": ("POSTGRES_DB", "DB_NAME"),
    "POSTGRES_SSLMODE": ("POSTGRES_SSLMODE", "DB_SSLMODE"),
}


def is_sensitive(var_name: str) -> bool:
    """Return True if var_name contains secrets or sensitive infra values."""
    upper = var_name.upper()
    if upper in SENSITIVE_VARS:
        return True
    for kw in ("SECRET", "PASSWORD", "KEY", "TOKEN", "HOST", "ORIGIN", "CREDENTIAL"):
        if kw in upper:
            return True
    return False


def mask_value(var_name: str, val: object) -> str:
    """Mask sensitive values for safe CI logging."""
    if is_sensitive(var_name):
        return "********"
    return f"'{val}'"


def parse_ecs_taskdef_env(task_def_doc: dict, container_name: str = "backend") -> dict:
    """Extract combined environment + secrets map from ECS task definition JSON."""
    task_def = task_def_doc.get("taskDefinition", task_def_doc)
    containers = task_def.get("containerDefinitions") or []
    if not containers:
        raise ValueError("No containerDefinitions found in ECS task definition JSON.")

    target = next((c for c in containers if c.get("name") == container_name), None)
    if target is None:
        names = [c.get("name", "<unnamed>") for c in containers]
        raise ValueError(f"Container '{container_name}' not found in task definition (found: {names}).")

    env_map = {}
    for item in target.get("environment") or []:
        name = item.get("name")
        if name:
            env_map[name] = {"value": item.get("value"), "is_secret": False}
    for item in target.get("secrets") or []:
        name = item.get("name")
        if name:
            env_map[name] = {"value": item.get("valueFrom"), "is_secret": True}
    return env_map


def find_entry(env_vars: dict, canonical_name: str):
    """Look up canonical variable name or supported AWS DB_* alias."""
    candidates = HOST_DB_ALIASES.get(canonical_name, (canonical_name,))
    for candidate in candidates:
        if candidate in env_vars:
            return candidate, env_vars[candidate]
    return canonical_name, None


def is_valid_sha256_hex(raw: str) -> bool:
    """Validate a 64-character lowercase/uppercase hexadecimal digest."""
    return len(raw) == 64 and all(c in "0123456789abcdefABCDEF" for c in raw)


def validate_entry_value(name: str, val_str: str, is_secret: bool, target_env: str, errors: list, warnings: list):
    """Validate non-secret configuration values without logging sensitive content."""
    if is_secret:
        return
    if name == "JWT_SECRET":
        if len(val_str) < 32:
            errors.append("JWT_SECRET value must be at least 32 characters (256 bits)")
        if "super-secret" in val_str.lower() or "super secret" in val_str.lower():
            errors.append("JWT_SECRET must not use insecure default 'super-secret' values")
    if name == "GIN_MODE" and val_str.lower() != "release":
        warnings.append(f"GIN_MODE is set to '{val_str}', recommended 'release' for {target_env}")
    if name == "REQUIRE_CAPTCHA" and val_str.lower() != "true":
        warnings.append(f"REQUIRE_CAPTCHA is set to '{val_str}', expected 'true'")
    if name == "PRODUCTION_ENV" and target_env in ("prod", "production") and val_str.lower() != "true":
        errors.append(f"PRODUCTION_ENV must be 'true' for production deployment (currently '{val_str}')")


def validate_production_guards(env_vars: dict, target_env: str, errors: list):
    """Enforce ENFORCE_POLICY=true, valid PW_TERMS, and RDS SSL mode when in prod."""
    prod_entry = env_vars.get("PRODUCTION_ENV", {})
    prod_is_true = str(prod_entry.get("value") or "").strip().lower() == "true"
    if not (prod_is_true or target_env in ("prod", "production")):
        return

    ep_entry = env_vars.get("ENFORCE_POLICY")
    ep_val = str((ep_entry or {}).get("value") or "").strip().lower()
    if not ep_entry:
        errors.append("Missing mandatory environment variable when PRODUCTION_ENV=true: 'ENFORCE_POLICY'")
        print("[-] ENFORCE_POLICY      : MISSING (Must be 'true' in production)")
    elif not ep_entry.get("is_secret") and ep_val != "true":
        errors.append(f"ENFORCE_POLICY must be 'true' when PRODUCTION_ENV is true (currently '{ep_entry.get('value')}')")
        print(f"[-] ENFORCE_POLICY      : INVALID ('{ep_entry.get('value')}', must be 'true')")
    else:
        print("[+] ENFORCE_POLICY      : CONFIGURED (value: 'true')")

    for policy_env in ("PW_TERMS", "PW_PRIVACY"):
        entry = env_vars.get(policy_env)
        val = str((entry or {}).get("value") or "").strip()
        if not entry or not val:
            errors.append(f"Missing mandatory environment variable when PRODUCTION_ENV=true: '{policy_env}'")
            print(f"[-] {policy_env:20s}: MISSING (SHA-256 hex digest required in production)")
        elif not entry.get("is_secret") and not is_valid_sha256_hex(val):
            errors.append(f"{policy_env} must be a valid 64-character hexadecimal SHA-256 digest")
            print(f"[-] {policy_env:20s}: INVALID (Must be 64-char SHA-256 hex)")
        else:
            print(f"[+] {policy_env:20s}: CONFIGURED")

    ssl_key, ssl_entry = find_entry(env_vars, "POSTGRES_SSLMODE")
    ssl_val = str((ssl_entry or {}).get("value") or "").strip().lower()
    if not ssl_entry or (not ssl_entry.get("is_secret") and ssl_val not in ("require", "verify-ca", "verify-full")):
        errors.append(f"{ssl_key} must be 'require', 'verify-ca', or 'verify-full' for RDS TLS enforcement")
        print(f"[-] {ssl_key:20s}: INVALID/MISSING (Must enforce RDS SSL)")
    else:
        print(f"[+] {ssl_key:20s}: CONFIGURED ({mask_value(ssl_key, ssl_val)})")


def verify_task_definition(task_def_doc: dict, target_env: str, container_name: str = "backend") -> tuple:
    """Validate ECS backend task definition environment; returns (errors, warnings)."""
    env_vars = parse_ecs_taskdef_env(task_def_doc, container_name=container_name)
    required_vars = [
        "REQUIRE_CAPTCHA",
        "GIN_MODE",
        "JWT_SECRET",
        "PRODUCTION_ENV",
        "POSTGRES_HOST",
        "POSTGRES_DB",
        "EXPECTED_ORIGIN_SECRET",
    ]
    errors, warnings = [], []

    print("=================================================================")
    print(f"ECS Fargate Task Definition Safeguard Check [{target_env.upper()}]")
    print("=================================================================")

    for var in required_vars:
        found_key, entry = find_entry(env_vars, var)
        if not entry:
            errors.append(f"Missing mandatory environment variable: '{var}'")
            print(f"[-] {var:22s}: MISSING (Not defined)")
            continue
        val_str = str(entry.get("value") or "").strip()
        if not val_str:
            errors.append(f"Variable '{found_key}' is declared with an empty value")
            print(f"[-] {found_key:22s}: EMPTY")
            continue
        src = "secrets (SSM/SecretsManager ARN)" if entry["is_secret"] else f"value: {mask_value(found_key, val_str)}"
        print(f"[+] {found_key:22s}: CONFIGURED ({src})")
        validate_entry_value(var, val_str, entry["is_secret"], target_env, errors, warnings)

    validate_production_guards(env_vars, target_env, errors)
    print("=================================================================")
    return errors, warnings


def main():
    parser = argparse.ArgumentParser(description="Verify AWS ECS task definition JSON")
    parser.add_argument("task_def_path", help="Path to ECS task definition JSON file")
    parser.add_argument("target_env", nargs="?", default="stage", help="Target environment (stage or prod)")
    parser.add_argument("--container-name", default="backend", help="Target container name in containerDefinitions")
    args = parser.parse_args()

    try:
        with open(args.task_def_path, "r", encoding="utf-8") as f:
            task_def_doc = json.load(f)
    except Exception as exc:
        print(f"ERROR: Failed to read ECS task definition JSON '{args.task_def_path}': {exc}")
        sys.exit(1)

    try:
        errors, warnings = verify_task_definition(task_def_doc, args.target_env.lower(), container_name=args.container_name)
    except ValueError as exc:
        print(f"ERROR: {exc}")
        sys.exit(1)

    if warnings:
        print("\nWarnings:")
        for w in warnings:
            print(f"  * WARNING: {w}")

    if errors:
        print("\nSafeguard Verification FAILED with errors:")
        for err in errors:
            print(f"  * ERROR: {err}")
        sys.exit(1)

    print("\nSafeguard Verification PASSED: All mandatory ECS task definition configuration verified.")


if __name__ == "__main__":
    main()
