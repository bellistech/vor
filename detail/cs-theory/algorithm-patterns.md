# Algorithm Patterns — Deep Dive

The why and the proofs behind classical algorithm design patterns — from sliding window invariants through Tarjan's lowlinks and Manacher's palindrome reuse.

## Setup

Computer science distinguishes "algorithm" from "ad-hoc problem-solving." Most problems you encounter in interviews, competitive programming, and production code are not unique snowflakes. They fall into recurring families. Recognizing the family unlocks the algorithm.

This is the pattern-recognition view of algorithms. Rather than seeing "an array problem," you see "a sliding-window problem with a monotone predicate." Rather than seeing "a graph problem," you see "Dijkstra applies because edges are non-negative and we need shortest path."

The patterns themselves are not arbitrary. Each is a structural insight: a property of the problem that allows certain operations to compose efficiently. Sliding window leverages the monotone property; binary search on answer leverages monotonicity of a predicate; greedy works when the exchange argument or matroid structure holds; DP works when there is optimal substructure.

This deep dive is the WHY — the correctness proofs, the impossibility cases (greedy fails for 0/1 knapsack), the complexity bounds, the structural conditions that justify each pattern. The HOW — code templates and specific problem reductions — lives in the cheatsheet. Here we focus on understanding so deeply that recognizing the pattern becomes second nature.

## Sliding Window Theory

The sliding window pattern computes some property over all subarrays of fixed size k (or all subarrays satisfying some condition). The key insight: rather than recomputing the property from scratch for each window, maintain it incrementally as the window shifts.

**The basic invariant:** Suppose property P over a subarray A[i..j] can be computed in O(1) given P over A[i..j-1] and A[j], plus the removal of A[i] when it falls out. If both addition and removal are O(1), the entire sweep is O(n).

**Example: maximum sum subarray of length k.**
- Initial: sum of A[0..k-1].
- Shift by 1: sum -= A[i], sum += A[j].
- Each shift is O(1). Total: O(n).

**The monotone-decision property.** A more powerful sliding-window pattern handles variable window size. The window grows or shrinks based on a condition. The condition is "monotone" if expanding the window can only make it less valid (in one direction) or more valid (in the other).

**Example: longest substring with at most k distinct characters.**
- Maintain a counter of distinct characters in window [left, right].
- Expand right: add A[right] to counter; if distinct count > k, shrink left until ≤ k.
- The window's "validity" is monotone in left (smaller left = more characters; can become invalid).
- Each character is added and removed at most once. Total: O(n).

**Why it works:** The "validity" predicate is monotone. If [left, right] is valid, then [left+1, right] is at least as valid (it contains a subset of characters). So as we expand right, we shrink left only as necessary; we never need to backtrack.

**Why O(n) total work:** Each element is added to the window once and removed once. Total operations: 2n. Constant per operation.

**Generalized window patterns:**
- Fixed-size window: O(n), simple shift.
- Monotone-condition window: O(n), expand/shrink based on predicate.
- Two pointers (next section): a special case of monotone window where movement is one-directional.

**Pitfalls:**
- Non-monotone predicates: e.g., "subarray with sum ≤ K and length ≤ M" — both conditions are monotone, but their intersection might not be. Often resolved by tracking multiple criteria.
- Negative numbers in sum-based windows: invalidate the simple expand/shrink logic. Need different techniques (prefix sums, segment trees).

## Two Pointers Theory

The two-pointer pattern uses two indices into a (typically sorted) array, moving them based on a condition. The reduction from O(n²) to O(n) is its hallmark.

**Sweep + Condition Pattern:** Two pointers (left, right) sweep across a sorted array. At each step, examine A[left] and A[right]; based on a condition, advance one or the other. Stop when pointers meet.

**Example: 2-Sum on sorted array.** Find indices i, j with A[i] + A[j] = target.
- Initialize left=0, right=n-1.
- While left < right:
  - sum = A[left] + A[right].
  - If sum == target: return (left, right).
  - If sum < target: left++ (need larger).
  - If sum > target: right-- (need smaller).
- Return not found.

