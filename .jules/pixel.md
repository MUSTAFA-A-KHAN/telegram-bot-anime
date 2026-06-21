## 2024-06-08 - Clean User Stats Formatting

**Learning:** When displaying multiple statistics for a user, using basic text with line breaks (`\n`) creates a dense, unstructured view. Switching to HTML formatting with `<b>` tags for labels, using emojis for visual hierarchy, and creating empty lines as section separators significantly improves scannability and presentation.

**Action:** Whenever sending statistics or multi-field data, format it as a structured card utilizing emojis and bold text via HTML or Markdown instead of plain text dumps.

## 2024-06-20 - Telegram Markdown Bolding

**Learning:** Telegram's simple `Markdown` mode is not identical to standard web Markdown. While standard Markdown uses double asterisks (`**bold**`) for bold text, Telegram's parser only recognizes single asterisks (`*bold*`) for bolding. Using double asterisks results in them being rendered literally as text (e.g., `**Settings**`), adding visual clutter and confusing users.

**Action:** When sending messages using `tgbotapi.ModeMarkdown` (or just `"Markdown"`), strictly use single asterisks (`*`) for bold styling and avoid double asterisks (`**`). Use `ModeMarkdownV2` if more complex formatting features are necessary, but simple `ModeMarkdown` requires the simpler syntax.
## 2024-06-21 - Visual Hierarchy in Rules Menus
**Learning:** Dense walls of text using standard bullet points (`•`) make reading game rules difficult on mobile. Replacing generic bullets with context-specific emojis (like `⌨️`, `🔍`, `🏳️`) combined with em-dashes (`—`) creates a scannable list where the action is immediately obvious. Adding an empty line after every logical section (like between "Players" and "Leaders") further improves the visual rhythm.
**Action:** Always format multi-line instruction blocks or rules using specific emojis instead of generic bullets, use em-dashes for spacing, and ensure bold section headers have adequate vertical spacing.
