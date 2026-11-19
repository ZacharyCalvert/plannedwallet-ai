#!/usr/bin/env python3
"""
Execute a SQL file against Amazon RDS PostgreSQL using an ephemeral AWS ECS
Fargate RunTask inside public subnets with no NAT Gateway.

Handles:
1. Gzip + base64 encoding the SQL file into an 8192-byte-safe ECS container override
2. Discovering the migration container name and awslogs CloudWatch log config
3. Running `aws ecs run-task` and waiting for `tasks-stopped`
4. Fetching stdout/stderr from CloudWatch Logs and optionally writing to GITHUB_STEP_SUMMARY
5. Failing fast if task launch fails, stops with errors, or container exitCode != 0
"""

import argparse
import base64
import gzip
import json
import os
import subprocess
import sys
import time

MAX_ECS_OVERRIDES_BYTES = 8192


def build_sql_shell_command(sql_bytes: bytes) -> str:
    """Compress and base64-encode SQL bytes into a POSIX /bin/sh psql pipeline."""
    compressed = gzip.compress(sql_bytes, mtime=0)
    b64 = base64.b64encode(compressed).decode("ascii")
    ensure_psql = (
        "set -e; "
        "if ! command -v psql >/dev/null 2>&1; then "
        "if command -v apk >/dev/null 2>&1; then "
        "apk add --no-cache postgresql16-client >/dev/null 2>&1 || apk add --no-cache postgresql-client; "
        "elif command -v apt-get >/dev/null 2>&1; then "
        "DEBIAN_FRONTEND=noninteractive apt-get update -qq >/dev/null && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq postgresql-client >/dev/null; "
        "elif command -v dnf >/dev/null 2>&1; then "
        "dnf install -y -q postgresql16 >/dev/null 2>&1 || dnf install -y -q postgresql; "
        "elif command -v yum >/dev/null 2>&1; then "
        "yum install -y -q postgresql16 >/dev/null 2>&1 || yum install -y -q postgresql; "
        "else echo 'psql not found and no supported package manager available' >&2; exit 127; fi; "
        "fi; "
    )
    return (
        f"{ensure_psql}"
        f"printf '%s' '{b64}' | base64 -d | gzip -d | "
        'PGPASSWORD="${DB_PASSWORD:-${POSTGRES_PASSWORD:-}}" '
        'PGSSLMODE="${DB_SSLMODE:-${POSTGRES_SSLMODE:-require}}" '
        'psql -v ON_ERROR_STOP=1 -h "${DB_HOST:-${POSTGRES_HOST:-}}" '
        '-p "${DB_PORT:-${POSTGRES_PORT:-5432}}" '
        '-U "${DB_USER:-${POSTGRES_USER:-}}" -d "${DB_NAME:-${POSTGRES_DB:-}}"'
    )


def build_ecs_overrides(container_name: str, sql_bytes: bytes) -> str:
    """Build ECS RunTask overrides JSON and validate it fits under ECS 8192-byte limit."""
    shell_cmd = build_sql_shell_command(sql_bytes)
    overrides = {
        "containerOverrides": [
            {
                "name": container_name,
                "command": ["/bin/sh", "-c", shell_cmd],
            }
        ]
    }
    encoded = json.dumps(overrides, separators=(",", ":"))
    if len(encoded.encode("utf-8")) > MAX_ECS_OVERRIDES_BYTES:
        raise ValueError(
            f"ECS overrides JSON ({len(encoded.encode('utf-8'))} bytes) exceeds "
            f"AWS ECS limit of {MAX_ECS_OVERRIDES_BYTES} bytes."
        )
    return encoded


def run_aws_cli(args: list, check: bool = True) -> subprocess.CompletedProcess:
    """Execute AWS CLI command and capture output."""
    cmd = ["aws"] + args
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if check and proc.returncode != 0:
        raise RuntimeError(f"AWS CLI failed ({' '.join(cmd[:4])}): {proc.stderr.strip()}")
    return proc


def describe_task_metadata(region: str, task_definition: str) -> dict:
    """Inspect container name and awslogs configuration from ECS task definition."""
    proc = run_aws_cli(
        [
            "ecs",
            "describe-task-definition",
            "--region",
            region,
            "--task-definition",
            task_definition,
            "--output",
            "json",
        ]
    )
    doc = json.loads(proc.stdout)
    containers = doc.get("taskDefinition", {}).get("containerDefinitions") or []
    first = containers[0] if containers else {}
    log_cfg = (first.get("logConfiguration") or {}).get("options") or {}
    return {
        "container_name": first.get("name") or "migration-runner",
        "image": first.get("image") or "",
        "log_group": log_cfg.get("awslogs-group", ""),
        "log_prefix": log_cfg.get("awslogs-stream-prefix", ""),
    }


def format_network_config(subnets: str, security_group: str) -> str:
    """Build ECS awsvpcConfiguration string from comma-separated subnets and SG."""
    clean_subnets = ",".join(s.strip() for s in subnets.split(",") if s.strip())
    clean_sg = security_group.strip()
    if not clean_subnets or not clean_sg:
        raise ValueError("MIGRATION_SUBNETS and MIGRATION_SG must be non-empty.")
    return (
        f"awsvpcConfiguration={{subnets=[{clean_subnets}],"
        f"securityGroups=[{clean_sg}],assignPublicIp=ENABLED}}"
    )


