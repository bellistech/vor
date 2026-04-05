# Consistent Hashing (System Design / Distributed Systems)

Implement a consistent hash ring with virtual nodes for even key distribution across servers, supporting O(log n) lookup and minimal key remapping on node changes.

## Problem

Implement a consistent hash ring with virtual nodes:

- **AddNode(node)** -- Add a node with virtual replicas to the ring.
- **RemoveNode(node)** -- Remove a node and all its replicas.
- **GetNode(key)** -- Return the node responsible for the given key.

**Constraints:**

- Virtual nodes improve distribution evenness.
- Use a sorted structure + binary search for O(log n) lookup.
- Thread-safe is a bonus (required for production use).

**Examples:**

```
ring = ConsistentHashRing(numReplicas=100)
ring.addNode("server1")
ring.addNode("server2")
ring.addNode("server3")

ring.getNode("mykey")     => "server2"  (deterministic)
ring.getNode("otherkey")  => "server1"  (deterministic)

ring.removeNode("server2")
ring.getNode("mykey")     => "server3"  (only server2's keys remap)
ring.getNode("otherkey")  => "server1"  (unchanged -- not on server2)
```

## Hints

- **Virtual nodes:** For each real node, create `numReplicas` virtual keys like
  `hash("server1#0")`, `hash("server1#1")`, etc. Map each virtual key hash to the
  real node name. More replicas = more even distribution.
- **Ring lookup:** Hash the key, then binary search the sorted hash list for the first
  hash >= the key hash. If past the end, wrap to index 0 (ring topology).
- **Minimal remapping:** When a node is added/removed, only keys between the new/removed
  node and its predecessor on the ring are affected. With `k` keys and `n` nodes,
  approximately `k/n` keys remap.

## Solution -- Go

```go
import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

type ConsistentHashRing struct {
	numReplicas int
	ring        map[uint64]string // hash -> node name
	sortedKeys  []uint64          // sorted hash values
	nodes       map[string]bool   // set of real node names
	mu          sync.RWMutex
}

func NewConsistentHashRing(numReplicas int) *ConsistentHashRing {
	return &ConsistentHashRing{
		numReplicas: numReplicas,
		ring:        make(map[uint64]string),
		nodes:       make(map[string]bool),
	}
}

func (c *ConsistentHashRing) hash(key string) uint64 {
	h := md5.Sum([]byte(key))
	return binary.BigEndian.Uint64(h[:8])
}

func (c *ConsistentHashRing) AddNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodes[node] {
		return
	}
	c.nodes[node] = true

	for i := 0; i < c.numReplicas; i++ {
		virtualKey := fmt.Sprintf("%s#%d", node, i)
		h := c.hash(virtualKey)
		c.ring[h] = node
		c.sortedKeys = append(c.sortedKeys, h)
	}

	sort.Slice(c.sortedKeys, func(i, j int) bool {
		return c.sortedKeys[i] < c.sortedKeys[j]
	})
}

func (c *ConsistentHashRing) RemoveNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.nodes[node] {
		return
	}
	delete(c.nodes, node)

	for i := 0; i < c.numReplicas; i++ {
		virtualKey := fmt.Sprintf("%s#%d", node, i)
		h := c.hash(virtualKey)
		delete(c.ring, h)
	}

	// Rebuild sorted keys
	c.sortedKeys = c.sortedKeys[:0]
	for h := range c.ring {
		c.sortedKeys = append(c.sortedKeys, h)
	}
	sort.Slice(c.sortedKeys, func(i, j int) bool {
		return c.sortedKeys[i] < c.sortedKeys[j]
	})
}

func (c *ConsistentHashRing) GetNode(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.sortedKeys) == 0 {
		return ""
	}

	h := c.hash(key)

	// Binary search for first hash >= h
	idx := sort.Search(len(c.sortedKeys), func(i int) bool {
		return c.sortedKeys[i] >= h
	})

	// Wrap around
	if idx >= len(c.sortedKeys) {
		idx = 0
	}

	return c.ring[c.sortedKeys[idx]]
}
```

## Solution -- Python

```python
import hashlib
import bisect
from typing import Optional, List


class ConsistentHashRing:
    def __init__(self, num_replicas: int = 150):
        self.num_replicas = num_replicas
        self.ring: dict = {}       # hash -> node name
        self.sorted_keys: List[int] = []  # sorted hash values
        self.nodes: set = set()    # set of real node names

    def _hash(self, key: str) -> int:
        """Generate a consistent hash value for a key."""
        digest = hashlib.md5(key.encode()).hexdigest()
        return int(digest, 16)

    def add_node(self, node: str) -> None:
        """Add a node with virtual replicas to the ring."""
        if node in self.nodes:
            return
        self.nodes.add(node)
        for i in range(self.num_replicas):
            virtual_key = f"{node}#{i}"
            h = self._hash(virtual_key)
            self.ring[h] = node
            bisect.insort(self.sorted_keys, h)

    def remove_node(self, node: str) -> None:
        """Remove a node and all its virtual replicas."""
        if node not in self.nodes:
            return
        self.nodes.discard(node)
        for i in range(self.num_replicas):
            virtual_key = f"{node}#{i}"
            h = self._hash(virtual_key)
            del self.ring[h]
            idx = bisect.bisect_left(self.sorted_keys, h)
            if idx < len(self.sorted_keys) and self.sorted_keys[idx] == h:
                self.sorted_keys.pop(idx)

    def get_node(self, key: str) -> Optional[str]:
        """Find the node responsible for the given key."""
        if not self.sorted_keys:
            return None

        h = self._hash(key)
        idx = bisect.bisect_left(self.sorted_keys, h)

        # Wrap around if past the end
        if idx >= len(self.sorted_keys):
            idx = 0

        return self.ring[self.sorted_keys[idx]]
```

