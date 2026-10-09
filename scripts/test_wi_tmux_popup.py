#!/usr/bin/env python3
"""Exercise the documented popup binding in an isolated tmux server and PTY."""
import json
import os
from pathlib import Path
import pty
import shutil
import subprocess
import tempfile
import time
import unittest


@unittest.skipUnless(shutil.which("tmux"), "tmux is not installed")
class PopupEnvironmentTests(unittest.TestCase):
    def test_creation_identity_and_invoking_client(self):
        self.verify_popup("I")

    def test_picker_uses_session_path_and_invoking_client(self):
        self.verify_popup("O")

    def verify_popup(self, key):
        docs = (Path(__file__).parent.parent / "docs/src/content/docs/guides/tmux.md").read_text()
        start = docs.index(f'bind-key {key} run-shell')
        binding = docs[start:docs.index("\n```", start)].replace("\\\n", " ")
        with tempfile.TemporaryDirectory() as root:
            sock = root + "/tmux.sock"
            result = root + "/env.json"
            helper = Path(root) / (".local/bin/wi-tmux-new" if key == "I" else "go/bin/wi")
            helper.parent.mkdir(parents=True)
            helper.write_text('#!/usr/bin/env python3\nimport os,json\n'
                              'json.dump({k:os.environ.get(k) for k in ("WI_ID","WI_TMUX_CLIENT")},'
                              f'open({result!r},"w"))\n')
            helper.chmod(0o700)
            if key == "I":
                # A wrong binding must fail, never fall through to installed wi.
                blocker = helper.parent / "wi"
                blocker.write_text("#!/bin/sh\nexit 99\n")
                blocker.chmod(0o700)
            env = {k: v for k, v in os.environ.items()
                   if not k.startswith(("WI_", "PI_", "XDG_", "DIRENV_"))
                   and k not in ("HOME", "TMUX", "TMUX_PANE")}
            env.update(TERM="xterm-256color", HOME=root)
            for key_name, directory in (("XDG_DATA_HOME", "data"), ("XDG_CONFIG_HOME", "config"),
                                        ("XDG_CACHE_HOME", "cache"), ("XDG_STATE_HOME", "state"),
                                        ("XDG_RUNTIME_DIR", "run"), ("PI_CODING_AGENT_DIR", "pi")):
                path = Path(root) / directory
                path.mkdir(mode=0o700)
                env[key_name] = str(path)
            master, slave = pty.openpty()
            client = subprocess.Popen(
                ["tmux", "-S", sock, "-f", "/dev/null", "new-session", "-s", "probe"],
                stdin=slave, stdout=slave, stderr=slave,
                env=env,
            )
            def tmux(*args):
                return subprocess.run(["tmux", "-S", sock, *args], check=True,
                                      text=True, capture_output=True)
            try:
                for _ in range(50):
                    if Path(sock).exists():
                        break
                    time.sleep(.1)
                tmux("set-environment", "-t", "probe", "WI_ID", "actual-item-id")
                # The global PATH has no wi. The popup must use the session PATH.
                tmux("set-environment", "-t", "probe", "PATH", str(helper.parent) + os.pathsep + os.environ["PATH"])
                actual_client = tmux("list-clients", "-F", "#{client_name}").stdout.strip()
                self.assertTrue(actual_client)
                # Send a key to this test client's PTY, not to a user pane.
                # Binding expansion and direct display-popup expansion differ.
                config = Path(root) / "binding.conf"
                config.write_text(binding + "\n")
                tmux("source-file", str(config))
                os.write(master, b"\x02" + key.encode())
                for _ in range(50):
                    if Path(result).exists():
                        break
                    time.sleep(.1)
                self.assertTrue(Path(result).exists(), "popup did not invoke the expected stub")
                observed = json.loads(Path(result).read_text())
                self.assertEqual(observed["WI_ID"], "actual-item-id")
                self.assertEqual(observed["WI_TMUX_CLIENT"], actual_client)
            finally:
                subprocess.run(["tmux", "-S", sock, "kill-server"], capture_output=True)
                client.wait(timeout=5)
                os.close(master)
                os.close(slave)


if __name__ == "__main__":
    unittest.main()
