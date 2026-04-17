# Trie / Prefix Tree (Design)

Implement a prefix tree supporting `insert`, `search`, and `startsWith` in O(L) time per operation, where L is the key length — the foundational data structure for autocomplete, IP routing, and dictionary membership.

## Problem

Design a data structure that supports the following operations for a collection of lowercase strings:

- `insert(word)` — insert a word into the trie.
- `search(word)` — return `true` iff the exact `word` is in the trie.
- `startsWith(prefix)` — return `true` iff any inserted word begins with `prefix`.

**Constraints:**

- `1 <= word.length, prefix.length <= 2000`
- `word` and `prefix` consist only of lowercase English letters.
- At most `3 * 10^4` calls total.

**Example:**

```
Trie trie = new Trie()
trie.insert("apple")
trie.search("apple")      // true
trie.search("app")        // false
trie.startsWith("app")    // true
trie.insert("app")
trie.search("app")        // true
```

## Hints

1. Each trie node has up to 26 children (one per lowercase letter) and an `isEnd` flag marking terminal words.
2. **Array vs hash map**: `[26]*TrieNode` is faster and cheaper at small alphabet sizes. `map[rune]*TrieNode` is memory-efficient when most letters are unused (sparse tries).
3. `insert` walks the string, creating new child nodes as needed, and marks the final node `isEnd = true`.
4. `search` walks to the final node and returns `isEnd`; `startsWith` walks to the final node and returns `true` regardless of `isEnd`.
5. The trick is reusing almost-identical walk logic for `search` and `startsWith` — factor out a `findNode(prefix)` helper.

## Solution -- Go

```go
package main

import "fmt"

type Trie struct {
	children [26]*Trie
	isEnd    bool
}

func NewTrie() *Trie {
	return &Trie{}
}

func (t *Trie) Insert(word string) {
	node := t
	for i := 0; i < len(word); i++ {
		idx := word[i] - 'a'
		if node.children[idx] == nil {
			node.children[idx] = &Trie{}
		}
		node = node.children[idx]
	}
	node.isEnd = true
}

func (t *Trie) findNode(s string) *Trie {
	node := t
	for i := 0; i < len(s); i++ {
		idx := s[i] - 'a'
		if node.children[idx] == nil {
			return nil
		}
		node = node.children[idx]
	}
	return node
}

func (t *Trie) Search(word string) bool {
	node := t.findNode(word)
	return node != nil && node.isEnd
}

func (t *Trie) StartsWith(prefix string) bool {
	return t.findNode(prefix) != nil
}

func main() {
	trie := NewTrie()
	trie.Insert("apple")
	if !trie.Search("apple") {
		panic("Test 1 FAILED")
	}
	if trie.Search("app") {
		panic("Test 2 FAILED: 'app' should not match")
	}
	if !trie.StartsWith("app") {
		panic("Test 3 FAILED")
	}
	trie.Insert("app")
	if !trie.Search("app") {
		panic("Test 4 FAILED")
	}

	// Empty trie
	empty := NewTrie()
	if empty.Search("a") {
		panic("Test 5 FAILED")
	}
	if empty.StartsWith("a") {
		panic("Test 6 FAILED")
	}

	// Multiple words sharing prefix
	t2 := NewTrie()
	for _, w := range []string{"apple", "application", "applet"} {
		t2.Insert(w)
	}
	if !t2.Search("apple") {
		panic("Test 7 FAILED")
	}
	if !t2.Search("application") {
		panic("Test 8 FAILED")
	}
	if t2.Search("app") {
		panic("Test 9 FAILED: 'app' not inserted")
	}
	if !t2.StartsWith("appl") {
		panic("Test 10 FAILED")
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
class Trie:
    def __init__(self) -> None:
        self.children: dict = {}  # char -> Trie
        self.is_end: bool = False

    def insert(self, word: str) -> None:
        node = self
        for ch in word:
            if ch not in node.children:
                node.children[ch] = Trie()
            node = node.children[ch]
        node.is_end = True

    def _find_node(self, s: str) -> "Trie | None":
        node = self
        for ch in s:
            if ch not in node.children:
                return None
            node = node.children[ch]
        return node

    def search(self, word: str) -> bool:
        node = self._find_node(word)
        return node is not None and node.is_end

    def starts_with(self, prefix: str) -> bool:
        return self._find_node(prefix) is not None


if __name__ == "__main__":
    trie = Trie()
    trie.insert("apple")
    assert trie.search("apple"), "Test 1"
    assert not trie.search("app"), "Test 2"
    assert trie.starts_with("app"), "Test 3"
    trie.insert("app")
    assert trie.search("app"), "Test 4"

    empty = Trie()
    assert not empty.search("a"), "Test 5"
    assert not empty.starts_with("a"), "Test 6"

    t2 = Trie()
    for w in ["apple", "application", "applet"]:
        t2.insert(w)
    assert t2.search("apple"), "Test 7"
    assert t2.search("application"), "Test 8"
    assert not t2.search("app"), "Test 9"
    assert t2.starts_with("appl"), "Test 10"

    # Unicode-ish — Python dict-based trie generalizes easily
    t3 = Trie()
    t3.insert("hello")
    t3.insert("help")
    assert t3.starts_with("hel"), "Test 11"
    assert not t3.starts_with("helpe"), "Test 12"

    print("All tests passed!")
```

## Solution -- Rust

