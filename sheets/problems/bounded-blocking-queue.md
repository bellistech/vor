# Bounded Blocking Queue (Concurrency / Synchronization)

Implement a thread-safe bounded blocking queue that blocks producers when full and consumers when empty.

## Problem

Design a bounded blocking queue with the following API:

1. **Enqueue(element)** -- Add an element to the back of the queue. If the queue is full,
   block the caller until space becomes available.
2. **Dequeue()** -- Remove and return the front element. If the queue is empty, block the
   caller until an element becomes available.
3. **Size()** -- Return the current number of elements in the queue.

**Constraints:**

- `1 <= capacity <= 100`
- Multiple producer and consumer threads may call Enqueue/Dequeue concurrently.
- Enqueue blocks when the queue is at capacity.
- Dequeue blocks when the queue is empty.
- The implementation must be free of deadlocks and race conditions.

**Examples:**

```
q = BoundedBlockingQueue(capacity=3)

Thread 1: q.enqueue(1)  => OK (queue: [1])
Thread 2: q.enqueue(2)  => OK (queue: [1, 2])
Thread 1: q.dequeue()   => 1  (queue: [2])
Thread 2: q.enqueue(3)  => OK (queue: [2, 3])
Thread 1: q.enqueue(4)  => OK (queue: [2, 3, 4])
Thread 2: q.enqueue(5)  => BLOCKS (queue full)
Thread 1: q.dequeue()   => 2  (Thread 2 unblocks, queue: [3, 4, 5])
```

## Hints

- **Mutex + Condition Variables:** Use two condition variables -- one for "not full"
  (producers wait on) and one for "not empty" (consumers wait on). Always wait in a
  `while` loop to handle spurious wakeups.
- **Language-specific approaches:** Go has buffered channels which naturally implement
  this pattern. TypeScript uses Promise-based cooperative blocking via async/await.
- **Signal vs. Broadcast:** `Signal`/`notify_one` wakes one waiter (sufficient for
  single-producer/single-consumer). `Broadcast`/`notify_all` is safer for
  multi-producer/multi-consumer but less efficient.

## Solution -- Go

```go
import (
	"sync"
)

// --- Approach 1: Channel-based (idiomatic Go) ---

type BoundedBlockingQueueChan struct {
	ch chan int
}

func NewBBQChan(capacity int) *BoundedBlockingQueueChan {
	return &BoundedBlockingQueueChan{ch: make(chan int, capacity)}
}

func (q *BoundedBlockingQueueChan) Enqueue(element int) {
	q.ch <- element // blocks when channel buffer is full
}

func (q *BoundedBlockingQueueChan) Dequeue() int {
	return <-q.ch // blocks when channel is empty
}

func (q *BoundedBlockingQueueChan) Size() int {
	return len(q.ch)
}

// --- Approach 2: Mutex + Cond ---

type BoundedBlockingQueue struct {
	capacity int
	queue    []int
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
}

func NewBBQ(capacity int) *BoundedBlockingQueue {
	q := &BoundedBlockingQueue{
		capacity: capacity,
		queue:    make([]int, 0, capacity),
	}
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	return q
}

func (q *BoundedBlockingQueue) Enqueue(element int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Wait while full
	for len(q.queue) >= q.capacity {
		q.notFull.Wait()
	}

	q.queue = append(q.queue, element)
	q.notEmpty.Signal() // wake a waiting consumer
}

func (q *BoundedBlockingQueue) Dequeue() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Wait while empty
	for len(q.queue) == 0 {
		q.notEmpty.Wait()
	}

	element := q.queue[0]
	q.queue = q.queue[1:]
	q.notFull.Signal() // wake a waiting producer
	return element
}

func (q *BoundedBlockingQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}
```

## Solution -- Python

```python
import threading
from collections import deque


class BoundedBlockingQueue:
    def __init__(self, capacity: int):
        self.capacity = capacity
        self.queue: deque = deque()
        self.cond = threading.Condition()

    def enqueue(self, element: int) -> None:
        with self.cond:
            # Wait while the queue is full
            while len(self.queue) >= self.capacity:
                self.cond.wait()

            self.queue.append(element)
            # Notify consumers that an element is available
            self.cond.notify_all()

    def dequeue(self) -> int:
        with self.cond:
            # Wait while the queue is empty
            while len(self.queue) == 0:
                self.cond.wait()

            element = self.queue.popleft()
            # Notify producers that space is available
            self.cond.notify_all()
            return element

    def size(self) -> int:
        with self.cond:
            return len(self.queue)
```

## Solution -- Rust

