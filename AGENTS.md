# AGENTS.md

- Use Go for this repository.
- Do not shell out to the existing `increase` binary.
- CLI and MCP must share one app-layer core.
- All writes are preview-first.
- Sensitive fields must be masked by default.
- MCP v1 uses local stdio only.
- Read `docs/spec.md` before implementing features.
- Run tests and lint before finishing work.