def fetch_cloudwatch_logs(region: str, log_group: str, log_stream: str) -> str:
    """Fetch CloudWatch log events with brief retry to allow ECS log flush."""
    if not log_group or not log_stream:
        return ""
    for attempt in range(4):
        proc = run_aws_cli(
            [
                "logs",
                "get-log-events",
                "--region",
                region,
                "--log-group-name",
                log_group,
                "--log-stream-name",
                log_stream,
                "--start-from-head",
                "--output",
                "json",
            ],
            check=False,
        )
        if proc.returncode == 0:
            events = json.loads(proc.stdout).get("events") or []
            if events or attempt == 3:
                return "\n".join(e.get("message", "") for e in events)
        time.sleep(2)
    return ""


def write_step_summary(title: str, logs_text: str):
    """Append formatted execution output to GITHUB_STEP_SUMMARY if configured."""
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY", "").strip()
    if not title or not summary_path:
        return
    with open(summary_path, "a", encoding="utf-8") as f:
        f.write(f"## {title}\n\n```text\n{logs_text or '(no log output)'}\n```\n")


def main():
    parser = argparse.ArgumentParser(description="Run SQL via AWS ECS Fargate RunTask")
    parser.add_argument("--sql-file", required=True, help="Path to SQL file to execute")
    parser.add_argument("--region", default=os.environ.get("AWS_REGION", "us-east-2"))
    parser.add_argument("--cluster", default=os.environ.get("ECS_CLUSTER", ""))
    parser.add_argument(
        "--task-definition",
        default=os.environ.get("ECS_MIGRATION_TASK_DEFINITION") or os.environ.get("ECS_TASK_DEFINITION", ""),
    )
    parser.add_argument("--subnets", default=os.environ.get("MIGRATION_SUBNETS", ""))
    parser.add_argument("--security-group", default=os.environ.get("MIGRATION_SG", ""))
    parser.add_argument("--summary-title", default="", help="Optional GitHub Step Summary heading")
    args = parser.parse_args()

    if not args.cluster or not args.task_definition:
        print("::error::ECS_CLUSTER and ECS_MIGRATION_TASK_DEFINITION must be set.")
        sys.exit(1)

    with open(args.sql_file, "rb") as f:
        sql_bytes = f.read()

    meta = describe_task_metadata(args.region, args.task_definition)
    container_name = meta["container_name"]
    overrides_json = build_ecs_overrides(container_name, sql_bytes)
    net_cfg = format_network_config(args.subnets, args.security_group)

    print(
        f"Running SQL '{args.sql_file}' on ECS cluster '{args.cluster}' "
        f"(container '{container_name}', image '{meta['image']}')..."
    )
    run_proc = run_aws_cli(
        [
            "ecs",
            "run-task",
            "--region",
            args.region,
            "--cluster",
            args.cluster,
            "--task-definition",
            args.task_definition,
            "--launch-type",
            "FARGATE",
            "--network-configuration",
            net_cfg,
            "--overrides",
            overrides_json,
            "--output",
            "json",
        ]
    )
    run_doc = json.loads(run_proc.stdout)
    failures = run_doc.get("failures") or []
    tasks = run_doc.get("tasks") or []
    if failures or not tasks:
        print(f"::error::Failed to launch ECS migration task: {json.dumps(failures)}")
        sys.exit(1)

    task_arn = tasks[0]["taskArn"]
    task_id = task_arn.rsplit("/", 1)[-1]
    print(f"ECS task started: {task_arn}")
    print("Waiting for ECS task to stop...")
    run_aws_cli(["ecs", "wait", "tasks-stopped", "--region", args.region, "--cluster", args.cluster, "--tasks", task_arn])

    desc_proc = run_aws_cli(
        [
            "ecs",
            "describe-tasks",
            "--region",
            args.region,
            "--cluster",
            args.cluster,
            "--tasks",
            task_arn,
            "--output",
            "json",
        ]
    )
    desc_task = (json.loads(desc_proc.stdout).get("tasks") or [{}])[0]
    container_result = next(
        (c for c in (desc_task.get("containers") or []) if c.get("name") == container_name),
        (desc_task.get("containers") or [{}])[0],
    )
    exit_code = container_result.get("exitCode")
    stopped_reason = desc_task.get("stoppedReason", "")

    log_stream = f"{meta['log_prefix']}/{container_name}/{task_id}" if meta["log_prefix"] else ""
    logs_text = fetch_cloudwatch_logs(args.region, meta["log_group"], log_stream)
    if logs_text:
        print("--- Container Execution Logs ---")
        print(logs_text)
        print("--- End Container Execution Logs ---")

    if args.summary_title and exit_code == 0:
        write_step_summary(args.summary_title, logs_text)

    if exit_code != 0:
        print(f"::error::ECS SQL task failed with exitCode={exit_code} stoppedReason={stopped_reason!r}")
        sys.exit(1)

    print("ECS SQL task completed successfully (exitCode=0).")


if __name__ == "__main__":
    main()
