## 2024-06-25 - Unified Cancel Commands for Interactive Games
**Learning:** Having standard navigational and state-reset commands (like /cancelgeo, /cancelanime) available across all context modes (DMs and groups, both standard and category bot) prevents users from becoming stuck in unwanted game states and provides a more pleasant, predictable interface.
**Action:** Always ensure new interactive modes implemented have a corresponding explicit cancel command mapped in both `bot.go` and `categoryBot.go` to maintain consistent conversational UX.
