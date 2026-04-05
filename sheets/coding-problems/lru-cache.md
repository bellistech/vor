# LRU Cache (Design / Linked Lists)

Design a fixed-capacity key-value store that evicts the least recently used entry on overflow, with O(1) get and put.

## Problem

Implement an `LRUCache` class supporting:

- `LRUCache(capacity)` -- initialise with a positive integer capacity.
- `get(key)` -- return the value if the key exists, otherwise return `-1`.
- `put(key, value)` -- update or insert. When the cache reaches capacity, evict the least recently used key before inserting the new one.

Both `get` and `put` must run in **O(1)** average time.

**Constraints:**

- `1 <= capacity <= 3000`
- `0 <= key <= 10^4`
- `0 <= value <= 10^5`
- At most `2 * 10^5` calls to `get` and `put`.

**Example:**

```
cache = LRUCache(2)
cache.put(1, 1)
cache.put(2, 2)
cache.get(1)       // 1
cache.put(3, 3)    // evicts key 2
cache.get(2)       // -1
cache.put(4, 4)    // evicts key 1
cache.get(1)       // -1
cache.get(3)       // 3
cache.get(4)       // 4
```

## Hints

1. You need two data structures working together: a **hash map** for O(1) key lookup and a **doubly linked list** for O(1) ordering/eviction.
2. The hash map maps each key to its corresponding **node** in the linked list, not to the value directly.
3. Use **sentinel** (dummy) head and tail nodes so you never have to check for null pointers when inserting or removing.
4. On every access (`get` or `put` of an existing key), **remove** the node from its current position and **re-insert at the front** (most-recently-used end).
5. On eviction, the victim is always `tail.prev` -- the node at the least-recently-used end.

## Solution -- Go