## Solution -- Rust

```rust
use std::collections::{HashSet, BTreeMap};

struct ConsistentHashRing {
    num_replicas: usize,
    ring: BTreeMap<u64, String>,  // hash -> node name
    nodes: HashSet<String>,
}

impl ConsistentHashRing {
    fn new(num_replicas: usize) -> Self {
        ConsistentHashRing {
            num_replicas,
            ring: BTreeMap::new(),
            nodes: HashSet::new(),
        }
    }

    /// Simple hash function (FNV-1a inspired)
    fn hash(key: &str) -> u64 {
        let mut h: u64 = 14695981039346656037;
        for byte in key.as_bytes() {
            h ^= *byte as u64;
            h = h.wrapping_mul(1099511628211);
        }
        h
    }

    fn add_node(&mut self, node: &str) {
        if self.nodes.contains(node) {
            return;
        }
        self.nodes.insert(node.to_string());

        for i in 0..self.num_replicas {
            let virtual_key = format!("{}#{}", node, i);
            let h = Self::hash(&virtual_key);
            self.ring.insert(h, node.to_string());
        }
    }

    fn remove_node(&mut self, node: &str) {
        if !self.nodes.remove(node) {
            return;
        }

        for i in 0..self.num_replicas {
            let virtual_key = format!("{}#{}", node, i);
            let h = Self::hash(&virtual_key);
            self.ring.remove(&h);
        }
    }

    fn get_node(&self, key: &str) -> Option<&str> {
        if self.ring.is_empty() {
            return None;
        }

        let h = Self::hash(key);

        // Find the first hash >= h using BTreeMap range
        if let Some((_, node)) = self.ring.range(h..).next() {
            return Some(node.as_str());
        }

        // Wrap around to the first entry
        if let Some((_, node)) = self.ring.iter().next() {
            return Some(node.as_str());
        }

        None
    }
}
```

## Solution -- TypeScript

```typescript
class ConsistentHashRing {
    private numReplicas: number;
    private ring: Map<number, string> = new Map();
    private sortedKeys: number[] = [];
    private nodes: Set<string> = new Set();

    constructor(numReplicas: number = 150) {
        this.numReplicas = numReplicas;
    }

    /** Simple hash function (FNV-1a inspired, 32-bit) */
    private hash(key: string): number {
        let h = 2166136261;
        for (let i = 0; i < key.length; i++) {
            h ^= key.charCodeAt(i);
            h = Math.imul(h, 16777619);
        }
        return h >>> 0; // ensure unsigned 32-bit
    }

    addNode(node: string): void {
        if (this.nodes.has(node)) return;
        this.nodes.add(node);

        for (let i = 0; i < this.numReplicas; i++) {
            const virtualKey = `${node}#${i}`;
            const h = this.hash(virtualKey);
            this.ring.set(h, node);
            this.sortedKeys.push(h);
        }

        this.sortedKeys.sort((a, b) => a - b);
    }

    removeNode(node: string): void {
        if (!this.nodes.has(node)) return;
        this.nodes.delete(node);

        const toRemove = new Set<number>();
        for (let i = 0; i < this.numReplicas; i++) {
            const virtualKey = `${node}#${i}`;
            const h = this.hash(virtualKey);
            this.ring.delete(h);
            toRemove.add(h);
        }

        this.sortedKeys = this.sortedKeys.filter((k) => !toRemove.has(k));
    }

    getNode(key: string): string | null {
        if (this.sortedKeys.length === 0) return null;

        const h = this.hash(key);

        // Binary search for first hash >= h
        let lo = 0;
        let hi = this.sortedKeys.length;
        while (lo < hi) {
            const mid = (lo + hi) >> 1;
            if (this.sortedKeys[mid] < h) {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }

        // Wrap around
        if (lo >= this.sortedKeys.length) {
            lo = 0;
        }

        return this.ring.get(this.sortedKeys[lo]) || null;
    }
}
```

## Complexity

| Metric | Value |
|--------|-------|
| GetNode | O(log(n * R)) where n = nodes, R = replicas per node |
| AddNode | O(R * log(n * R)) for R insorts into the sorted list |
| RemoveNode | O(R * log(n * R)) for R removals + re-sort |
| Space | O(n * R) for the ring and sorted keys |

## Tips

- **Replica count matters:** 100-200 virtual nodes per real node gives good distribution.
  Too few replicas leads to hotspots; too many wastes memory and slows add/remove.
- **Hash function choice:** MD5 is fine for distribution (not security). CRC32 is faster
  but has worse distribution. In production, use xxHash or MurmurHash3 for speed + quality.
- **BTreeMap in Rust** gives natural O(log n) lookup with `range()`, eliminating the need
  to maintain a separate sorted array. This is cleaner than the sorted-slice approach.
- **Read-write lock:** GetNode is a read operation and can run concurrently with other
  reads. Use `sync.RWMutex` (Go) or `RwLock` (Rust) to allow parallel lookups while
  serializing mutations.
- **Weighted nodes:** Give more virtual nodes to more powerful servers. If server A has
  2x the capacity of server B, give it 2x the replicas.
- **Jump hash** is an alternative for fixed node sets -- O(1) lookup, perfect distribution,
  but does not support arbitrary node addition/removal.

## See Also

- hashing
- distributed-systems
- load-balancing
- binary-search

## References

- [Consistent Hashing and Random Trees (Karger et al., 1997)](https://dl.acm.org/doi/10.1145/258533.258660)
- [Consistent Hashing (Wikipedia)](https://en.wikipedia.org/wiki/Consistent_hashing)
- [Dynamo: Amazon's Key-Value Store (DeCandia et al., 2007)](https://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf)
