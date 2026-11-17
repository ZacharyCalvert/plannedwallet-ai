#!/usr/bin/env python3
"""Enforce a minimum publication age (default 5 days) on adopted/updated PR dependencies.

Covers:
  - NPM packages in frontend-ui/package-lock.json
  - Go modules in backend-api/go.mod
  - Docker images in Dockerfiles and docker-compose.yml

Only dependencies added or modified relative to --base-ref (if provided) are evaluated,
allowing existing pinned dependencies to pass while blocking freshly published updates.
"""

from __future__ import annotations

import argparse
import dataclasses
import datetime as dt
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Callable, Dict, Iterable, List, Optional, Set, Tuple


DOCKER_FILES = [
    "backend-api/Dockerfile",
    "frontend-ui/Dockerfile",
    "nginx-gateway/Dockerfile",
    "docker-compose.yml",
]
GO_MOD_FILE = "backend-api/go.mod"
NPM_LOCK_FILE = "frontend-ui/package-lock.json"


@dataclasses.dataclass(frozen=True)
class DependencyCheckResult:
    ecosystem: str
    name: str
    version: str
    published_at: Optional[dt.datetime]
    age_days: Optional[float]
    passed: bool
    reason: str


def parse_iso_timestamp(raw: str) -> dt.datetime:
    clean = raw.strip()
    if clean.endswith("Z"):
        clean = clean[:-1] + "+00:00"
    if "." in clean:
        head, rest = clean.split(".", 1)
        tz_idx = max(rest.find("+"), rest.find("-"))
        if tz_idx != -1:
            frac = rest[:tz_idx][:6]
            tz = rest[tz_idx:]
            clean = f"{head}.{frac}{tz}"
        else:
            clean = f"{head}.{rest[:6]}"
    parsed = dt.datetime.fromisoformat(clean)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=dt.timezone.utc)
    return parsed.astimezone(dt.timezone.utc)


