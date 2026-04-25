# TypeScript (Programming Language)

Statically-typed superset of JavaScript that compiles to plain JS — adds a structural type system, generics, conditional and mapped types, and a deep tooling ecosystem on top of every JS runtime.

## Setup

### Install the compiler

```bash
# Global install (rare — most projects pin per-repo):
# npm install -g typescript
# tsc --version          # tsc is the compiler binary
#
# Per-project (the right answer):
# npm init -y
# npm install -D typescript @types/node
# npx tsc --init                                   // generates tsconfig.json with comments
```

### Pick a runtime

```bash
# Node.js (the workhorse — Node 18+ ships native fetch, AbortController, Web Streams):
# npx tsx script.ts                                // tsx — fastest dev runner; uses esbuild
# npx ts-node script.ts                            // older; slower; still common in legacy repos
# node --experimental-strip-types script.ts        // 22.6+ strips types at runtime, no transpile
# node script.js                                   // run after tsc compiles to JS
#
# Deno — TypeScript is the default:
# deno run script.ts                               // no tsconfig needed; permissions are explicit
# deno run --allow-net --allow-read script.ts
#
# Bun — bundler + runtime + test runner in one binary:
# bun run script.ts                                // no transpile step; native TS
# bun test                                         // runs *.test.ts and *.spec.ts
```

### Initialize a project

```bash
# A minimal modern Node + TS project:
# mkdir my-app && cd my-app
# npm init -y
# npm install -D typescript @types/node tsx vitest
# npx tsc --init --strict --target ES2022 --module NodeNext --moduleResolution NodeNext
# mkdir src && echo 'console.log("hello")' > src/index.ts
# npx tsx src/index.ts                             // run with hot dev runner
# npx tsc --noEmit                                 // type-check only, no JS output
```

## tsconfig.json Essentials

### The strict baseline (do this, always)

```bash
# // tsconfig.json — the non-negotiable strict modern config
# {
#   "compilerOptions": {
#     "target": "ES2022",                          // emit modern JS — Node 18+, evergreen browsers
#     "module": "NodeNext",                        // ESM-first; "CommonJS" only for legacy
#     "moduleResolution": "NodeNext",              // matches "module"; uses package.json "exports"
#     "lib": ["ES2023", "DOM", "DOM.Iterable"],   // available global types
#     "jsx": "react-jsx",                          // for React 17+; "preserve" for Next.js
#
#     "strict": true,                              // enables 8 strict flags — non-negotiable
#     "noUncheckedIndexedAccess": true,            // arr[0] becomes T | undefined
#     "exactOptionalPropertyTypes": true,          // { x?: T } is NOT { x: T | undefined }
#     "noImplicitOverride": true,                  // override keyword required on subclass methods
#     "noFallthroughCasesInSwitch": true,
#     "noPropertyAccessFromIndexSignature": true,  // forces obj["x"] for index signatures
#
#     "outDir": "./dist",
#     "rootDir": "./src",
#     "declaration": true,                         // emit .d.ts alongside .js
#     "sourceMap": true,
#     "esModuleInterop": true,                     // import fs from "fs" works
#     "forceConsistentCasingInFileNames": true,
#     "skipLibCheck": true,                        // skip type-checking node_modules
#     "resolveJsonModule": true,                   // import data from "./data.json"
#
#     "baseUrl": ".",
#     "paths": { "@/*": ["src/*"] }                // "@/foo" -> "./src/foo"
#   },
#   "include": ["src/**/*"],
#   "exclude": ["node_modules", "dist"]
# }
```

### Target vs lib vs module — the trio

```bash
# target:           output JS syntax level (ES2022 = native await/optional chaining/private #)
# lib:              built-in TYPE definitions available (DOM, ES2023, WebWorker, etc.)
# module:           output module format (NodeNext, ES2022, CommonJS, AMD, UMD, System, None)
# moduleResolution: how "import 'x'" is resolved (NodeNext, Bundler, Node10, Classic)
#
# Rules of thumb:
# - Library code:     module=ES2022, moduleResolution=Bundler, declaration=true
# - Node app:         module=NodeNext, moduleResolution=NodeNext
# - Browser bundler:  module=ESNext,  moduleResolution=Bundler   // Vite/webpack/esbuild handle the rest
```

## Variables: const / let / `as const` / `satisfies`

### Declaration

```bash
# const x = 42;                  // immutable binding; TS infers number
# let y = 0;                     // mutable
# var z = 1;                     // never use var — function-scoped, hoisted, no temporal dead zone
#
# const PI = 3.14;               // type: number
# const PI_LIT = 3.14 as const;  // type: 3.14 — narrow literal
# const dirs = ["n","s","e","w"] as const;     // type: readonly ["n","s","e","w"]
```

### `satisfies` (TS 4.9+) — the operator everyone reaches for

```bash
# // Without satisfies — you lose the narrow type:
# const config: Record<string, string | number> = { host: "x", port: 8080 };
# config.port.toFixed(2);        // ERROR — port is string|number
#
# // With satisfies — keeps the inferred narrow shape AND validates the constraint:
# const config = { host: "x", port: 8080 } satisfies Record<string, string | number>;
# config.port.toFixed(2);        // OK — port is inferred as number
# config.host.toUpperCase();     // OK — host is inferred as string
```

### `const` vs `as const`

```bash
# const a = { kind: "circle", radius: 5 };
# // type: { kind: string; radius: number }
#
# const b = { kind: "circle", radius: 5 } as const;
# // type: { readonly kind: "circle"; readonly radius: 5 }
#
# // as const is what makes literal types "stick" through inference.
```

## Primitive Types

### The seven primitives

```bash
# const s: string = "hello";                       // 16-bit UTF-16 code units
# const n: number = 42;                            // 64-bit IEEE-754 float (no int/float distinction)
# const b: boolean = true;
# const big: bigint = 9007199254740993n;           // arbitrary-precision integer; n suffix
# const sym: symbol = Symbol("id");                // unique identity; use for non-clashing keys
# const u: undefined = undefined;                  // missing value
# const nul: null = null;                          // explicit no value (distinct from undefined)
#
# // strictNullChecks splits null/undefined OUT of every other type.
# // That single flag is the entire reason TS catches the bugs JS misses.
```

### `any` vs `unknown` vs `never` vs `void`

