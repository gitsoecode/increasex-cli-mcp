---
name: increasex
description: Unified skill for operating and extending `increasex`. Use when an agent is working with `mcp__increasex` for balances, account numbers, transfers, approvals, cards, external accounts, programs, events, or documents, or when implementing new IncreaseX CLI and MCP features in this repository.
---

# IncreaseX

Use this as the default playbook for both operator workflows and repository development.

## Operating Rules

1. Resolve context before taking action.
   Prefer read tools such as `list_accounts`, `resolve_account`, `list_account_numbers`, `list_external_accounts`, `list_transfers`, `list_programs`, `list_digital_card_profiles`, `list_events`, and `list_documents` before proposing a write.
2. Keep writes preview-first.
   Call every write tool without `dry_run`, or with `dry_run=true`, first. Only call the same tool with `dry_run=false` after you have a matching `confirmation_token`.
3. Reuse the same effective payload between preview and execution.
   If anything substantive changes, generate a fresh preview.
4. Prefer explicit `idempotency_key` values on writes.
   Suggest one unless the user is intentionally replaying a prior preview or the workflow already provides a stable key.
5. Keep sensitive values masked by default.
   Prefer `retrieve_card_details` over `retrieve_card_sensitive_details`. Do not ask for or expose PAN, CVV, PIN, or full account numbers unless the user explicitly requests that level of access.
6. Require explicit user confirmation before money movement or irreversible changes.
   This includes transfers, approvals, cancellations, account closure, disabling account numbers, and PIN updates.
7. Prefer the narrowest resource-shaped tool.
   Use `create_account_transfer` and `create_ach_transfer` instead of compatibility aliases when possible.

## Operator Workflows

### Cash Ops

1. Discover the working account set with `resolve_account` or `list_accounts`.
2. Pull balances with `get_balance`.
3. Review recent movement with `list_recent_transactions`.
   Use `since` and `until` when the user gives explicit bounds; otherwise explain the default range you used.
4. Review pending or recent outbound activity with `list_transfers`.
5. Use `retrieve_transfer` only when a specific transfer needs detail beyond the summary list.
6. If program or card-profile context matters, use `list_programs` or `retrieve_program`.
7. Present summaries in plain language: account, available balance, current balance, notable recent transactions, and queued or pending transfers.

### Account Numbers And Routing

1. Start with `list_account_numbers`.
   Filter by `account_id` when the account is known.
2. Use `retrieve_account_number` for the default masked read of one routing setup.
3. Use `retrieve_account_number_sensitive_details` only when the user explicitly needs the full unmasked account number.
4. Recommend a dedicated account number when the user wants cleaner reconciliation, separate vendor or investor instructions, or isolated inbound controls.
5. Use `create_account_number` in preview mode first.
   Include `inbound_ach.debit_status` and `inbound_checks.status` only when the user has a real need for those controls.
6. Use `disable_account_number` only after confirming the exact account number id and operational impact.
7. When a workflow needs a source account number for RTP, FedNow, or wire flows, discover one here before moving to transfer execution.

### Money Movement

1. Decide the rail before drafting the payload.
   Use `create_account_transfer` for internal movement, `create_ach_transfer` for ACH, `create_real_time_payments_transfer` for RTP, `create_fednow_transfer` for FedNow, and `create_wire_transfer` for wires.
2. Confirm the source and destination primitives.
   Use `resolve_account`, `list_account_numbers`, and `list_external_accounts` to discover missing ids instead of guessing.
3. Prefer stored external accounts when available.
   Use raw routing and account numbers only when the user intentionally wants an ad hoc destination.
4. Call out rail-specific requirements before previewing.
   RTP, FedNow, and some wire flows need `source_account_number_id`. ACH and wire flows may use either an `external_account_id` or raw bank details.
5. Generate a preview first.
   Keep `require_approval=true` when the user wants a queued transfer instead of immediate execution.
6. Before execution, restate the exact rail, amount, source, destination, and approval mode.
7. Execute only with the preview-matched `confirmation_token`.

### Approvals

1. Start with `list_transfer_queue` for the relevant rail.
2. If more detail is needed, call `retrieve_transfer` with the same rail and exact `transfer_id`.
3. Summarize the queued action before doing anything else: rail, amount, source, destination, status, and whether cancellation is safer than approval.
4. Preview the decision with `approve_transfer` or `cancel_transfer`.
5. Only execute after the user confirms the exact transfer id and rail.
6. If the transfer is no longer pending approval, stop and re-check current state instead of forcing the action.

### Cards

1. Start with `list_cards`.
2. Prefer `retrieve_card_details` for routine inspection.
   Only use `retrieve_card_sensitive_details` when the user explicitly asks for PAN, CVV, or PIN-level access.
3. Use `list_digital_card_profiles` or `retrieve_digital_card_profile` before setting `digital_wallet.digital_card_profile_id` on a new card.
4. Use `retrieve_program` when a program’s default digital card profile or commercial terms matter.
5. Use `create_card_details_iframe` when the user needs a secure handoff to view card details without exposing them directly in chat.
6. Keep `create_card` and `update_card_pin` preview-first and avoid repeating sensitive values in summaries or logs.

## Development Workflow

1. Inspect the official Increase API docs and `increase-go` request and response types before writing code.
2. Prefer typed `increase-go` services when available; only fall back to narrow raw client calls when the public API is documented but the SDK surface is missing.
3. Implement shared app-layer logic first.
4. Add CLI and MCP surfaces only after the shared logic exists.
5. Preserve preview-first behavior for every write path.
6. Normalize outputs and mask sensitive fields by default.
7. Add tests for request mapping, normalization, idempotency-sensitive behavior, and confirmation-token behavior.
8. Update docs when the public surface changes.
