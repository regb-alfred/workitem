---
title: Delegate work
description: Create self-contained tasks and send prompts to agents.
---

Delegation works best when the receiving agent does not need the conversation that produced the task. These examples assume current Wi item context; outside an item, select a [registered repository](../../concepts/repositories/) with `--repo NAME`.

## Create and start in one command

```bash
wi new \
  --desc-file audit.md \
  --start --agent-mode headless \
  --title "Audit dependency updates"
```

This command:

1. stores `audit.md` as the item description
2. moves the item to `working`
3. materializes its managed workspace
4. starts an unattached runtime
5. sends the configured startup message to the agent, asking it to read `wi show` and work on the task

Use `--prompt TEXT` with `--start` to override the startup message without changing the description. Resume never sends this message again.

TUI is the default mode when `--agent-mode` is omitted. The command starts the normal tmux runtime but does not switch or attach your current terminal.

## Write a stronger handoff

For a substantial task, use a file:

```bash
wi new --desc-file handoff.md --title "Fix stale runtime ownership"
```

A useful handoff contains:

- why the work matters
- exact observed behavior or reproduction commands
- relevant files and constraints
- a concrete next step
- observable completion criteria

Add `--start` to that creation command to send the configured initial message immediately. For backlog work already filed, enter its TUI:

```bash
wi start fix-stale-runtime-ownership
```

Existing-item `wi start` only opens the runtime; it sends no message. Ask the agent to read `wi show` yourself, or use runtime control.

## Prompt-only questions

A description is optional. For a one-off question, supply an explicit title and startup prompt:

```bash
wi new --title "Investigate token refresh hangs" --start \
  --prompt "Why does token refresh sometimes hang?"
```

The prompt goes to the primary conversation unchanged; no description is derived from it. Wi always requires `--title` and derives only the slug. The [tmux popup](../tmux/#create-from-a-prompt-popup) is an external harness that can generate a title with a separate ephemeral Pi call before invoking Wi.

## Send follow-up work

```bash
wi agent control send --follow-up --item fix-stale-runtime-ownership \
  "Also test PID reuse and report the focused test command."
```

Use `--file` for a multiline message. Delivery and task completion are different events. Monitor the runtime or enter its TUI to inspect the result:

```bash
wi agent monitor --item fix-stale-runtime-ownership --limit 50
wi switch fix-stale-runtime-ownership
```

## Keep authority narrow

An agent sandbox can receive `WI_AGENT_DAEMON_SOCKET`, which points to a restricted daemon endpoint. That endpoint permits safe managed-slot backlog creation but rejects lifecycle, workspace, runtime, terminal, deletion, shutdown, and `--start` operations.

This lets an agent file follow-up work without granting control over the user's active environment.