```bash
# let bad: any;                  // opts OUT of type-checking — avoid
# let good: unknown;             // type-safe any — must NARROW before use
# function err(): never { throw new Error("nope"); }  // returns nothing, ever
# function log(): void {}        // returns nothing meaningful (undefined is fine)
#
# // unknown forces the narrow:
# function parse(raw: unknown): number {
#   if (typeof raw === "number") return raw;       // narrowed
#   if (typeof raw === "string") return Number(raw);
#   throw new TypeError("bad input");
# }
```

## Literal Types & Template Literal Types

### Literal unions

```bash
# type Direction = "north" | "south" | "east" | "west";
# type HttpStatus = 200 | 301 | 404 | 500;
# type Bit = 0 | 1;
#
# const dir: Direction = "north";
# // dir = "up";                 // ERROR
```

### Template literal types

```bash
# type Greeting<T extends string> = `hello, ${T}`;
# type G = Greeting<"world">;    // "hello, world"
#
# type Method = "GET" | "POST";
# type Path = "/users" | "/posts";
# type Route = `${Method} ${Path}`;
# // "GET /users" | "GET /posts" | "POST /users" | "POST /posts"
#
# type CamelCase<S extends string> =
#   S extends `${infer L}_${infer R}${infer Rest}` ? `${L}${Uppercase<R>}${CamelCase<Rest>}` : S;
# type X = CamelCase<"hello_world">;   // "helloWorld"
```

## Arrays & Tuples

### Arrays

```bash
# const xs: number[] = [1, 2, 3];                  // T[] form
# const ys: Array<number> = [1, 2, 3];             // generic form — identical
# const ro: readonly number[] = [1, 2, 3];         // immutable view
# const ro2: ReadonlyArray<number> = [1, 2, 3];
#
# // noUncheckedIndexedAccess turns this on:
# const first = xs[0];           // type: number | undefined  (without flag: number — bug!)
```

### Tuples — fixed-length, fixed-types

```bash
# const pair: [string, number] = ["age", 30];
# const triple: [string, number, boolean] = ["active", 1, true];
#
# // Named tuples — for readability and editor hints:
# type Point = [x: number, y: number, z: number];
# const p: Point = [1, 2, 3];
#
# // Rest in tuples:
# type StrThenNums = [string, ...number[]];
# const t: StrThenNums = ["x", 1, 2, 3];
#
# // Readonly tuple:
# const coords: readonly [number, number] = [10, 20];
# // coords[0] = 99;             // ERROR — readonly
```

## Object types vs `Record<K, V>` vs `Map<K, V>`

### Object literal types

```bash
# type User = { id: number; name: string };        // sealed shape
# const u: User = { id: 1, name: "Ada" };
#
# type Bag = { [k: string]: number };              // index signature — open-ended
# const b: Bag = { x: 1, y: 2 };
```

### `Record<K, V>` — built-in helper

```bash
# type Roles = Record<string, string[]>;           // identical to { [k: string]: string[] }
# type StatusMap = Record<"active" | "inactive", number>;   // exact keys, all required
#
# const sm: StatusMap = { active: 5, inactive: 0 };
# // const sm2: StatusMap = { active: 5 };         // ERROR — missing "inactive"
```

### When to reach for `Map<K, V>` instead

```bash
# // Object/Record:    string|number|symbol keys; insertion-order via Object.keys
# // Map:              ANY key (objects, instances); preserves insertion order; .size is O(1)
#
# const m = new Map<string, number>();
# m.set("k", 1);
# m.get("k");                   // number | undefined
# m.has("k");                   // boolean
# m.delete("k");
# m.size;                       // 0
#
# // Iterate in insertion order:
# for (const [k, v] of m) { console.log(k, v); }
#
# // Use Map when:
# // - keys are non-string (e.g. DOM nodes, class instances)
# // - you need .size frequently
# // - you'll iterate often (Map is cache-friendlier than Object for hot loops)
```

## Set & Map

### Set — unique values, insertion order

```bash
# const s = new Set<number>([1, 2, 3, 2]);         // {1, 2, 3}
# s.add(4);
# s.has(2);                     // true
# s.delete(1);
# s.size;                       // 3
#
# // Convert array <-> set (dedupe pattern):
# const unique = [...new Set([1, 1, 2, 3, 2])];    // [1, 2, 3]
#
# // ES2025 set algebra:
# const a = new Set([1, 2, 3]);
# const b = new Set([2, 3, 4]);
# a.union(b);                   // {1,2,3,4}
# a.intersection(b);            // {2,3}
# a.difference(b);              // {1}
# a.symmetricDifference(b);     // {1,4}
```

### Map vs WeakMap / Set vs WeakSet

```bash
# // WeakMap: keys are GC-eligible — entry vanishes when key is unreferenced.
# // Useful for attaching metadata to objects without preventing collection.
# const meta = new WeakMap<object, string>();
# const obj = {};
# meta.set(obj, "tag");
# // when obj is unreferenced, the entry is collected.
#
# // WeakMap restrictions: keys MUST be objects; not iterable; no .size.
```

## Enum (regular, const, ambient — when to avoid)

### Numeric enum (auto-increments)

```bash
# enum Direction { Up, Down, Left, Right }    // 0, 1, 2, 3
# const d: Direction = Direction.Up;
# console.log(Direction[0]);                  // "Up" — reverse mapping (numeric only)
```

### String enum

```bash
# enum Status {
#   Active = "ACTIVE",
#   Inactive = "INACTIVE",
#   Pending = "PENDING",
# }
```

### const enum — inlined at compile time

```bash
# const enum Color { Red = "#f00", Green = "#0f0", Blue = "#00f" }
# const c = Color.Red;          // emitted as: const c = "#f00";  — no runtime object
#
# // Caveat: const enums break under isolatedModules (Babel, esbuild, swc, ts-jest in some configs).
```

### When to avoid enums

```bash
# // Reasons to prefer string union types over enums:
# // - tree-shakable (enums are not)
# // - JSON-friendly (no reverse-mapping cruft)
# // - structural (interop with libraries that don't import your enum)
# // - no runtime cost
# type Status = "active" | "inactive" | "pending";
# const s: Status = "active";
```

## Type Aliases vs Interfaces

### When each makes sense

