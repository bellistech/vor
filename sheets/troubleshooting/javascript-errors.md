# JavaScript Errors

Verbatim JS / Node / TypeScript error messages with cause and fix. Browser, Node, framework, bundler, and tooling. Covers V8 wording, the strict-mode variants, and the SpiderMonkey/JavaScriptCore alternates.

## Setup

JavaScript exposes a small native error hierarchy. All thrown errors are usually instances of one of these classes:

```text
Error
 |- TypeError          (operation on a value of wrong type)
 |- ReferenceError     (referencing an undeclared identifier)
 |- SyntaxError        (parsing failure: thrown by parser, not user code)
 |- RangeError         (numeric / size out of range)
 |- URIError           (decodeURI[Component]/encodeURI[Component] malformed)
 |- EvalError          (legacy; almost never thrown by modern engines)
 |- AggregateError     (Promise.any when all reject; collects .errors)
```

Every Error has three standard own properties.

```javascript
const e = new TypeError("bad input");
e.name;     // "TypeError"
e.message;  // "bad input"
e.stack;    // multi-line string, V8-formatted
```

V8 (Node, Chrome, Edge) `.stack` looks like:

```text
TypeError: Cannot read properties of undefined (reading 'foo')
    at handle (/app/src/handler.js:42:18)
    at Server.<anonymous> (/app/src/server.js:88:12)
    at Server.emit (node:events:514:28)
```

SpiderMonkey (Firefox) `.stack` is newline-separated `function@file:line:col`:

```text
handle@file:///app/src/handler.js:42:18
@file:///app/src/server.js:88:12
```

JavaScriptCore (Safari) is a hybrid; messages are mostly the same as V8 with a slightly different tone (`undefined is not an object (evaluating 'x.foo')`).

Read a stack trace top-down: the topmost line is the throwing site, each subsequent line is its caller. Async stacks (Node 12+) include `async` markers:

```text
TypeError: Cannot read properties of null (reading 'id')
    at getUser (/app/db.js:18:11)
    at async loadProfile (/app/profile.js:7:18)
```

Source maps map a minified position back to the original. Browsers honour `//# sourceMappingURL=app.js.map` automatically when DevTools is open. Node honours `--enable-source-maps` (Node 12.12+) for stack traces of transpiled code.

```bash
node --enable-source-maps dist/server.js
```

`Error.captureStackTrace(target, callerFn)` (V8-only) attaches `.stack` to a target object, optionally pruning the calling frame from the trace.

```javascript
class AppError extends Error {
  constructor(msg) {
    super(msg);
    this.name = "AppError";
    Error.captureStackTrace?.(this, AppError);
  }
}
```

`Error.stackTraceLimit` (V8-only) sets the depth (default 10).

```javascript
Error.stackTraceLimit = 100;
```

`error.cause` (ES2022) chains a wrapped error.

```javascript
try { JSON.parse(s); }
catch (cause) { throw new Error("bad config", { cause }); }
```

In Node 16.9+ the formatter prints `[cause]: ...` underneath.

## TypeError

`TypeError` is thrown when an operation is performed on a value of an inappropriate type — calling a non-function, dereferencing null/undefined, modifying a frozen object, etc. The verbatim wording differs slightly by engine and Node version.

### Cannot read properties of undefined / null

V8 (modern, Node 16+):

```text
TypeError: Cannot read properties of undefined (reading 'foo')
TypeError: Cannot read properties of null (reading 'foo')
```

V8 (legacy, Node 14 and older):

```text
TypeError: Cannot read property 'foo' of undefined
TypeError: Cannot read property 'foo' of null
```

SpiderMonkey:

```text
TypeError: x is undefined
TypeError: x is null
```

JavaScriptCore:

```text
TypeError: undefined is not an object (evaluating 'x.foo')
```

Cause: dereferencing a missing value. Usually because an API returned a different shape than expected.

```javascript
const data = await fetch("/api/user").then(r => r.json());
console.log(data.user.name); // throws if data.user is undefined
```

Fix — optional chaining (ES2020):

```javascript
console.log(data?.user?.name ?? "anonymous");
```

Fix — guard:

```javascript
if (data && data.user) console.log(data.user.name);
```

Fix — destructuring with default:

```javascript
const { user = {} } = data ?? {};
const { name = "anonymous" } = user;
```

### TypeError: Cannot read properties of undefined (reading 'map')

A specific instance of the above; almost always means an API returned `null` (or `{}`) when an array was expected.

```javascript
// Broken
const items = await fetch("/api/items").then(r => r.json());
return items.map(renderItem);
```

Fix:

```javascript
const { items = [] } = await fetch("/api/items").then(r => r.json());
return items.map(renderItem);
```

### X is not a function

```text
TypeError: x.foo is not a function
TypeError: undefined is not a function
```

Causes:

- Method does not exist (typo): `arr.foreach(...)` instead of `arr.forEach`.
- Lost `this` binding when extracting a method.
- Importing default vs named: `import lodash from "lodash"` then calling `lodash(...)`.

Broken — lost this:

```javascript
class Counter { inc() { this.n++; } }
const c = new Counter(); c.n = 0;
const f = c.inc; f(); // TypeError: Cannot read properties of undefined (reading 'n')
```

Fix:

```javascript
const f = c.inc.bind(c);
// or
const f = () => c.inc();
```

### X is not iterable

```text
TypeError: x is not iterable
TypeError: undefined is not iterable (cannot read property Symbol(Symbol.iterator))
```

Cause: `for...of` or spread on something that is not iterable.

```javascript
const obj = { a: 1, b: 2 };
for (const v of obj) {} // throws
```

Fix — iterate keys / values / entries:

```javascript
for (const v of Object.values(obj)) {}
for (const [k, v] of Object.entries(obj)) {}
```

### Cannot assign to read only property

```text
TypeError: Cannot assign to read only property 'foo' of object '#<Object>'
```

Cause: assigning to a frozen, sealed, or non-writable property; common in strict mode.

```javascript
"use strict";
const o = Object.freeze({ x: 1 });
o.x = 2; // throws in strict; silent in sloppy
```

Fix — clone:

```javascript
const o2 = { ...o, x: 2 };
```

### Cannot redefine property

```text
TypeError: Cannot redefine property: foo
```

Cause: `Object.defineProperty` on a non-configurable property.

```javascript
Object.defineProperty(window, "Symbol", { value: undefined }); // throws
```

Fix — define on a fresh object, or use `configurable: true` initially.

### Cannot convert undefined or null to object

```text
TypeError: Cannot convert undefined or null to object
```

Cause: `Object.keys(undefined)`, `Object.assign(target, undefined)`, spreading `null`.

```javascript
const out = { ...x }; // OK in ES2018+ even if x is null
const keys = Object.keys(x); // throws if x is null/undefined
```

Fix:

```javascript
const keys = Object.keys(x ?? {});
```

### Assignment to constant variable

```text
TypeError: Assignment to constant variable.
```

```javascript
const x = 1; x = 2; // throws
```

Fix: use `let`. Note — `const` only forbids rebinding, not mutation.

### X is not defined (strict mode)

```text
TypeError: 'caller' is not defined
TypeError: 'arguments' is not defined
```

Cause: accessing `arguments`, `caller`, `callee` in strict mode where they are forbidden. Usually appears as a runtime `ReferenceError` when a name is undeclared; in strict-mode functions or ES modules certain names are deliberately unavailable.

### Reduce of empty array with no initial value

```text
TypeError: Reduce of empty array with no initial value
```

```javascript
[].reduce((a, b) => a + b); // throws
```

Fix — pass initial value:

```javascript
[].reduce((a, b) => a + b, 0); // 0
```

### Class constructor cannot be invoked without 'new'

```text
TypeError: Class constructor X cannot be invoked without 'new'
```

```javascript
class Foo {}
Foo(); // throws — must be `new Foo()`
```

Fix — always `new`. If you have a function that should also work without `new`, use a regular function, not a class.

### Invalid attempt to spread non-iterable instance

```text
TypeError: Found non-callable @@iterator
TypeError: Spread syntax requires ...iterable[Symbol.iterator] to be a function
```

```javascript
const o = { a: 1 };
const arr = [...o]; // throws
```

Fix:

```javascript
const arr = [...Object.values(o)];
```

### Converting circular structure to JSON

```text
TypeError: Converting circular structure to JSON
    --> starting at object with constructor 'Object'
    |     property 'self' -> object with constructor 'Object'
    --- property 'self' closes the circle
```

Cause: `JSON.stringify` on an object with a cycle.

Fix — replacer that drops cycles:

```javascript
function safeStringify(obj) {
  const seen = new WeakSet();
  return JSON.stringify(obj, (k, v) => {
    if (typeof v === "object" && v !== null) {
      if (seen.has(v)) return "[Circular]";
      seen.add(v);
    }
    return v;
  });
}
```

Or `util.inspect(obj)` in Node which handles cycles natively.

## ReferenceError

A `ReferenceError` is thrown when an unknown identifier is referenced.

### X is not defined

```text
ReferenceError: foo is not defined
```

Causes:

- Typo on a variable name.
- Forgot to import.
- Using a browser-only global in Node, or vice versa.