```rust
#[derive(Default)]
struct Trie {
    children: [Option<Box<Trie>>; 26],
    is_end: bool,
}

impl Trie {
    fn new() -> Self {
        Self::default()
    }

    fn insert(&mut self, word: String) {
        let mut node = self;
        for ch in word.bytes() {
            let idx = (ch - b'a') as usize;
            node = node.children[idx].get_or_insert_with(|| Box::new(Trie::new()));
        }
        node.is_end = true;
    }

    fn find_node(&self, s: &str) -> Option<&Trie> {
        let mut node = self;
        for ch in s.bytes() {
            let idx = (ch - b'a') as usize;
            match &node.children[idx] {
                Some(next) => node = next,
                None => return None,
            }
        }
        Some(node)
    }

    fn search(&self, word: String) -> bool {
        self.find_node(&word).map_or(false, |n| n.is_end)
    }

    fn starts_with(&self, prefix: String) -> bool {
        self.find_node(&prefix).is_some()
    }
}

fn main() {
    let mut trie = Trie::new();
    trie.insert("apple".to_string());
    assert!(trie.search("apple".to_string()));
    assert!(!trie.search("app".to_string()));
    assert!(trie.starts_with("app".to_string()));
    trie.insert("app".to_string());
    assert!(trie.search("app".to_string()));

    let empty = Trie::new();
    assert!(!empty.search("a".to_string()));
    assert!(!empty.starts_with("a".to_string()));

    let mut t2 = Trie::new();
    for w in ["apple", "application", "applet"] {
        t2.insert(w.to_string());
    }
    assert!(t2.search("apple".to_string()));
    assert!(t2.search("application".to_string()));
    assert!(!t2.search("app".to_string()));
    assert!(t2.starts_with("appl".to_string()));

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
class Trie {
    private children: Map<string, Trie> = new Map();
    private isEnd: boolean = false;

    insert(word: string): void {
        let node: Trie = this;
        for (const ch of word) {
            if (!node.children.has(ch)) {
                node.children.set(ch, new Trie());
            }
            node = node.children.get(ch)!;
        }
        node.isEnd = true;
    }

    private findNode(s: string): Trie | null {
        let node: Trie = this;
        for (const ch of s) {
            const next = node.children.get(ch);
            if (!next) return null;
            node = next;
        }
        return node;
    }

    search(word: string): boolean {
        const node = this.findNode(word);
        return node !== null && node.isEnd;
    }

    startsWith(prefix: string): boolean {
        return this.findNode(prefix) !== null;
    }
}

const trie = new Trie();
trie.insert("apple");
console.assert(trie.search("apple"), "Test 1");
console.assert(!trie.search("app"), "Test 2");
console.assert(trie.startsWith("app"), "Test 3");
trie.insert("app");
console.assert(trie.search("app"), "Test 4");

const empty = new Trie();
console.assert(!empty.search("a"), "Test 5");
console.assert(!empty.startsWith("a"), "Test 6");

const t2 = new Trie();
for (const w of ["apple", "application", "applet"]) {
    t2.insert(w);
}
console.assert(t2.search("apple"), "Test 7");
console.assert(t2.search("application"), "Test 8");
console.assert(!t2.search("app"), "Test 9");
console.assert(t2.startsWith("appl"), "Test 10");

console.log("All tests passed!");
```

## Complexity

| Operation     | Time | Space per word |
|---------------|------|----------------|
| `insert`      | O(L) | O(L × σ) worst case |
| `search`      | O(L) | -- |
| `startsWith`  | O(L) | -- |
| Overall space | -- | O(ΣL × σ) |

Where `L` = key length, `σ` = alphabet size (26 for lowercase English), `ΣL` = total characters across all inserted words.

## Tips

- **Array children (`[26]*Node`) vs hash children (`Map<char, Node>`)**: fixed-array is 2–5× faster per step (no hashing) but uses 26× the pointer space even on sparse branches. Hash-based is cheaper on sparse tries and handles Unicode or large alphabets naturally.
- **`search` is just `findNode` + `isEnd` check; `startsWith` is `findNode != null`**. Factor the walk into one helper — copy-pasting invites divergence.
- **Memory pressure is real**. A trie of 1M 100-char English words with array children uses roughly $10^8 \times 26 \times 8$ bytes $\approx$ 20 GB worst case. Compression techniques: radix tree / PATRICIA trie (merge single-child chains), DAWG (collapse suffixes), HAT-trie (hybrid hash + trie).
- **Delete** is the hidden hard operation. Naive delete removes `isEnd` at the terminal node; full delete requires walking up, removing nodes that have no children and aren't `isEnd`. Use a parent pointer or a recursive bottom-up cleanup.
- **Autocomplete** is just DFS from the prefix node: collect all words in the subtree. Top-K autocomplete augments each node with a priority queue of top-K popular suffixes — O(L + K) per query.
- **IP routing** uses a binary trie (Patricia / radix tree) where edges carry bit strings rather than single characters. Longest-prefix match is the search operation, essential for BGP/OSPF forwarding tables.
- **Word Search II (LC 212)** pairs a trie of dictionary words with DFS over a 2D grid, pruning impossible branches early — a 1000× speedup over per-word backtracking.

## See Also

- [Word Search II](word-search-ii.md) -- trie + DFS on a grid; canonical trie application.
- [Design Add and Search Words](design-add-search-words.md) -- trie with `.` wildcard, recursive search.
- [Replace Words](replace-words.md) -- shortest-root replacement via trie lookup.
- [LRU Cache](lru-cache.md) -- another classic design problem pairing two data structures.

## References

- LeetCode 208: Implement Trie (Prefix Tree)
- LeetCode 211: Design Add and Search Words Data Structure
- LeetCode 212: Word Search II
- Fredkin, *Trie Memory* (Communications of the ACM, 1960) — original paper
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 11 (String matching alternatives)
- Knuth, *The Art of Computer Programming*, Vol. 3, Section 6.3 (Digital searching)
