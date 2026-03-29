# IncreaseX

`increasex` is a Go-based wrapper around Increase with:

- a human-friendly CLI
- a local MCP server over `stdio`
- one shared application core for both interfaces
- preview-first write flows for safer money movement and object creation

The project is intentionally built so CLI commands and MCP tools use the same auth, validation, normalization, preview, and execution paths.

## Current Status

This repository currently includes:

- local auth and profile handling
- human CLI commands for auth, accounts, balances, transactions, transfers, cards, and `mcp serve`
- interactive selectors in TTY sessions for account-driven flows
- preview-first write flows with confirmation tokens
- a local MCP server with a curated tool surface for Codex and other MCP-capable hosts

What this is not:

- not a replacement for the official Increase CLI
- not a hosted or remote MCP service
- not a browser UI

## Requirements

- Go 1.26.1+
- an Increase API key
- macOS keychain access if you want secure local credential storage

## Quick Start

Build the binary:

```bash
go build -o increasex ./cmd/increasex
```

Log in with a durable local profile:

```bash
./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY
./increasex auth status
```

Verify the CLI works:

```bash
./increasex accounts
./increasex balance --account-id account_xxx
```

Register the MCP server with Codex:

```bash
codex mcp add increasex -- "$(pwd)/increasex" mcp serve
codex mcp list
```

Then start a fresh Codex session and ask:

```text
List the available MCP tool namespaces in this session.
```

You should see `mcp__increasex`.

## Build

From the repo root:

```bash
go build -o increasex ./cmd/increasex
```

You only need to rebuild when the code changes.

For one-off runs during development:

```bash
go run ./cmd/increasex --help
```

## Install

If you stay in the repo directory, run the binary with:

```bash
./increasex --help
```

If you want to run `increasex` without `./`, place the binary somewhere on your `PATH`, for example:

```bash
mv increasex /usr/local/bin/increasex
```

Or add this repo directory to your shell `PATH`.

## Authentication

`increasex` supports two auth patterns:

1. Recommended: store credentials locally with durable agent support
2. Session-only: print shell exports and `eval` them

Credential resolution order is:

1. explicit flags
2. environment variables
3. durable local credentials file
4. keychain mirror

### Recommended Login

Use the default automatic mode:

```bash
./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY
```

`auto` mode writes a user-only durable credentials file for CLI and MCP use across sessions, and mirrors to Keychain when available.

Store credentials in the durable file only:

```bash
./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY --storage file
```

Store credentials in Keychain only:

```bash
./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY --storage keychain
```

### Session-Only Login

If you do not want to store credentials:

```bash
eval "$(./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY --print-env)"
```

That sets shell environment variables for the current session only.

If you already have stored credentials and want to load them into the current shell intentionally:

```bash
eval "$(./increasex auth export)"
```

### Check Auth

```bash
./increasex auth status
./increasex auth whoami
```

`auth status` reports whether a durable file credential is available, whether a Keychain credential is available, and whether MCP is ready without needing shell exports.

### Log Out

```bash
./increasex auth logout
```

## Global Flags

All top-level commands support:

- `--profile`
- `--env`
- `--json`
- `--interactive`
- `--yes`
- `--debug`
- `--api-key`

### Codex / MCP Flow

The recommended Codex flow is:

```bash
./increasex auth login
./increasex auth status
codex mcp add increasex -- "$(pwd)/increasex" mcp serve
codex
```

Important:

- do not manually run `./increasex mcp serve` for normal Codex usage
- Codex should launch `increasex mcp serve` itself
- start a fresh Codex session after adding or changing the MCP config

That flow continues working across terminal sessions because `increasex` reads the durable local credential file directly.

Examples:

```bash
./increasex --env sandbox accounts
./increasex --json accounts
./increasex --profile finance balance --account-id account_xxx
```

## CLI Usage

## Read Commands

List accounts:

