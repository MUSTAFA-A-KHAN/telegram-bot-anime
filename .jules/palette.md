## 2024-03-24 - Expandable Blockquotes for Scannability
**Learning:** Wrapping long repetitive lists like leaderboards in Telegram's `<blockquote expandable>` HTML tag makes the UI much cleaner, more scannable, and mobile-friendly without losing any information. It improves the UX by preventing large walls of text from overtaking the chat view.
**Action:** When implementing or fixing UI outputs that display lists (such as leaderboards or participant lists), use the `<blockquote expandable>` tag instead of standard `<blockquote>` or just plaintext to keep the chat tidy.
