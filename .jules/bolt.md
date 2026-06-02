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
