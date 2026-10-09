---
name: wi
description: Use when the user wants to manage work items or use the wi cli.
---

`wi` is a cli to manage work items for git-based projects.

- Each work item is isolated in its own git worktree and is assigned to its own agent session.
- `wi` is primarly a cli designed for human operators and many commands are meant for human use.
- Avoid using any create/edit commands until explicitly asked to by the user.
- Do not perform navigation, workspace, terminal, or workflow cleanup operations directly as they can impact the currently visible item for the user.
- Most `wi` commands automatically select the current work item based on where the cli is ran and normal operations should not need to select a different work item.

## Manage repos

Git repositories are the top-level organizations for work items, a work item is always spawned within the context of a repo.
Wi keeps a global store of all repos that it manages, and a repo must be added explicitly before being able to create a work item in it.

```bash
wi repo list
wi repo add NAME --from GIT_CHECKOUT_PATH --branch DEFAULT_INTEGRATION_BRANCH
```

Use a local checkout of the repo you want to start managing, set the integration branch that `wi` will use to merge back the changes on each work item, if you omit it, it will try to discover it automatically and usually resolve main or master.

Usually you don't need to specify a repo as you will file items related to the current item which is already associated with the right repo.

## Work item context

An agent session is always running in the context of a work item.
You can inspect `WI_*` env variables if you need to understand in which work item context you are currently running.

To inspect the item description and the task:

```bash
wi show --json
```

## Filing work items

Create a new work item with its description in a markdown file:

```bash
wi new --title "Item title" --desc-file DESC_FILE
```

It default to the repo of the current work item, use `--repo REPO_NAME` if you want to file the item in another repo.
The command will file the work item but not start it.

The title should be short and help quickly recognize the item.
Titles are ephemeral, they don't need to capture fully the task but contain enough hints that they can be differentiated among currently active items.
Good titles: `Fix flag xyz`, `Reproduce banner flicker`
Bad title: `issue follow-up`, `Fix the flag abc identified on banner 42 on June 16 when user click twice`

The description should be written in a descriptive style as you would do on a ticket tracking system.
It should be very actionable and include enough context to be self-contained.
If you file an item as a related work or follow up work, consider that the person that will work on it might have no context of your current session and all they will know is the repository at the chosen branch along with the item description that you sent.

An item is always created from the default integration branch of the repo; if you want to start it from the commit in the current worktree, use `--base=@`.

A new item is added in the backlog and not acted on. You can start an agent automatically to tackle it with `--start`.

## Completing work items

When the work on the item is completed and commited on the local branch, use:

```bash
wi merge
```

to merge it to the default integration branch.
This command does not push or sync the remote, if this is required this should be done with regular git commands.

Once an item is merged and all work scoped to the task is settled (or accounted for in follow-up items), an item can be archived.
Never run the archive command without explicit instructions from the user.
Usually this is rather ran as part of review and management of items, not directly within an item, so use:

```bash
wi archive --item ITEM_NAME
```

## Update item state

Avoid updating wi items unless explicitly asked to do so.

```bash
wi label --json
wi label backend reliability
wi label --remove blocked

wi state show --json
wi state set waiting
wi state set working
wi state set backlog
wi state set archived
```

`state set` changes durable lifecycle only. It does not start/stop agents or clean terminal/workspace resources.

## Review work items

Use when asked to inventory, review, or triage current work:

```bash
wi list --json
wi list --json --state working
wi list --json --label +backend --label -blocked
wi list --json --all # To include archived items
```
