1. **Add `SaveGameState` to `repository/dbManager.go`:**
   - In `repository/dbManager.go`, add `SaveGameState(client *mongo.Client, collectionName string, chatID int64, state interface{})`. This function will use `UpdateOne` with `$set` and `upsert=true` to save game state to a collection.
2. **Verify `SaveGameState` addition:**
   - Read `repository/dbManager.go` to ensure `SaveGameState` was added correctly.
3. **Add `LoadAllGameStates` to `repository/dbManager.go`:**
   - In `repository/dbManager.go`, add `LoadAllGameStates(client *mongo.Client, collectionName string) ([]bson.M, error)` to retrieve all states from the specified collection.
4. **Verify `LoadAllGameStates` addition:**
   - Read `repository/dbManager.go` to ensure `LoadAllGameStates` was added correctly.
5. **Update Word Guess (Croc) State Management in `bot.go`:**
   - In `controller/bot.go`, add a structure `ChatStateDoc` suitable for MongoDB (without mutexes).
   - Create `saveChatStateAsync(chatID int64, state *ChatState)`. It gets a read lock, copies data to `ChatStateDoc`, and calls `repository.SaveGameState` in a goroutine (`collection: "ChatStates"`).
   - Add calls to `saveChatStateAsync` whenever the state changes (e.g., word set, user changed, reset).
   - In `StartBot`, load existing states from the `"ChatStates"` collection and populate the `chatStates` map.
6. **Verify `bot.go` edits:**
   - Run `go build -v ./...` and use `read_file` to confirm changes in `bot.go` are correct and syntactically sound.
7. **Update Word Guess State Management in `categoryBot.go`:**
   - In `controller/categoryBot/categoryBot.go`, define `CategoryChatStateDoc` and create `saveCategoryChatStateAsync(chatID int64, state *ChatState)` using collection `"CategoryChatStates"`.
   - Call it when state updates, and load from it in `StartBot`.
8. **Verify `categoryBot.go` edits:**
   - Run `go build -v ./...` and check changes.
9. **Update Wordle Game State Management:**
   - In `controller/wordlebot/wordlebot.go`, add `WordleStateDoc` and `saveWordleStateAsync` (`collection: "WordleStates"`).
   - Call on game start, guess, cancel, and end.
   - Add an init function `LoadSavedStates(client *mongo.Client)` to load states from DB.
10. **Verify `wordlebot.go` edits:**
   - Run `go build -v ./...` and verify changes.
11. **Update Scramy Game State Management:**
   - In `controller/scramybot/scramybot.go`, add `ScramyStateDoc` and `saveScramyStateAsync` (`collection: "ScramyStates"`).
   - Ignore Channels (`CancelChan`) and Mutexes when extracting state. Save on words found, score updates, game start/cancel.
   - Add `LoadSavedStates(client *mongo.Client)` to populate maps.
12. **Verify `scramybot.go` edits:**
   - Run `go build -v ./...` and verify changes.
13. **Update bot initializations:**
   - Call `wordlebot.LoadSavedStates(client)` and `scramybot.LoadSavedStates(client)` in `controller/bot.go` and `controller/categoryBot/categoryBot.go` inside `StartBot`.
14. **Verify codebase compiles and test:**
   - Run `go build -v ./...` and `go test ./...` to verify everything works and passes tests.
15. **Complete pre-commit steps:**
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.
16. **Submit:**
   - Commit the change with the message "feat: Persist game states in MongoDB to survive restarts"
