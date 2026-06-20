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
## 2025-02-23 - Optimize String Normalization by bypassing Rune extraction for ASCII
**Learning:** `for _, r := range s` combined with `unicode.ToLower` introduces unnecessary overhead for standard ASCII texts due to rune decoding, memory allocs for utf8 lookups, and function calls.
**Action:** Iterate string byte-by-byte (`for i:=0; i<len(s); { c:=s[i]... }`). For characters `< utf8.RuneSelf`, handle logic (e.g., casing, space/letter checking) inline directly using ASCII byte math (`c+32`). Keep rune fallback for non-ASCII characters.

## 2025-02-23 - Optimize String Iteration in escape functions
**Learning:** Iterating over a string using `for _, char := range text` implicitly decodes UTF-8 runes for every character. In functions that primarily check and modify ASCII characters (like escaping Markdown formatting symbols), this rune decoding adds unnecessary CPU and memory overhead compared to direct byte access.
**Action:** Replace `for _, char := range text` with a byte-indexed loop (`for i := 0; i < len(text); i++`) when searching for and appending ASCII-only characters using `strings.Builder`. Write directly via `builder.WriteByte()` instead of `builder.WriteRune()` to bypass decoding overhead.

## 2025-02-23 - Avoid Strings Split and Join for Line-by-Line Processing
**Learning:** Using `strings.Split` to break down strings by newline, processing each string, and then putting it back together with `strings.Join` generates a significant number of heap allocations and intermediate slice structures. In hot paths (like markdown text translation handlers), this creates severe GC overhead.
**Action:** Replace `strings.Split` and `strings.Join` with an inline `for` loop over string bytes, identifying `\n` characters manually and processing lines directly via `strings.Builder`.
## 2025-02-23 - Optimize Map Lookups and Allocations in Wordle Loops
**Learning:** Using `map[rune]int` for frequency counting in hot paths, combined with `fmt.Sprintf` and `strings.ToUpper` for formatting string outputs (like game boards), results in substantial heap allocations and limits execution speed.
**Action:** Replace `map[rune]int` with a fixed-size `[256]int` array for counting ASCII letter frequencies. Replace `fmt.Sprintf` and runtime string manipulators with pre-allocated `strings.Builder` (`sb.Grow()`) and inline byte-level uppercase conversions to minimize heap allocations and avoid reflection-based formatting.

## 2025-02-23 - Optimize Integer to Superscript Conversion
**Learning:** Using `fmt.Sprintf` for integer-to-string conversion combined with `map[rune]rune` for character mapping in hot loops (like formatting digits in game board loops) introduces significant overhead due to reflection, implicit rune decoding, and hash map allocations.
**Action:** Replace `fmt.Sprintf("%d", num)` with `strconv.Itoa(num)`. Replace `map[rune]rune` with a fixed `[...]string` array mapping the 10 digits to their pre-computed superscript equivalents. Loop through the resulting string by byte, handling valid digits using the array and writing output efficiently with `strings.Builder`.

## 2025-02-23 - Optimize capitalisation string allocations
**Learning:** Using `strings.ToUpper` and `strings.ToLower` for basic string capitalization generates unnecessary string allocations, especially in a tight loop. Furthermore, omitting the lowercasing part when optimizing breaks the function contract.
**Action:** When capitalizing or changing the case of strings guaranteed to be ASCII (e.g., validated game guesses), use direct byte slice mutation (e.g., `b := []byte(word); b[0] -= 32`) instead of `strings.ToUpper` or `strings.ToLower` to eliminate unnecessary string allocations, while ensuring that the full function behavior is reproduced (e.g. iterating and applying casing).

## 2025-02-23 - Avoid unsafe type casts on IDs
**Learning:** Casting Telegram User IDs (`int64`) to `int` before using `strconv.Itoa` poses a truncation risk on 32-bit systems where `int` maxes out at 2.1 billion.
**Action:** When converting 64-bit identifiers to strings, always use `strconv.FormatInt(userID, 10)` rather than downcasting to `int` for `strconv.Itoa`.
