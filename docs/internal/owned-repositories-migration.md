# Owned-repository migration

This is an internal record of the completed offline schema 4-to-5 conversion, not a normal CLI workflow or an automatic upgrade path. Current user contracts live in [Repositories](../src/content/docs/concepts/repositories.md), [Workspaces](../src/content/docs/concepts/workspaces.md), and [Delegate work](../src/content/docs/guides/delegation.md).

## Local conversion

The operator-authorized conversion replaced present home and managed claims with fresh managed assignments at the captured HEAD. It preserves the index separately from unstaged changes, including binary patches, deletions, modes, symlinks and untracked files. Absent claims remain absent. Archived items retain lifecycle, descriptions, history and source commits.

Each replacement has a new claim identity and a higher generation. Old process/terminal ownership and activity observations are dropped rather than rebound to a new path. Old worktrees and external checkouts are retained. The script never stashes, commits, resets, cleans or deletes source files.

The one-shot converter lives in `internal/migraterepositories/` during deployment. It consumes a daemon-exported inventory, creates and verifies stores/worktrees, records a protected replay plan, then converts a staging database. The schema change is 4 to 5. Prior events, command outcomes, import digests, source offsets and native content survive. New onboarding and per-item migration events record the former repository and checkout.

Two archived items have stale source paths. Each has one same-origin local candidate containing its created-from commit. The old `please-fix` implementation branch is already absent from that candidate. The migration reports missing historical branches; it never fabricates a recovered tip.

Ignored `.env` files, build caches, Git LFS data and other auxiliary stores are not implicitly copied. Submodules and unmerged indexes require a separate explicit plan. The sources remain available for inspecting those dependencies. Git dirty-state equivalence is not a claim that the build environment is identical.

The live cutover completed after an outside-Wi operator stopped all runtimes. It converted 29 repositories, 92 items, and 39 replacement assignments, retaining existing events, command outcomes, import offsets, and native content. No agents were restarted automatically. Seven already-missing historical implementation branches were reported, not fabricated.

Any future conversion must stop verified writers, hold the database lock through the final checkpoint and replacement, and preserve a matching executable and native-content backup. Never rebind a live process to a replacement claim.

## Recovery

Back up `wi.db`, owned Git stores, native descriptions/conversations and relevant source Git state. Keep a matching executable with each database checkpoint. Moving the data root also requires Git worktree metadata repair because `.git` files contain concrete paths. Do not add symlink repository fixes.

Rollback stops the new daemon and restores a matching database/executable pair. It does not alter external checkouts. Prepared replacement stores/worktrees are retained for inspection until the operator confirms cleanup. A rollback after new domain activity must account for that activity rather than silently discard it.
