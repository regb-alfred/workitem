# Web integration with main

Rebased onto `e6ef6cf` without creating commits. All earlier local changes to `agentdriver/`, `internal/agentdriver/`, `internal/app/`, and the existing driver-protocol documentation were replaced by main's implementations.

The web adapter uses main's readable-history and content APIs, title generation, verified runtime controls, idle guards, fresh-runtime startup guard, and archive partial outcomes. Project selection comes from the registered-repository API, including empty repositories. Creation resolves the owned bare store, integration-branch base commit, and repository labels through the daemon. Names are display values, never repository identity or filesystem-derived paths.

## Remaining core gap

Main's `item.title.set` changes the display title only. It cannot finalize a provisional item's slug and checkout branch together after returning its permanent ID. Waiting for model-generated naming before creation would delay the durable acknowledgement and prevent an immediate conversation link.

The retained extension is `item.naming.finalize` in `internal/coordinator/domain.go`, with tests in `domain_naming_test.go`. It atomically updates title, unique slug, and implementation-branch specification using an expected revision, preserving the permanent ID and immutable source commit. It rejects existing or previously materialized claims, primary conversations, waiting/archived items, repository-home records, and explicitly chosen branches. It performs no Git operations and does not rename an actual branch or native conversation. Once materialized, ordinary title edits preserve slug and branch.

This is the only core-engine change relative to main. There are no driver changes. The HTTP server, background receipts, and UI remain access-layer code; native agent content is not added to retained events or projections.

## Verification

- `go test -short ./...`
- `go test -race ./internal/web ./internal/cli ./internal/coordinator`
- `node --test internal/web/ui_test.cjs`
- `go vet ./internal/web ./internal/cli ./internal/coordinator`

Manual demo runs use a fresh isolated HOME/XDG root, Wi-owned registered fixture repositories, and a fake Pi executable. Neither the installed binary nor the operator's live daemon is replaced.
