---
title: wi documentation
description: Learn how to organize and run durable AI-assisted development work with wi.
---

`wi` is a Linux-first command-line tool for organizing AI-assisted development as durable work items. A work item keeps the task description, checkout, agent conversation, lifecycle, and history together across restarts.

:::note[New to wi?]
Follow the [installation guide](getting-started/installation/) and then the [quick start](getting-started/quick-start/). The quick start takes you through both interactive and unattended agent workflows.
:::

## How wi fits together

A work item is the durable center of the workflow. Git worktrees, coding-agent runtimes, and tmux sessions are resources attached to it. They can stop or be recreated without losing the task.

| Part | What it represents |
| --- | --- |
| Work item | Durable task identity, description, state, labels, and history |
| Repository | Named independent bare Git store and integration policy |
| Workspace | An isolated Git checkout assigned to the item |
| Agent | The selected configured agent definition and its driver |
| Conversation | The item's persistent driver-owned primary conversation |
| Runtime | The current TUI or headless process using that conversation |
| Attention | A ranking of active items ready for human input |

Read [Work items](concepts/work-items/) and [Lifecycle](concepts/lifecycle/) for the full model.

## A common workflow

Register the repository, then create and delegate a task:

```bash
wi repo add project --from /path/to/repository --branch main
wi new --title "Add request retries" --repo project --desc-file task.md --start
```

Check active work and enter the agent session:

```bash
wi list
wi switch add-request-retries
```

After reviewing and committing the changes:

```bash
wi merge --item add-request-retries main
wi archive add-request-retries
```

`--start` submits a configurable message asking the agent to read the task using `wi show`. `--prompt` overrides only that message, not the description. If you create backlog work with `--desc-file`, `wi start` opens the runtime but does not submit the description. The [quick start](getting-started/quick-start/) demonstrates both forms.

## Continue by task

- [Repositories](concepts/repositories/) covers onboarding, starting commits, and renaming.
- [Everyday workflow](guides/everyday-workflow/) covers capture, supervision, integration, and cleanup.
- [Delegate work](guides/delegation/) covers background TUI and headless agents.
- [Review agent work](guides/review-and-follow-up/) covers diffs, tests, and corrections.
- [Use tmux](guides/tmux/) covers interactive navigation.
- [Troubleshooting](reference/troubleshooting/) explains common refusals and failures.
- [Command map](reference/commands/) lists commands by responsibility.
