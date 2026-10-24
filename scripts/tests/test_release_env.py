"""Release container configuration regression without Docker or external services."""

from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]
COMPOSE = ROOT / "deploy/release/compose.yaml"


class ReleaseEnvTests(unittest.TestCase):
    def test_release_container_issuer_matches_repository_configuration(self):
        # The runtime image only copies the binary: a YAML-only issuer would stop startup.
        compose = COMPOSE.read_text()
        match = re.search(r"^\s+JWT_ISSUER:\s*\$\{JWT_ISSUER:-([^}]+)\}\s*$", compose, re.M)
        self.assertIsNotNone(match, "release backend must receive a nonempty JWT_ISSUER")
        for name in ("config.production.yaml", "config.development.yaml"):
            configured = re.search(r'^\s+issuer:\s*"([^"\n]+)"\s*$',
                                   (ROOT / "config" / name).read_text(), re.M)
            self.assertIsNotNone(configured)
            self.assertEqual(match.group(1), configured.group(1),
                             "changing issuer on container upgrade would invalidate existing tokens")