```bash
# // type alias — anything: union, intersection, primitive, tuple, conditional, mapped...
# type ID = string | number;
# type Point = readonly [number, number];
# type Nullable<T> = T | null;
#
# // interface — object shapes, classes, declaration merging
# interface User {
#   id: number;
#   name: string;
#   readonly createdAt: Date;
# }
# interface Admin extends User { permissions: string[]; }
#
# // Interfaces can MERGE across declarations (a feature for global augmentation):
# interface Window { myApi: { version: string }; }    // adds to lib.dom.d.ts
#
# // Rule of thumb:
# // - Library public API:           interface (extensible by consumers via merge)
# // - Internal/algebraic types:     type alias (more flexible)
# // - Anything not an object shape: type alias (only option)
```

## Union & Intersection Types

```bash
# type ID = string | number;                       // UNION — either
# function show(id: ID) {
#   if (typeof id === "string") return id.toUpperCase();
#   return id.toFixed(0);
# }
#
# type Timestamped = { createdAt: Date };
# type Named = { name: string };
# type Entity = Named & Timestamped;               // INTERSECTION — both
# const e: Entity = { name: "Ada", createdAt: new Date() };
#
# // Intersections of incompatible primitives produce never:
# type Impossible = string & number;               // never
```

## Discriminated Unions

### The `kind:` pattern

```bash
# type Shape =
#   | { kind: "circle"; radius: number }
#   | { kind: "rect";   width: number; height: number }
#   | { kind: "tri";    base: number; height: number };
#
# function area(s: Shape): number {
#   switch (s.kind) {
#     case "circle": return Math.PI * s.radius ** 2;
#     case "rect":   return s.width * s.height;
#     case "tri":    return 0.5 * s.base * s.height;
#     default: {
#       const _exhaustive: never = s;              // compile error if a kind is missed
#       throw new Error(`unhandled: ${_exhaustive}`);
#     }
#   }
# }
```

### Why the `never` check matters

```bash
# // Add a fourth variant later:
# // | { kind: "ellipse"; rx: number; ry: number };
# // The switch above stops compiling — TS tells you exactly where to handle the new case.
# // This is compile-time exhaustiveness — the single biggest reason discriminated unions exist.
```

## Optional & Nullable

### `T | undefined` vs `?:`

```bash
# // exactOptionalPropertyTypes flag matters here:
# type A = { x?: number };                         // x can be ABSENT
# type B = { x: number | undefined };              // x must be PRESENT, value undefined
#
# const a1: A = {};               // OK
# const a2: A = { x: undefined }; // OK without exactOptionalPropertyTypes; ERROR with it
# const b1: B = {};               // ERROR — x is required
# const b2: B = { x: undefined }; // OK
```

### `??`, `?.`, and the `!` non-null assertion

```bash
# const v = x ?? 0;              // null OR undefined → fallback (NOT 0/""/false)
# const v2 = x || 0;             // ANY falsy → fallback (use ?? unless you really mean falsy)
#
# user?.address?.city            // optional chaining — short-circuits on null/undefined
# arr?.[0]                       // optional indexed access
# fn?.()                         // optional call
#
# // ?? assignment / ?.= assignment:
# obj.x ??= "default";           // assign if null/undefined
#
# // The non-null assertion — escape hatch you should rarely use:
# const el = document.getElementById("x")!;        // tells TS "trust me, not null"
# // This bypasses the type system. Prefer narrowing with a guard.
```

## Type Narrowing

### `typeof`, `instanceof`, `in`, equality

```bash
# function fmt(v: string | number | Date): string {
#   if (typeof v === "string") return v.toUpperCase();
#   if (typeof v === "number") return v.toFixed(2);
#   if (v instanceof Date)     return v.toISOString();
#   const _: never = v;
#   throw new Error("unreachable");
# }
#
# // `in` operator — discriminates by property presence:
# function move(s: { kind: "c"; r: number } | { kind: "r"; w: number }) {
#   if ("r" in s) s.r;           // narrowed to circle
#   else          s.w;           // narrowed to rect
# }
#
# // Equality narrowing — kind: tags exploit this:
# if (s.kind === "circle") s.radius;
```

### User-defined type guards (`x is T`)

```bash
# type User = { id: number; name: string };
#
# function isUser(v: unknown): v is User {
#   return typeof v === "object" && v !== null
#       && "id" in v && typeof (v as User).id === "number"
#       && "name" in v && typeof (v as User).name === "string";
# }
#
# const raw: unknown = JSON.parse(input);
# if (isUser(raw)) { raw.name.toUpperCase(); }     // narrowed
```

### Assertion functions (`asserts x is T`)

```bash
# function assertDefined<T>(v: T | null | undefined): asserts v is T {
#   if (v == null) throw new Error("missing");
# }
#
# const x: string | undefined = process.env.NAME;
# assertDefined(x);
# x.toUpperCase();               // x: string here
#
# // assert into specific type:
# function assertString(v: unknown): asserts v is string {
#   if (typeof v !== "string") throw new TypeError("not string");
# }
```

## Generics

### Generic functions

```bash
# function identity<T>(x: T): T { return x; }
# const n = identity(42);        // T inferred: number
# const s = identity<string>("hi");
#
# function first<T>(xs: readonly T[]): T | undefined { return xs[0]; }
#
# function merge<A extends object, B extends object>(a: A, b: B): A & B {
#   return { ...a, ...b };
# }
```

### Generic constraints with `extends`

```bash
# function len<T extends { length: number }>(x: T): number { return x.length; }
# len("hi");                     // 2
# len([1,2,3]);                  // 3
# len({ length: 99 });           // 99
# // len(42);                    // ERROR — number has no .length
#
# // keyof constraint — index a known property:
# function pick<T, K extends keyof T>(obj: T, key: K): T[K] {
#   return obj[key];
# }
# const u = { id: 1, name: "Ada" };
# pick(u, "name");               // string
# // pick(u, "missing");         // ERROR — not a key
```

### Generic defaults

```bash
# interface Box<T = string> { value: T; }
# const b1: Box = { value: "x" };          // T defaults to string
# const b2: Box<number> = { value: 42 };
#
# function fetchJSON<T = unknown>(url: string): Promise<T> {
#   return fetch(url).then(r => r.json());
# }
# const data = await fetchJSON<User>("/api/me");
```

## Conditional Types

### `T extends U ? X : Y`

