# Algorithm Patterns

The recognize-then-template reference for picking the right algorithmic technique from the shape of an input. Every pattern includes triggers, templates, and worked Python + Go examples.

## Setup

Algorithm patterns are reusable structural blueprints — once you recognize one, the implementation drops out of a template. The art of competitive coding and interview-grade engineering is *pattern recognition first, code second*. If you find yourself coding without a pattern in mind, stop and reclassify the problem.

The recognize-then-template approach has three steps:

1. **Classify the input.** Sorted array? Graph? Tree? Stream? Matrix? Intervals? Each shape suggests a small family of techniques.
2. **Identify the operation.** Searching? Counting? Optimizing? Aggregating over windows? Each operation narrows the family further.
3. **Drop in the template.** Once you have the pattern, the boilerplate writes itself: the bug surface shrinks to the problem-specific bits.

This sheet is built so a terminal-bound engineer never has to web-search "is this BFS or Dijkstra?" or "what's the rolling-array trick for knapsack?" — the recipe is here, with both Python and Go in every section.

### When to apply each (decision tree, summary)

```
Input form ──► Pattern family
─────────────────────────────────
Sorted array ──► binary search, two pointers, merge intervals
Unsorted array ──► hashmap, sort+scan, prefix sum, sliding window
Linked list ──► two pointers (slow/fast), recursion
Tree ──► DFS, BFS, tree DP
Graph ──► DFS, BFS, Dijkstra, Bellman-Ford, MST
Matrix ──► DP (path), BFS (shortest), DFS (flood fill)
Intervals ──► sort+merge, sweep line
String ──► two pointers, sliding window, KMP/Z, suffix array, trie
Stream ──► reservoir sampling, two heaps, count-min sketch
Bits ──► XOR, AND mask, OR mask
Recurrence ──► memoized recursion / bottom-up DP
Decisions ──► backtracking or DP
```

The rest of the sheet expands each branch with templates, edge cases, and runnable code.

## Pattern Decision Tree

The single most useful skill is reading the *shape* of the input and immediately narrowing the candidate patterns. Below is the canonical lookup; the rest of this sheet is a library of templates keyed off these triggers.

### By input shape

```
sorted array
    └─ find element / boundary       → binary search lower/upper bound
    └─ pair sums to target           → two pointers (opposite direction)
    └─ remove duplicates / partition → two pointers (same direction)
    └─ overlapping ranges            → merge intervals

unsorted array
    └─ subarray contiguous, fixed k  → sliding window (fixed)
    └─ subarray contiguous, variable → sliding window (variable)
    └─ subarray sum equals K         → prefix sum + hashmap
    └─ next greater / smaller        → monotonic stack
    └─ kth largest / smallest        → heap or quickselect
    └─ count smaller after self      → BIT / merge sort

linked list
    └─ cycle detection               → fast/slow (Floyd)
    └─ middle / nth-from-end         → fast/slow / two pointers
    └─ reverse                       → iterative pointer flip

matrix (grid)
    └─ shortest path 4-dir, weight 1 → BFS
    └─ all paths / fill region       → DFS (with visited)
    └─ count islands                 → DFS or union-find
    └─ path with min/max sum         → DP

tree
    └─ traverse                      → DFS recursive (pre/in/post)
    └─ level by level                → BFS queue
    └─ best subtree value            → tree DP (post-order)
    └─ LCA                           → binary lifting / Tarjan

graph
    └─ unweighted shortest path      → BFS
    └─ non-negative weighted SP      → Dijkstra
    └─ negative weights              → Bellman-Ford
    └─ all-pairs SP                  → Floyd-Warshall
    └─ MST                           → Kruskal or Prim
    └─ topo order / cycle detect     → DFS / Kahn BFS
    └─ connected components          → DFS / union-find

intervals
    └─ overlap detection / merge     → sort by start + sweep
    └─ count overlaps at point       → sweep line
    └─ insert into sorted intervals  → linear scan + merge

string
    └─ exact substring match         → KMP, Z-algorithm
    └─ multiple patterns at once     → Aho-Corasick
    └─ longest common prefix on set  → trie
    └─ palindrome substrings         → Manacher / DP
    └─ longest with K distinct chars → sliding window

stream / unknown size
    └─ kth largest                   → min-heap of size k
    └─ median                        → two heaps
    └─ random sample                 → reservoir sampling

bit operations
    └─ duplicates / missing single   → XOR
    └─ subset enumeration            → bitmask DP
    └─ flag tests                    → AND mask
```

### By operation

```
optimize over choices  → DP (overlapping) or backtracking (independent)
count something        → combinatorics or DP
test feasibility       → BFS/DFS reachability
shortest / longest path→ BFS, Dijkstra, Bellman-Ford depending on weights
find an answer in space→ binary search on answer
balance two halves     → two heaps
windowed aggregate     → sliding window or monotonic deque
range update           → difference array or segment tree
range query            → prefix sum or segment tree or sparse table
```

When two patterns both fit, prefer the simpler one — prefix sum over segment tree, BFS over Dijkstra with weight 1, two pointers over hashmap when the array is already sorted. Simplicity is correctness insurance.

## Sliding Window — Fixed Size

**Trigger:** subarray or substring of fixed length k where you want a per-window aggregate (sum, max, count of property).

**Template:**

1. Build the window over indices `[0, k-1]`.
2. Slide one step at a time: add `arr[right]`, remove `arr[left]`.
3. Track the answer at each step.

The window invariant: at the start of iteration `i ≥ k`, the window contains exactly elements `arr[i-k .. i-1]`.

### Worked example: max sum subarray of size k (Python)

```python
def max_sum_window(arr, k):
    if len(arr) < k:
        return None
    window_sum = sum(arr[:k])
    best = window_sum
    for i in range(k, len(arr)):
        window_sum += arr[i] - arr[i - k]
        if window_sum > best:
            best = window_sum
    return best

assert max_sum_window([2, 1, 5, 1, 3, 2], 3) == 9
```

### Worked example: max sum subarray of size k (Go)

```go
package main

import "fmt"

func maxSumWindow(arr []int, k int) int {
    if len(arr) < k {
        return 0
    }
    sum := 0
    for i := 0; i < k; i++ {
        sum += arr[i]
    }
    best := sum
    for i := k; i < len(arr); i++ {
        sum += arr[i] - arr[i-k]
        if sum > best {
            best = sum
        }
    }
    return best
}

func main() {
    fmt.Println(maxSumWindow([]int{2, 1, 5, 1, 3, 2}, 3)) // 9
}
```

### First negative number in every window of size k (Python)

```python
from collections import deque

def first_negative_in_windows(arr, k):
    out = []
    negs = deque()
    for i, x in enumerate(arr):
        if x < 0:
            negs.append(i)
        if i >= k - 1:
            while negs and negs[0] <= i - k:
                negs.popleft()
            out.append(arr[negs[0]] if negs else 0)
    return out

print(first_negative_in_windows([12, -1, -7, 8, -15, 30, 16, 28], 3))
# [-1, -1, -7, -15, -15, 0]
```

### Max in window of size k (Go, monotonic deque)

```go
func maxInWindow(arr []int, k int) []int {
    dq := []int{} // indices
    out := []int{}
    for i, v := range arr {
        for len(dq) > 0 && dq[0] <= i-k {
            dq = dq[1:]
        }
        for len(dq) > 0 && arr[dq[len(dq)-1]] < v {
            dq = dq[:len(dq)-1]
        }
        dq = append(dq, i)
        if i >= k-1 {
            out = append(out, arr[dq[0]])
        }
    }
    return out
}
```

**Edge cases:**

- `k > len(arr)` — return empty / sentinel.
- `k == 0` — undefined; reject.
- All negatives, all positives, all zero — verify your aggregate handles each.

## Sliding Window — Variable Size

**Trigger:** find the longest/shortest subarray or substring satisfying a property; the window grows on the right and shrinks on the left when the property is violated.

**Template:**

```
left = 0
state = empty
for right in range(n):
    add arr[right] to state
    while invariant_violated(state):
        remove arr[left] from state
        left += 1
    update answer with arr[left:right+1]
```

The invariant is the property you must preserve. The window's left edge advances exactly as far as needed to restore it.

### Longest substring with at most K distinct characters (Python)

```python
from collections import Counter

def longest_k_distinct(s, k):
    if k == 0:
        return 0
    counts = Counter()
    left = 0
    best = 0
    for right, ch in enumerate(s):
        counts[ch] += 1
        while len(counts) > k:
            counts[s[left]] -= 1
            if counts[s[left]] == 0:
                del counts[s[left]]
            left += 1
        if right - left + 1 > best:
            best = right - left + 1
    return best

assert longest_k_distinct("eceba", 2) == 3  # "ece"
```

### Longest substring with at most K distinct characters (Go)

```go
func longestKDistinct(s string, k int) int {
    if k == 0 {
        return 0
    }
    counts := map[byte]int{}
    left, best := 0, 0
    for right := 0; right < len(s); right++ {
        counts[s[right]]++
        for len(counts) > k {
            counts[s[left]]--
            if counts[s[left]] == 0 {
                delete(counts, s[left])
            }
            left++
        }
        if right-left+1 > best {
            best = right - left + 1
        }
    }
    return best
}
```

### Minimum window substring (Python)

```python
from collections import Counter

def min_window(s, t):
    if not t or not s:
        return ""
    need = Counter(t)
    missing = len(t)
    left = best_l = 0
    best_len = float('inf')
    for right, ch in enumerate(s, 1):
        if need[ch] > 0:
            missing -= 1
        need[ch] -= 1
        if missing == 0:
            while need[s[left]] < 0:
                need[s[left]] += 1
                left += 1
            if right - left < best_len:
                best_len = right - left
                best_l = left
            need[s[left]] += 1
            missing += 1
            left += 1
    return "" if best_len == float('inf') else s[best_l:best_l + best_len]

assert min_window("ADOBECODEBANC", "ABC") == "BANC"
```

### Smallest subarray with sum ≥ S (Go)

```go
func smallestSubarrayWithSum(arr []int, s int) int {
    left, sum := 0, 0
    best := -1
    for right := 0; right < len(arr); right++ {
        sum += arr[right]
        for sum >= s {
            length := right - left + 1
            if best == -1 || length < best {
                best = length
            }
            sum -= arr[left]
            left++
        }
    }
    return best
}
```

**Edge cases:**

- Empty input — return zero-length sentinel.
- All elements equal — exercises the inner while loop boundary.
- Target unachievable — return `-1` or empty string consistently.

## Two Pointers — Same Direction

**Trigger:** in-place rearrangement of a single array; the slow pointer marks the "frontier of valid output", the fast pointer scans.

**Template:**

```
slow = 0
for fast in range(n):
    if keep(arr[fast]):
        arr[slow] = arr[fast]
        slow += 1
return slow  # length of the kept prefix
```

### Remove duplicates from sorted array (Python)

```python
def remove_duplicates(arr):
    if not arr:
        return 0
    slow = 1
    for fast in range(1, len(arr)):
        if arr[fast] != arr[slow - 1]:
            arr[slow] = arr[fast]
            slow += 1
    return slow

a = [1, 1, 2, 2, 3]
n = remove_duplicates(a)
assert a[:n] == [1, 2, 3]
```

### Move zeros to end, preserve order of non-zeros (Go)

```go
func moveZeros(arr []int) {
    slow := 0
    for fast := 0; fast < len(arr); fast++ {
        if arr[fast] != 0 {
            arr[slow], arr[fast] = arr[fast], arr[slow]
            slow++
        }
    }
}
```

### Partition by predicate (Python)

```python
def partition(arr, pred):
    slow = 0
    for fast in range(len(arr)):
        if pred(arr[fast]):
            arr[slow], arr[fast] = arr[fast], arr[slow]
            slow += 1
    return slow  # index of first non-matching

a = [3, 1, 4, 1, 5, 9, 2, 6]
n = partition(a, lambda x: x % 2 == 0)
# Even prefix length n; even values come first
```

### Remove element in place (Go)

```go
func removeElement(arr []int, val int) int {
    slow := 0
    for fast := 0; fast < len(arr); fast++ {
        if arr[fast] != val {
            arr[slow] = arr[fast]
            slow++
        }
    }
    return slow
}
```

**Edge cases:** empty input, all elements match, no element matches.

## Two Pointers — Opposite Direction

**Trigger:** sorted array (or palindrome check) where moving from both ends inward is meaningful.

**Template:**

```
left, right = 0, n - 1
while left < right:
    if condition(arr[left], arr[right]):
        # use arr[left], arr[right]
        left += 1; right -= 1
    elif need_more_left(arr[left], arr[right]):
        left += 1
    else:
        right -= 1
```

### Two-sum sorted (Python)

```python
def two_sum_sorted(arr, target):
    left, right = 0, len(arr) - 1
    while left < right:
        s = arr[left] + arr[right]
        if s == target:
            return left, right
        if s < target:
            left += 1
        else:
            right -= 1
    return None

assert two_sum_sorted([1, 2, 3, 4, 6], 6) == (1, 3)
```

### Container with most water (Go)

```go
func maxArea(height []int) int {
    left, right := 0, len(height)-1
    best := 0
    for left < right {
        h := height[left]
        if height[right] < h {
            h = height[right]
        }
        area := h * (right - left)
        if area > best {
            best = area
        }
        if height[left] < height[right] {
            left++
        } else {
            right--
        }
    }
    return best
}
```

### Trapping rain water (Python)

```python
def trap(height):
    left, right = 0, len(height) - 1
    left_max = right_max = 0
    water = 0
    while left < right:
        if height[left] < height[right]:
            if height[left] >= left_max:
                left_max = height[left]
            else:
                water += left_max - height[left]
            left += 1
        else:
            if height[right] >= right_max:
                right_max = height[right]
            else:
                water += right_max - height[right]
            right -= 1
    return water

assert trap([0,1,0,2,1,0,1,3,2,1,2,1]) == 6
```

### Valid palindrome (Go)

```go
import "unicode"

func isPalindrome(s string) bool {
    runes := []rune(s)
    left, right := 0, len(runes)-1
    for left < right {
        for left < right && !isAlnum(runes[left]) {
            left++
        }
        for left < right && !isAlnum(runes[right]) {
            right--
        }
        if unicode.ToLower(runes[left]) != unicode.ToLower(runes[right]) {
            return false
        }
        left++
        right--
    }
    return true
}

func isAlnum(r rune) bool {
    return unicode.IsLetter(r) || unicode.IsDigit(r)
}
```

**Edge cases:** length 0 or 1 → trivially valid; verify with `assert is_palindrome("") == True`.

## Fast/Slow Pointer (Floyd)

**Trigger:** detect a cycle, find the middle, or work with implicit linked structures (functional iteration `x = f(x)`).

**Template:**

```
slow = start
fast = start
while fast and fast.next:
    slow = slow.next
    fast = fast.next.next
    if slow is fast:
        return cycle_detected
return None
```

### Detect cycle in linked list (Python)

```python
class Node:
    def __init__(self, val, nxt=None):
        self.val = val
        self.next = nxt

def has_cycle(head):
    slow = fast = head
    while fast and fast.next:
        slow = slow.next
        fast = fast.next.next
        if slow is fast:
            return True
    return False
```

### Find cycle start (Floyd's tortoise + hare, Python)

```python
def cycle_start(head):
    slow = fast = head
    while fast and fast.next:
        slow = slow.next
        fast = fast.next.next
        if slow is fast:
            break
    else:
        return None
    if not fast or not fast.next:
        return None
    slow = head
    while slow is not fast:
        slow = slow.next
        fast = fast.next
    return slow
```

The math: if before the meeting point `slow` covered distance `d` and `fast` covered `2d`, the difference is the cycle length multiple. Resetting `slow` to head and walking both at speed 1 makes them meet exactly at the cycle start.

### Middle of linked list (Go)

```go
type Node struct {
    Val  int
    Next *Node
}

func middle(head *Node) *Node {
    slow, fast := head, head
    for fast != nil && fast.Next != nil {
        slow = slow.Next
        fast = fast.Next.Next
    }
    return slow
}
```

### Happy number (Python)

```python
def is_happy(n):
    def step(x):
        return sum(int(d) ** 2 for d in str(x))
    slow = n
    fast = step(n)
    while fast != 1 and slow != fast:
        slow = step(slow)
        fast = step(step(fast))
    return fast == 1

assert is_happy(19) is True
```

**Edge cases:** empty list → no cycle; single self-loop; cycle of length 1 (self-pointer).

## Prefix Sum

**Trigger:** repeated range sum queries on an immutable array, or a count of subarrays whose sum equals `k`.

**Template:**

```
prefix[0] = 0
prefix[i+1] = prefix[i] + arr[i]   for i in 0..n-1
range_sum(L, R) = prefix[R+1] - prefix[L]
```

