# Single Number III (Bit Manipulation)

Find the two elements that appear exactly once in an array where every other element appears exactly twice, using O(n) time and O(1) space.

## Problem

Given an integer array `nums` where every element appears exactly twice except for two
elements which appear exactly once, find those two unique elements.

**Constraints:**

- `2 <= len(nums) <= 3 * 10^4`
- `-2^31 <= nums[i] <= 2^31 - 1`
- Exactly two elements appear once; all others appear twice.
- Must use O(n) time and O(1) space.

**Examples:**

```
[1, 2, 1, 3, 2, 5]      => [3, 5]     (3 and 5 appear once)
[-1, 0]                  => [-1, 0]    (negative number case)
[0, 1]                   => [0, 1]     (zero as unique element)
[2, 2, 3, 3, 5, 7]      => [5, 7]     (uniques at end)
[1, 1, 2, 2, 3, 3, 4, 5] => [4, 5]    (longer array)
[100, 200, 100, 200, 1, 999] => [1, 999]  (large gap between uniques)
```

## Related Problems

| Problem | Key Difference | Approach |
|---------|---------------|----------|
| Single Number I | 1 unique, rest appear 2x | XOR all => answer |
| Single Number II | 1 unique, rest appear 3x | Bit counting mod 3 |
| **Single Number III** | **2 unique, rest appear 2x** | **XOR + partition** |
| Missing Number | 1 missing from [0..n] | XOR with indices |
| Find Duplicate | 1 duplicate in [1..n] | Floyd's cycle detection |

## Walkthrough

The algorithm has three phases. Consider `nums = [1, 2, 1, 3, 2, 5]`:

**Phase 1 -- XOR all:** `1 ^ 2 ^ 1 ^ 3 ^ 2 ^ 5 = 6` (binary `110`).
Since `1 ^ 1 = 0` and `2 ^ 2 = 0`, only `3 ^ 5 = 6` survives.

**Phase 2 -- Find distinguishing bit:** `6 & -6 = 6 & (..1010) = 010 = 2`.
Bit 1 is set in `3` (`011`) but not in `5` (`101`). This bit separates the two unknowns.

**Phase 3 -- Partition and XOR:**
- Group A (bit 1 set): `{2, 3, 2}`. XOR: `2 ^ 3 ^ 2 = 3`.
- Group B (bit 1 clear): `{1, 1, 5}`. XOR: `1 ^ 1 ^ 5 = 5`.
- Result: `[3, 5]`.

Each paired number has both copies in the same group (same bit value), so they cancel.

## Hints

- **Step 1:** XOR all numbers. Pairs cancel out, leaving `a ^ b` (the XOR of the two
  unique numbers).
- **Step 2:** Find any bit where `a` and `b` differ. The lowest set bit of `a ^ b` works:
  `diff = xor & -xor` isolates the lowest set bit.
- **Step 3:** Partition all numbers into two groups based on whether that bit is set.
  XOR each group independently -- paired numbers cancel within their group, leaving
  `a` in one group and `b` in the other.
- **Edge case -- negative numbers:** Two's complement ensures XOR and bit extraction work
  correctly. The lowest set bit trick `x & -x` is valid for all nonzero integers.
- **Edge case -- zero as a unique number:** If one of the unique numbers is 0, the XOR of
  all numbers equals the other unique number. The lowest set bit of that value still
  correctly separates them (0 has no bits set, so it always lands in the "bit not set" group).

## Solution -- Go

