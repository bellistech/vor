# Group Anagrams (Hashing / Strings)

Given a list of strings, partition them into groups where each group contains only mutual anagrams.

## Problem

Given an array of strings `strs`, group the anagrams together. You can return the answer in any order.

Two strings are **anagrams** if and only if they contain the same characters with the same frequencies (i.e., one is a rearrangement of the other).

**Constraints:**

- `1 <= strs.length <= 10^4`
- `0 <= strs[i].length <= 100`
- `strs[i]` consists of lowercase English letters only

**Examples:**

```
Input:  ["eat","tea","tan","ate","nat","bat"]
Output: [["eat","tea","ate"],["tan","nat"],["bat"]]

Input:  [""]
Output: [[""]]

Input:  ["a"]
Output: [["a"]]
```

Groups may be returned in any order, and strings within a group may appear in any order.

## Hints

1. Two strings are anagrams if their sorted character sequences are identical.
2. Use a hash map keyed by the sorted string; each value is the list of original strings sharing that key.
3. Sorting each string costs `O(k log k)` where `k` is the string length. Doing this for `n` strings gives `O(n * k log k)`.
4. Alternative: instead of sorting, build a fixed-size character-count array (26 slots for `a`-`z`) as the key. This drops the per-string cost to `O(k)`, giving `O(n * k)` overall.
5. In languages where arrays are not hashable (Python), convert the count array to a tuple for use as a dict key.

## Solution -- Go

