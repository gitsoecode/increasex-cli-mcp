---
name: increasex-account-numbers-and-routing
description: Account-number and routing workflow for `mcp__increasex`. Use when an agent needs to inspect routing details, create or disable account numbers, explain inbound ACH or check settings, or decide whether a dedicated account number should be created for a vendor, investor, product flow, or reconciliation workflow.
---

# IncreaseX Account Numbers And Routing

Use this workflow for inbound-routing tasks:

1. Start with `list_account_numbers`.
   Filter by `account_id` when the account is known.
2. Use `retrieve_account_number` when the user needs the masked summary for one specific routing setup.
3. Explain masked outputs clearly.
   Routing numbers may be fully visible, while account numbers remain masked by default. Treat the masked value as confirmation, not as a shareable full credential.
4. Recommend a dedicated account number when the user wants cleaner reconciliation, separate vendor/investor instructions, or isolated inbound controls.
5. Use `create_account_number` in preview mode first.
   Include `inbound_ach.debit_status` and `inbound_checks.status` only when the user has a real need for those controls.
6. Use `disable_account_number` only after confirming the exact account number id and the operational impact.
7. When a workflow needs a source account number for RTP, FedNow, or wire flows, use this skill to discover one before handing off to money movement.

Keep guidance operational: why this account number exists, what inbound controls are set, and what downstream workflow it supports.