```bash
./increasex accounts
./increasex accounts --status open
./increasex accounts --limit 50
```

Get a balance:

```bash
./increasex balance --account-id account_xxx
```

List transactions:

```bash
./increasex transactions --account-id account_xxx
./increasex transactions --account-id account_xxx --since 2026-03-01T00:00:00Z
./increasex transactions --account-id account_xxx --category account_transfer_intention
```

List cards:

```bash
./increasex cards
./increasex cards --account-id account_xxx
```

Get masked card details:

```bash
./increasex cards get --card-id card_xxx
```

## Interactive CLI

When running in a TTY, `increasex` can use interactive selectors instead of requiring every ID up front.

Examples:

```bash
./increasex accounts
./increasex balance
./increasex transactions
./increasex transfer internal
./increasex transfer external
```

Current interactive behavior includes:

- searchable account selection
- searchable card selection
- action selection after listing accounts
- confirmation selection before write execution

If you want to force interactive prompts:

```bash
./increasex --interactive accounts
```

If you use `--json`, interactive UI is suppressed.

## Preview-First Writes

All write commands still use a preview-first flow, but the human CLI now defaults to execution mode with confirmation:

- default CLI behavior: preview, confirm, then execute
- `--dry-run` forces preview-only mode
- preview returns a summary and `confirmation_token`
- MCP write tools still default to `dry_run=true`
- if you do not pass a token manually in the CLI, the CLI previews first and then prompts you before execution

### Create an Account

Preview only:

```bash
./increasex accounts create --name "Operating" --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex accounts create --name "Operating"
```

### Close an Account

Preview only:

```bash
./increasex accounts close --account-id account_xxx --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex accounts close --account-id account_xxx
```

### Create an Account Number

Preview only:

```bash
./increasex accounts create-number --account-id account_xxx --name "Vendor Receipts" --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex accounts create-number --account-id account_xxx --name "Vendor Receipts"
```

### Internal Transfer

Preview only:

```bash
./increasex transfer internal \
  --from-account-id account_from \
  --to-account-id account_to \
  --amount-cents 5000 \
  --description "Ops funding" \
  --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex transfer internal \
  --from-account-id account_from \
  --to-account-id account_to \
  --amount-cents 5000 \
  --description "Ops funding"
```

### ACH Transfer

Preview only:

```bash
./increasex transfer external \
  --rail ach \
  --account-id account_xxx \
  --amount-cents 5000 \
  --statement-descriptor "VENDOR PAY" \
  --account-number 123456789 \
  --routing-number 021000021 \
  --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex transfer external \
  --rail ach \
  --account-id account_xxx \
  --amount-cents 5000 \
  --statement-descriptor "VENDOR PAY" \
  --account-number 123456789 \
  --routing-number 021000021
```

### RTP Transfer

Preview only:

```bash
./increasex transfer external \
  --rail rtp \
  --creditor-name "Vendor LLC" \
  --remittance-information "Invoice 1001" \
  --source-account-number-id account_number_xxx \
  --destination-account-number 123456789 \
  --destination-routing-number 021000021 \
  --rtp-amount-cents 5000 \
  --dry-run
```

### FedNow Transfer

Preview only:

```bash
./increasex transfer external \
  --rail fednow \
  --fednow-account-id account_xxx \
  --fednow-amount-cents 5000 \
  --fednow-creditor-name "Vendor LLC" \
  --fednow-debtor-name "My Company" \
  --fednow-source-account-number-id account_number_xxx \
  --fednow-account-number 123456789 \
  --fednow-routing-number 021000021 \
  --fednow-remittance "Invoice 1001" \
  --dry-run
```

### Wire Transfer

Preview only:

```bash
./increasex transfer external \
  --rail wire \
  --wire-account-id account_xxx \
  --wire-amount-cents 5000 \
  --wire-beneficiary-name "Vendor LLC" \
  --wire-message-to-recipient "Invoice 1001" \
  --wire-account-number 123456789 \
  --wire-routing-number 021000021 \
  --dry-run
```