```bash
# type IsString<T> = T extends string ? "yes" : "no";
# type A = IsString<"hi">;       // "yes"
# type B = IsString<42>;         // "no"
```

### Distributive over unions

```bash
# type ToArray<T> = T extends any ? T[] : never;
# type X = ToArray<string | number>;       // string[] | number[]
#
# // Wrap in [] to disable distribution:
# type ToArray2<T> = [T] extends [any] ? T[] : never;
# type Y = ToArray2<string | number>;      // (string | number)[]
```

### `infer` — extract types from positions

```bash
# type ReturnTypeOf<F> = F extends (...args: any[]) => infer R ? R : never;
# type R = ReturnTypeOf<() => number>;     // number
#
# type ElementOf<T> = T extends (infer E)[] ? E : never;
# type E = ElementOf<string[]>;            // string
#
# type FirstParam<F> = F extends (a: infer A, ...rest: any[]) => any ? A : never;
```

## Mapped Types

### Iterate over keys of a type

```bash
# type Readonlyify<T> = { readonly [K in keyof T]: T[K] };
# type Optional<T> = { [K in keyof T]?: T[K] };
# type Nullable<T> = { [K in keyof T]: T[K] | null };
#
# type User = { id: number; name: string };
# type ROUser = Readonlyify<User>;
# type PartialUser = Optional<User>;
```

### Modifiers `+? -? +readonly -readonly`

```bash
# type StripOptional<T> = { [K in keyof T]-?: T[K] };
# type StripReadonly<T> = { -readonly [K in keyof T]: T[K] };
#
# type In = { readonly a?: number; b: string };
# type Out = StripOptional<StripReadonly<In>>;     // { a: number; b: string }
```

### `as` remapping (TS 4.1+)

```bash
# // Prefix every key:
# type Prefixed<T, P extends string> = { [K in keyof T as `${P}${string & K}`]: T[K] };
# type X = Prefixed<{ id: number; name: string }, "user_">;
# // { user_id: number; user_name: string }
#
# // Filter keys (drop a key by mapping to never):
# type RemoveKey<T, K> = { [P in keyof T as P extends K ? never : P]: T[P] };
```

## Utility Types

### The full bestiary

```bash
# type User = { id: number; name: string; email?: string };
#
# Partial<User>;           // { id?: number; name?: string; email?: string }
# Required<User>;          // { id: number; name: string; email: string }
# Readonly<User>;          // { readonly id: number; readonly name: string; ... }
# Pick<User, "id"|"name"|"email"> ;   // subset
# Omit<User, "email">;     // { id: number; name: string }
# Record<"a"|"b", User>;   // { a: User; b: User }
#
# Extract<"a"|"b"|"c", "a"|"c">;      // "a" | "c"
# Exclude<"a"|"b"|"c", "a">;          // "b" | "c"
# NonNullable<string|null|undefined>; // string
#
# ReturnType<typeof JSON.parse>;      // any
# Parameters<typeof setTimeout>;      // [callback: ..., ms?: ...]
# ConstructorParameters<typeof Date>; // [] | [...]
# InstanceType<typeof Date>;          // Date
#
# Awaited<Promise<string>>;           // string
# Awaited<Promise<Promise<number>>>;  // number — recursively unwraps
#
# Uppercase<"hi">;                    // "HI"
# Lowercase<"HI">;                    // "hi"
# Capitalize<"hi">;                   // "Hi"
# Uncapitalize<"Hi">;                 // "hi"
```

## keyof, typeof, indexed access

### `keyof` — keys as a union

```bash
# type User = { id: number; name: string };
# type K = keyof User;             // "id" | "name"
# type V = User[K];                // number | string  — indexed access
#
# function get<T, K extends keyof T>(o: T, k: K): T[K] { return o[k]; }
```

### `typeof value` — get the type of a runtime value

```bash
# const cfg = { host: "x", port: 8080 } as const;
# type Cfg = typeof cfg;          // { readonly host: "x"; readonly port: 8080 }
# type Host = Cfg["host"];        // "x"
#
# // Useful to mirror a const object as a type:
# const ROLES = { admin: 1, user: 2, guest: 3 } as const;
# type Role = keyof typeof ROLES;  // "admin" | "user" | "guest"
# type Code = (typeof ROLES)[Role]; // 1 | 2 | 3
```

## Functions

### Declarations / expressions / arrow

```bash
# function add(a: number, b: number): number { return a + b; }
# const sub = function (a: number, b: number): number { return a - b; };
# const mul = (a: number, b: number): number => a * b;
#
# // Optional / default params:
# function greet(name: string, greeting: string = "hi"): string { return `${greeting}, ${name}`; }
#
# // Rest:
# function sum(...xs: number[]): number { return xs.reduce((a,b) => a+b, 0); }
```

### `this` binding rules

```bash
# // Arrow functions inherit `this` from enclosing lexical scope.
# // Regular functions get `this` from how they're called.
#
# class Counter {
#   n = 0;
#   incArrow = () => { this.n++; };           // bound to instance
#   incReg() { this.n++; }                    // `this` depends on call site
# }
# const c = new Counter();
# const f = c.incReg; f();                    // ERROR at runtime — `this` is undefined
# const g = c.incArrow; g();                  // OK — arrow captured `this`
#
# // Explicit `this` parameter (zero runtime cost — TS-only):
# function clickHandler(this: HTMLElement, ev: Event) { this.classList.add("clicked"); }
```

## Function overloads

```bash
# // Overload signatures + ONE implementation:
# function pad(value: string, length: number): string;
# function pad(value: number, length: number): string;
# function pad(value: string | number, length: number): string {
#   return String(value).padStart(length, "0");
# }
#
# pad("4", 3);                   // "004"
# pad(4, 3);                     // "004"
# // pad(true, 3);               // ERROR — no overload matches
#
# // Modern alternative: a single signature with a union and conditional return type.
```

## Classes

### Modifiers and parameter properties

```bash
# class User {
#   public readonly id: number;        // public is default
#   protected name: string;             // visible to subclasses
#   private _secret: string;            // visible only here
#   #internal = 0;                      // ECMAScript private — runtime-private
#
#   constructor(id: number, name: string, secret: string) {
#     this.id = id; this.name = name; this._secret = secret;
#   }
# }
#
# // Parameter properties — shorthand for assign-and-declare:
# class User2 {
#   constructor(
#     public readonly id: number,
#     protected name: string,
#     private _secret: string,
#   ) {}                              // fields are auto-initialized from params
# }
```

