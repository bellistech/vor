# Word Ladder (BFS / Implicit Graph)

Find the shortest transformation sequence from a start word to an end word, changing one letter at a time, where every intermediate word is in a dictionary — solved as BFS on an implicit graph.

## Problem

Given two words `beginWord` and `endWord`, and a dictionary `wordList`, return the number of words in the shortest transformation sequence from `beginWord` to `endWord` such that:

- Only one letter can be changed at a time.
- Each transformed word must exist in the dictionary (except `beginWord`).

Return `0` if no such sequence exists. The length of the returned sequence includes `beginWord` and `endWord`.

**Constraints:**

- `1 <= beginWord.length <= 10`
- `endWord.length == beginWord.length`
- `1 <= wordList.length <= 5000`
- `wordList[i].length == beginWord.length`
- All words consist of lowercase English letters.
- No duplicates in `wordList`.

**Example 1:**

```
Input:  beginWord = "hit", endWord = "cog",
        wordList = ["hot","dot","dog","lot","log","cog"]
Output: 5
Explanation: "hit" -> "hot" -> "dot" -> "dog" -> "cog" (5 words)
```

**Example 2:**

```
Input:  beginWord = "hit", endWord = "cog",
        wordList = ["hot","dot","dog","lot","log"]
Output: 0
Explanation: "cog" is not in wordList, no valid sequence.
```

## Hints

1. Model each word as a graph node; two words are neighbours if they differ in exactly one letter. You need the **shortest path** in an unweighted graph — **BFS**.
2. Don't build the edge list explicitly (O(n² · L) edges). Generate neighbours on demand: for each position, try all 25 alternate letters and check dictionary membership in O(L).
3. Use a `set` for the dictionary so membership is O(L) average (hash).
4. Use a `visited` set (or mutate the dictionary) to prevent cycles and revisits.
5. **Bidirectional BFS** halves the effective search radius — expand from both ends, swap to the smaller frontier each step, terminate when frontiers intersect. Typically 10–100× faster in practice.

## Solution -- Go

