# Edit Distance (Dynamic Programming / Strings)

Compute the minimum number of single-character operations (insert, delete, replace) to transform one string into another.

## Problem

Given two strings `word1` and `word2`, return the minimum number of operations required
to convert `word1` into `word2`.

The three permitted operations are:

1. **Insert** a character into word1
2. **Delete** a character from word1
3. **Replace** a character in word1 with a different character

This minimum count is known as the **Levenshtein distance** (or edit distance).

**Constraints:**

- `0 <= word1.length, word2.length <= 500`
- `word1` and `word2` consist of lowercase English letters.

**Examples:**

```
"horse" -> "ros" => 3
  horse -> rorse  (replace 'h' with 'r')
  rorse -> rose   (delete 'r')
  rose  -> ros    (delete 'e')

"intention" -> "execution" => 5

"" -> "abc" => 3  (insert 'a', 'b', 'c')

"kitten" -> "sitting" => 3
  kitten -> sitten  (replace 'k' with 's')
  sitten -> sittin  (replace 'e' with 'i')
  sittin -> sitting (insert 'g')
```

## Hints

- **Wagner-Fischer algorithm:** Build a 2D table `dp[i][j]` representing the edit
  distance between `word1[:i]` and `word2[:j]`.
- **Base cases:** `dp[i][0] = i` (delete all chars from word1), `dp[0][j] = j`
  (insert all chars from word2).
- **Recurrence:** If characters match, `dp[i][j] = dp[i-1][j-1]` (no operation needed).
  Otherwise, take 1 + the minimum of delete, insert, or replace.
- **Space optimization:** Only two rows are needed at a time, reducing space from
  O(mn) to O(min(m, n)).

## Solution -- Go

```go
func minDistance(word1, word2 string) int {
	m, n := len(word1), len(word2)

	// dp[i][j] = edit distance between word1[:i] and word2[:j]
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i // delete all from word1
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j // insert all from word2
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if word1[i-1] == word2[j-1] {
				dp[i][j] = dp[i-1][j-1] // match, no cost
			} else {
				dp[i][j] = 1 + min3(
					dp[i-1][j],   // delete
					dp[i][j-1],   // insert
					dp[i-1][j-1], // replace
				)
			}
		}
	}

	return dp[m][n]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
```

## Solution -- Python

```python
class Solution:
    def min_distance(self, word1: str, word2: str) -> int:
        m, n = len(word1), len(word2)

        # dp[i][j] = edit distance of word1[:i] and word2[:j]
        dp = [[0] * (n + 1) for _ in range(m + 1)]

        # Base cases
        for i in range(m + 1):
            dp[i][0] = i  # delete all chars from word1
        for j in range(n + 1):
            dp[0][j] = j  # insert all chars from word2

        for i in range(1, m + 1):
            for j in range(1, n + 1):
                if word1[i - 1] == word2[j - 1]:
                    dp[i][j] = dp[i - 1][j - 1]  # characters match, no operation
                else:
                    dp[i][j] = 1 + min(
                        dp[i - 1][j],      # delete from word1
                        dp[i][j - 1],      # insert into word1
                        dp[i - 1][j - 1],  # replace
                    )

        return dp[m][n]

    def min_distance_optimized(self, word1: str, word2: str) -> int:
        """Space-optimized to O(min(m,n)) using two rows."""
        m, n = len(word1), len(word2)

        # Make word2 the shorter one for space optimization
        if m < n:
            return self.min_distance_optimized(word2, word1)

        prev = list(range(n + 1))
        curr = [0] * (n + 1)

        for i in range(1, m + 1):
            curr[0] = i
            for j in range(1, n + 1):
                if word1[i - 1] == word2[j - 1]:
                    curr[j] = prev[j - 1]
                else:
                    curr[j] = 1 + min(prev[j], curr[j - 1], prev[j - 1])
            prev, curr = curr, prev

        return prev[n]
```

## Solution -- Rust

```rust
struct Solution;

impl Solution {
    fn min_distance(word1: &str, word2: &str) -> usize {
        let w1: Vec<char> = word1.chars().collect();
        let w2: Vec<char> = word2.chars().collect();
        let m = w1.len();
        let n = w2.len();

        // dp[i][j] = edit distance of word1[..i] and word2[..j]
        let mut dp = vec![vec![0usize; n + 1]; m + 1];

        // Base cases
        for i in 0..=m {
            dp[i][0] = i;
        }
        for j in 0..=n {
            dp[0][j] = j;
        }

        for i in 1..=m {
            for j in 1..=n {
                if w1[i - 1] == w2[j - 1] {
                    dp[i][j] = dp[i - 1][j - 1]; // match
                } else {
                    dp[i][j] = 1 + dp[i - 1][j]      // delete
                                     .min(dp[i][j - 1])   // insert
                                     .min(dp[i - 1][j - 1]); // replace
                }
            }
        }

        dp[m][n]
    }
}
```

