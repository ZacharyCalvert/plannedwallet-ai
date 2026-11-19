import io
import sys
import unittest
from contextlib import redirect_stdout
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import verify_ecs_taskdef_env as vet  # noqa: E402


VALID_TERMS_HASH = "b08f3d8b1cd44544888a78ed13316c0910725944bc46808ab542e36d9b2fd84c"


def sample_ecs_task_def(**env_overrides):
    environment = {
        "REQUIRE_CAPTCHA": "true",
        "GIN_MODE": "release",
        "PRODUCTION_ENV": "true",
        "ENFORCE_POLICY": "true",
        "PW_TERMS": VALID_TERMS_HASH,
        "POSTGRES_HOST": "psql-pw-stage.abc123.us-east-2.rds.amazonaws.com",
        "POSTGRES_DB": "plannedwallet",
        "POSTGRES_SSLMODE": "require",
    }
    secrets = {
        "JWT_SECRET": "arn:aws:ssm:us-east-2:123456789012:parameter/pw/stage/jwt_secret",
        "EXPECTED_ORIGIN_SECRET": "arn:aws:ssm:us-east-2:123456789012:parameter/pw/stage/origin_secret",
        "POSTGRES_PASSWORD": "arn:aws:ssm:us-east-2:123456789012:parameter/pw/stage/db_password",
    }
    for k, v in env_overrides.items():
        if v is None:
            environment.pop(k, None)
            secrets.pop(k, None)
        else:
            environment[k] = v

    return {
        "family": "task-pw-stage-api",
        "containerDefinitions": [
            {
                "name": "backend",
                "image": "123456789012.dkr.ecr.us-east-2.amazonaws.com/ecr-pw-stage-api:oldsha",
                "environment": [{"name": k, "value": v} for k, v in environment.items()],
                "secrets": [{"name": k, "valueFrom": v} for k, v in secrets.items()],
            }
        ],
    }


class VerifyEcsTaskDefEnvTests(unittest.TestCase):
    def test_valid_prod_config_and_secret_masking(self):
        doc = sample_ecs_task_def()
        buf = io.StringIO()
        with redirect_stdout(buf):
            errors, warnings = vet.verify_task_definition(doc, "prod")
        out = buf.getvalue()
        self.assertEqual(errors, [])
        self.assertEqual(warnings, [])
        self.assertNotIn("rds.amazonaws.com", out)
        self.assertNotIn("123456789012", out)

    def test_missing_backend_container_raises(self):
        doc = {"family": "task-pw-stage-api", "containerDefinitions": [{"name": "other"}]}
        with self.assertRaises(ValueError):
            vet.verify_task_definition(doc, "stage", container_name="backend")

    def test_missing_origin_secret_fails(self):
        doc = sample_ecs_task_def(EXPECTED_ORIGIN_SECRET=None)
        with redirect_stdout(io.StringIO()):
            errors, _ = vet.verify_task_definition(doc, "prod")
        self.assertTrue(any("EXPECTED_ORIGIN_SECRET" in e for e in errors))

    def test_enforce_policy_must_be_true_when_production_env_true(self):
        doc = sample_ecs_task_def(ENFORCE_POLICY="false")
        with redirect_stdout(io.StringIO()):
            errors, _ = vet.verify_task_definition(doc, "stage")
        self.assertTrue(any("ENFORCE_POLICY" in e for e in errors))

    def test_rds_sslmode_must_require_tls_in_prod(self):
        doc = sample_ecs_task_def(POSTGRES_SSLMODE="disable")
        with redirect_stdout(io.StringIO()):
            errors, _ = vet.verify_task_definition(doc, "prod")
        self.assertTrue(any("SSLMODE" in e for e in errors))

    def test_db_aliases_supported(self):
        doc = sample_ecs_task_def(
            POSTGRES_HOST=None,
            POSTGRES_DB=None,
            POSTGRES_SSLMODE=None,
            DB_HOST="psql-pw.internal",
            DB_NAME="plannedwallet",
            DB_SSLMODE="require",
        )
        with redirect_stdout(io.StringIO()):
            errors, _ = vet.verify_task_definition(doc, "stage")
        self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