```go
package main

import "fmt"

func ladderLength(beginWord, endWord string, wordList []string) int {
	dict := make(map[string]bool)
	for _, w := range wordList {
		dict[w] = true
	}
	if !dict[endWord] {
		return 0
	}

	// Bidirectional BFS
	begin := map[string]bool{beginWord: true}
	end := map[string]bool{endWord: true}
	visited := make(map[string]bool)
	visited[beginWord] = true
	visited[endWord] = true
	steps := 1

	for len(begin) > 0 && len(end) > 0 {
		// Expand smaller frontier
		if len(begin) > len(end) {
			begin, end = end, begin
		}
		next := make(map[string]bool)
		for word := range begin {
			bs := []byte(word)
			for i := 0; i < len(bs); i++ {
				orig := bs[i]
				for c := byte('a'); c <= 'z'; c++ {
					if c == orig {
						continue
					}
					bs[i] = c
					cand := string(bs)
					if end[cand] {
						return steps + 1
					}
					if dict[cand] && !visited[cand] {
						visited[cand] = true
						next[cand] = true
					}
				}
				bs[i] = orig
			}
		}
		begin = next
		steps++
	}
	return 0
}

func main() {
	if got := ladderLength("hit", "cog", []string{"hot", "dot", "dog", "lot", "log", "cog"}); got != 5 {
		panic(fmt.Sprintf("Test 1 FAILED: got %d", got))
	}
	if got := ladderLength("hit", "cog", []string{"hot", "dot", "dog", "lot", "log"}); got != 0 {
		panic(fmt.Sprintf("Test 2 FAILED: got %d", got))
	}
	if got := ladderLength("a", "c", []string{"a", "b", "c"}); got != 2 {
		panic(fmt.Sprintf("Test 3 FAILED: got %d", got))
	}
	// endWord not in dict
	if got := ladderLength("hit", "zzz", []string{"hot"}); got != 0 {
		panic(fmt.Sprintf("Test 4 FAILED: got %d", got))
	}
	// beginWord == endWord is not a transformation; per spec typically 0 or 1 depending.
	// LeetCode answer: 0 if endWord not reachable AND unchanged.
	// Here we check the unreachable-length-0 case:
	if got := ladderLength("hot", "dog", []string{"hot", "dog"}); got != 0 {
		panic(fmt.Sprintf("Test 5 FAILED: got %d", got))
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List
from collections import deque


def ladder_length(begin_word: str, end_word: str, word_list: List[str]) -> int:
    dictionary = set(word_list)
    if end_word not in dictionary:
        return 0

    begin, end = {begin_word}, {end_word}
    visited = {begin_word, end_word}
    steps = 1

    while begin and end:
        if len(begin) > len(end):
            begin, end = end, begin

        next_frontier = set()
        for word in begin:
            for i in range(len(word)):
                for c in "abcdefghijklmnopqrstuvwxyz":
                    if c == word[i]:
                        continue
                    candidate = word[:i] + c + word[i + 1:]
                    if candidate in end:
                        return steps + 1
                    if candidate in dictionary and candidate not in visited:
                        visited.add(candidate)
                        next_frontier.add(candidate)
        begin = next_frontier
        steps += 1

    return 0


if __name__ == "__main__":
    assert ladder_length("hit", "cog", ["hot", "dot", "dog", "lot", "log", "cog"]) == 5, "Test 1"
    assert ladder_length("hit", "cog", ["hot", "dot", "dog", "lot", "log"]) == 0, "Test 2"
    assert ladder_length("a", "c", ["a", "b", "c"]) == 2, "Test 3"
    assert ladder_length("hit", "zzz", ["hot"]) == 0, "Test 4"
    assert ladder_length("hot", "dog", ["hot", "dog"]) == 0, "Test 5"
    # Simple 3-step chain
    assert ladder_length("hot", "dog", ["hot", "dot", "dog"]) == 3, "Test 6"
    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::HashSet;

fn ladder_length(begin_word: String, end_word: String, word_list: Vec<String>) -> i32 {
    let dict: HashSet<String> = word_list.into_iter().collect();
    if !dict.contains(&end_word) {
        return 0;
    }

    let mut begin: HashSet<String> = HashSet::from([begin_word.clone()]);
    let mut end: HashSet<String> = HashSet::from([end_word.clone()]);
    let mut visited: HashSet<String> = HashSet::new();
    visited.insert(begin_word);
    visited.insert(end_word);
    let mut steps = 1;

    while !begin.is_empty() && !end.is_empty() {
        if begin.len() > end.len() {
            std::mem::swap(&mut begin, &mut end);
        }
        let mut next: HashSet<String> = HashSet::new();
        for word in &begin {
            let mut bytes: Vec<u8> = word.bytes().collect();
            for i in 0..bytes.len() {
                let orig = bytes[i];
                for c in b'a'..=b'z' {
                    if c == orig {
                        continue;
                    }
                    bytes[i] = c;
                    let cand = String::from_utf8(bytes.clone()).unwrap();
                    if end.contains(&cand) {
                        return steps + 1;
                    }
                    if dict.contains(&cand) && !visited.contains(&cand) {
                        visited.insert(cand.clone());
                        next.insert(cand);
                    }
                }
                bytes[i] = orig;
            }
        }
        begin = next;
        steps += 1;
    }
    0
}

fn main() {
    let wl = |v: &[&str]| v.iter().map(|s| s.to_string()).collect::<Vec<_>>();
    assert_eq!(
        ladder_length("hit".into(), "cog".into(), wl(&["hot", "dot", "dog", "lot", "log", "cog"])),
        5
    );
    assert_eq!(
        ladder_length("hit".into(), "cog".into(), wl(&["hot", "dot", "dog", "lot", "log"])),
        0
    );
    assert_eq!(ladder_length("a".into(), "c".into(), wl(&["a", "b", "c"])), 2);
    assert_eq!(ladder_length("hit".into(), "zzz".into(), wl(&["hot"])), 0);
    assert_eq!(ladder_length("hot".into(), "dog".into(), wl(&["hot", "dot", "dog"])), 3);
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function ladderLength(beginWord: string, endWord: string, wordList: string[]): number {
    const dict = new Set(wordList);
    if (!dict.has(endWord)) return 0;

    let begin = new Set<string>([beginWord]);
    let end = new Set<string>([endWord]);
    const visited = new Set<string>([beginWord, endWord]);
    let steps = 1;

    while (begin.size > 0 && end.size > 0) {
        if (begin.size > end.size) {
            [begin, end] = [end, begin];
        }
        const next = new Set<string>();
        for (const word of begin) {
            const chars = word.split("");
            for (let i = 0; i < chars.length; i++) {
                const orig = chars[i];
                for (let c = 97; c <= 122; c++) {
                    const ch = String.fromCharCode(c);
                    if (ch === orig) continue;
                    chars[i] = ch;
                    const cand = chars.join("");
                    if (end.has(cand)) return steps + 1;
                    if (dict.has(cand) && !visited.has(cand)) {
                        visited.add(cand);
                        next.add(cand);
                    }
                }
                chars[i] = orig;
            }
        }
        begin = next;
        steps++;
    }
    return 0;
}

console.assert(ladderLength("hit", "cog", ["hot", "dot", "dog", "lot", "log", "cog"]) === 5, "T1");
console.assert(ladderLength("hit", "cog", ["hot", "dot", "dog", "lot", "log"]) === 0, "T2");
console.assert(ladderLength("a", "c", ["a", "b", "c"]) === 2, "T3");
console.assert(ladderLength("hit", "zzz", ["hot"]) === 0, "T4");
console.assert(ladderLength("hot", "dog", ["hot", "dot", "dog"]) === 3, "T5");
console.log("All tests passed!");
```