The hashmap variant counts subarrays summing to `k` in O(n): for each prefix `s`, increment the count of `(s - k)` already seen.

### Subarray sum equals K (Python)

```python
from collections import defaultdict

def subarray_sum(arr, k):
    counts = defaultdict(int)
    counts[0] = 1
    prefix = 0
    total = 0
    for x in arr:
        prefix += x
        total += counts[prefix - k]
        counts[prefix] += 1
    return total

assert subarray_sum([1, 1, 1], 2) == 2
```

### Range sum query, immutable (Go)

```go
type NumArray struct {
    prefix []int
}

func NewNumArray(nums []int) *NumArray {
    p := make([]int, len(nums)+1)
    for i, x := range nums {
        p[i+1] = p[i] + x
    }
    return &NumArray{prefix: p}
}

func (a *NumArray) SumRange(l, r int) int {
    return a.prefix[r+1] - a.prefix[l]
}
```

### Pivot index (Python)

```python
def pivot_index(arr):
    total = sum(arr)
    left = 0
    for i, x in enumerate(arr):
        if left == total - left - x:
            return i
        left += x
    return -1

assert pivot_index([1, 7, 3, 6, 5, 6]) == 3
```

### Equal-sum splits into three parts (Go)

```go
func canThreePartsEqualSum(arr []int) bool {
    total := 0
    for _, v := range arr {
        total += v
    }
    if total%3 != 0 {
        return false
    }
    target := total / 3
    sum, parts := 0, 0
    for _, v := range arr {
        sum += v
        if sum == target {
            parts++
            sum = 0
        }
    }
    return parts >= 3
}
```

**Edge cases:** negative numbers, zero target (multiple subarrays summing to 0), empty input.

## Difference Array

**Trigger:** many range updates `[L, R] += k`, few point queries (or all queries deferred to the end). Inverse of prefix sum.

**Template:**

```
diff[L]   += k
diff[R+1] -= k
arr[i] = sum(diff[0..i])  # cumulative sum reconstructs final state
```

The brilliance: O(1) per update, O(n) total reconstruction. Brute force is O(n) per update.

### Range add queries (Python)

```python
def apply_updates(n, updates):
    diff = [0] * (n + 1)
    for start, end, k in updates:
        diff[start] += k
        diff[end + 1] -= k
    arr = [0] * n
    running = 0
    for i in range(n):
        running += diff[i]
        arr[i] = running
    return arr

print(apply_updates(5, [(1, 3, 2), (2, 4, 3), (0, 2, -2)]))
# [-2, 0, 3, 5, 3]
```

### Bookings problem — corporate flight bookings (Go)

```go
func corpFlightBookings(bookings [][]int, n int) []int {
    diff := make([]int, n+1)
    for _, b := range bookings {
        first, last, seats := b[0]-1, b[1]-1, b[2]
        diff[first] += seats
        diff[last+1] -= seats
    }
    res := make([]int, n)
    running := 0
    for i := 0; i < n; i++ {
        running += diff[i]
        res[i] = running
    }
    return res
}
```

### Carpool capacity check (Python)

```python
def car_pooling(trips, capacity):
    diff = [0] * 1001
    for passengers, start, end in trips:
        diff[start] += passengers
        diff[end] -= passengers
    running = 0
    for d in diff:
        running += d
        if running > capacity:
            return False
    return True
```

**Edge cases:** updates with `R == n - 1` mean `R+1 == n` — your diff must have `n+1` slots. Negative `k` works the same.

## Monotonic Stack

**Trigger:** for each element you need the "next greater" or "previous smaller" — anything that asks about strict ordering with the nearest neighbor.

**Template (next greater element):**

```
stack = []  # holds indices, arr[stack] strictly decreasing
for i in range(n):
    while stack and arr[stack[-1]] < arr[i]:
        j = stack.pop()
        result[j] = arr[i]
    stack.append(i)
```

Each element is pushed and popped at most once → O(n) total.

### Next greater element (Python)

```python
def next_greater(arr):
    n = len(arr)
    res = [-1] * n
    stack = []
    for i in range(n):
        while stack and arr[stack[-1]] < arr[i]:
            j = stack.pop()
            res[j] = arr[i]
        stack.append(i)
    return res

assert next_greater([2, 1, 2, 4, 3]) == [4, 2, 4, -1, -1]
```

### Daily temperatures (Go)

```go
func dailyTemperatures(t []int) []int {
    n := len(t)
    res := make([]int, n)
    stack := []int{}
    for i := 0; i < n; i++ {
        for len(stack) > 0 && t[stack[len(stack)-1]] < t[i] {
            j := stack[len(stack)-1]
            stack = stack[:len(stack)-1]
            res[j] = i - j
        }
        stack = append(stack, i)
    }
    return res
}
```

### Largest rectangle in histogram (Python)

```python
def largest_rectangle(h):
    h = h + [0]  # sentinel forces flush
    stack = []
    best = 0
    for i, x in enumerate(h):
        while stack and h[stack[-1]] > x:
            top = stack.pop()
            left = stack[-1] if stack else -1
            area = h[top] * (i - left - 1)
            if area > best:
                best = area
        stack.append(i)
    return best

assert largest_rectangle([2, 1, 5, 6, 2, 3]) == 10
```

### Trapping rain water (monotonic stack, Go)

```go
func trapStack(height []int) int {
    stack := []int{}
    water := 0
    for i, h := range height {
        for len(stack) > 0 && height[stack[len(stack)-1]] < h {
            top := stack[len(stack)-1]
            stack = stack[:len(stack)-1]
            if len(stack) == 0 {
                break
            }
            left := stack[len(stack)-1]
            width := i - left - 1
            bounded := min2(h, height[left]) - height[top]
            water += width * bounded
        }
        stack = append(stack, i)
    }
    return water
}

func min2(a, b int) int { if a < b { return a }; return b }
```

### Remove K digits (Python)

```python
def remove_k_digits(num, k):
    stack = []
    for d in num:
        while k > 0 and stack and stack[-1] > d:
            stack.pop()
            k -= 1
        stack.append(d)
    while k > 0:
        stack.pop()
        k -= 1
    res = "".join(stack).lstrip("0")
    return res or "0"

assert remove_k_digits("1432219", 3) == "1219"
```

**Edge cases:** ties (use `<` vs `<=` deliberately); empty result after deletions; sentinel value at end to drain stack.

## Monotonic Deque

**Trigger:** windowed maximum or minimum in O(n); sliding-window aggregate where pure sum doesn't work.

**Template:**

```
dq = deque()  # holds indices, arr[dq] monotonic decreasing for max-window
for i, x in enumerate(arr):
    while dq and dq[0] <= i - k:        # drop expired front
        dq.popleft()
    while dq and arr[dq[-1]] < x:       # maintain monotonicity
        dq.pop()
    dq.append(i)
    if i >= k - 1:
        out.append(arr[dq[0]])           # front is window max
```

### Max in sliding window O(n) (Python)

```python
from collections import deque

def max_sliding_window(arr, k):
    dq = deque()
    out = []
    for i, x in enumerate(arr):
        while dq and dq[0] <= i - k:
            dq.popleft()
        while dq and arr[dq[-1]] < x:
            dq.pop()
        dq.append(i)
        if i >= k - 1:
            out.append(arr[dq[0]])
    return out

assert max_sliding_window([1, 3, -1, -3, 5, 3, 6, 7], 3) == [3, 3, 5, 5, 6, 7]
```

### Shortest subarray with sum ≥ K (Go)

```go
func shortestSubarray(nums []int, k int) int {
    n := len(nums)
    prefix := make([]int, n+1)
    for i, v := range nums {
        prefix[i+1] = prefix[i] + v
    }
    dq := []int{}
    best := n + 1
    for i := 0; i <= n; i++ {
        for len(dq) > 0 && prefix[i]-prefix[dq[0]] >= k {
            if i-dq[0] < best {
                best = i - dq[0]
            }
            dq = dq[1:]
        }
        for len(dq) > 0 && prefix[dq[len(dq)-1]] >= prefix[i] {
            dq = dq[:len(dq)-1]
        }
        dq = append(dq, i)
    }
    if best == n+1 {
        return -1
    }
    return best
}
```

**Edge cases:** distinguish strict vs non-strict monotonicity carefully; `<` keeps duplicates, `<=` drops them.

## Binary Search — Classic Lower/Upper Bound

**Trigger:** sorted (or monotonic) data; you want the smallest index satisfying a predicate, or the value at that index.

**Template (leftmost-true on a predicate):**

```
lo, hi = 0, n   # hi is one past the last index
while lo < hi:
    mid = (lo + hi) // 2
    if pred(mid):
        hi = mid
    else:
        lo = mid + 1
return lo  # in [0, n]; n means no index satisfies pred
```

The loop maintains `pred(hi-1) is True or hi == n` and `pred(lo-1) is False or lo == 0`. When `lo == hi`, you've narrowed to the boundary.

### Lower bound and upper bound (Python)

```python
def lower_bound(arr, target):
    lo, hi = 0, len(arr)
    while lo < hi:
        mid = (lo + hi) // 2
        if arr[mid] < target:
            lo = mid + 1
        else:
            hi = mid
    return lo

def upper_bound(arr, target):
    lo, hi = 0, len(arr)
    while lo < hi:
        mid = (lo + hi) // 2
        if arr[mid] <= target:
            lo = mid + 1
        else:
            hi = mid
    return lo

a = [1, 2, 2, 3, 4]
assert lower_bound(a, 2) == 1
assert upper_bound(a, 2) == 3
```

### First and last position of element (Go)

```go
func searchRange(nums []int, target int) []int {
    lo := lowerBound(nums, target)
    if lo == len(nums) || nums[lo] != target {
        return []int{-1, -1}
    }
    return []int{lo, lowerBound(nums, target+1) - 1}
}

func lowerBound(nums []int, x int) int {
    lo, hi := 0, len(nums)
    for lo < hi {
        mid := (lo + hi) / 2
        if nums[mid] < x {
            lo = mid + 1
        } else {
            hi = mid
        }
    }
    return lo
}
```

### Always use the same template

Pick *one* binary-search template and use it religiously. The `lo < hi`, `hi = mid` / `lo = mid + 1` form is the most error-resistant. Avoid mixing styles in the same codebase.

**Pitfalls:**

- `mid = (lo + hi) / 2` overflows in C / Java; in Go and Python prefer `lo + (hi-lo)//2` defensively.
- If you use `hi = n - 1`, you must use `while lo <= hi` and `lo = mid + 1` / `hi = mid - 1` — *consistently*. Easy to mis-use.

## Binary Search — On Answer

**Trigger:** the question asks for an extreme value (min/max) that satisfies some monotonic predicate, but the answer is *not* an array index — it's a quantity (capacity, days, distance, time).

**Pattern:**

1. Define `feasible(x): bool` — "is `x` a valid answer?"
2. Verify `feasible` is monotonic: if `feasible(x)`, then `feasible(x')` for all `x' > x` (or vice versa).
3. Binary search on the answer space `[lo, hi]`.

### Capacity to ship in D days (Python)

```python
def ship_within_days(weights, days):
    def feasible(cap):
        used = 0
        cur = 0
        for w in weights:
            if cur + w > cap:
                used += 1
                cur = 0
            cur += w
        return used + 1 <= days

    lo, hi = max(weights), sum(weights)
    while lo < hi:
        mid = (lo + hi) // 2
        if feasible(mid):
            hi = mid
        else:
            lo = mid + 1
    return lo

assert ship_within_days([1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 5) == 15
```

### Split array largest sum (Go)

```go
func splitArray(nums []int, k int) int {
    lo, hi := 0, 0
    for _, v := range nums {
        if v > lo {
            lo = v
        }
        hi += v
    }
    feasible := func(target int) bool {
        groups := 1
        sum := 0
        for _, v := range nums {
            if sum+v > target {
                groups++
                sum = 0
            }
            sum += v
        }
        return groups <= k
    }
    for lo < hi {
        mid := (lo + hi) / 2
        if feasible(mid) {
            hi = mid
        } else {
            lo = mid + 1
        }
    }
    return lo
}
```

### Minimum days to make M bouquets (Python)

```python
def min_days(bloom, m, k):
    if m * k > len(bloom):
        return -1
    def feasible(day):
        flowers = bouquets = 0
        for d in bloom:
            if d <= day:
                flowers += 1
                if flowers == k:
                    bouquets += 1
                    flowers = 0
            else:
                flowers = 0
        return bouquets >= m
    lo, hi = min(bloom), max(bloom)
    while lo < hi:
        mid = (lo + hi) // 2
        if feasible(mid):
            hi = mid
        else:
            lo = mid + 1
    return lo
```

### Aggressive cows (Go)

```go
import "sort"

func aggressiveCows(stalls []int, cows int) int {
    sort.Ints(stalls)
    feasible := func(d int) bool {
        last := stalls[0]
        placed := 1
        for _, s := range stalls[1:] {
            if s-last >= d {
                last = s
                placed++
            }
        }
        return placed >= cows
    }
    lo, hi := 1, stalls[len(stalls)-1]-stalls[0]
    for lo < hi {
        mid := (lo + hi + 1) / 2
        if feasible(mid) {
            lo = mid
        } else {
            hi = mid - 1
        }
    }
    return lo
}
```

This last variant uses `(lo+hi+1)/2` to avoid an infinite loop when `lo = hi - 1` and `feasible(mid)` is true — that's "binary search rounding up" for "largest feasible".

## Binary Search — Variants

### Peak finding (Python)

A peak element is strictly greater than its neighbors. With `arr[-1] = arr[n] = -∞`, a peak always exists and we can find one in O(log n).

```python
def find_peak(arr):
    lo, hi = 0, len(arr) - 1
    while lo < hi:
        mid = (lo + hi) // 2
        if arr[mid] < arr[mid + 1]:
            lo = mid + 1
        else:
            hi = mid
    return lo
```

### Search in rotated sorted array (Go)

```go
func searchRotated(nums []int, target int) int {
    lo, hi := 0, len(nums)-1
    for lo <= hi {
        mid := (lo + hi) / 2
        if nums[mid] == target {
            return mid
        }
        if nums[lo] <= nums[mid] { // left half sorted
            if nums[lo] <= target && target < nums[mid] {
                hi = mid - 1
            } else {
                lo = mid + 1
            }
        } else {                    // right half sorted
            if nums[mid] < target && target <= nums[hi] {
                lo = mid + 1
            } else {
                hi = mid - 1
            }
        }
    }
    return -1
}
```

### Minimum in rotated sorted array (Python)

```python
def find_min(nums):
    lo, hi = 0, len(nums) - 1
    while lo < hi:
        mid = (lo + hi) // 2
        if nums[mid] > nums[hi]:
            lo = mid + 1
        else:
            hi = mid
    return nums[lo]
```

### Search in a 2D matrix (sorted rows + sorted columns, staircase, Go)

```go
func searchMatrix(matrix [][]int, target int) bool {
    if len(matrix) == 0 {
        return false
    }
    rows, cols := len(matrix), len(matrix[0])
    r, c := 0, cols-1
    for r < rows && c >= 0 {
        if matrix[r][c] == target {
            return true
        }
        if matrix[r][c] > target {
            c--
        } else {
            r++
        }
    }
    return false
}
```

## Merge Intervals

**Trigger:** intervals that may overlap; you need to combine overlapping ones, count meeting rooms, find gaps, etc.

**Template:**

1. Sort by start time.
2. Walk through; if `next.start <= current.end`, extend `current.end = max(current.end, next.end)`. Else flush `current` to output.

### Merge overlapping intervals (Python)

```python
def merge(intervals):
    intervals.sort(key=lambda x: x[0])
    out = []
    for start, end in intervals:
        if out and out[-1][1] >= start:
            out[-1] = (out[-1][0], max(out[-1][1], end))
        else:
            out.append((start, end))
    return out

assert merge([(1,3),(2,6),(8,10),(15,18)]) == [(1,6),(8,10),(15,18)]
```

### Insert interval (Go)

```go
func insert(intervals [][]int, newI []int) [][]int {
    res := [][]int{}
    i, n := 0, len(intervals)
    for i < n && intervals[i][1] < newI[0] {
        res = append(res, intervals[i])
        i++
    }
    for i < n && intervals[i][0] <= newI[1] {
        if intervals[i][0] < newI[0] {
            newI[0] = intervals[i][0]
        }
        if intervals[i][1] > newI[1] {
            newI[1] = intervals[i][1]
        }
        i++
    }
    res = append(res, newI)
    for i < n {
        res = append(res, intervals[i])
        i++
    }
    return res
}
```

### Non-overlapping intervals — minimum erasures (Python)

```python
def erase_overlap(intervals):
    intervals.sort(key=lambda x: x[1])
    end = float('-inf')
    removed = 0
    for s, e in intervals:
        if s >= end:
            end = e
        else:
            removed += 1
    return removed
```

