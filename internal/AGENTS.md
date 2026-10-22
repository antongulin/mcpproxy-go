# Go Implementation Contracts

## Purpose

Implement the proxy runtime, upstream connections, persistent state, and public MCP/REST behavior. Read the [repository contract](../AGENTS.md) first.

## Ownership

- `config/`: configuration types, JSON loading/saving, validation, copying, and merge semantics.
- `storage/`: BBolt records and synchronous/asynchronous projections.
- `runtime/`: configuration lifecycle, services, events, and upstream orchestration.
- `server/`: MCP tools, dispatch gates, input/output handling, and HTTP server lifecycle.
- `httpapi/` and `contracts/`: REST requests, shared DTOs, and conversions.
- Other packages retain their existing responsibilities, including `upstream/`, `oauth/`, `secret/`, `index/`, and `serveredition/`.

## Local Contracts

- A server configuration field may cross `config.ServerConfig`, `storage.UpstreamRecord`, synchronous manager reads/writes, asynchronous writes, MCP management builders, and REST DTO conversions. Trace the actual consumers and preserve omitted-versus-explicit update semantics.
- `runtime.SaveConfiguration()` rebuilds the server list from storage. A field stored only in the JSON configuration can disappear during a later save. Verify persistence through save, reopen, unrelated updates, and reload when changing that contract.
- Tool dispatch uses the shared `toolGate` and its captured persisted server record. Preserve authorization, identity, quarantine, and enabled-tool decisions across dispatch paths.
- `server/mcp_input_validation.go` validates indexed input schemas before dispatch and renders argument errors; `server/mcp.go` also handles upstream invalid-params errors. Keep public schema, error shape, and dispatch behavior aligned with their existing tests when changing either path.
- Keep administrative projections consistent with shared DTOs and existing credential masking. Use synthetic fixtures for secrets and request data.
- Preserve Personal and Server build boundaries. Server-only behavior uses the existing `server` build tag; a default binary is Personal edition.

## Work Guidance

Use targeted source navigation and read the actual call sites before changes. Keep Go tests beside the owning package. Prefer temporary BBolt/config files and loopback HTTP fixtures to shared service state. Isolate test processes and data directories from running user services.

## Verification

- Run focused existing package tests for the changed behavior, then the applicable repository CI gates. Root commands and `.github/workflows/unit-tests.yml` describe the broader checks.
- `go test ./internal/server -run 'Test(PreDispatchValidation|CreateDetailedErrorResponse|ClassifyUpstreamInvalidParams)' -count=1` covers existing argument validation and error classification with local fixtures.
- For configuration or DTO changes, select the relevant tests in `config/`, `storage/`, `runtime/`, `httpapi/`, and `contracts/` from the actual affected consumers. Check synchronous and asynchronous persistence when both carry the field.
- Run `gofmt` on changed Go files and `git diff --check`. Builds require the root's frontend asset preparation when verifying embedded UI behavior.

## Child DOX Index

None. This contract owns the `internal/` subtree.