Fix — import or declare:

```javascript
import { foo } from "./mod.js";
// or
const foo = require("./mod");
```

### Cannot access 'X' before initialization

```text
ReferenceError: Cannot access 'foo' before initialization
```

Cause: temporal dead zone — referencing a `let`/`const` before its declaration in the same scope.

```javascript
console.log(foo); // throws TDZ
let foo = 1;
```

Fix — declare first, or move the access after the declaration. Note this differs from `var`, which would print `undefined`.

### Assignment to undeclared variable (strict mode)

```text
ReferenceError: assignment to undeclared variable foo
```

In strict mode (and ES modules) implicit globals are forbidden.

```javascript
"use strict";
foo = 1; // throws
```

Fix — declare with `let` / `const` / `var`.

### __dirname is not defined in ES module scope

```text
ReferenceError: __dirname is not defined in ES module scope
```

Cause: `__dirname` and `__filename` only exist in CommonJS. In ES modules you must derive them.

```javascript
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";
const __filename = fileURLToPath(import.meta.url);
const __dirname  = dirname(__filename);
```

Or in Node 20.11+ / 21.2+:

```javascript
const __dirname = import.meta.dirname;
```

### require is not defined in ES module scope

```text
ReferenceError: require is not defined in ES module scope, you can use import instead
```

Cause: file is treated as ESM (extension `.mjs` or `package.json` has `"type": "module"`), but uses `require`.

Fix — switch to `import`, or rename the file to `.cjs`, or:

```javascript
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
```

### process is not defined

```text
ReferenceError: process is not defined
```