### Interval list intersection (Go)

```go
func intervalIntersection(a [][]int, b [][]int) [][]int {
    res := [][]int{}
    i, j := 0, 0
    for i < len(a) && j < len(b) {
        lo := max2(a[i][0], b[j][0])
        hi := min2(a[i][1], b[j][1])
        if lo <= hi {
            res = append(res, []int{lo, hi})
        }
        if a[i][1] < b[j][1] {
            i++
        } else {
            j++
        }
    }
    return res
}

func max2(a, b int) int { if a > b { return a }; return b }
```

## Sort and Scan

**Trigger:** problem where the "right order" trivializes the rest. After sorting, a single pass with bookkeeping resolves the answer.

### 3-Sum (Python)

```python
def three_sum(nums):
    nums.sort()
    res = []
    n = len(nums)
    for i in range(n - 2):
        if i > 0 and nums[i] == nums[i - 1]:
            continue
        left, right = i + 1, n - 1
        while left < right:
            s = nums[i] + nums[left] + nums[right]
            if s < 0:
                left += 1
            elif s > 0:
                right -= 1
            else:
                res.append([nums[i], nums[left], nums[right]])
                while left < right and nums[left] == nums[left + 1]:
                    left += 1
                while left < right and nums[right] == nums[right - 1]:
                    right -= 1
                left += 1
                right -= 1
    return res
```

### Meeting rooms II — minimum rooms required (Go, heap)

```go
import (
    "container/heap"
    "sort"
)

type IntHeap []int
func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *IntHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *IntHeap) Pop() any { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

func minMeetingRooms(intervals [][]int) int {
    sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
    h := &IntHeap{}
    heap.Init(h)
    for _, it := range intervals {
        if h.Len() > 0 && (*h)[0] <= it[0] {
            heap.Pop(h)
        }
        heap.Push(h, it[1])
    }
    return h.Len()
}
```

### Car pooling (Python)

```python
def car_pooling(trips, capacity):
    events = []
    for passengers, start, end in trips:
        events.append((start, passengers))
        events.append((end, -passengers))
    events.sort()
    cur = 0
    for _, delta in events:
        cur += delta
        if cur > capacity:
            return False
    return True
```

## Greedy with Proof Obligation

A greedy algorithm makes a locally optimal choice and never reconsiders. It only works when the local choice is provably part of some globally optimal solution. The two formal foundations are *exchange arguments* and *matroid theory*.

### Activity selection — proof by exchange

Sort by finish time, then take each activity whose start ≥ last finish.

```python
def activity_select(intervals):
    intervals.sort(key=lambda x: x[1])
    end = float('-inf')
    chosen = []
    for s, e in intervals:
        if s >= end:
            chosen.append((s, e))
            end = e
    return chosen
```

**Proof sketch:** suppose an optimal schedule `OPT` doesn't pick the earliest-finishing activity `a₁`. Replace its first chosen activity with `a₁`; since `a₁` finishes earliest, all later choices in `OPT` are still valid. So a schedule starting with `a₁` is also optimal — induct on the remaining problem.

### Huffman coding — tree-merge invariant

Merge the two least-frequent symbols repeatedly. Invariant: the optimal-cost prefix tree contains the two least-frequent symbols as siblings at the deepest level.

### Minimum spanning tree — cut property

For any cut of the graph, the lightest edge crossing it belongs to *some* MST. Both Kruskal (sort edges) and Prim (extend frontier) exploit this.

### Job scheduling with deadlines — exchange argument

Sort by profit descending; place each job in the latest open slot ≤ its deadline. If you can't, skip it. Proof: any swap of an unplaced higher-profit job with a placed lower-profit job that fits cannot decrease total profit.

### When greedy fails — counter-examples

- **0/1 knapsack.** Greedy by `value/weight` fails on `[(60, 10), (100, 20), (120, 30)]` with capacity 50. Greedy picks {0,1} = 160; optimal is {1,2} = 220. Need DP.
- **Coin change with non-canonical denominations.** Coins `[1, 3, 4]`, target 6: greedy gives 4+1+1 = 3 coins; DP gives 3+3 = 2 coins.
- **TSP.** Nearest-neighbor heuristic can be arbitrarily bad.

The discipline: *if you can't sketch the proof, your greedy is probably wrong.* When in doubt, write the DP.

## Divide and Conquer — Merge

**Trigger:** problems decomposable into two independent halves whose results merge in linear time. T(n) = 2T(n/2) + O(n) → O(n log n).

### Merge sort (Python)

```python
def merge_sort(arr):
    if len(arr) <= 1:
        return arr
    mid = len(arr) // 2
    left = merge_sort(arr[:mid])
    right = merge_sort(arr[mid:])
    return _merge(left, right)

def _merge(a, b):
    out = []
    i = j = 0
    while i < len(a) and j < len(b):
        if a[i] <= b[j]:
            out.append(a[i]); i += 1
        else:
            out.append(b[j]); j += 1
    out.extend(a[i:]); out.extend(b[j:])
    return out
```

### Counting inversions (Go)

```go
func countInversions(arr []int) int64 {
    buf := make([]int, len(arr))
    var rec func(lo, hi int) int64
    rec = func(lo, hi int) int64 {
        if hi-lo <= 1 {
            return 0
        }
        mid := (lo + hi) / 2
        inv := rec(lo, mid) + rec(mid, hi)
        i, j, k := lo, mid, 0
        for i < mid && j < hi {
            if arr[i] <= arr[j] {
                buf[k] = arr[i]; i++
            } else {
                buf[k] = arr[j]; j++
                inv += int64(mid - i)
            }
            k++
        }
        for i < mid { buf[k] = arr[i]; i++; k++ }
        for j < hi { buf[k] = arr[j]; j++; k++ }
        copy(arr[lo:hi], buf[:k])
        return inv
    }
    return rec(0, len(arr))
}
```

### Closest pair of points (sketch, Python)

```python
def closest_pair(points):
    pts = sorted(points)
    def rec(lo, hi):
        if hi - lo <= 3:
            best = float('inf')
            for i in range(lo, hi):
                for j in range(i + 1, hi):
                    d = ((pts[i][0]-pts[j][0])**2 + (pts[i][1]-pts[j][1])**2) ** 0.5
                    if d < best:
                        best = d
            return best
        mid = (lo + hi) // 2
        midx = pts[mid][0]
        d = min(rec(lo, mid), rec(mid, hi))
        strip = [p for p in pts[lo:hi] if abs(p[0] - midx) < d]
        strip.sort(key=lambda p: p[1])
        for i in range(len(strip)):
            for j in range(i + 1, min(i + 8, len(strip))):
                dd = ((strip[i][0]-strip[j][0])**2 + (strip[i][1]-strip[j][1])**2) ** 0.5
                if dd < d:
                    d = dd
        return d
    return rec(0, len(pts))
```

## Divide and Conquer — Quick Selection

**Trigger:** find the kth smallest in O(n) average time without sorting the whole array.

### Quickselect (Python)

```python
import random

def quickselect(arr, k):
    arr = list(arr)
    lo, hi = 0, len(arr) - 1
    while lo <= hi:
        pivot_idx = random.randint(lo, hi)
        pivot = arr[pivot_idx]
        arr[pivot_idx], arr[hi] = arr[hi], arr[pivot_idx]
        store = lo
        for i in range(lo, hi):
            if arr[i] < pivot:
                arr[i], arr[store] = arr[store], arr[i]
                store += 1
        arr[store], arr[hi] = arr[hi], arr[store]
        if store == k:
            return arr[store]
        if store < k:
            lo = store + 1
        else:
            hi = store - 1
    return -1

assert quickselect([3, 1, 5, 2, 4], 2) == 3  # 0-indexed kth smallest
```

### Kth largest (Go)

```go
import "math/rand"

func quickselect(arr []int, k int) int {
    lo, hi := 0, len(arr)-1
    for lo <= hi {
        p := lo + rand.Intn(hi-lo+1)
        pivot := arr[p]
        arr[p], arr[hi] = arr[hi], arr[p]
        store := lo
        for i := lo; i < hi; i++ {
            if arr[i] < pivot {
                arr[i], arr[store] = arr[store], arr[i]
                store++
            }
        }
        arr[store], arr[hi] = arr[hi], arr[store]
        if store == k {
            return arr[store]
        }
        if store < k {
            lo = store + 1
        } else {
            hi = store - 1
        }
    }
    return -1
}
```

For the *kth largest*, call `quickselect(arr, len(arr) - k)`.

## Divide and Conquer — Strassen / Karatsuba

These give *sub-quadratic* multiplication algorithms by trading multiplications for additions.

### Karatsuba — integer multiplication, O(n^log2(3)) ≈ O(n^1.585)

For numbers `x = x1·B + x0` and `y = y1·B + y0`, normal multiplication uses 4 sub-products:

```
xy = x1·y1·B² + (x1·y0 + x0·y1)·B + x0·y0
```

Karatsuba uses 3:

```
z2 = x1·y1
z0 = x0·y0
z1 = (x1+x0)·(y1+y0) - z2 - z0
xy = z2·B² + z1·B + z0
```

```python
def karatsuba(x, y):
    if x < 10 or y < 10:
        return x * y
    n = max(len(str(x)), len(str(y)))
    half = n // 2
    B = 10 ** half
    x1, x0 = divmod(x, B)
    y1, y0 = divmod(y, B)
    z2 = karatsuba(x1, y1)
    z0 = karatsuba(x0, y0)
    z1 = karatsuba(x1 + x0, y1 + y0) - z2 - z0
    return z2 * (B ** 2) + z1 * B + z0
```

### Strassen — matrix multiplication, O(n^log2(7)) ≈ O(n^2.81)

Standard matrix multiply does 8 multiplications per 2×2 block; Strassen reduces to 7. Practical speedup begins around `n ≈ 64` due to constant-factor cost. Used as a kernel inside more advanced asymptotically-faster algorithms (Coppersmith-Winograd family). For a working implementation, see CLRS or numerical-recipes references.

## Dynamic Programming — Recognition

DP is the algorithmic technique for problems with two structural properties:

1. **Optimal substructure** — the optimal solution can be assembled from optimal solutions of subproblems.
2. **Overlapping subproblems** — naive recursion would recompute the same subproblems many times.

Recognition triggers:

- A problem asking for the *count* or *best* value over many decisions or combinations.
- A recursive formulation that produces the same arguments repeatedly.
- A problem on a DAG of choices (sometimes implicit).

Once you recognize DP, the implementation has three flavors:

- **Memoized recursion (top-down).** Easiest to derive from the recurrence; can blow stack on deep states.
- **Bottom-up table.** Compute states in dependency order; cache-friendly.
- **Rolling array.** When `dp[i]` only depends on `dp[i-1]` (or a few priors), keep just those.

The single most useful DP exercise: write the recurrence, then translate to memoized recursion, *then* convert to bottom-up if performance demands it.

## DP — 1D

**Pattern:** `state[i]` depends on a constant number of earlier `state[j]` for `j < i`.

### Fibonacci (Python)

```python
def fib(n):
    if n < 2:
        return n
    a, b = 0, 1
    for _ in range(n - 1):
        a, b = b, a + b
    return b
```

### Climbing stairs (Go)

```go
func climbStairs(n int) int {
    if n < 2 {
        return 1
    }
    a, b := 1, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}
```

### House robber (Python)

```python
def rob(nums):
    prev, cur = 0, 0
    for x in nums:
        prev, cur = cur, max(cur, prev + x)
    return cur
```

### Maximum subarray (Kadane's algorithm, Go)

```go
func maxSubArray(nums []int) int {
    cur, best := nums[0], nums[0]
    for i := 1; i < len(nums); i++ {
        if nums[i] > cur+nums[i] {
            cur = nums[i]
        } else {
            cur += nums[i]
        }
        if cur > best {
            best = cur
        }
    }
    return best
}
```

### Longest increasing subsequence — O(n²) (Python)

```python
def lis(nums):
    if not nums:
        return 0
    dp = [1] * len(nums)
    for i in range(len(nums)):
        for j in range(i):
            if nums[j] < nums[i]:
                dp[i] = max(dp[i], dp[j] + 1)
    return max(dp)
```

## DP — 2D

**Pattern:** `state[i][j]` depends on `state[i-1][j]`, `state[i][j-1]`, and/or `state[i-1][j-1]`.

### Edit distance (Levenshtein, Python)

```python
def edit_distance(a, b):
    m, n = len(a), len(b)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(m + 1):
        dp[i][0] = i
    for j in range(n + 1):
        dp[0][j] = j
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if a[i - 1] == b[j - 1]:
                dp[i][j] = dp[i - 1][j - 1]
            else:
                dp[i][j] = 1 + min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
    return dp[m][n]

assert edit_distance("kitten", "sitting") == 3
```

### Longest common subsequence (Go)

```go
func lcs(a, b string) int {
    m, n := len(a), len(b)
    dp := make([][]int, m+1)
    for i := range dp {
        dp[i] = make([]int, n+1)
    }
    for i := 1; i <= m; i++ {
        for j := 1; j <= n; j++ {
            if a[i-1] == b[j-1] {
                dp[i][j] = dp[i-1][j-1] + 1
            } else if dp[i-1][j] > dp[i][j-1] {
                dp[i][j] = dp[i-1][j]
            } else {
                dp[i][j] = dp[i][j-1]
            }
        }
    }
    return dp[m][n]
}
```

### Unique paths in m×n grid (Python)

```python
def unique_paths(m, n):
    dp = [1] * n
    for _ in range(1, m):
        for j in range(1, n):
            dp[j] += dp[j - 1]
    return dp[-1]
```

### Minimum path sum (Go)

```go
func minPathSum(grid [][]int) int {
    m, n := len(grid), len(grid[0])
    for i := 0; i < m; i++ {
        for j := 0; j < n; j++ {
            if i == 0 && j == 0 {
                continue
            }
            if i == 0 {
                grid[i][j] += grid[i][j-1]
            } else if j == 0 {
                grid[i][j] += grid[i-1][j]
            } else {
                if grid[i-1][j] < grid[i][j-1] {
                    grid[i][j] += grid[i-1][j]
                } else {
                    grid[i][j] += grid[i][j-1]
                }
            }
        }
    }
    return grid[m-1][n-1]
}
```

### Maximal square in binary matrix (Python)

```python
def maximal_square(matrix):
    if not matrix:
        return 0
    m, n = len(matrix), len(matrix[0])
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    best = 0
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if matrix[i - 1][j - 1] == '1':
                dp[i][j] = 1 + min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
                if dp[i][j] > best:
                    best = dp[i][j]
    return best * best
```

### Regex matching (Python)

```python
def is_match(s, p):
    m, n = len(s), len(p)
    dp = [[False] * (n + 1) for _ in range(m + 1)]
    dp[0][0] = True
    for j in range(1, n + 1):
        if p[j - 1] == '*':
            dp[0][j] = dp[0][j - 2]
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if p[j - 1] == '*':
                dp[i][j] = dp[i][j - 2]
                if p[j - 2] == s[i - 1] or p[j - 2] == '.':
                    dp[i][j] = dp[i][j] or dp[i - 1][j]
            elif p[j - 1] == s[i - 1] or p[j - 1] == '.':
                dp[i][j] = dp[i - 1][j - 1]
    return dp[m][n]
```

## DP — Knapsack 0/1

**Pattern:** items used at most once; capacity `W`; pick subset to maximize value.

**Recurrence:** `dp[i][w] = max(dp[i-1][w], dp[i-1][w - weight[i]] + value[i])`.

**1D rolling:** iterate `w` from `W` down to `weight[i]` so each item is used once.

### 0/1 knapsack (Python)

```python
def knapsack_01(weights, values, W):
    dp = [0] * (W + 1)
    for i, w in enumerate(weights):
        for cap in range(W, w - 1, -1):
            if dp[cap - w] + values[i] > dp[cap]:
                dp[cap] = dp[cap - w] + values[i]
    return dp[W]

assert knapsack_01([2, 3, 4, 5], [3, 4, 5, 6], 5) == 7
```

### Subset sum (Go)

```go
func subsetSum(nums []int, target int) bool {
    dp := make([]bool, target+1)
    dp[0] = true
    for _, v := range nums {
        for c := target; c >= v; c-- {
            if dp[c-v] {
                dp[c] = true
            }
        }
    }
    return dp[target]
}
```

### Partition equal subset sum (Python)

```python
def can_partition(nums):
    total = sum(nums)
    if total % 2 != 0:
        return False
    target = total // 2
    dp = [False] * (target + 1)
    dp[0] = True
    for x in nums:
        for c in range(target, x - 1, -1):
            dp[c] = dp[c] or dp[c - x]
    return dp[target]
```

### Target sum — count assignments (Go)