## Complexity

| Aspect | Bound |
|--------|-------|
| Single-direction BFS time | O(N · L² · 26) |
| Bidirectional BFS time    | O(N · L² · 26) worst case, much faster in practice |
| Space                     | O(N · L) |

Where `N` = dictionary size, `L` = word length. Generating neighbours per word is `26 · L` candidates, each requiring `O(L)` for string construction and `O(L)` for hash.

## Tips

- **Bidirectional BFS** is the killer optimisation. Let $b$ = branching factor, $d$ = shortest-path length. Single BFS visits $O(b^d)$ nodes; bidirectional visits $O(2 b^{d/2})$ — an exponential reduction.
- **Always expand the smaller frontier**. When frontiers are unbalanced, expanding the smaller one converges faster and uses less memory.
- **Don't pre-build the graph**. Pre-computing adjacency is $O(N^2 \cdot L)$ — wasteful when most pairs aren't neighbours. On-demand neighbour generation is $O(L \cdot 26)$ per node.
- **Alternative adjacency via wildcards**: for each dictionary word, generate patterns like `h*t`, `*ot`, `ho*` and bucket words by pattern. Then neighbours of `hot` are the union of words sharing any of its patterns (minus `hot` itself). Precomputation: $O(N \cdot L^2)$; lookup: $O(L)$. Useful when the alphabet is large or the dictionary is re-queried many times.
- The **Word Ladder II** variant asks for all shortest paths — more complex. Track parent pointers or predecessor sets during BFS and reconstruct with DFS from `endWord`.
- In production (e.g. spell checkers, genome sequence alignment), **edit-distance-1 neighbours** generalise to Levenshtein BFS with insertions/deletions — same structural pattern but with variable-length candidate lists.

## See Also

- [Edit Distance](edit-distance.md) -- related single-character transformation, but optimal cost instead of shortest path.
- [Course Schedule](course-schedule.md) -- another graph problem (topological order over an explicit graph).
- [Binary Tree Level Order](binary-tree-level-order.md) -- BFS over an explicit tree.
- [Word Ladder II](word-ladder-ii.md) -- all shortest sequences, not just the length.

## References

- LeetCode 127: Word Ladder
- LeetCode 126: Word Ladder II (all shortest transformations)
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 22 (Elementary Graph Algorithms, BFS)
- Pohl, *Bi-directional Search* (1971) — original formulation of bidirectional BFS
