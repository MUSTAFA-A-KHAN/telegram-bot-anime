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
## 2024-06-05 - Optimize string escaping by avoiding multiple ReplaceAll calls
**Learning:** Calling `strings.ReplaceAll` in a loop for multiple characters creates numerous intermediate string allocations and iterates over the string multiple times, becoming a performance bottleneck in frequently called functions like Markdown escaping.
**Action:** Replace multiple `strings.ReplaceAll` calls with a single pass over the string using `strings.Builder` and a `switch` statement to handle special characters. This reduces time complexity from O(M*N) to O(N) and significantly reduces allocations.
## 2024-06-06 - Optimize String Normalization by Avoiding ReplaceAllString
**Learning:** Using `regexp.ReplaceAllString` to remove punctuation in hot paths, combined with multiple strings passes (e.g., `strings.Fields` and `strings.Join` for whitespace), causes severe allocation overhead and slowness due to regex engine evaluation and intermediate string creation.
**Action:** Replace `regexp.ReplaceAllString` for simple character filtering with a single-pass `strings.Builder` and the `unicode` package (e.g., checking `unicode.IsLetter`, `unicode.IsDigit`, `unicode.IsSpace`). This eliminates regex overhead and allocation churn, bringing O(N) execution and significant speedups.

## 2025-02-23 - Avoid Maps for ASCII Lookups
**Learning:** Using `map[rune]bool` to check if a character exists in a small, ASCII-only character set (e.g., english letters) introduces significant overhead compared to simple array lookups, especially inside hot loops evaluating thousands of words. Furthermore, using `strings.ReplaceAll` and `strings.ToLower` for basic string normalization before validation causes unnecessary allocations.
**Action:** Replace `map[rune]bool` with a fixed-size `[128]bool` or `[256]bool` array for O(1) ascii character existence checks. Iterate over strings using byte-indices (`for i := 0; i < len(s); i++`) instead of `range` to avoid implicit rune decoding overhead.

## 2024-06-08 - Use Fixed Arrays for Small ASCII Lookup Hot Paths
**Learning:** Using `make(map[rune]bool)` or `make(map[rune]int)` inside tight loops evaluating many words creates significant memory allocation overhead. Since the application mostly handles fixed 5-letter uppercase ASCII words, map lookups are unnecessarily heavy.
**Action:** Replace `map[rune]bool` and `map[rune]int` with fixed-size arrays (`[256]bool` and `[256]int`) for counting and tracking seen characters. Iterate over the strings using byte indices (`w[i]`) instead of `range` to eliminate implicit rune decoding overhead and drastically speed up execution.
## 2025-02-23 - Eordle Candidate Validation Optimization
**Learning:** Checking string characters with `strings.ToUpper` and checking constraints with `[]map[rune]bool` and `strings.ContainsRune` within hot loops when searching for valid Eordle words creates major garbage collection and string allocation overhead.
**Action:** Replace `strings.ToUpper` inside loops with inline bitwise ASCII conversions. Replace the slice of rune maps with a fixed `[5][256]bool` array. Eliminate all internal allocations in `validateCandidate` by utilizing byte indices and a `seen` fixed array for tracking characters.
