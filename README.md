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
- human CLI commands for auth, accounts, balances, transactions, transfers, external accounts, cards, and `mcp serve`
- TTY menus for `increasex` and `increasex transfer`
- interactive selectors for account, card, external-account, and transfer-driven flows
- preview-first write flows with confirmation tokens
- transfer approval queue actions in both CLI and MCP
- a local MCP server with a curated, grouped tool surface for Codex and other MCP-capable hosts

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
./increasex
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
sudo mv increasex /usr/local/bin/increasex
```

On many macOS systems, writing to `/usr/local/bin` requires `sudo`.

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
eval "$(./increasex auth export --confirm)"
```

`auth export` prints the raw API key to stdout before you `eval` it. Treat it as an intentional, secret-bearing escape hatch rather than the default auth path.

### Check Auth

```bash
./increasex auth status
./increasex auth whoami
```

`auth status` reports whether a durable file credential is available, whether a Keychain credential is available, and whether MCP is ready without needing shell exports. `auth whoami` validates the current credential and shows the active profile, environment, token source, and resolved entity context without printing the API key.

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
./increasex transactions --account-id account_xxx --period last-7d
./increasex transactions --account-id account_xxx --since 2026-03-01T00:00:00Z --until 2026-03-15T23:59:59Z
./increasex transactions --account-id account_xxx --category account_transfer_intention
```

When no time period is supplied, `transactions` defaults to the last 30 days.

List cards:

```bash
./increasex cards
./increasex cards --account-id account_xxx
```

Retrieve masked card details:

```bash
./increasex cards retrieve --card-id card_xxx
```

Retrieve card details:

```bash
./increasex cards details --card-id card_xxx
```

List external accounts:

```bash
./increasex external-accounts
./increasex external-accounts --status active
```

Retrieve an external account:

```bash
./increasex external-accounts retrieve --external-account-id external_account_xxx
```

List transfers:

```bash
./increasex transfer list --rail account
./increasex transfer list --rail ach --status pending_approval
```

Retrieve a transfer:

```bash
./increasex transfer retrieve --rail wire --transfer-id wire_transfer_xxx
```

List the approval queue:

```bash
./increasex transfer queue --rail ach
```

## Interactive CLI

When running in a TTY, `increasex` can use interactive selectors instead of requiring every ID up front.

Examples:

```bash
./increasex
./increasex accounts
./increasex balance
./increasex transactions
./increasex transfer
./increasex transfer internal
./increasex transfer external
./increasex external-accounts
./increasex cards
```

Current interactive behavior includes:

- root and transfer menus instead of no-subcommand dead ends
- searchable account selection
- searchable card selection
- searchable external-account selection
- transfer approval queue selection
- action selection after listing accounts
- explicit Back and Exit options in nested interactive selectors
- typed `back` and `exit` support in free-text prompts
- confirmation selection before write execution

If you want to force interactive prompts:

```bash
./increasex --interactive accounts
```

If you use `--json`, interactive UI is suppressed.

## Preview-First Writes

All write commands and MCP tools use a preview-first flow:

- default CLI behavior: preview, confirm, then execute
- `--dry-run` forces preview-only mode
- preview returns a summary and `confirmation_token`
- MCP omitted `dry_run` stays preview-first
- MCP `dry_run=true` previews
- MCP `dry_run=false` only executes when you also provide a valid `confirmation_token`
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
  --rail real_time_payments \
  --rtp-creditor-name "Vendor LLC" \
  --rtp-remittance-information "Invoice 1001" \
  --rtp-source-account-number-id account_number_xxx \
  --rtp-destination-account-number 123456789 \
  --rtp-destination-routing-number 021000021 \
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

### Create an External Account

Preview only:

```bash
./increasex external-accounts create \
  --description "Primary vendor destination" \
  --routing-number 021000021 \
  --account-number 123456789 \
  --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex external-accounts create \
  --description "Primary vendor destination" \
  --routing-number 021000021 \
  --account-number 123456789
```

### Approve a Transfer

Preview only:

```bash
./increasex transfer approve --rail ach --transfer-id ach_transfer_xxx --dry-run
```

Default execute flow with interactive confirmation:

```bash
./increasex transfer approve --rail ach --transfer-id ach_transfer_xxx
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

Discovery tool:

- `describe_capabilities`

Read tools:

- `list_accounts`
- `resolve_account`
- `list_account_numbers`
- `retrieve_account_number`
- `list_programs`
- `retrieve_program`
- `get_balance`
- `list_recent_transactions` with optional `since` and `until` RFC3339 bounds
- `list_events`
- `retrieve_event`
- `list_documents`
- `retrieve_document`
- `list_cards`
- `list_digital_card_profiles`
- `retrieve_digital_card_profile`
- `retrieve_card_details`
- `retrieve_card_sensitive_details`
- `create_card_details_iframe`
- `list_external_accounts`
- `retrieve_external_account`
- `list_transfers`
- `retrieve_transfer`
- `list_transfer_queue`

Write tools:

- `create_account`
- `close_account`
- `create_account_number`
- `disable_account_number`

Example MCP transaction filter input:

```json
{
  "account_id": "account_xxx",
  "since": "2026-03-01T00:00:00Z",
  "until": "2026-03-15T23:59:59Z"
}
```
- `create_account_transfer`
- `create_ach_transfer`
- `create_real_time_payments_transfer`
- `create_fednow_transfer`
- `create_wire_transfer`
- `approve_transfer`
- `cancel_transfer`
- `create_external_account`
- `update_external_account`
- `create_card`
- `update_card_pin`

Compatibility aliases still exist for the older `move_money_*` transfer tool names.

The repository also includes a unified [`increasex` skill](./skills/increasex/SKILL.md) for operating `mcp__increasex` safely and for implementing new IncreaseX features in this repo.

### Install Skills

#### General Install

```bash
npx skills add https://github.com/gitsoecode/increasex-cli-mcp --skill increasex
```

Restart Codex after installing new skills so they are picked up in a fresh session.

#### Claude Code Only

```bash
mkdir -p .claude/skills/increasex
curl -L https://raw.githubusercontent.com/gitsoecode/increasex-cli-mcp/main/skills/increasex/SKILL.md -o .claude/skills/increasex/SKILL.md
```

### MCP Write Pattern

All MCP writes are preview-first:

1. Call the tool without `dry_run`, or with `dry_run=true`
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

- [docs/spec.md](./docs/spec.md)
- [docs/parity_matrix.md](./docs/parity_matrix.md)
- [docs/smoke_test_matrix.md](./docs/smoke_test_matrix.md)
- [AGENTS.md](./AGENTS.md)