```go
func singleNumber(nums []int) [2]int {
	// Step 1: XOR all numbers -- pairs cancel, leaving a ^ b.
	//
	// XOR properties used here:
	//   x ^ x = 0      (self-inverse: paired elements cancel)
	//   x ^ 0 = x      (identity: zeros drop out)
	//   associative and commutative (order doesn't matter)
	//
	// After XOR-ing the entire array, all paired elements vanish,
	// leaving only a ^ b where a, b are the two unique elements.
	xorAll := 0
	for _, n := range nums {
		xorAll ^= n
	}

	// Step 2: Find lowest set bit where a and b differ.
	//
	// In two's complement: -x = ~x + 1.
	// The expression x & -x isolates the lowest set bit.
	// Example: 6 (110) & -6 (...010) = 2 (010).
	//
	// This bit position is guaranteed to differ between a and b,
	// because xorAll = a ^ b and a bit is 1 in XOR only when the
	// corresponding bits of the two operands differ.
	diffBit := xorAll & (-xorAll)

	// Step 3: Partition all numbers into two groups by that bit,
	// then XOR within each group to isolate a and b.
	//
	// Group A: numbers where the distinguishing bit is set (1)
	// Group B: numbers where the distinguishing bit is clear (0)
	//
	// Both copies of every paired number share the same bit value,
	// so they land in the same group and cancel via XOR.
	// a and b land in opposite groups (by construction).
	var a, b int
	for _, n := range nums {
		if n&diffBit != 0 {
			a ^= n
		} else {
			b ^= n
		}
	}

	// Sort the two-element result for deterministic output.
	result := [2]int{a, b}
	if result[0] > result[1] {
		result[0], result[1] = result[1], result[0]
	}
	return result
}
```

## Solution -- Python

```python
from typing import List


class Solution:
    def single_number(self, nums: List[int]) -> List[int]:
        # Step 1: XOR all numbers => xor_all = a ^ b
        # Python integers have arbitrary precision, so no overflow concerns.
        xor_all = 0
        for num in nums:
            xor_all ^= num

        # Step 2: Find a distinguishing bit (lowest set bit).
        # Python's integers are arbitrary-precision, but the & -x trick
        # still works because Python uses a conceptual infinite sign
        # extension for negative numbers.
        diff_bit = xor_all & (-xor_all)

        # Step 3: Partition into two groups and XOR each.
        # Every paired element goes to the same group (same bit value).
        # The two unique elements go to different groups.
        a, b = 0, 0
        for num in nums:
            if num & diff_bit:
                a ^= num
            else:
                b ^= num

        return sorted([a, b])
```

## Solution -- Rust

```rust
fn single_number(nums: &[i32]) -> [i32; 2] {
    // Step 1: XOR all numbers
    let xor_all = nums.iter().fold(0i32, |acc, &n| acc ^ n);

    // Step 2: Find lowest set bit (differs between a and b)
    // Use wrapping_neg to handle potential overflow with i32::MIN
    let diff_bit = xor_all & xor_all.wrapping_neg();

    // Step 3: Partition and XOR each group
    let mut a = 0i32;
    let mut b = 0i32;
    for &n in nums {
        if n & diff_bit != 0 {
            a ^= n;
        } else {
            b ^= n;
        }
    }

    let mut result = [a, b];
    result.sort();
    result
}
```

## Solution -- TypeScript

```typescript
function singleNumber(nums: number[]): [number, number] {
    // Step 1: XOR all numbers
    let xorAll = 0;
    for (const n of nums) {
        xorAll ^= n;
    }

    // Step 2: Find lowest set bit (differs between a and b)
    // For JS bitwise ops work on 32-bit ints, so this is safe
    const diffBit = xorAll & -xorAll;

    // Step 3: Partition and XOR each group
    let a = 0;
    let b = 0;
    for (const n of nums) {
        if (n & diffBit) {
            a ^= n;
        } else {
            b ^= n;
        }
    }

    return a < b ? [a, b] : [b, a];
}
```

## Bit Manipulation Reference

The core operations used in this problem:

