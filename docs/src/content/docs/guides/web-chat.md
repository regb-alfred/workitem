---
title: Local web chat
description: Access work items and their existing primary agents from a local browser.
---

`wi web` serves an experimental local work list and full-screen chat:

```bash
wi web
# Or choose a fixed loopback port:
wi web --port 8080
```

Open the private URL printed by the command. It includes a random access secret in its fragment. The page removes that fragment and keeps the secret in the tab's session storage. Refreshing the same tab works; a new tab needs the launch URL. Ctrl-C stops serving but leaves agents running.

## Using the MVP

The list uses the same section grouping and attention ordering as `wi list`, with columns for item, agent, worktree, deep work, labels, and repository. Colors distinguish attention, busy agents, problems, clean worktrees, and changed worktrees. Status text remains visible alongside color.

The Projects multiselect starts with all projects selected. Select one or several projects to narrow the list and Next navigation. Select all includes new projects automatically; Clear all shows an empty list. Selection is kept when navigating between the list and a conversation in the same page. Projects use repository identity, so repositories with the same folder name remain distinct.

Click a work item to read its primary conversation. Its title appears in the top bar; the right panel shows title, description, project, checkout, labels, and runtime details. On narrow screens, ticket details appear below the chat. Next uses the same attention policy, configured priority, and label filters as Wi's CLI within the selected projects. Navigation never starts a runtime or changes lifecycle.

The browser reuses the current verified TUI or headless owner. It does not change modes, acquire conversation ownership, or interrupt a turn. Use Send if idle to submit a prompt without steering a turn that became busy. Queue follow-up explicitly requests after-run delivery. A successful submission is not proof of completion. If a response is lost, check history before retrying; there is no automatic retry.

## Starting conversations

New is a centered chat-style composer: choose a repository, write a message, and Send. There is no title field or item-creation form. A current conversation's repository or sole filtered repository is preselected unless you already have a draft. A single known repository is selected automatically. Drafts survive navigation within the page.

Send first saves the work item and its prompt through the daemon and records working intent. The server returns a receipt once those steps are durable, then generates a title, materializes a managed checkout, starts the default agent headlessly, and submits the saved prompt in the background. The composer clears, retains the repository, and focuses the message box again. Enter sends; Shift+Enter inserts a newline, both here and in an existing conversation. A six-second "Work item started" toast links to the conversation. That link uses the immutable work-item ID returned after durable creation, so it works before Pi generates the title or initializes the primary conversation. Opening the link while startup is pending shows a centered loading wheel only. History and live reads begin when startup finishes; genuine startup failures remain visible. The toast confirms saved working intent, not a completed agent startup or turn. The page never automatically switches to the new conversation. An Attention badge uses the existing list policy; startup problems have an explicit View conversation link.

Pi generates the title with a separate, bounded print-mode model request. That request has no tools, extensions, project context, or persisted session. Creation assigns the permanent ID immediately with provisional naming. After title generation, a revision-checked daemon command finalizes the title, unique slug, and implementation-branch specification together, before any checkout claim or primary conversation exists. It does not rename an actual Git branch. Existing claims, conversations, lifecycle changes, or explicitly chosen branches block finalization; ordinary later title edits leave slug and branch unchanged. The permanent ID and source commit never change. If title generation fails or is unsupported, a short task summary is used instead. The original prompt is submitted unchanged to the primary agent. A startup receipt records handoff, not turn completion. Immediate native rejection is reported as a startup problem; subsequent runtime/model failures use normal agent observations.

The repository picker reads Wi's repository registry, including repositories with no items and those hidden by item label filters. It displays current registered names; selection and Next filtering use permanent repository IDs. New resolves the selected repository's owned bare store, integration-branch commit, and default labels through the daemon. It does not depend on the external onboarding checkout. Register repositories through `wi repo add NAME --from SOURCE`, then reopen New.

Startup is bounded to four concurrent jobs and 32 outstanding requests. Receipts are rebuildable, bounded server-memory metadata, not durable jobs. Stopping the web server interrupts unfinished startup without aborting already-running primary agents. The saved working item remains actionable through Wi's normal observations, and startup can be continued with the CLI. Server restart does not silently restart agents.