### Create a Card

Preview only:

```bash
./increasex cards create --account-id account_xxx --description "Ops card" --dry-run
```

With billing address:

```bash
./increasex cards create \
  --account-id account_xxx \
  --description "Ops card" \
  --billing-line1 "123 Main St" \
  --billing-city "San Francisco" \
  --billing-state "CA" \
  --billing-postal-code "94105"
```

Default execute flow with interactive confirmation:

```bash
./increasex cards create --account-id account_xxx --description "Ops card"
```

## JSON Output

Use `--json` to get stable machine-readable responses:

```bash
./increasex --json accounts
./increasex --json balance --account-id account_xxx
```

Response shape:

```json
{
  "ok": true,
  "request_id": "optional",
  "data": {}
}
```

Errors use:

```json
{
  "ok": false,
  "request_id": "optional",
  "error": {
    "code": "validation_error",
    "message": "human readable message"
  }
}
```

## MCP Usage

`increasex` exposes a local MCP server over `stdio`.

This server is:

- local only
- `stdio` transport only
- intended to be launched by an MCP-capable host such as Codex

### Codex Setup

Register the server:

```bash
codex mcp add increasex -- "$(pwd)/increasex" mcp serve
```

Check that Codex knows about it:

```bash
codex mcp list
```

Then start a fresh Codex session and ask it to list available MCP namespaces or tools.

### Manual Debugging

You can run the server manually:

```bash
./increasex mcp serve
```

If you do that, a blinking cursor is expected. The process is waiting for MCP messages on standard input. This is useful for debugging, but it is not how you normally use the server from Codex.

The agent does not receive your API key. `increasex` resolves auth locally from flags, environment, or stored profiles, then makes Increase API requests itself.

### MCP Tool Surface

Read tools:

- `list_accounts`
- `resolve_account`
- `get_balance`
- `list_recent_transactions`
- `list_cards`
- `retrieve_card_details`

Write tools:

- `move_money_internal`
- `create_account`
- `close_account`
- `create_account_number`
- `move_money_external_ach`
- `move_money_external_rtp`
- `move_money_external_fednow`
- `move_money_external_wire`
- `create_card`

### MCP Write Pattern

All MCP writes are two-step:

1. Call the tool with default `dry_run=true`
2. Receive preview details and a `confirmation_token`
3. Call the same tool again with `dry_run=false` and the same effective payload

That confirmation token is the only intentional cross-call server state in v1.

## Troubleshooting

If `./increasex mcp serve` appears to hang, that is expected. It is waiting for MCP messages on `stdin`.

If Codex does not show `mcp__increasex`:

1. Rebuild the binary:

```bash
go build -o increasex ./cmd/increasex
```

2. Confirm the MCP entry exists:

```bash
codex mcp list
```

3. Start a fresh Codex session.

Already-open Codex sessions do not hot-reload newly fixed MCP servers.

If you see:

```json
{
  "ok": false,
  "error": {
    "message": "auth_error: no credentials found for the selected profile"
  }
}
```

then log in first:

```bash
./increasex auth login --name default --env sandbox --api-key YOUR_INCREASE_API_KEY
./increasex auth status
```

Then retry:

```bash
./increasex accounts
```

If `increasex` is “command not found”, either:

- run `./increasex` from the repo root, or
- move the binary onto your `PATH`

## Development

Run tests:

```bash
GOCACHE=/tmp/increasex-gocache go test ./...
```

Build:

```bash
GOCACHE=/tmp/increasex-gocache go build -o increasex ./cmd/increasex
```

See also:

- [docs/spec.md](/Users/jessevaughan/Projects/Increase_CLI_wrapper_MCP/docs/spec.md)
- [AGENTS.md](/Users/jessevaughan/Projects/Increase_CLI_wrapper_MCP/AGENTS.md)