def git_read_file(ref: str, path: str) -> Optional[str]:
    if not ref:
        return None
    proc = subprocess.run(
        ["git", "show", f"{ref}:{path}"],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        return None
    return proc.stdout


def disk_read_file(path: str) -> Optional[str]:
    if not os.path.exists(path):
        return None
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


def http_get_json(url: str, headers: Optional[Dict[str, str]] = None, timeout: int = 20) -> Tuple[int, Dict[str, str], dict]:
    req_headers = {"User-Agent": "plannedwallet-pr-dependency-gate/1.0"}
    if headers:
        req_headers.update(headers)
    req = urllib.request.Request(url, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            body = resp.read().decode("utf-8")
            resp_headers = {k.lower(): v for k, v in resp.headers.items()}
            return resp.status, resp_headers, json.loads(body) if body else {}
    except urllib.error.HTTPError as exc:
        resp_headers = {k.lower(): v for k, v in exc.headers.items()} if exc.headers else {}
        return exc.code, resp_headers, {}


def escape_go_path(path: str) -> str:
    out: List[str] = []
    for ch in path:
        if "A" <= ch <= "Z":
            out.append("!" + ch.lower())
        else:
            out.append(ch)
    return "".join(out)


def parse_go_mod(content: Optional[str]) -> Dict[str, str]:
    deps: Dict[str, str] = {}
    if not content:
        return deps
    in_require = False
    for raw_line in content.splitlines():
        line = raw_line.split("//", 1)[0].strip()
        if not line:
            continue
        if line == "require (":
            in_require = True
            continue
        if in_require and line == ")":
            in_require = False
            continue
        if line.startswith("require "):
            rest = line[len("require "):].strip()
            if rest.startswith("("):
                in_require = True
                continue
            parts = rest.split()
            if len(parts) >= 2:
                deps[parts[0]] = parts[1]
            continue
        if in_require:
            parts = line.split()
            if len(parts) >= 2:
                deps[parts[0]] = parts[1]
    return deps


def parse_package_lock(content: Optional[str]) -> Dict[Tuple[str, str], str]:
    """Return mapping of (lock_key, package_name) -> resolved version."""
    deps: Dict[Tuple[str, str], str] = {}
    if not content:
        return deps
    data = json.loads(content)
    packages = data.get("packages") or {}
    for pkg_path, meta in packages.items():
        if not pkg_path or not isinstance(meta, dict) or meta.get("link"):
            continue
        version = meta.get("version")
        if not version:
            continue
        marker = "node_modules/"
        idx = pkg_path.rfind(marker)
        if idx == -1:
            continue
        pkg_name = meta.get("name") or pkg_path[idx + len(marker):]
        deps[(pkg_path, pkg_name)] = version
    return deps


def _extract_image_from_from_line(line: str) -> Optional[Tuple[str, Optional[str]]]:
    tokens = line.split()[1:]
    filtered = [t for t in tokens if not t.startswith("--")]
    if not filtered:
        return None
    image = filtered[0]
    alias: Optional[str] = None
    if len(filtered) >= 3 and filtered[1].upper() == "AS":
        alias = filtered[2].lower()
    return image, alias


def parse_docker_images(files: Dict[str, Optional[str]]) -> Set[str]:
    images: Set[str] = set()
    for path, content in files.items():
        if not content:
            continue
        if path.endswith("Dockerfile"):
            stage_aliases: Set[str] = set()
            for raw in content.splitlines():
                line = raw.split("#", 1)[0].strip()
                if not line.upper().startswith("FROM "):
                    continue
                parsed = _extract_image_from_from_line(line)
                if not parsed:
                    continue
                image, alias = parsed
                if alias:
                    stage_aliases.add(alias)
                if image.lower() == "scratch" or image.lower() in stage_aliases:
                    continue
                images.add(image)
        elif path.endswith(".yml") or path.endswith(".yaml"):
            for raw in content.splitlines():
                line = raw.split("#", 1)[0].strip()
                match = re.match(r"^image:\s*['\"]?([^\s'\"]+)['\"]?", line)
                if not match:
                    continue
                img = match.group(1)
                default_match = re.match(r"^\$\{[^:}]+:-([^}]+)\}$", img)
                if default_match:
                    img = default_match.group(1)
                images.add(img)
    return images


def split_image_ref(image: str) -> Tuple[str, str, str, Optional[str]]:
    digest: Optional[str] = None
    ref = image.strip()
    if "@" in ref:
        ref, digest = ref.split("@", 1)
    parts = ref.split("/")
    if len(parts) > 1 and ("." in parts[0] or ":" in parts[0] or parts[0] == "localhost"):
        registry = parts[0]
        rest = "/".join(parts[1:])
    else:
        registry = "docker.io"
        rest = ref if "/" in ref else f"library/{ref}"
    if ":" in rest:
        repo, tag = rest.rsplit(":", 1)
    else:
        repo, tag = rest, "latest"
    return registry, repo, tag, digest


def fetch_npm_publish_time(pkg_name: str, version: str) -> dt.datetime:
    encoded = urllib.parse.quote(pkg_name, safe="@")
    status, _, data = http_get_json(f"https://registry.npmjs.org/{encoded}")
    if status != 200:
        raise RuntimeError(f"npm registry returned HTTP {status} for {pkg_name}")
    times = data.get("time") or {}
    raw_ts = times.get(version)
    if not raw_ts:
        raise RuntimeError(f"publish timestamp missing for {pkg_name}@{version}")
    return parse_iso_timestamp(raw_ts)


def fetch_go_publish_time(module_path: str, version: str) -> dt.datetime:
    escaped_mod = escape_go_path(module_path)
    escaped_ver = escape_go_path(version)
    url = f"https://proxy.golang.org/{escaped_mod}/@v/{escaped_ver}.info"
    status, _, data = http_get_json(url)
    if status != 200 or not data.get("Time"):
        raise RuntimeError(f"proxy.golang.org returned HTTP {status} for {module_path}@{version}")
    return parse_iso_timestamp(data["Time"])


def _parse_www_authenticate(header_val: str) -> Tuple[str, Dict[str, str]]:
    scheme, _, rest = header_val.partition(" ")
    params = dict(re.findall(r'([a-zA-Z0-9_]+)="([^"]*)"', rest))
    return scheme, params


def _fetch_oci_config_created(registry_host: str, repo: str, ref: str) -> dt.datetime:
    v2_base = f"https://{registry_host}/v2"
    manifest_accept = ", ".join([
        "application/vnd.oci.image.index.v1+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.docker.distribution.manifest.v2+json",
    ])
    headers = {"Accept": manifest_accept}
    status, resp_headers, data = http_get_json(f"{v2_base}/{repo}/manifests/{ref}", headers=headers)
    if status == 401 and "www-authenticate" in resp_headers:
        _, params = _parse_www_authenticate(resp_headers["www-authenticate"])
        realm = params.get("realm")
        if realm:
            query = urllib.parse.urlencode({k: v for k, v in params.items() if k in ("service", "scope")})
            _, _, tok_data = http_get_json(f"{realm}?{query}")
            token = tok_data.get("token") or tok_data.get("access_token")
            if token:
                headers["Authorization"] = f"Bearer {token}"
                status, _, data = http_get_json(f"{v2_base}/{repo}/manifests/{ref}", headers=headers)
    if status != 200:
        raise RuntimeError(f"OCI registry {registry_host} returned HTTP {status} for {repo}:{ref}")
    if "manifests" in data and not data.get("config"):
        manifests = data["manifests"]
        chosen = next(
            (m for m in manifests if (m.get("platform") or {}).get("architecture") == "amd64" and (m.get("platform") or {}).get("os") == "linux"),
            manifests[0],
        )
        _, _, data = http_get_json(f"{v2_base}/{repo}/manifests/{chosen['digest']}", headers=headers)
    cfg = data.get("config") or {}
    cfg_digest = cfg.get("digest")
    if not cfg_digest:
        raise RuntimeError(f"OCI manifest missing config digest for {repo}:{ref}")
    cfg_status, _, cfg_data = http_get_json(
        f"{v2_base}/{repo}/blobs/{cfg_digest}",
        headers={k: v for k, v in headers.items() if k == "Authorization"},
    )
    if cfg_status != 200 or not cfg_data.get("created"):
        raise RuntimeError(f"OCI config missing created timestamp for {repo}:{ref}")
    return parse_iso_timestamp(cfg_data["created"])


def fetch_docker_publish_time(image: str) -> dt.datetime:
    registry, repo, tag, digest = split_image_ref(image)
    if registry in ("docker.io", "registry-1.docker.io"):
        hub_url = f"https://hub.docker.com/v2/repositories/{repo}/tags/{urllib.parse.quote(tag)}"
        status, _, data = http_get_json(hub_url)
        if status == 200:
            if digest:
                for img_entry in data.get("images") or []:
                    if img_entry.get("digest") == digest and img_entry.get("last_pushed"):
                        return parse_iso_timestamp(img_entry["last_pushed"])
            raw_ts = data.get("tag_last_pushed") or data.get("last_updated")
            if raw_ts:
                return parse_iso_timestamp(raw_ts)
        registry_host = "registry-1.docker.io"
    else:
        registry_host = registry
    return _fetch_oci_config_created(registry_host, repo, digest or tag)


def diff_dict(base: Dict, head: Dict) -> List[Tuple]:
    changed = []
    for k, v in head.items():
        if k not in base or base[k] != v:
            changed.append((k, v))
    return changed


def evaluate_age(
    ecosystem: str,
    name: str,
    version: str,
    fetcher: Callable[[], dt.datetime],
    now: dt.datetime,
    min_age: dt.timedelta,
) -> DependencyCheckResult:
    try:
        published_at = fetcher()
    except Exception as exc:
        return DependencyCheckResult(
            ecosystem, name, version, None, None, False, f"metadata lookup failed: {exc}"
        )
    age = now - published_at
    age_days = age.total_seconds() / 86400.0
    if age < min_age:
        remaining = min_age - age
        rem_hours = int(remaining.total_seconds() // 3600) + 1
        return DependencyCheckResult(
            ecosystem,
            name,
            version,
            published_at,
            age_days,
            False,
            f"< {min_age.days}d required (~{rem_hours}h remaining)",
        )
    return DependencyCheckResult(
        ecosystem,
        name,
        version,
        published_at,
        age_days,
        True,
        f"meets >= {min_age.days}d minimum",
    )


def collect_pr_checks(
    base_ref: str,
    min_age_days: int,
    now: Optional[dt.datetime] = None,
    npm_fetcher: Callable[[str, str], dt.datetime] = fetch_npm_publish_time,
    go_fetcher: Callable[[str, str], dt.datetime] = fetch_go_publish_time,
    docker_fetcher: Callable[[str], dt.datetime] = fetch_docker_publish_time,
) -> List[DependencyCheckResult]:
    now_utc = now or dt.datetime.now(dt.timezone.utc)
    min_age = dt.timedelta(days=min_age_days)
    results: List[DependencyCheckResult] = []

    # 1. NPM packages (frontend-ui/package-lock.json)
    base_npm = parse_package_lock(git_read_file(base_ref, NPM_LOCK_FILE)) if base_ref else {}
    head_npm = parse_package_lock(disk_read_file(NPM_LOCK_FILE))
    seen_npm: Set[Tuple[str, str]] = set()
    for (_, pkg_name), ver in sorted(diff_dict(base_npm, head_npm), key=lambda item: (item[0][1], item[1])):
        if (pkg_name, ver) in seen_npm:
            continue
        seen_npm.add((pkg_name, ver))
        results.append(
            evaluate_age("npm", pkg_name, ver, lambda p=pkg_name, v=ver: npm_fetcher(p, v), now_utc, min_age)
        )

    # 2. Go modules (backend-api/go.mod)
    base_go = parse_go_mod(git_read_file(base_ref, GO_MOD_FILE)) if base_ref else {}
    head_go = parse_go_mod(disk_read_file(GO_MOD_FILE))
    for mod, ver in sorted(diff_dict(base_go, head_go)):
        results.append(
            evaluate_age("golang", mod, ver, lambda m=mod, v=ver: go_fetcher(m, v), now_utc, min_age)
        )

    # 3. Docker images (Dockerfiles + docker-compose.yml)
    base_docker = parse_docker_images({p: git_read_file(base_ref, p) for p in DOCKER_FILES}) if base_ref else set()
    head_docker = parse_docker_images({p: disk_read_file(p) for p in DOCKER_FILES})
    for img in sorted(head_docker - base_docker):
        reg, repo_part, tag_part, digest_part = split_image_ref(img)
        display_name = repo_part if reg == "docker.io" else f"{reg}/{repo_part}"
        ver_label = f"{tag_part}@{digest_part[:19]}" if digest_part else tag_part
        results.append(
            evaluate_age("docker", display_name, ver_label, lambda i=img: docker_fetcher(i), now_utc, min_age)
        )

    return results


def format_results_table(results: List[DependencyCheckResult]) -> str:
    headers = ("Status", "Ecosystem", "Dependency", "Version", "Published (UTC)", "Age (Days)", "Notes")
    rows: List[Tuple[str, str, str, str, str, str, str]] = []
    for r in results:
        status = "PASS" if r.passed else "FAIL"
        pub = r.published_at.strftime("%Y-%m-%dT%H:%M:%SZ") if r.published_at else "unknown"
        age_str = f"{r.age_days:.2f}" if r.age_days is not None else "n/a"
        rows.append((status, r.ecosystem, r.name, r.version, pub, age_str, r.reason))

    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))

    def fmt_row(vals: Tuple[str, ...]) -> str:
        return " | ".join(val.ljust(widths[i]) for i, val in enumerate(vals))

    sep = "-+-".join("-" * w for w in widths)
    lines = [fmt_row(headers), sep]
    lines.extend(fmt_row(row) for row in rows)
    return "\n".join(lines)


def format_markdown_summary(results: List[DependencyCheckResult], min_age_days: int) -> str:
    lines = [
        f"### Dependency Cooldown Check ({min_age_days}-Day Minimum Age)",
        "",
        "| Status | Ecosystem | Dependency | Version | Published (UTC) | Age (Days) | Notes |",
        "| :--- | :--- | :--- | :--- | :--- | ---: | :--- |",
    ]
    if not results:
        lines.append("| PASS | - | *(none modified)* | - | - | - | No adopted or updated dependencies detected |")
        return "\n".join(lines) + "\n"
    for r in results:
        status = "PASS" if r.passed else "FAIL"
        pub = r.published_at.strftime("%Y-%m-%dT%H:%M:%SZ") if r.published_at else "unknown"
        age_str = f"{r.age_days:.2f}" if r.age_days is not None else "n/a"
        lines.append(f"| {status} | {r.ecosystem} | `{r.name}` | `{r.version}` | {pub} | {age_str} | {r.reason} |")
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify adopted NPM, Go, and Docker dependencies meet minimum publication age.")
    parser.add_argument("--base-ref", default="", help="Git base commit/ref to diff against (empty checks all dependencies).")
    parser.add_argument("--min-age-days", type=int, default=5, help="Minimum days since publish required before adoption.")
    args = parser.parse_args()

    results = collect_pr_checks(args.base_ref, args.min_age_days)
    print(f"Checked {len(results)} adopted/updated dependency artifact(s) against {args.min_age_days}-day cooldown policy.")
    for r in results:
        status = "PASS" if r.passed else "FAIL"
        pub = r.published_at.strftime("%Y-%m-%dT%H:%M:%SZ") if r.published_at else "unknown"
        age_str = f"{r.age_days:.2f}d" if r.age_days is not None else "n/a"
        print(f"[{status}] [{r.ecosystem}] {r.name}@{r.version} | published={pub} | age={age_str} | {r.reason}")

    if results:
        print("\n" + format_results_table(results) + "\n")

    summary_path = os.environ.get("GITHUB_STEP_SUMMARY", "").strip()
    if summary_path:
        with open(summary_path, "a", encoding="utf-8") as f:
            f.write(format_markdown_summary(results, args.min_age_days))

    violations = [r for r in results if not r.passed]
    if violations:
        print(f"FAILED: {len(violations)} dependency quarantine violation(s) found.")
        return 1
    print("PASSED: All evaluated dependencies satisfy the minimum publication age requirement.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
