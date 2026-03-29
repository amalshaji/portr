# Repository Guidelines

## Project Structure & Module Organization
The repo root is flattened. Go entrypoints live in `cmd/portr` and `cmd/portrd`, with shared application code under `internal/`. Use `internal/server/admin` for admin server work, `internal/server/admin/web-v2` for the React + shadcn admin UI, and `internal/client/dashboard/ui` for the Svelte dashboard. Keep integration coverage in `tests/`, database changes in `migrations/`, and product docs in `docs-v2/`. Do not add new code under the legacy `tunnel/` path.

## Build, Test, and Development Commands
Use `go build ./...` to compile all Go packages and binaries. Run `go test ./...` before opening a PR. Build the CLI with `make buildcli`, which outputs `./portr`.

For the Svelte dashboard:
- `pnpm --dir internal/client/dashboard/ui install`
- `pnpm --dir internal/client/dashboard/ui dev`
- `pnpm --dir internal/client/dashboard/ui build`
- `pnpm --dir internal/client/dashboard/ui check`

For the admin UI:
- `bun --cwd internal/server/admin/web-v2 run build`
- `bun --cwd internal/server/admin/web-v2 run lint`

## Coding Style & Naming Conventions
Follow `gofmt` defaults for Go. Keep package names lowercase, exported identifiers in PascalCase, and test files named `*_test.go`. In TypeScript, React, and Svelte files, match the existing two-space indentation. Follow current frontend structure such as `src/components/ui`, `src/pages`, and `src/lib`. Use `github.com/charmbracelet/log` for structured logging. Keep comments minimal and in English.

## Testing Guidelines
Add or update tests for every behavior change. Place package-level Go tests beside the code they cover, and broader server or admin coverage under `tests/server/*_test.go`. If you change `internal/client/dashboard/ui`, rebuild `internal/client/dashboard/ui/dist` in the same change so embedded assets stay in sync.

## Commit & Pull Request Guidelines
Recent history favors short, imperative commit messages, sometimes with a prefix such as `fix:`. Examples include `fix: surface ssh listener failures after startup` and `Add portr logs command`. Keep commits scoped, self-review before requesting feedback, summarize the checks you ran, link related issues when relevant, and include screenshots for UI changes.