### `static`, `abstract`, getters/setters

```bash
# abstract class Shape {
#   abstract area(): number;          // subclass MUST implement
#   static origin = { x: 0, y: 0 };
# }
#
# class Circle extends Shape {
#   constructor(public r: number) { super(); }
#   area(): number { return Math.PI * this.r ** 2; }
#
#   get diameter(): number { return this.r * 2; }
#   set diameter(d: number) { this.r = d / 2; }
# }
#
# // const s = new Shape();         // ERROR — abstract class
# const c = new Circle(5);
# c.diameter;                       // 10
```

## Inheritance & `super`, mixins

```bash
# class Animal {
#   constructor(public name: string) {}
#   greet() { return `hi from ${this.name}`; }
# }
#
# class Dog extends Animal {
#   constructor(name: string, public breed: string) {
#     super(name);                  // MUST call super() before using this
#   }
#   override greet(): string {      // noImplicitOverride catches missing/wrong override
#     return super.greet() + " (woof)";
#   }
# }
#
# // Mixin pattern — function returning a class:
# type Ctor<T = {}> = new (...a: any[]) => T;
# function Timestamped<TBase extends Ctor>(Base: TBase) {
#   return class extends Base {
#     createdAt = new Date();
#   };
# }
# class TUser extends Timestamped(User) {}
```

## Modules

### ESM imports / exports

```bash
# // foo.ts
# export const PI = 3.14;
# export function area(r: number) { return PI * r * r; }
# export default function main() {}
# export type ID = string | number;
# export class Box {}
#
# // consumer.ts
# import main, { PI, area } from "./foo.js";        // .js extension required under NodeNext!
# import * as foo from "./foo.js";                  // namespace import
# import type { ID } from "./foo.js";               // type-only — erased at runtime
# import { type ID, area } from "./foo.js";         // mixed; `type` modifier on imported names
#
# // Re-export:
# export { area, PI } from "./foo.js";
# export * from "./foo.js";
# export { default } from "./foo.js";
```

### Default vs named — pick named

```bash
# // Default exports:
# // - rename freely on import (footgun for find-replace tooling)
# // - can't be tree-shaken as cleanly
# // - awkward when you need both type-only and value imports
#
# // Named exports:
# // - explicit, refactor-safe, tree-shake well
# // - the answer for ~99% of code
```

### Declaration merging (interfaces only)

```bash
# // Two interface declarations with the same name MERGE — feature, not bug:
# interface User { id: number; }
# interface User { name: string; }
# const u: User = { id: 1, name: "Ada" };           // both fields required
#
# // Type aliases CANNOT merge — duplicate is an error.
# // Useful for global augmentation:
# // declare global { interface Window { myApi: { v: string } } }
```

## Async / Promise / async-await

### Promises

```bash
# async function load(): Promise<User> {
#   const res = await fetch("/api/user");
#   if (!res.ok) throw new Error(`HTTP ${res.status}`);
#   return res.json() as Promise<User>;             // cast — JSON.parse returns any
# }
#
# // .then/.catch — usually use await instead:
# load().then(u => console.log(u)).catch(e => console.error(e));
```

### Promise combinators

```bash
# // All — fail-fast: first rejection rejects the whole batch:
# const [a, b, c] = await Promise.all([fa(), fb(), fc()]);
#
# // AllSettled — never rejects; you inspect each result:
# const results = await Promise.allSettled([fa(), fb()]);
# // [{ status: "fulfilled"|"rejected", value/reason }, ...]
#
# // Race — first settled (resolve OR reject) wins:
# const winner = await Promise.race([slow(), fast()]);
#
# // Any — first FULFILLED wins; rejects only if ALL reject:
# const ok = await Promise.any([flakyA(), flakyB()]);
```

### AbortController + timeout pattern

```bash
# async function fetchWithTimeout(url: string, ms: number): Promise<Response> {
#   const ac = new AbortController();
#   const timer = setTimeout(() => ac.abort(), ms);
#   try {
#     return await fetch(url, { signal: ac.signal });
#   } finally {
#     clearTimeout(timer);
#   }
# }
#
# // Node 17.3+ / browsers — built-in AbortSignal.timeout:
# const r = await fetch(url, { signal: AbortSignal.timeout(5_000) });
```

## Iterators & Generators

```bash
# // Iterable protocol — has [Symbol.iterator]() returning an Iterator
# function* range(start: number, end: number, step = 1): Generator<number> {
#   for (let i = start; i < end; i += step) yield i;
# }
#
# for (const n of range(0, 5)) console.log(n);     // 0 1 2 3 4
#
# // Spread an iterable:
# const xs = [...range(0, 3)];
#
# // Async iterables — for-await-of:
# async function* lines(file: string): AsyncGenerator<string> {
#   const rl = readline.createInterface({ input: fs.createReadStream(file) });
#   for await (const line of rl) yield line;
# }
# for await (const line of lines("data.txt")) process(line);
```

## Error Handling

### try / catch / finally

```bash
# try {
#   risky();
# } catch (e) {                                     // since TS 4.4: e is `unknown`
#   if (e instanceof Error) console.error(e.message, e.stack);
#   else                    console.error("non-error thrown:", e);
# } finally {
#   cleanup();
# }
```

### Custom Error subclasses

```bash
# class HttpError extends Error {
#   constructor(public status: number, message: string, options?: { cause?: unknown }) {
#     super(message, options);
#     this.name = "HttpError";
#     Object.setPrototypeOf(this, HttpError.prototype);   // restore prototype chain
#   }
# }
#
# // Throw with cause (ES2022):
# try { JSON.parse(bad); }
# catch (e) {
#   throw new HttpError(400, "bad input", { cause: e });
# }
```

### Narrowing `unknown` in catch

```bash
# // useUnknownInCatchVariables (default since 4.4) makes catch variables `unknown`:
# try { /* ... */ } catch (e) {
#   const msg = e instanceof Error ? e.message : String(e);
#   logger.error(msg);
# }
```

## Decorators (TC39 stage 3 — TS 5.0+)

### Modern decorators (no flag)

