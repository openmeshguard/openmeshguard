"""Test release upload authorization with an isolated fake GitHub CLI."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class PreflightTests(unittest.TestCase):
    def run_case(self, status, body, *, repository_status=200, token=True, malformed=False, network_error=False):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            gh = root / "gh"
            gh.write_text(
                '#!' + sys.executable + '\n'
                'import json, os, sys\n'
                'fixture=json.loads(os.environ["PREFLIGHT_FIXTURE"])\n'
                'if fixture["network_error"]: sys.exit(1)\n'
                'repo="/releases/" not in sys.argv[-1]\n'
                'status=fixture["repository_status"] if repo else fixture["status"]\n'
                'body={"full_name":"openmeshguard/openmeshguard"} if repo else fixture["body"]\n'
                'print(f"HTTP/2.0 {status} Status\\nContent-Type: application/json\\n")\n'
                'print("bad JSON" if fixture["malformed"] and not repo else json.dumps(body))\n'
                'sys.exit(0 if status==200 else 1)\n'
            )
            gh.chmod(0o755)
            env = dict(os.environ)
            env.pop("GH_TOKEN", None)
            env.pop("GITHUB_TOKEN", None)
            if token:
                env["GH_TOKEN"] = "test-only-token"
            env["PATH"] = str(root) + os.pathsep + env["PATH"]
            env["PREFLIGHT_FIXTURE"] = json.dumps(dict(
                status=status, body=body, repository_status=repository_status, malformed=malformed, network_error=network_error
            ))
            return subprocess.run(
                [sys.executable, str(Path(__file__).with_name("preflight.py")), "v0.1.0"],
                env=env, capture_output=True, text=True, check=False,
            )

    def test_absent_release_allowed(self):
        self.assertEqual(self.run_case(404, {"message": "Not Found"}).returncode, 0)

    def test_existing_draft_allowed(self):
        self.assertEqual(self.run_case(200, {"tag_name": "v0.1.0", "draft": True}).returncode, 0)

    def test_public_release_rejected(self):
        result = self.run_case(200, {"tag_name": "v0.1.0", "draft": False})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing uploads", result.stderr)

    def test_fail_closed_for_errors(self):
        for status in (401, 403, 429, 500):
            with self.subTest(status=status):
                self.assertNotEqual(self.run_case(status, {}).returncode, 0)
        self.assertNotEqual(self.run_case(404, {}, repository_status=404).returncode, 0)
        self.assertNotEqual(self.run_case(404, {}, network_error=True).returncode, 0)
        self.assertNotEqual(self.run_case(404, {}, token=False).returncode, 0)
        self.assertNotEqual(self.run_case(200, {}, malformed=True).returncode, 0)
        self.assertNotEqual(self.run_case(200, {"tag_name": "different", "draft": True}).returncode, 0)
        self.assertNotEqual(self.run_case(200, {"tag_name": "v0.1.0"}).returncode, 0)


if __name__ == "__main__":
    unittest.main()
