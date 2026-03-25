# TypeScript (Typed JavaScript Superset)

> Statically typed superset of JavaScript that compiles to plain JS; adds type safety, interfaces, and advanced type features.

## Primitive Types

### Basic Types

```typescript
let str: string = "hello";
let num: number = 42;           // No int/float distinction
let big: bigint = 100n;
let bool: boolean = true;
let nul: null = null;
let undef: undefined = undefined;
let sym: symbol = Symbol("id");
```

### Special Types

```typescript
let anything: any = "no checking";       // Opts out of type checking
let unknown: unknown = getValue();       // Type-safe any — must narrow before use
let nothing: void = undefined;           // Functions that return nothing
let impossible: never = throwError();    // Functions that never return
```

## Union, Intersection, and Literal Types

### Union Types

```typescript
let id: string | number = "abc";
id = 42;  // Also valid

function format(value: string | number): string {
  if (typeof value === "string") return value.toUpperCase();
  return value.toFixed(2);
}
```

### Intersection Types

```typescript
type Timestamped = { createdAt: Date };
type Named = { name: string };
type User = Named & Timestamped;

const user: User = { name: "Alice", createdAt: new Date() };
```

### Literal Types

```typescript
type Direction = "north" | "south" | "east" | "west";
type HttpStatus = 200 | 301 | 404 | 500;
type Toggle = true | false;

let dir: Direction = "north";
```

## Tuples

```typescript
let pair: [string, number] = ["age", 30];
let triple: [string, number, boolean] = ["active", 1, true];

// Named tuples (for readability)
type Point = [x: number, y: number, z: number];

// Rest elements
type StringAndNumbers = [string, ...number[]];

// Readonly tuple
const coords: readonly [number, number] = [10, 20];
```

## Interfaces

```typescript
interface User {
  id: number;
  name: string;
  email?: string;              // Optional
  readonly createdAt: Date;    // Immutable after creation
}

// Extending
interface Admin extends User {
  permissions: string[];
}

// Index signatures
interface Dictionary {
  [key: string]: string;
}

// Function signature in interface
interface Formatter {
  (value: string): string;
}

// Merging (declaration merging)
interface User {
  age: number;    // Merged with original User interface
}
```

## Type Aliases

```typescript
type ID = string | number;
type Callback = (data: string) => void;
type Pair<T> = [T, T];
type Nullable<T> = T | null;

// Discriminated unions
type Shape =
  | { kind: "circle"; radius: number }
  | { kind: "rect"; width: number; height: number };

function area(s: Shape): number {
  switch (s.kind) {
    case "circle": return Math.PI * s.radius ** 2;
    case "rect":   return s.width * s.height;
  }
}
```

## Generics

### Functions

```typescript
function identity<T>(arg: T): T {
  return arg;
}

function first<T>(arr: T[]): T | undefined {
  return arr[0];
}

// Multiple type parameters
function merge<A, B>(a: A, b: B): A & B {
  return { ...a, ...b };
}
```

### Constraints

```typescript
function getLength<T extends { length: number }>(item: T): number {
  return item.length;
}

// keyof constraint
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
  return obj[key];
}
```

### Generic Interfaces and Classes

```typescript
interface Repository<T> {
  findById(id: string): Promise<T>;
  save(entity: T): Promise<void>;
}

class Box<T> {
  constructor(private value: T) {}
  getValue(): T { return this.value; }
}
```

## Utility Types

```typescript
// Partial — all properties optional
type PartialUser = Partial<User>;              // { id?: number; name?: string; ... }

// Required — all properties required
type RequiredUser = Required<User>;

// Readonly — all properties readonly
type FrozenUser = Readonly<User>;

// Pick — select specific properties
type UserName = Pick<User, "id" | "name">;     // { id: number; name: string }

// Omit — exclude specific properties
type PublicUser = Omit<User, "email">;

// Record — construct object type
type Roles = Record<string, string[]>;         // { [key: string]: string[] }
type StatusMap = Record<"active" | "inactive", User[]>;

// Extract / Exclude — filter union types
type T1 = Extract<"a" | "b" | "c", "a" | "c">;    // "a" | "c"
type T2 = Exclude<"a" | "b" | "c", "a">;           // "b" | "c"

// NonNullable — remove null and undefined
type T3 = NonNullable<string | null | undefined>;   // string

// ReturnType — extract function return type
type T4 = ReturnType<typeof JSON.parse>;            // any

// Parameters — extract function parameter types
type T5 = Parameters<typeof setTimeout>;            // [callback: ..., ms?: ...]

// Awaited — unwrap Promise type
type T6 = Awaited<Promise<string>>;                 // string
```