```bash
# // Class method decorator (stage 3):
# function log<This, Args extends unknown[], Return>(
#   value: (this: This, ...args: Args) => Return,
#   ctx: ClassMethodDecoratorContext<This, (this: This, ...args: Args) => Return>,
# ) {
#   return function (this: This, ...args: Args): Return {
#     console.log(`call ${String(ctx.name)}(${args.join(", ")})`);
#     return value.call(this, ...args);
#   };
# }
#
# class Calc {
#   @log
#   add(a: number, b: number) { return a + b; }
# }
```

### Legacy decorators (still common)

```bash
# // tsconfig: { "experimentalDecorators": true, "emitDecoratorMetadata": true }
# // Used by NestJS, TypeORM, Angular. Different API from stage-3.
#
# function Component(meta: { selector: string }): ClassDecorator {
#   return target => { (target as any).meta = meta; };
# }
# @Component({ selector: "my-app" })
# class App {}
```

## Strict Null Checks

```bash
# // The single most important compiler flag.
# // With strictNullChecks (part of "strict"), null and undefined are SEPARATE types.
#
# function head(xs: number[]): number { return xs[0]; }     // number — but really undefined possible!
# // With noUncheckedIndexedAccess: head returns number|undefined — caller must handle.
#
# function find<T>(xs: T[], pred: (x: T) => boolean): T | undefined {
#   for (const x of xs) if (pred(x)) return x;
#   return undefined;                                       // explicit; type system enforces it
# }
#
# const u = find(users, x => x.id === 5);
# u.name;                                                    // ERROR — u is User | undefined
# u?.name;                                                   // OK — string | undefined
# if (u) u.name;                                             // OK — narrowed
```

## JSON parsing

### `JSON.parse` returns `any` — guard it

```bash
# const raw = JSON.parse(input);          // type: any (NOT unknown — bug in lib types)
# // Treat as unknown immediately:
# const data: unknown = JSON.parse(input);
#
# function isUser(v: unknown): v is User {
#   return typeof v === "object" && v !== null
#       && "id" in v && typeof v.id === "number"
#       && "name" in v && typeof v.name === "string";
# }
# if (!isUser(data)) throw new Error("bad payload");
# data.name;                              // narrowed
```

### Reach for zod (or valibot / arktype) for non-trivial shapes

```bash
# import { z } from "zod";
# const User = z.object({ id: z.number().int(), name: z.string().min(1) });
# type User = z.infer<typeof User>;
# const parsed = User.parse(JSON.parse(input));     // throws ZodError on failure
```

## File I/O

### Node fs/promises (the default)

```bash
# import { readFile, writeFile, appendFile } from "node:fs/promises";
# const content = await readFile("data.txt", "utf8");
# await writeFile("out.txt", content);
# await appendFile("log.txt", "line\n");
#
# // Streams for large files:
# import { createReadStream } from "node:fs";
# import readline from "node:readline";
# const rl = readline.createInterface({ input: createReadStream("big.txt") });
# for await (const line of rl) process(line);
```

### Deno

```bash
# const text = await Deno.readTextFile("data.txt");
# await Deno.writeTextFile("out.txt", text);
# // run: deno run --allow-read --allow-write script.ts
```

### Bun File API

```bash
# const file = Bun.file("data.txt");
# const text = await file.text();
# await Bun.write("out.txt", text);                 // also accepts streams, bytes
```

## HTTP client (fetch)

