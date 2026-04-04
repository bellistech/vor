# The Type Theory of TypeScript — Structural Typing, Variance, and the Type Algebra

> *TypeScript's type system is a structural, gradually-typed, Turing-complete type-level language layered on top of JavaScript. It features union/intersection types (set-theoretic), conditional types (type-level if/else), mapped types (type-level map), and template literal types (type-level string manipulation).*

---

## 1. Structural Typing vs Nominal Typing

### Structural Subtyping Rule

TypeScript uses **structural typing**: type compatibility is determined by shape, not name.

$$A <: B \iff \forall p \in \text{props}(B), \ p \in \text{props}(A) \land \text{type}_A(p) <: \text{type}_B(p)$$

A is a subtype of B if A has all of B's properties with compatible types.

```typescript
interface Point2D { x: number; y: number }
interface Point3D { x: number; y: number; z: number }

// Point3D <: Point2D (has all properties of Point2D plus more)
let p2: Point2D = { x: 1, y: 2, z: 3 } as Point3D;  // OK
```

### Contrast with Nominal Typing

In Java/C#, two classes with identical fields are **not** interchangeable unless they share an inheritance relationship. In TypeScript, they are.

### Excess Property Checking

Fresh object literals get **stricter** checking — excess properties are errors:

```typescript
let p: Point2D = { x: 1, y: 2, z: 3 };  // ERROR: 'z' not in Point2D
let obj = { x: 1, y: 2, z: 3 };
let p2: Point2D = obj;                    // OK (not a fresh literal)
```

This is a **pragmatic** exception to structural typing — catches typos.

---

## 2. Union and Intersection Types — Set Theory

### Types as Sets

A type is the **set of all values** that inhabit it:

| Type | Set |
|:-----|:----|
| `number` | $\{..., -1, 0, 1, 1.5, ...\}$ |
| `string` | $\{"", "a", "ab", ...\}$ |
| `"hello"` | $\{"hello"\}$ (singleton set) |
| `never` | $\emptyset$ (empty set) |
| `unknown` | Universal set $U$ |

### Union = Set Union

$$A \mid B \cong A \cup B$$

A value of type `A | B` is in $A$ or in $B$ (or both).

### Intersection = Set Intersection

$$A \& B \cong A \cap B$$

A value of type `A & B` has all properties of $A$ and all properties of $B$.

### Algebraic Laws

| Law | Formula | Example |
|:----|:--------|:--------|
| Identity | $T \mid \text{never} = T$ | `string | never = string` |
| Annihilation | $T \mid \text{unknown} = \text{unknown}$ | `string | unknown = unknown` |
| Identity | $T \& \text{unknown} = T$ | `string & unknown = string` |
| Annihilation | $T \& \text{never} = \text{never}$ | `string & never = never` |
| Distributive | $A \& (B \mid C) = (A \& B) \mid (A \& C)$ | Intersection distributes over union |
| Idempotent | $A \mid A = A$ | `string | string = string` |

---

## 3. Variance

### Definition

Variance describes how subtyping of components affects subtyping of compound types.

| Variance | Rule | Notation |
|:---------|:-----|:---------|
| Covariant | If $A <: B$ then $F(A) <: F(B)$ | Same direction |
| Contravariant | If $A <: B$ then $F(B) <: F(A)$ | Reversed |
| Invariant | Neither direction | Exact match only |
| Bivariant | Both directions | TypeScript methods (unsound!) |

### Variance in Practice

```typescript
// Covariant: readonly arrays, Promise<T>, return types
type ReadonlyArray<T> // If Dog <: Animal, ReadonlyArray<Dog> <: ReadonlyArray<Animal>

// Contravariant: function parameter types (with strictFunctionTypes)
type Fn<T> = (x: T) => void
// If Dog <: Animal, Fn<Animal> <: Fn<Dog>   (reversed!)

// Invariant: mutable arrays, mutable properties
// Array<Dog> is NOT a subtype of Array<Animal> (could insert Cat via Animal ref)
```

### The Unsoundness

TypeScript intentionally has some unsound spots:

1. **Method parameter bivariance** (without `strictFunctionTypes`)
2. **Enum assignability** — enums are assignable to `number`
3. **`any`** — escapes all type checking

---

## 4. Conditional Types — Type-Level If/Else

### Syntax

$$T \text{ extends } U \ ? \ X : Y$$

If $T <: U$, evaluates to $X$; otherwise $Y$.

### Distribution over Unions

When $T$ is a **naked type parameter** and a union, conditional types **distribute**:

$$(A \mid B) \text{ extends } U \ ? \ X : Y = (A \text{ extends } U \ ? \ X : Y) \mid (B \text{ extends } U \ ? \ X : Y)$$

### Worked Example: `Extract` and `Exclude`

```typescript
type Exclude<T, U> = T extends U ? never : T;

type T = Exclude<"a" | "b" | "c", "a" | "c">;
// = ("a" extends "a"|"c" ? never : "a")     → never
// | ("b" extends "a"|"c" ? never : "b")     → "b"
// | ("c" extends "a"|"c" ? never : "c")     → never
// = never | "b" | never = "b"
```

### `infer` — Pattern Matching in Types

```typescript
type ReturnType<T> = T extends (...args: any[]) => infer R ? R : never;

type T = ReturnType<(x: number) => string>;  // string
```