```go
func findTargetSumWays(nums []int, target int) int {
    sum := 0
    for _, v := range nums { sum += v }
    if (sum+target)%2 != 0 || sum < abs(target) {
        return 0
    }
    s := (sum + target) / 2
    dp := make([]int, s+1)
    dp[0] = 1
    for _, v := range nums {
        for c := s; c >= v; c-- {
            dp[c] += dp[c-v]
        }
    }
    return dp[s]
}

func abs(x int) int { if x < 0 { return -x }; return x }
```

## DP — Unbounded Knapsack

**Pattern:** items reusable; iterate `w` *forward* so the same item can be picked again at the same outer level.

### Coin change — minimum coins (Python)

```python
def coin_change(coins, amount):
    dp = [amount + 1] * (amount + 1)
    dp[0] = 0
    for c in coins:
        for x in range(c, amount + 1):
            if dp[x - c] + 1 < dp[x]:
                dp[x] = dp[x - c] + 1
    return -1 if dp[amount] > amount else dp[amount]

assert coin_change([1, 2, 5], 11) == 3
```

### Coin change — number of ways (Go)

```go
func change(amount int, coins []int) int {
    dp := make([]int, amount+1)
    dp[0] = 1
    for _, c := range coins {
        for x := c; x <= amount; x++ {
            dp[x] += dp[x-c]
        }
    }
    return dp[amount]
}
```

### Combination sum IV — order matters (Python)

```python
def combination_sum4(nums, target):
    dp = [0] * (target + 1)
    dp[0] = 1
    for x in range(1, target + 1):
        for v in nums:
            if x >= v:
                dp[x] += dp[x - v]
    return dp[target]
```

The order of loops matters: outer over capacity ⇒ permutations; outer over items ⇒ combinations.

### Word break (Go)

```go
func wordBreak(s string, dict []string) bool {
    set := map[string]bool{}
    for _, w := range dict {
        set[w] = true
    }
    n := len(s)
    dp := make([]bool, n+1)
    dp[0] = true
    for i := 1; i <= n; i++ {
        for j := 0; j < i; j++ {
            if dp[j] && set[s[j:i]] {
                dp[i] = true
                break
            }
        }
    }
    return dp[n]
}
```

## DP — Longest Increasing Subsequence

### O(n²) DP (Python)

Already shown in 1D section.

### O(n log n) with binary search (patience sort, Python)

```python
import bisect

def lis_fast(nums):
    tails = []
    for x in nums:
        i = bisect.bisect_left(tails, x)
        if i == len(tails):
            tails.append(x)
        else:
            tails[i] = x
    return len(tails)

assert lis_fast([10, 9, 2, 5, 3, 7, 101, 18]) == 4
```

### LIS reconstruction (Python)

```python
import bisect

def lis_reconstruct(nums):
    tails, tails_idx, prev = [], [], [-1] * len(nums)
    for i, x in enumerate(nums):
        j = bisect.bisect_left(tails, x)
        if j == len(tails):
            tails.append(x)
            tails_idx.append(i)
        else:
            tails[j] = x
            tails_idx[j] = i
        if j > 0:
            prev[i] = tails_idx[j - 1]
    out = []
    k = tails_idx[-1] if tails_idx else -1
    while k != -1:
        out.append(nums[k])
        k = prev[k]
    return list(reversed(out))
```

### LIS in Go (binary search variant)

```go
import "sort"

func lengthOfLIS(nums []int) int {
    tails := []int{}
    for _, x := range nums {
        i := sort.SearchInts(tails, x)
        if i == len(tails) {
            tails = append(tails, x)
        } else {
            tails[i] = x
        }
    }
    return len(tails)
}
```

## DP — Longest Common Subsequence / Substring

### LCS table + path reconstruction (Python)

```python
def lcs_path(a, b):
    m, n = len(a), len(b)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if a[i - 1] == b[j - 1]:
                dp[i][j] = dp[i - 1][j - 1] + 1
            else:
                dp[i][j] = max(dp[i - 1][j], dp[i][j - 1])
    out = []
    i, j = m, n
    while i and j:
        if a[i - 1] == b[j - 1]:
            out.append(a[i - 1])
            i -= 1; j -= 1
        elif dp[i - 1][j] >= dp[i][j - 1]:
            i -= 1
        else:
            j -= 1
    return "".join(reversed(out))

assert lcs_path("ABCBDAB", "BDCABA") == "BDAB"
```

### Longest common substring (Go)

```go
func longestCommonSubstring(a, b string) string {
    m, n := len(a), len(b)
    dp := make([][]int, m+1)
    for i := range dp {
        dp[i] = make([]int, n+1)
    }
    bestLen, bestEnd := 0, 0
    for i := 1; i <= m; i++ {
        for j := 1; j <= n; j++ {
            if a[i-1] == b[j-1] {
                dp[i][j] = dp[i-1][j-1] + 1
                if dp[i][j] > bestLen {
                    bestLen = dp[i][j]
                    bestEnd = i
                }
            }
        }
    }
    return a[bestEnd-bestLen : bestEnd]
}
```

## DP — Edit Distance

Already shown above. The standard Levenshtein distance recurrence is:

```
dp[i][j] = 0 if i == 0 == j
         = i if j == 0
         = j if i == 0
         = dp[i-1][j-1]                              if a[i-1] == b[j-1]
         = 1 + min(dp[i-1][j], dp[i][j-1], dp[i-1][j-1]) otherwise
```

Variations:

- **Hamming distance.** Same length only; count mismatches.
- **Damerau-Levenshtein.** Adds adjacent transposition as a single operation.
- **Longest common subsequence.** Replace gives 0 when characters match, ∞ otherwise; result encodes LCS length via `dp[m][n] = m + n - 2·LCS`.

## DP — Interval DP

**Pattern:** `dp[i][j]` is the answer on the *subarray* (or subsequence) `[i..j]`. Iterate by interval *length*, then start position.

### Matrix chain multiplication (Python)

```python
def matrix_chain_order(p):
    n = len(p) - 1
    dp = [[0] * n for _ in range(n)]
    for length in range(2, n + 1):
        for i in range(n - length + 1):
            j = i + length - 1
            dp[i][j] = float('inf')
            for k in range(i, j):
                cost = dp[i][k] + dp[k + 1][j] + p[i] * p[k + 1] * p[j + 1]
                if cost < dp[i][j]:
                    dp[i][j] = cost
    return dp[0][n - 1]

assert matrix_chain_order([10, 30, 5, 60]) == 4500
```

### Burst balloons (Go)

```go
func maxCoins(nums []int) int {
    n := len(nums) + 2
    pad := make([]int, n)
    pad[0], pad[n-1] = 1, 1
    copy(pad[1:], nums)
    dp := make([][]int, n)
    for i := range dp {
        dp[i] = make([]int, n)
    }
    for length := 2; length < n; length++ {
        for i := 0; i+length < n; i++ {
            j := i + length
            for k := i + 1; k < j; k++ {
                v := dp[i][k] + dp[k][j] + pad[i]*pad[k]*pad[j]
                if v > dp[i][j] {
                    dp[i][j] = v
                }
            }
        }
    }
    return dp[0][n-1]
}
```

### Palindrome partitioning — minimum cuts (Python)

```python
def min_cuts(s):
    n = len(s)
    is_pal = [[False] * n for _ in range(n)]
    for i in range(n):
        is_pal[i][i] = True
    for length in range(2, n + 1):
        for i in range(n - length + 1):
            j = i + length - 1
            if s[i] == s[j]:
                is_pal[i][j] = length == 2 or is_pal[i + 1][j - 1]
    cuts = list(range(n))
    for i in range(n):
        if is_pal[0][i]:
            cuts[i] = 0
            continue
        for j in range(1, i + 1):
            if is_pal[j][i] and cuts[j - 1] + 1 < cuts[i]:
                cuts[i] = cuts[j - 1] + 1
    return cuts[-1]

assert min_cuts("aab") == 1
```

### Merge stones (sketch)

`dp[i][j][k]` = minimum cost to merge `stones[i..j]` into exactly `k` piles. Iterate by length, then by `k`. Used in competition DP problems.

## DP — Tree DP

**Pattern:** post-order DFS — child results bubble up; each node combines them; sometimes followed by a "rerooting" pass that sends parent state down.

### Diameter of tree (Python)

```python
class TreeNode:
    def __init__(self, val, children=None):
        self.val = val
        self.children = children or []

def diameter(root):
    best = 0
    def depth(node):
        nonlocal best
        if not node:
            return 0
        first = second = 0
        for c in node.children:
            d = depth(c)
            if d > first:
                second = first
                first = d
            elif d > second:
                second = d
        if first + second > best:
            best = first + second
        return first + 1
    depth(root)
    return best
```

### House robber III (binary tree, Go)

```go
type Tree struct {
    Val         int
    Left, Right *Tree
}

func robTree(root *Tree) int {
    var rob func(*Tree) (int, int)
    rob = func(n *Tree) (int, int) {
        if n == nil {
            return 0, 0
        }
        lWith, lWithout := rob(n.Left)
        rWith, rWithout := rob(n.Right)
        with := n.Val + lWithout + rWithout
        wo := max2(lWith, lWithout) + max2(rWith, rWithout)
        return with, wo
    }
    a, b := rob(root)
    return max2(a, b)
}
```

### Binary tree maximum path sum (Python)

```python
class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val; self.left = left; self.right = right

def max_path_sum(root):
    best = float('-inf')
    def gain(n):
        nonlocal best
        if not n:
            return 0
        left = max(gain(n.left), 0)
        right = max(gain(n.right), 0)
        path = n.val + left + right
        if path > best:
            best = path
        return n.val + max(left, right)
    gain(root)
    return best
```

### Rerooting technique (Python sketch)

```python
def rerooting(adj):
    n = len(adj)
    down = [0] * n
    up = [0] * n

    def dfs1(v, p):
        for u in adj[v]:
            if u != p:
                dfs1(u, v)
                down[v] = max(down[v], down[u] + 1)

    def dfs2(v, p):
        for u in adj[v]:
            if u != p:
                # propagate up state to u based on v's other children
                up[u] = max(up[v], down[v]) + 1
                dfs2(u, v)

    dfs1(0, -1)
    dfs2(0, -1)
    return [max(down[v], up[v]) for v in range(n)]
```

Rerooting computes per-node answers (e.g., distance to farthest node) in O(n) total instead of O(n²).

## DP — Bitmask DP

**Pattern:** state is a *subset* encoded as a bitmask. Useful when `n ≤ ~20`.

### Travelling salesman — O(2^n · n²) (Python)

```python
import math

def tsp(dist):
    n = len(dist)
    INF = float('inf')
    dp = [[INF] * n for _ in range(1 << n)]
    dp[1][0] = 0
    for mask in range(1, 1 << n):
        for u in range(n):
            if not (mask & (1 << u)):
                continue
            for v in range(n):
                if mask & (1 << v):
                    continue
                next_mask = mask | (1 << v)
                cost = dp[mask][u] + dist[u][v]
                if cost < dp[next_mask][v]:
                    dp[next_mask][v] = cost
    full = (1 << n) - 1
    return min(dp[full][u] + dist[u][0] for u in range(1, n))
```

### Assigning jobs to workers (Go)

```go
func assignJobs(cost [][]int) int {
    n := len(cost)
    INF := 1 << 30
    dp := make([]int, 1<<n)
    for i := range dp { dp[i] = INF }
    dp[0] = 0
    for mask := 0; mask < (1 << n); mask++ {
        if dp[mask] == INF { continue }
        worker := bitsCount(mask)
        if worker == n { continue }
        for j := 0; j < n; j++ {
            if mask&(1<<j) != 0 { continue }
            next := mask | (1 << j)
            v := dp[mask] + cost[worker][j]
            if v < dp[next] { dp[next] = v }
        }
    }
    return dp[(1<<n)-1]
}

func bitsCount(x int) int {
    c := 0
    for x > 0 { c += x & 1; x >>= 1 }
    return c
}
```

### Smallest sufficient team — bitmask over skills (Python)

```python
def smallest_sufficient_team(req_skills, people):
    n = len(req_skills)
    skill_idx = {s: i for i, s in enumerate(req_skills)}
    masks = [0] * len(people)
    for i, p in enumerate(people):
        for s in p:
            if s in skill_idx:
                masks[i] |= 1 << skill_idx[s]
    full = (1 << n) - 1
    INF = float('inf')
    dp = {0: []}
    for i, m in enumerate(masks):
        for state in list(dp.keys()):
            new_state = state | m
            if new_state == state:
                continue
            new_team = dp[state] + [i]
            if new_state not in dp or len(dp[new_state]) > len(new_team):
                dp[new_state] = new_team
    return dp[full]
```

### Partition into K equal sum subsets (Go)

```go
func canPartitionKSubsets(nums []int, k int) bool {
    sum := 0
    for _, v := range nums { sum += v }
    if sum%k != 0 { return false }
    target := sum / k
    n := len(nums)
    used := make([]bool, 1<<n)
    sums := make([]int, 1<<n)
    used[0] = true
    for mask := 0; mask < (1 << n); mask++ {
        if !used[mask] { continue }
        for i := 0; i < n; i++ {
            if mask&(1<<i) != 0 { continue }
            cand := sums[mask] + nums[i]
            if cand%target > target { continue }
            if cand%target <= target {
                next := mask | (1 << i)
                used[next] = true
                sums[next] = cand
            }
        }
    }
    return used[(1<<n)-1] && sums[(1<<n)-1] == sum
}
```

## DP — Digit DP

**Pattern:** count integers ≤ N satisfying some digit property. State usually includes:

- `pos` — current digit position (left to right)
- `tight` — whether we're still bounded by N's prefix
- `started` — whether we've placed a non-zero digit
- task-specific state (sum, last digit, etc.)

### Count numbers ≤ N with digit sum equal to S (Python)

```python
from functools import lru_cache

def count_with_digit_sum(N, S):
    digits = list(map(int, str(N)))
    n = len(digits)
    @lru_cache(maxsize=None)
    def dp(pos, total, tight):
        if pos == n:
            return 1 if total == S else 0
        upper = digits[pos] if tight else 9
        result = 0
        for d in range(upper + 1):
            if total + d > S:
                break
            result += dp(pos + 1, total + d, tight and d == upper)
        return result
    return dp(0, 0, True)
```

### Non-negative integers without consecutive ones (Python)

```python
def find_integers(N):
    bits = bin(N)[2:]
    n = len(bits)
    @lru_cache(maxsize=None)
    def dp(pos, prev, tight):
        if pos == n:
            return 1
        upper = int(bits[pos]) if tight else 1
        total = 0
        for d in range(upper + 1):
            if d == 1 and prev == 1:
                continue
            total += dp(pos + 1, d, tight and d == upper)
        return total
    return dp(0, 0, True)
```

## DP — Probability and Expected Value

### Knight on a chessboard — probability of staying on board after K moves (Python)

```python
def knight_prob(N, K, r, c):
    dirs = [(-2,-1),(-2,1),(-1,-2),(-1,2),(1,-2),(1,2),(2,-1),(2,1)]
    grid = [[0.0] * N for _ in range(N)]
    grid[r][c] = 1.0
    for _ in range(K):
        nxt = [[0.0] * N for _ in range(N)]
        for i in range(N):
            for j in range(N):
                if grid[i][j] == 0:
                    continue
                for dr, dc in dirs:
                    ni, nj = i + dr, j + dc
                    if 0 <= ni < N and 0 <= nj < N:
                        nxt[ni][nj] += grid[i][j] / 8
        grid = nxt
    return sum(sum(row) for row in grid)
```

### Expected dice rolls — bounded random walks

For a random walk on a finite Markov chain, expected hitting times satisfy linear equations `E[h(s)] = 1 + Σ p(s, s') · E[h(s')]` for non-terminal states. Solve via Gaussian elimination or, if the chain is acyclic on the state graph, by topological-order DP.

## DP — Game Theory / Minimax

**Pattern:** two perfectly-rational adversaries. `dp[state] = best outcome for the player to move, assuming the opponent then plays optimally`.

### Stone game — even number of piles, win or lose (Python)

```python
def stone_game(piles):
    n = len(piles)
    dp = [[0] * n for _ in range(n)]
    for i in range(n):
        dp[i][i] = piles[i]
    for length in range(2, n + 1):
        for i in range(n - length + 1):
            j = i + length - 1
            dp[i][j] = max(piles[i] - dp[i + 1][j], piles[j] - dp[i][j - 1])
    return dp[0][n - 1] > 0
```

### Predict the winner (Go)

```go
func predictTheWinner(nums []int) bool {
    n := len(nums)
    dp := make([][]int, n)
    for i := range dp {
        dp[i] = make([]int, n)
        dp[i][i] = nums[i]
    }
    for length := 2; length <= n; length++ {
        for i := 0; i+length-1 < n; i++ {
            j := i + length - 1
            a := nums[i] - dp[i+1][j]
            b := nums[j] - dp[i][j-1]
            if a > b { dp[i][j] = a } else { dp[i][j] = b }
        }
    }
    return dp[0][n-1] >= 0
}
```

