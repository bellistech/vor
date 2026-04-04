# JavaScript (Dynamic, Multi-Paradigm Programming Language)

The language of the web, running in browsers and on servers via Node.js.

## Variables and Data Types

### Variable Declarations

```javascript
// const - block-scoped, cannot be reassigned (preferred)
const PI = 3.14159;

// let - block-scoped, can be reassigned
let count = 0;
count = 1;

// var - function-scoped, hoisted (avoid in modern code)
var legacy = "old style";
```

### Data Types

```javascript
// Primitives
const str = "hello";           // string
const num = 42;                // number (integer and float are the same type)
const big = 9007199254740993n; // BigInt
const bool = true;             // boolean
const nothing = null;          // null (intentional absence)
let undef;                     // undefined (uninitialized)
const sym = Symbol("id");      // symbol (unique identifier)

// Type checking
typeof "hello";                // "string"
typeof 42;                     // "number"
typeof true;                   // "boolean"
typeof undefined;              // "undefined"
typeof null;                   // "object" (historical bug)
Array.isArray([1, 2]);         // true
```

## Strings

```javascript
// Template literals (backticks for interpolation and multiline)
const name = "world";
const greeting = `Hello, ${name}!`;
const multiline = `Line one
Line two`;

// Common methods
"hello".toUpperCase();                 // "HELLO"
"HELLO".toLowerCase();                 // "hello"
"hello world".includes("world");       // true
"hello world".startsWith("hello");     // true
"hello world".indexOf("world");        // 6
"hello world".slice(0, 5);             // "hello"
"hello world".split(" ");              // ["hello", "world"]
"  hello  ".trim();                    // "hello"
"ha".repeat(3);                        // "hahaha"
"hello world".replace("world", "JS");  // "hello JS"
"a.b.c".replaceAll(".", "/");          // "a/b/c"
"hello".padStart(10, ".");             // ".....hello"
```

## Arrays

```javascript
// Creation
const arr = [1, 2, 3, 4, 5];
const filled = Array(5).fill(0);       // [0, 0, 0, 0, 0]
const range = Array.from({ length: 5 }, (_, i) => i + 1); // [1, 2, 3, 4, 5]

// Transformation (return new arrays)
const doubled = arr.map(n => n * 2);           // [2, 4, 6, 8, 10]
const evens = arr.filter(n => n % 2 === 0);    // [2, 4]
const sum = arr.reduce((acc, n) => acc + n, 0); // 15
const flat = [[1, 2], [3, 4]].flat();          // [1, 2, 3, 4]
const mapped = arr.flatMap(n => [n, n * 2]);   // [1, 2, 2, 4, 3, 6, ...]

// Search
arr.find(n => n > 3);                  // 4 (first match)
arr.findIndex(n => n > 3);             // 3 (index of first match)
arr.includes(3);                       // true
arr.some(n => n > 4);                  // true (at least one)
arr.every(n => n > 0);                 // true (all match)

// Mutation
arr.push(6);                           // append to end
arr.pop();                             // remove from end
arr.unshift(0);                        // prepend to start
arr.shift();                           // remove from start
arr.splice(1, 2);                      // remove 2 items at index 1
arr.sort((a, b) => a - b);            // sort numerically ascending

// Iteration
arr.forEach(n => console.log(n));      // no return value

// Spread (shallow copy and merge)
const copy = [...arr];
const merged = [...arr, ...evens];
```

## Objects

```javascript
// Object literal
const person = { name: "Alice", age: 30, role: "admin" };

// Computed property keys
const key = "status";
const obj = { [key]: "active" };       // { status: "active" }

// Shorthand (variable name matches key)
const name = "Alice";
const user = { name };                 // { name: "Alice" }

// Destructuring
const { name: userName, age, role = "user" } = person; // default value for role

// Spread (shallow copy and merge)
const updated = { ...person, age: 31 };

// Common operations
Object.keys(person);                   // ["name", "age", "role"]
Object.values(person);                 // ["Alice", 30, "admin"]
Object.entries(person);                // [["name","Alice"], ["age",30], ...]
Object.fromEntries([["a", 1]]);        // { a: 1 }
Object.assign({}, person, { age: 31 }); // merge (older style)
"name" in person;                      // true
person.hasOwnProperty("name");         // true

// Optional chaining and nullish coalescing
const city = person?.address?.city;             // undefined (no error)
const port = config.port ?? 3000;               // 3000 if port is null/undefined
const len = person?.hobbies?.length ?? 0;       // safe nested access with default
```