```bash
# // fetch is native in Node 18+, Deno, Bun, and all modern browsers — no library needed.
# const res = await fetch("https://api.example.com/users/1", {
#   method: "GET",
#   headers: { authorization: `Bearer ${token}` },
#   signal: AbortSignal.timeout(5_000),
# });
# if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`);
# const user = (await res.json()) as User;
#
# // POST JSON:
# const r = await fetch("/api/users", {
#   method: "POST",
#   headers: { "content-type": "application/json" },
#   body: JSON.stringify({ name: "Ada" }),
# });
```

## Date / Intl / Temporal

### `Date` — the legacy

```bash
# const now = new Date();
# const ms = Date.now();                            // unix ms — number
# const iso = now.toISOString();                    // "2026-04-25T10:30:00.000Z"
# const parsed = new Date("2026-04-25T10:30:00Z");
#
# // Date is mutable and timezone-confusing. For real work, reach for Temporal or date-fns.
```

### `Intl` — formatting and collation

```bash
# new Intl.DateTimeFormat("en-GB", { dateStyle: "long" }).format(now);  // "25 April 2026"
# new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" }).format(1234.5);
# new Intl.RelativeTimeFormat("en").format(-3, "day");                  // "3 days ago"
# new Intl.Collator("en", { sensitivity: "base" }).compare("a", "Á");   // 0 — case/accent-insensitive
```

### `Temporal` (stage 3 proposal)

```bash
# // Polyfill: import "@js-temporal/polyfill"
# const today = Temporal.Now.plainDateISO();         // 2026-04-25
# const next = today.add({ days: 7 });
# const d = Temporal.PlainDate.from("2026-12-25");
# const dur = today.until(d).total({ unit: "day" });
```

## Regex

```bash
# const re = /(?<year>\d{4})-(?<month>\d{2})-(?<day>\d{2})/;
# const m = "2026-04-25".match(re);
# m?.groups?.year;                                  // "2026"
#
# // Flags: g (global), i (insensitive), m (multiline), s (dotAll), u (unicode), y (sticky), v (extended unicode)
# const all = "aaa".matchAll(/a/g);                  // iterator of RegExpExecArray
# for (const hit of all) console.log(hit.index);
#
# // Constructor form (when pattern is dynamic — REMEMBER to escape backslashes):
# const re2 = new RegExp(String.raw`\d+`, "g");
#
# // replaceAll with regex:
# "a1b2c3".replace(/\d/g, "*");                     // "a*b*c*"
# "a1b2c3".replaceAll(/\d/g, "*");                  // same — replaceAll requires /g flag for regex
```

## Declaration Files

### `.d.ts` — types only, no runtime

```bash
# // env.d.ts
# declare const __VERSION__: string;
# interface ImportMetaEnv {
#   readonly VITE_API_URL: string;
# }
# interface ImportMeta {
#   readonly env: ImportMetaEnv;
# }
#
# // ambient module — for a JS lib without types:
# declare module "untyped-lib" {
#   export function doIt(input: string): number;
#   export default function main(): void;
# }
#
# // Wildcard module — for non-JS imports through a bundler:
# declare module "*.svg" { const url: string; export default url; }
# declare module "*.css" { const styles: Record<string, string>; export default styles; }
```

### `types` vs `typings` package field

```bash
# // package.json — modern shape:
# {
#   "main": "./dist/index.js",
#   "types": "./dist/index.d.ts",
#   "exports": {
#     ".": {
#       "import": "./dist/index.mjs",
#       "require": "./dist/index.cjs",
#       "types":   "./dist/index.d.ts"
#     }
#   }
# }
# // "typings" is an old alias for "types" — same meaning. Use "types".
```

## Module resolution and `paths`

```bash
# // tsconfig.json
# {
#   "compilerOptions": {
#     "baseUrl": ".",
#     "paths": {
#       "@/*":       ["src/*"],
#       "@shared/*": ["packages/shared/src/*"]
#     }
#   }
# }
#
# // Code:
# import { db } from "@/lib/db";
# import type { User } from "@shared/types";
#
# // CRITICAL: tsc does NOT rewrite paths during emit.
# // - Bundlers (esbuild, vite, webpack, rollup, swc) honor paths.
# // - Pure Node + tsc requires a runtime resolver: tsconfig-paths, tsx, ts-node, etc.
```

## Build / Run

### tsc — the reference compiler

```bash
# npx tsc                           # build per tsconfig.json
# npx tsc --watch                   # watch mode
# npx tsc --noEmit                  # type-check only — what CI should run
# npx tsc --build packages/*/       # project references — incremental builds
# npx tsc --listFiles               # debug: see what files were included
# npx tsc --traceResolution         # debug: see why a module resolved (or didn't)
```

### Faster transpilers

```bash
# esbuild         — Go, blazing; transpile only, no type-check
# swc             — Rust, used by Next.js / Deno / Vitest internals
# tsup            — esbuild wrapper for libraries (.cjs + .esm + .d.ts)
# vite            — esbuild + rollup; the modern dev/build choice for SPAs
# bun build       — Zig; bundle + transpile in one binary
# deno compile    — produce a self-contained binary from a TS entrypoint
#
# Pattern: tsc for types (--noEmit in CI), fast transpiler for emit.
```

### Run TS without a build step

```bash
# npx tsx src/index.ts                                 // fastest dev runner (esbuild under the hood)
# node --import tsx src/index.ts                        // tsx as a Node loader
# node --experimental-strip-types src/index.ts          // 22.6+ — only erases types, no transform
# bun run src/index.ts
# deno run src/index.ts
```

## Test (vitest, jest, node:test) with TypeScript

### vitest — recommended for new projects

```bash
# // package.json scripts:  "test": "vitest"
# // vitest.config.ts:
# import { defineConfig } from "vitest/config";
# export default defineConfig({ test: { globals: true, environment: "node" } });
#
# // src/sum.test.ts
# import { describe, it, expect } from "vitest";
# import { sum } from "./sum";
# describe("sum", () => {
#   it("adds", () => { expect(sum(1, 2)).toBe(3); });
# });
```

### node:test (built-in, zero deps)

```bash
# // src/sum.test.ts
# import test from "node:test";
# import assert from "node:assert/strict";
# import { sum } from "./sum.js";
#
# test("sum adds", () => { assert.equal(sum(1, 2), 3); });
#
# // run:  node --test --import tsx 'src/**/*.test.ts'
# //       node --test 'dist/**/*.test.js'      // after tsc
```

### jest (legacy but ubiquitous)

```bash
# // npm i -D jest ts-jest @types/jest
# // jest.config.ts:
# export default {
#   preset: "ts-jest",
#   testEnvironment: "node",
#   testMatch: ["**/*.test.ts"],
# };
```

## Lint / Format

```bash
# // ESLint — most popular linter, plugin-rich:
# npm i -D eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin
# // eslint.config.js (flat config, ESLint 9+):
# import tseslint from "@typescript-eslint/eslint-plugin";
# export default [{ files: ["**/*.ts"], plugins: { "@typescript-eslint": tseslint } }];
#
# // Prettier — opinionated formatter:
# npx prettier --write .
#
# // Biome — Rust-based; lint + format in one binary, near-instant:
# npx @biomejs/biome init
# npx @biomejs/biome check --apply .
#
# // Use ONE formatter (Prettier OR Biome, not both). Lint = ESLint or Biome.
```

## Common Gotchas

### `==` vs `===` — always `===`

```bash
# 0 == "0";                        // true   — type coercion
# 0 === "0";                       // false  — strict equality, no coercion
# null == undefined;               // true   — only "useful" coercion
# null === undefined;              // false
#
# // Convention: use === everywhere EXCEPT `x == null` (catches both null and undefined).
```

### `this` binding

```bash
# class C { x = 1; m() { return this.x; } }
# const c = new C();
# const m = c.m;
# m();                             // ERROR — this is undefined in strict mode
# m.call(c);                       // 1
# c.m.bind(c)();                   // 1 — bound copy
# // Or: define as arrow on instance: m = () => this.x;
```

### Type erasure at runtime

```bash
# // Types vanish in compiled JS. NO instanceof check on a type alias / interface.
# interface User { id: number; }
# function isUser(v: unknown): v is User { /* MUST write the runtime check yourself */ }
#
# // class identity DOES survive — instanceof works:
# class HttpError extends Error {}
# err instanceof HttpError;        // OK at runtime
```

### Structural typing surprises

```bash
# // TS uses STRUCTURAL typing — anything with the right shape passes:
# type Named = { name: string };
# class Person { constructor(public name: string) {} }
# class Pet    { constructor(public name: string) {} }
# const x: Named = new Person("a");        // OK
# const y: Named = new Pet("b");           // OK — structurally identical
# // Use brand types when you NEED nominal-style identity (see Idioms).
```

### `any` vs `unknown`

```bash
# const a: any = JSON.parse(s);
# a.foo.bar.baz;                  // compiles, blows up at runtime
#
# const u: unknown = JSON.parse(s);
# u.foo;                          // ERROR — must narrow first
# // unknown is `any` with the safety on. ALWAYS prefer it.
```

### `interface` declaration merging surprise

```bash
# // Two interfaces with the same name MERGE silently. Two `type` aliases — error.
# // This bites when third-party @types augments your interface accidentally.
# // Library authors: prefer `type` for sealed shapes; interface for INTENDED extensibility.
```

### `as` casts hide bugs

```bash
# const n = "abc" as unknown as number;    // compiles, runtime is "abc"
# // Casts bypass the type system entirely. Replace with:
# // - a type guard
# // - a runtime parser (zod)
# // - a satisfies check
```

### `JSON.parse` returns `any` (not `unknown`)

```bash
# const v = JSON.parse(s);        // type: any  — historical bug in lib types
# // Fix at the boundary:
# const v: unknown = JSON.parse(s);
# // Or use a parser library: z.parse(JSON.parse(s))
```

## Performance Tips

```bash
# - Avoid `any` — every `any` is a hole in the type wall. Use `unknown` and narrow.
# - Narrow EARLY — if a value enters as `unknown`, validate it at the boundary, not deep in calls.
# - Project references for monorepos — split into composite tsconfigs:
#   { "compilerOptions": { "composite": true }, "references": [{ "path": "../shared" }] }
#   tsc --build is incremental and parallel.
# - tsc --noEmit in CI — catches type errors without writing files.
# - skipLibCheck: true — node_modules type-check eats minutes on big monorepos.
# - incremental: true + tsBuildInfoFile — reuse type info across runs.
# - Avoid huge union types in hot paths — narrow types help inference and speed.
# - Watch tsc --extendedDiagnostics for "checkTime" hotspots when builds get slow.
```

## Idioms

### Result-style typing for errors

```bash
# // Go-style ok/err pair:
# type Ok<T> = { ok: true; value: T };
# type Err<E> = { ok: false; error: E };
# type Result<T, E = Error> = Ok<T> | Err<E>;
#
# function parseInt2(s: string): Result<number, string> {
#   const n = Number(s);
#   return Number.isNaN(n) ? { ok: false, error: "not a number" } : { ok: true, value: n };
# }
#
# const r = parseInt2("42");
# if (r.ok) r.value.toFixed(2);    // narrowed to number
# else      console.error(r.error);
```

### Branded types for nominal-ish typing

```bash
# // Plain strings collide structurally — give them brands:
# type Brand<K, T> = K & { readonly __brand: T };
# type UserId  = Brand<string, "UserId">;
# type OrderId = Brand<string, "OrderId">;
#
# function userId(s: string): UserId  { return s as UserId; }
# function orderId(s: string): OrderId { return s as OrderId; }
#
# function getUser(id: UserId) {}
# getUser(userId("u_1"));         // OK
# // getUser(orderId("o_1"));     // ERROR — distinct brand
```

### Exhaustiveness checks with `never`

```bash
# function area(s: Shape): number {
#   switch (s.kind) {
#     case "circle": return Math.PI * s.radius ** 2;
#     case "rect":   return s.width * s.height;
#     default: { const _: never = s; throw new Error(`unhandled: ${_}`); }
#   }
# }
# // Add a new variant — TS errors at the assignment to never. Safety with no test required.
```

### `using` for explicit resource management (TS 5.2+)

```bash
# // ES2024-style — auto-disposes at scope end (LIFO order):
# class FileHandle {
#   [Symbol.dispose]() { this.close(); }
#   close() {}
# }
# function open(p: string): FileHandle { return new FileHandle(); }
#
# {
#   using f = open("data.txt");
#   // ... use f
# }                              // f[Symbol.dispose]() runs here automatically
#
# // Async variant: await using f = openAsync(...);   uses Symbol.asyncDispose
```

## Tips

- Turn on every strict flag (`strict`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitOverride`). Once you live with them, you can't go back.
- Prefer `unknown` over `any`. Treat `any` as a code smell — every `any` should be a TODO comment.
- Use `satisfies` to validate-without-widening. It's the operator that finally fixed config typing.
- Discriminated unions with a `kind:` tag and a `never` exhaustiveness check are the idiomatic way to model variants in TS.
- Avoid enums in libraries — use string literal unions for tree-shaking and JSON friendliness.
- `as const` is the fastest way to lock a literal value's narrow type through inference.
- For runtime data validation (HTTP bodies, env vars, user input), use a library — zod, valibot, arktype.
- `tsc --noEmit` belongs in every CI job. The fast transpiler emits, tsc checks.
- Branded types give you nominal-style safety on top of a structural system, with zero runtime cost.
- Read the TS release notes — every minor version (5.x) ships features worth adopting.