**Correctness proof:** We need to show that whenever we move a pointer, no valid pair is skipped.
- Suppose A[left] + A[right] < target. Any pair (left, right') with right' < right has A[right'] ≤ A[right], so A[left] + A[right'] ≤ A[left] + A[right] < target. No valid pair starts at left. We can safely move left.
- Symmetrically, if sum > target, we can safely move right.
- So at each step, the discarded pointer cannot be part of any valid pair.

**Complexity:** Each pointer moves at most n times. Total: O(n).

**Generalizations:**
- 3-Sum: fix one element, then 2-Sum on the rest. O(n²).
- 4-Sum: fix two elements, then 2-Sum. O(n³).
- k-Sum (general): O(n^(k-1)) with fixed prefix.

**Two-pointer for partitioning:** Lomuto partition (used in quicksort) uses two pointers — one for the partition boundary, one for the scan. Hoare partition uses two pointers from opposite ends.

**Two-pointer in linked lists:**
- Cycle detection: slow (1 step) and fast (2 steps) pointers. Floyd's algorithm.
- Find middle: slow and fast pointers; when fast reaches end, slow is at middle.
- Find n-th from end: advance fast n steps, then both move together.

**The pattern's elegance:** Reduces nested-loop O(n²) to single-loop O(n) by exploiting structure (sorted order, monotone condition). The reduction is rigorous and proof-supported.

## Binary Search on Answer

Binary search is usually taught as searching for an element in a sorted array. A more powerful variant: binary search on the answer to an optimization problem.

**Pattern:** Suppose you want to find the minimum (or maximum) X such that some predicate P(X) is true. If P is **monotone** in X — i.e., P(X) implies P(Y) for all Y ≥ X — then binary search applies.

**Bisection theorem:** Let f: [a, b] → ℝ be a continuous function with f(a) < 0 < f(b). Then there exists c ∈ [a, b] with f(c) = 0. By repeated bisection, we can find c to any precision in O(log(1/ε)) iterations. The key is monotonicity (or at least continuity for IVT) — without it, bisection can miss zeros.

For decision problems with discrete X (integers), binary search converges in O(log(b-a)) steps if P is monotone.

**Example: minimum capacity to ship packages within D days.**
- Predicate P(C): can we ship all packages within D days using capacity C?
- P is monotone: larger C makes shipping easier.
- Binary search C in [max(weights), sum(weights)]. P(low) = false (can't ship single package), P(high) = true (one big day). Bisect until found.
- Each predicate check: O(n). Total: O(n log(sum)).

**Example: median of two sorted arrays.**
- Binary search on the partition point in array 1. The partition determines the position in array 2 (must split the combined array equally). Check whether the partition is valid (max of left ≤ min of right). Monotone in the partition position.
- O(log min(n, m)).

**Example: K-th smallest element in a sorted matrix.**
- Predicate P(X): are there at least K elements ≤ X?
- Binary search X in [matrix.min, matrix.max].
- P is monotone in X.
- Each predicate check: O(n) using two-pointer scan (since rows are sorted).
- Total: O(n log(max - min)).

**When binary search on answer fails:**
- Non-monotone predicate: P(X) might oscillate. Then binary search loses correctness; ternary search or other techniques may apply.
- Continuous predicates with stiffly varying P: numerical precision concerns.
- Integer ranges with gaps: must verify discrete monotonicity.

**Variations:**
- Lower bound (first true): converges to the minimum X satisfying P.
- Upper bound (last false): converges to the maximum X failing P; X+1 is the answer.
- Real-valued: bisect until interval is < ε.

## Greedy Correctness

A greedy algorithm makes the locally optimal choice at each step, hoping the result is globally optimal. Surprisingly, this works for many problems — but not all. Proving greedy correctness requires a structural argument.

**Exchange Argument:** The most common proof technique. To show greedy produces an optimal solution:
1. Assume an optimal solution O.
2. Show that O can be transformed (one exchange at a time) into the greedy solution G without increasing cost.
3. Therefore G is optimal.

**Example: Activity selection.** Given activities with start and end times, select the maximum number of non-overlapping activities.

Greedy: sort by end time, repeatedly pick the activity with the earliest end time that doesn't conflict.

Proof: Let O = (o₁, o₂, ...) be an optimal solution sorted by start time. Let G = (g₁, g₂, ...) be the greedy solution. We show G has at least as many activities.

By induction: at each step k, G's first k activities have end times ≤ O's first k activities. (Induction base: g₁ has the earliest end time overall, so g₁'s end ≤ o₁'s end.)

If O continues with o_{k+1}, then any activity starting after o_k's end can be in O. Since g_k's end ≤ o_k's end, the same activity can follow g_k. Greedy will pick the earliest-ending such activity, ensuring g_{k+1}'s end ≤ o_{k+1}'s end.

Thus G can match or exceed O. QED.

**Matroid Theory:** A more powerful framework. A matroid is a pair (E, I) where E is a finite ground set and I is a collection of "independent" subsets satisfying:
- ∅ ∈ I.
- Hereditary: A ∈ I and B ⊆ A implies B ∈ I.
- Exchange: A, B ∈ I, |A| < |B|, then there exists x ∈ B \ A such that A ∪ {x} ∈ I.

**Theorem (Edmonds):** For any matroid, the greedy algorithm — sort elements by weight, add each greedily if it preserves independence — finds the maximum-weight independent set.

**Examples of matroids:**
- Graphic matroid: elements = edges, independent sets = forests. Kruskal's algorithm follows.
- Uniform matroid: elements = any set, independent = any subset of size ≤ k.
- Linear matroid: elements = vectors, independent = linearly independent.

**Stay-Ahead Argument:** Show that at every step, greedy's solution is at least as "ahead" as any optimal solution.

**Example: Minimum number of platforms (interval scheduling).** Schedule meetings to use minimum number of rooms. Greedy: process in order of start times; assign each meeting to a free room (or a new one if none free).

Proof: at any moment, greedy uses no more rooms than any solution must, because greedy reuses rooms as soon as they become free.

## Greedy Counter-Examples

Greedy does not always work. The 0/1 knapsack is the canonical counterexample.

**Fractional Knapsack: Greedy Works.** Items have weights w_i and values v_i. Capacity W. Take fractions of items.

Greedy: sort by v_i/w_i (value-per-weight) descending; take the most valuable items until full.

Proof: Suppose for contradiction the optimal solution differs from greedy. There must be some item i not taken (or partially taken) by optimal that has higher v/w than some item j taken (more) by optimal. Swap a small fraction: remove dx of j (gain v_j/w_j · dx) and add dx of i (gain v_i/w_i · dx). Since v_i/w_i > v_j/w_j, the swap improves. Optimal not optimal — contradiction.

**0/1 Knapsack: Greedy Fails.** Items cannot be split. Greedy by v/w can fail.

Counterexample: capacity W = 50.
- Item 1: weight 10, value 60. v/w = 6.
- Item 2: weight 20, value 100. v/w = 5.
- Item 3: weight 30, value 120. v/w = 4.

Greedy by v/w: take item 1 (value 60), then item 2 (value 100). Total: 30 weight, 160 value. Cannot fit item 3 (would total 60 > 50).

Optimal: take item 2 and item 3. Total: 50 weight, 220 value. Better.

The greedy choice (item 1) was locally best per unit weight but suboptimal globally because it left "wasted" capacity.

**Why greedy fails for 0/1 knapsack:** The exchange argument doesn't go through. We cannot exchange a fraction; we must swap whole items. The discrete nature breaks the proof.

**The general lesson:** Greedy works when the structure has the matroid (or similar) property; it fails for many discrete optimization problems. To know if greedy applies, attempt the exchange argument; if it fails, use DP.

**Other classic non-greedy problems:**
- Coin change with arbitrary denominations: greedy can fail. E.g., coins {1, 3, 4}, target 6: greedy picks 4+1+1=3 coins; optimal is 3+3=2 coins.
- Traveling salesman: nearest-neighbor heuristic can be 2x worse than optimal.
- Longest path: greedy in random order is meaningless.

For these, DP or exact methods are needed.

## DP Bellman's Principle

Dynamic Programming (DP) was named by Richard Bellman in the 1950s, working on multistage decision processes for the RAND Corporation. The key principle:

**Bellman's Principle of Optimality:** An optimal policy has the property that, regardless of the initial state and decision, the remaining decisions must constitute an optimal policy with respect to the state resulting from the first decision.

In simpler terms: **optimal substructure**. The optimal solution to a problem includes optimal solutions to subproblems. Combined with **overlapping subproblems** (the same subproblems recur), this lets us compute each subproblem once.

**Optimal substructure example: shortest path.** The shortest path from A to C through B = shortest path from A to B + shortest path from B to C. (For graphs without negative cycles.)

**Counter-example: longest simple path.** The longest simple path from A to C through B is NOT longest A→B + longest B→C, because the two parts may share vertices. No optimal substructure → DP doesn't apply directly.

**Tabulation vs Memoization:**
- **Memoization (top-down):** Recursive function with cache. Solves only the subproblems actually needed.
- **Tabulation (bottom-up):** Iterative, fills a table. Computes all subproblems in dependency order.

**Equivalence:** Both produce the same answer with the same complexity (assuming all subproblems would be visited). Memoization is often easier to write; tabulation is often faster (no recursion overhead, better cache locality).

**Memoization can be more efficient when many subproblems are unreached.** For example, in a recursive algorithm where most branches prune early, memoization avoids the unused subproblems.

**Tabulation can be more efficient when state ordering is complex.** For example, in 2D DP with dependencies in multiple directions.

**Space optimization:** Many DP problems allow rolling arrays. If dp[i] depends only on dp[i-1] and dp[i-2], we need only O(1) space. The 2D Knapsack DP (size O(n·W)) often becomes O(W) with care.

**The DP recipe:**
1. Identify subproblems (state).
2. Define recurrence (transition).
3. Identify base cases.
4. Order computation (tabulation order or memoization with recursion).
5. Trace back if needed (recover the actual solution, not just the value).

## DP Recurrence Templates

**1D DP:** dp[i] depends on dp[i-1], dp[i-2], etc.

**Fibonacci:** dp[i] = dp[i-1] + dp[i-2]. O(n) time, O(1) space with rolling.

**House robber:** dp[i] = max(dp[i-1], dp[i-2] + house[i]). Choose to skip or take the i-th house.

**Climbing stairs:** dp[n] = dp[n-1] + dp[n-2]. Number of ways to climb taking 1 or 2 steps.

**Maximum product subarray:** Track both max and min ending at i (since negative * negative = positive).

**2D DP:** dp[i][j] from grid neighbors.

**Unique paths in grid:** dp[i][j] = dp[i-1][j] + dp[i][j-1]. With obstacles: skip obstacle cells.

**Minimum path sum:** dp[i][j] = grid[i][j] + min(dp[i-1][j], dp[i][j-1]).

**Triangle:** dp[i][j] = min(dp[i+1][j], dp[i+1][j+1]) + triangle[i][j]. Bottom-up.

**Knapsack 0/1:** dp[i][w] = max(dp[i-1][w], dp[i-1][w - w_i] + v_i). The recurrence: don't take item i (dp[i-1][w]) or take it (dp[i-1][w-w_i] + v_i).

Space: dp[w] = max(dp[w], dp[w - w_i] + v_i), iterating w in reverse to avoid using item i twice.

**Unbounded Knapsack:** Items can repeat. dp[w] = max(dp[w], dp[w - w_i] + v_i), iterating w forward.

**LIS (Longest Increasing Subsequence):**

O(n²) standard: dp[i] = 1 + max(dp[j] for j < i if A[j] < A[i]). dp[i] is the LIS ending at i. Final answer: max(dp).

**O(n log n) via patience sort:** Maintain a list of "piles." For each A[i], place it on the leftmost pile whose top is ≥ A[i] (replace), or start a new pile. Number of piles = LIS length.

This is the patience-sort technique. Equivalently: maintain an array tails where tails[k] = smallest possible tail of an LIS of length k+1. For each A[i], binary search for the leftmost tails ≥ A[i] and replace it. The length of tails grows.

Proof: tails is sorted. A[i] either extends the LIS (appended to tails) or improves a smaller LIS's tail.

**LCS (Longest Common Subsequence):**

dp[i][j] = LCS of A[0..i-1] and B[0..j-1].

If A[i-1] == B[j-1]: dp[i][j] = dp[i-1][j-1] + 1.
Else: dp[i][j] = max(dp[i-1][j], dp[i][j-1]).

O(nm) time, O(nm) space (or O(min(n,m)) with rolling).

**Edit Distance (Levenshtein):**

dp[i][j] = edit distance between A[0..i-1] and B[0..j-1].

If A[i-1] == B[j-1]: dp[i][j] = dp[i-1][j-1].
Else: dp[i][j] = 1 + min(dp[i-1][j-1] (replace), dp[i-1][j] (delete A), dp[i][j-1] (insert)).

O(nm) time. Used in spell checkers, diff utilities.

**Interval DP:** dp[i][j] over intervals A[i..j].

**Matrix Chain Multiplication:** dp[i][j] = min cost to multiply matrices i..j. dp[i][j] = min over k of (dp[i][k] + dp[k+1][j] + cost(i, k, j)).

**Burst Balloons:** dp[i][j] = max coins from bursting balloons in (i, j) exclusive. Transition: choose the last balloon to burst.

**Tree DP:** Postorder DFS, compute dp values for each subtree.

**Maximum Independent Set in a tree:** dp[v][0] = max IS not including v, dp[v][1] = max IS including v.
- dp[v][0] = sum over children c of max(dp[c][0], dp[c][1]).
- dp[v][1] = value(v) + sum over children c of dp[c][0].

**House robber III (binary tree):** Same idea: choose to rob this node or its children.

**Diameter of tree:** For each node, find the two longest paths to leaves; diameter is the max sum.

**Bitmask DP:** State = subset of items, encoded as a bitmask.

**TSP:** dp[mask][i] = min cost visiting cities in mask, ending at i. Transition: dp[mask | (1<<j)][j] = min(dp[mask][i] + dist(i, j)) for j not in mask. O(n² · 2^n).

**Subset sum / coverage problems:** dp[mask] = some property over the subset.

**Permutations:** dp[mask] = number of valid prefixes corresponding to mask of placed items.

Bitmask DP scales to ~20-25 elements (2^20 = ~10^6 states).

**Digit DP:** Count integers in [L, R] satisfying some digit-wise property.

State: (position, tight, leading_zero, additional_state). "Tight" tracks whether we're still bounded by the original digits. "Leading zero" tracks if we've placed a non-zero digit yet.

Recurrence: for each digit d in [0..9] (or [0..tight ? digit[pos] : 9]), recurse.

Memoize on (pos, tight, leading_zero, state). Often O(D · 2 · 2 · S) where D = digit count, S = additional state size.

Used for counting numbers with specific digit patterns, sum of digits, digit DP on tree problems, etc.

## Backtracking Theory

Backtracking is depth-first search over the solution space, with pruning when a partial solution cannot lead to a valid full solution.

**Pattern:**
1. Choose: try each possible move from the current state.
2. Constrain: check if the partial solution can still be extended.
3. Goal: if a complete valid solution, record it.
4. Backtrack: undo the move and try the next.

**Branch and Bound:** Pruning via upper or lower bounds. If the best possible completion of a partial solution is worse than the current best, prune.

**Example: N-Queens.**
- State: which column queen has been placed in for each row.
- Move: try each column for the next row.
- Constraint: no two queens in same column or diagonal.
- Pruning: if a queen's placement creates a conflict, abandon this branch.

**Example: Sudoku solver.**
- State: filled cells.
- Move: try each digit for an empty cell.
- Constraint: no row/column/box conflict.
- Pruning: detected conflict aborts the branch.

**Example: Combination sum.**
- State: chosen elements with their sum.
- Move: include or exclude each remaining candidate.
- Pruning: if sum > target, prune; if sum + max(remaining) < target, prune.

**Branch-and-Bound for optimization:**
- Maintain the best solution found so far.
- When a branch's lower bound (for minimization) ≥ best, prune.
- Lower bounds come from relaxations (e.g., LP relaxation of integer programs).

**TSP via branch-and-bound:** Lower bound = sum of two smallest edges per node. If a partial path's bound ≥ best, prune.

**Backtracking complexity:** Exponential in worst case (no pruning), but pruning often reduces by orders of magnitude in practice. Hard to bound rigorously without problem-specific analysis.

## BFS Optimality

Breadth-First Search finds shortest paths in unweighted graphs.

**Theorem:** In an unweighted graph, BFS from source s computes d(s, v) for every reachable v, where d is the number of edges on the shortest path.

**Proof (invariant):** When BFS visits a node v, d(s, v) equals the smallest k such that v is at distance ≤ k from s.

By induction on k:
- Base: d(s, s) = 0, visited at level 0.
- Inductive step: assume all nodes at distance k are visited at level k. When level k is processed, we explore each node's neighbors. A neighbor v is at distance k+1 (if it wasn't already visited). It is added to level k+1. So when level k+1 is processed, all nodes at distance k+1 are visited.

The invariant ensures that the order of visiting is by increasing distance. The visited set forms increasing-distance "shells."

**Why BFS doesn't work for weighted graphs:** A node may be reachable via a 5-edge path of total weight 10, or a 2-edge path of total weight 100. BFS prefers the 2-edge path (visits sooner), but the 5-edge path has smaller total weight. The "level" abstraction breaks.

For weighted graphs (non-negative), use Dijkstra. For weighted graphs (with negative edges), use Bellman-Ford or Floyd-Warshall.

**Variants of BFS:**
- 0-1 BFS: edges have weights 0 or 1. Use a deque; weight-0 edges are added to the front, weight-1 edges to the back. O(V+E).
- Multi-source BFS: start with multiple sources in the initial queue. Useful for "shortest distance to any source" problems.
- Bidirectional BFS: search from both ends; meet in the middle. Halves the depth.

## Dijkstra Correctness

Dijkstra's algorithm finds shortest paths from a source in graphs with non-negative edge weights.

**Algorithm:** Maintain a priority queue of (distance, node). Initialize source's distance to 0; all others to ∞. Repeat: pop the closest unvisited node u; for each neighbor v, relax: if dist[u] + w(u, v) < dist[v], update.

**Theorem:** When Dijkstra pops a node u, dist[u] is the true shortest-path distance.

**Proof (correctness invariant):** When u is popped, every node v with dist[v] < dist[u] has already been popped with their true distances.

By induction:
- Base: source is popped first with dist=0, which is correct.
- Inductive step: suppose the invariant holds for all popped nodes so far. Let u be the next popped. Suppose for contradiction there is a shorter path P from source to u with total weight < dist[u].

Path P passes through some sequence of nodes. Let x be the first node on P that has not been popped. (x might be u itself.) The predecessor of x on P, call it w, has been popped, and dist[w] is correct. Then dist[x] ≤ dist[w] + weight(w, x) (since we relaxed w's neighbors when w was popped). So dist[x] ≤ part of P from source to x = part of total weight of P ≤ total weight of P < dist[u].

But dist[x] < dist[u] contradicts our choice of u (the priority queue would pop x first). Contradiction.

So no such P exists; dist[u] is correct. QED.

**Critical assumption: non-negative weights.** With negative edges, the proof fails. Specifically, "relaxed w's neighbors when w was popped" assumes w's distance is final; with negative edges, dist[w] could decrease later.

**Complexity:**
- With binary heap: O((V + E) log V).
- With Fibonacci heap: O(E + V log V). Theoretical improvement; practical wins are modest due to constants.
- With array (unsorted): O(V²). Better when E is dense (E ~ V²).

## Bellman-Ford

Handles negative edges, detects negative cycles.

**Algorithm:** Relax all E edges, V-1 times. If any relaxation succeeds in the V-th iteration, there's a negative cycle.

**Why V-1 iterations:** A simple path has at most V-1 edges. After V-1 relaxations, all shortest paths to all reachable nodes from a fixed source have been considered.

**Proof:** Let d_k(v) = shortest path length using at most k edges. After iteration k, dist[v] = d_k(v) for all v. Since the longest simple path has V-1 edges, d_{V-1}(v) is the actual shortest path length (assuming no negative cycle).

If a V-th relaxation succeeds, the path was not "finished" after V-1; a negative cycle exists.

**Complexity:** O(V · E). Slower than Dijkstra but handles negatives.

**SPFA (Shortest Path Faster Algorithm):** Bellman-Ford with a queue. Only processes nodes whose distance changed. Often faster in practice but same worst-case O(V · E).

## Floyd-Warshall

All-pairs shortest paths. O(V³) DP.

**Recurrence:** d[i][j][k] = min(d[i][j][k-1], d[i][k][k-1] + d[k][j][k-1]).

Where d[i][j][k] = shortest path from i to j using only intermediate nodes 0..k-1.

**Base case:** d[i][j][0] = w(i, j) if edge exists, ∞ otherwise.

**Final:** d[i][j][V] = shortest path using any subset of nodes as intermediates.

**Implementation:** Roll k dimension. dp[i][j] is updated in place: for each k, for each (i, j), dp[i][j] = min(dp[i][j], dp[i][k] + dp[k][j]).

**Proof of correctness:** Inductive on k. After processing k, dp[i][j] holds the shortest path from i to j using intermediates from 0..k-1.

Base: k=0, no intermediates, just direct edges.

Inductive: when processing k, the path from i to j either uses k as an intermediate or not. If not, dp[i][j] is unchanged (correct from k-1). If yes, the path is i → ... → k → ... → j; both halves use intermediates from 0..k-1. dp[i][k] and dp[k][j] are correct, so dp[i][j] = dp[i][k] + dp[k][j].

**Negative cycles:** detected if dp[i][i] becomes negative for any i.

**Use case:** dense graphs (V³ < V·E·log V when E ≥ V²/log V), or all-pairs needed.

## A* Optimality

A* is informed search: BFS/Dijkstra with a heuristic h(v) estimating distance to goal.

**Algorithm:** Priority queue ordered by f(v) = g(v) + h(v), where g(v) = known distance from source. Pop, expand, update.

**Admissible heuristic:** h(v) ≤ true distance from v to goal. Heuristic never overestimates.

**Consistent (monotonic) heuristic:** h(u) ≤ c(u, v) + h(v) for any edge (u, v). A consistency stronger than admissibility.

**Theorem:** With an admissible heuristic, A* finds the optimal path. With a consistent heuristic, A* never re-expands a node.

**Proof of admissible optimality:** Let goal be reached with f-value f*. Any node u on the optimal path has g(u) ≤ true distance from source to u, and h(u) ≤ true distance from u to goal. So f(u) ≤ true distance from source to goal = f*. Therefore u is expanded before any node v with f(v) > f*. The optimal path is found before any worse path completes.

**Proof of consistency**: when A* pops node u, g(u) is optimal. Otherwise, suppose g(u) is not optimal. Then a better path P exists. Let x be the first node on P that hasn't been popped. By consistency, g(x) along P + h(x) ≤ true total < f*. But the priority queue would have prioritized x. Contradiction.

**Common heuristics:**
- Manhattan distance (grid problems): always admissible.
- Euclidean distance: admissible if movement is unrestricted.
- Pattern databases (15-puzzle): precomputed exact distances for sub-problems.
- Differential heuristics: triangle inequality with a few "landmarks."

**Performance:** A* is typically much faster than Dijkstra when the heuristic is informative. With h ≡ 0, A* degenerates to Dijkstra.

## MST Cut Property

The Minimum Spanning Tree (MST) of a connected weighted graph is a subset of edges that forms a tree spanning all vertices with minimum total weight.

**Cut Property Theorem:** For any cut (S, V \ S) of the graph, the minimum-weight edge crossing the cut is in some MST.

**Proof (exchange argument):** Suppose T is an MST not containing the minimum cut edge e. Consider the path in T between e's endpoints. This path crosses the cut at some edge e' ≠ e. Remove e' and add e: weight decreases (or stays same with ties), connectivity preserved. So T was not minimum (or another MST exists with e). QED.

**Cycle Property:** The maximum-weight edge in any cycle is not in any MST. Symmetric to cut property.

These two properties form the foundation of MST algorithms.

**Kruskal's Algorithm:** Sort edges by weight. For each edge, add if it doesn't create a cycle (use Union-Find). O(E log E).

Justified by cut property: when we consider edge e, e is the minimum-weight unused edge. The cut between e's endpoints' components is crossed by e (and e is the minimum such edge by sort order). So e is in some MST.

**Prim's Algorithm:** Start from any vertex. Maintain a "tree" set. Repeatedly add the minimum-weight edge from tree to non-tree. O(E log V) with binary heap.

Justified by cut property: at each step, the cut between tree and non-tree is crossed by the chosen edge, which is minimum.

**Borůvka's Algorithm:** In parallel, each tree finds its minimum outgoing edge. Add all such edges. Repeat. O(E log V) but parallelizable.

## Union-Find Path Compression and Union by Rank

Union-Find (Disjoint Set Union) maintains a partition of elements with two operations: union(x, y) merges sets, find(x) returns the representative.

**Implementation:** Each element has a parent. find traverses up to the root. union connects two roots.

**Union by Rank:** When merging, attach the shorter tree under the taller. Rank = upper bound on tree height.

**Path Compression:** When find traverses to root, point all visited nodes directly to the root.

**Theorem (Tarjan 1975):** With both optimizations, m operations on n elements take O(m · α(n)) time, where α is the inverse Ackermann function.

α(n) is incredibly slow-growing: α(n) ≤ 4 for any n we will ever encounter (≤ 2^65536). For practical purposes, Union-Find operations are O(1).

**Proof sketch (highly compressed):** Define the "rank" of an element by its tree height upper bound. Group nodes into "levels" based on their rank. Each find operation can promote a node to a higher level, but each node spends O(α(n)) "credits" during its rise through levels. Amortized, each operation is O(α(n)).

The full proof is intricate, involving "buckets" of ranks and careful counting of operations within each bucket. See CLRS Chapter 21 for the rigorous version.

**Variants:**
- Weighted Union-Find: track size of each set.
- Union-Find with rollback: support undo for offline queries (e.g., Tarjan's offline LCA).
- Union by Rank vs Union by Size: equivalent O(α(n)) bound.

## KMP Failure Function

Knuth-Morris-Pratt (1977) is a string-matching algorithm with linear time. The key data structure is the failure function (also called partial-match table).

**Failure Function:** For a pattern P, fail[i] = length of the longest proper prefix of P[0..i] that is also a suffix.

"Proper prefix" means not equal to the whole string. "Also a suffix" means ends at position i.

Example: P = "abcabd". fail[0] = 0 (length 0). fail[1] = 0. fail[2] = 0. fail[3] = 1 ("a" is suffix and prefix). fail[4] = 2 ("ab" suffix and prefix). fail[5] = 0.

**Computing fail[]:** O(P) time. Use the array itself as a state machine: when extending, if the next character matches, increment; if not, fall back to fail[fail[i-1]+...]] until a match or 0.

**Matching:** Compare P against text T. On mismatch at P[j] vs T[i], set j = fail[j-1] and continue (don't reset i). Each character of T is examined at most twice (once forward, once backward via fail).

**Linearity proof:** Total work = comparisons. Each forward comparison advances i. Each fallback decreases j. Since j ≤ i, total fallbacks ≤ total advances = O(n). Total comparisons = O(n + m).

**Why fail captures the right info:** When mismatch occurs at P[j] vs T[i], we've matched P[0..j-1] to T[i-j..i-1]. We want the longest prefix of P that's also a suffix of T[i-j..i-1] = the longest prefix of P that's also a suffix of P[0..j-1] = fail[j-1]. This becomes the new j; continue matching P[j] vs T[i].

**KMP avoids re-scanning T** because the matched portion of T fits the structure of P's prefixes.

## Z-Algorithm

The Z-algorithm computes the Z-array: Z[i] = length of the longest substring starting at i that matches a prefix of the string.

**Z-Box:** During computation, maintain (l, r) — the rightmost Z-match interval seen so far. For position i:
- If i > r: compute Z[i] from scratch by comparison.
- If i ≤ r: use Z[i - l] as initial guess (since S[i..r] matches S[i-l..r-l]). If Z[i-l] < r - i, then Z[i] = Z[i-l]. Otherwise extend.

**Linearity:** Each comparison either advances r (which only increases) or terminates Z[i]. Total work is O(n).

**Equivalence with KMP:** Z and fail can be derived from each other. Z is sometimes preferred for its conceptual simplicity.

**Use cases:**
- String matching: append "$" + text to pattern, compute Z; positions where Z = pattern length are matches.
- Periodicity detection: longest period of S = position where Z[i] = n - i.
- Longest common prefix queries.

## Aho-Corasick

Aho-Corasick (1975) extends KMP to multiple patterns. It builds an automaton over all patterns simultaneously, with failure links for efficient matching.

**Goto, Failure, Output Functions:**
- Goto: for state q and character c, the next state if c continues a pattern.
- Failure: when goto fails, the next state to try (analogous to KMP's fail).
- Output: patterns ending at this state.

**Construction:**
1. Build trie of all patterns. (Goto = trie edges.)
2. BFS from the root. For each state q, compute failure[q] using parent's failure (BFS ensures parent is processed first).
3. Compute output[q] by combining state's own pattern endings + output[failure[q]].

**Matching:** Walk through text. At each character, follow goto if possible; otherwise follow failure. At each state, report output[q].

**Complexity:** Construction O(sum of pattern lengths · |alphabet|). Matching O(|text| + matches).

**Use cases:**
- Multiple-pattern search (URL filtering, virus signatures).
- DNA motif matching.
- Bioinformatics, plagiarism detection.

## Manacher's Algorithm

Manacher's (1975) finds all longest palindromic substrings in O(n).

**Palindrome Centers:** Each palindrome has a center (a character or between two characters). For each center c, let r[c] = radius of the longest palindrome centered at c.

**Idea:** As we sweep, exploit symmetry. If we have computed r[1..i-1], we know the longest palindrome ending around some "rightmost" position. For a new center i within that palindrome, its mirror has known radius. Use the mirror radius as initial guess; extend if necessary.

**Algorithm:**
- Insert "#" between each pair of characters. This makes every palindrome odd-length, simplifying the center notion.
- Maintain (center, right) — the rightmost-extending palindrome's center and right boundary.
- For each i:
  - If i < right: r[i] = min(right - i, r[2*center - i]) (mirror).
  - Else: r[i] = 0.
  - Try to extend r[i] by direct comparison.
  - If i + r[i] > right: update (center, right) to (i, i + r[i]).

**Linearity:** Each iteration extends r[i] by some amount, but `right` only increases. Total extensions across all i = O(n). Total work = O(n).

**Output:** The longest palindrome is the i with the largest r[i], translated back from the "#" inserted version.

**Manacher's elegance:** It treats all palindromes uniformly (odd-length via "#") and reuses prior work via symmetry. The proof of O(n) is via amortized analysis on `right`.

## Tarjan's SCC

Strongly Connected Components (SCCs) are maximal subsets where every pair of nodes has a path between them.

**Tarjan's Algorithm (1972):** O(V+E). Single DFS with stack.

**Lowlink:** For each node v, lowlink[v] = minimum disc[v'] over all v' reachable via DFS-tree edges plus at most one back-edge or cross-edge to an ancestor.

**Algorithm:**
1. DFS. On entering v, assign disc[v] = lowlink[v] = next index. Push v onto stack.
2. For each neighbor w:
   - If unvisited: recurse; lowlink[v] = min(lowlink[v], lowlink[w]).
   - If w on stack: lowlink[v] = min(lowlink[v], disc[w]).
3. After processing all neighbors, if lowlink[v] == disc[v]: v is the root of an SCC. Pop stack until v; that's the SCC.

**Why it works:**
- An SCC's root (the node first visited in the DFS) has lowlink = disc (no path back to ancestors).
- Non-root SCC nodes have lowlink < disc (can reach an ancestor).
- The stack maintains "nodes in unfinished SCCs."

**Proof:** When DFS exits v, lowlink[v] correctly captures the minimum disc reachable. If lowlink[v] == disc[v], v has no path back to ancestors; the stack from v upward is the SCC.

**Complexity:** O(V+E). Each edge examined once.

## Kosaraju's SCC

Alternative O(V+E) SCC algorithm. Two DFS passes.

**Algorithm:**
1. DFS on the original graph, recording finish times.
2. Reverse the graph (transpose).
3. DFS on the reversed graph, processing nodes in decreasing finish time. Each tree of the second DFS is an SCC.

**Why it works:** Finish time captures DFS-tree topological order on the SCC condensation. Processing in reverse finish order on the transposed graph visits one SCC at a time.

**Proof:** In the original graph, the SCC with the latest finish time is a "source" SCC (no incoming edges from other SCCs). Reversing makes it a "sink" — DFS from any of its nodes stays within that SCC. Subsequent DFS from another node finds the next SCC.

**Comparison:** Tarjan's is conceptually more elegant and uses one DFS. Kosaraju's is easier to teach. Both are O(V+E).

## Network Flow

A flow network is a directed graph with capacities on edges. We want to send maximum flow from source s to sink t, respecting capacities and conservation.

**Max-Flow Min-Cut Theorem (Ford-Fulkerson 1956):** The maximum flow equals the minimum cut capacity.

A cut is a partition (S, T) with s ∈ S, t ∈ T. Cut capacity = sum of capacities of edges from S to T.

**Proof:**
- Max-flow ≤ min-cut: for any flow F and cut (S, T), F = net flow across the cut ≤ sum of edge capacities = cut capacity.
- Max-flow ≥ min-cut: If F is a maximum flow, the residual graph has no augmenting path. The set S = {nodes reachable from s in residual} is a cut with cut capacity = F.

Specifically, edges from S to T are saturated (residual = 0), and edges from T to S have zero flow (else there'd be a path s → ... → in T → in S, contradicting saturation). So flow value = sum of S→T capacities = cut capacity.

**Ford-Fulkerson Method:** Repeatedly find an augmenting path in the residual graph; augment flow by the bottleneck capacity. Terminates when no augmenting path exists.

**Complexity:** Depends on path-finding strategy. With BFS (Edmonds-Karp), O(V·E²). With Dinic's blocking flow, O(V²·E). With link-cut trees, O(VE log V).

**Edmonds-Karp:** Augment along shortest (BFS) paths. Each path increases distance to t by ≥ 1 each "round," giving O(VE) phases × O(E) per phase = O(VE²).

**Dinic's Algorithm:** Compute level graph (BFS from source). Find blocking flow (multiple augmenting paths simultaneously) in level graph. Repeat with new level graph. O(V²·E).

**Push-Relabel (Goldberg-Tarjan):** Maintain a "preflow" (allows excess at intermediate nodes). Push excess to neighbors with lower height; relabel to escape stuck states. O(V²·E) standard, O(V³) FIFO variant.

## Bipartite Matching

A bipartite graph has two disjoint vertex sets U, V with edges only between them. A matching is a set of edges with no shared endpoints.

**König's Theorem:** In bipartite graphs, the maximum matching size equals the minimum vertex cover size.

**Reduction to max-flow:** Create source s, connect to all of U with capacity 1. Connect each U vertex to V via existing edges (capacity 1). Connect each V vertex to sink t with capacity 1. Max flow = max matching.

**Hopcroft-Karp Algorithm (1973):** Repeatedly find a set of vertex-disjoint augmenting paths. Each phase increases path length by ≥ 1. O(E · √V).

**Application: assignment problems, stable marriage, scheduling.**

**Hungarian Algorithm:** Weighted bipartite matching (assignment problem). O(V³).

## Convex Hull

Given a set of points, find the smallest convex polygon containing all of them.

**Graham Scan (1972):**
1. Find the lowest point P (break ties by leftmost). It is on the hull.
2. Sort other points by polar angle around P.
3. Scan: for each point, if the last two hull points and the new point form a right turn, pop the last hull point. Push the new point.

O(n log n) due to sorting. Linear scan is O(n).

**Andrew's Monotone Chain:**
1. Sort by x (break ties by y).
2. Build lower hull: scan points left to right, maintaining counterclockwise turns.
3. Build upper hull: scan right to left.
4. Concatenate.

O(n log n). Cleaner than Graham scan; standard in competitive programming.

**Rotating Calipers:** A pair of parallel lines tangent to the hull. Rotate them; one line stays on a vertex while the other rotates to the next vertex. Used for:
- Diameter of hull (max distance between two hull points): O(n).
- Width: O(n).
- Closest pair on hull: O(n).

## Sweep Line Theory

A sweep-line algorithm processes events in sorted order along one axis. A data structure tracks the "active set" of objects intersecting the current sweep line.

**Pattern:**
1. Represent objects as events (point events, start events, end events).
2. Sort events by x-coordinate.
3. Maintain active data structure (set, priority queue, segment tree).
4. Process events in order; at each event, update active set and answer queries.

**Example: line segment intersection (Bentley-Ottmann).**
- Events: segment start, segment end, intersection.
- Active set: segments currently crossing the sweep line, ordered by y.
- Process events: insert/remove segments; on insert, check intersection with neighbors; on intersection, swap order.

O((n + k) log n) for n segments and k intersections.

**Example: skyline problem.** Given buildings (xleft, xright, height), output the skyline outline.
- Events: building start (height up), building end (height down).
- Active set: heights of buildings currently active. Use a max-heap.
- Process events: update active set; if max height changes, output a point.

O(n log n).

**Example: closest pair of points.**
- Sort by x. Sweep line sweeps right.
- Active set: points within x-distance d of sweep position, ordered by y.
- For each new point, check active points within y-distance d.

O(n log n).

## Coordinate Compression

When values span a large range but only n distinct values appear, compress to [0..n-1].

**Algorithm:**
1. Collect all values into a list.
2. Sort and deduplicate.
3. Map each original value to its index in the sorted list.

**Use cases:**
- Segment trees over coordinates: avoids 10^9 size; only n distinct values matter.
- Offline queries: process all queries before sorting.
- Counting inversions: standard reduction.

**Online compression:** Use a balanced BST or hash map for incremental compression.

## Reservoir Sampling Proof

Sample k items from a stream of unknown length N.

**Algorithm:**
- Initialize reservoir with first k items.
- For each item i > k: with probability k/i, replace a random reservoir item with item i.

**Theorem:** After processing all N items, every item is in the reservoir with probability k/N.

**Proof by induction:**
- Base (i = k): items 1..k are all in reservoir. Each has probability k/k = 1 = k/k. (Trivially.)
- Inductive step: suppose after step i-1, each of items 1..i-1 has probability k/(i-1).

At step i:
- Item i is added with probability k/i.
- Existing items: let item j (1 ≤ j ≤ i-1) be in the reservoir with probability k/(i-1).

P(j in reservoir after step i) = P(j in reservoir after step i-1) × P(j survives step i).

P(j survives step i) = P(item i not added) + P(item i added but j not displaced).
- P(item i not added) = 1 - k/i.
- P(item i added) = k/i. Given added, displaced = 1/k. P(j not displaced) = 1 - 1/k.

P(j survives) = (1 - k/i) + (k/i)(1 - 1/k) = 1 - k/i + k/i - 1/i = 1 - 1/i = (i-1)/i.

P(j in reservoir after step i) = k/(i-1) × (i-1)/i = k/i.

By induction, after step i, every item from 1 to i has probability k/i of being in the reservoir.

At step N: probability k/N. QED.

## Quickselect Average O(n)

Quickselect finds the k-th smallest element in expected linear time.

**Algorithm:** Pick a pivot. Partition. If k is in the left, recurse left; if in the right, recurse right; if at pivot, return.

**Expected complexity:** O(n).

**Proof (random pivot):** Let T(n) = expected work on input of size n.

T(n) = n + (1/n) Σ_{i=0}^{n-1} max(T(i), T(n-1-i)).

The expectation: with probability 1/n, the pivot is the i-th smallest. We recurse on the larger side, which has size max(i, n-1-i).

Approximation: T(n) ≤ n + (2/n) Σ_{i=n/2}^{n-1} T(i). Bounding the sum: T(n) ≤ n + 2/n × (n/2) × T(3n/4). So T(n) ≤ n + T(3n/4).

This is a recurrence T(n) = T(3n/4) + n, which solves to T(n) = O(n) by repeated substitution: n + 3n/4 + 9n/16 + ... = 4n.

**Worst case:** O(n²) with bad pivot (always picks min or max). Random pivots make this exponentially unlikely.

**Median-of-medians:** Deterministic O(n). Pick the median of medians (groups of 5) as pivot. Guarantees pivot is in [3n/10, 7n/10] (roughly). Recurrence: T(n) = T(n/5) + T(7n/10) + n. By Akra-Bazzi, T(n) = O(n).

## Decision-Tree Lower Bound for Sorting

Any comparison-based sorting algorithm requires Ω(n log n) comparisons.

**Decision Tree Model:** Each internal node compares two elements; the binary outcome determines the next node. Leaves represent permutations.

**Theorem:** Any comparison sort requires at least log₂(n!) ≈ n log n - n + O(log n) comparisons in the worst case.

**Proof:** A binary tree with L leaves has depth ≥ log₂(L). The decision tree must have at least n! leaves (one per permutation). So depth ≥ log₂(n!) = Θ(n log n) by Stirling.

**Stirling's approximation:** log₂(n!) = n log₂ n - n log₂ e + O(log n) ≈ n log₂ n.

The bound is tight: merge sort and heapsort achieve O(n log n).

**Caveats:**
- Applies only to comparison-based sorts.
- Counting sort, radix sort, bucket sort can be O(n + k) for integer-key sorts.
- Average-case bounds are also Ω(n log n) (information-theoretic lower bound on identifying the permutation).

## Final Reflections

These patterns are not arbitrary recipes. Each captures a structural property of the problem that allows certain operations to compose efficiently. Sliding window leverages monotonicity of a sweep predicate. Two pointers exploits sorted-array structure. DP requires optimal substructure. Greedy requires matroid or exchange-argument structure. Network flow exploits the duality of cuts and flows.

When a problem doesn't fit one of these patterns, the question becomes: what structure does it have? Sometimes the answer is "none — use brute force or approximation." Sometimes the answer is a more obscure pattern (segment trees, polynomial time approximation, randomized methods). The art is recognizing.

The patterns covered here represent the canonical toolkit. Mastering them — not as memorized templates but as understood structures — turns most "coding problems" into pattern recognition followed by adaptation. The proofs are not academic exercises; they are the source of confidence that the algorithm is correct.

## Segment Trees and Range Queries

Segment trees support range queries and updates in O(log n) time on arrays.

**Structure:** A binary tree where each node represents a range. Leaves are individual elements; internal nodes aggregate their children.

**Operations:**
- Update(i, v): set A[i] = v. O(log n).
- Query(l, r): aggregate A[l..r]. O(log n).

**Example aggregations:** sum, min, max, gcd, xor.

**Implementation:** Array-based, size 4n. Node i has children 2i and 2i+1.

**Lazy propagation:** Range updates (e.g., add v to A[l..r]) require lazy tags. Mark a node as "needs to apply update" without recursively updating children. Push down on subsequent queries.

**Use cases:**
- Range sum / max / min queries.
- Range updates with point queries.
- Range updates with range queries (lazy propagation).
- Persistence: keep all historical versions.

**Variants:**
- Fenwick tree (Binary Indexed Tree, BIT): simpler, O(log n), but limited to certain aggregations.
- Sparse table: O(1) queries for idempotent operations (like min, max, gcd) on static arrays.
- Wavelet tree: support k-th smallest in range queries.

## Heavy-Light Decomposition

For trees, decompose into paths to support range queries on tree paths.

**Idea:** Each non-leaf node has one "heavy" child (subtree of largest size) and the others are "light." Heavy edges form chains.

**Property:** Path from any node to root crosses O(log n) light edges (since each light edge halves the subtree size).

**Use:** Path queries (sum / min / max along path from u to v) reduce to O(log n) chain segments. Each chain is queried via segment tree in O(log n).

Total: O(log² n) per query.

**Use cases:**
- LCA + path aggregation.
- Subtree sum + path sum.
- Tree dynamic queries.

## Persistent Data Structures

A persistent data structure preserves the previous version when modified. The new version coexists with the old.

**Path copying:** Copy only the path from root to modified node. New version shares unchanged subtrees.

**Persistent stack:** Push/pop returns a new stack; old version intact. O(1) per operation.

**Persistent segment tree:** O(log n) per update, with O(log n) extra memory.

**Use cases:**
- Time-travel queries: state at any past time.
- Functional programming: immutable data.
- Offline algorithms: process queries in order, then go back.

## Suffix Arrays and Suffix Trees

For string problems involving multiple substring queries.

**Suffix array:** Array of starting indices of all suffixes, sorted lexicographically. O(n log n) construction; O(n) with SA-IS algorithm.

**LCP (Longest Common Prefix) array:** LCP[i] = length of LCP between suffixes SA[i] and SA[i-1]. Constructed in O(n) via Kasai's algorithm.

**Suffix tree:** Trie of all suffixes, with edges labeled by substrings. O(n) construction via Ukkonen's algorithm.

**Use cases:**
- Substring search: O(m log n) per query (binary search on suffix array).
- Number of distinct substrings: n(n+1)/2 - sum(LCP).
- Longest repeated substring: max LCP.
- Longest common substring of multiple strings.

## Mo's Algorithm

Offline algorithm for range queries. Sort queries cleverly to share computation.

**Algorithm:** Sort queries by (block of left endpoint, right endpoint). Process in order. Move pointers (left, right) to match each query, applying delta updates.

**Complexity:** O((n + q) · √n) for n elements and q queries.

**Use cases:**
- Range queries with offline batch.
- Distinct elements in range.
- Range queries on trees (with Euler tour + Mo's).

## Sqrt Decomposition

Divide array into √n blocks of size √n. Maintain block-level aggregates.

**Update(i, v):** O(1) — update element and block.
**Query(l, r):** Process completely-covered blocks (O(√n)) plus partial blocks at endpoints (O(√n)).

**Use cases:**
- Range updates and queries when segment tree is overkill.
- Online queries with simpler implementation.
- Stress testing simpler than segment trees.

## Number Theory Algorithms

**Sieve of Eratosthenes:** Find all primes up to N in O(N log log N).

**Euler's Totient Function φ(n):** Count integers coprime to n. Compute via sieve in O(N log log N).

**Modular Exponentiation:** a^b mod m in O(log b).

**Modular Inverse:** Via extended Euclidean or Fermat's little theorem.

**Chinese Remainder Theorem:** Solve system of congruences.

**Miller-Rabin:** Primality test, O(k log³ n) for k iterations. Probabilistically correct.

**Pollard's Rho:** Factor n in O(n^(1/4)) expected time.

These are workhorses for cryptography, combinatorics, and competitive programming.

## Geometry Algorithms

**Cross Product:** Determines turn direction (CW, CCW, collinear).

**Polygon Area:** Shoelace formula, O(n).

**Point in Polygon:** Ray casting, O(n).

**Line Intersection:** Solve linear system, O(1).

**Closest Pair:** Divide and conquer, O(n log n).

**Half-Plane Intersection:** Sort by angle, monotone stack, O(n log n).

**Voronoi Diagram:** Fortune's algorithm, O(n log n).

**Delaunay Triangulation:** Dual of Voronoi.

**Convex Hull Trick:** Maintain upper/lower envelope of lines for DP optimization.

**Li Chao Tree:** Segment tree variant for line queries.

## Game Theory

**Minimax:** Recursive evaluation of game trees. Adversary minimizes; protagonist maximizes.

**Alpha-Beta Pruning:** Cut branches that can't affect the outcome. Reduces from O(b^d) to O(b^(d/2)) in best case.

**Sprague-Grundy Theorem:** For impartial games (both players have same moves), Grundy number determines who wins. XOR of Grundy numbers across independent subgames gives the overall Grundy number.

**Nim:** Classic impartial game. P (first player loses) iff XOR of pile sizes is 0.

**Game DP:** Define state, transitions, base cases. Solve with memoization.

## String Algorithms Beyond KMP

**Suffix Automaton:** Compact representation of all substrings. O(n) construction.

**Suffix Tree:** Generalized suffix automaton. O(n) construction (Ukkonen's).

**FM-Index:** Compressed full-text index based on Burrows-Wheeler Transform.

**Hash-based Matching:** Polynomial rolling hash. O(n+m) expected.

**Rabin-Karp:** Hash-based string search. Expected O(n+m), worst-case O(nm).

**Boyer-Moore:** Heuristic string search. Sublinear in best case.

## Graph Coloring

**Greedy Coloring:** Order vertices, color each with smallest available. May use more colors than chromatic number.

**Welsh-Powell:** Sort by degree, color greedily.

**DSatur:** Saturation-based heuristic.

**Backtracking:** Exact, exponential in worst case.

**4-Color Theorem:** Any planar graph is 4-colorable. Proven by computer (Appel-Haken 1976).

**Brooks' Theorem:** A connected non-complete graph with max degree Δ has chromatic number ≤ Δ (except odd cycles).

## Maximum Bipartite Matching Beyond Hopcroft-Karp

**Hungarian Algorithm:** Weighted bipartite matching (assignment problem). O(V³).

**Min-Cost Max-Flow:** General. SPFA-based, O(V · E²) typical.

**Kuhn's Algorithm:** O(V · E) bipartite matching via DFS augmenting paths. Simpler than Hopcroft-Karp.

## Dynamic Connectivity

Maintain connected components under edge insertions and deletions.

**Online insertions only:** Union-Find, O(α(n)) per op.

**Online deletions only:** Process deletions offline; reverse for insertions.

**Online both:** Holm-Lichtenberg-Thorup: O(log² n) amortized per op.

**Offline both:** Use link-cut trees or segment tree with offline processing.

## Link-Cut Trees

Dynamic trees supporting:
- Link two trees.
- Cut an edge.
- Path queries (sum, min, max along path).

**Implementation:** Splay trees over preferred paths.

**Complexity:** O(log n) amortized per operation.

**Use cases:**
- Dynamic graph algorithms.
- Online flow algorithms (Sleator-Tarjan).
- Online tree-DP.

## Centroid Decomposition

Decompose tree into centroids recursively. Each subtree has half the nodes.

**Property:** Tree of centroids has depth O(log n).

**Use cases:**
- Tree path queries with offline batch.
- Compute distances on tree.
- Solve "path problems" (number of paths with property).

## Euler Tour Technique

Represent tree as a flat array via DFS preorder/postorder. Subtree corresponds to a contiguous range.

**Use cases:**
- Subtree queries reduce to range queries.
- Apply segment tree / Fenwick tree to tree.

## DP on DAG

Tree DP generalizes to DAG via topological sort. Each node's dp depends on predecessors.

**Reachability:** Boolean OR of predecessors.

**Number of paths:** Sum over predecessors.

**Longest path:** Max of predecessors + 1.

**Bitmask DP on DAG:** Combine bitmask state with topological order.

## DP Optimization Techniques

**Convex Hull Trick:** When DP transition is dp[i] = min over j of (a_j · x_i + b_j), maintain lower envelope of lines.

**Divide and Conquer DP:** When transition has "monotonicity" property, divide search space.

**Knuth's Optimization:** When opt[i][j] is monotonic, reduce O(n³) to O(n²).

**SOS (Sum over Subsets) DP:** Compute sums over all subsets of a bitmask in O(2^n · n).

These advanced DP techniques are essential for competitive programming and certain optimization problems.

## Bit Manipulation Patterns

**Counting set bits:** popcount, Brian Kernighan's algorithm.

**Parity:** XOR all bits.

**Lowest set bit:** x & -x.

**Highest set bit:** Log via bit-scan-reverse.

**Subset enumeration:** Iterate through subsets of a mask.

**Bit-DP:** Use bitmask as DP state (covered above).

**XOR tricks:**
- a ^ b ^ b = a (cancellation).
- Find single non-duplicate: XOR all elements.
- Find missing number: XOR with 1..n.

## Common Algorithm Selection Heuristics

**Sort then process:** When order matters or eliminates duplicates.

**Hash for fast lookup:** When membership / counting is needed.

**Heap for top-k:** Min-heap or max-heap.

**Trie for prefix queries:** Strings with shared prefixes.

**Segment tree for range:** Aggregate over ranges.

**Stack for matching:** Parentheses, monotonic.

**Queue for BFS:** Level-by-level processing.

**Two pointers / sliding window for sequences:** Sorted or condition-based.

**Binary search for monotone:** Predicate over solution space.

**DP for optimal substructure:** Combine sub-results.

**Greedy for matroid:** Provable correctness.

## Practical Wisdom

**Profile before optimizing:** Asymptotic complexity matters at scale; constants matter at small n. Profile real workloads.

**Choose simplest algorithm meeting requirements:** Maintainability matters more than 2x speed.

**Reuse libraries:** std::sort, std::priority_queue, std::set. Hand-rolled is usually slower and buggier.

**Test with extreme cases:** Empty input, single element, very large input.

**Verify with brute force:** Implement O(n²) brute force; compare on small inputs.

**Modular code:** Decompose into helper functions. Easier to debug.

**Memory layout:** Cache locality often matters more than O().

**Exit early when possible:** Short-circuit evaluation, early-exit conditions.

## Building Algorithm Intuition

To master algorithm patterns:

1. **Solve, then study:** Don't memorize templates without understanding. Solve problems first, then read solutions.

2. **Read others' code:** Editorials, contest solutions, open-source implementations.

3. **Implement classics:** Sort algorithms, BFS/DFS, Dijkstra, KMP, segment tree. Implementing solidifies understanding.

4. **Practice variants:** Same pattern with different constraints. Builds adaptability.

5. **Time yourself:** Competitive programming and time-bounded interviews.

6. **Stay curious:** Read papers on new algorithms (e.g., recent matrix multiplication results, new sorting bounds).

7. **Think about lower bounds:** Why can't we do better? What's the information content?

The patterns are tools. Knowing the tools is necessary but not sufficient. Knowing when to use which tool — and adapting them to specific constraints — is the art.

## References

- Cormen, T.H., Leiserson, C.E., Rivest, R.L., Stein, C. "Introduction to Algorithms" (CLRS), 4th edition. MIT Press.
- Sedgewick, R., Wayne, K. "Algorithms," 4th edition. Addison-Wesley.
- Skiena, S. "The Algorithm Design Manual," 3rd edition. Springer.
- Knuth, D.E. "The Art of Computer Programming," Volume 3: Sorting and Searching.
- Tarjan, R.E. (1972). "Depth-First Search and Linear Graph Algorithms." SIAM J. Comput.
- Kosaraju, S.R. (1978). "Strongly connected components." Unpublished manuscript.
- Knuth, D.E., Morris, J.H., Pratt, V.R. (1977). "Fast Pattern Matching in Strings." SIAM J. Comput.
- Aho, A.V., Corasick, M.J. (1975). "Efficient string matching: an aid to bibliographic search." CACM.
- Manacher, G. (1975). "A new linear-time on-line algorithm for finding the smallest initial palindrome of a string." JACM.
- Bellman, R. (1957). "Dynamic Programming." Princeton University Press.
- Ford, L.R., Fulkerson, D.R. (1956). "Maximal flow through a network." Canadian Journal of Mathematics.
- Edmonds, J., Karp, R.M. (1972). "Theoretical improvements in algorithmic efficiency for network flow problems." JACM.
- Dinic, E.A. (1970). "Algorithm for solution of a problem of maximum flow in networks with power estimation."
- Goldberg, A.V., Tarjan, R.E. (1988). "A new approach to the maximum flow problem." JACM.
- Hopcroft, J.E., Karp, R.M. (1973). "An n^5/2 algorithm for maximum matchings in bipartite graphs." SIAM J. Comput.
- Edmonds, J. (1971). "Matroids and the greedy algorithm." Mathematical Programming.
- Tarjan, R.E. (1975). "Efficiency of a Good But Not Linear Set Union Algorithm." JACM.
- Vitter, J.S. (1985). "Random sampling with a reservoir." TOMS.
- Bentley, J.L., Ottmann, T.A. (1979). "Algorithms for reporting and counting geometric intersections." IEEE Trans. Comput.
- Graham, R.L. (1972). "An efficient algorithm for determining the convex hull of a finite planar set." Information Processing Letters.
- Hart, P.E., Nilsson, N.J., Raphael, B. (1968). "A Formal Basis for the Heuristic Determination of Minimum Cost Paths." IEEE Transactions on Systems Science and Cybernetics.
- Akra, M., Bazzi, L. (1998). "On the solution of linear recurrence equations." Computational Optimization and Applications.
- Erickson, J. "Algorithms." Free online textbook. https://jeffe.cs.illinois.edu/teaching/algorithms/
