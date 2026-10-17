#!/usr/bin/env python3
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HOOK = Path(__file__).with_name("sign-and-notarize.sh")

class SigningGate(unittest.TestCase):
    def run_hook(self, target="darwin", snapshot="false", failure="", status="Accepted"):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            fake = root / "fake"
            fake.write_text("""#!/bin/bash
set -eu
name="$(basename "$0")"
echo "$name $*" >> "$CALL_LOG"
if [[ "$name" == "$FAIL_COMMAND" ]]; then exit 23; fi
case "$name" in
codesign)
  if [[ "$1" == --display ]]; then
    printf 'Authority=Developer ID Application: Test\nTeamIdentifier=TESTTEAM\nTimestamp=now\nflags=0x10000(runtime)\n'
  fi ;;
xcrun) printf '{"status":"%s"}\n' "$NOTARY_STATUS" ;;
esac
""")
            fake.chmod(0o755)
            for command in ["codesign", "ditto", "xcrun"]:
                (root / command).symlink_to(fake)
            env = os.environ | {
                "PATH": str(root) + os.pathsep + os.environ["PATH"],
                "CALL_LOG": str(root / "calls"), "FAIL_COMMAND": failure,
                "NOTARY_STATUS": status,
                "APPLE_SIGNING_IDENTITY": "Developer ID Application: Test",
                "APPLE_SIGNING_KEYCHAIN": "test", "APPLE_TEAM_ID": "TESTTEAM",
                "APPLE_API_KEY_PATH": "test.p8", "APPLE_API_KEY_ID": "test",
                "APPLE_API_ISSUER": "test",
            }
            result = subprocess.run(["bash", str(HOOK), target, "binary", snapshot], env=env, capture_output=True)
            log = root / "calls"
            return result.returncode, log.read_text() if log.exists() else ""

    def test_release_requires_accepted_notary(self):
        code, calls = self.run_hook()
        self.assertEqual(code, 0)
        self.assertIn("--timestamp --options runtime", calls)
        self.assertIn("--check-notarization", calls)
        code, calls = self.run_hook(status="Invalid")
        self.assertNotEqual(code, 0)
        self.assertNotIn("--check-notarization", calls)

    def test_command_failures_stop_publication(self):
        for command in ["codesign", "ditto", "xcrun"]:
            with self.subTest(command=command):
                code, _ = self.run_hook(failure=command)
                self.assertNotEqual(code, 0)

    def test_only_nonmac_and_snapshots_skip_signing(self):
        for target, snapshot in [("linux", "false"), ("windows", "false"), ("darwin", "true")]:
            code, calls = self.run_hook(target, snapshot)
            self.assertEqual(code, 0)
            self.assertEqual(calls, "")
        self.assertNotEqual(self.run_hook("unexpected")[0], 0)

if __name__ == "__main__":
    unittest.main()
