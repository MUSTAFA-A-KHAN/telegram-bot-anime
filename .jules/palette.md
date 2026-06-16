## 2025-02-20 - Adding explicit escape hatches for interactive modes
**Learning:** When adding new interactive text-based modes (like guessing games) to a Telegram bot, users can easily get stuck in an active state if they change their mind or don't know the answer. Missing a cancellation command (like `/cancelanime`) degrades UX and traps the user in a mode they didn't intend to stay in.
**Action:** Always provide and consistently name explicit cancellation commands (e.g., `/cancel<mode>`) that clear the active state memory and return the user to the default bot flow.
