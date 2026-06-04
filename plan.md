1. **Understand the Wordle dictionary bug**
   - User says wordle accepts "Rider" via `categorybot` but NOT via `controller/bot`.
   - Searching the code reveals `wordlebot.LoadWordleWords()` is called in `controller/categoryBot/categoryBot.go` but NOT in `controller/bot.go` (or `main.go` / wherever `bot.go` initializes).
   - This means `validWordleWords` is not populated when running `controller/bot.go`, so ANY valid guess gets rejected as "not a valid word" (or falls back to an empty map) because `validWordleWords[guess]` is false.

2. **Add `wordlebot.LoadWordleWords()` call to `controller/bot.go`**
   - Find the initialization function in `controller/bot.go` (e.g. `InitBot`, `StartBot`, or `main` equivalent inside `controller/bot.go`).
   - Call `wordlebot.LoadWordleWords()` and log any error, similar to what's done in `categoryBot.go`.

3. **Verify Pre-Commit steps**
   - Run tests (`go test -v ./...`).
   - Re-build the application (`go build -v ./...`).

4. **Submit the changes**
