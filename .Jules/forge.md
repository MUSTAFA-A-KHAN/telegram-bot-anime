2024-06-06 - Prevent Chat Feed Spam After Shop Purchases

Learning: When building Telegram Bot conversational interfaces, it's a better user experience to edit an existing message with updated content (like showing an inventory screen after a purchase) instead of sending entirely new messages, which pushes previous buttons up and clutters the chat history.

Action: Always look for opportunities to replace  or similar new-message functions with in-place edits (e.g.,  which edits the message) when handling callbacks within a single, continuous user workflow.
2024-06-06 - Prevent Chat Feed Spam After Shop Purchases

Learning: When building Telegram Bot conversational interfaces, it's a better user experience to edit an existing message with updated content (like showing an inventory screen after a purchase) instead of sending entirely new messages, which pushes previous buttons up and clutters the chat history.

Action: Always look for opportunities to replace 'showShop' or similar new-message functions with in-place edits (e.g., 'showInventory' which edits the message) when handling callbacks within a single, continuous user workflow.
## 2023-10-25 - Fixing Missing Navigation in Settings

**Learning:** When navigating between different views with inline keyboards, make sure all "Back" button callbacks (like `settings_main`) are actually handled in the main `switch callback.Data` block. It can easily be missed when refactoring or adding new settings menus.
**Action:** Always check the full lifecycle of navigation flows and ensure all button paths have defined handlers.

2024-10-18 - Display User Balance in Shop Interfaces

Learning: When building an economy feature like an in-game shop, users must be able to view their available currency balance within the shop interface itself to prevent failed purchases and friction.
Action: Updated the `showShop` and `editShopMain` functions to fetch and present the user's `Wordle Points` balance prominently when the shop menu is rendered.
