# Valid Parentheses (Stack / Strings)

Given a string containing just the characters '(', ')', '{', '}', '[' and ']', determine if the input string is valid.

## Problem

Given a string `s` containing only the characters `(`, `)`, `{`, `}`, `[` and `]`, determine if the input string is valid.

A string is **valid** if:

1. Open brackets are closed by the same type of brackets.
2. Open brackets are closed in the correct order.
3. Every close bracket has a corresponding open bracket of the same type.

**Constraints:**

- `1 <= s.length <= 10^4`
- `s` consists of parentheses only: `(){}[]`

**Examples:**

```
Input:  "()"
Output: true

Input:  "()[]{}"
Output: true

Input:  "(]"
Output: false

Input:  "([)]"
Output: false

Input:  "{[]}"
Output: true
```

## Hints

1. Use a stack to track opening brackets as you scan left to right.
2. When you encounter an opening bracket, push it onto the stack.
3. When you encounter a closing bracket, pop the stack and check that the popped bracket matches.
4. If the stack is empty when you try to pop, or the brackets don't match, the string is invalid.
5. After processing all characters, the stack must be empty for the string to be valid.

## Solution -- Go

```go
package main

import "fmt"

func isValid(s string) bool {
	stack := []rune{}
	pairs := map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}

	for _, c := range s {
		switch c {
		case '(', '[', '{':
			stack = append(stack, c)
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != pairs[c] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}

	return len(stack) == 0
}

func main() {
	tests := []struct {
		input    string
		expected bool
	}{
		{"()", true},
		{"()[]{}", true},
		{"(]", false},
		{"([)]", false},
		{"{[]}", true},
		{"", true},
		{"(", false},
		{")", false},
		{"((()))", true},
		{"{[()]}", true},
	}

	for _, t := range tests {
		result := isValid(t.input)
		if result != t.expected {
			panic(fmt.Sprintf("isValid(%q) = %v, want %v", t.input, result, t.expected))
		}
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
class Solution:
    def is_valid(self, s: str) -> bool:
        stack: list[str] = []
        pairs = {')': '(', ']': '[', '}': '{'}

        for c in s:
            if c in '([{':
                stack.append(c)
            elif c in ')]}':
                if not stack or stack[-1] != pairs[c]:
                    return False
                stack.pop()

        return len(stack) == 0


if __name__ == "__main__":
    sol = Solution()

    assert sol.is_valid("()") is True, "Test 1 failed"
    assert sol.is_valid("()[]{}") is True, "Test 2 failed"
    assert sol.is_valid("(]") is False, "Test 3 failed"
    assert sol.is_valid("([)]") is False, "Test 4 failed"
    assert sol.is_valid("{[]}") is True, "Test 5 failed"
    assert sol.is_valid("") is True, "Test 6 failed"
    assert sol.is_valid("(") is False, "Test 7 failed"
    assert sol.is_valid(")") is False, "Test 8 failed"
    assert sol.is_valid("((()))") is True, "Test 9 failed"
    assert sol.is_valid("{[()]}") is True, "Test 10 failed"

    print("All tests passed!")
```

## Solution -- Rust

```rust
struct Solution;

impl Solution {
    fn is_valid(s: String) -> bool {
        let mut stack: Vec<char> = Vec::new();

        for c in s.chars() {
            match c {
                '(' | '[' | '{' => stack.push(c),
                ')' => {
                    if stack.pop() != Some('(') {
                        return false;
                    }
                }
                ']' => {
                    if stack.pop() != Some('[') {
                        return false;
                    }
                }
                '}' => {
                    if stack.pop() != Some('{') {
                        return false;
                    }
                }
                _ => {}
            }
        }

        stack.is_empty()
    }
}

fn main() {
    assert!(Solution::is_valid("()".to_string()));
    assert!(Solution::is_valid("()[]{}".to_string()));
    assert!(!Solution::is_valid("(]".to_string()));
    assert!(!Solution::is_valid("([)]".to_string()));
    assert!(Solution::is_valid("{[]}".to_string()));
    assert!(Solution::is_valid("".to_string()));
    assert!(!Solution::is_valid("(".to_string()));
    assert!(!Solution::is_valid(")".to_string()));
    assert!(Solution::is_valid("((()))".to_string()));
    assert!(Solution::is_valid("{[()]}".to_string()));

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function isValid(s: string): boolean {
    const stack: string[] = [];
    const pairs: Record<string, string> = {
        ")": "(",
        "]": "[",
        "}": "{",
    };

    for (const c of s) {
        if (c === "(" || c === "[" || c === "{") {
            stack.push(c);
        } else if (c === ")" || c === "]" || c === "}") {
            if (stack.length === 0 || stack[stack.length - 1] !== pairs[c]) {
                return false;
            }
            stack.pop();
        }
    }

    return stack.length === 0;
}

// Tests
console.assert(isValid("()") === true, "Test 1 failed");
console.assert(isValid("()[]{}") === true, "Test 2 failed");
console.assert(isValid("(]") === false, "Test 3 failed");
console.assert(isValid("([)]") === false, "Test 4 failed");
console.assert(isValid("{[]}") === true, "Test 5 failed");
console.assert(isValid("") === true, "Test 6 failed");
console.assert(isValid("(") === false, "Test 7 failed");
console.assert(isValid(")") === false, "Test 8 failed");
console.assert(isValid("((()))") === true, "Test 9 failed");
console.assert(isValid("{[()]}") === true, "Test 10 failed");
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | $O(n)$ where $n$ = length of the string |
| Space | $O(n)$ worst case when all characters are opening brackets |

## Tips

- A string of odd length can never be valid -- you can check this as an early return.
- The stack approach naturally handles nested brackets like `{[()]}` because it enforces LIFO ordering.
- Common mistake: forgetting to check that the stack is empty at the end. The string `"((("` has no mismatches but is still invalid.
- Common mistake: forgetting to check for an empty stack before popping. The string `")"` would cause an underflow.
- In Go, a slice (`[]rune`) works as a stack: `append` to push, reslice to pop.
- In Rust, `stack.pop()` returns `Option<char>`, so you can compare directly with `Some('(')`.

## See Also

- [Generate Parentheses](generate-parentheses.md) -- generate all valid combinations
- [Longest Valid Parentheses](longest-valid-parentheses.md) -- find longest valid substring
- [Min Remove to Make Valid Parentheses](min-remove-to-make-valid-parentheses.md) -- remove minimum brackets

## References

- LeetCode 20: Valid Parentheses -- https://leetcode.com/problems/valid-parentheses/
- NeetCode explanation -- https://neetcode.io/solutions/valid-parentheses
