---
name: increasex-operator-guardrails
description: Safe operating procedure for `mcp__increasex` tasks. Use when an agent is reading balances, routing details, transfers, approvals, cards, external accounts, programs, events, or documents through IncreaseX and needs the default safety rules for preview-first writes, masking, confirmation tokens, and idempotency.
---

# IncreaseX Operator Guardrails

Apply these rules on every `mcp__increasex` task:

1. Resolve context before taking action.
   Prefer read tools such as `list_accounts`, `resolve_account`, `list_account_numbers`, `list_external_accounts`, `list_transfers`, `list_programs`, `list_digital_card_profiles`, `list_events`, and `list_documents` before proposing a write.
2. Keep writes preview-first.
   Call every write tool without `dry_run` or with `dry_run=true` first. Only call the same tool with `dry_run=false` after you have a matching `confirmation_token`.
3. Do not improvise execution payloads.
   Reuse the same effective payload between preview and execution. If anything substantive changes, generate a fresh preview.
4. Prefer explicit `idempotency_key` values on writes.
   Suggest one unless the user is intentionally replaying a prior preview or the workflow already provides a stable key.
5. Keep sensitive values masked by default.
   Prefer `retrieve_card_details` over `retrieve_card_sensitive_details`. Do not ask for or expose PAN, CVV, PIN, or full account numbers unless the user explicitly requests that level of access.
6. Require explicit user confirmation before money movement or irreversible changes.
   This includes transfers, approvals, cancellations, account closure, disabling account numbers, and PIN updates.
7. Use the narrowest tool that fits the task.
   Prefer resource-shaped tool names such as `create_account_transfer` and `create_ach_transfer` instead of compatibility aliases.
8. Explain operational risk briefly when it matters.
   Mention the rail, amount, source, destination, and approval state before execution so the user can confirm the exact action.

When blocked by missing identifiers, use discovery tools instead of guessing IDs.
