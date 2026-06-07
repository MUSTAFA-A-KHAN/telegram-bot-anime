## 2024-05-18 - Added Back Buttons to Sub-menus

**Learning:** When users launch sub-menus from a main settings menu, failing to provide a back button leads to a frustrating experience where they must dismiss the prompt and re-open the settings menu from scratch.

**Action:** Ensure all multi-level inline keyboard menus have a "🔙 Back" button to smoothly return to the previous level without abandoning the menu structure entirely.
## 2024-06-07 - Add missing navigation entry for settings menu
**Learning:** Telegram inline menus that use "Back" navigation rely entirely on mapping a callback data string (e.g., `settings_main`) to a `tgbotapi.NewEditMessageText` action. If the root menu is launched via a text command (`/settings`) but lacks a corresponding callback handler in the `switch` statement for the `Back` button, users become trapped in sub-menus. The `tgbotapi` wrapper library swallows unhandled callbacks silently rather than throwing errors.
**Action:** When designing nested inline keyboards in Telegram, always ensure that the root entry point has a corresponding `callback.Data` case registered in the main message loop so users can navigate back out of leaf nodes.
