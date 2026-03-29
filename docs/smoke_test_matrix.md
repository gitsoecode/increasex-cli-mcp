# Smoke Test Matrix

Use this checklist to smoke-test CLI and MCP parity against sandbox credentials.

## Accounts

- CLI: `increasex accounts`
- CLI: `increasex accounts create --dry-run`
- CLI: `increasex accounts create`
- CLI: `increasex accounts create-number --dry-run`
- CLI: `increasex account-numbers`
- CLI: `increasex account-numbers retrieve --account-number-id ...`
- CLI: `increasex account-numbers disable --account-number-id ... --dry-run`
- CLI: `increasex accounts close --dry-run`
- MCP: `list_accounts`
- MCP: `list_account_numbers`
- MCP: `retrieve_account_number`
- MCP: `create_account`
- MCP: `create_account_number`
- MCP: `disable_account_number`
- MCP: `close_account`

## Balances and Transactions

- CLI: `increasex balance --account-id ...`
- CLI: `increasex transactions --account-id ...`
- CLI: `increasex transactions --account-id ... --period last-30d`
- CLI: `increasex transactions --account-id ... --since 2026-03-01T00:00:00Z --until 2026-03-15T23:59:59Z`
- MCP: `get_balance`
- MCP: `list_recent_transactions`
- MCP: `list_recent_transactions` with `since` and `until`
- MCP: `list_programs`
- MCP: `retrieve_program`
- MCP: `list_events`
- MCP: `retrieve_event`
- MCP: `list_documents`
- MCP: `retrieve_document`

## Cards

- CLI: `increasex cards`
- CLI: `increasex cards retrieve --card-id ...`
- CLI: `increasex cards details --card-id ...`
- CLI: `increasex cards create-details-iframe --card-id ...`
- CLI: `increasex cards create --dry-run`
- CLI: `increasex cards update-pin --card-id ... --dry-run`
- MCP: `list_cards`
- MCP: `list_digital_card_profiles`
- MCP: `retrieve_digital_card_profile`
- MCP: `retrieve_card_details`
- MCP: `retrieve_card_sensitive_details`
- MCP: `create_card_details_iframe`
- MCP: `create_card`
- MCP: `update_card_pin`

## External Accounts

- CLI: `increasex external-accounts`
- CLI: `increasex external-accounts retrieve --external-account-id ...`
- CLI: `increasex external-accounts create --dry-run`
- CLI: `increasex external-accounts update --external-account-id ... --dry-run`
- MCP: `list_external_accounts`
- MCP: `retrieve_external_account`
- MCP: `create_external_account`
- MCP: `update_external_account`

## Transfers

- CLI: `increasex transfer internal --dry-run`
- CLI: `increasex transfer external --rail ach --dry-run`
- CLI: `increasex transfer external --rail real_time_payments --dry-run`
- CLI: `increasex transfer external --rail fednow --dry-run`
- CLI: `increasex transfer external --rail wire --dry-run`
- CLI: `increasex transfer list --rail account`
- CLI: `increasex transfer retrieve --rail ach --transfer-id ...`
- CLI: `increasex transfer queue --rail ach`
- CLI: `increasex transfer approve --rail ach --transfer-id ... --dry-run`
- CLI: `increasex transfer cancel --rail ach --transfer-id ... --dry-run`
- MCP: `create_account_transfer`
- MCP: `create_ach_transfer`
- MCP: `create_real_time_payments_transfer`
- MCP: `create_fednow_transfer`
- MCP: `create_wire_transfer`
- MCP: `list_transfers`
- MCP: `retrieve_transfer`
- MCP: `list_transfer_queue`
- MCP: `approve_transfer`
- MCP: `cancel_transfer`

## TTY Menus

- CLI: run `increasex` in a TTY and confirm the root menu appears
- CLI: run `increasex transfer` in a TTY and confirm the transfer menu appears
- CLI: confirm Back and Exit options exist in the transfer menu
- CLI: run `increasex accounts` in a TTY and confirm nested account selectors expose Back and Exit options
- CLI: run `increasex cards` and `increasex external-accounts` in a TTY and confirm nested selectors expose Back and Exit options
- CLI: start an interactive write flow with a free-text prompt and confirm typing `back` returns to the previous step
- CLI: start an interactive write flow with a free-text prompt and confirm typing `exit` returns to the shell
- CLI: confirm non-TTY or `--json` invocation skips menus

## MCP Discovery

- MCP: `describe_capabilities`
- Ask Codex what tools are available and confirm the grouped descriptions are legible
