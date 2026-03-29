# Increase MCP Tool Development

Use this workflow when implementing or modifying `increasex` features:

1. Inspect the official Increase API docs and `increase-go` request and response types before writing code.
2. Implement shared app-layer logic first.
3. Add CLI and MCP surfaces only after the shared logic exists.
4. Preserve preview-first behavior for every write path.
5. Normalize outputs and mask sensitive fields by default.
6. Add tests for request mapping, normalization, and confirmation-token behavior.
7. Update docs when the public surface changes.
