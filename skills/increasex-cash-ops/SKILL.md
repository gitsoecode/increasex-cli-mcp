---
name: increasex-cash-ops
description: Cash operations workflow for `mcp__increasex`. Use when an agent needs to summarize account posture, resolve accounts, inspect balances, review recent transactions, compare transfer activity, or produce a concise cash-position update from IncreaseX data.
---

# IncreaseX Cash Ops

Use this workflow for read-heavy treasury tasks:

1. Discover the working account set with `resolve_account` or `list_accounts`.
2. Pull balances with `get_balance` for the relevant accounts.
3. Review recent movement with `list_recent_transactions`.
   Use `since` and `until` when the user gives explicit bounds; otherwise keep the window tight and explain the default range you used.
4. Review pending or recent outbound activity with `list_transfers`.
   Filter by `rail`, `status`, `account_id`, or `external_account_id` when the question is specific.
5. Use `retrieve_transfer` only when a specific transfer needs detail beyond the summary list.
6. If program or card-profile context matters for an account, use `list_programs` or `retrieve_program`.
7. Present cash summaries in plain language:
   account name or id, available balance, current balance, notable recent transactions, and any queued or pending transfers that affect liquidity.

Do not move money from this skill alone. Hand off to the money-movement or approval workflow when the user wants action.