If startup or initial delivery fails, the item remains and the composer stays ready for another message. Inspect the retained conversation rather than resending it as a new task. If the HTTP response is lost or creation is uncertain, the draft is preserved and retry is blocked until you explicitly confirm you checked the list. There is no automatic creation retry or exactly-once HTTP guarantee.

Starting or resuming an existing ticket's agent remains a CLI action:

```bash
wi start --agent-mode headless item-slug
wi resume --agent-mode headless --item item-slug
# For already working items with no live owner:
wi agent runtime ensure --mode headless --item item-slug
```

Do not ensure a headless runtime while a TUI owner is active. The browser can access that TUI owner directly when its driver supports the new content protocol.

Installing this version does not update drivers already running. Those owners remain untouched. Readable history still works with the new history reader; live output and idle-only Send require the updated driver. Existing owners can accept explicit follow-up if they advertise that control. Wait for a turn to settle and explicitly stop/restart the owner when you want the new functionality. Web archive refuses a live owner without the new idle-guard support; it does not silently restart it.

Archive confirms intent and invokes the existing safety workflow with force disabled. Dirty or branch-mismatched workspaces and busy agents block archive. An idle agent may shut down before a later cleanup step fails. The page reports retained shutdown/terminal-cleanup warnings and re-reads durable lifecycle after an attempt. It never rolls those steps back or claims failure means nothing changed. Checkout-claim release is committed with the archived manifest, so a failed final save can leave the claim assigned even after terminal cleanup.

## Current limits

- Bundled Pi implements readable history and live content. Other drivers need the optional public APIs and must advertise `idle_guard` for idle-only controls and web archive of a live owner.
- The page displays plain text, code as preformatted text, and tool summaries. Thinking and non-text blocks have placeholders. Markdown rendering, attachments, and older-history paging are not implemented.
- History follows one native entry branch, up to 200 messages, 16 KiB per message, and 1 MiB total text. Files over 64 MiB and legacy records without tree IDs are refused with a warning. The reader never rewrites native content.
- The TUI content endpoint reports its in-memory leaf. Without that endpoint, the history reader selects the last persisted leaf and warns that an older TUI's current tree position may differ.
- Live output is a replaceable preview of the currently streaming assistant message, bounded to 256 KiB, plus activity text. The server polls the driver snapshot every 400 ms and sends NDJSON to the browser. History refreshes every three seconds. Saved messages come only from native history. A completed preview can disappear briefly before the next history refresh, and output completed entirely between polls appears in history rather than as a preview.
- Reconnecting replaces the preview and reloads history; it does not replay deltas or persist browser messages. This MVP does not implement a snapshot watermark or lossless event replay.
- Multiple tabs share the same primary conversation. They have separate drafts, not separate runtime owners.
- No implicit start on navigation, browser lifecycle editing, forced archive, runtime migration, remote tools, or hosted access.

## Security boundary

The server binds only to `127.0.0.1`. The bearer secret grants local operator authority to read conversations, create tickets in known repositories, and submit coding-agent work. Keep the launch URL private. Authentication applies to all API reads, streams, and mutations; static assets contain no secret. Requests must have the exact loopback Host. Cross-origin and cross-site requests are rejected; mutations require same-origin JSON and the bearer header. There is no CORS support or state-changing GET.

The client uses `textContent`, never conversation HTML. CSP forbids external assets, inline scripts, and framing. Content responses use no-store, no-referrer, and nosniff. Request, transcript and live-preview sizes, active requests, streams, and timeouts are bounded. The access server connects to the operator daemon; the restricted agent endpoint cannot serve this UI.

This is not a defense against a compromised browser profile, another process already acting as the same Unix user, or stolen access secrets. Tab session storage is readable by page scripts, unlike an HttpOnly cookie. The MVP avoids a cookie/session exchange; an HttpOnly session is a possible later hardening step. Public hosting and remote browser access are unsupported.
