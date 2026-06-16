
## 2024-05-19 - Wordle Board Generation Strings Allocation
**Learning:** In the `wordlebot` generation loop, `fmt.Sprintf` and `strings.ToUpper` for formatting guesses inside `buildWordleBoard` generated a large number of runtime allocations because it's executed frequently. The overhead of reflection and intermediate string slices significantly impacted performance.
**Action:** Replace `fmt.Sprintf` with string concatenation or direct `strings.Builder.WriteString`/`WriteByte`. Instead of `strings.ToUpper`, directly convert ASCII bounds by doing `c - 32` within the builder loop. Also, avoid `fmt.Sprintf` when converting small numbers like superscript digits - using an array of preset strings combined with `strconv.Itoa` reduces allocation overhead dramatically.
