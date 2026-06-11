## 2024-05-18 - Added Back Buttons to Sub-menus

**Learning:** When users launch sub-menus from a main settings menu, failing to provide a back button leads to a frustrating experience where they must dismiss the prompt and re-open the settings menu from scratch.

**Action:** Ensure all multi-level inline keyboard menus have a "🔙 Back" button to smoothly return to the previous level without abandoning the menu structure entirely.
## 2026-06-10 - Add ForceReply to User Prompts\n**Learning:** To prompt users for specific textual input (e.g., asking for a price or value), using a standard message can be confusing. Using `tgbotapi.ForceReply{ForceReply: true}` automatically opens their keyboard and sets their message as a reply, improving the UX and clarifying intent.\n**Action:** When intercepting user input in a conversation flow, attach `ForceReply` to the prompt message to explicitly guide the user to reply.
## 2026-06-11 - Visual Carousels\n**Learning:** When displaying lists of items (like inventories or marketplaces) in Telegram, long text dumps provide poor UX. Using paginated visual carousels with 'Next' and 'Prev' inline buttons and dynamically generated composite images significantly improves scannability and user delight.\n**Action:** For lists of items with visual components, build paginated navigation that edits a single media message in place.
