# Repository Guidelines

## Project Structure & Module Organization
- `cmd/notify_slack`: primary CLI entry point for sending messages.
- `cmd/output`: helper generator for piping sample output during development.
- `internal/cli`, `internal/slack`, `internal/throttle`, `internal/config`: reusable packages covering argument parsing, Slack clients, rate control, and configuration.
- `bin/`: build artifacts created by `make`; do not hand-edit.
- Tests live alongside their Go sources with `_test.go`; aggregated coverage persists in `coverage.out`.

## Build, Test, and Development Commands
- `make`: builds both CLIs with git-based version metadata.
- `go install ./cmd/...`: installs binaries into your Go toolchain for local use.
- `make test`: runs `go test -shuffle on -cover -count 10 ./...` to exercise the suite with coverage.
- `make vet`, `make errcheck`, `make staticcheck`: run static analyzers; ensure a clean pass before opening a PR.

## Coding Style & Naming Conventions
- Use standard Go formatting (`go fmt ./...` or editor gofmt-on-save); keep imports managed with `goimports`.
- Keep packages cohesive; place shared helpers under the appropriate `internal/<domain>` directory.
- Name exported items descriptively (`SlackClient`, `ThrottleQueue`) and errors with context via `fmt.Errorf("...: %w", err)`.
- Document non-obvious behavior using short Go doc comments above the declaration.

## Testing Guidelines
- Add `_test.go` files next to the code under test and prefer table-driven test cases.
- Mock Slack calls by swapping the client in `internal/slack`; avoid hitting real APIs in CI.
- Maintain coverage comparable to the current baseline when running `make test`.
- Run `make cover` before submission if the change affects larger flows, reviewing the HTML report when necessary.

## Commit & Pull Request Guidelines
- Follow the existing conventional commit style (`chore(deps): ...`, `fix(deps): ...`, `feature: ...`).
- Keep commits focused, buildable, and include regenerated outputs when required.
- Pull requests must outline the change, note verification commands (`make`, `make test`, lint targets), and link to related issues.
- Provide screenshots or CLI transcripts when modifying user-facing output or documentation.

## Configuration Tips
- Store Slack credentials in a local `notify_slack.toml` (see README) and avoid committing secrets.
- Use the `-debug` and `-interval` flags to throttle notifications while validating changes locally.
