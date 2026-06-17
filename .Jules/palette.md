## 2024-05-18 - Added Back Buttons to Sub-menus

**Learning:** When users launch sub-menus from a main settings menu, failing to provide a back button leads to a frustrating experience where they must dismiss the prompt and re-open the settings menu from scratch.

**Action:** Ensure all multi-level inline keyboard menus have a "🔙 Back" button to smoothly return to the previous level without abandoning the menu structure entirely.

## 2024-11-10 - Add ForceReply to User Prompts
**Learning:** To prompt users for specific textual input (e.g., asking for a price or value), using a standard message can be confusing. Using `tgbotapi.ForceReply{ForceReply: true}` automatically opens their keyboard and sets their message as a reply, improving the UX and clarifying intent.
**Action:** When intercepting user input in a conversation flow, attach `ForceReply` to the prompt message to explicitly guide the user to reply.

## 2025-01-20 - Use ForceReply for Invalid Custom Words
**Learning:** For interactive text-guessing interactions that require user context (like inputting a custom word), if the user inputs an invalid string, it is vital to keep the input channel clear and explicit. A simple message response ("Invalid word") might be missed or require manual re-replying. Adding `ForceReply` to the error message ensures the keyboard remains open and explicitly forces the user to try again, reducing friction in conversational forms.
**Action:** When rejecting invalid user input in a stateful text prompt and asking them to try again, attach `ForceReply` to the error message using `tgbotapi.ForceReply{ForceReply: true}`.
## 2026-06-10 - Add ForceReply to User Prompts\n**Learning:** To prompt users for specific textual input (e.g., asking for a price or value), using a standard message can be confusing. Using `tgbotapi.ForceReply{ForceReply: true}` automatically opens their keyboard and sets their message as a reply, improving the UX and clarifying intent.\n**Action:** When intercepting user input in a conversation flow, attach `ForceReply` to the prompt message to explicitly guide the user to reply.
## 2024-05-18 - Added Back Buttons to Terminal Settings Prompts

**Learning:** When users update a setting from a main sub-menu (e.g., selecting Wordle color "Classic"), the resulting confirmation message ("✅ Wordle color updated to Classic") often strips away the inline keyboard, stranding the user. They must manually dismiss the message and re-invoke the settings menu to continue customizing.

**Action:** Ensure all multi-level inline keyboard menus, especially terminal confirmation messages indicating a successful state change, always include a "🔙 Back" button (or "🔙 Back to Settings") to smoothly return the user to the previous level without abandoning the menu structure entirely.
## $(date +%Y-%m-%d) - Add ForceReply to user input messages
**Learning:** For interactive text-guessing interactions that requires user context, a simple message might be missed or require manual 'replying'. Adding ForceReply to the message ensures the user prompt keyboard forces the user to reply to the bot message, linking the contexts seamlessly.
**Action:** When creating text prompts in Telegram Bots that wait for user inputs to update a specific state (like a custom word), attach `ForceReply` to the sent message using `tgbotapi.ForceReply{ForceReply: true}`.