## Functions

```javascript
// Function declaration (hoisted)
function greet(name) {
  return `Hello, ${name}!`;
}

// Arrow function (concise, lexical `this`)
const greet = (name) => `Hello, ${name}!`;

// Default parameters
const greet = (name = "world") => `Hello, ${name}!`;

// Rest parameters (variable arguments)
const sum = (...nums) => nums.reduce((a, b) => a + b, 0);

// Destructured parameters
const fullName = ({ first, last }) => `${first} ${last}`;
fullName({ first: "Jane", last: "Doe" });

// Immediately invoked function expression (IIFE)
(() => {
  console.log("Runs immediately");
})();

// Higher-order function (takes or returns a function)
const multiplier = (factor) => (n) => n * factor;
const double = multiplier(2);
double(5);                             // 10
```

## Promises and Async/Await

### Promises

```javascript
// Creating a promise
const fetchData = () => new Promise((resolve, reject) => {
  setTimeout(() => resolve("data"), 1000);
});

// Chaining
fetchData()
  .then(data => data.toUpperCase())
  .then(result => console.log(result))
  .catch(err => console.error(err))
  .finally(() => console.log("Done"));

// Combinators
Promise.all([p1, p2, p3]);            // resolves when ALL resolve, rejects on first rejection
Promise.race([p1, p2]);               // resolves/rejects with first settled
Promise.allSettled([p1, p2]);          // waits for all, never rejects
Promise.any([p1, p2]);                // resolves with first fulfilled, rejects if all reject
```

### Async/Await

```javascript
// Async function (always returns a promise)
async function loadUser(id) {
  try {
    const res = await fetch(`/api/users/${id}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const user = await res.json();
    return user;
  } catch (err) {
    console.error("Failed to load user:", err);
    throw err;
  }
}

// Parallel execution with async/await
const [users, posts] = await Promise.all([
  fetch("/api/users").then(r => r.json()),
  fetch("/api/posts").then(r => r.json()),
]);

// Top-level await (in ES modules)
const config = await fetch("/config.json").then(r => r.json());
```

## Classes

```javascript
// Class declaration
class Animal {
  #sound;                              // private field

  constructor(name, sound) {
    this.name = name;
    this.#sound = sound;
  }

  speak() {
    return `${this.name} says ${this.#sound}`;
  }

  static create(name, sound) {         // static method
    return new Animal(name, sound);
  }
}

// Inheritance
class Dog extends Animal {
  constructor(name) {
    super(name, "Woof");               // call parent constructor
  }

  fetch(item) {
    return `${this.name} fetches the ${item}`;
  }
}

const rex = new Dog("Rex");
rex.speak();                           // "Rex says Woof"
rex instanceof Animal;                 // true
```

## Modules

```javascript
// Named exports (math.js)
export const PI = 3.14159;
export function add(a, b) { return a + b; }

// Default export (logger.js)
export default class Logger {
  log(msg) { console.log(msg); }
}

// Named imports
import { PI, add } from "./math.js";

// Default import
import Logger from "./logger.js";

// Rename on import
import { add as sum } from "./math.js";

// Import all as namespace
import * as math from "./math.js";
math.add(1, 2);

// Dynamic import (code splitting)
const module = await import("./heavy-module.js");
```

## Error Handling

```javascript
// try / catch / finally
try {
  JSON.parse("invalid json");
} catch (err) {
  console.error(err.message);          // "Unexpected token i..."
} finally {
  console.log("Cleanup");
}

// Custom error class
class ValidationError extends Error {
  constructor(field, message) {
    super(message);
    this.name = "ValidationError";
    this.field = field;
  }
}

// Throw and catch custom errors
try {
  throw new ValidationError("email", "Invalid email format");
} catch (err) {
  if (err instanceof ValidationError) {
    console.error(`${err.field}: ${err.message}`);
  } else {
    throw err;                         // re-throw unexpected errors
  }
}
```

## DOM Manipulation

```javascript
// Selecting elements
const el = document.getElementById("app");
const el = document.querySelector(".card");          // first match
const els = document.querySelectorAll(".card");       // all matches (NodeList)