### Nim with single pile — XOR sum

The Sprague-Grundy theorem: in a multi-pile Nim game, the first player loses if and only if the XOR of all pile sizes is 0.

## Backtracking

**Pattern:** explore a tree of choices via recursion. At each node:

1. **Choose** — make a candidate selection.
2. **Explore** — recurse.
3. **Unchoose** — restore state for the next candidate.

Backtracking is DP without memoization — appropriate when the state space is large but solutions are sparse, or when you need *all* solutions, not just one optimum.

### Permutations (Python)

```python
def permutations(nums):
    res = []
    def bt(path, used):
        if len(path) == len(nums):
            res.append(path[:])
            return
        for i in range(len(nums)):
            if used[i]:
                continue
            used[i] = True
            path.append(nums[i])
            bt(path, used)
            path.pop()
            used[i] = False
    bt([], [False] * len(nums))
    return res
```

### Subsets (Go)

```go
func subsets(nums []int) [][]int {
    res := [][]int{}
    var bt func(start int, path []int)
    bt = func(start int, path []int) {
        cp := make([]int, len(path))
        copy(cp, path)
        res = append(res, cp)
        for i := start; i < len(nums); i++ {
            path = append(path, nums[i])
            bt(i+1, path)
            path = path[:len(path)-1]
        }
    }
    bt(0, []int{})
    return res
}
```

### Combination sum — values reusable (Python)

```python
def combination_sum(candidates, target):
    candidates.sort()
    res = []
    def bt(start, path, remaining):
        if remaining == 0:
            res.append(path[:])
            return
        for i in range(start, len(candidates)):
            v = candidates[i]
            if v > remaining:
                break
            path.append(v)
            bt(i, path, remaining - v)
            path.pop()
    bt(0, [], target)
    return res
```

### N-Queens (Python)

```python
def solve_n_queens(n):
    cols, diag1, diag2 = set(), set(), set()
    res = []
    def bt(r, board):
        if r == n:
            res.append(["".join(row) for row in board])
            return
        for c in range(n):
            if c in cols or (r - c) in diag1 or (r + c) in diag2:
                continue
            cols.add(c); diag1.add(r - c); diag2.add(r + c)
            board[r][c] = 'Q'
            bt(r + 1, board)
            board[r][c] = '.'
            cols.remove(c); diag1.remove(r - c); diag2.remove(r + c)
    bt(0, [['.'] * n for _ in range(n)])
    return res

print(len(solve_n_queens(8)))  # 92
```

### Word search (Go, grid)

```go
func wordExists(board [][]byte, word string) bool {
    m, n := len(board), len(board[0])
    var dfs func(r, c, k int) bool
    dfs = func(r, c, k int) bool {
        if k == len(word) { return true }
        if r < 0 || r >= m || c < 0 || c >= n || board[r][c] != word[k] {
            return false
        }
        ch := board[r][c]
        board[r][c] = '#'
        ok := dfs(r+1, c, k+1) || dfs(r-1, c, k+1) || dfs(r, c+1, k+1) || dfs(r, c-1, k+1)
        board[r][c] = ch
        return ok
    }
    for i := 0; i < m; i++ {
        for j := 0; j < n; j++ {
            if dfs(i, j, 0) { return true }
        }
    }
    return false
}
```

### Sudoku solver (Python)

```python
def solve_sudoku(board):
    def is_valid(r, c, ch):
        for i in range(9):
            if board[r][i] == ch or board[i][c] == ch:
                return False
            br, bc = 3 * (r // 3) + i // 3, 3 * (c // 3) + i % 3
            if board[br][bc] == ch:
                return False
        return True

    def bt():
        for r in range(9):
            for c in range(9):
                if board[r][c] == '.':
                    for ch in '123456789':
                        if is_valid(r, c, ch):
                            board[r][c] = ch
                            if bt():
                                return True
                            board[r][c] = '.'
                    return False
        return True
    bt()
```

### Word break II — return all sentences (Python)

```python
def word_break_all(s, wordDict):
    word_set = set(wordDict)
    memo = {}
    def helper(start):
        if start in memo:
            return memo[start]
        if start == len(s):
            return [""]
        out = []
        for end in range(start + 1, len(s) + 1):
            piece = s[start:end]
            if piece in word_set:
                for tail in helper(end):
                    out.append(piece + (" " + tail if tail else ""))
        memo[start] = out
        return out
    return helper(0)
```

## Backtracking — Pruning

The naive backtracking tree grows exponentially. Pruning is the difference between solving in a millisecond and timing out.

- **Sort + skip duplicates.** When generating combinations from sorted input with repeats, `if i > start and arr[i] == arr[i-1]: continue` avoids duplicate branches.
- **Bound checking.** If `current_value > best_so_far` and the function is monotonic, return immediately.
- **Early termination.** Once a single answer is found and only one is needed, propagate `True` up the recursion.
- **Constraint propagation.** Sudoku solvers can use bitmasks per row/column/box; flagging the most-constrained cell first slashes the search space.
- **Symmetry breaking.** When the problem has symmetric solutions (e.g., N-Queens reflective symmetry), force lexicographic ordering on the first row to halve enumeration.

### Combination sum II — duplicates allowed in input but each used once (Python)

```python
def combination_sum2(candidates, target):
    candidates.sort()
    res = []
    def bt(start, path, remaining):
        if remaining == 0:
            res.append(path[:])
            return
        for i in range(start, len(candidates)):
            if i > start and candidates[i] == candidates[i - 1]:
                continue
            v = candidates[i]
            if v > remaining:
                break
            path.append(v)
            bt(i + 1, path, remaining - v)
            path.pop()
    bt(0, [], target)
    return res
```

## BFS

**Pattern:** explore a graph (or implicit graph) layer by layer using a queue. The first time you reach a target equals the shortest unweighted distance.

### Shortest path in unweighted graph (Python)

```python
from collections import deque

def shortest_path(adj, src, dst):
    if src == dst:
        return 0
    seen = {src}
    q = deque([(src, 0)])
    while q:
        u, d = q.popleft()
        for v in adj[u]:
            if v == dst:
                return d + 1
            if v not in seen:
                seen.add(v)
                q.append((v, d + 1))
    return -1
```

### Word ladder (Go)

```go
import "strings"

func ladderLength(beginWord, endWord string, wordList []string) int {
    set := map[string]bool{}
    for _, w := range wordList {
        set[w] = true
    }
    if !set[endWord] { return 0 }
    type pair struct { word string; depth int }
    q := []pair{{beginWord, 1}}
    for len(q) > 0 {
        cur := q[0]; q = q[1:]
        if cur.word == endWord { return cur.depth }
        bs := []byte(cur.word)
        for i := 0; i < len(bs); i++ {
            orig := bs[i]
            for c := byte('a'); c <= 'z'; c++ {
                if c == orig { continue }
                bs[i] = c
                w := string(bs)
                if set[w] {
                    delete(set, w)
                    q = append(q, pair{w, cur.depth + 1})
                }
            }
            bs[i] = orig
        }
    }
    return 0
    _ = strings.Contains // referenced to silence unused
}
```

### Rotting oranges (Python, multi-source)

```python
def oranges_rotting(grid):
    from collections import deque
    rows, cols = len(grid), len(grid[0])
    q = deque()
    fresh = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == 2:
                q.append((r, c, 0))
            elif grid[r][c] == 1:
                fresh += 1
    minutes = 0
    while q:
        r, c, t = q.popleft()
        minutes = max(minutes, t)
        for dr, dc in [(1,0),(-1,0),(0,1),(0,-1)]:
            nr, nc = r + dr, c + dc
            if 0 <= nr < rows and 0 <= nc < cols and grid[nr][nc] == 1:
                grid[nr][nc] = 2
                fresh -= 1
                q.append((nr, nc, t + 1))
    return -1 if fresh else minutes
```

### Minimum knight moves (Go)

```go
func minKnightMoves(x, y int) int {
    if x == 0 && y == 0 { return 0 }
    dirs := [][2]int{{2,1},{2,-1},{-2,1},{-2,-1},{1,2},{1,-2},{-1,2},{-1,-2}}
    type pos struct{ r, c, d int }
    seen := map[[2]int]bool{{0,0}:true}
    q := []pos{{0,0,0}}
    for len(q) > 0 {
        cur := q[0]; q = q[1:]
        for _, d := range dirs {
            nr, nc := cur.r+d[0], cur.c+d[1]
            if nr == x && nc == y { return cur.d + 1 }
            key := [2]int{nr, nc}
            if !seen[key] && nr > -3 && nc > -3 {
                seen[key] = true
                q = append(q, pos{nr, nc, cur.d + 1})
            }
        }
    }
    return -1
}
```

### Binary tree level order (Python)

```python
def level_order(root):
    from collections import deque
    if not root: return []
    q = deque([root])
    out = []
    while q:
        level = []
        for _ in range(len(q)):
            n = q.popleft()
            level.append(n.val)
            if n.left: q.append(n.left)
            if n.right: q.append(n.right)
        out.append(level)
    return out
```

## DFS — Recursive

**Pattern:** explore as deep as possible, then back up. Natural fit for problems with recursive substructure (trees, paths, partitioning).

### Tree pre/in/post order (Python)

```python
def preorder(root, out=None):
    if out is None: out = []
    if not root: return out
    out.append(root.val)
    preorder(root.left, out)
    preorder(root.right, out)
    return out

def inorder(root, out=None):
    if out is None: out = []
    if not root: return out
    inorder(root.left, out)
    out.append(root.val)
    inorder(root.right, out)
    return out
```

### Graph DFS with parent tracking (Go)

```go
func graphDFS(adj [][]int, start int) []int {
    visited := make([]bool, len(adj))
    out := []int{}
    var dfs func(u int)
    dfs = func(u int) {
        if visited[u] { return }
        visited[u] = true
        out = append(out, u)
        for _, v := range adj[u] {
            dfs(v)
        }
    }
    dfs(start)
    return out
}
```

### Path sum II — all root-to-leaf paths (Python)

```python
def path_sum(root, target):
    res = []
    def dfs(node, path, remaining):
        if not node: return
        path.append(node.val)
        if not node.left and not node.right and remaining == node.val:
            res.append(path[:])
        else:
            dfs(node.left, path, remaining - node.val)
            dfs(node.right, path, remaining - node.val)
        path.pop()
    dfs(root, [], target)
    return res
```

## DFS — Iterative