## See Also

- javascript
- polyglot
- rust
- go
- python
- bash
- java
- ruby
- lua
- c
- make
- webassembly

## References

- [TypeScript Documentation](https://www.typescriptlang.org/docs/) -- official docs and getting started
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html) -- core language guide
- [TypeScript Playground](https://www.typescriptlang.org/play) -- run and share TypeScript online
- [tsconfig Reference](https://www.typescriptlang.org/tsconfig) -- every compiler option explained
- [TypeScript Cheat Sheets](https://www.typescriptlang.org/cheatsheets) -- official visual quick references
- [TypeScript Release Notes](https://www.typescriptlang.org/docs/handbook/release-notes/overview.html) -- per-version changelog (3.x through 5.x)
- [TypeScript GitHub](https://github.com/microsoft/TypeScript) -- source code, issues, design discussions
- [DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped) -- community type definitions for npm packages
- [Type Challenges](https://github.com/type-challenges/type-challenges) -- collection of TypeScript type puzzles
- [Total TypeScript](https://www.totaltypescript.com/) -- Matt Pocock's deep-dive tutorials and tips
- [TC39 Proposals](https://github.com/tc39/proposals) -- upcoming JavaScript features that flow into TS
- [Temporal Proposal](https://tc39.es/proposal-temporal/docs/) -- modern date/time API (stage 3)
- [Zod](https://zod.dev/) -- runtime schema validation with type inference
- [Vitest](https://vitest.dev/) -- modern Vite-native test runner with TS support
- [Bun Documentation](https://bun.sh/docs) -- runtime, bundler, test runner with native TS
- [Deno Manual](https://docs.deno.com/) -- secure runtime with first-class TypeScript
- [Node TypeScript Docs](https://nodejs.org/en/learn/typescript/introduction) -- official Node guide for TS
- [esbuild](https://esbuild.github.io/) -- the Go-based fast bundler/transpiler
- [SWC](https://swc.rs/) -- the Rust-based fast transpiler
- [npm Registry](https://www.npmjs.com/) -- package registry (TypeScript and JavaScript)
