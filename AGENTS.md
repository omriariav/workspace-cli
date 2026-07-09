# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` holds Cobra command implementations (one file per service command).
- `internal/` contains core packages: `auth/`, `client/`, `config/`, `printer/`.
- `main.go` and `cmd/gws/main.go` are entry points for `go run` and the CLI.
- `bin/` is the local build output (`make build`).

## Build, Test, and Development Commands
- `make build`: builds `./bin/gws`.
- `make run ARGS="gmail list --max 5"`: runs the CLI with arguments.
- `make test`: runs unit tests (`go test ./...`).
- `make test-race`: runs tests with the race detector.
- `make vet`: runs `go vet` for static analysis.
- `make fmt`: formats code with `gofmt -s -w .`.
- `make tidy`: tidies `go.mod`/`go.sum`.

## Coding Style & Naming Conventions
- Go code follows standard formatting (`gofmt`); use tabs via `gofmt`.
- Package names are short and lowercase; filenames are `snake_case.go` where needed.
- CLI flags follow Cobra conventions (e.g., `--format`, `--calendar-id`).

## Testing Guidelines
- Primary framework: Go `testing` package.
- Keep tests close to the package under `internal/` or `cmd/` as needed.
- Name tests with `TestXxx` and table-driven subtests where appropriate.
- Run `make test` before submitting changes; use `make test-race` for concurrency changes.

## Commit & Pull Request Guidelines
- Commit messages are short, imperative, and unprefixed (e.g., "Add Makefile and cmd/gws entry point").
- PRs should include a clear description, the rationale, and testing notes.
- Link related issues if applicable; include screenshots only for user-visible CLI output changes.

## Release Safety
- Release work is PR-only. Do not push release commits, version bumps, tags, or GitHub releases directly from local `main`.
- Before a release is merged or published, capture evidence for the PR URL, green GitHub checks on the latest PR commit, and the Codex PR review comment.
- Run `make release VERSION=x.y.z` only from a clean, up-to-date `main` after the release PR has merged. The release tag must point at the merged PR commit on `main`.
- If a release starts from an unreviewed commit or missing CI/review evidence, stop immediately, delete any draft release/tag that was created, restore `main` with a revert if needed, and reopen the release through a PR.

## Security & Configuration Notes
- Credentials live outside the repo in `~/.config/gws/config.yaml` and `~/.config/gws/token.json`.
- Do not commit secrets; prefer `GWS_CLIENT_ID`/`GWS_CLIENT_SECRET` env vars for local runs.
