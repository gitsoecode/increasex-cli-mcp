---
name: increase-mcp-tool-development
description: Implement or modify `increasex` CLI and MCP features in this repository. Use when working on Increase API integrations, shared app-layer services, MCP tool schemas, preview-first write flows, masking behavior, or parity between CLI and MCP.
---

# Increase MCP Tool Development

Use this workflow when implementing or modifying `increasex` features:

1. Inspect the official Increase API docs and `increase-go` request and response types before writing code.
2. Prefer typed `increase-go` services when available; only fall back to narrow raw client calls when the public API is documented but the SDK surface is missing.
3. Treat idempotency as a first-class write concern. Preserve or expose `idempotency_key` on write paths and never weaken the existing preview-first confirmation flow.
4. Implement shared app-layer logic first.
5. Add CLI and MCP surfaces only after the shared logic exists.
6. Preserve preview-first behavior for every write path.
7. Normalize outputs and mask sensitive fields by default.
8. Add tests for request mapping, normalization, idempotency-sensitive behavior, and confirmation-token behavior.
9. Update docs when the public surface changes.