```go
package main

import (
	"fmt"
	"sort"
)

func groupAnagrams(strs []string) [][]string {
	groups := make(map[string][]string)

	for _, s := range strs {
		// Sort the string to create the anagram key
		runes := []rune(s)
		sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
		key := string(runes)
		groups[key] = append(groups[key], s)
	}

	result := make([][]string, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	return result
}

// Alternative: O(n*k) using character counts as key
func groupAnagramsCounting(strs []string) [][]string {
	groups := make(map[[26]int][]string)

	for _, s := range strs {
		var key [26]int
		for _, c := range s {
			key[c-'a']++
		}
		groups[key] = append(groups[key], s)
	}

	result := make([][]string, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	return result
}

func main() {
	result := groupAnagrams([]string{"eat", "tea", "tan", "ate", "nat", "bat"})
	fmt.Println("Sorting approach:", result)

	result2 := groupAnagramsCounting([]string{"eat", "tea", "tan", "ate", "nat", "bat"})
	fmt.Println("Counting approach:", result2)

	// Verify group count
	if len(result) != 3 {
		panic(fmt.Sprintf("Expected 3 groups, got %d", len(result)))
	}
	if len(result2) != 3 {
		panic(fmt.Sprintf("Expected 3 groups (counting), got %d", len(result2)))
	}

	// Single element
	r := groupAnagrams([]string{"a"})
	if len(r) != 1 || len(r[0]) != 1 || r[0][0] != "a" {
		panic("Single element test failed")
	}

	// Empty string
	r = groupAnagrams([]string{""})
	if len(r) != 1 || len(r[0]) != 1 || r[0][0] != "" {
		panic("Empty string test failed")
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List
from collections import defaultdict


class Solution:
    def group_anagrams(self, strs: List[str]) -> List[List[str]]:
        # Map: sorted string -> list of anagrams
        groups: dict = defaultdict(list)
        for s in strs:
            key = "".join(sorted(s))
            groups[key].append(s)
        return list(groups.values())

    def group_anagrams_counting(self, strs: List[str]) -> List[List[str]]:
        """Alternative O(n*k) approach using character counts as key."""
        groups: dict = defaultdict(list)
        for s in strs:
            # Create a tuple of 26 character counts
            counts = [0] * 26
            for c in s:
                counts[ord(c) - ord('a')] += 1
            groups[tuple(counts)].append(s)
        return list(groups.values())


if __name__ == "__main__":
    s = Solution()

    # Helper: compare groups ignoring order
    def groups_match(a: List[List[str]], b: List[List[str]]) -> bool:
        return sorted([sorted(g) for g in a]) == sorted([sorted(g) for g in b])

    # Basic test
    result = s.group_anagrams(["eat", "tea", "tan", "ate", "nat", "bat"])
    expected = [["eat", "tea", "ate"], ["tan", "nat"], ["bat"]]
    assert groups_match(result, expected), f"Test 1 failed: {result}"

    # Empty string
    result = s.group_anagrams([""])
    assert groups_match(result, [[""]]), f"Test 2 failed: {result}"

    # Single element
    result = s.group_anagrams(["a"])
    assert groups_match(result, [["a"]]), f"Test 3 failed: {result}"

    # No anagrams
    result = s.group_anagrams(["abc", "def", "ghi"])
    assert groups_match(result, [["abc"], ["def"], ["ghi"]]), f"Test 4 failed: {result}"

    # All anagrams
    result = s.group_anagrams(["ab", "ba", "ab"])
    assert groups_match(result, [["ab", "ba", "ab"]]), f"Test 5 failed: {result}"

    # Test counting approach too
    result = s.group_anagrams_counting(["eat", "tea", "tan", "ate", "nat", "bat"])
    assert groups_match(result, expected), f"Counting test failed: {result}"

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::HashMap;

struct Solution;

impl Solution {
    fn group_anagrams(strs: Vec<String>) -> Vec<Vec<String>> {
        let mut groups: HashMap<String, Vec<String>> = HashMap::new();

        for s in strs {
            // Sort characters to get the anagram key
            let mut chars: Vec<char> = s.chars().collect();
            chars.sort_unstable();
            let key: String = chars.into_iter().collect();

            groups.entry(key).or_default().push(s);
        }

        groups.into_values().collect()
    }

    /// Alternative: O(n*k) using character count array as key
    fn group_anagrams_counting(strs: Vec<String>) -> Vec<Vec<String>> {
        let mut groups: HashMap<[u8; 26], Vec<String>> = HashMap::new();

        for s in strs {
            let mut key = [0u8; 26];
            for b in s.bytes() {
                key[(b - b'a') as usize] += 1;
            }
            groups.entry(key).or_default().push(s);
        }

        groups.into_values().collect()
    }
}

fn main() {
    let input: Vec<String> = vec!["eat", "tea", "tan", "ate", "nat", "bat"]
        .into_iter()
        .map(String::from)
        .collect();

    let result = Solution::group_anagrams(input.clone());
    assert_eq!(result.len(), 3, "Expected 3 groups");

    let result2 = Solution::group_anagrams_counting(input);
    assert_eq!(result2.len(), 3, "Expected 3 groups (counting)");

    // Single element
    let r = Solution::group_anagrams(vec!["a".to_string()]);
    assert_eq!(r.len(), 1);
    assert_eq!(r[0], vec!["a".to_string()]);

    // Empty string
    let r = Solution::group_anagrams(vec!["".to_string()]);
    assert_eq!(r.len(), 1);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function groupAnagrams(strs: string[]): string[][] {
    const groups = new Map<string, string[]>();

    for (const s of strs) {
        // Sort characters to create the anagram key
        const key = s.split("").sort().join("");
        if (!groups.has(key)) {
            groups.set(key, []);
        }
        groups.get(key)!.push(s);
    }

    return Array.from(groups.values());
}

/** Alternative: O(n*k) using character counts as key */
function groupAnagramsCounting(strs: string[]): string[][] {
    const groups = new Map<string, string[]>();

    for (const s of strs) {
        const counts = new Array(26).fill(0);
        for (const c of s) {
            counts[c.charCodeAt(0) - 97]++;
        }
        const key = counts.join(",");
        if (!groups.has(key)) {
            groups.set(key, []);
        }
        groups.get(key)!.push(s);
    }

    return Array.from(groups.values());
}

// Tests
function groupsMatch(a: string[][], b: string[][]): boolean {
    const normalize = (g: string[][]) =>
        g.map((x) => [...x].sort().join(",")).sort();
    const na = normalize(a);
    const nb = normalize(b);
    return na.length === nb.length && na.every((v, i) => v === nb[i]);
}

const input = ["eat", "tea", "tan", "ate", "nat", "bat"];
const expected = [["eat", "tea", "ate"], ["tan", "nat"], ["bat"]];

console.assert(groupsMatch(groupAnagrams(input), expected), "Test 1 failed");
console.assert(
    groupsMatch(groupAnagramsCounting(input), expected),
    "Counting test failed"
);
console.assert(
    groupsMatch(groupAnagrams([""]), [[""]]),
    "Empty string test failed"
);
console.assert(
    groupsMatch(groupAnagrams(["a"]), [["a"]]),
    "Single element test failed"
);
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time (sorting) | $O(n \cdot k \log k)$ where $n$ = number of strings, $k$ = max string length |
| Time (counting) | $O(n \cdot k)$ |
| Space | $O(n \cdot k)$ to store all strings in the hash map |

## Tips

- The sorting approach is simpler to write and sufficient for interviews. Mention the counting approach as an optimization.
- In Go, `[26]int` is directly usable as a map key since fixed-size arrays are comparable. This makes the counting approach especially clean.
- In Python, lists are unhashable, so convert the count array to a `tuple` before using it as a dict key.
- In Rust, `[u8; 26]` implements `Hash + Eq`, so it works as a `HashMap` key out of the box.
- In TypeScript/JavaScript, arrays cannot be map keys by value; serialize the count array to a string with `join(",")`.
- Watch for edge cases: empty strings (`""`) are valid input and are anagrams of each other.
- The problem says "lowercase English letters only", so 26 slots suffice. If the character set were larger (Unicode), sorting is more practical than counting.

## See Also

- [Valid Anagram](valid-anagram.md) -- the two-string special case
- [Find All Anagrams in a String](find-all-anagrams-in-a-string.md) -- sliding window variant
- [Sort Characters By Frequency](sort-characters-by-frequency.md) -- related frequency counting

## References

- LeetCode 49: Group Anagrams -- https://leetcode.com/problems/group-anagrams/
- NeetCode explanation -- https://neetcode.io/solutions/group-anagrams
