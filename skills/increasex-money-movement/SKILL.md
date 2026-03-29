---
name: increasex-money-movement
description: Money movement workflow for `mcp__increasex`. Use when an agent needs to create internal transfers or outbound ACH, Real-Time Payments, FedNow, or wire transfers and must choose the right rail, discover required source or destination identifiers, preview the transfer, and execute only after explicit confirmation.
---

# IncreaseX Money Movement

Use this workflow for outbound movement:

1. Decide the rail before drafting the payload.
   Use `create_account_transfer` for internal movement, `create_ach_transfer` for ACH, `create_real_time_payments_transfer` for RTP, `create_fednow_transfer` for FedNow, and `create_wire_transfer` for wires.
2. Confirm the source and destination primitives.
   Use `resolve_account`, `list_account_numbers`, and `list_external_accounts` to discover missing ids instead of guessing.
3. Prefer stored external accounts when available.
   Use raw routing and account numbers only when the user intentionally wants an ad hoc destination.
4. Call out rail-specific requirements before previewing.
   RTP, FedNow, and some wire flows need `source_account_number_id`. ACH and wire flows may use either an `external_account_id` or raw bank details, depending on the case.
5. Generate a preview first.
   Keep `require_approval=true` when the user wants a queued transfer instead of immediate execution.
6. Before execution, restate the exact rail, amount, source, destination, and approval mode.
7. Execute only with the preview-matched `confirmation_token`.

If a transfer is already queued, switch to the approval workflow instead of creating a duplicate.