// Creating and appending
const div = document.createElement("div");
div.textContent = "Hello";
div.classList.add("card");
document.body.appendChild(div);

// Modifying
el.innerHTML = "<strong>Bold</strong>";               // set HTML (be careful with XSS)
el.textContent = "Safe text";                         // set text only
el.setAttribute("data-id", "42");
el.style.color = "red";
el.classList.toggle("active");

// Events
el.addEventListener("click", (e) => {
  e.preventDefault();
  console.log("Clicked", e.target);
});
```

## Fetch API and JSON

```javascript
// GET request
const res = await fetch("/api/users");
const users = await res.json();

// POST request
const res = await fetch("/api/users", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ name: "Alice", age: 30 }),
});

// JSON utilities
const str = JSON.stringify({ a: 1 }, null, 2);        // pretty-print with 2-space indent
const obj = JSON.parse('{"a": 1}');
```

## Set and Map

```javascript
// Set (unique values)
const s = new Set([1, 2, 3, 2, 1]);   // Set {1, 2, 3}
s.add(4);
s.has(3);                              // true
s.delete(2);
s.size;                                // 3
const unique = [...new Set(arr)];      // deduplicate an array

// Map (key-value, any type as key)
const m = new Map();
m.set("name", "Alice");
m.set(42, "the answer");
m.get("name");                         // "Alice"
m.has(42);                             // true
m.size;                                // 2
for (const [key, val] of m) { /* iterate */ }
```

## Generators

```javascript
// Generator function (yields values lazily)
function* range(start, end) {
  for (let i = start; i <= end; i++) {
    yield i;
  }
}

const gen = range(1, 5);
gen.next();                            // { value: 1, done: false }
gen.next();                            // { value: 2, done: false }

// Spread a generator into an array
const nums = [...range(1, 5)];         // [1, 2, 3, 4, 5]
```

## Common Patterns

```javascript
// Debounce (delay execution until idle)
const debounce = (fn, ms) => {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  };
};

// Deep clone (structured clone, modern)
const clone = structuredClone(original);

// Grouping objects by a key
const grouped = Object.groupBy(users, u => u.department);

// Sleep / delay
const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));
await sleep(1000);
```

## Tips

- Always use `const` by default; switch to `let` only when reassignment is needed. Avoid `var`.
- Use `===` (strict equality) instead of `==` to avoid type coercion surprises.
- Arrow functions do not have their own `this`; they inherit it from the enclosing scope.
- `Array.from()` converts array-like objects (NodeList, arguments) into real arrays.
- Use optional chaining (`?.`) and nullish coalescing (`??`) to safely handle null/undefined values.
- `Promise.allSettled()` is safer than `Promise.all()` when you want results regardless of individual failures.
- Prefer `for...of` for iterating arrays and `for...in` for object keys (but `Object.keys()` is often clearer).
- Use `structuredClone()` instead of `JSON.parse(JSON.stringify())` for deep cloning with proper handling of dates, maps, and sets.
- Template literals support tagged templates for custom string processing (e.g., `html\`<p>${text}</p>\``).
- Private class fields (`#field`) are enforced at the language level, unlike the underscore convention.
- In Node.js, use `import`/`export` with `"type": "module"` in `package.json`, or `.mjs` file extensions.

## See Also

- typescript, json, html, css, npm, regex

## References

- [MDN JavaScript Reference](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference) -- comprehensive API and language docs
- [MDN JavaScript Guide](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide) -- tutorials from basics to advanced
- [ECMA-262 Specification](https://tc39.es/ecma262/) -- the living ECMAScript standard
- [TC39 Proposals](https://github.com/tc39/proposals) -- upcoming language features and their stages
- [Node.js Documentation](https://nodejs.org/docs/latest/api/) -- Node.js API reference
- [npm Registry](https://www.npmjs.com/) -- package registry and search
- [V8 Blog](https://v8.dev/blog) -- JavaScript engine internals and performance
- [Can I Use](https://caniuse.com/) -- browser compatibility tables for JS and Web APIs
- [JavaScript.info](https://javascript.info/) -- modern JavaScript tutorial
- [ECMAScript Compatibility Table](https://compat-table.github.io/compat-table/es6/) -- feature support across engines
