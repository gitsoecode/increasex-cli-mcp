---
name: increasex-approval-desk
description: Transfer approval workflow for `mcp__increasex`. Use when an agent needs to review pending approvals, inspect a queued transfer, and preview or execute an approval or cancellation for the exact rail and transfer id the user confirms.
---

# IncreaseX Approval Desk

Use this workflow for queued transfers:

1. Start with `list_transfer_queue` for the relevant rail.
2. If more detail is needed, call `retrieve_transfer` with the same rail and the exact `transfer_id`.
3. Summarize the queued action before doing anything else:
   rail, amount, source account, destination, status, and whether cancellation is a safer path than approval.
4. Preview the decision with `approve_transfer` or `cancel_transfer`.
5. Only execute after the user confirms the exact transfer id and rail.
6. If the transfer is no longer pending approval, stop and re-check current state instead of forcing the action.

Do not create a replacement transfer unless the user explicitly asks for a new one.