When recursion depth might exceed the stack (Python's default 1000) or you need explicit control over backtracking state.

### Iterative DFS with stack (Python)

```python
def dfs_iter(adj, start):
    seen = set([start])
    stack = [start]
    out = []
    while stack:
        u = stack.pop()
        out.append(u)
        for v in adj[u]:
            if v not in seen:
                seen.add(v)
                stack.append(v)
    return out
```

### Iterative inorder traversal (Go)

```go
func inorderIter(root *TreeNode) []int {
    out := []int{}
    stack := []*TreeNode{}
    cur := root
    for cur != nil || len(stack) > 0 {
        for cur != nil {
            stack = append(stack, cur)
            cur = cur.Left
        }
        cur = stack[len(stack)-1]
        stack = stack[:len(stack)-1]
        out = append(out, cur.Val)
        cur = cur.Right
    }
    return out
}
```

## DFS Applications

### Connected components (Python)

```python
def count_components(n, edges):
    adj = [[] for _ in range(n)]
    for a, b in edges:
        adj[a].append(b); adj[b].append(a)
    seen = [False] * n
    count = 0
    for i in range(n):
        if not seen[i]:
            count += 1
            stack = [i]
            seen[i] = True
            while stack:
                u = stack.pop()
                for v in adj[u]:
                    if not seen[v]:
                        seen[v] = True
                        stack.append(v)
    return count
```

### Cycle detection in directed graph — recursion stack (Go)

```go
func hasCycle(adj [][]int) bool {
    n := len(adj)
    visited := make([]int, n) // 0=unvisited, 1=in stack, 2=done
    var dfs func(u int) bool
    dfs = func(u int) bool {
        if visited[u] == 1 { return true }
        if visited[u] == 2 { return false }
        visited[u] = 1
        for _, v := range adj[u] {
            if dfs(v) { return true }
        }
        visited[u] = 2
        return false
    }
    for i := 0; i < n; i++ {
        if visited[i] == 0 && dfs(i) { return true }
    }
    return false
}
```

### Number of islands (Python)

```python
def num_islands(grid):
    if not grid: return 0
    rows, cols = len(grid), len(grid[0])
    def dfs(r, c):
        if r < 0 or r >= rows or c < 0 or c >= cols or grid[r][c] != '1':
            return
        grid[r][c] = '0'
        for dr, dc in [(1,0),(-1,0),(0,1),(0,-1)]:
            dfs(r + dr, c + dc)
    count = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                count += 1
                dfs(r, c)
    return count
```

## Multi-Source BFS

**Pattern:** initialize the BFS queue with *all* sources at distance 0. The wavefront expands from every source simultaneously; each cell receives the distance to its nearest source.

### Walls and gates (Python)

```python
def walls_and_gates(rooms):
    from collections import deque
    if not rooms: return
    rows, cols = len(rooms), len(rooms[0])
    INF = 2**31 - 1
    q = deque()
    for r in range(rows):
        for c in range(cols):
            if rooms[r][c] == 0:
                q.append((r, c))
    while q:
        r, c = q.popleft()
        for dr, dc in [(1,0),(-1,0),(0,1),(0,-1)]:
            nr, nc = r + dr, c + dc
            if 0 <= nr < rows and 0 <= nc < cols and rooms[nr][nc] == INF:
                rooms[nr][nc] = rooms[r][c] + 1
                q.append((nr, nc))
```

### Rotting oranges revisited as multi-source (Go)

Already covered above; the key insight is the queue starts with every initially-rotten orange.

## 0-1 BFS

**Pattern:** when edge weights are only 0 and 1, you can find shortest paths in O(V + E) using a deque instead of a priority queue. Push 0-weight neighbors to the *front*, 1-weight neighbors to the *back*.

### Minimum cost path in 0/1 weighted grid (Python)

```python
from collections import deque

def min_cost_grid(grid):
    rows, cols = len(grid), len(grid[0])
    dist = [[float('inf')] * cols for _ in range(rows)]
    dist[0][0] = 0
    dq = deque([(0, 0)])
    while dq:
        r, c = dq.popleft()
        for dr, dc, w in [(1,0,0),(-1,0,1),(0,1,0),(0,-1,1)]:
            nr, nc = r + dr, c + dc
            if 0 <= nr < rows and 0 <= nc < cols:
                nd = dist[r][c] + w
                if nd < dist[nr][nc]:
                    dist[nr][nc] = nd
                    if w == 0:
                        dq.appendleft((nr, nc))
                    else:
                        dq.append((nr, nc))
    return dist[rows - 1][cols - 1]
```

## Topological Sort

**Pattern:** linearize a DAG so every edge goes from earlier to later in the order. Two implementations:

- **Kahn's BFS.** Repeatedly remove a 0-in-degree node and decrement its neighbors. Detects cycles by leftover nodes.
- **DFS post-order.** Reverse post-order traversal of the DAG.

### Kahn's algorithm (Python)

```python
from collections import deque

def topo_sort(n, edges):
    adj = [[] for _ in range(n)]
    indeg = [0] * n
    for a, b in edges:
        adj[a].append(b); indeg[b] += 1
    q = deque(i for i in range(n) if indeg[i] == 0)
    out = []
    while q:
        u = q.popleft()
        out.append(u)
        for v in adj[u]:
            indeg[v] -= 1
            if indeg[v] == 0:
                q.append(v)
    return out if len(out) == n else []  # empty = cycle
```

### Course schedule II (Go)

```go
func findOrder(n int, prereq [][]int) []int {
    adj := make([][]int, n)
    indeg := make([]int, n)
    for _, p := range prereq {
        adj[p[1]] = append(adj[p[1]], p[0])
        indeg[p[0]]++
    }
    q := []int{}
    for i := 0; i < n; i++ {
        if indeg[i] == 0 { q = append(q, i) }
    }
    out := []int{}
    for len(q) > 0 {
        u := q[0]; q = q[1:]
        out = append(out, u)
        for _, v := range adj[u] {
            indeg[v]--
            if indeg[v] == 0 { q = append(q, v) }
        }
    }
    if len(out) != n { return []int{} }
    return out
}
```

### DFS-based topological sort (Python)

```python
def topo_dfs(n, edges):
    adj = [[] for _ in range(n)]
    for a, b in edges:
        adj[a].append(b)
    color = [0] * n  # 0 white, 1 gray, 2 black
    out = []
    has_cycle = False
    def dfs(u):
        nonlocal has_cycle
        if color[u] == 1:
            has_cycle = True; return
        if color[u] == 2: return
        color[u] = 1
        for v in adj[u]:
            dfs(v)
        color[u] = 2
        out.append(u)
    for i in range(n):
        if color[i] == 0:
            dfs(i)
    return [] if has_cycle else list(reversed(out))
```

## Bidirectional BFS

**Pattern:** for shortest path between a single source and target on a large graph, expand frontiers from *both* ends and stop when they meet. The branching factor `b` raised to depth `d/2` is much smaller than `b^d`.

### Word ladder optimized (Python)

```python
def ladder_length(begin, end, word_list):
    word_set = set(word_list)
    if end not in word_set:
        return 0
    front, back = {begin}, {end}
    depth = 1
    while front and back:
        if len(front) > len(back):
            front, back = back, front
        next_front = set()
        for w in front:
            for i in range(len(w)):
                for c in 'abcdefghijklmnopqrstuvwxyz':
                    if c == w[i]:
                        continue
                    nw = w[:i] + c + w[i+1:]
                    if nw in back:
                        return depth + 1
                    if nw in word_set:
                        next_front.add(nw)
                        word_set.remove(nw)
        front = next_front
        depth += 1
    return 0
```

## Dijkstra

**Pattern:** shortest path from a single source on a graph with **non-negative** edge weights. Uses a min-heap to always extract the closest unvisited vertex.

### Single-source shortest path (Python)

```python
import heapq

def dijkstra(adj, src):
    n = len(adj)
    dist = [float('inf')] * n
    dist[src] = 0
    heap = [(0, src)]
    while heap:
        d, u = heapq.heappop(heap)
        if d > dist[u]:
            continue
        for v, w in adj[u]:
            nd = d + w
            if nd < dist[v]:
                dist[v] = nd
                heapq.heappush(heap, (nd, v))
    return dist
```

### Network delay time (Go)

```go
import "container/heap"

type pqItem struct{ node, dist int }
type pq []pqItem
func (p pq) Len() int            { return len(p) }
func (p pq) Less(i, j int) bool  { return p[i].dist < p[j].dist }
func (p pq) Swap(i, j int)       { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x any)         { *p = append(*p, x.(pqItem)) }
func (p *pq) Pop() any           { old := *p; x := old[len(old)-1]; *p = old[:len(old)-1]; return x }

func networkDelayTime(times [][]int, n, k int) int {
    adj := make([][][2]int, n+1)
    for _, t := range times {
        adj[t[0]] = append(adj[t[0]], [2]int{t[1], t[2]})
    }
    dist := make([]int, n+1)
    for i := range dist { dist[i] = 1 << 30 }
    dist[k] = 0
    h := &pq{{k, 0}}
    heap.Init(h)
    for h.Len() > 0 {
        cur := heap.Pop(h).(pqItem)
        if cur.dist > dist[cur.node] { continue }
        for _, e := range adj[cur.node] {
            nd := cur.dist + e[1]
            if nd < dist[e[0]] {
                dist[e[0]] = nd
                heap.Push(h, pqItem{e[0], nd})
            }
        }
    }
    best := 0
    for i := 1; i <= n; i++ {
        if dist[i] > best { best = dist[i] }
    }
    if best == 1<<30 { return -1 }
    return best
}
```

### Cheapest flights with K stops (Python)

```python
import heapq

def find_cheapest(n, flights, src, dst, k):
    adj = [[] for _ in range(n)]
    for u, v, w in flights:
        adj[u].append((v, w))
    heap = [(0, src, 0)]  # (cost, node, stops)
    while heap:
        cost, u, stops = heapq.heappop(heap)
        if u == dst:
            return cost
        if stops > k:
            continue
        for v, w in adj[u]:
            heapq.heappush(heap, (cost + w, v, stops + 1))
    return -1
```

## Dijkstra — Variants

### K-th shortest path

Maintain a count array of how many times each node has been popped. Stop expanding from a node after it has been popped K times.

```python
import heapq

def kth_shortest(adj, src, dst, k):
    counts = [0] * len(adj)
    heap = [(0, src)]
    while heap:
        d, u = heapq.heappop(heap)
        counts[u] += 1
        if u == dst and counts[u] == k:
            return d
        if counts[u] > k:
            continue
        for v, w in adj[u]:
            heapq.heappush(heap, (d + w, v))
    return -1
```

### Dial's algorithm

When edge weights are bounded small integers, replace the heap with an array of buckets indexed by current distance. Each push is O(1); the algorithm runs in O(V + E + max_dist).

## Bellman-Ford

**Pattern:** shortest path with negative edge weights; can detect negative cycles. Relax every edge `V - 1` times.

### Bellman-Ford (Python)

```python
def bellman_ford(n, edges, src):
    INF = float('inf')
    dist = [INF] * n
    dist[src] = 0
    for _ in range(n - 1):
        updated = False
        for u, v, w in edges:
            if dist[u] != INF and dist[u] + w < dist[v]:
                dist[v] = dist[u] + w
                updated = True
        if not updated:
            break
    # Negative cycle detection
    for u, v, w in edges:
        if dist[u] != INF and dist[u] + w < dist[v]:
            return None  # negative cycle reachable
    return dist
```

### Cheapest flights within K stops via Bellman-Ford (Go)

```go
func findCheapestPriceBF(n int, flights [][]int, src, dst, k int) int {
    INF := 1 << 30
    dist := make([]int, n)
    for i := range dist { dist[i] = INF }
    dist[src] = 0
    for i := 0; i <= k; i++ {
        next := append([]int{}, dist...)
        for _, f := range flights {
            if dist[f[0]] == INF { continue }
            if dist[f[0]]+f[2] < next[f[1]] {
                next[f[1]] = dist[f[0]] + f[2]
            }
        }
        dist = next
    }
    if dist[dst] == INF { return -1 }
    return dist[dst]
}
```

## Floyd-Warshall

**Pattern:** all-pairs shortest paths in O(V³); handles negative edges (no negative cycles). Uses a triply-nested loop.

### Floyd-Warshall (Python)

```python
def floyd_warshall(n, edges):
    INF = float('inf')
    dist = [[INF] * n for _ in range(n)]
    for i in range(n):
        dist[i][i] = 0
    for u, v, w in edges:
        dist[u][v] = min(dist[u][v], w)
    for k in range(n):
        for i in range(n):
            for j in range(n):
                if dist[i][k] + dist[k][j] < dist[i][j]:
                    dist[i][j] = dist[i][k] + dist[k][j]
    return dist
```

### Transitive closure (Go)

```go
func transitiveClosure(n int, edges [][]int) [][]bool {
    reach := make([][]bool, n)
    for i := range reach { reach[i] = make([]bool, n); reach[i][i] = true }
    for _, e := range edges { reach[e[0]][e[1]] = true }
    for k := 0; k < n; k++ {
        for i := 0; i < n; i++ {
            for j := 0; j < n; j++ {
                if reach[i][k] && reach[k][j] { reach[i][j] = true }
            }
        }
    }
    return reach
}
```

## A*

**Pattern:** Dijkstra augmented with a heuristic `h(n)` that estimates the remaining cost to the goal. Priority is `g(n) + h(n)`.

- **Admissible:** `h(n) ≤ true distance to goal` — guarantees an optimal solution.
- **Consistent (monotone):** `h(n) ≤ cost(n, n') + h(n')` — guarantees each node is expanded at most once.

### 8-puzzle / sliding puzzle (Python sketch)

```python
import heapq

def a_star_grid(grid, start, goal, neighbors, h):
    open_set = [(h(start), 0, start)]
    came_from = {start: None}
    g = {start: 0}
    while open_set:
        _, cur_g, u = heapq.heappop(open_set)
        if u == goal:
            path = []
            while u is not None:
                path.append(u)
                u = came_from[u]
            return list(reversed(path))
        for v, cost in neighbors(u):
            ng = cur_g + cost
            if ng < g.get(v, float('inf')):
                g[v] = ng
                came_from[v] = u
                heapq.heappush(open_set, (ng + h(v), ng, v))
    return None
```

### Manhattan distance heuristic for 8-puzzle (Python)

```python
def manhattan(state, goal):
    total = 0
    for i, v in enumerate(state):
        if v == 0:
            continue
        gi = goal.index(v)
        total += abs(i // 3 - gi // 3) + abs(i % 3 - gi % 3)
    return total
```

## MST — Kruskal

**Pattern:** sort all edges by weight; include each edge if its endpoints are in different components (use union-find).

### Kruskal's algorithm (Python)

```python
class DSU:
    def __init__(self, n):
        self.p = list(range(n))
        self.r = [0] * n
    def find(self, x):
        while self.p[x] != x:
            self.p[x] = self.p[self.p[x]]
            x = self.p[x]
        return x
    def union(self, a, b):
        ra, rb = self.find(a), self.find(b)
        if ra == rb: return False
        if self.r[ra] < self.r[rb]: ra, rb = rb, ra
        self.p[rb] = ra
        if self.r[ra] == self.r[rb]: self.r[ra] += 1
        return True

def kruskal(n, edges):
    edges.sort(key=lambda e: e[2])
    dsu = DSU(n)
    total = 0
    used = []
    for u, v, w in edges:
        if dsu.union(u, v):
            total += w
            used.append((u, v, w))
            if len(used) == n - 1:
                break
    return total, used
```

## MST — Prim

**Pattern:** start from any vertex; greedily add the lightest edge crossing the visited/unvisited cut. Implemented with a min-heap, structurally similar to Dijkstra.

### Prim's algorithm (Go)

```go
import "container/heap"

func primMST(n int, edges [][]int) int {
    adj := make([][][2]int, n)
    for _, e := range edges {
        adj[e[0]] = append(adj[e[0]], [2]int{e[1], e[2]})
        adj[e[1]] = append(adj[e[1]], [2]int{e[0], e[2]})
    }
    visited := make([]bool, n)
    h := &pq{{0, 0}}
    heap.Init(h)
    total, count := 0, 0
    for h.Len() > 0 && count < n {
        cur := heap.Pop(h).(pqItem)
        if visited[cur.node] { continue }
        visited[cur.node] = true
        total += cur.dist
        count++
        for _, e := range adj[cur.node] {
            if !visited[e[0]] {
                heap.Push(h, pqItem{e[0], e[1]})
            }
        }
    }
    if count < n { return -1 }
    return total
}
```

## Union-Find

**Pattern:** maintain a forest where each tree's root represents a connected component. Two operations:

- **find(x)** — return the root, with path compression.
- **union(a, b)** — merge components, with union by rank.

Amortized α(n) per operation — effectively constant.

### Union-find template (Python)

```python
class DSU:
    def __init__(self, n):
        self.parent = list(range(n))
        self.rank = [0] * n
        self.size = [1] * n
        self.components = n
    def find(self, x):
        root = x
        while self.parent[root] != root:
            root = self.parent[root]
        while self.parent[x] != root:
            self.parent[x], x = root, self.parent[x]
        return root
    def union(self, a, b):
        ra, rb = self.find(a), self.find(b)
        if ra == rb: return False
        if self.rank[ra] < self.rank[rb]:
            ra, rb = rb, ra
        self.parent[rb] = ra
        self.size[ra] += self.size[rb]
        if self.rank[ra] == self.rank[rb]:
            self.rank[ra] += 1
        self.components -= 1
        return True
```

### Connected components (Go)

```go
type DSU struct {
    parent, rank []int
}

func newDSU(n int) *DSU {
    parent := make([]int, n)
    for i := range parent { parent[i] = i }
    return &DSU{parent: parent, rank: make([]int, n)}
}

func (d *DSU) find(x int) int {
    for d.parent[x] != x {
        d.parent[x] = d.parent[d.parent[x]]
        x = d.parent[x]
    }
    return x
}

func (d *DSU) union(a, b int) bool {
    ra, rb := d.find(a), d.find(b)
    if ra == rb { return false }
    if d.rank[ra] < d.rank[rb] { ra, rb = rb, ra }
    d.parent[rb] = ra
    if d.rank[ra] == d.rank[rb] { d.rank[ra]++ }
    return true
}
```

### Redundant connection (Python)

```python
def find_redundant(edges):
    dsu = DSU(len(edges) + 1)
    for u, v in edges:
        if not dsu.union(u, v):
            return [u, v]
    return []
```

### Accounts merge (Python)

```python
def accounts_merge(accounts):
    email_to_idx = {}
    for i, acct in enumerate(accounts):
        for e in acct[1:]:
            email_to_idx.setdefault(e, []).append(i)
    dsu = DSU(len(accounts))
    for idxs in email_to_idx.values():
        for j in range(1, len(idxs)):
            dsu.union(idxs[0], idxs[j])
    groups = {}
    for e, idxs in email_to_idx.items():
        root = dsu.find(idxs[0])
        groups.setdefault(root, set()).add(e)
    out = []
    for root, emails in groups.items():
        out.append([accounts[root][0]] + sorted(emails))
    return out
```

## Trie

**Pattern:** prefix tree storing strings. Each node has a children map and an end-of-word flag. Operations are O(L) where L is string length, independent of how many strings are stored.

### Trie template (Python)

```python
class Trie:
    def __init__(self):
        self.children = {}
        self.is_end = False
    def insert(self, word):
        node = self
        for ch in word:
            node = node.children.setdefault(ch, Trie())
        node.is_end = True
    def search(self, word):
        node = self
        for ch in word:
            if ch not in node.children:
                return False
            node = node.children[ch]
        return node.is_end
    def starts_with(self, prefix):
        node = self
        for ch in prefix:
            if ch not in node.children:
                return False
            node = node.children[ch]
        return True
```

### Trie in Go

```go
type Trie struct {
    children [26]*Trie
    isEnd    bool
}

func (t *Trie) Insert(word string) {
    node := t
    for i := 0; i < len(word); i++ {
        c := word[i] - 'a'
        if node.children[c] == nil {
            node.children[c] = &Trie{}
        }
        node = node.children[c]
    }
    node.isEnd = true
}

func (t *Trie) Search(word string) bool {
    node := t
    for i := 0; i < len(word); i++ {
        c := word[i] - 'a'
        if node.children[c] == nil { return false }
        node = node.children[c]
    }
    return node.isEnd
}
```

### Word search II — DFS over grid using a trie (Python)

```python
def find_words(board, words):
    trie = Trie()
    for w in words:
        trie.insert(w)
    rows, cols = len(board), len(board[0])
    res = set()
    def dfs(r, c, node, path):
        if node.is_end:
            res.add(path)
        if r < 0 or r >= rows or c < 0 or c >= cols:
            return
        ch = board[r][c]
        if ch == '#' or ch not in node.children:
            return
        board[r][c] = '#'
        for dr, dc in [(1,0),(-1,0),(0,1),(0,-1)]:
            dfs(r + dr, c + dc, node.children[ch], path + ch)
        board[r][c] = ch
    for r in range(rows):
        for c in range(cols):
            dfs(r, c, trie, "")
    return list(res)
```

### Longest common prefix of an array of strings (Go)

```go
func longestCommonPrefix(strs []string) string {
    if len(strs) == 0 { return "" }
    prefix := strs[0]
    for i := 1; i < len(strs); i++ {
        for len(prefix) > 0 && (len(strs[i]) < len(prefix) || strs[i][:len(prefix)] != prefix) {
            prefix = prefix[:len(prefix)-1]
        }
    }
    return prefix
}
```

## KMP

**Pattern:** O(n + m) substring matching by precomputing the *failure function* (also called LPS — longest proper prefix that is also a suffix).

### KMP failure function (Python)

```python
def kmp_failure(p):
    fail = [0] * len(p)
    k = 0
    for i in range(1, len(p)):
        while k > 0 and p[k] != p[i]:
            k = fail[k - 1]
        if p[k] == p[i]:
            k += 1
        fail[i] = k
    return fail

def kmp_search(text, pattern):
    if not pattern: return 0
    fail = kmp_failure(pattern)
    j = 0
    for i in range(len(text)):
        while j > 0 and pattern[j] != text[i]:
            j = fail[j - 1]
        if pattern[j] == text[i]:
            j += 1
        if j == len(pattern):
            return i - j + 1
    return -1
```

### KMP in Go

```go
func kmpSearch(text, pat string) int {
    if len(pat) == 0 { return 0 }
    fail := make([]int, len(pat))
    k := 0
    for i := 1; i < len(pat); i++ {
        for k > 0 && pat[k] != pat[i] { k = fail[k-1] }
        if pat[k] == pat[i] { k++ }
        fail[i] = k
    }
    j := 0
    for i := 0; i < len(text); i++ {
        for j > 0 && pat[j] != text[i] { j = fail[j-1] }
        if pat[j] == text[i] { j++ }
        if j == len(pat) { return i - j + 1 }
    }
    return -1
}
```

## Z-Algorithm

**Pattern:** alternative to KMP. The Z-array `Z[i]` is the length of the longest substring starting at `i` that matches a prefix of the string.

### Z-array construction (Python)

```python
def z_array(s):
    n = len(s)
    z = [0] * n
    z[0] = n
    l = r = 0
    for i in range(1, n):
        if i < r:
            z[i] = min(r - i, z[i - l])
        while i + z[i] < n and s[z[i]] == s[i + z[i]]:
            z[i] += 1
        if i + z[i] > r:
            l, r = i, i + z[i]
    return z

def z_search(text, pattern):
    s = pattern + "$" + text
    z = z_array(s)
    matches = []
    for i in range(len(pattern) + 1, len(s)):
        if z[i] == len(pattern):
            matches.append(i - len(pattern) - 1)
    return matches
```

### Z-array in Go

```go
func zArray(s string) []int {
    n := len(s)
    z := make([]int, n)
    z[0] = n
    l, r := 0, 0
    for i := 1; i < n; i++ {
        if i < r {
            z[i] = min2(r-i, z[i-l])
        }
        for i+z[i] < n && s[z[i]] == s[i+z[i]] {
            z[i]++
        }
        if i+z[i] > r {
            l, r = i, i+z[i]
        }
    }
    return z
}
```

## Aho-Corasick

**Pattern:** match a *set* of patterns simultaneously in O(n + sum of pattern lengths + matches). Builds a trie augmented with failure links and output links — like KMP for many patterns.

### Aho-Corasick (Python sketch)

```python
from collections import deque, defaultdict

class AhoCorasick:
    def __init__(self):
        self.goto = [defaultdict(int)]
        self.out = [[]]
        self.fail = [0]

    def add(self, word):
        cur = 0
        for ch in word:
            if ch not in self.goto[cur]:
                self.goto.append(defaultdict(int))
                self.out.append([])
                self.fail.append(0)
                self.goto[cur][ch] = len(self.goto) - 1
            cur = self.goto[cur][ch]
        self.out[cur].append(word)

    def build(self):
        q = deque()
        for ch, nxt in self.goto[0].items():
            self.fail[nxt] = 0
            q.append(nxt)
        while q:
            u = q.popleft()
            for ch, v in self.goto[u].items():
                f = self.fail[u]
                while f and ch not in self.goto[f]:
                    f = self.fail[f]
                self.fail[v] = self.goto[f].get(ch, 0) if f != v else 0
                self.out[v].extend(self.out[self.fail[v]])
                q.append(v)

    def search(self, text):
        cur = 0
        matches = []
        for i, ch in enumerate(text):
            while cur and ch not in self.goto[cur]:
                cur = self.fail[cur]
            cur = self.goto[cur].get(ch, 0)
            for w in self.out[cur]:
                matches.append((i - len(w) + 1, w))
        return matches
```

Used for spam filters, intrusion detection, and DPI engines that scan for thousands of signatures simultaneously.

## Suffix Array / Suffix Tree

**Pattern:** all suffixes of a string sorted lexicographically. Combined with the LCP (longest common prefix) array, supports many substring queries in linear time after construction.

### Suffix array via sort (Python, O(n² log n))

```python
def suffix_array(s):
    return sorted(range(len(s)), key=lambda i: s[i:])
```

### LCP array using Kasai's algorithm (Python)

```python
def kasai(s, sa):
    n = len(s)
    rank = [0] * n
    for i in range(n):
        rank[sa[i]] = i
    h = 0
    lcp = [0] * (n - 1)
    for i in range(n):
        if rank[i] > 0:
            j = sa[rank[i] - 1]
            while i + h < n and j + h < n and s[i + h] == s[j + h]:
                h += 1
            lcp[rank[i] - 1] = h
            if h > 0:
                h -= 1
    return lcp
```

### Longest repeated substring

```python
def longest_repeated(s):
    sa = suffix_array(s)
    lcp = kasai(s, sa)
    if not lcp:
        return ""
    idx = max(range(len(lcp)), key=lambda i: lcp[i])
    length = lcp[idx]
    start = sa[idx]
    return s[start:start + length]
```

## Sweep Line

**Pattern:** convert geometric or interval problems into a sequence of *events* sorted by some axis (usually x). Process events in order, maintaining a data structure of currently-active items.

### Skyline problem (Python)

```python
import heapq

def get_skyline(buildings):
    events = []
    for L, R, H in buildings:
        events.append((L, -H, R))
        events.append((R, 0, 0))
    events.sort()
    res = []
    heap = [(0, float('inf'))]
    for x, negH, R in events:
        while heap[0][1] <= x:
            heapq.heappop(heap)
        if negH:
            heapq.heappush(heap, (negH, R))
        height = -heap[0][0]
        if not res or res[-1][1] != height:
            res.append([x, height])
    return res
```

### Meeting rooms II via sweep (Go)

```go
import "sort"

func minMeetingRoomsSweep(intervals [][]int) int {
    starts := make([]int, len(intervals))
    ends := make([]int, len(intervals))
    for i, it := range intervals {
        starts[i] = it[0]; ends[i] = it[1]
    }
    sort.Ints(starts); sort.Ints(ends)
    rooms, j := 0, 0
    for i := 0; i < len(starts); i++ {
        if starts[i] < ends[j] {
            rooms++
        } else {
            j++
        }
    }
    return rooms
}
```

### Rectangle area union (sketch)

Sweep on x; for each event, maintain a set of active y-intervals; sum the covered y-length × Δx between consecutive events. Often implemented with a segment tree counting covered length.

## Coordinate Compression

**Pattern:** when values are sparse but their *ranks* matter, replace each value with its index in the sorted unique list. Now you can use array-indexed structures (BIT, segment tree) without huge memory.

### Coordinate compression (Python)

```python
def compress(arr):
    sorted_unique = sorted(set(arr))
    rank = {v: i for i, v in enumerate(sorted_unique)}
    return [rank[v] for v in arr]
```

### Count smaller after self via BIT (Python)

```python
def count_smaller(nums):
    sorted_unique = sorted(set(nums))
    rank = {v: i + 1 for i, v in enumerate(sorted_unique)}
    bit = [0] * (len(rank) + 2)
    def update(i):
        while i < len(bit):
            bit[i] += 1; i += i & -i
    def query(i):
        s = 0
        while i > 0:
            s += bit[i]; i -= i & -i
        return s
    res = [0] * len(nums)
    for i in range(len(nums) - 1, -1, -1):
        r = rank[nums[i]]
        res[i] = query(r - 1)
        update(r)
    return res
```

### Reverse pairs (Go) — count `i < j` with `nums[i] > 2 * nums[j]`

Use modified merge sort with counting; coordinate compression simplifies range counting if you switch to BIT.

## Reservoir Sampling

**Pattern:** sample `k` items uniformly at random from a stream of *unknown* length using O(k) memory. The classic application is choosing one random line from a huge file in a single pass.

### Sample one (Python)

```python
import random

def reservoir_one(stream):
    chosen = None
    for i, x in enumerate(stream, 1):
        if random.randint(1, i) == 1:
            chosen = x
    return chosen
```

### Sample k (Algorithm R, Go)

```go
import "math/rand"

func reservoirK(stream []int, k int) []int {
    if len(stream) <= k {
        return append([]int{}, stream...)
    }
    sample := make([]int, k)
    copy(sample, stream[:k])
    for i := k; i < len(stream); i++ {
        j := rand.Intn(i + 1)
        if j < k {
            sample[j] = stream[i]
        }
    }
    return sample
}
```

**Probability proof.** After observing item `i ≥ k`, the probability of choosing it for slot `j` is `k / i`. By induction, every previously-chosen item has probability `k / i` of remaining in the reservoir. So all items in the stream have equal probability `k / N` after the stream ends.

## Quickselect

Already covered in detail above. Worth restating: quickselect finds the kth smallest in O(n) average time using partition. Used as the kernel of the *median-of-medians* algorithm for guaranteed O(n) worst case.

## Cycle in Linked List / Array

Floyd's tortoise + hare detects cycles in any "follow the next pointer" structure. The *array variant* treats `arr[i]` as a function `f(i) → arr[i]` and finds the cycle in the iteration sequence.

### Find duplicate number (Floyd on indices, Python)

```python
def find_duplicate(nums):
    slow = fast = nums[0]
    while True:
        slow = nums[slow]
        fast = nums[nums[fast]]
        if slow == fast:
            break
    slow = nums[0]
    while slow != fast:
        slow = nums[slow]
        fast = nums[fast]
    return slow

assert find_duplicate([1, 3, 4, 2, 2]) == 2
```

The trick: with values in `[1, n]` and `n + 1` slots, pigeonhole guarantees a duplicate. Treating values as next-pointers means the duplicate is the cycle entrance.

## Pattern: Top-K Elements

**Trigger:** find the `k` largest or smallest, or `k` most frequent. Maintain a heap of size `k`.

- For **k largest**, use a *min-heap*: push everything, pop when size > k, the heap then holds the k largest.
- For **k smallest**, use a *max-heap*.

### Top-k frequent words (Python)

```python
import heapq
from collections import Counter

def top_k_frequent(words, k):
    cnt = Counter(words)
    return heapq.nsmallest(k, cnt.keys(), key=lambda w: (-cnt[w], w))
```

### Kth largest in stream (Go)

```go
import "container/heap"

type IntMinHeap []int
func (h IntMinHeap) Len() int            { return len(h) }
func (h IntMinHeap) Less(i, j int) bool  { return h[i] < h[j] }
func (h IntMinHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *IntMinHeap) Push(x any)         { *h = append(*h, x.(int)) }
func (h *IntMinHeap) Pop() any           { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

type KthLargest struct { k int; h *IntMinHeap }

func Constructor(k int, nums []int) KthLargest {
    h := &IntMinHeap{}
    heap.Init(h)
    kl := KthLargest{k: k, h: h}
    for _, x := range nums { kl.Add(x) }
    return kl
}

func (kl *KthLargest) Add(val int) int {
    heap.Push(kl.h, val)
    if kl.h.Len() > kl.k { heap.Pop(kl.h) }
    return (*kl.h)[0]
}
```

## Pattern: K Closest

Same heap pattern, distance metric instead of value.

### K closest points to origin (Python)

```python
import heapq

def k_closest(points, k):
    heap = []
    for x, y in points:
        d = x * x + y * y
        if len(heap) < k:
            heapq.heappush(heap, (-d, x, y))
        elif -heap[0][0] > d:
            heapq.heapreplace(heap, (-d, x, y))
    return [[x, y] for _, x, y in heap]
```

## Pattern: Median from Stream

**Pattern:** maintain two heaps:

- **lower** — max-heap holding the smaller half.
- **upper** — min-heap holding the larger half.

Invariants: `len(lower) == len(upper)` or `len(lower) == len(upper) + 1`. Median is `lower[0]` (if odd) or `(lower[0] + upper[0]) / 2` (if even).

### Median finder (Python)

```python
import heapq

class MedianFinder:
    def __init__(self):
        self.lower = []  # max-heap (negated)
        self.upper = []  # min-heap

    def add(self, num):
        heapq.heappush(self.lower, -num)
        heapq.heappush(self.upper, -heapq.heappop(self.lower))
        if len(self.upper) > len(self.lower):
            heapq.heappush(self.lower, -heapq.heappop(self.upper))

    def median(self):
        if len(self.lower) > len(self.upper):
            return -self.lower[0]
        return (-self.lower[0] + self.upper[0]) / 2
```

### Median finder (Go)

```go
import "container/heap"

type MaxHeap []int
func (h MaxHeap) Len() int           { return len(h) }
func (h MaxHeap) Less(i, j int) bool { return h[i] > h[j] }
func (h MaxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *MaxHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *MaxHeap) Pop() any { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

type MedianFinder struct {
    lo *MaxHeap
    hi *IntMinHeap
}

func NewMedianFinder() *MedianFinder {
    return &MedianFinder{lo: &MaxHeap{}, hi: &IntMinHeap{}}
}

func (m *MedianFinder) AddNum(n int) {
    heap.Push(m.lo, n)
    heap.Push(m.hi, heap.Pop(m.lo))
    if m.hi.Len() > m.lo.Len() {
        heap.Push(m.lo, heap.Pop(m.hi))
    }
}

func (m *MedianFinder) FindMedian() float64 {
    if m.lo.Len() > m.hi.Len() {
        return float64((*m.lo)[0])
    }
    return float64((*m.lo)[0]+(*m.hi)[0]) / 2.0
}
```

## Pattern: Two Heaps

The two-heap pattern generalizes beyond medians. Any time you need to maintain two halves with a balance invariant — sliding-window median, IPO problem (capital-K projects), task scheduling with priorities — reach for two heaps.

### Sliding window median sketch

Maintain two heaps mapped to a hashmap of "lazily deleted" indices. Push a new element; rebalance; mark the leaving element for deletion; pop lazily when it surfaces.

## Pattern: Bit Manipulation

### XOR for missing / duplicate

```python
def find_single(nums):
    result = 0
    for x in nums:
        result ^= x
    return result
```

XOR is its own inverse, so doubled elements cancel; the lone element survives.

### Missing number (Python)

```python
def missing_number(nums):
    n = len(nums)
    expected = n * (n + 1) // 2
    return expected - sum(nums)
    # Or via XOR:
    # x = 0
    # for i in range(n + 1): x ^= i
    # for v in nums: x ^= v
    # return x
```

### Counting bits up to N (Go, Brian Kernighan trick)

```go
func countBits(n int) []int {
    out := make([]int, n+1)
    for i := 1; i <= n; i++ {
        out[i] = out[i & (i - 1)] + 1
    }
    return out
}
```

### Sum of two integers without using `+` (Python)

```python
def get_sum(a, b):
    mask = 0xFFFFFFFF
    while b & mask:
        carry = (a & b) << 1
        a = a ^ b
        b = carry
    return a & mask if a <= 0x7FFFFFFF else ~(a ^ mask)
```

### Subset enumeration via bitmask (Python)

```python
def subsets_bitmask(arr):
    n = len(arr)
    out = []
    for mask in range(1 << n):
        sub = []
        for i in range(n):
            if mask & (1 << i):
                sub.append(arr[i])
        out.append(sub)
    return out
```

### Iterate over submasks of a mask (Go)

```go
func iterateSubmasks(mask int, fn func(int)) {
    sub := mask
    for sub > 0 {
        fn(sub)
        sub = (sub - 1) & mask
    }
    fn(0)
}
```

## Pattern: Mathematical

### Modular arithmetic — fast exponentiation (Python)

```python
def pow_mod(base, exp, mod):
    result = 1
    base %= mod
    while exp:
        if exp & 1:
            result = result * base % mod
        base = base * base % mod
        exp >>= 1
    return result
```

### Combinatorics — Pascal's triangle (Go)

```go
func pascalsTriangle(n int) [][]int {
    out := make([][]int, n)
    for i := 0; i < n; i++ {
        out[i] = make([]int, i+1)
        out[i][0], out[i][i] = 1, 1
        for j := 1; j < i; j++ {
            out[i][j] = out[i-1][j-1] + out[i-1][j]
        }
    }
    return out
}
```

### Matrix exponentiation for nth Fibonacci (Python)

```python
def matmul(A, B, mod=None):
    n = len(A)
    C = [[0] * n for _ in range(n)]
    for i in range(n):
        for k in range(n):
            for j in range(n):
                C[i][j] += A[i][k] * B[k][j]
                if mod: C[i][j] %= mod
    return C

def matpow(M, p, mod=None):
    n = len(M)
    result = [[1 if i == j else 0 for j in range(n)] for i in range(n)]
    base = M
    while p:
        if p & 1:
            result = matmul(result, base, mod)
        base = matmul(base, base, mod)
        p >>= 1
    return result

def fib(n):
    if n == 0: return 0
    M = matpow([[1, 1], [1, 0]], n)
    return M[0][1]
```

### GCD and LCM (Go)

```go
func gcd(a, b int) int {
    for b != 0 { a, b = b, a%b }
    return a
}
func lcm(a, b int) int {
    return a / gcd(a, b) * b
}
```

### Sieve of Eratosthenes (Python)

```python
def sieve(n):
    is_prime = [True] * (n + 1)
    is_prime[0] = is_prime[1] = False
    for i in range(2, int(n ** 0.5) + 1):
        if is_prime[i]:
            for j in range(i * i, n + 1, i):
                is_prime[j] = False
    return [i for i in range(n + 1) if is_prime[i]]
```

## Pattern: Streaming Sketches

Beyond reservoir sampling, real-world streams use probabilistic data structures.

### Count-Min Sketch

A 2D array `cm[d][w]` of counters. `add(x)`: for each row `i`, hash `x` to a column and increment `cm[i][col]`. `query(x)`: minimum over all rows.

- Overestimates; never underestimates.
- Space O(d · w); error bounded by `2 / w` of total stream length with probability 1 - 2^-d.

### HyperLogLog

Estimates the *number of distinct elements* in a stream using O(log log n) bits per register. Used in Redis (`PFADD`, `PFCOUNT`), in Google's BigQuery, and in any system needing cardinality estimates without storing the full set.

### Bloom filter (Python)

```python
import hashlib

class BloomFilter:
    def __init__(self, size, hashes):
        self.size = size
        self.hashes = hashes
        self.bits = [False] * size
    def _idx(self, item, seed):
        h = hashlib.md5(f"{seed}-{item}".encode()).hexdigest()
        return int(h, 16) % self.size
    def add(self, item):
        for i in range(self.hashes):
            self.bits[self._idx(item, i)] = True
    def contains(self, item):
        return all(self.bits[self._idx(item, i)] for i in range(self.hashes))
```

False positives possible; false negatives never. Ideal for "is this URL probably in the blacklist?".

## Decision Tree — When do I reach for X?

This is the final lookup. Keep it open while you read a problem statement.

```
Sorted array
├─ find element / boundary               → binary search lower/upper bound
├─ pair sums to target                   → two pointers (opposite)
├─ remove duplicates                     → two pointers (same)
├─ overlapping ranges                    → merge intervals
└─ rotation / peak                       → binary search variants

Unsorted array
├─ subarray contiguous, fixed window     → sliding window (fixed)
├─ subarray contiguous, variable window  → sliding window (variable)
├─ subarray sum equals K                 → prefix sum + hashmap
├─ next greater / smaller                → monotonic stack
├─ kth largest / smallest                → heap or quickselect
└─ count smaller after self              → BIT or merge sort

Linked list
├─ cycle detection                       → fast/slow pointer
├─ middle / nth-from-end                 → fast/slow pointer
└─ reverse                               → iterative pointer flip

Range queries on static data             → prefix sum, segment tree, sparse table
Range updates                            → difference array, segment tree (lazy)
Recurring overlapping subproblems        → DP
Decision among many independent choices  → backtracking
Decision among many overlapping choices  → DP

Tree
├─ traverse                              → DFS recursive
├─ level by level                        → BFS queue
├─ best subtree value                    → tree DP (post-order)
└─ LCA / ancestor                        → binary lifting / Tarjan

Graph
├─ shortest path, unweighted             → BFS
├─ shortest path, non-negative weights   → Dijkstra
├─ shortest path, negative weights       → Bellman-Ford
├─ all-pairs shortest path               → Floyd-Warshall
├─ MST                                   → Kruskal or Prim
├─ topological order / cycle detect      → Kahn's BFS or DFS
└─ connected components                  → DFS or union-find

String
├─ exact substring                       → KMP or Z-algorithm
├─ multiple patterns                     → Aho-Corasick
├─ longest common prefix on set          → trie
├─ longest with K distinct chars         → sliding window (variable)
└─ palindrome substrings                 → Manacher or DP

Stream
├─ kth largest                           → min-heap of size k
├─ median                                → two heaps
└─ random sample                         → reservoir sampling

Bits
├─ duplicates / missing single           → XOR
├─ subset enumeration                    → bitmask DP
└─ flag tests                            → AND mask

Subarray min / max                       → monotonic stack or deque
Sliding-window aggregate (max/min)       → monotonic deque
Sliding-window aggregate (sum)           → running sum
```

## Common Errors

These are the failure modes that bite practitioners again and again. Memorize the diagnosis as much as the fix.

- **Picking the wrong pattern.** "I need shortest path" → BFS only works for unweighted; Dijkstra requires non-negative weights; Bellman-Ford for negative weights; Floyd-Warshall for all-pairs. Re-read the constraints.
- **Off-by-one in binary search.** Inconsistent loop conditions (`lo <= hi` vs `lo < hi`) and inconsistent updates (`hi = mid` vs `hi = mid - 1`). Pick a single template and stick to it.
- **Not handling empty inputs.** Sliding window with `k > len(arr)`, sort + scan with empty array, BFS with no source. Always test the empty case first.
- **Integer overflow in mid calculation.** `mid = (lo + hi) // 2` can overflow in C/Java. Use `mid = lo + (hi - lo) // 2`. In Python overflow is impossible — but in Go `int` is platform-dependent (32 or 64 bit), so be careful.
- **Recursion depth limit.** Python defaults to 1000; deep DFS on a 10⁵-node tree explodes. Either rewrite iteratively with an explicit stack or `sys.setrecursionlimit(...)`.
- **Mutable shared state in backtracking.** Forgetting to `path.pop()` after recursing leaves stale state for the next branch. Always pair `choose` with `unchoose`.
- **Stale visited set in BFS.** Marking visited at *dequeue* time instead of enqueue time can cause exponential blowup as multiple copies of the same node enter the queue.
- **Off-by-one on prefix sums.** `prefix[0] = 0` and `range_sum(L, R) = prefix[R+1] - prefix[L]`. Mixing 0-indexed and 1-indexed prefix arrays is a perennial bug.
- **Forgetting `++left` or `--right` in two pointers.** A loop that fails to advance pointers in some branch becomes infinite. Always verify every branch updates at least one pointer.
- **Modifying input during iteration.** Removing zeros from an array while iterating over it skips elements. Use a separate index or rebuild.
- **Negative weights with Dijkstra.** Dijkstra silently produces wrong results — it doesn't crash. If weights can be negative, use Bellman-Ford.
- **Tail recursion not optimized.** Python and Java do not eliminate tail calls. Deep tail recursion still blows the stack.

## Common Gotchas

Each pair below shows the broken code first, then the fix. The patterns recur frequently in interviews and production code.

### Gotcha 1: Binary search infinite loop

```python
# BROKEN — infinite loop when lo = hi - 1 and pred(mid) is true
while lo < hi:
    mid = (lo + hi) // 2  # rounds down
    if pred(mid):
        lo = mid          # lo never changes
    else:
        hi = mid - 1
```

```python
# FIXED — round up when retaining lo
while lo < hi:
    mid = (lo + hi + 1) // 2
    if pred(mid):
        lo = mid
    else:
        hi = mid - 1
```

### Gotcha 2: Sliding window left advance

```python
# BROKEN — comparing wrong index when shrinking
while right - left + 1 > k:
    counts[s[left]] -= 1   # decrements before bound check
    left += 1
```

```python
# FIXED — use invariant violation, not position math, as the trigger
while len(counts) > k:
    counts[s[left]] -= 1
    if counts[s[left]] == 0:
        del counts[s[left]]
    left += 1
```

### Gotcha 3: Prefix sum hashmap initialization

```python
# BROKEN — can't count subarrays starting from index 0
counts = {}
prefix = 0
for x in arr:
    prefix += x
    total += counts.get(prefix - k, 0)
    counts[prefix] = counts.get(prefix, 0) + 1
```

```python
# FIXED — seed the hashmap with prefix sum 0 already seen once
counts = {0: 1}
prefix = 0
for x in arr:
    prefix += x
    total += counts.get(prefix - k, 0)
    counts[prefix] = counts.get(prefix, 0) + 1
```

### Gotcha 4: Backtracking forgot to undo

```python
# BROKEN — cols set is never reset on return
def bt(r):
    if r == n: res.append(...); return
    for c in range(n):
        if c in cols: continue
        cols.add(c)
        bt(r + 1)
        # forgot cols.remove(c)
```

```python
# FIXED — always pair choose with unchoose
def bt(r):
    if r == n: res.append(...); return
    for c in range(n):
        if c in cols: continue
        cols.add(c)
        bt(r + 1)
        cols.remove(c)
```

### Gotcha 5: BFS marking visited too late

```python
# BROKEN — marks visited at dequeue; same node enqueued many times
q = deque([src])
while q:
    u = q.popleft()
    if u in seen: continue
    seen.add(u)
    for v in adj[u]:
        q.append(v)
```

```python
# FIXED — mark visited at enqueue
seen = {src}
q = deque([src])
while q:
    u = q.popleft()
    for v in adj[u]:
        if v not in seen:
            seen.add(v)
            q.append(v)
```

### Gotcha 6: 0/1 knapsack iteration order

```python
# BROKEN — iterating capacity forward lets the same item be reused
for w in range(weight, W + 1):
    dp[w] = max(dp[w], dp[w - weight] + value)
```

```python
# FIXED — iterate capacity backward for 0/1 knapsack
for w in range(W, weight - 1, -1):
    dp[w] = max(dp[w], dp[w - weight] + value)
```

### Gotcha 7: Dijkstra outdated heap entries

```python
# BROKEN — assumes the popped distance is still current
while heap:
    d, u = heapq.heappop(heap)
    for v, w in adj[u]:
        if d + w < dist[v]:
            ...
```

```python
# FIXED — skip stale entries
while heap:
    d, u = heapq.heappop(heap)
    if d > dist[u]:
        continue
    for v, w in adj[u]:
        if d + w < dist[v]:
            ...
```

### Gotcha 8: Two-pointer 3-sum duplicate skipping

```python
# BROKEN — skipping after using i causes missed triplets
for i in range(n):
    if nums[i] == nums[i - 1]:  # also skips i == 0
        continue
```

```python
# FIXED — guard the skip with i > start
for i in range(n):
    if i > 0 and nums[i] == nums[i - 1]:
        continue
```

### Gotcha 9: Topological sort losing isolated nodes

```python
# BROKEN — only enqueues nodes that appear in edges
q = deque([n for n in nodes_with_indeg_zero])
```

```python
# FIXED — all nodes start with in-degree 0 unless an edge gave them one
indeg = [0] * n
for a, b in edges:
    indeg[b] += 1
q = deque(i for i in range(n) if indeg[i] == 0)
```

### Gotcha 10: Fast/slow pointer null dereference

```python
# BROKEN — fast.next.next can NPE if fast.next is None
while fast and fast.next.next:
    ...
```

```python
# FIXED — verify fast.next first
while fast and fast.next:
    slow = slow.next
    fast = fast.next.next
```

### Gotcha 11: Quickselect on already-sorted arrays

```python
# BROKEN — picking the last element as pivot makes quickselect O(n²) on sorted input
pivot = arr[hi]
```

```python
# FIXED — randomize the pivot
pivot_idx = random.randint(lo, hi)
arr[pivot_idx], arr[hi] = arr[hi], arr[pivot_idx]
pivot = arr[hi]
```

### Gotcha 12: Modifying a dictionary while iterating

```python
# BROKEN — RuntimeError: dictionary changed size during iteration
for k in counts:
    if counts[k] == 0:
        del counts[k]
```

```python
# FIXED — collect keys first, then delete
for k in [k for k, v in counts.items() if v == 0]:
    del counts[k]
```

## Idioms

These are the higher-level patterns that organize how you *think* about problems, not just how you code them.

- **"If it looks like a graph, it is a graph."** Many problems on grids, strings, or ranks of items are cleaner when restated as graph traversals. The state space *is* the graph.
- **"DP is recursion + memoization."** If you can write the recurrence, you can write the DP. Start top-down; convert to bottom-up only when you need the speedup or the iterative space optimization.
- **"Binary search the answer when the search space is monotonic."** "What's the minimum capacity / largest sum / max distance?" with a feasibility predicate that's monotonic in the answer → binary search.
- **"Two pointers are 90% of array problems."** Sliding window, partitioning, dedup, target-sum on sorted, palindrome — all variants of two pointers.
- **"Recursion + memoization is always equivalent to bottom-up DP."** They differ in stack usage and cache locality, not in time complexity. Use whichever reads cleaner; switch only when forced.
- **"When in doubt, sort."** A surprising number of problems become trivial after sorting: 3-sum, meeting rooms, interval merge, greedy scheduling.
- **"Hashmaps trade space for time."** Whenever you find yourself doing a `O(n²)` lookup, ask "could a hashmap make this O(n)?"
- **"BFS for shortest, DFS for existence."** BFS guarantees the first path found is shortest. DFS is fine when you only need to know whether a path exists.
- **"Heaps are the answer to 'top-k' and 'median.'"** Two heaps for medians; one heap of size k for top-k.
- **"Union-find for offline connectivity."** When you don't need real-time path queries between specific endpoints, union-find is faster than DFS for "are they connected?" — and faster than maintaining adjacency lists when components merge frequently.
- **"Difference arrays beat segment trees for range update + final read."** If all updates come *before* all reads, difference arrays are O(1) per update.
- **"Bit manipulation when n ≤ 20."** Subset DP, traveling salesman variants, and assignment problems become tractable with bitmasks at small `n`.
- **"Coordinate compression makes BIT/segment-tree-friendly indexing."** When values are sparse, replacing them with ranks lets you use array-indexed structures.
- **"The problem is the algorithm; the implementation is the problem."** Once you've named the pattern, you've solved 80% of the problem; the remaining 20% is bookkeeping. If implementation feels hard, you may have the wrong pattern.
- **"Read the constraints first."** `n ≤ 20` shouts bitmask DP. `n ≤ 5000` allows O(n²) DP. `n ≤ 10⁵` demands O(n log n). `n ≤ 10⁹` requires O(log n) or math. Constraints are the strongest pattern hint you'll get.

## See Also

- big-o-complexity — when each pattern's running time matters; how to choose between O(n log n) and O(n²) approaches
- graph-theory — formal foundations behind BFS/DFS/Dijkstra/MST
- complexity-theory — P, NP, NP-complete; why some patterns can't help
- distributed-systems — when you need to move beyond single-machine algorithms
- paxos — consensus is itself an algorithmic pattern
- crdt — patterns for conflict-free distributed state
- coding-problems — every entry there exercises one of these patterns

## References

- Cormen, Leiserson, Rivest, Stein — *Introduction to Algorithms* (CLRS), 4th ed., MIT Press, 2022. The canonical algorithm reference.
- Sedgewick, Wayne — *Algorithms*, 4th ed., Addison-Wesley, 2011. Companion to the Princeton Algorithms Coursera course; clearer prose than CLRS.
- Aziz, Lee, Prakash — *Elements of Programming Interviews* (EPI), 2nd ed., 2018. Pattern-driven interview prep with worked examples.
- Skiena — *The Algorithm Design Manual*, 3rd ed., Springer, 2020. Excellent "war stories" plus pattern catalog.
- Halim — *Competitive Programming 4*, 2020. Strong on contest-style techniques (BIT, segment tree, network flow).
- Knuth — *The Art of Computer Programming*, vols. 1–4. The foundational treatment of every pattern.
- Sedgewick — Princeton Coursera "Algorithms, Part I" and "Part II". Free, comprehensive video course.
- Erickson — *Algorithms*, 1st ed., 2019. Free open-source textbook; available at https://jeffe.cs.illinois.edu/teaching/algorithms/
- competitive-programming.io — practice problem set organized by pattern.
- LeetCode patterns guides — community-curated lists of "fifteen patterns" and "75 essential problems" used as interview-prep canon.
- Dasgupta, Papadimitriou, Vazirani — *Algorithms*, McGraw-Hill, 2008. Concise treatment of D&C, DP, and graph algorithms.
- Tarjan — *Data Structures and Network Algorithms*, SIAM, 1983. Classical reference for union-find and graph algorithm analysis.
- Sleator, Tarjan — "A Data Structure for Dynamic Trees", 1983. Original treatment of link-cut trees.
- Karatsuba — "Multiplication of multidigit numbers on automata", Doklady Akademii Nauk SSSR, 1962. Original sub-quadratic multiplication.
- Strassen — "Gaussian elimination is not optimal", Numerische Mathematik, 1969. Sub-cubic matrix multiplication.
- Aho, Corasick — "Efficient string matching: an aid to bibliographic search", CACM 1975. Multi-pattern matching algorithm.
- Knuth, Morris, Pratt — "Fast pattern matching in strings", SIAM J. Comp., 1977. KMP failure function.
- Z-algorithm — Gusfield, *Algorithms on Strings, Trees, and Sequences*, Cambridge, 1997.
- Floyd — "Nondeterministic algorithms", JACM 1967. Tortoise-and-hare cycle detection.
- Manacher — "A new linear-time on-line algorithm finding the smallest initial palindrome of a string", JACM 1975. Linear-time palindrome detection.
- Knuth — "Algorithms in Modern Mathematics and Computer Science", Springer, 1981. Reservoir sampling proofs and analysis.
- Cormode, Muthukrishnan — "An improved data stream summary: the count-min sketch and its applications", J. Algorithms, 2005.
- Flajolet, Fusy, Gandouet, Meunier — "HyperLogLog: the analysis of a near-optimal cardinality estimation algorithm", DMTCS 2007.
- Bloom — "Space/time trade-offs in hash coding with allowable errors", CACM 1970.
- man pages — `man 3 qsort`, `man 3 bsearch` (POSIX), `go doc sort.Search`, `go doc container/heap`, Python `bisect`, `heapq`, `functools.lru_cache`.