```go
package main

import "fmt"

type dllNode struct {
	key, val   int
	prev, next *dllNode
}

type LRUCache struct {
	capacity   int
	cache      map[int]*dllNode
	head, tail *dllNode // sentinels
}

func NewLRUCache(capacity int) *LRUCache {
	head := &dllNode{}
	tail := &dllNode{}
	head.next = tail
	tail.prev = head
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[int]*dllNode),
		head:     head,
		tail:     tail,
	}
}

func (c *LRUCache) remove(node *dllNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache) addToFront(node *dllNode) {
	node.next = c.head.next
	node.prev = c.head
	c.head.next.prev = node
	c.head.next = node
}

func (c *LRUCache) Get(key int) int {
	node, ok := c.cache[key]
	if !ok {
		return -1
	}
	c.remove(node)
	c.addToFront(node)
	return node.val
}

func (c *LRUCache) Put(key, value int) {
	if node, ok := c.cache[key]; ok {
		c.remove(node)
		node.val = value
		c.addToFront(node)
		return
	}

	if len(c.cache) >= c.capacity {
		lru := c.tail.prev
		c.remove(lru)
		delete(c.cache, lru.key)
	}

	node := &dllNode{key: key, val: value}
	c.cache[key] = node
	c.addToFront(node)
}

func main() {
	cache := NewLRUCache(2)
	cache.Put(1, 1)
	cache.Put(2, 2)

	if v := cache.Get(1); v != 1 {
		panic(fmt.Sprintf("Test 1 FAILED: got %d", v))
	}

	cache.Put(3, 3) // evicts key 2
	if v := cache.Get(2); v != -1 {
		panic(fmt.Sprintf("Test 2 FAILED: got %d", v))
	}

	cache.Put(4, 4) // evicts key 1
	if v := cache.Get(1); v != -1 {
		panic(fmt.Sprintf("Test 3 FAILED: got %d", v))
	}
	if v := cache.Get(3); v != 3 {
		panic(fmt.Sprintf("Test 4 FAILED: got %d", v))
	}
	if v := cache.Get(4); v != 4 {
		panic(fmt.Sprintf("Test 5 FAILED: got %d", v))
	}

	// Update existing key
	cache2 := NewLRUCache(2)
	cache2.Put(1, 1)
	cache2.Put(2, 2)
	cache2.Put(1, 10) // update, makes key 1 most recent
	cache2.Put(3, 3)  // evicts key 2
	if v := cache2.Get(2); v != -1 {
		panic(fmt.Sprintf("Test 6 FAILED: got %d", v))
	}
	if v := cache2.Get(1); v != 10 {
		panic(fmt.Sprintf("Test 7 FAILED: got %d", v))
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
class DLLNode:
    """Doubly linked list node."""
    def __init__(self, key: int = 0, val: int = 0):
        self.key = key
        self.val = val
        self.prev: 'DLLNode' = None  # type: ignore
        self.next: 'DLLNode' = None  # type: ignore


class LRUCache:
    def __init__(self, capacity: int):
        self.capacity = capacity
        self.cache: dict = {}  # key -> DLLNode

        # Sentinel head and tail nodes to simplify edge cases
        self.head = DLLNode()
        self.tail = DLLNode()
        self.head.next = self.tail
        self.tail.prev = self.head

    def _remove(self, node: DLLNode) -> None:
        """Remove a node from the doubly linked list."""
        node.prev.next = node.next
        node.next.prev = node.prev

    def _add_to_front(self, node: DLLNode) -> None:
        """Add a node right after head (most recently used position)."""
        node.next = self.head.next
        node.prev = self.head
        self.head.next.prev = node
        self.head.next = node

    def get(self, key: int) -> int:
        if key not in self.cache:
            return -1
        node = self.cache[key]
        # Move to front (most recently used)
        self._remove(node)
        self._add_to_front(node)
        return node.val

    def put(self, key: int, value: int) -> None:
        if key in self.cache:
            # Update existing: remove, update val, add to front
            node = self.cache[key]
            self._remove(node)
            node.val = value
            self._add_to_front(node)
        else:
            if len(self.cache) >= self.capacity:
                # Evict least recently used (node before tail)
                lru = self.tail.prev
                self._remove(lru)
                del self.cache[lru.key]

            # Insert new node
            node = DLLNode(key, value)
            self.cache[key] = node
            self._add_to_front(node)


if __name__ == "__main__":
    cache = LRUCache(2)
    cache.put(1, 1)
    cache.put(2, 2)
    assert cache.get(1) == 1, "Test 1 failed"
    cache.put(3, 3)  # evicts key 2
    assert cache.get(2) == -1, "Test 2 failed"
    cache.put(4, 4)  # evicts key 1
    assert cache.get(1) == -1, "Test 3 failed"
    assert cache.get(3) == 3, "Test 4 failed"
    assert cache.get(4) == 4, "Test 5 failed"

    # Additional tests
    cache2 = LRUCache(1)
    cache2.put(1, 10)
    assert cache2.get(1) == 10, "Test 6 failed"
    cache2.put(2, 20)  # evicts key 1
    assert cache2.get(1) == -1, "Test 7 failed"
    assert cache2.get(2) == 20, "Test 8 failed"

    # Update existing key
    cache3 = LRUCache(2)
    cache3.put(1, 1)
    cache3.put(2, 2)
    cache3.put(1, 10)  # update key 1, makes it most recent
    cache3.put(3, 3)   # should evict key 2, not key 1
    assert cache3.get(2) == -1, "Test 9 failed"
    assert cache3.get(1) == 10, "Test 10 failed"

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::HashMap;

struct Node {
    key: i32,
    val: i32,
    prev: usize,
    next: usize,
}

struct LRUCache {
    capacity: usize,
    map: HashMap<i32, usize>, // key -> index in nodes
    nodes: Vec<Node>,
    head: usize, // sentinel
    tail: usize, // sentinel
}

impl LRUCache {
    fn new(capacity: i32) -> Self {
        let mut nodes = Vec::new();
        // Index 0 = head sentinel, Index 1 = tail sentinel
        nodes.push(Node { key: -1, val: -1, prev: 0, next: 1 }); // head
        nodes.push(Node { key: -1, val: -1, prev: 0, next: 1 }); // tail
        LRUCache {
            capacity: capacity as usize,
            map: HashMap::new(),
            nodes,
            head: 0,
            tail: 1,
        }
    }

    fn remove(&mut self, idx: usize) {
        let prev = self.nodes[idx].prev;
        let next = self.nodes[idx].next;
        self.nodes[prev].next = next;
        self.nodes[next].prev = prev;
    }

    fn add_to_front(&mut self, idx: usize) {
        let old_first = self.nodes[self.head].next;
        self.nodes[idx].next = old_first;
        self.nodes[idx].prev = self.head;
        self.nodes[old_first].prev = idx;
        self.nodes[self.head].next = idx;
    }

    fn get(&mut self, key: i32) -> i32 {
        if let Some(&idx) = self.map.get(&key) {
            self.remove(idx);
            self.add_to_front(idx);
            self.nodes[idx].val
        } else {
            -1
        }
    }

    fn put(&mut self, key: i32, value: i32) {
        if let Some(&idx) = self.map.get(&key) {
            self.remove(idx);
            self.nodes[idx].val = value;
            self.add_to_front(idx);
        } else {
            if self.map.len() >= self.capacity {
                // Evict LRU (node before tail)
                let lru_idx = self.nodes[self.tail].prev;
                let lru_key = self.nodes[lru_idx].key;
                self.remove(lru_idx);
                self.map.remove(&lru_key);
                // Reuse the slot
                self.nodes[lru_idx].key = key;
                self.nodes[lru_idx].val = value;
                self.map.insert(key, lru_idx);
                self.add_to_front(lru_idx);
            } else {
                // Add new node
                let idx = self.nodes.len();
                self.nodes.push(Node { key, val: value, prev: 0, next: 0 });
                self.map.insert(key, idx);
                self.add_to_front(idx);
            }
        }
    }
}

fn main() {
    let mut cache = LRUCache::new(2);
    cache.put(1, 1);
    cache.put(2, 2);
    assert_eq!(cache.get(1), 1);
    cache.put(3, 3); // evicts key 2
    assert_eq!(cache.get(2), -1);
    cache.put(4, 4); // evicts key 1
    assert_eq!(cache.get(1), -1);
    assert_eq!(cache.get(3), 3);
    assert_eq!(cache.get(4), 4);

    // Update existing key
    let mut cache2 = LRUCache::new(2);
    cache2.put(1, 1);
    cache2.put(2, 2);
    cache2.put(1, 10); // update key 1
    cache2.put(3, 3);  // evicts key 2
    assert_eq!(cache2.get(2), -1);
    assert_eq!(cache2.get(1), 10);

    // Capacity 1
    let mut cache3 = LRUCache::new(1);
    cache3.put(1, 10);
    assert_eq!(cache3.get(1), 10);
    cache3.put(2, 20); // evicts key 1
    assert_eq!(cache3.get(1), -1);
    assert_eq!(cache3.get(2), 20);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
class DLLNode {
    key: number;
    val: number;
    prev: DLLNode | null = null;
    next: DLLNode | null = null;

    constructor(key: number = 0, val: number = 0) {
        this.key = key;
        this.val = val;
    }
}

class LRUCache {
    private capacity: number;
    private cache: Map<number, DLLNode> = new Map();
    private head: DLLNode; // sentinel
    private tail: DLLNode; // sentinel

    constructor(capacity: number) {
        this.capacity = capacity;
        this.head = new DLLNode();
        this.tail = new DLLNode();
        this.head.next = this.tail;
        this.tail.prev = this.head;
    }

    private remove(node: DLLNode): void {
        node.prev!.next = node.next;
        node.next!.prev = node.prev;
    }

    private addToFront(node: DLLNode): void {
        node.next = this.head.next;
        node.prev = this.head;
        this.head.next!.prev = node;
        this.head.next = node;
    }

    get(key: number): number {
        const node = this.cache.get(key);
        if (!node) return -1;
        this.remove(node);
        this.addToFront(node);
        return node.val;
    }

    put(key: number, value: number): void {
        const existing = this.cache.get(key);
        if (existing) {
            this.remove(existing);
            existing.val = value;
            this.addToFront(existing);
            return;
        }

        if (this.cache.size >= this.capacity) {
            const lru = this.tail.prev!;
            this.remove(lru);
            this.cache.delete(lru.key);
        }

        const node = new DLLNode(key, value);
        this.cache.set(key, node);
        this.addToFront(node);
    }
}

// Tests
const cache = new LRUCache(2);
cache.put(1, 1);
cache.put(2, 2);
console.assert(cache.get(1) === 1, "Test 1 failed");
cache.put(3, 3); // evicts key 2
console.assert(cache.get(2) === -1, "Test 2 failed");
cache.put(4, 4); // evicts key 1
console.assert(cache.get(1) === -1, "Test 3 failed");
console.assert(cache.get(3) === 3, "Test 4 failed");
console.assert(cache.get(4) === 4, "Test 5 failed");

// Update existing key
const cache2 = new LRUCache(2);
cache2.put(1, 1);
cache2.put(2, 2);
cache2.put(1, 10); // update key 1
cache2.put(3, 3);  // evicts key 2
console.assert(cache2.get(2) === -1, "Test 6 failed");
console.assert(cache2.get(1) === 10, "Test 7 failed");

// Capacity 1
const cache3 = new LRUCache(1);
cache3.put(1, 10);
console.assert(cache3.get(1) === 10, "Test 8 failed");
cache3.put(2, 20); // evicts key 1
console.assert(cache3.get(1) === -1, "Test 9 failed");
console.assert(cache3.get(2) === 20, "Test 10 failed");

console.log("All tests passed!");
```

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| `get`     | O(1) | --    |
| `put`     | O(1) | --    |
| Overall   | --   | O(capacity) |

