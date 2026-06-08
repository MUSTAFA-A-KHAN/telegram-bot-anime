1. **Define Reusable Keyboards**:
   - In `bot.go` and `categoryBot.go`, define standard navigation keyboards for the image stats.
   - For global: "Word Guess Image Global", "Wordle Image Global", "Scramy Image Global".
   - For group: "Word Guess Image Group", "Wordle Image Group", "Scramy Image Group".

2. **Update Image Sending Logic**:
   - In the callbacks for `statsimg_global_...` and `statsimg_group_...`, after generating the image bytes:
     - When constructing the `tgbotapi.NewPhotoUpload`, we cannot just pass `chatID` and `FileBytes`. We need to assign `ReplyMarkup = &keyboard` to the message before sending it.
     - We will do this for all 6 cases in both `bot.go` and `categoryBot.go`.

3. **Verify compilation**:
   - Run `go build -v ./...` to ensure no errors.

4. **Pre-commit Steps**:
   - Run verification, review, and learning steps.

5. **Submit**:
   - Submit the change.