`infer R` introduces a type variable `R` that the compiler solves for by unification.

---

## 5. Mapped Types — Type-Level Map

### Syntax

```typescript
type Mapped<T> = {
    [K in keyof T]: Transform<T[K]>
};
```

This iterates over all keys of `T` and applies a transformation to each value type.

### Built-in Mapped Types

| Utility | Definition | Effect |
|:--------|:-----------|:-------|
| `Partial<T>` | `{ [K in keyof T]?: T[K] }` | All properties optional |
| `Required<T>` | `{ [K in keyof T]-?: T[K] }` | All properties required |
| `Readonly<T>` | `{ readonly [K in keyof T]: T[K] }` | All properties readonly |
| `Pick<T, K>` | `{ [P in K]: T[P] }` | Subset of properties |
| `Record<K, V>` | `{ [P in K]: V }` | Homogeneous object type |

### Key Remapping (4.1+)

```typescript
type Getters<T> = {
    [K in keyof T as `get${Capitalize<string & K>}`]: () => T[K]
};

// { name: string, age: number } → { getName: () => string, getAge: () => number }
```

---

## 6. Template Literal Types — Type-Level String Operations

### Basic Syntax

```typescript
type Greeting = `Hello, ${string}`;
// Matches: "Hello, world", "Hello, ", "Hello, 42" ...

type EventName = `${"click" | "focus"}_${"start" | "end"}`;
// = "click_start" | "click_end" | "focus_start" | "focus_end"
```

### Combinatorial Explosion

For union types, template literals compute the **Cartesian product**:

$$|A \cup B| \times |C \cup D| = |A| \times |C| + |A| \times |D| + |B| \times |C| + |B| \times |D|$$

```typescript
type Digit = "0"|"1"|"2"|"3"|"4"|"5"|"6"|"7"|"8"|"9";
type TwoDigit = `${Digit}${Digit}`;  // 100 literal types
```

### Intrinsic String Types

| Type | Effect |
|:-----|:-------|
| `Uppercase<S>` | `"hello"` → `"HELLO"` |
| `Lowercase<S>` | `"HELLO"` → `"hello"` |
| `Capitalize<S>` | `"hello"` → `"Hello"` |
| `Uncapitalize<S>` | `"Hello"` → `"hello"` |

---

## 7. The Type System is Turing Complete

TypeScript's type system can compute arbitrary functions at compile time. It has:
- **Recursion** (recursive conditional types)
- **Branching** (conditional types)
- **Data structures** (tuple types as lists)
- **Pattern matching** (`infer`)

### Example: Type-Level Fibonacci

```typescript
type Fib<N extends number, A extends any[] = [1], B extends any[] = []> =
    B["length"] extends N
        ? A["length"]
        : Fib<N, [...A, ...B], A>;

type F5 = Fib<5>;  // 8
type F6 = Fib<6>;  // 13
```

This uses tuple lengths as natural numbers (Peano arithmetic at the type level).

### Recursion Depth Limit

TypeScript has a recursion depth limit of **~1000** for type instantiation. Exceeding it produces: `Type instantiation is excessively deep and possibly infinite.`

---

## 8. Type Narrowing and Control Flow Analysis

### Narrowing Guards

| Guard | Narrows To |
|:------|:-----------|
| `typeof x === "string"` | `string` |
| `x instanceof Foo` | `Foo` |
| `"prop" in x` | `{ prop: unknown }` |
| `x === null` | `null` |
| `x != null` | Exclude `null` and `undefined` |
| Discriminated union check | Specific variant |

### Discriminated Unions

```typescript
type Shape =
    | { kind: "circle"; radius: number }
    | { kind: "rect"; width: number; height: number };

function area(s: Shape): number {
    switch (s.kind) {
        case "circle": return Math.PI * s.radius ** 2;  // narrowed to circle
        case "rect": return s.width * s.height;          // narrowed to rect
    }
}
```

### Exhaustiveness Checking via `never`

```typescript
function assertNever(x: never): never {
    throw new Error("Unexpected: " + x);
}
// If a new variant is added to Shape but not handled,
// x won't be 'never' → compile error
```

---

## 9. Summary of Type-Level Concepts

| Concept | Analogy | TypeScript |
|:--------|:--------|:-----------|
| Types as sets | Set theory | `union = ∪`, `intersection = ∩` |
| Subtyping | Subset relation | $A <: B \iff A \subseteq B$ |
| Conditional types | If/else | `T extends U ? X : Y` |
| Mapped types | `Array.map` | `{ [K in keyof T]: ... }` |
| Template literals | String interpolation | `` `${A}_${B}` `` |
| `infer` | Pattern matching / unification | `T extends F<infer R> ? R : never` |
| `never` | Empty set $\emptyset$ | Bottom type |
| `unknown` | Universal set $U$ | Top type |
| `any` | Escape hatch | Both top and bottom (unsound) |

---

*TypeScript is two languages: a runtime language (JavaScript) and a compile-time language (the type system). The type system is genuinely a functional programming language with recursion, pattern matching, and data structures — it just happens to erase completely before execution.*

## Prerequisites

- JavaScript language fundamentals (prototypes, closures, event loop)
- Type theory (structural typing, union/intersection types, generics)
- Type inference and type narrowing concepts
- Module systems (ESM, CommonJS) and declaration files
