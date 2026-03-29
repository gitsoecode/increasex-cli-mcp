---
name: increasex-card-ops
description: Card operations workflow for `mcp__increasex`. Use when an agent needs to list cards, inspect masked card details, create a details iframe, create a new card, update a PIN, or reason about digital card profile selection while keeping sensitive card data protected by default.
---

# IncreaseX Card Ops

Use this workflow for card management:

1. Start with `list_cards` to find the working card set.
2. Prefer `retrieve_card_details` for routine inspection.
   Only use `retrieve_card_sensitive_details` when the user explicitly asks for PAN, CVV, or PIN-level access.
3. Use `list_digital_card_profiles` or `retrieve_digital_card_profile` before setting `digital_wallet.digital_card_profile_id` on a new card.
4. Use `retrieve_program` when a program’s default digital card profile or commercial terms matter for the decision.
5. Use `create_card_details_iframe` when the user needs a secure handoff to view card details without exposing them directly in chat.
6. Keep `create_card` preview-first and confirm the account, description, billing details, and digital-wallet settings before execution.
7. Keep `update_card_pin` preview-first and avoid repeating the PIN in summaries or logs.

If a card workflow turns into transfer or approval work, hand off to the matching transfer skill.
