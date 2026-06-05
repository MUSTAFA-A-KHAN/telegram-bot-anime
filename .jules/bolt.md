## 2024-05-30 - [Performance Optimization for SCRAMY Dictionary Search]
**Learning:** Checking word validity inside a nested loop when generating scrambled letters was creating an O(N*M) bottleneck during game generation, which scaled linearly with the number of valid words and generation iterations.
**Action:** Replaced the character-by-character scan and array list checking by preprocessing the word lists into `validWordsMap` and pre-constructing a character set mapping. This reduces validity checking during generation to faster hash lookups.
## 2024-06-01 - Singleton MongoDB Client
**Learning:** The MongoDB Go driver establishes a connection pool. Creating a new `mongo.Client` on every request or in multiple places drains resources, degrades performance, and can lead to connection exhaustion, especially on cloud clusters.
**Action:** Always implement a singleton pattern (e.g., using `sync.Once`) for the database client so the connection pool is reused across the entire application.
## 2024-06-01 - Global Regexp Compilation
**Learning:** In Go, calling `regexp.MustCompile` inside a frequently executed function recompiles the regex on every invocation, causing significant unnecessary overhead.
**Action:** Always declare `*regexp.Regexp` variables globally at the package level when possible, compiling them once during initialization to reuse across function calls safely.
## 2024-05-15 - Optimize Regexp Compilation
**Learning:** Found dynamic `regexp.MustCompile` in multiple files inside frequently executed loops/functions throughout this Go codebase (e.g. `service/StringUtilsService.go`, `controller/translator/handlers.go`). Compiling regex at runtime is an expensive operation in Go and significantly degrades performance.
**Action:** Always declare `*regexp.Regexp` variables globally at the package level instead of inside functions or loops. Recompilation overhead should be strictly avoided.
## 2024-06-03 - Safe Double-Checked Locking in Go
**Learning:** Standard double-checked locking using a simple pointer check (`if ptr != nil`) and a `sync.Mutex` is structurally unsafe in Go due to the memory model. The compiler can reorder instructions, causing a reading goroutine to observe a non-nil pointer *before* the underlying struct is fully initialized, leading to a data race and potential panic.
**Action:** Always implement Double-Checked Locking safely in Go by using the `sync/atomic` package (e.g., `atomic.Pointer[T]`). This provides the necessary memory barriers for safe, lock-free reads while allowing the ability to retry failed initializations (unlike `sync.Once`).
