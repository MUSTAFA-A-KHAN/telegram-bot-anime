1. **Change Command Handlers (`/statsimage` and `/statsimageglobal`)**:
   - In `controller/bot.go` and `controller/categoryBot/categoryBot.go`:
   - When `/statsimage` is called, instead of sending a text prompt with an inline keyboard, immediately generate the default group image (e.g. "CrocEn") and send it with the "statsimg_group" inline keyboard.
   - When `/statsimageglobal` is called, generate the default global image ("CrocEn") and send it with the "statsimg_global" inline keyboard.

2. **Change Callback Handlers**:
   - In both files, for the `statsimg_global_*` and `statsimg_group_*` callbacks, use `tgbotapi.EditMessageMediaConfig` to update the photo of the existing message in place.
   - The media being sent will be a `tgbotapi.NewInputMediaPhoto` wrapped around a `tgbotapi.FileBytes`.
   - Update the message's `ReplyMarkup` alongside the media edit to preserve the buttons.

3. **Verify Compilation**:
   - Run `go build -v ./...`.

4. **Pre-commit Steps**:
   - Run verification, tests, and memory.

5. **Submit**:
   - Submit the change.
