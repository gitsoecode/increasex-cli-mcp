# CLI and MCP Parity Matrix

This file tracks the supported core workflows that must exist in both the CLI and MCP.

## Accounts

| Workflow | CLI | MCP |
| --- | --- | --- |
| List accounts | `increasex accounts` | `list_accounts` |
| Resolve account | interactive account pickers | `resolve_account` |
| Get balance | `increasex balance` | `get_balance` |
| Create account | `increasex accounts create` | `create_account` |
| Close account | `increasex accounts close` | `close_account` |
| List account numbers | `increasex account-numbers` | `list_account_numbers` |
| Retrieve account number | `increasex account-numbers retrieve` | `retrieve_account_number` |
| Create account number | `increasex accounts create-number` | `create_account_number` |
| Disable account number | `increasex account-numbers disable` | `disable_account_number` |

## Transactions

| Workflow | CLI | MCP |
| --- | --- | --- |
| List recent transactions | `increasex transactions` | `list_recent_transactions` |

## Cards

| Workflow | CLI | MCP |
| --- | --- | --- |
| List cards | `increasex cards` | `list_cards` |
| Retrieve masked card details | `increasex cards retrieve` | `retrieve_card_details` |
| Retrieve card details | `increasex cards details` | `retrieve_card_sensitive_details` |
| Create details iframe | `increasex cards create-details-iframe` | `create_card_details_iframe` |
| Create card | `increasex cards create` | `create_card` |
| Update PIN | `increasex cards update-pin` | `update_card_pin` |

## External Accounts

| Workflow | CLI | MCP |
| --- | --- | --- |
| List external accounts | `increasex external-accounts` | `list_external_accounts` |
| Retrieve external account | `increasex external-accounts retrieve` | `retrieve_external_account` |
| Create external account | `increasex external-accounts create` | `create_external_account` |
| Update external account | `increasex external-accounts update` | `update_external_account` |

## Transfers

| Workflow | CLI | MCP |
| --- | --- | --- |
| Create account transfer | `increasex transfer internal` | `create_account_transfer` |
| Create ACH transfer | `increasex transfer external --rail ach` | `create_ach_transfer` |
| Create Real-Time Payments transfer | `increasex transfer external --rail real_time_payments` | `create_real_time_payments_transfer` |
| Create FedNow transfer | `increasex transfer external --rail fednow` | `create_fednow_transfer` |
| Create wire transfer | `increasex transfer external --rail wire` | `create_wire_transfer` |
| List transfers | `increasex transfer list` | `list_transfers` |
| Retrieve transfer | `increasex transfer retrieve` | `retrieve_transfer` |
| List approval queue | `increasex transfer queue` | `list_transfer_queue` |
| Approve transfer | `increasex transfer approve` | `approve_transfer` |
| Cancel transfer | `increasex transfer cancel` | `cancel_transfer` |

## Compatibility MCP Aliases

The MCP still exposes these compatibility aliases while the preferred names are the resource-shaped variants above:

- `move_money_internal`
- `move_money_external_ach`
- `move_money_external_rtp`
- `move_money_external_fednow`
- `move_money_external_wire`