Cause: running browser bundle that references `process.env.NODE_ENV` (typical when bundler doesn't inline it).

Fix — webpack `DefinePlugin`, vite injects `import.meta.env` instead, or shim:

```javascript
// vite.config.js
define: { "process.env": {} }
```

### window is not defined

```text
ReferenceError: window is not defined
```

Cause: accessing `window` during SSR (Next.js, Remix) where Node has no DOM.

Fix — guard:

```javascript
if (typeof window !== "undefined") { /* browser-only */ }
```

In Next.js, restrict to client component:

```javascript
"use client";
import { useEffect } from "react";
useEffect(() => { console.log(window.innerWidth); }, []);
```

### document is not defined

Same family as `window`. Same fix.

### globalThis fallback

For environment-agnostic globals:

```javascript
const g = globalThis; // works in browser, Node, workers
```

## SyntaxError

Thrown by the parser, before any code runs. Cannot be caught by a `try/catch` in the same file (the file never starts executing). Caught for `JSON.parse`, `eval`, and dynamic `new Function(...)`.

### Unexpected token X

```text
SyntaxError: Unexpected token '}'
SyntaxError: Unexpected token 'export'
SyntaxError: Unexpected token <
```

Cause: parser hit a token it didn't expect. Common reasons:

- Missing brace / paren / bracket.
- ESM syntax (`import`/`export`) in a CJS file.
- HTML response parsed as JSON (the `<` is the start of `<!DOCTYPE html>`).
- TypeScript / JSX in a plain `.js` file run by Node directly.

### Unexpected token < in JSON at position 0

```text
SyntaxError: Unexpected token < in JSON at position 0
```

Cause: server returned HTML (often a 404 page) but client called `.json()`.

Fix — check status / content type first:

```javascript
const r = await fetch(url);
if (!r.ok) throw new Error(`HTTP ${r.status}`);
const ct = r.headers.get("content-type") ?? "";
if (!ct.includes("application/json")) throw new Error(`bad CT: ${ct}`);
const data = await r.json();
```

### Unexpected end of JSON input

```text
SyntaxError: Unexpected end of JSON input
```

Cause: empty string or truncated body.

```javascript
JSON.parse(""); // throws
```

Fix:

```javascript
const text = await r.text();
const data = text ? JSON.parse(text) : null;
```

### Unexpected non-whitespace character after JSON at position N

```text
SyntaxError: Unexpected non-whitespace character after JSON at position 23
```

Cause: trailing data after a valid JSON object — most often two JSON objects concatenated (NDJSON misread as JSON).

Fix — split by `\n` and parse each line, or use a streaming parser like `JSONStream`.

### Unexpected end of input

```text
SyntaxError: Unexpected end of input
```

Cause: source ends mid-statement — unclosed `{`, `(`, or string. The line/col in the error is approximate.

### Identifier 'X' has already been declared

```text
SyntaxError: Identifier 'foo' has already been declared
```

Cause: same `let`/`const` name twice in the same scope.

```javascript
const foo = 1;
const foo = 2; // throws
```

Note: `var` redeclaration is allowed.

### missing ) after argument list

```text
SyntaxError: missing ) after argument list
```

Cause: unbalanced parentheses in a call.

```javascript
console.log("hello"; // missing )
```

### await is only valid in async functions and the top level bodies of modules

```text
SyntaxError: await is only valid in async functions and the top level bodies of modules
```

Cause: `await` in a function not marked `async`, or in a CommonJS file (top-level await is only allowed in ES modules).

Fix:

```javascript
async function run() {
  const x = await fetch("/x");
}
```

For top-level await, use ESM (`.mjs` or `"type": "module"`).

### Cannot use import statement outside a module

```text
SyntaxError: Cannot use import statement outside a module
```

Causes:

- Running a script with `import` syntax under CommonJS rules.
- Browser `<script>` tag missing `type="module"`.

Fix — Node:

```json
// package.json
{ "type": "module" }
```

Or rename the file `.mjs`.

Fix — browser:

```html
<script type="module" src="./app.js"></script>
```

### Octal literals are not allowed in strict mode

```text
SyntaxError: Octal literals are not allowed in strict mode.
```

Cause: leading-zero octal like `0755`.

Fix — use `0o755` (ES6 octal prefix) or decimal.

### Unexpected reserved word

```text
SyntaxError: Unexpected reserved word 'await'
SyntaxError: Unexpected reserved word 'enum'
```

Cause: using a reserved keyword as a name. Note `await`, `async`, `let`, `static`, `yield`, `enum`, `implements`, `private`, `protected`, `public`, `interface`, `package` are reserved in some contexts.

### Invalid or unexpected token

```text
SyntaxError: Invalid or unexpected token
```

Cause: unrecognised character — usually a stray smart-quote (`"` `"` `'` `'`) instead of a straight quote, or a non-printing character pasted in.

### Unterminated template literal

```text
SyntaxError: Unterminated template literal
```

Cause: missing closing backtick.

```javascript
const s = `hello;
```

## RangeError

Numeric or size value out of valid range.

### Maximum call stack size exceeded

```text
RangeError: Maximum call stack size exceeded
```

Cause: unbounded recursion (often mutual recursion or accidental infinite loop in proxies / getters).

```javascript
function f() { return f(); }
f(); // throws
```

Fix — base case, or convert to iteration. Default V8 stack is ~10–11k frames.

### Invalid array length

```text
RangeError: Invalid array length
```

```javascript
new Array(-1);     // throws
new Array(2 ** 32);// throws (max is 2^32 - 1)
[].length = -1;    // throws
```

Fix — clamp to a valid 32-bit unsigned integer.

### Invalid string length

```text
RangeError: Invalid string length
```

V8 caps strings at ~2^29 - 24 bytes (~512 MB). Concatenating beyond that throws.

Fix — write to a stream (`fs.createWriteStream`) instead of buffering.

### Invalid time value

```text
RangeError: Invalid time value
```

Cause: calling `.toISOString()` on an invalid `Date`.

```javascript
new Date("nope").toISOString(); // throws
```

Fix — check first:

```javascript
const d = new Date(s);
if (Number.isNaN(d.getTime())) throw new Error("bad date");
```

### toFixed argument must be between 0 and 100

```text
RangeError: toFixed() digits argument must be between 0 and 100
```

```javascript
(1.234).toFixed(101); // throws
```

### Invalid count value

```text
RangeError: Invalid count value
```

```javascript
"x".repeat(-1);
"x".repeat(Number.POSITIVE_INFINITY);
```

### Invalid array buffer length

```text
RangeError: Invalid array buffer length
```

```javascript
new ArrayBuffer(-1);
new Uint8Array(2 ** 31); // exceeds typed-array max element count
```

## URIError

Thrown by URI handling functions on malformed input.

### URI malformed

```text
URIError: URI malformed
```

```javascript
decodeURIComponent("%E0%A4%A"); // throws (incomplete sequence)
decodeURIComponent("%ZZ");      // throws (not hex)
```

Fix — encode first, or guard:

```javascript
function safeDecode(s) {
  try { return decodeURIComponent(s); }
  catch { return s; }
}
```

The asymmetry: `encodeURIComponent` accepts any string, `decodeURIComponent` requires valid `%XX` triplets where XX is hex.

## EvalError

`EvalError` is essentially historical; modern engines do not throw it. The class still exists in the spec for backward-compatibility with very old code that called `eval` in unusual ways. You may construct one (`new EvalError("x")`) but you will not see one from runtime built-ins.

## AggregateError

Introduced in ES2021 with `Promise.any`. Wraps multiple errors.

```text
AggregateError: All promises were rejected
```

```javascript
try {
  await Promise.any([
    Promise.reject(new Error("a")),
    Promise.reject(new Error("b")),
  ]);
} catch (e) {
  e.name;     // "AggregateError"
  e.errors;   // [Error: a, Error: b]
}
```

Fix — inspect `.errors` to surface the underlying causes.

You can also create your own:

```javascript
throw new AggregateError([new Error("x"), new Error("y")], "multi");
```

## Promise / async errors

### Unhandled promise rejection

Node:

```text
(node:1234) UnhandledPromiseRejectionWarning: Error: boom
(node:1234) [DEP0018] DeprecationWarning: Unhandled promise rejections are deprecated.
    In the future, promise rejections that are not handled will terminate the
    Node.js process with a non-zero exit code.
```

Modern Node (15+):

```text
node:internal/process/promises:288
            triggerUncaughtException(err, true /* fromPromise */);
            ^
Error: boom
```

By default Node 15+ exits with code 1 on unhandled rejection.

Browser console:

```text
Uncaught (in promise) Error: boom
```

Cause: a rejected promise without `.catch` or `try/await/catch`.

```javascript
fetch("/x"); // no .catch — rejection becomes "unhandled"
```

Fix:

```javascript
fetch("/x").catch(err => log.error({ err }, "fetch failed"));
```

Or attach a top-level handler (last resort, log only):

```javascript
process.on("unhandledRejection", (reason, promise) => {
  console.error("unhandled rejection:", reason);
});
```

Control behaviour with `--unhandled-rejections=`:

```bash
node --unhandled-rejections=strict app.js   # exit on first (default since 15)
node --unhandled-rejections=warn app.js     # warn only
node --unhandled-rejections=throw app.js    # uncaught exception
node --unhandled-rejections=none app.js     # silent
```

### Promise.any: All promises were rejected

Same as `AggregateError` above. Always handle:

```javascript
try {
  const winner = await Promise.any(reqs);
} catch (e) {
  console.error("all failed", e.errors);
}
```

### Promise.all is not a function

```text
TypeError: Promise.all is not a function
```

Cause: rare; only in extremely old runtimes (IE) or when a library overrode `Promise`. Fix — load a polyfill (`core-js`) or use a real `Promise`.

### Cannot await non-thenable

There is no error literally named "Cannot await non-thenable" — `await` of a non-thenable simply resolves to that value. This is a footgun, not an error:

```javascript
async function run() {
  const v = await 42; // v === 42
  const v2 = await { not: "a promise" }; // v2 === { not: "a promise" }
}
```

If you forgot to call the async function:

```javascript
const data = await fetchData; // awaits the function value, not the call
```

Fix — call it:

```javascript
const data = await fetchData();
```

### Forgot await on async function call

No error, but you have a `Promise` where you expected the value:

```javascript
async function getName() { return "Alice"; }
const name = getName();         // Promise { 'Alice' }
console.log("hello " + name);   // "hello [object Promise]"
```

Fix:

```javascript
const name = await getName();
```

### Errors swallowed in non-awaited async

```javascript
async function main() {
  doWork(); // forgot await — rejection becomes unhandled later
}
```

Fix — `await` or chain `.catch`.

## Module loader errors (Node)

### Cannot find module

```text
Error [ERR_MODULE_NOT_FOUND]: Cannot find module '/app/missing.js' imported from /app/index.js
Error: Cannot find module 'lodash'
Require stack:
- /app/index.js
```

Causes: not installed, wrong path, missing `.js` extension in ESM.

Fix:

```bash
npm install lodash
```

ESM **requires** the file extension:

```javascript
import { foo } from "./mod";    // ERR_MODULE_NOT_FOUND
import { foo } from "./mod.js"; // OK
```

### require() of ES Module is not supported

```text
Error [ERR_REQUIRE_ESM]: require() of ES Module /app/node_modules/chalk/source/index.js
from /app/index.js not supported.
Instead change the require of index.js in /app/index.js to a dynamic import() which is available in all CommonJS modules.
```

Cause: a CommonJS file `require()`s a package that ships ESM-only (e.g. chalk 5+, node-fetch 3+).

Fixes:

- Convert your file to ESM.
- Use dynamic import:

```javascript
const chalk = (await import("chalk")).default;
```

- Pin to an older CJS version (`chalk@4`, `node-fetch@2`).

### Cannot use import statement outside a module

(Also a `SyntaxError`; here from the loader's perspective.) See SyntaxError section.

### Unknown file extension

```text
Error [ERR_UNKNOWN_FILE_EXTENSION]: Unknown file extension ".ts" for /app/index.ts
```

Cause: Node was asked to import a `.ts` file directly. Node has no built-in TypeScript loader (until 22+ with `--experimental-strip-types`).

Fixes:

- Compile first with `tsc`.
- Use `tsx` or `ts-node`:

```bash
npx tsx app.ts
node --import tsx app.ts
```

- Node 22+:

```bash
node --experimental-strip-types app.ts
```

### Package subpath not exported

```text
Error [ERR_PACKAGE_PATH_NOT_EXPORTED]: Package subpath './lib/foo' is not defined by "exports" in /app/node_modules/pkg/package.json
```

Cause: package's `package.json` declares `"exports"` and the path you imported isn't listed.

Fix — import a public entry:

```javascript
import { foo } from "pkg";       // works
import { foo } from "pkg/foo";   // only if "exports": { "./foo": "..." }
```

### Directory import not supported

```text
Error [ERR_UNSUPPORTED_DIR_IMPORT]: Directory import '/app/utils' is not supported resolving ES modules
```

Cause: ESM does not implicitly resolve to `index.js` in a directory.

Fix:

```javascript
import { x } from "./utils/index.js";
```

### CJS / ESM interop summary

| Setting | File runs as |
|---|---|
| `package.json: "type": "commonjs"` (or absent) | `.js` -> CJS, `.mjs` -> ESM, `.cjs` -> CJS |
| `package.json: "type": "module"` | `.js` -> ESM, `.mjs` -> ESM, `.cjs` -> CJS |

For libraries supporting both, use conditional exports:

```json
{
  "exports": {
    ".": {
      "import": "./dist/index.mjs",
      "require": "./dist/index.cjs"
    }
  }
}
```

Run ESM in Jest:

```bash
node --experimental-vm-modules node_modules/.bin/jest
```

## fetch / network errors

### fetch failed (Node 18+ undici)

```text
TypeError: fetch failed
    at fetch (/node_modules/undici/...)
    [cause]: Error: connect ECONNREFUSED 127.0.0.1:443
```

The `cause` chain has the actual reason. Always inspect `.cause`:

```javascript
try { await fetch(url); }
catch (e) { console.error(e.cause ?? e); }
```

Common causes inside `.cause`:

- `ECONNREFUSED` — nothing listening.
- `ENOTFOUND` — DNS failure.
- `ECONNRESET` — peer dropped connection mid-response.
- `UND_ERR_HEADERS_TIMEOUT` — server didn't send headers within 300s.
- `UND_ERR_BODY_TIMEOUT` — server stalled mid-body.
- `UND_ERR_SOCKET` — socket-level error.
- `CERT_HAS_EXPIRED` — TLS cert past validity.

### Failed to fetch (browser)

```text
TypeError: Failed to fetch
TypeError: NetworkError when attempting to fetch resource
```

(Chrome / Firefox respectively.) Generic — could be CORS, offline, DNS, mixed-content (HTTPS page calling HTTP), blocked by client (uBlock), or server unreachable. Check the **Network** tab; CORS errors typically show the request as cancelled with a CORS message in console.

### AbortError: The user aborted a request

```text
AbortError: The user aborted a request.
DOMException: The operation was aborted.
```

```javascript
const ctrl = new AbortController();
setTimeout(() => ctrl.abort(), 5000);
const r = await fetch(url, { signal: ctrl.signal });
```

Catch and treat as timeout:

```javascript
try { await fetch(url, { signal: ctrl.signal }); }
catch (e) {
  if (e.name === "AbortError") { /* timed out */ }
  else throw e;
}
```

In Node 17.3+ use `AbortSignal.timeout(ms)`:

```javascript
const r = await fetch(url, { signal: AbortSignal.timeout(5000) });
```

### Request body is null

```text
TypeError: Request body is null
```

Cause: passed `null` body where body is required for the method.

### body used already

```text
TypeError: Body has already been consumed.
TypeError: body stream already read
```

Cause: calling `.json()` then `.text()` on the same `Response`.

Fix — clone before consuming twice:

```javascript
const r = await fetch(url);
const r2 = r.clone();
const a = await r.json();
const b = await r2.text();
```

## CORS errors (browser)

CORS messages appear in the browser console, never on the server. They cannot be caught in JS — `fetch` rejects with the generic `Failed to fetch` and the **console** carries the detail.

### Blocked by CORS policy

```text
Access to fetch at 'https://api.example.com/users' from origin 'https://app.example.com'
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present
on the requested resource.
```

Fix — server must send:

```text
Access-Control-Allow-Origin: https://app.example.com
```

Or for development:

```text
Access-Control-Allow-Origin: *
```

But `*` is incompatible with credentials.

### Preflight fails

```text
Access to fetch at 'X' from origin 'Y' has been blocked by CORS policy:
Response to preflight request doesn't pass access control check:
It does not have HTTP ok status.
```

Cause: server returned non-2xx to the `OPTIONS` request.

Fix — handle `OPTIONS` and return 204:

```javascript
// express
app.options("*", cors());
```

### Origin mismatch

```text
The 'Access-Control-Allow-Origin' header has a value 'https://other.com'
that is not equal to the supplied origin.
```

Fix — echo the request origin (allow-list it on the server) instead of hardcoding.

### Header not allowed

```text
Request header field Authorization is not allowed by Access-Control-Allow-Headers
in preflight response.
```

Fix:

```text
Access-Control-Allow-Headers: Authorization, Content-Type
```

### Credentials require explicit allow

```text
The value of the 'Access-Control-Allow-Credentials' header in the response is ''
which must be 'true' when the request's credentials mode is 'include'.
```

Fix — server:

```text
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: https://app.example.com    # exact, not *
```

Client:

```javascript
fetch(url, { credentials: "include" });
```

### Opaque responses

A `mode: "no-cors"` request resolves with an "opaque" `Response` whose `.status` is `0`, body is unreadable, and which is unsuitable for anything beyond `<img>` / `<script>` use.

```javascript
const r = await fetch(url, { mode: "no-cors" });
r.ok; // false
r.status; // 0
await r.text(); // ""
```

## Node process / OS errors

### ENOTFOUND

```text
Error: getaddrinfo ENOTFOUND example.invalid
    code: 'ENOTFOUND',
    errno: -3008,
    syscall: 'getaddrinfo'
```

Cause: DNS lookup failed. Wrong hostname or DNS unreachable.

### ECONNREFUSED

```text
Error: connect ECONNREFUSED 127.0.0.1:5432
    code: 'ECONNREFUSED',
    errno: -111
```

Cause: nothing is listening on that host:port. Service down, wrong port, firewall.

### ECONNRESET

```text
Error: read ECONNRESET
Error: socket hang up
    code: 'ECONNRESET'
```

Cause: peer closed connection without `FIN`. Frequently from idle timeouts on load balancers.

Fix — agent keep-alive tuning, retry on idempotent ops.

### EADDRINUSE

```text
Error: listen EADDRINUSE: address already in use :::3000
```

Cause: another process is bound to that port.

```bash
lsof -iTCP:3000 -sTCP:LISTEN     # find PID
kill -TERM <pid>
# or
fuser -k 3000/tcp
```

### EACCES (binding privileged port)

```text
Error: listen EACCES: permission denied 0.0.0.0:80
```

Cause: ports < 1024 require root on Unix.

Fixes:

- Use a higher port and reverse-proxy.
- Linux: `setcap`:

```bash
sudo setcap 'cap_net_bind_service=+ep' $(which node)
```

### EACCES (filesystem)

```text
Error: EACCES: permission denied, open '/etc/hosts'
```

Fix — `chmod` / `chown`, or run with appropriate user.

### ENOENT

```text
Error: ENOENT: no such file or directory, open '/app/missing.json'
```

Cause: file doesn't exist; relative path resolved from `process.cwd()`, not the script's dir.

Fix — use `__dirname`:

```javascript
const p = path.join(__dirname, "data.json");
```

### EMFILE: too many open files

```text
Error: EMFILE: too many open files, open '/...'
```

Cause: leaked file descriptors, or `ulimit` too low.

```bash
ulimit -n        # check
ulimit -n 65535  # raise (session)
```

Linux persistent: edit `/etc/security/limits.conf`. macOS: `launchctl limit maxfiles`.

In code — close streams, use a connection pool, or use the `graceful-fs` package which queues `fs` calls when limits are hit.

### ENOSPC: no space left on device

```text
Error: ENOSPC: no space left on device, write
Error: ENOSPC: System limit for number of file watchers reached
```

Two distinct meanings:

- Disk full: `df -h`, clean up.
- Linux inotify limit (file watchers; common with webpack-dev-server, chokidar):

```bash
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### EPERM

```text
Error: EPERM: operation not permitted, unlink 'C:\...\file'
```

Common on Windows when a file is open in another process, or read-only attribute is set.

### ETIMEDOUT

```text
Error: connect ETIMEDOUT
```

Cause: the SYN had no response within the OS timeout.

### EPIPE

```text
Error: write EPIPE
```

Cause: writing to a pipe whose reader is gone — typical when piping to `head`:

```bash
node -e "for (let i=0;i<1e6;i++) console.log(i)" | head
```

Fix — handle `error` on stdout, or terminate gracefully:

```javascript
process.stdout.on("error", (err) => { if (err.code === "EPIPE") process.exit(0); });
```

## JSON parsing errors

### Unexpected token < in JSON at position 0

Got HTML, expected JSON. Almost always a misrouted request returning a 404 page. (See SyntaxError section for the fix.)

### Unexpected end of JSON input

Empty body or truncated.

### Unexpected non-whitespace character after JSON

Two JSON documents concatenated.

### Bad surrogate pair in JSON

```text
SyntaxError: Bad control character in string literal in JSON at position N
```

Cause: literal control char in a JSON string. Fix — escape (`\n`, ` `).

## Async stack traces

V8 has had useful async stack traces since 10.x and **zero-cost async stack traces** since 12.x. You'll see frames marked `async`:

```text
Error: boom
    at deepest (/app/x.js:5:9)
    at async middle (/app/x.js:10:3)
    at async top (/app/x.js:15:3)
```

If you don't see them in older Node, enable:

```bash
node --async-stack-traces app.js
```

Promise tick boundaries in older runtimes truncated the stack — libraries like `longjohn`, `bluebird` (`Promise.config({ longStackTraces: true })`), and `trace`/`clarify` reconstructed full traces at significant CPU cost. Modern engines no longer need them.

When you `try/catch` an awaited call, the catch's `e.stack` includes the async caller. If you re-throw across a process boundary (e.g. into a worker), serialise `.stack` separately.

## process.exit / signals / shutdown

### process.exit was called with code 1

There's no actual error message of this exact text from Node — but tooling (Jest, npm) commonly prints:

```text
process.exit was called with code 1
npm ERR! code 1
```

Cause: explicit `process.exit(1)` somewhere, or an unhandled rejection (Node 15+).

### kill ESRCH

```text
Error: kill ESRCH
```

Cause: `process.kill(pid)` for a PID that no longer exists. Catch and ignore in cleanup paths.

### Graceful shutdown

```javascript
function shutdown(signal) {
  return async () => {
    console.log(`received ${signal}, draining...`);
    server.close(); // stop accepting new connections
    await db.disconnect();
    process.exit(0);
  };
}
process.on("SIGTERM", shutdown("SIGTERM"));
process.on("SIGINT", shutdown("SIGINT"));
```

### SIGKILL is uncatchable

`SIGKILL` (9) and `SIGSTOP` cannot be trapped. `kill -TERM <pid>` is the correct way to ask a Node process to shut down cleanly.

### exit code conventions

- `0` — success.
- `1` — generic error.
- `2` — misuse / bad CLI flags.
- `130` — interrupted by SIGINT (128 + 2).
- `137` — killed by SIGKILL (128 + 9), e.g. OOM-killer.
- `143` — terminated by SIGTERM (128 + 15).

## EventEmitter errors

### MaxListenersExceededWarning

```text
(node:1234) MaxListenersExceededWarning: Possible EventEmitter memory leak detected.
11 data listeners added to [Socket]. Use emitter.setMaxListeners() to increase limit
```

Cause: more than 10 listeners attached to one event (default limit). Often a real leak from forgetting `removeListener` / `off`.

Fix — investigate; if intentional, raise:

```javascript
emitter.setMaxListeners(50);
// or globally
require("events").defaultMaxListeners = 50;
```

### Unhandled error event

```text
events.js:174
      throw err;
      ^
Error: Unhandled 'error' event
    at Socket.emit (events.js:172:17)
```

Cause: `emitter.emit("error", err)` with no `error` listener. The default behaviour is to throw the error.

Fix:

```javascript
emitter.on("error", (err) => log.error({ err }));
```

### once vs on

```javascript
emitter.on("data", handler);     // every event, must remove
emitter.once("data", handler);   // first event only, auto-removed
emitter.off("data", handler);    // alias for removeListener
emitter.removeAllListeners("data");
```

`once` returns a Promise variant (Node 11.13+):

```javascript
const [first] = await events.once(emitter, "ready");
```

## Buffer / TypedArray errors

### offset out of bounds

```text
RangeError: offset out of bounds
RangeError: byteOffset is out of bounds
RangeError: Out of range index
```

Cause: writing/reading past a buffer's length.

```javascript
const buf = Buffer.alloc(4);
buf.writeUInt32BE(0, 4); // throws — needs 4 bytes starting at offset 4
```

### ERR_INVALID_ARG_TYPE

```text
TypeError [ERR_INVALID_ARG_TYPE]: The first argument must be of type string or
an instance of Buffer, ArrayBuffer, or Array or an Array-like Object. Received undefined
```

Cause: passing the wrong type to a Buffer / fs API.

### Buffer.write encoding

```text
TypeError: Unknown encoding: utf
```

Cause: encoding name typo. Valid: `utf8` (or `utf-8`), `utf16le`, `latin1`, `base64`, `base64url`, `hex`, `ascii`, `binary` (alias for `latin1`).

### deprecated Buffer constructor

```text
(node:1234) [DEP0005] DeprecationWarning: Buffer() is deprecated due to security
and usability issues. Please use the Buffer.alloc(), Buffer.allocUnsafe(),
or Buffer.from() methods instead.
```

Fix:

```javascript
Buffer.from(string, "utf8")    // from string
Buffer.from(arrayBuffer)       // from AB
Buffer.alloc(size)             // zero-filled
Buffer.allocUnsafe(size)       // uninitialised, faster
```

## Stream errors

### write after end

```text
Error: write after end
    code: 'ERR_STREAM_WRITE_AFTER_END'
```

Cause: `stream.write()` after `stream.end()`.

### read after destroy

```text
Error: Cannot call write after a stream was destroyed
    code: 'ERR_STREAM_DESTROYED'
```

### push() after EOF

```text
Error: stream.push() after EOF
    code: 'ERR_STREAM_PUSH_AFTER_EOF'
```

Cause: `Readable.push(...)` after a `null` push (which signals EOF).

### Premature close

```text
Error: Premature close
    code: 'ERR_STREAM_PREMATURE_CLOSE'
```

Cause: stream finished without finishing — pipeline saw EOF before all data flowed.

### pipeline pattern

The `.pipe()` method does not propagate errors; use `pipeline` (Node 10+) or `pipeline/promises` (Node 15+):

```javascript
import { pipeline } from "node:stream/promises";
import { createReadStream, createWriteStream } from "node:fs";
import { createGzip } from "node:zlib";

await pipeline(
  createReadStream("in"),
  createGzip(),
  createWriteStream("out.gz"),
);
```

Errors from any stage are forwarded to the awaited promise.

### Web Streams interop

Node 18+ has `ReadableStream` / `WritableStream` (Web Streams API) and converters:

```javascript
import { Readable } from "node:stream";
const web = Readable.toWeb(nodeStream);
const node = Readable.fromWeb(webStream);
```

## DNS errors

```text
Error: getaddrinfo ENOTFOUND example.invalid
Error: getaddrinfo EAI_AGAIN example.com
Error: queryA ENODATA example.com
Error: queryA ESERVFAIL example.com
```

`ENOTFOUND` = NXDOMAIN. `EAI_AGAIN` = temporary failure. `ENODATA` = no records of requested type. `ESERVFAIL` = upstream server failed.

`dns.lookup` (default in `net.connect` and friends) uses libuv's threadpool and the OS resolver (`getaddrinfo`); it respects `/etc/hosts` and `nsswitch.conf`. `dns.resolve*` functions use Node's pure-JS DNS client and skip the OS — useful for testing, but won't see your hosts file.

```javascript
import { promises as dns } from "node:dns";
await dns.lookup("example.com");                // OS resolver
await dns.resolve4("example.com");              // direct DNS, A records
```

A flood of `dns.lookup` calls can exhaust the libuv threadpool (default 4 threads). Set `UV_THREADPOOL_SIZE`:

```bash
UV_THREADPOOL_SIZE=64 node app.js
```

Node 17+ has an in-process DNS cache (`--dns-result-order`, `dns.setDefaultResultOrder("ipv4first")`).

## TLS / HTTPS certificate errors

### Unable to verify the first certificate

```text
Error: unable to verify the first certificate
    code: 'UNABLE_TO_VERIFY_LEAF_SIGNATURE'
```

Cause: server didn't include intermediate certs and Node's root store doesn't have them.

Fix — fix the server (send intermediates), or:

```bash
NODE_EXTRA_CA_CERTS=/etc/ssl/certs/ca.pem node app.js
```

### Self signed certificate

```text
Error: self signed certificate
    code: 'DEPTH_ZERO_SELF_SIGNED_CERT'
Error: self signed certificate in certificate chain
    code: 'SELF_SIGNED_CERT_IN_CHAIN'
```

Cause: cert is signed by its own subject, not a trusted CA.

Fix — add the CA file to `NODE_EXTRA_CA_CERTS`, or for clients explicitly trust:

```javascript
import { Agent, fetch } from "undici";
const agent = new Agent({ connect: { ca: fs.readFileSync("ca.pem") } });
await fetch(url, { dispatcher: agent });
```

**Anti-pattern:** disabling verification.

```javascript
// don't do this in production
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
fetch(url, { agent: new https.Agent({ rejectUnauthorized: false }) });
```

This logs a warning and silently disables protection.

### Certificate has expired

```text
Error: certificate has expired
    code: 'CERT_HAS_EXPIRED'
```

Fix — renew. (`openssl x509 -enddate -noout -in cert.pem` shows expiry.)

### Hostname mismatch

```text
Error: Hostname/IP does not match certificate's altnames:
  Host: api.example.com. is not in the cert's altnames: DNS:other.example.com
    code: 'ERR_TLS_CERT_ALTNAME_INVALID'
```

Cause: cert's CN/SAN doesn't include the hostname you connected to.

Fix — get a cert covering that hostname; SNI is automatic in Node so no extra config needed.

### Other TLS error codes

```text
ERR_TLS_HANDSHAKE_TIMEOUT
ERR_OSSL_SSL_NO_PROTOCOLS_AVAILABLE
ERR_OSSL_DIGEST_TOO_BIG_FOR_RSA_KEY
ERR_OSSL_PEM_NO_START_LINE
DEPTH_ZERO_SELF_SIGNED_CERT
UNABLE_TO_GET_ISSUER_CERT_LOCALLY
```

`ERR_OSSL_PEM_NO_START_LINE` usually means a key file is binary (DER) where PEM was expected; convert with `openssl`.

## WebSocket / Socket.IO errors

### WebSocket is not open

```text
Error: WebSocket is not open: readyState 3 (CLOSED)
Error: WebSocket is not open: readyState 0 (CONNECTING)
```

Cause: `ws.send` before `OPEN` event, or after close.

States:

```text
0 CONNECTING
1 OPEN
2 CLOSING
3 CLOSED
```

Fix:

```javascript
ws.addEventListener("open", () => ws.send("hi"));
```

### Closed before connection established

```text
Error: WebSocket was closed before the connection was established
```

Cause: `ws.close()` during the handshake, or server reset the connection.

### Common close codes

```text
1000 normal
1001 going away (e.g. tab close)
1002 protocol error
1003 unsupported data
1006 abnormal closure (no code received)
1007 invalid frame payload data
1008 policy violation
1009 message too big
1011 internal error
4xxx application-defined
```

### Socket.IO

```text
Error: server error
Error: xhr poll error
Error: websocket error
```

Always log `socket.id`, `reason`, the error event:

```javascript
socket.on("connect_error", (err) => {
  console.error(err.message, err.cause);
});
```

## React common errors

### Each child should have a unique key

```text
Warning: Each child in a list should have a unique "key" prop.
Check the render method of `Foo`. See https://reactjs.org/link/warning-keys for more information.
```

Cause: rendering a list without `key` prop on each child.

Fix:

```javascript
items.map(it => <li key={it.id}>{it.name}</li>)
```

Use a stable identifier, not array index, when items can be reordered/inserted.

### Cannot update a component while rendering

```text
Warning: Cannot update a component (`A`) while rendering a different component (`B`).
To locate the bad setState() call inside `B`, follow the stack trace as described in
https://reactjs.org/link/setstate-in-render
```

Cause: calling `setState` synchronously inside another component's render.

Fix — move it into `useEffect`, an event handler, or `useSyncExternalStore`.

### Too many re-renders

```text
Error: Too many re-renders. React limits the number of renders to prevent an infinite loop.
```

Cause: `setState` called every render with no condition.

Broken:

```javascript
function C() {
  const [n, setN] = useState(0);
  setN(n + 1); // every render
  return <div />;
}
```

Fix — guard or use `useEffect`:

```javascript
useEffect(() => { setN(n => n + 1); }, []);
```

### Hooks can only be called inside the body of a function component

```text
Error: Invalid hook call. Hooks can only be called inside of the body of a function component.
This could happen for one of the following reasons:
1. You might have mismatching versions of React and the renderer (such as React DOM)
2. You might be breaking the Rules of Hooks
3. You might have more than one copy of React in the same app
```

Causes:

- Calling a hook conditionally or inside a loop.
- Calling a hook from a non-component non-hook function.
- Two copies of React (often via `npm link`).

Fix — keep hooks at the top level of components/hooks; check `npm ls react` to ensure exactly one.

### useEffect must not return anything besides a function

```text
Warning: useEffect must not return anything besides a function, which is used for clean-up.
You returned a Promise. Effect callbacks are synchronous to prevent race conditions.
Put the async function inside.
```

Cause: `useEffect(async () => { ... })`.

Fix:

```javascript
useEffect(() => {
  let cancelled = false;
  (async () => {
    const data = await fetchUser();
    if (!cancelled) setUser(data);
  })();
  return () => { cancelled = true; };
}, []);
```

### Hydration failed (Next.js / SSR)

```text
Error: Hydration failed because the initial UI does not match what was rendered on the server.
Warning: Text content did not match. Server: "0" Client: "1"
```

Cause: server-rendered HTML differs from client first render — typical sources: `Date.now()`, `Math.random()`, locale formatting, `window`-dependent code in the initial render, or a browser extension injecting markup.

Fix — only render after mount:

```javascript
const [mounted, setMounted] = useState(false);
useEffect(() => setMounted(true), []);
if (!mounted) return null;
```

Or move the variable bit into `useEffect`.

### Element type is invalid

```text
Error: Element type is invalid: expected a string (for built-in components) or a class/function
(for composite components) but got: undefined. You likely forgot to export your component
from the file it's defined in, or you might have mixed up default and named imports.
```

Causes:

- Forgot `export default`.
- `import Foo from "./foo"` when foo only `export const Foo`.

Fix — match the import to the export:

```javascript
// foo.js
export default function Foo() {}     // -> import Foo from "./foo"
export function Foo() {}              // -> import { Foo } from "./foo"
```

### Objects are not valid as a React child

```text
Error: Objects are not valid as a React child (found: object with keys {a, b}).
If you meant to render a collection of children, use an array instead.
```

Fix — render a primitive or stringify:

```javascript
<pre>{JSON.stringify(obj, null, 2)}</pre>
```

### Each prop must be specified

```text
Warning: Failed prop type: The prop `foo` is marked as required in `Bar`, but its value is `undefined`.
```

Migrate to TypeScript or `zod` props for stronger guarantees; PropTypes runtime checks are dev-only.

### Stale closure in useEffect

No specific error — bugs include event handlers reading old state. Fix with refs or include the value in deps:

```javascript
useEffect(() => {
  const id = setInterval(() => setN(n => n + 1), 1000); // functional update
  return () => clearInterval(id);
}, []);
```

## Framework-specific errors

### Next.js

```text
Error: NEXT_NOT_FOUND        # thrown by notFound() — internal, handled by Next
Error: NEXT_REDIRECT         # thrown by redirect() — internal
Error: getServerSideProps cannot be used with App Router
Error: You're importing a component that needs useState. It only works in a Client Component
Error: Image with src "/x.png" has invalid "src" property. Make sure it starts with "/"
Error: Error: Module not found: Can't resolve 'fs' (or 'path', 'crypto')
Error: Application error: a server-side exception has occurred
```

`fs` import in browser context — gate with `typeof window === "undefined"` or move to a server file/route.

### Vue

```text
[Vue warn]: Property "foo" was accessed during render but is not defined on instance.
[Vue warn]: Avoid mutating a prop directly since the value will be overwritten whenever the parent component re-renders.
[Vue warn]: Maximum recursive updates exceeded.
[Vue warn]: Failed to resolve component: my-comp.
```

Fix prop mutation by using `v-model` or emitting events.

### Angular

```text
Error: ExpressionChangedAfterItHasBeenCheckedError: Expression has changed after it was checked.
```

Cause: a value was changed after change detection completed for a parent component (synchronous mutation in `ngAfterViewInit`).

Fix — defer with `Promise.resolve().then(...)`, `setTimeout(0)`, or `cdRef.detectChanges()` after the change.

```text
NullInjectorError: No provider for HttpClient!
```

Add `provideHttpClient()` (Angular 15+) or import `HttpClientModule`.

### regeneratorRuntime is not defined

```text
ReferenceError: regeneratorRuntime is not defined
```

Cause: babel-transformed `async`/generator code without the runtime. Modern targets (ES2017+) emit native async; or:

```bash
npm install regenerator-runtime
```

```javascript
import "regenerator-runtime/runtime";
```

## Webpack / Vite / bundler errors

### Module not found

```text
Module not found: Error: Can't resolve 'lodash' in '/app/src'
```

Fix — install or correct the path; check `tsconfig.paths` / webpack `resolve.alias` if using aliases.

### Cannot find module or its corresponding type declarations

```text
Cannot find module 'foo' or its corresponding type declarations.ts(2307)
```

Fix:

```bash
npm i -D @types/foo
# or
echo "declare module 'foo';" > types/foo.d.ts
```

Then ensure `tsconfig.json` `include` covers `types/`.

### Module parse failed: Unexpected token

```text
Module parse failed: Unexpected token (1:0)
You may need an appropriate loader to handle this file type.
```

Cause: webpack saw JSX/TS without a loader.

Fix — webpack rule:

```javascript
{ test: /\.(t|j)sx?$/, use: "babel-loader" }
```

### ENOENT during build

Same as Node ENOENT — usually a hardcoded relative path.

### Circular dependency warning

```text
Critical dependency: the request of a dependency is an expression
Circular dependency detected: a.js -> b.js -> a.js
```

Refactor to extract shared types into a third module.

### Vite: Failed to resolve import

```text
[plugin:vite:import-analysis] Failed to resolve import "X" from "Y". Does the file exist?
```

Fix — check extension (Vite is strict about `.ts` vs `.js` in some cases) and `vite.config.ts` aliases.

### Out of memory during build

```text
FATAL ERROR: Reached heap limit Allocation failed - JavaScript heap out of memory
```

```bash
NODE_OPTIONS=--max_old_space_size=8192 npm run build
```

## TypeScript errors

TypeScript errors are prefixed with `TSxxxx`. They appear at compile time only.

### TS2304: Cannot find name

```text
TS2304: Cannot find name 'foo'.
```

Cause: identifier not declared, not imported, or not in `tsconfig.json` types.

### TS2307: Cannot find module

```text
TS2307: Cannot find module 'foo' or its corresponding type declarations.
```

Same fix as the bundler version above.

### TS2339: Property does not exist on type

```text
TS2339: Property 'foo' does not exist on type 'Bar'.
```

Cause: typo or wrong type. **Don't** silence with `as any`. Narrow:

```typescript
function isFoo(x: unknown): x is { foo: string } {
  return typeof x === "object" && x !== null && "foo" in x;
}
```

### TS2322: Type 'X' is not assignable to type 'Y'

```text
TS2322: Type 'string' is not assignable to type 'number'.
```

The most common assignment compatibility error. Fix the source value or widen the target type.

### TS2345: Argument of type 'X' is not assignable to parameter of type 'Y'

```text
TS2345: Argument of type 'string | undefined' is not assignable to parameter of type 'string'.
  Type 'undefined' is not assignable to type 'string'.
```

Fix — narrow before the call:

```typescript
if (s !== undefined) f(s);
// or
f(s ?? "");
```

### TS2693: only refers to a type, but is being used as a value

```text
TS2693: 'Foo' only refers to a type, but is being used as a value here.
```

Cause: using an `interface` or `type` as a runtime value.

```typescript
interface User { name: string; }
const u = User; // TS2693
```

For runtime checks, use a class, a Zod schema, or an enum.

### TS7006: Parameter implicitly has an 'any' type

```text
TS7006: Parameter 'x' implicitly has an 'any' type.
```

`noImplicitAny: true` requires annotation:

```typescript
function f(x: number) { return x + 1; }
```

### TS2532: Object is possibly 'undefined'

```text
TS2532: Object is possibly 'undefined'.
```

Cause: `strictNullChecks: true` and a value typed `T | undefined`.

```typescript
const a: number[] = [];
a[0].toFixed();   // TS2532
a[0]?.toFixed();  // OK
```

### TS18048: 'X' is possibly 'undefined'

Same family as 2532, distinct code introduced in 4.8 for clearer messages.

### TS2769: No overload matches this call

```text
TS2769: No overload matches this call.
  Overload 1 of 3, '(...)': ...
  Overload 2 of 3, '(...)': ...
```

Cause: function has multiple signatures, none of which fits your call site exactly. Read each overload and align argument types.

### TS2515: Non-abstract class does not implement inherited abstract member

```text
TS2515: Non-abstract class 'B' does not implement inherited abstract member 'm' from class 'A'.
```

Add the method, or mark `B` `abstract`.

### TS2741: Property is missing in type

```text
TS2741: Property 'name' is missing in type '{ id: number; }' but required in type 'User'.
```

Provide the missing field, or make it optional in the type (`name?: string`).

### TS2540: Cannot assign to read-only property

```text
TS2540: Cannot assign to 'foo' because it is a read-only property.
```

Fields marked `readonly` (or computed via `as const`).

### TS18003: No inputs were found in config file

```text
TS18003: No inputs were found in config file 'tsconfig.json'.
Specified 'include' paths were '["src/**/*"]' and 'exclude' paths were '[]'.
```

Cause: empty / wrong `include`. Fix the glob.

### TS6133: 'X' is declared but its value is never read

```text
TS6133: 'foo' is declared but its value is never read.
```

`noUnusedLocals: true`. Remove the variable, or prefix with `_`:

```typescript
function f(_unused: number) { /* ... */ }
```

### TS2375: exactOptionalPropertyTypes

```text
TS2375: Type '{ foo: undefined }' is not assignable to type '{ foo?: string }' with 'exactOptionalPropertyTypes: true'.
```

With that flag, `foo?: string` and `foo: string | undefined` are different. Omit the property entirely instead of setting it to `undefined`.

### tsc useful flags

```bash
tsc --noEmit                 # type-check without writing JS
tsc --watch                  # incremental
tsc --strict                 # all strict-* flags on
tsc --listFiles              # show every file pulled in
tsc --traceResolution        # debug resolution
tsc --explainFiles           # why each file is included
tsc --generateTrace ./trace  # perf trace, view at /tracing in Chrome
```

## NPM / Yarn / pnpm errors

### npm ERR! code ERESOLVE

```text
npm ERR! code ERESOLVE
npm ERR! ERESOLVE unable to resolve dependency tree
npm ERR! While resolving: app@1.0.0
npm ERR! Found: react@17.0.0
npm ERR! Could not resolve dependency:
npm ERR! peer react@"^18.0.0" from some-pkg@1.0.0
```

Causes: a package's `peerDependencies` aren't satisfied.

Fixes:

- Upgrade the offending dep so peers match.
- `npm install --legacy-peer-deps` (npm 7+ behaviour reverted).
- `npm install --force` (last resort).

### Peer dep missing

```text
npm WARN ERESOLVE overriding peer dependency
npm WARN While resolving: app@1.0.0
npm WARN Found: react@17 ...
```

Just warnings — install the peer to silence.

### EACCES permission denied

```text
npm ERR! Error: EACCES: permission denied, mkdir '/usr/local/lib/node_modules/foo'
```

Cause: global install without write permission to the global prefix.

Fixes (preferred):

- Use `nvm` or `fnm` so npm prefix is in your home dir.
- Or set a user prefix:

```bash
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
export PATH="$HOME/.npm-global/bin:$PATH"
```

Avoid `sudo npm install -g`.

### npm ERR! 401 Unauthorized

```text
npm ERR! 401 Unauthorized - GET https://registry.npmjs.org/private-pkg
```

Cause: missing/expired auth token.

```bash
npm whoami
npm login
# or
npm config set //registry.npmjs.org/:_authToken=$NPM_TOKEN
```

For a private registry, configure scope:

```bash
npm config set @myorg:registry https://npm.myorg.com/
npm config set //npm.myorg.com/:_authToken=$TOKEN
```

### npm ERR! 404

```text
npm ERR! 404 Not Found - GET https://registry.npmjs.org/package
npm ERR! 404 'package@1.0.0' is not in the npm registry
```

Typo, scope mismatch, or unpublished version.

### Yarn frozen lockfile failed

```text
error An unexpected error occurred: "https://...".
error Lockfile would have been created. Run `yarn install --no-frozen-lockfile` to fix.
```

```text
error This package doesn't seem to be present in your lockfile.
```

Cause: `package.json` and `yarn.lock` disagree (typical CI failure).

Fix locally:

```bash
yarn install
git add yarn.lock
```

### pnpm ERR_PNPM_OUTDATED_LOCKFILE

```text
ERR_PNPM_OUTDATED_LOCKFILE  Cannot install with "frozen-lockfile" because pnpm-lock.yaml is not up to date with package.json
```

Same root cause; run `pnpm install` and commit the lockfile.

### pnpm peer dependency issues

```text
WARN peer dependency missing
```

pnpm is stricter than npm about peers; resolve by adding to root or setting `auto-install-peers=true` in `.npmrc`.

### Lockfile conflicts

Merge conflicts in `package-lock.json` / `yarn.lock` are common. Resolve by:

```bash
git checkout --theirs package-lock.json
npm install                # rewrite lockfile cleanly
git add package-lock.json
```

### EINTEGRITY

```text
npm ERR! code EINTEGRITY
npm ERR! integrity checksum failed when using sha512: wanted X but got Y
```

Cache corruption.

```bash
npm cache verify
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

## Common gotchas — broken then fixed

### 1. this binding lost in callback

```javascript
class C {
  constructor() { this.n = 0; }
  inc() { this.n++; }
}
const c = new C();
setTimeout(c.inc, 0);                 // TypeError: Cannot read properties of undefined
setTimeout(() => c.inc(), 0);         // OK
setTimeout(c.inc.bind(c), 0);         // OK
```

### 2. == vs ===

```javascript
0 == "";          // true
0 == "0";         // true
"" == "0";        // false
null == undefined;// true
NaN == NaN;       // false
[] == false;      // true
[] == 0;          // true
[null] == 0;      // true
```

Use `===`. The only common exception is `value == null` to test both `null` and `undefined` simultaneously.

### 3. var hoisting vs let/const TDZ

```javascript
console.log(a);   // undefined (var hoists)
var a = 1;

console.log(b);   // ReferenceError (TDZ)
let b = 1;
```

Always prefer `const`; `let` only when reassignment is needed.

### 4. JSON.parse(undefined) throws

```javascript
JSON.parse(undefined); // SyntaxError: "undefined" is not valid JSON
JSON.parse(null);      // null  (the string "null" is valid JSON)
JSON.parse("");        // SyntaxError: Unexpected end of JSON input
```

Always `JSON.parse(s ?? "null")` or guard.

### 5. parseInt without radix

```javascript
parseInt("08");        // 8 in modern engines, 0 in ES3
parseInt("0x10");      // 16
parseInt("10", 2);     // 2
Number("08");          // 8
```

Always pass radix or use `Number()`.

### 6. Mutating array during forEach

```javascript
const a = [1, 2, 3, 4];
a.forEach(v => { if (v % 2 === 0) a.splice(a.indexOf(v), 1); });
// a may be [1, 3, 4] — element skipped because indices shift
```

Fix — filter or iterate a copy:

```javascript
const a2 = a.filter(v => v % 2 !== 0);
```

### 7. Date.parse format quirks

```javascript
new Date("2024-01-15");    // UTC midnight
new Date("2024/01/15");    // local midnight (browser-dependent)
new Date("15/01/2024");    // Invalid Date
Date.parse("2024-01-15");  // 1705276800000 (UTC)
```

Always pass ISO 8601 (`YYYY-MM-DDTHH:mm:ssZ`); for anything else use `date-fns` / `Temporal`.

### 8. Number.parseFloat edge cases

```javascript
Number.parseInt("0.1");    // 0
Number.parseFloat("0.1");  // 0.1
parseInt("0.1");           // 0
parseFloat("foo");         // NaN
Number("foo");             // NaN
Number(" 1 ");             // 1   (trims)
parseInt(" 1 ");           // 1
parseInt("1abc");          // 1   (stops at first non-digit)
Number("1abc");            // NaN
```

### 9. Object.keys order for numeric strings

```javascript
const o = { "10": "ten", "2": "two", a: "A" };
Object.keys(o); // ["2", "10", "a"]
```

Integer-like keys are sorted numerically and listed first; string keys follow in insertion order. To preserve insertion order even for numeric keys, use a `Map`.

### 10. typeof null === 'object'

```javascript
typeof null;        // 'object' — historical bug, never to be fixed
typeof undefined;   // 'undefined'
typeof NaN;         // 'number'
typeof [];          // 'object'
typeof function(){};// 'function'
```

Fix — explicit checks:

```javascript
x === null
Array.isArray(x)
Number.isNaN(x)
```

### 11. async function returns Promise even without return

```javascript
async function f() { /* no return */ }
f();                     // Promise<undefined>
typeof f().then;         // "function"
```

Don't combine with implicit `return undefined` expecting a sync flow.

### 12. Forgot await

```javascript
async function load() { return 1; }
const x = load();    // Promise, not 1
const y = await load();
```

### 13. Spreading null

```javascript
const a = { ...null };       // OK in ES2018+, yields {}
const b = { ...undefined };  // OK, yields {}
const c = [...null];         // TypeError: null is not iterable
```

### 14. Shadowing in for-loops

```javascript
for (var i = 0; i < 3; i++) setTimeout(() => console.log(i), 0); // 3 3 3
for (let i = 0; i < 3; i++) setTimeout(() => console.log(i), 0); // 0 1 2
```

### 15. Array length truthiness

```javascript
const a = [];
if (a) {} // true — empty array is truthy
if (a.length) {} // false — explicit
```

### 16. Promise is not awaitable in Promise.all on a single value

```javascript
const v = await Promise.all(somePromise);  // TypeError: somePromise is not iterable
const v = await Promise.all([somePromise]);// OK
const v = await somePromise;               // simpler
```

### 17. Object spread shallow

```javascript
const a = { nested: { x: 1 } };
const b = { ...a };
b.nested.x = 2;       // also mutates a.nested.x
```

Fix — `structuredClone(a)` (Node 17+, browsers) or per-field clones.

### 18. Empty object as default is shared

```javascript
function f({ a = {} } = {}) { /* still a fresh {} each call */ }
function g(arr = []) { /* fresh [] each call */ }
```

Default expressions are evaluated each call, so this is fine — unlike Python's notorious mutable-default trap.

## Debugging

### Built-in inspector

```bash
node --inspect app.js          # listen on 9229, attach later
node --inspect-brk app.js      # break on first line, wait for debugger
node --inspect=0.0.0.0:9229    # remote
```

Open Chrome → `chrome://inspect` → "Open dedicated DevTools for Node". Source maps work automatically.

VSCode: add `.vscode/launch.json`:

```json
{
  "type": "node",
  "request": "launch",
  "program": "${workspaceFolder}/app.js",
  "skipFiles": ["<node_internals>/**"],
  "outFiles": ["${workspaceFolder}/dist/**/*.js"]
}
```

### console levels

```javascript
console.debug("low-level diagnostics"); // hidden by default in Chrome
console.log("default");
console.info("informational");
console.warn("yellow");
console.error("red");
console.trace("stack from this point");
console.table([{ a: 1 }, { a: 2 }]);
console.group("label"); /* nested */ console.groupEnd();
console.time("t"); /* ... */ console.timeEnd("t");
console.count("hits");
console.assert(condition, "message if false");
```

In Node, levels are mostly stylistic; in browsers they affect filtering. `console.trace` is the cheapest way to ask "who called me?".

### util.inspect

For circular and deep structures:

```javascript
import { inspect } from "node:util";
console.log(inspect(obj, { depth: null, colors: true, breakLength: 100 }));
```

Set per-class:

```javascript
class X { [Symbol.for("nodejs.util.inspect.custom")]() { return "X{...}"; } }
```

### --trace-warnings

```bash
node --trace-warnings app.js
```

Prints stacks for every warning (deprecations, max-listeners, etc.), pinpointing the source.

### --trace-uncaught / --trace-deprecation

```bash
node --trace-uncaught app.js
node --trace-deprecation app.js
```

### CPU profiling

```bash
node --prof app.js                # writes isolate-*.log
node --prof-process isolate-*.log # human report
```

Or DevTools "Performance" tab while the inspector is attached.

### Heap snapshots

```bash
node --heapsnapshot-near-heap-limit=3 app.js   # auto on OOM
node --inspect ...                              # then "Memory" tab in DevTools
```

Or programmatically:

```javascript
import v8 from "node:v8";
v8.writeHeapSnapshot("/tmp/heap.heapsnapshot");
```

Open in Chrome DevTools "Memory" panel.

### Diagnostic reports (Node 12+)

```bash
node --report-on-fatalerror --report-on-signal --report-uncaught-exception app.js
```

On crash a JSON report is written to the cwd with the JS stack, native stack, libuv handles, env, and resource usage. Useful for postmortems on production crashes.

```javascript
process.report.writeReport("/tmp/report.json");
```

### --inspect-brk + DevTools

To attach to a running server:

```bash
kill -SIGUSR1 <pid>           # tells Node to start the inspector
```

Then visit `chrome://inspect`.

### node --watch (Node 18.11+)

```bash
node --watch app.js
node --watch --watch-path=./src --env-file=.env app.js
```

Auto-restarts on file changes; replaces nodemon for simple cases.

## Cross-platform npm scripts

Windows uses `cmd.exe` (no POSIX shell) so this fails on Windows:

```json
{
  "scripts": {
    "dev": "NODE_ENV=development node app.js"
  }
}
```

Fix — `cross-env`:

```bash
npm install --save-dev cross-env
```

```json
{ "scripts": { "dev": "cross-env NODE_ENV=development node app.js" } }
```

For multi-platform parallel scripts:

```bash
npm install --save-dev npm-run-all
```

```json
{
  "scripts": {
    "build:client": "vite build",
    "build:server": "tsc",
    "build": "run-p build:*"
  }
}
```

`rm -rf` is not on Windows; use `rimraf` or `del-cli`. `cp` differs; use `shx cp` or `cpy`. Path separators differ — always build paths with `path.join` rather than concatenation.

Node 20.6+ supports `--env-file` to load `.env` natively, replacing `dotenv` for simple cases:

```bash
node --env-file=.env app.js
```

## Idioms

### Always log Error objects

```javascript
catch (e) {
  log.error({ err: e }, "request failed");   // logs message AND stack
  // not: log.error(e.message)               // throws away the trace
}
```

`pino` and `winston` serialise Error properly when logged under the conventional `err` key.

### Subclass Error properly

```javascript
class HttpError extends Error {
  constructor(status, body, options) {
    super(`HTTP ${status}`, options);
    this.name = "HttpError";
    this.status = status;
    this.body = body;
  }
}
throw new HttpError(404, body, { cause: original });
```

Set `name` so `instanceof` and `e.name` work. Pass `options` so `cause` is preserved (ES2022).

### Never throw strings

```javascript
throw "boom";                // bad — no stack, no .message
throw new Error("boom");     // good
```

### Always handle promise rejections

```javascript
fetch(url).catch(err => log.warn({ err }, "fetch failed"));
```

Or with await:

```javascript
try { await fetch(url); }
catch (err) { log.warn({ err }, "fetch failed"); }
```

### Distinguish operational vs programmer errors

- Operational: network down, file missing, bad input. Recover.
- Programmer: bug in your code (TypeError, ReferenceError, RangeError). Crash and let the supervisor restart.

Don't catch broad ranges in `try/catch` and continue — only catch operational errors you can handle.

### Structured logging

```javascript
import pino from "pino";
const log = pino({ level: process.env.LOG_LEVEL ?? "info" });

try { await op(); }
catch (err) {
  log.error({ err, requestId, userId }, "op failed");
  throw err; // re-throw so caller sees it too
}
```

JSON output is much easier to query in Elasticsearch / Loki than free-form text.

### Use try/finally for cleanup

```javascript
const conn = await db.connect();
try { return await conn.query(sql); }
finally { conn.release(); }
```

Async errors still trigger `finally`.

### Promise.allSettled when one failure shouldn't kill all

```javascript
const results = await Promise.allSettled(reqs);
const ok = results.filter(r => r.status === "fulfilled").map(r => r.value);
const errs = results.filter(r => r.status === "rejected").map(r => r.reason);
```

### Time out anything network

```javascript
async function fetchWithTimeout(url, ms) {
  return fetch(url, { signal: AbortSignal.timeout(ms) });
}
```

### Don't use globalThis as a cache

It's tempting to attach state to `globalThis` to persist across reloads in dev — easy to leak and hard to test.

### Catch JSON parse errors

```javascript
function safeParse(s) {
  try { return [JSON.parse(s), null]; }
  catch (e) { return [null, e]; }
}
const [data, err] = safeParse(input);
```

## See Also

- [javascript](../languages/javascript.md)
- [typescript](../languages/typescript.md)
- [polyglot](../languages/polyglot.md)
- [troubleshooting/python-errors](python-errors.md)
- [troubleshooting/http-errors](http-errors.md)
- [troubleshooting/tls-errors](tls-errors.md)

## References

- MDN: Errors reference — https://developer.mozilla.org/Web/JavaScript/Reference/Errors
- MDN: Error — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Error
- MDN: TypeError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/TypeError
- MDN: ReferenceError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/ReferenceError
- MDN: SyntaxError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/SyntaxError
- MDN: RangeError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/RangeError
- MDN: URIError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/URIError
- MDN: AggregateError — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/AggregateError
- MDN: Promise — https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Promise
- MDN: fetch — https://developer.mozilla.org/docs/Web/API/Fetch_API
- MDN: CORS — https://developer.mozilla.org/docs/Web/HTTP/CORS
- MDN: Errors — CORS error reasons — https://developer.mozilla.org/docs/Web/HTTP/CORS/Errors
- Node.js: Errors — https://nodejs.org/api/errors.html
- Node.js: Error codes — https://nodejs.org/api/errors.html#nodejs-error-codes
- Node.js: process — https://nodejs.org/api/process.html
- Node.js: ECMAScript modules — https://nodejs.org/api/esm.html
- Node.js: Diagnostic report — https://nodejs.org/api/report.html
- Node.js: Inspector — https://nodejs.org/en/learn/getting-started/debugging
- Node.js: HTTPS — https://nodejs.org/api/https.html
- Node.js: TLS — https://nodejs.org/api/tls.html
- Node.js: Stream — https://nodejs.org/api/stream.html
- Node.js: Buffer — https://nodejs.org/api/buffer.html
- Node.js: Events — https://nodejs.org/api/events.html
- Node.js: dns — https://nodejs.org/api/dns.html
- Node.js: net — https://nodejs.org/api/net.html
- TypeScript Handbook — https://www.typescriptlang.org/docs/handbook/intro.html
- TypeScript: Compiler options — https://www.typescriptlang.org/tsconfig
- TypeScript: Error reference — https://typescript.tv/errors/
- ECMA-262 (ECMAScript spec) — https://tc39.es/ecma262/
- React: Errors and warnings — https://react.dev/reference/react
- Next.js: Error handling — https://nextjs.org/docs/app/building-your-application/routing/error-handling
- Vue: Error handling — https://vuejs.org/api/application.html#app-config-errorhandler
- Angular: ExpressionChangedAfterItHasBeenCheckedError — https://angular.dev/errors/NG0100
- Webpack: Module not found — https://webpack.js.org/configuration/resolve/
- Vite: Troubleshooting — https://vitejs.dev/guide/troubleshooting.html
- npm: ERESOLVE — https://docs.npmjs.com/common-errors
- pnpm: Errors — https://pnpm.io/errors
- Yarn: CLI errors — https://yarnpkg.com/advanced/error-codes
- WHATWG: Fetch Standard — https://fetch.spec.whatwg.org/
- IETF: RFC 6455 — The WebSocket Protocol — https://datatracker.ietf.org/doc/html/rfc6455
