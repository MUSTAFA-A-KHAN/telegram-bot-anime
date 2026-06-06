2024-06-06 - Prevent Chat Feed Spam After Shop Purchases

Learning: When building Telegram Bot conversational interfaces, it's a better user experience to edit an existing message with updated content (like showing an inventory screen after a purchase) instead of sending entirely new messages, which pushes previous buttons up and clutters the chat history.

Action: Always look for opportunities to replace  or similar new-message functions with in-place edits (e.g.,  which edits the message) when handling callbacks within a single, continuous user workflow.
2024-06-06 - Prevent Chat Feed Spam After Shop Purchases

Learning: When building Telegram Bot conversational interfaces, it's a better user experience to edit an existing message with updated content (like showing an inventory screen after a purchase) instead of sending entirely new messages, which pushes previous buttons up and clutters the chat history.

Action: Always look for opportunities to replace 'showShop' or similar new-message functions with in-place edits (e.g., 'showInventory' which edits the message) when handling callbacks within a single, continuous user workflow.
