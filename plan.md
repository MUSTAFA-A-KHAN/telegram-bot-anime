1. **Add `EditMessageTextWithStyledButtons` to `view/custom_http.go`**
   - Create a new struct `EditMessageRequest` with `ChatID`, `MessageID`, `Text`, `ParseMode`, and `ReplyMarkup`.
   - Implement `EditMessageTextWithStyledButtons` similarly to `SendMessageWithStyledButtons` but targeting the `/editMessageText` Telegram API endpoint.

2. **Update `handleCallbackQuery` in `controller/bot.go`**
   - For callback data cases `statsglobal_*` and `statsgroup_*`, replace `view.SendMessageWithStyledButtons` with `view.EditMessageTextWithStyledButtons`.
   - Pass `callback.Message.MessageID` to the new function.

3. **Update `handleCallbackQuery` in `controller/categoryBot/categoryBot.go`**
   - Apply the same changes as above for consistency across bot controllers.

4. **Run Pre-Commit Checks**
   - Execute all checks to ensure proper testing, verification, review, and reflection are done.

5. **Commit and Submit**
   - Submit the changes using the `submit` tool.
