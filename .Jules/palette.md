## 2026-06-07 - In-place updates for active games
**Learning:** When making layout adjustments (e.g. font view changes) via inline keyboard callbacks, replacing the message entirely disrupts context, especially in games where board state is visible in the message. In contrast, updating the message in place feels seamless.
**Action:** When updating a layout preference, attempt to re-render the current visible component (in this case, the game board message) in place rather than sending a new message.
