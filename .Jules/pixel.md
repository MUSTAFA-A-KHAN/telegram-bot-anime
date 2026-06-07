## 2024-06-07 - Better Spacing in Leaderboards

**Learning:** Leaderboards look messy when names and scores aren't aligned and Unicode emojis mess up monospace alignment in `<pre>` tags.

**Action:** Removed `<pre>` tags and manual padding from leaderboards. Used a simple inline layout pattern (`<b>[rank]</b> [name] — [score]`) to ensure clean rendering on Telegram clients.
