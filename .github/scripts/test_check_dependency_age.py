import datetime as dt
import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check_dependency_age as cda


class TestDependencyAgeGate(unittest.TestCase):
    def test_escape_go_path(self):
        self.assertEqual(cda.escape_go_path("github.com/BurntSushi/toml"), "github.com/!burnt!sushi/toml")
        self.assertEqual(cda.escape_go_path("github.com/gin-gonic/gin"), "github.com/gin-gonic/gin")

    def test_parse_go_mod(self):
        raw = """
module github.com/plannedwallet

go 1.23

require github.com/pkg/single v1.2.3

require (
    github.com/gin-gonic/gin v1.10.0
    golang.org/x/crypto v0.23.0 // indirect
)
"""
        parsed = cda.parse_go_mod(raw)
        self.assertEqual(parsed, {
            "github.com/pkg/single": "v1.2.3",
            "github.com/gin-gonic/gin": "v1.10.0",
            "golang.org/x/crypto": "v0.23.0",
        })

    def test_parse_package_lock(self):
        lock = json.dumps({
            "lockfileVersion": 3,
            "packages": {
                "": {"name": "frontend-ui", "version": "0.1.0"},
                "node_modules/@types/node": {"version": "22.0.0"},
                "node_modules/next/node_modules/nanoid": {"version": "3.3.7"},
            },
        })
        parsed = cda.parse_package_lock(lock)
        self.assertEqual(parsed[("node_modules/@types/node", "@types/node")], "22.0.0")
        self.assertEqual(parsed[("node_modules/next/node_modules/nanoid", "nanoid")], "3.3.7")
        self.assertEqual(len(parsed), 2)

    def test_parse_docker_images_ignores_scratch_and_stage_aliases(self):
        files = {
            "backend-api/Dockerfile": """
FROM golang:1.23-alpine AS builder
FROM scratch
COPY --from=builder /app/server /server
""",
            "frontend-ui/Dockerfile": """
FROM node:25-alpine AS builder
FROM builder AS second
FROM nginx:alpine
""",
            "docker-compose.yml": """
services:
  postgres:
    image: postgres:16
  custom:
    image: ${IMG:-ghcr.io/org/custom:1.2.3}
""",
        }
        imgs = cda.parse_docker_images(files)
        self.assertEqual(imgs, {"golang:1.23-alpine", "node:25-alpine", "nginx:alpine", "postgres:16", "ghcr.io/org/custom:1.2.3"})

    def test_split_image_ref(self):
        self.assertEqual(cda.split_image_ref("golang:1.23-alpine"), ("docker.io", "library/golang", "1.23-alpine", None))
        self.assertEqual(
            cda.split_image_ref("ghcr.io/owner/app:v1@sha256:abc"),
            ("ghcr.io", "owner/app", "v1", "sha256:abc"),
        )

    def test_evaluate_age_quarantine_cutoff_and_reporting(self):
        # Reference current UTC execution time so the test mirrors the live rolling 5-day window
        # used by collect_pr_checks (which defaults to dt.datetime.now(dt.timezone.utc)).
        now = dt.datetime.now(dt.timezone.utc)
        min_age = dt.timedelta(days=5)

        too_new = cda.evaluate_age(
            "npm", "react", "19.1.0",
            lambda: now - dt.timedelta(days=4, hours=23),
            now, min_age,
        )
        self.assertFalse(too_new.passed)
        self.assertEqual(too_new.ecosystem, "npm")
        self.assertIsNotNone(too_new.published_at)
        self.assertIsNotNone(too_new.age_days)
        self.assertLess(too_new.age_days, 5.0)

        old_enough = cda.evaluate_age(
            "golang", "github.com/gin-gonic/gin", "v1.10.0",
            lambda: now - dt.timedelta(days=5, minutes=1),
            now, min_age,
        )
        self.assertTrue(old_enough.passed)
        self.assertGreaterEqual(old_enough.age_days, 5.0)

        table = cda.format_results_table([too_new, old_enough])
        self.assertIn("Status", table)
        self.assertIn("Published (UTC)", table)
        self.assertIn("Age (Days)", table)
        self.assertIn("FAIL", table)
        self.assertIn("PASS", table)
        self.assertIn("react", table)
        self.assertIn("github.com/gin-gonic/gin", table)

        md = cda.format_markdown_summary([too_new, old_enough], 5)
        self.assertIn("| FAIL | npm | `react` | `19.1.0` |", md)
        self.assertIn("| PASS | golang | `github.com/gin-gonic/gin` | `v1.10.0` |", md)


if __name__ == "__main__":
    unittest.main()