```
XOR truth table:        Lowest set bit extraction:
  0 ^ 0 = 0               x = 12 = 1100
  0 ^ 1 = 1              -x =      0100
  1 ^ 0 = 1             x&-x =     0100 = 4
  1 ^ 1 = 0

Self-inverse property:  Partition example (diffBit = 2 = 010):
  a ^ a = 0               3  = 011  => bit set     => Group A
  a ^ 0 = a               5  = 101  => bit clear   => Group B
                           2  = 010  => bit set     => Group A
                           2  = 010  => bit set     => Group A
                           1  = 001  => bit clear   => Group B
                           1  = 001  => bit clear   => Group B

                           Group A: 3 ^ 2 ^ 2 = 3
                           Group B: 5 ^ 1 ^ 1 = 5
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(n) -- two passes over the array (one for XOR-all, one for partition) |
| Space | O(1) -- only four scalar variables (xorAll, diffBit, a, b) |
| Comparisons | 0 -- no element-to-element comparisons needed |

## Tips

- **Why `xor & -xor` works:** In two's complement, `-x = ~x + 1`. The lowest set bit
  of `x` is the only bit that survives the AND. For example, `0b1100 & 0b0100 = 0b0100`.
  This is a classic bit manipulation trick used in Fenwick trees and other algorithms.
- **Rust's `wrapping_neg`:** Negating `i32::MIN` overflows in Rust (debug mode panics).
  Using `wrapping_neg()` avoids this. The XOR result is unlikely to be `i32::MIN`, but
  defensive coding matters. In C/C++, the same overflow is undefined behavior.
- **JavaScript 32-bit limitation:** Bitwise operators in JS work on 32-bit signed integers.
  This means the approach is correct for values in `[-2^31, 2^31 - 1]` but not for
  arbitrary large numbers (which JS would represent as floats). Use `BigInt` for larger
  ranges if needed.
- **This extends Single Number I:** In that problem, XOR all numbers gives the single
  unique element directly. Here, the extra step is partitioning to separate the two
  unique values. Single Number II (one unique, others appear 3x) uses a different
  approach with bit counting modulo 3.
- **Why partitioning works:** Every paired number `p` has the same value of `p & diffBit`,
  so both copies go to the same group. The two unique numbers go to different groups
  (by construction, `diffBit` is a bit where they differ). Within each group, paired
  numbers cancel, leaving one unique number.
- **Any distinguishing bit works**, not just the lowest. The lowest set bit is simply
  the easiest to extract. You could also use the highest set bit via `Integer.highestOneBit`
  or bit shifting, but the lowest-set-bit trick is more concise.
- **Sorting the output** is important for deterministic results. Without sorting, the
  order depends on which unique number has the distinguishing bit set (group A vs group B),
  which varies by input. Sort the two-element result for stable output.
- **O(1) space is achieved** because only four scalar variables are used (`xorAll`,
  `diffBit`, `a`, `b`). No hash maps, sets, or auxiliary arrays are needed. This is
  the key advantage over the hash-map approach which uses O(n) space.
- **Comparison with hash map approach:** A hash map counting occurrences solves this
  in O(n) time and O(n) space. The XOR approach trades space for cleverness -- same
  time complexity but constant space. The hash map approach generalizes to k unique
  elements; the XOR approach is specific to k=1 or k=2.
- **Testing edge cases:** Always test with negative numbers, zeros, arrays of length 2
  (the minimum), and arrays where the two unique numbers are adjacent or far apart in
  value. The algorithm handles all these cases identically.

## See Also

- bit-manipulation
- xor-properties
- single-number
- single-number-ii
- bitwise-operations

## References

- [LeetCode 260 -- Single Number III](https://leetcode.com/problems/single-number-iii/)
- [LeetCode 136 -- Single Number I](https://leetcode.com/problems/single-number/)
- [LeetCode 137 -- Single Number II](https://leetcode.com/problems/single-number-ii/)
- [Two's Complement (Wikipedia)](https://en.wikipedia.org/wiki/Two%27s_complement)
- [XOR and its properties](https://en.wikipedia.org/wiki/Exclusive_or)
- [Bit Manipulation Tricks (Stanford)](https://graphics.stanford.edu/~seander/bithacks.html)
- [Fenwick Tree / BIT (uses x & -x)](https://en.wikipedia.org/wiki/Fenwick_tree)