## Solution -- TypeScript

```typescript
function minDistance(word1: string, word2: string): number {
    const m = word1.length;
    const n = word2.length;

    // dp[i][j] = edit distance of word1[0..i] and word2[0..j]
    const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0));

    // Base cases
    for (let i = 0; i <= m; i++) dp[i][0] = i;
    for (let j = 0; j <= n; j++) dp[0][j] = j;

    for (let i = 1; i <= m; i++) {
        for (let j = 1; j <= n; j++) {
            if (word1[i - 1] === word2[j - 1]) {
                dp[i][j] = dp[i - 1][j - 1]; // match
            } else {
                dp[i][j] =
                    1 +
                    Math.min(
                        dp[i - 1][j],     // delete
                        dp[i][j - 1],     // insert
                        dp[i - 1][j - 1]  // replace
                    );
            }
        }
    }

    return dp[m][n];
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(m * n) -- fill every cell of the (m+1) x (n+1) table |
| Space | O(m * n) for full table; O(min(m, n)) with two-row optimization |

## Walkthrough

### Tracing the DP table for "horse" -> "ros"

```
        ""    r    o    s
  ""  [  0    1    2    3 ]
  h   [  1    1    2    3 ]
  o   [  2    2    1    2 ]
  r   [  3    2    2    2 ]
  s   [  4    3    3    2 ]
  e   [  5    4    4    3 ]
```

Reading the answer: `dp[5][3] = 3`.

Backtracking from `dp[5][3]`:
- `dp[5][3] = 3`, came from `dp[4][2] + 1` (delete 'e')
- `dp[4][2] = 3`, came from `dp[3][1] + 1` (delete second 'r')
- `dp[3][1] = 2`, came from `dp[2][0] + 1` ... actually from `dp[2][1] + 1` (replace 'r' with ... wait)

Let us trace more carefully. The optimal edit sequence:
1. Replace 'h' with 'r': `dp[1][1] = 1` (from `dp[0][0] + 1`)
2. Match 'o' = 'o': `dp[2][2] = 1` (from `dp[1][1]`)
3. Delete 'r': `dp[3][2] = 2` (from `dp[2][2] + 1`)
4. Match 's' = 's': `dp[4][3] = 2` (from `dp[3][2]`)
5. Delete 'e': `dp[5][3] = 3` (from `dp[4][3] + 1`)

This gives the sequence: horse -> rorse -> rose -> ros (3 operations).

## Tips

- **The three operations map to three neighbors** in the DP table:
  - `dp[i-1][j]` = delete (word1 shrinks by one character)
  - `dp[i][j-1]` = insert (word2 shrinks by one character, equivalent to inserting in word1)
  - `dp[i-1][j-1]` = replace (both strings shrink by one)
- **When characters match**, the cost is 0 (carry forward the diagonal value). Do not
  add 1 -- this is a common off-by-one bug.
- **Space optimization** is straightforward: you only need the previous row and the
  current row. In the Python solution, swapping `prev` and `curr` avoids allocation.
- **Backtracking the operations:** To recover the actual edit sequence, trace backwards
  through the DP table from `dp[m][n]` to `dp[0][0]`, recording which operation
  (diagonal, up, left) was chosen at each step.
- **The distance is symmetric:** `edit_distance(A, B) = edit_distance(B, A)`.
  This follows from the fact that each operation has an inverse (insert/delete are
  inverses, replace is its own inverse).
- **Related problems:** The edit distance framework generalizes to weighted edit distances
  (different costs per operation), Damerau-Levenshtein distance (adding transposition as
  a fourth operation), and biological sequence alignment (Needleman-Wunsch, Smith-Waterman).
- **Lower bounds:** `|len(word1) - len(word2)|` is a trivial lower bound. The actual
  distance is always at least this value because you need at least that many
  insertions or deletions to equalize lengths.
- **Upper bound:** `max(len(word1), len(word2))` is a trivial upper bound -- you can
  always delete all of word1 and insert all of word2.

## See Also

- dynamic-programming
- string-algorithms
- sequence-alignment
- longest-common-subsequence

## References

- [LeetCode 72 -- Edit Distance](https://leetcode.com/problems/edit-distance/)
- [Wagner-Fischer Algorithm (Wikipedia)](https://en.wikipedia.org/wiki/Wagner%E2%80%93Fischer_algorithm)
- [Levenshtein Distance (Wikipedia)](https://en.wikipedia.org/wiki/Levenshtein_distance)
