---
title: Agents and conversations
description: Separate agent definitions, drivers, durable conversations, and replaceable runtimes.
---

Each work item selects one agent definition and has at most one durable primary conversation. Only one runtime may own that conversation at a time.

An **agent definition** is the configured agent selected by a user, such as `default` or `web-search`. A **driver** integrates an agent family with wi. Multiple definitions may use the same driver with different settings. The implicit `default` definition uses the standard Pi driver when no configuration overrides it.

```bash
wi new --agent web-search "Research upstream behavior"
wi start --new --agent web-search "Investigate rendering"
```

The conversation is part of the item's history. The runtime is a replaceable driver and backend process that opens and controls it. Stopping a runtime does not discard the conversation.

## TUI mode

TUI mode is the default:

```bash
wi start item-slug
wi switch item-slug
```

A driver advertising native TUI support runs its backend inside a wi-owned tmux session. You can leave and re-enter the session while the conversation continues to belong to the item. The standard Pi driver runs native interactive Pi this way.

## Headless mode

Headless mode does not use tmux:

```bash
wi start --agent-mode headless item-slug
```

A driver must advertise headless support to use this mode. Stop the current runtime before switching modes:

```bash
wi agent runtime stop item-slug
wi agent runtime status item-slug
wi switch item-slug
```

Wait until status reports that the runtime is offline before starting the other mode.

## Runtime observation and control

```bash
wi agent runtime status item-slug
wi agent status item-slug
wi agent monitor --limit 50 item-slug
wi agent control abort --item item-slug
wi agent runtime stop item-slug
```

Agent status reflects whether the current runtime appears busy, idle, or in need of inspection. It does not depend on how the prompt was submitted.

## Send input

Send a message to the active runtime:

```bash
wi agent control send --item item-slug "Run the focused tests and explain failures."
```

While the agent is busy, the default steering message runs after the current assistant turn and tool calls. Use `--follow-up` when the message must wait for the current run to settle. Use `--file` or `--stdin` for longer input.

Control delivery confirms only that the active driver accepted the message. Use `wi agent monitor`, or enter the TUI, to inspect the result. Drivers advertise which controls they support and must reject unsupported operations rather than emulate TUI keystrokes.

## One runtime at a time

TUI and Headless runtimes cannot own the same conversation simultaneously. Stop the active runtime before starting another mode.

Force cleanup stops a process only after `wi` verifies that it still belongs to the item. A stale runtime record will not cause an unrelated process to be stopped.
