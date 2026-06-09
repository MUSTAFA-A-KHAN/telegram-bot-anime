## 2024-06-09 - Avoid Preformatted Tables in Telegram Mobile
**Learning:** Fixed-width tables using `<pre>` tags render poorly on mobile Telegram clients and break alignment completely when user-equipped Unicode emojis are present in the table.
**Action:** Use an inline, clean HTML structure like `<b>[rank]</b> [name] — [score]` and rely on emoji-based bullets for hierarchical display.