```rust
use std::collections::VecDeque;
use std::sync::{Arc, Condvar, Mutex};

struct BoundedBlockingQueue {
    inner: Mutex<VecDeque<i32>>,
    capacity: usize,
    not_empty: Condvar,
    not_full: Condvar,
}

impl BoundedBlockingQueue {
    fn new(capacity: usize) -> Self {
        BoundedBlockingQueue {
            inner: Mutex::new(VecDeque::with_capacity(capacity)),
            capacity,
            not_empty: Condvar::new(),
            not_full: Condvar::new(),
        }
    }

    fn enqueue(&self, element: i32) {
        let mut queue = self.inner.lock().unwrap();

        // Wait while full
        while queue.len() >= self.capacity {
            queue = self.not_full.wait(queue).unwrap();
        }

        queue.push_back(element);
        self.not_empty.notify_one(); // wake a waiting consumer
    }

    fn dequeue(&self) -> i32 {
        let mut queue = self.inner.lock().unwrap();

        // Wait while empty
        while queue.is_empty() {
            queue = self.not_empty.wait(queue).unwrap();
        }

        let element = queue.pop_front().unwrap();
        self.not_full.notify_one(); // wake a waiting producer
        element
    }

    fn size(&self) -> usize {
        self.inner.lock().unwrap().len()
    }
}
```

## Solution -- TypeScript

```typescript
class BoundedBlockingQueue {
    private capacity: number;
    private queue: number[] = [];
    // Waiters: producers waiting for space, consumers waiting for items
    private producerWaiters: Array<{ element: number; resolve: () => void }> = [];
    private consumerWaiters: Array<(value: number) => void> = [];

    constructor(capacity: number) {
        this.capacity = capacity;
    }

    async enqueue(element: number): Promise<void> {
        // If a consumer is waiting, deliver directly
        if (this.consumerWaiters.length > 0) {
            const resolve = this.consumerWaiters.shift()!;
            resolve(element);
            return;
        }

        // If there is room, add to queue
        if (this.queue.length < this.capacity) {
            this.queue.push(element);
            return;
        }

        // Queue is full; wait until space is available
        return new Promise<void>((resolve) => {
            this.producerWaiters.push({ element, resolve });
        });
    }

    async dequeue(): Promise<number> {
        // If items in queue, take one
        if (this.queue.length > 0) {
            const element = this.queue.shift()!;

            // If a producer is waiting, let them add their element
            if (this.producerWaiters.length > 0) {
                const waiter = this.producerWaiters.shift()!;
                this.queue.push(waiter.element);
                waiter.resolve();
            }

            return element;
        }

        // If a producer is waiting, take directly from them
        if (this.producerWaiters.length > 0) {
            const waiter = this.producerWaiters.shift()!;
            waiter.resolve();
            return waiter.element;
        }

        // Queue is empty; wait until an item is available
        return new Promise<number>((resolve) => {
            this.consumerWaiters.push(resolve);
        });
    }

    size(): number {
        return this.queue.length;
    }
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(1) amortized per enqueue/dequeue (excluding blocking wait time) |
| Space | O(capacity) -- the queue stores at most `capacity` elements |

## Tips

- **Always use `while` loops around `wait()`, never `if`.** Condition variable waits
  are subject to spurious wakeups -- the condition must be re-checked after waking.
- **Two condition variables are cleaner than one.** Using separate "not full" and "not
  empty" conditions avoids waking the wrong type of waiter. With a single condition,
  you must use `notify_all`/`broadcast` which is less efficient.
- **Go's buffered channels** are the idiomatic solution in Go -- they are bounded blocking
  queues by design. The channel approach eliminates manual lock management entirely.
- **TypeScript's single-threaded model** means "blocking" is cooperative via Promises.
  The producer/consumer waiter arrays store resolve callbacks that are invoked when the
  condition is met. This is fundamentally different from OS-level thread blocking.
- **Deadlock prevention:** Ensure every `wait()` is paired with a corresponding
  `signal()`/`notify()` on the other side. If a producer enqueues, it must signal
  the "not empty" condition. If a consumer dequeues, it must signal "not full."
- **The `Size()` method** must also hold the lock to avoid data races, even though
  it only reads. In Go's channel approach, `len(ch)` is inherently racy but acceptable
  for monitoring purposes.

## See Also

- concurrency
- mutexes-and-locks
- condition-variables
- producer-consumer-pattern
- channels

## References

- [LeetCode 1188 -- Design Bounded Blocking Queue](https://leetcode.com/problems/design-bounded-blocking-queue/)
- [Producer-Consumer Problem (Wikipedia)](https://en.wikipedia.org/wiki/Producer%E2%80%93consumer_problem)
- [Go Channels (Go Tour)](https://go.dev/tour/concurrency/2)
