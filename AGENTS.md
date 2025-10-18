# Repository Guidelines

## Project Structure & Module Organization
Code entry lives in `cmd/jjui/main.go`; it wires up UI packages under `internal/ui/**`. Domain logic is grouped by feature: revset handling in `internal/ui/revset`, operations under `internal/ui/operations`, and Jujutsu adapters in `internal/jj`. Shared helpers (config, screen, parser) sit in `internal/{config,screen,parser}`. Tests and fixtures live in `test/` with golden data under `test/testdata`. GitHub workflows and release metadata live in `.github/`, while reproducible dev shells are defined in `flake.nix` and `nix/`.

## Build, Test, and Development Commands
- `go run ./cmd/jjui` – launch the TUI against your current `jj` repo.
- `go build ./cmd/jjui` – verify the binary compiles; add `-o bin/jjui` to test distribution builds.
- `go test ./...` – run unit tests, including parser tests that use `test/testdata`.
- `nix develop` – optional: drop into a shell with pinned Go and toolchain versions.

## Coding Style & Naming Conventions
Use Go 1.24 tooling; run `gofmt` or `go fmt ./...` before committing. Follow standard Go naming: exported API uses PascalCase, internals use camelCase, package-level files use snake_case (for example, `revision_menu.go`). Keep UI state structs in the `model` pattern already used in `internal/ui/**`. Avoid introducing third-party dependencies without discuss first (open an issue or PR note).

## Testing Guidelines
Prefer table-driven `*_test.go` files colocated with the code under test; integration helpers in `test/` are shared across packages. When adding parser or UI logic, extend fixtures in `test/testdata` so golden assertions remain readable. Run `go test ./...` before submitting; new features should carry tests or rationale for omissions. Keep `DEBUG=1 go run ./cmd/jjui` handy to capture `debug.log` when reproducing issues.

## Commit & Pull Request Guidelines
Match the existing conventional format (`type(scope): summary`), e.g., `feat(ui/oplog): add op filtering`. Keep messages in the imperative and focused on one change set. For PRs, include a concise summary, reproduction steps or screenshots for UI tweaks, and note any dependency or `nix/vendor-hash` updates. Link issues when applicable and call out follow-up work explicitly.

## Environment & Security Notes
Target `jj` v0.26+ during manual testing. Store credentials outside the repo; the app shells out to `jj`, so verify any new command invocations respect user configuration and avoid writing outside the workspace.
