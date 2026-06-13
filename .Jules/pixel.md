## 2024-06-09 - Avoid Preformatted Tables in Telegram Mobile
**Learning:** Fixed-width tables using `<pre>` tags render poorly on mobile Telegram clients and break alignment completely when user-equipped Unicode emojis are present in the table.
**Action:** Use an inline, clean HTML structure like `<b>[rank]</b> [name] — [score]` and rely on emoji-based bullets for hierarchical display.

## 2024-06-11 - Use Blockquotes for Statistics Cards
**Learning:** Dense plain text dumps of user statistics are hard to read and lack visual hierarchy in Telegram messages.
**Action:** When displaying user statistics or multi-field data, format them as structured cards using `<blockquote>` tags and empty lines to separate sections. Always escape user inputs with `html.EscapeString` to prevent HTML parsing errors.
## 2026-06-13 - Improve Visual Hierarchy for Help Screens

**Learning:** Dense text blocks with standard numbered lists are hard to scan on mobile devices and increase cognitive load. Breaking text into distinct groups with bold, emoji-prefixed headers and bullet points significantly improves readability.

**Action:** When creating or modifying long textual outputs (like  or  commands), structure the text into clear sections using emojis for hierarchy and bullet points () for lists.

## 2024-06-12 - Improve Visual Hierarchy for Help Screens

**Learning:** Dense text blocks with standard numbered lists are hard to scan on mobile devices and increase cognitive load. Breaking text into distinct groups with bold, emoji-prefixed headers and bullet points significantly improves readability.

**Action:** When creating or modifying long textual outputs (like `/rules` or `/help` commands), structure the text into clear sections using emojis for hierarchy and bullet points (`•`) for lists.
