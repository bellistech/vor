# Word Break (Dynamic Programming / Strings)

Determine if a string can be segmented into a space-separated sequence of dictionary words.

## Problem

Given a string `s` and a list of strings `wordDict`, return `true` if `s` can be
segmented into a space-separated sequence of one or more dictionary words. The same
word in the dictionary may be reused multiple times in the segmentation.

**Constraints:**

- `1 <= s.length <= 300`
- `1 <= wordDict.length <= 1000`
- `1 <= wordDict[i].length <= 20`
- `s` and `wordDict[i]` consist of lowercase English letters.
- All strings in `wordDict` are unique.

**Examples:**

```
"leetcode", ["leet","code"] => true
  "leet" + "code" are both in the dictionary.

"applepenapple", ["apple","pen"] => true
  "apple" + "pen" + "apple" — words may be reused.

"catsandog", ["cats","dog","sand","and","cat"] => false
  No valid segmentation exists.

"cars", ["car","ca","rs"] => true
  "ca" + "rs"
```

## Hints

- **DP array:** `dp[i]` = true if `s[0:i]` can be segmented into dictionary words.
- **Base case:** `dp[0] = true` (empty prefix is trivially segmentable).
- **Transition:** For each position `i`, check all `j < i` where `dp[j]` is true and
  `s[j:i]` is in the dictionary. If any such `j` exists, set `dp[i] = true`.
- **Optimization:** Use a hash set for O(1) dictionary lookups. Limit the inner loop
  to the maximum word length in the dictionary.

## Solution -- Go

```go
func wordBreak(s string, wordDict []string) bool {
	words := make(map[string]bool, len(wordDict))
	for _, w := range wordDict {
		words[w] = true
	}

	// dp[i] = true if s[:i] can be segmented
	dp := make([]bool, len(s)+1)
	dp[0] = true

	for i := 1; i <= len(s); i++ {
		for j := 0; j < i; j++ {
			if dp[j] && words[s[j:i]] {
				dp[i] = true
				break
			}
		}
	}

	return dp[len(s)]
}
```

## Solution -- Python

```python
class Solution:
    def word_break(self, s: str, word_dict: list[str]) -> bool:
        words = set(word_dict)

        # dp[i] = True if s[:i] can be segmented
        dp = [False] * (len(s) + 1)
        dp[0] = True

        for i in range(1, len(s) + 1):
            for j in range(i):
                if dp[j] and s[j:i] in words:
                    dp[i] = True
                    break

        return dp[len(s)]

    def word_break_optimized(self, s: str, word_dict: list[str]) -> bool:
        """Optimized: only check substrings up to max word length."""
        words = set(word_dict)
        max_len = max(len(w) for w in words) if words else 0

        dp = [False] * (len(s) + 1)
        dp[0] = True

        for i in range(1, len(s) + 1):
            for j in range(max(0, i - max_len), i):
                if dp[j] and s[j:i] in words:
                    dp[i] = True
                    break

        return dp[len(s)]
```

## Solution -- Rust

```rust
struct Solution;

impl Solution {
    fn word_break(s: &str, word_dict: Vec<&str>) -> bool {
        use std::collections::HashSet;
        let words: HashSet<&str> = word_dict.into_iter().collect();
        let n = s.len();

        // dp[i] = true if s[..i] can be segmented
        let mut dp = vec![false; n + 1];
        dp[0] = true;

        for i in 1..=n {
            for j in 0..i {
                if dp[j] && words.contains(&s[j..i]) {
                    dp[i] = true;
                    break;
                }
            }
        }

        dp[n]
    }
}
```

## Solution -- TypeScript

```typescript
function wordBreak(s: string, wordDict: string[]): boolean {
    const words = new Set(wordDict);

    // dp[i] = true if s.slice(0, i) can be segmented
    const dp: boolean[] = new Array(s.length + 1).fill(false);
    dp[0] = true;

    for (let i = 1; i <= s.length; i++) {
        for (let j = 0; j < i; j++) {
            if (dp[j] && words.has(s.slice(j, i))) {
                dp[i] = true;
                break;
            }
        }
    }

    return dp[s.length];
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(n^2 * m) -- for each of n positions, check up to n previous positions with O(m) substring comparison (m = avg word length) |
| Space | O(n) for the DP array + O(k) for the word set where k = total characters in dictionary |

## Walkthrough

### Tracing the DP for "leetcode", ["leet", "code"]

```
s = "leetcode"
words = {"leet", "code"}

dp[0] = true  (base case: empty string)
dp[1]: check s[0:1]="l" — not in words.  dp[1] = false
dp[2]: check s[0:2]="le" — not in words. dp[2] = false
dp[3]: check s[0:3]="lee" — not in words. dp[3] = false
dp[4]: check s[0:4]="leet" — in words AND dp[0]=true!  dp[4] = true
dp[5]: check s[0:5]..s[4:5]="c" — none match. dp[5] = false
dp[6]: check s[4:6]="co" — not in words. dp[6] = false
dp[7]: check s[4:7]="cod" — not in words. dp[7] = false
dp[8]: check s[4:8]="code" — in words AND dp[4]=true!  dp[8] = true

Result: dp[8] = true → "leet" + "code"
```

### Tracing "catsandog", ["cats","dog","sand","and","cat"]

```
dp[0] = true
dp[3]: s[0:3]="cat" — in words, dp[0]=true → dp[3] = true
dp[4]: s[0:4]="cats" — in words, dp[0]=true → dp[4] = true
dp[6]: s[3:6]="san" — no. s[0:6]="catsan" — no. dp[6] = false
dp[7]: s[3:7]="sand" — in words, dp[3]=true → dp[7] = true
        also s[4:7]="and" — in words, dp[4]=true → dp[7] = true
dp[8]: s[7:8]="o" — no. No valid j. dp[8] = false
dp[9]: s[7:9]="og" — no. No valid j. dp[9] = false

Result: dp[9] = false — cannot reach the end.
```

## Tips

- **Use a set for the dictionary** for O(1) lookups. A list would add an O(k) factor.
- **Limit inner loop range:** If the longest word has length L, only check
  `j` in `[max(0, i-L), i)`. This reduces the inner loop from O(n) to O(L).
- **BFS alternative:** Treat positions as nodes, edges as dictionary words. BFS from
  position 0 to position n. Same complexity but different mental model.
- **Trie optimization:** For very large dictionaries, build a trie and walk it
  character by character to find all matching words at each position.
- **Backtracking to find segmentations:** To find the actual word segmentation (not
  just true/false), store which word was used at each `dp[i]` and trace back.
- **This is NOT a greedy problem.** Greedy matching (longest or shortest first)
  fails: "catsand" with ["cat", "cats", "and"] -- greedy "cats" leaves "and" (OK),
  but greedy "cat" leaves "sand" (not in dict). DP considers all possibilities.
- **Common follow-up:** "Word Break II" asks for all possible segmentations (backtracking + memoization).

## See Also

- dynamic-programming
- string-algorithms
- longest-increasing-subsequence

## References

- [LeetCode 139 -- Word Break](https://leetcode.com/problems/word-break/)
- [LeetCode 140 -- Word Break II](https://leetcode.com/problems/word-break-ii/)
