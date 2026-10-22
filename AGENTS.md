# Repository Guidelines

## Purpose and Ownership

MCPProxy provides MCP tool discovery and dispatch, a local REST API, and a web/tray interface. This root is the canonical repository contract. It owns `cmd/`, `frontend/`, `web/`, `native/`, `tests/`, `scripts/`, `docs/`, and `specs/` except for the scoped contracts in the Child DOX Index. Read [internal/AGENTS.md](internal/AGENTS.md) for Go implementation work and [.github/AGENTS.md](.github/AGENTS.md) for workflow changes.

For the maintained agent workflow, use `~/.agents/instructions/WORKFLOW.md` or its [canonical fallback](https://github.com/antongulin/baseline/blob/main/baseline/02-agents.md). Keep task evidence outside stable repository instructions.

## Read and Maintain the Contracts

- Before editing, read this root and every `AGENTS.md` along the path to the target. A child adds local detail and cannot weaken the repository contract. Report conflicting instructions before affected work.
- After a meaningful change, reconcile the nearest owning contract, affected parent indexes, and behavior documentation. A documentation pass may leave accurate files unchanged.
- Add a child only for a durable ownership boundary, using Purpose, Ownership, Local Contracts, Work Guidance, Verification, and Child DOX Index. Keep its verification commands tied to actual project checks.
- Preserve existing legacy instruction files; report conflicts and entry points that do not reach this root. Creating or migrating client-specific copies is separate work.

The contract hierarchy adapts [agent0ai/dox](https://github.com/agent0ai/dox/tree/765ae4ac02cc884eefcd41a3d0f71941721adb89), verified 2026-10-22. Its MIT notice is retained in [docs/dox-LICENSE.txt](docs/dox-LICENSE.txt).

## Project Structure & Module Organization
- `cmd/mcpproxy` hosts the core daemon entrypoint; `cmd/mcpproxy-tray` builds the CGO-based tray binary.
- Runtime logic, HTTP handlers, storage, and upstream orchestration live in `internal/` (`internal/runtime`, `internal/httpapi`, `internal/upstream`, `internal/storage`).
- Vue/Tailwind assets are kept in `frontend/`; bundled static output is embedded from `web/`.
- Integration and regression assets reside under `tests/` and `scripts/`; Go unit tests sit beside source files.

## Build, Test, and Development Commands
- `make build` — regenerate OpenAPI, build/copy frontend assets, then compile the proxy and tray binaries for the host platform.
- `go build ./cmd/mcpproxy` — quick core rebuild; `GOOS=darwin CGO_ENABLED=1 go build ./cmd/mcpproxy-tray` validates the macOS tray target.
- `npm install && npm run build` inside `frontend/` — install web deps and emit production assets.
- CI uses `npm ci` with `frontend/package-lock.json`. Copy `frontend/dist/` to `web/frontend/dist/` for a binary with the built UI; a fresh checkout embeds only the tracked placeholder/fallback.
- `scripts/run-web-smoke.sh` — spin up the proxy and execute the Playwright smoke suite.
- `scripts/verify-api.sh` — exercise the `/api/v1` REST surface with curated curl calls.

## Coding Style & Naming Conventions
- Run `gofmt` (tabs, goimports defaults) on all Go sources; prefer descriptive package-level names (`runtime`, `upstream`, `storage`).
- TypeScript/Vue files follow existing formatting conventions; components use `PascalCase.vue`, composables use `useThing.ts`. The frontend `npm run lint` script currently reports a skipped lint migration; it does not check formatting or code. Use `npm run type-check` and the relevant tests for verification.
- Configuration and DTO structs live in shared packages (`internal/contracts`, `internal/httpapi`); avoid anonymous maps between layers.

## Testing Guidelines
- Unit tests: `go test ./internal/...` (set `GOCACHE`/`GOMODCACHE` to workspace paths when sandboxed).
- Targeted suites: `go test ./internal/server -run TestMCP -v` for lifecycle checks; `npm run test -- --run` inside `frontend/` for a finite unit-test run.
- End-to-end: `scripts/run-e2e-tests.sh` after ensuring `@modelcontextprotocol/server-everything` is warmed and reachable.
- Name Go tests `TestFeatureScenario`; snapshot or fixture data belong under `tests/` with explicit prefixes.

## Commit & Pull Request Guidelines
- Use concise, imperative commit messages (e.g., `Fix upstream disable locking`); avoid AI co-author tags.
- PR descriptions should summarize impact, list verification commands, and link relevant P# items or issues; attach UI screenshots or log excerpts when behavior changes.
- Keep change scopes bounded to one surface (runtime vs. tray vs. web); document follow-up ideas in `IMPROVEMENTS.md`.

## Security & Configuration Tips
- Never hardcode secrets; load them via the tray secure store or environment lookups in `internal/secret`.
- When editing configs, prefer `runtime.SaveConfiguration()` flows so disk state and in-memory state stay aligned; regenerated files land in `~/.mcpproxy/`.

## Active Technologies
- `go.mod` requires Go 1.25.5; the existing CI workflows select the Go 1.25 line. The default build is Personal edition; `-tags server` selects Server edition. Keep `go.mod`, `go.sum`, and frontend lockfiles reproducible.
- BBolt (storage), Chi router (HTTP), Zap (logging), regexp (stdlib), existing ActivityService (026-pii-detection)
- BBolt database (`~/.mcpproxy/config.db`) - ActivityRecord.Metadata extension (026-pii-detection)

## Recent Changes
- 026-pii-detection: Added Go 1.24 (toolchain go1.24.10) + BBolt (storage), Chi router (HTTP), Zap (logging), regexp (stdlib), existing ActivityService

## Child DOX Index

- [internal/AGENTS.md](internal/AGENTS.md): Go configuration, persistence, runtime, transport, and shared API contracts.
- [.github/AGENTS.md](.github/AGENTS.md): CI, review, packaging, and publication workflow contracts.

The existing cockpit role instructions remain scoped instruction assets under `specs/`; they do not select a worker or grant actions for an unrelated task:

- Spec 045: [backend](specs/045-paperclip-cockpit/agent-instructions/backend-engineer/AGENTS.md), [CEO](specs/045-paperclip-cockpit/agent-instructions/ceo/AGENTS.md), [critic](specs/045-paperclip-cockpit/agent-instructions/critic/AGENTS.md), [frontend](specs/045-paperclip-cockpit/agent-instructions/frontend-engineer/AGENTS.md), [macOS](specs/045-paperclip-cockpit/agent-instructions/macos-engineer/AGENTS.md), [QA](specs/045-paperclip-cockpit/agent-instructions/qa-tester/AGENTS.md), [release](specs/045-paperclip-cockpit/agent-instructions/release-engineer/AGENTS.md).
- Spec 064: [shared](specs/064-glass-cockpit/agent-instructions/_shared/AGENTS.md), [backend](specs/064-glass-cockpit/agent-instructions/backend-engineer/AGENTS.md), [CEO](specs/064-glass-cockpit/agent-instructions/ceo/AGENTS.md), [Codex reviewer](specs/064-glass-cockpit/agent-instructions/codex-reviewer/AGENTS.md), [engineer](specs/064-glass-cockpit/agent-instructions/engineer/AGENTS.md), [frontend](specs/064-glass-cockpit/agent-instructions/frontend-engineer/AGENTS.md), [Kimi reviewer](specs/064-glass-cockpit/agent-instructions/kimi-reviewer/AGENTS.md), [macOS](specs/064-glass-cockpit/agent-instructions/macos-engineer/AGENTS.md), [QA](specs/064-glass-cockpit/agent-instructions/qa-tester/AGENTS.md).
