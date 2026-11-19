import base64
import gzip
import json
import random
import sys
import unittest
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import run_ecs_sql as ecs_sql  # noqa: E402


class RunEcsSqlTests(unittest.TestCase):
    def test_build_ecs_overrides_fits_init_sql_under_8192_bytes(self):
        init_sql_path = SCRIPT_DIR.parent.parent / "backend-api" / "internal" / "database" / "init.sql"
        sql_bytes = init_sql_path.read_bytes()
        raw_overrides = ecs_sql.build_ecs_overrides("migration-runner", sql_bytes)
        self.assertLess(len(raw_overrides.encode("utf-8")), ecs_sql.MAX_ECS_OVERRIDES_BYTES)

        parsed = json.loads(raw_overrides)
        self.assertEqual(parsed["containerOverrides"][0]["name"], "migration-runner")
        cmd = parsed["containerOverrides"][0]["command"]
        self.assertEqual(cmd[:2], ["/bin/sh", "-c"])

        # Extract base64 gzip blob and verify round-trip equals init.sql
        token = cmd[2].split("printf '%s' '", 1)[1].split("'", 1)[0]
        decoded = gzip.decompress(base64.b64decode(token))
        self.assertEqual(decoded, sql_bytes)

    def test_build_ecs_overrides_rejects_oversized_sql(self):
        rng_bytes = random.Random(42).randbytes(10000)
        with self.assertRaises(ValueError):
            ecs_sql.build_ecs_overrides("migration-runner", rng_bytes)

    def test_format_network_config(self):
        cfg = ecs_sql.format_network_config("subnet-aaa, subnet-bbb", "sg-12345")
        self.assertEqual(
            cfg,
            "awsvpcConfiguration={subnets=[subnet-aaa,subnet-bbb],securityGroups=[sg-12345],assignPublicIp=ENABLED}",
        )
        with self.assertRaises(ValueError):
            ecs_sql.format_network_config("", "sg-12345")


if __name__ == "__main__":
    unittest.main()
