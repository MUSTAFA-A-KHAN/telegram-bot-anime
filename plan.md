1. **Analyze existing user prompts for Custom Word:**
    - Currently, `controller/bot.go` and `controller/categoryBot/categoryBot.go` prompt the user for the custom word using `view.SendMessage`.
    - This is a text interaction flow. The memory states that using `ForceReply` on messages that prompt for user input is a UX improvement because it automatically opens the keyboard and sets the message as a reply.
2. **Update the prompts to use `ForceReply`:**
    - I will update `controller/bot.go` where it prompts for the custom word to use `tgbotapi.ForceReply{ForceReply: true}`.
    - I will update `controller/categoryBot/categoryBot.go` similarly.
    - To do this, I will need to use a function like `bot.Send(msg)` directly instead of `view.SendMessage`, because `view.SendMessage` does not expose a way to set `ReplyMarkup` for `ForceReply`. This matches the implementation in `controller/collectible/collectible_controller.go`.
3. **Execute Pre-commit Steps:**
    - Run the pre-commit instructions using `pre_commit_instructions` tool to make sure proper testing, verifications, reviews and reflections are done.
4. **Submit PR:**
    - Submit the changes using the `submit` tool.