- **Hash map lookup** is O(1) average.
- **Doubly linked list insert/remove** is O(1) when you hold a pointer to the node.
- **Space** is O(capacity) for storing at most `capacity` nodes plus the hash map entries.

## Tips

- **Sentinel nodes eliminate null checks.** Without them, every insert and remove must handle head/tail as special cases. With them, the list always has at least two nodes and the pointer-swapping logic is uniform.
- **Store the key inside each list node.** When you evict from the tail, you need the key to delete the corresponding hash map entry. Forgetting to store it is the most common implementation bug.
- **The Rust version uses index-based linked lists** (a `Vec<Node>` with `usize` prev/next) instead of raw pointers, which satisfies the borrow checker without `unsafe`. It also reuses evicted slots to avoid unbounded `Vec` growth.
- **Python's `OrderedDict`** can solve this in fewer lines (`move_to_end` + `popitem`), but interviewers typically want you to build the data structure from scratch.
- **Watch for the update case.** `put` on an existing key must both update the value and move the node to the front -- otherwise your eviction order is wrong.

## See Also

- [LFU Cache](lfu-cache.md) -- evict by frequency instead of recency; requires a second dimension of ordering.
- [Design HashMap](design-hashmap.md) -- the underlying hash map primitive.
- [Flatten Nested List Iterator](flatten-nested-list-iterator.md) -- another design problem combining data structures.

## References

- LeetCode 146: LRU Cache
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 11 (Hash Tables) and Chapter 10 (Linked Lists)
- Linux kernel `include/linux/list.h` -- production doubly-linked-list macros with sentinel pattern
