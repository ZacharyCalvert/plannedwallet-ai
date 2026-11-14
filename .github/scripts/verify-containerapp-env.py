#!/usr/bin/env python3
"""
Safeguard script to inspect that an Azure Container App has expected environment
variables configured before deploying a new revision.

Mandatory variables:
- REQUIRE_CAPTCHA
- GIN_MODE
- JWT_SECRET
- PRODUCTION_ENV
- POSTGRES_HOST
- POSTGRES_DB
- EXPECTED_FDID

SECURITY NOTE:
This repository is public. All secrets, database connection details (POSTGRES_HOST,
POSTGRES_DB), credentials, and proxy identifiers (EXPECTED_FDID) are strictly
masked in console output to prevent credential/infrastructure leakage in public logs.
"""

import json
import sys

# Set of sensitive variables that must never have raw values printed to stdout
SENSITIVE_VARS = {
    "JWT_SECRET",
    "POSTGRES_HOST",
    "POSTGRES_DB",
    "EXPECTED_FDID",
    "POSTGRES_PASSWORD",
    "POSTGRES_USER",
}


def is_sensitive(var_name):
    """Check if a variable name corresponds to a sensitive secret or infrastructure detail."""
    upper = var_name.upper()
    if upper in SENSITIVE_VARS:
        return True
    for kw in ["SECRET", "PASSWORD", "KEY", "TOKEN", "HOST", "FDID", "CREDENTIAL"]:
        if kw in upper:
            return True
    return False


def parse_container_env(app_config):
    """Extract environment variable map across all containers in the template."""
    template = app_config.get("properties", {}).get("template", {})
    if not template:
        template = app_config.get("template", {})

    containers = template.get("containers", [])
    if not containers:
        containers = app_config.get("containers", [])

    if not containers:
        print("ERROR: No containers found in Container App configuration JSON.")
        sys.exit(1)

    env_map = {}
    for c in containers:
        container_name = c.get("name", "app")
        for item in c.get("env") or []:
            name = item.get("name")
            if name:
                env_map[name] = {
                    "container": container_name,
                    "value": item.get("value"),
                    "secretRef": item.get("secretRef"),
                }
    return env_map


def main():
    if len(sys.argv) < 2:
        print("Usage: verify-containerapp-env.py <config_json_path> [target_env]")
        sys.exit(1)

    config_path = sys.argv[1]
    target_env = sys.argv[2] if len(sys.argv) > 2 else "stage"
    target_env = target_env.lower()

    try:
        with open(config_path, "r", encoding="utf-8") as f:
            app_config = json.load(f)
    except Exception as exc:
        print(f"ERROR: Failed to read or parse Container App JSON from '{config_path}': {exc}")
        sys.exit(1)

    env_vars = parse_container_env(app_config)

    required_vars = [
        "REQUIRE_CAPTCHA",
        "GIN_MODE",
        "JWT_SECRET",
        "PRODUCTION_ENV",
        "POSTGRES_HOST",
        "POSTGRES_DB",
        "EXPECTED_FDID",
    ]

    print("=================================================================")
    print(f"Container App Environment Safeguard Check [{target_env.upper()}]")
    print("=================================================================")

    errors = []
    warnings = []

    for var in required_vars:
        if var not in env_vars:
            errors.append(f"Missing mandatory environment variable: '{var}'")
            print(f"[-] {var:20s}: MISSING (Not defined)")
            continue

        entry = env_vars[var]
        val = entry.get("value")
        secret_ref = entry.get("secretRef")

        if (val is None or str(val).strip() == "") and not secret_ref:
            errors.append(f"Variable '{var}' is declared but has neither 'value' nor 'secretRef'")
            print(f"[-] {var:20s}: EMPTY (Neither value nor secretRef set)")
            continue

        if secret_ref:
            print(f"[+] {var:20s}: CONFIGURED (via secretRef='{secret_ref}')")
        else:
            display_val = "********" if is_sensitive(var) else f"'{val}'"
            print(f"[+] {var:20s}: CONFIGURED (value: {display_val})")

            # Value validation checks (never printing sensitive contents)
            if var == "JWT_SECRET":
                if len(str(val)) < 32:
                    errors.append("JWT_SECRET value must be at least 32 characters (256 bits)")
                if "super-secret" in str(val).lower():
                    errors.append("JWT_SECRET must not use insecure default 'super-secret' values")

            if var == "GIN_MODE" and str(val).lower() != "release":
                warnings.append(f"GIN_MODE is set to '{val}', recommended 'release' for {target_env}")

            if var == "REQUIRE_CAPTCHA" and str(val).lower() != "true":
                warnings.append(f"REQUIRE_CAPTCHA is set to '{val}', expected 'true'")

            if var == "PRODUCTION_ENV" and target_env in ["prod", "production"] and str(val).lower() != "true":
                errors.append(f"PRODUCTION_ENV must be 'true' for production deployment (currently '{val}')")

    # Informational checks for additional optional configuration
    optional_vars = [
        "POSTGRES_USER",
        "POSTGRES_PASSWORD",
        "POSTGRES_PORT",
        "POSTGRES_SSLMODE",
        "CORS_ALLOWED_CLIENTS",
    ]
    print("-----------------------------------------------------------------")
    print("Additional Configuration Checks:")
    for ovar in optional_vars:
        if ovar in env_vars:
            entry = env_vars[ovar]
            ref = entry.get("secretRef")
            v = entry.get("value")
            if ref:
                disp = f"secretRef='{ref}'"
            else:
                disp_val = "********" if is_sensitive(ovar) else f"'{v}'"
                disp = f"value: {disp_val}"
            print(f"[i] {ovar:20s}: CONFIGURED ({disp})")
        else:
            print(f"[i] {ovar:20s}: NOT SET (Optional / default applied)")

    print("=================================================================")

    if warnings:
        print("\nWarnings:")
        for w in warnings:
            print(f"  * WARNING: {w}")

    if errors:
        print("\nSafeguard Verification FAILED with errors:")
        for err in errors:
            print(f"  * ERROR: {err}")
        sys.exit(1)

    print("\nSafeguard Verification PASSED: All mandatory environment variables are verified.")


if __name__ == "__main__":
    main()
