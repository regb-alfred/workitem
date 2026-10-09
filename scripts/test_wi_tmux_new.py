#!/usr/bin/env python3
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("wi-tmux-new")
STUB = '''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
name=Path(sys.argv[0]).name
args=sys.argv[1:]
with open(os.environ["CALLS"],"a") as log:
 log.write(json.dumps({"name":name,"args":args,"cwd":os.getcwd(),"stdin":sys.stdin.read() if name=="pi" else ""})+"\\n")
mode=os.environ.get("STUB_MODE","")
if name=="pi":
 if mode=="pi failure": sys.exit(7)
 print("Bad\\nTitle" if mode=="bad title" else "Investigate token refresh")
elif args==["show","--json"]:
 print(json.dumps({"manifest":{"id":"current-item","repository":{"id":"" if mode=="no repo" else "repo-id"}}}))
elif args[0]=="new":
 if mode=="startup failure":
  print("created work item retained-item, but could not start its agent",file=sys.stderr)
  sys.exit(1)
 print(json.dumps({"new":{"manifest":{"id":"new-item"}}}))
elif args!=["switch","new-item"]: sys.exit(2)
'''


class PopupTests(unittest.TestCase):
    def invoke(self, prompt="Why does refresh hang?", mode="", context=True):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root)
            for name in ("wi", "pi"):
                (path / name).write_text(STUB)
                (path / name).chmod(0o700)
            env = dict(os.environ, PATH=f"{root}:{os.environ['PATH']}",
                       CALLS=f"{root}/calls", STUB_MODE=mode,
                       WI_ID="current-item" if context else "",
                       WI_TMUX_CLIENT="client-1")
            result = subprocess.run([str(SCRIPT)], input=prompt + "\n", text=True,
                                    capture_output=True, env=env)
            calls = [json.loads(line) for line in (path / "calls").read_text().splitlines()] if (path / "calls").exists() else []
            return result, calls

    def test_original_prompt_and_implicit_repository(self):
        prompt = '@secret.md $(touch /tmp/nope) "quoted" \\ path  '
        result, calls = self.invoke(prompt)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([c["name"] for c in calls], ["wi", "pi", "wi", "wi"])
        self.assertEqual(calls[1]["stdin"], prompt)
        for flag in ("--print", "--no-session", "--no-tools", "--no-extensions", "--no-mcp", "--no-context-files", "--no-approve"):
            self.assertIn(flag, calls[1]["args"])
        self.assertTrue(Path(calls[1]["cwd"]).name.startswith("wi-title-"))
        self.assertEqual(calls[2]["args"], ["new", "--title=Investigate token refresh", "--start", "--prompt=" + prompt, "--json"])
        self.assertEqual(calls[3]["args"], ["switch", "new-item"])

    def test_flag_like_prompts_are_data(self):
        for prompt in ("--help", "--json", "@secret.md"):
            result, calls = self.invoke(prompt)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("--prompt=" + prompt, calls[2]["args"])

    def test_cancel(self):
        result, calls = self.invoke("   ")
        self.assertEqual(result.returncode, 0)
        self.assertEqual(len(calls), 1)

    def test_missing_context(self):
        result, calls = self.invoke(context=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls, [])

    def test_failures_do_not_create_or_switch(self):
        for mode, count in (("no repo", 1), ("pi failure", 2), ("bad title", 2), ("startup failure", 3)):
            with self.subTest(mode=mode):
                result, calls = self.invoke(mode=mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(len(calls), count)
                self.assertFalse(any(c["args"][0] == "switch" for c in calls))


if __name__ == "__main__":
    unittest.main()