## Enums

```typescript
// Numeric enum (auto-increments from 0)
enum Direction {
  Up,       // 0
  Down,     // 1
  Left,     // 2
  Right,    // 3
}

// String enum
enum Status {
  Active = "ACTIVE",
  Inactive = "INACTIVE",
  Pending = "PENDING",
}

// const enum (inlined at compile time, no runtime object)
const enum Color {
  Red = "#ff0000",
  Green = "#00ff00",
  Blue = "#0000ff",
}

// Prefer union types over enums for most cases
type Status2 = "active" | "inactive" | "pending";
```

## Type Guards

```typescript
// typeof
function process(value: string | number) {
  if (typeof value === "string") {
    value.toUpperCase();    // TypeScript knows it's string
  }
}

// instanceof
function handle(err: Error | string) {
  if (err instanceof Error) {
    err.message;
  }
}

// in operator
function move(shape: Shape) {
  if ("radius" in shape) {
    shape.radius;           // Narrowed to circle
  }
}

// Custom type guard
function isUser(obj: unknown): obj is User {
  return typeof obj === "object" && obj !== null && "id" in obj && "name" in obj;
}

// Assertion function
function assertDefined<T>(val: T | null | undefined): asserts val is T {
  if (val == null) throw new Error("Value is null or undefined");
}
```

## tsconfig.json

### Key Options

```jsonc
{
  "compilerOptions": {
    // Target and module
    "target": "ES2022",               // Output JS version
    "module": "NodeNext",             // Module system
    "moduleResolution": "NodeNext",   // Module resolution strategy
    "lib": ["ES2022", "DOM"],         // Available type definitions

    // Strict mode (recommended)
    "strict": true,                   // Enable all strict checks
    "noUncheckedIndexedAccess": true, // Indexed access returns T | undefined
    "exactOptionalPropertyTypes": true,

    // Output
    "outDir": "./dist",
    "rootDir": "./src",
    "declaration": true,              // Generate .d.ts files
    "sourceMap": true,

    // Interop
    "esModuleInterop": true,          // CommonJS/ESM interop
    "forceConsistentCasingInFileNames": true,
    "skipLibCheck": true,

    // Path aliases
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

## Declaration Files

```typescript
// types.d.ts — ambient declarations
declare module "untyped-lib" {
  export function doSomething(input: string): number;
}

// Global augmentation
declare global {
  interface Window {
    myApi: { version: string };
  }
}

// Type-only imports
import type { User } from "./models";
```

## Tips

- Prefer `unknown` over `any`; it forces you to narrow the type before using it.
- Use `as const` to infer literal types from values: `const dirs = ["n", "s"] as const`.
- Discriminated unions (tagged unions) with a shared `kind` field are the idiomatic way to model variants.
- `satisfies` operator (TS 5.0+) validates a value matches a type while preserving the narrower inferred type.
- Avoid enums in libraries; use string union types instead for better tree-shaking and interoperability.
- Use `noUncheckedIndexedAccess` to catch array/object index access bugs at compile time.
- Template literal types enable type-safe string manipulation: `type Route = \`/api/${string}\``.

## References

- [TypeScript Documentation](https://www.typescriptlang.org/docs/) -- official docs and getting started
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html) -- core language guide
- [TypeScript Playground](https://www.typescriptlang.org/play) -- run and share TypeScript online
- [tsconfig Reference](https://www.typescriptlang.org/tsconfig) -- every compiler option explained
- [TypeScript Cheat Sheets](https://www.typescriptlang.org/cheatsheets) -- official visual quick references
- [TypeScript Release Notes](https://www.typescriptlang.org/docs/handbook/release-notes/overview.html) -- changelog per version
- [TypeScript GitHub](https://github.com/microsoft/TypeScript) -- source code, issues, and design discussions
- [DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped) -- community type definitions for npm packages
- [npm Registry](https://www.npmjs.com/) -- package registry (TypeScript and JavaScript)
- [Type Challenges](https://github.com/type-challenges/type-challenges) -- collection of TypeScript type puzzles
