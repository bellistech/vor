# The Internals of HTML — WHATWG Parser, DOM, and the Browser Pipeline

> *HTML is not a markup language so much as a forgiveness contract between authors and browsers. The WHATWG specification encodes that forgiveness as an enormous tokenizer state machine, an insertion-mode tree builder, an adoption agency for misnested formatting elements, and a DOM model whose mutation observability is itself part of the platform. Beneath the simple veneer of `<p>hello</p>` lives a parser that recovers from every conceivable corruption, a tree whose live collections re-query on access, a shadow DOM that builds composition without compromising encapsulation, and an event loop whose microtask checkpoint dominates modern web performance debates. This deep dive walks the pipeline from raw bytes to pixels.*

---

## 1. The HTML Parsing Algorithm — Top Level

### Two Coupled State Machines

The WHATWG HTML parser is two state machines operating in lockstep — the **tokenizer** and the **tree-construction** stage. The tokenizer reads characters one at a time and emits tokens (start tag, end tag, character, comment, DOCTYPE, EOF). The tree-construction stage consumes those tokens and mutates the DOM tree, choosing among twenty-three insertion modes that determine how each token is interpreted in the current context.

Crucially, the two stages are not independent — tree construction can change the tokenizer state. When the tree builder encounters a `<script>` start tag, it switches the tokenizer into **script-data state** so that `</` inside the script body is treated as text rather than a tag opener. Without this back-channel, every JavaScript regex containing `</script>` would terminate the script prematurely. The same coupling powers `<style>` (RAWTEXT state), `<title>` and `<textarea>` (RCDATA state), and `<noscript>`/`<iframe>` content.

### The Forgiveness Doctrine

XML's design philosophy was strict: a single malformed character halts parsing with a fatal error. HTML chose the opposite — *every* sequence of bytes must produce a DOM. There are no fatal errors, only **parse errors** which produce diagnostic output but never abort. This commitment is encoded as concrete recovery rules at every state transition. An end tag in a state that does not allow one becomes a parse error and is processed as if the corresponding open tag had been seen. A `<table>` inside a `<p>` triggers implicit `</p>` closure. An EOF mid-attribute is recovered with a final attribute value of empty string. The result: a parser whose output for `<b><i></b></i>` is well-defined and identical across every conforming implementation.

Section 13.2 of the HTML living standard formalizes this — *"This specification defines the parsing rules for HTML documents, whether they are syntactically correct or not."* The forgiveness is not laxity; it is mathematically rigorous, with each malformed input mapping to exactly one DOM tree.

### Top-Level Parser Pseudocode

```javascript
// Conceptual driver loop — the real spec uses two separate state machines
function parseHTML(byteStream) {
  const tokenizer = new Tokenizer(byteStream);
  const treeBuilder = new TreeBuilder();

  while (!tokenizer.atEOF()) {
    const token = tokenizer.nextToken();          // tokenizer state machine
    treeBuilder.processToken(token);              // tree-construction state machine
    if (treeBuilder.requestedTokenizerState) {
      tokenizer.state = treeBuilder.requestedTokenizerState;
      treeBuilder.requestedTokenizerState = null;
    }
  }
  treeBuilder.processToken(EOF);
  return treeBuilder.document;
}
```

### Encoding Sniffing — the Pre-Parser

Before tokenization can begin, the parser must determine the character encoding. The algorithm is layered:

1. Use BOM if present (UTF-8 EF BB BF, UTF-16 BE/LE).
2. Use the HTTP `Content-Type` charset parameter if any.
3. Run the **prescan** — read up to 1024 bytes looking for `<meta charset>` or `<meta http-equiv="content-type">`.
4. Use any explicit user override.
5. Detect by frequency analysis (chardet-style heuristics).
6. Default to environment encoding (typically Windows-1252 or UTF-8 for modern locales).

The prescan is a tiny tokenizer that ignores attribute values it does not need — it must not commit to a tree shape because the encoding it discovers may invalidate every byte processed so far.

---

## 2. Tokenizer States

The tokenizer has approximately **80 states** in the current spec (section 13.2.5). Most are concerned with edge cases — DOCTYPE handling, character references, and the various quoted/unquoted attribute value paths. The core dozen handle the vast majority of real-world parsing.

### Core State Table

| State | Trigger | Emits | Transitions To |
|:---|:---|:---|:---|
| Data | Default text content | Character tokens | Tag open on `<`, character ref on `&` |
| Tag open | After `<` | (none yet) | End tag open on `/`, Tag name on letter, Markup decl on `!`, Bogus comment on `?` |
| End tag open | After `</` | (none yet) | Tag name on letter, Bogus comment otherwise |
| Tag name | After tag-opener letter | Start/end tag (deferred) | Before-attribute on space, Self-closing on `/`, Data on `>` |
| Before attribute name | After tag name + space | (none yet) | Attribute name on letter, Self-closing on `/`, Data on `>` |
| Attribute name | After attr letter | Attribute name (deferred) | Before-value on `=`, After-name on space, Self-closing on `/` |
| Before attribute value | After `=` | (none yet) | Double-quoted on `"`, Single-quoted on `'`, Unquoted otherwise |
| Attribute value (double-quoted) | After `="` | Attribute value chars | After-quoted on `"`, Char ref on `&` |
| Attribute value (single-quoted) | After `='` | Attribute value chars | After-quoted on `'`, Char ref on `&` |
| Attribute value (unquoted) | After `=x` | Attribute value chars | Before-attr on space, Data on `>` |
| RAWTEXT | Inside `<style>` | Character tokens | RAWTEXT less-than on `<` |
| RCDATA | Inside `<title>`, `<textarea>` | Character tokens (with refs) | RCDATA less-than on `<`, char ref on `&` |
| Script data | Inside `<script>` | Character tokens | Script-data less-than on `<` |
| Comment | After `<!--` | (accumulating) | Comment-end on `-` |
| DOCTYPE | After `<!DOCTYPE` | DOCTYPE token (deferred) | Before-name, Name, Public-id, System-id sub-states |

### RAWTEXT vs RCDATA — The Subtle Difference

Both pass through almost-arbitrary content until they see a matching end tag. The difference: **RCDATA** still recognizes character references (`&amp;`, `&#x1F600;`), while **RAWTEXT** does not. This matters for `<title>You &amp; me</title>` — the title shows `You & me`. But for `<style>.x::before { content: "&amp;" }</style>` the entity stays literal because CSS does its own escaping.

### Script Data — The Most Pathological State

Script data has six sub-states to handle the nested-comment edge case from the early Netscape era:

```html
<script>
  // <!--   <script> </script>    -->
  document.write('hello');
</script>
```

Originally, browsers wanted scripts to survive being wrapped in `<!--` `-->` for backwards compatibility with browsers that did not understand `<script>`. The result is a tokenizer maze: **script-data**, **script-data-less-than-sign**, **script-data-end-tag-open**, **script-data-end-tag-name**, **script-data-escape-start**, **script-data-escaped**, **script-data-escaped-dash**, **script-data-escaped-dash-dash**, **script-data-escaped-less-than-sign**, **script-data-escaped-end-tag-open**, **script-data-escaped-end-tag-name**, **script-data-double-escape-start**, **script-data-double-escaped**, **script-data-double-escaped-dash**, **script-data-double-escaped-dash-dash**, **script-data-double-escaped-less-than-sign**, **script-data-double-escape-end**.

Modern advice: use external scripts and CSP, and never write `</script>` in inline JavaScript. If you must, escape it: `"<\/script>"`.

### Character Reference State — Named, Decimal, Hex

```html
&amp;     → &        (named)
&#38;     → &        (decimal)
&#x26;    → &        (hex)
&copy     → ©        (legacy semicolon-less, only certain refs)
&notin;   → ∉        (modern, semicolon required)
```

The spec maintains a **named character reference table** of approximately 2,200 entries. Importantly, some legacy references like `&copy` work without the trailing semicolon, but only for a hardcoded list (about 100 entries). New references introduced after HTML5 require the semicolon — `&commat;` produces `@`, but `&commat` (no semicolon) is a parse error and remains literal.

### Putting It Together — Tokenize a Real Tag

```
input:   <a href="x">y</a>

Data state         '<'  → Tag open state
Tag open state     'a'  → Tag name state (start tag, name="a")
Tag name state     ' '  → Before attribute name state
Before attr name   'h'  → Attribute name state (name="h")
Attribute name     'r'  → Attribute name state (name="hr")
Attribute name     'e'  → Attribute name state (name="hre")
Attribute name     'f'  → Attribute name state (name="href")
Attribute name     '='  → Before attribute value state
Before attr value  '"'  → Attribute value (double-quoted) state
Attr value (dq)    'x'  → Attribute value (double-quoted), append 'x'
Attr value (dq)    '"'  → After attribute value (quoted) state
After attr value   '>'  → Data state, EMIT start tag {a, href=x}
Data state         'y'  → Data state, EMIT character 'y'
Data state         '<'  → Tag open state
Tag open state     '/'  → End tag open state
End tag open       'a'  → Tag name state (end tag, name="a")
Tag name state     '>'  → Data state, EMIT end tag {a}
```

---

## 3. Tree Construction

After tokenization comes tree construction (section 13.2.6). The tree builder maintains:

- **Stack of open elements** — currently unclosed start tags.
- **List of active formatting elements** — `<b>`, `<i>`, `<u>`, etc. that need automatic re-opening across breaks.
- **Insertion mode** — one of 23 modes that dispatches on token type.
- **Original insertion mode** — saved when entering text/InTableText modes.
- **Stack of template insertion modes** — for nested `<template>` content.
- **Head element pointer**, **form element pointer** — quick access to special singletons.
- **Frameset-ok flag** — tracks whether `<frameset>` is still legal.
- **Scripting flag**, **fragment-parse flag**, **foster-parenting flag** — runtime state.

### Insertion Mode Cheat Sheet

| Mode | Entered When | Primary Job |
|:---|:---|:---|
| Initial | Document start | Process DOCTYPE, switch to before-html |
| Before html | After DOCTYPE | Insert `<html>` (auto if missing) |
| Before head | After `<html>` | Insert `<head>` (auto if missing) |
| In head | Inside `<head>` | Process metadata, scripts, styles |
| In head noscript | After `<noscript>` in head | Limited token set |
| After head | After `</head>` | Insert `<body>` (auto if missing) |
| In body | Main content area | The big one — most rules live here |
| Text | Inside RAWTEXT/RCDATA | Accumulate characters |
| In table | Inside `<table>` | Foster-parent stray content |
| In table text | Pending characters in table | Buffer, then dispatch |
| In caption | Inside `<caption>` | Like in-body with caption-aware closure |
| In column group | Inside `<colgroup>` | Only `<col>` and end tag legal |
| In table body | Inside `<tbody>`/`<thead>`/`<tfoot>` | Manage row insertion |
| In row | Inside `<tr>` | Manage cell insertion |
| In cell | Inside `<td>`/`<th>` | Like in-body with cell-close handling |
| In select | Inside `<select>` | Restricted token set |
| In select in table | `<select>` inside table | Auto-close on table tags |
| In template | Inside `<template>` content | Stack of pushed modes |
| After body | After `</body>` | Final whitespace/comments |
| In frameset | Inside `<frameset>` | Frameset descendants only |
| After frameset | After `</frameset>` | Tail handling |
| After after body | After `</html>` | EOF or stray comments |
| After after frameset | After `</html>` post-frameset | Same |

### The In-Body Mode — The Beating Heart

Most pages live almost entirely in **in-body** mode. It is the largest and hairiest insertion mode in the spec, with rules for every legal start and end tag:

- `<p>` — close any open `<p>` first (implicit closure), then push new `<p>`.
- `<li>` — close ancestor `<li>` first if one exists in list-item scope.
- `<a>` — if an `<a>` is in the active formatting list, run the adoption agency, then push new `<a>`.
- `<h1>`–`<h6>` — close any open heading, then push.
- `<form>` — only one form pointer at a time — if already set, ignore.
- `<button>` — close any open button first.

End tags follow analogous rules — `</p>` with no open `<p>` synthesizes one then closes it, ensuring `<p></p>` and `</p>` produce the same DOM (an empty `<p>` element).

### Active Formatting Elements

Formatting elements (`<b>`, `<i>`, `<u>`, `<em>`, `<strong>`, `<font>`, `<a>`, `<s>`, `<small>`, `<big>`, `<tt>`, `<code>`, `<nobr>`, `<strike>`) are tracked in a separate **list of active formatting elements** in addition to the open elements stack. This duality lets the parser re-open them automatically across structure breaks:

```html
<p><b>bold <i>italic</p><p>still both</p>
```

The DOM produced:

```
<p><b>bold <i>italic</i></b></p>
<p><b><i>still both</i></b></p>
```

When `</p>` closes the first paragraph, `<b>` and `<i>` are popped from the open elements stack but remain in the active formatting list. When the next `<p>` opens, the spec's **reconstruct the active formatting elements** algorithm walks the list and pushes new clones of each onto the open elements stack, attaching them inside the new `<p>`. The result is intuitive — the bold/italic styling carries across the paragraph break — and the spec defines it down to the exact tree shape.

### The Adoption Agency Algorithm

When formatting elements are *misnested* — closed in the wrong order relative to other formatting/structural elements — the spec invokes the **adoption agency algorithm**. This is the most notorious section of HTML parsing, sometimes called *"the worst code in the entire web platform"*. It handles cases like:

```html
<b><i></b></i>
```

Naive popping would close `<b>` first, leaving `<i>` orphaned. The adoption agency runs an **eight-step outer loop** with an inner **innermost-formatting-element search**, performing tree surgery: it identifies the formatting element to "adopt" (here `<i>`), re-parents the subtree, clones the formatting element so the styling persists, and updates the active formatting list. The tree produced:

```
<b></b>
<i></i>
```

— with the `<i>` adopted out of `<b>`'s scope and re-emitted at the same level, preserving the author's intent that italic continues.

### Foster Parenting in Tables

When stray content appears inside a `<table>` but outside a cell, the parser performs **foster parenting** — content is removed from its natural position and inserted *before* the table:

```html
<table>oops<tr><td>cell</td></tr></table>
```

The text `oops` is foster-parented before the table:

```
<text>oops</text>
<table>
  <tbody>
    <tr><td>cell</td></tr>
  </tbody>
</table>
```

This rule exists because authors used to write `<table>` then immediately put text or even `<form>` inside the table without a proper cell. Foster parenting recovers those documents into a tree that still renders sensibly.

### Implicit `<tbody>`, `<html>`, `<head>`, `<body>`

The parser silently inserts elements that the author omitted. A document with no `<html>` tag still gets one. A `<table>` with bare `<tr>` children gets an implicit `<tbody>` inserted. This is what makes `<table><tr><td>x</td></tr></table>` valid — the spec inserts `<tbody>` automatically.

---

## 4. The DOM Tree Model

The DOM (specified in `dom.spec.whatwg.org`) is the output of the parser and the input to layout. It is a tree of **Node** objects with a fixed set of subtypes.

### Node Hierarchy

```
EventTarget
└── Node
    ├── Document
    │   └── HTMLDocument / XMLDocument
    ├── DocumentFragment
    │   └── ShadowRoot
    ├── DocumentType
    ├── Element
    │   ├── HTMLElement
    │   │   ├── HTMLDivElement
    │   │   ├── HTMLAnchorElement
    │   │   ├── HTMLInputElement
    │   │   └── ... (one per tag)
    │   ├── SVGElement
    │   └── MathMLElement
    ├── CharacterData
    │   ├── Text
    │   │   └── CDATASection
    │   ├── Comment
    │   └── ProcessingInstruction
    └── Attr  (attached but not a child)
```

### Key Invariants

- A **Document** has at most one DocumentElement child (typically `<html>`) and one DocumentType child.
- A **DocumentFragment** has no parent — it is a portable subtree.
- A **ShadowRoot** is a DocumentFragment with a host element pointer.
- **Attr** is a Node subtype but is *not* a child of its element — `attr.parentNode === null`. Attributes attach via the element's attribute list.
- **Text** nodes are mutable strings — adjacent text nodes are *not* automatically merged unless `normalize()` is called.

### Document vs DocumentFragment

```javascript
const doc = document;                          // the live page document
const frag = document.createDocumentFragment(); // detached, unparented
frag.appendChild(document.createElement('div'));
frag.appendChild(document.createElement('span'));

// Inserting a fragment into the live tree moves its children individually:
document.body.appendChild(frag);
console.log(frag.childNodes.length);  // 0  — fragment is now empty
```

This is the canonical batched-insertion idiom: build the subtree off-document, then attach it in one `appendChild` so that layout/style invalidation runs once.

### querySelector vs getElementById

`document.getElementById('foo')` is O(1) — every Document maintains a hash table of `id` → Element built and updated by the parser and DOM mutation hooks. `document.querySelector('#foo')` is O(n) by spec — it must walk the document in tree order matching the selector — but most engines optimize the `#id` fast-path to O(1) in practice. For complex selectors there is no shortcut.

```javascript
// Equivalent results, very different costs in pathological cases:
const a = document.getElementById('foo');                  // O(1) guaranteed
const b = document.querySelector('#foo');                  // O(n) by spec, O(1) in practice
const c = document.querySelector('.outer .inner > span');  // O(n × selector-depth)
```

### Element vs HTMLElement vs HTMLDivElement

Every DOM element is at minimum an **Element**. HTML elements additionally inherit from **HTMLElement**, which adds properties like `dataset`, `hidden`, `tabIndex`, `accessKey`, `contentEditable`. Each tag has its own subclass — `HTMLAnchorElement` adds `href`, `HTMLInputElement` adds `value`, `checked`, `form`, etc. The parser creates the right subclass based on the start tag.

### Text Node Edge Cases

```javascript
const div = document.createElement('div');
div.textContent = 'hello world';
console.log(div.childNodes.length);           // 1 — one Text node
console.log(div.firstChild.nodeType);         // 3 — Text

div.appendChild(document.createTextNode('!'));
console.log(div.childNodes.length);           // 2 — adjacent but separate
console.log(div.textContent);                 // 'hello world!'

div.normalize();
console.log(div.childNodes.length);           // 1 — merged
```

---

## 5. Live vs Static Collections

A subtle but performance-critical DOM design choice: many traversal results are **live** — they re-query the document on each access — while others are **static snapshots**.

### The Two APIs

| API | Returns | Live? |
|:---|:---|:---:|
| `document.getElementsByTagName('p')` | HTMLCollection | Yes |
| `document.getElementsByClassName('x')` | HTMLCollection | Yes |
| `element.children` | HTMLCollection | Yes |
| `document.forms`, `document.images`, `document.links` | HTMLCollection | Yes |
| `element.querySelectorAll('.x')` | NodeList | No (static) |
| `element.childNodes` | NodeList | Yes (the exception) |

### The Live-Collection Pitfall

```javascript
const items = document.getElementsByClassName('item');
console.log(items.length);  // suppose 5

for (let i = 0; i < items.length; i++) {
  document.body.appendChild(items[i].cloneNode(true));
  // items.length grows to 6 after iteration 0, then 7, 8, ...
  // Infinite loop!
}
```

Because `items` is live, every `appendChild` of a clone increases its length. This burns CPU forever. Fix: snapshot first.

```javascript
// Either iterate the snapshot:
const snapshot = Array.from(document.getElementsByClassName('item'));
for (const item of snapshot) document.body.appendChild(item.cloneNode(true));

// Or use querySelectorAll which is already static:
const items = document.querySelectorAll('.item');
for (const item of items) document.body.appendChild(item.cloneNode(true));
```

### Why Live Collections Exist

In the original DOM Level 1 (1998), `getElementsByTagName` was specified live so that scripts written *before* the document finished loading would still see new elements as they parsed in. With modern script timing (DOMContentLoaded, defer, etc.) this is rarely needed, and the perf cost — every property access potentially re-queries — has driven the modern preference for `querySelectorAll`. The DOM spec keeps live collections for backwards compatibility but explicitly discourages new APIs from returning them.

### Modern Preference

```javascript
// AVOID:
const els = document.getElementsByClassName('x');    // live, surprising

// PREFER:
const els = document.querySelectorAll('.x');          // static, predictable
const arr = [...document.querySelectorAll('.x')];    // real array
```

---

## 6. Custom Elements and Web Components

Custom elements (whatwg HTML section 4.13) let authors define their own element types with full lifecycle integration. Combined with shadow DOM and templates, they form the **Web Components** suite.

### Defining an Autonomous Custom Element

```javascript
class MyCounter extends HTMLElement {
  static observedAttributes = ['count', 'step'];

  constructor() {
    super();
    this._shadow = this.attachShadow({ mode: 'open' });
    this._shadow.innerHTML = `<button>+</button><span></span>`;
    this._button = this._shadow.querySelector('button');
    this._span = this._shadow.querySelector('span');
    this._onClick = () => this.count = this.count + this.step;
  }

  connectedCallback() {
    this._button.addEventListener('click', this._onClick);
    this._render();
  }

  disconnectedCallback() {
    this._button.removeEventListener('click', this._onClick);
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (name === 'count' || name === 'step') this._render();
  }

  adoptedCallback(oldDocument, newDocument) {
    // Called when moved to a new document via document.adoptNode
  }

  get count() { return Number(this.getAttribute('count') ?? 0); }
  set count(v) { this.setAttribute('count', String(v)); }

  get step() { return Number(this.getAttribute('step') ?? 1); }
  set step(v) { this.setAttribute('step', String(v)); }

  _render() {
    this._span.textContent = ` ${this.count}`;
  }
}

customElements.define('my-counter', MyCounter);
```

Usage:

```html
<my-counter count="0" step="1"></my-counter>
```

### The Upgrade Lifecycle

Every element name passes through a state machine:

| State | Meaning |
|:---|:---|
| `undefined` | Name not in registry — element is HTMLUnknownElement |
| `uncustomized` | Name registered but element not yet upgraded |
| `failed` | Constructor threw; never upgrades |
| `precustomized` | In progress |
| `custom` | Successfully upgraded, full lifecycle active |

When `customElements.define('my-counter', MyCounter)` is called *after* the parser has already created `<my-counter>` elements, the spec runs the **upgrade algorithm** for each existing element: re-prototypes it, calls the constructor as a "post-construction" hook, then calls `connectedCallback` if connected. This is why custom elements work correctly regardless of script load order.

### Customized Built-In Elements

Instead of inventing a new tag, you can extend a built-in:

```javascript
class FancyButton extends HTMLButtonElement {
  connectedCallback() {
    this.classList.add('fancy');
  }
}
customElements.define('fancy-button', FancyButton, { extends: 'button' });
```

```html
<button is="fancy-button">Click me</button>
```

This preserves built-in semantics — accessibility, form participation, default styling — while adding custom behavior. Safari does not implement `is=` so these are less portable than autonomous custom elements.

### Form-Associated Custom Elements

```javascript
class MyTextInput extends HTMLElement {
  static formAssociated = true;
  static observedAttributes = ['value'];

  constructor() {
    super();
    this._internals = this.attachInternals();
    this._shadow = this.attachShadow({ mode: 'open' });
    this._shadow.innerHTML = `<input>`;
    this._input = this._shadow.querySelector('input');
    this._input.addEventListener('input', () => {
      this._internals.setFormValue(this._input.value);
    });
  }

  formAssociatedCallback(form) { /* attached to a form */ }
  formDisabledCallback(disabled) { /* form/fieldset disabled */ }
  formResetCallback() { this._input.value = ''; }
  formStateRestoreCallback(state, mode) { this._input.value = state; }
}
customElements.define('my-input', MyTextInput);
```

`attachInternals()` returns an `ElementInternals` object that exposes form-participation methods (`setFormValue`, `setValidity`), the accessibility-state surface (`role`, `ariaLabel`), and shadow-root reference. This is the modern way to build form widgets without using `<input>` directly.

---

## 7. Shadow DOM

Shadow DOM (DOM section 4.8) provides scoped DOM and CSS encapsulation by attaching a **shadow tree** to a host element.

### Open vs Closed

```javascript
const openHost = document.createElement('div');
const openShadow = openHost.attachShadow({ mode: 'open' });
console.log(openHost.shadowRoot);   // ShadowRoot — externally accessible

const closedHost = document.createElement('div');
const closedShadow = closedHost.attachShadow({ mode: 'closed' });
console.log(closedHost.shadowRoot); // null — externally inaccessible
```

Open shadow DOM is the norm. Closed mode is a weak "hide from external scripts" signal, easily defeated by overriding `attachShadow` itself, so it offers minimal real protection — most authors should use open.

### CSS Encapsulation Boundaries

Styles defined outside the shadow do not penetrate inside (except for inheritable properties on the host like `color` and `font-family`). Styles defined inside the shadow do not leak out. Selectors stop at the shadow boundary.

```html
<style>
  /* This selector cannot match elements inside any shadow root */
  .external { color: red; }
</style>

<my-card>
  #shadow-root (open)
    <style>
      /* This is scoped to this shadow root only */
      :host { display: block; padding: 1em; }
      :host([active]) { border-color: blue; }
      :host-context(.dark) { background: #111; color: #eee; }
      ::slotted(h2) { font-size: 1.4em; }      /* light-DOM children */
      ::slotted(*) { margin: 0; }
    </style>
    <slot name="title"></slot>
    <slot></slot>  <!-- default slot -->
</my-card>
```

### Slotting and Composition

```html
<my-card>
  <h2 slot="title">Hello</h2>
  <p>Body text in default slot.</p>
</my-card>
```

The light-DOM children are **distributed** into named slots inside the shadow tree. They remain in the light DOM (their `parentElement` is still `<my-card>`), but they render at the slot's position. The composition is purely visual — DOM tree-walking from the document root never enters the shadow. To traverse, use `element.shadowRoot` or `slot.assignedNodes()`.

### Constructible Stylesheets and adoptedStyleSheets

```javascript
const sheet = new CSSStyleSheet();
sheet.replaceSync(`
  :host { display: block; }
  button { padding: 0.5em 1em; }
`);

class MyButton extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });
    this.shadowRoot.adoptedStyleSheets = [sheet];   // shared, no re-parse
    this.shadowRoot.innerHTML = `<button><slot></slot></button>`;
  }
}
customElements.define('my-button', MyButton);
```

A single `CSSStyleSheet` instance can be adopted by hundreds of shadow roots — no re-parse, no duplicate rule storage. This is the modern replacement for inlining `<style>` in every component instance.

### Focus Traversal Across Shadow Boundaries

`document.activeElement` returns the highest-level shadow host that contains focus, *not* the focused element itself. To find the actual focused element, traverse:

```javascript
function deepActiveElement() {
  let el = document.activeElement;
  while (el && el.shadowRoot && el.shadowRoot.activeElement) {
    el = el.shadowRoot.activeElement;
  }
  return el;
}
```

Tab navigation does cross shadow boundaries — the spec defines a flattened tab-order that visits focusable elements in the slotted/composed tree.

---

## 8. Templates and Slots

### `<template>` — The Inert Subtree

```html
<template id="card">
  <article class="card">
    <h3 class="title"></h3>
    <p class="body"></p>
  </article>
</template>

<script>
  const tmpl = document.getElementById('card');
  console.log(tmpl.content);  // DocumentFragment
  console.log(tmpl.content.querySelector('.title'));  // works
  console.log(tmpl.querySelector('.title'));          // also works (light DOM lookup)

  // Clone for each card:
  function makeCard(title, body) {
    const node = tmpl.content.cloneNode(true);
    node.querySelector('.title').textContent = title;
    node.querySelector('.body').textContent = body;
    return node;
  }

  document.body.appendChild(makeCard('First', 'Hello'));
  document.body.appendChild(makeCard('Second', 'World'));
</script>
```

The parser puts `<template>` content into a separate **template contents document** rather than the main document, so:

- `<script>` inside a template does *not* execute until cloned and inserted.
- `<img>` inside a template does *not* fetch.
- `<style>` inside a template is parsed but does not apply.

This makes templates safe to define anywhere without runtime cost.

### Named and Default Slots

```html
<template id="page-tmpl">
  <header><slot name="header">Default header</slot></header>
  <main><slot></slot></main>
  <footer><slot name="footer">Default footer</slot></footer>
</template>

<my-page>
  <h1 slot="header">Custom header</h1>
  <p>Body content goes in the default slot.</p>
  <small slot="footer">Copyright 2026</small>
</my-page>
```

The default slot (no `name` attribute) catches anything without an explicit `slot=` attribute. Each slot can have **fallback content** that renders when no light-DOM children are slotted in.

### `slotchange` Event

```javascript
shadowRoot.querySelector('slot[name="header"]').addEventListener('slotchange', e => {
  const assigned = e.target.assignedNodes({ flatten: true });
  console.log(`Header now has ${assigned.length} nodes`);
});
```

---

## 9. Living-Standard HTML5 Surface

### No More Versioned Snapshots

HTML5 was the last numbered version. Since 2014 the spec has been a **living standard** at `html.spec.whatwg.org`, updated continuously by editor commits with no version field. Browsers ship features as they reach **Cumulative Reference** maturity. The W3C maintains a periodic snapshot but the canonical specification is the WHATWG living standard.

### Content Models — The Categorization Layer

Every HTML element belongs to zero or more **content categories** that describe what it can contain and where it can appear.

| Category | Members (sample) | Allowed Where |
|:---|:---|:---|
| Metadata | `<title>`, `<meta>`, `<link>`, `<style>`, `<script>` | `<head>` |
| Flow | `<p>`, `<div>`, `<table>`, `<ul>`, headings | Most body locations |
| Sectioning | `<article>`, `<section>`, `<nav>`, `<aside>` | Subset of flow |
| Heading | `<h1>`–`<h6>`, `<hgroup>` | Subset of flow |
| Phrasing | `<a>`, `<span>`, `<em>`, `<strong>`, `<img>`, `<input>`, `<button>` | Inside paragraph-equivalent contexts |
| Embedded | `<img>`, `<iframe>`, `<video>`, `<canvas>`, `<svg>` | Subset of phrasing |
| Interactive | `<a href>`, `<button>`, `<input>`, `<select>`, `<textarea>` | Subset of flow/phrasing |
| Scripting | `<script>`, `<noscript>`, `<template>` | Many places |

### Content Model Per Element

The spec defines a **content model** per element — a regex-like description of what children are allowed:

- `<p>`'s content model: phrasing content. Disallows nested `<div>`, `<table>`.
- `<ul>`'s content model: zero or more `<li>` (and script-supporting elements).
- `<table>`'s content model: optional `<caption>`, optional `<colgroup>`s, optional `<thead>`, then `<tbody>` or `<tr>`s, then optional `<tfoot>`.
- `<button>`'s content model: phrasing content but no interactive content (cannot nest `<a>`, `<button>`, `<input>` inside).

### Categorization by Context

Some elements have **transparent content models** — they take on the content model of their parent. `<a>` is transparent — `<a>` inside `<p>` is phrasing, `<a>` inside `<div>` is flow. This is what allows `<a>` to wrap arbitrary content (the author's "block link" idiom) while still being constrained correctly within phrasing contexts.

### The Conformance Levels

- **MUST / MUST NOT** — required for conformance. Authoring violations.
- **SHOULD / SHOULD NOT** — recommended. Best practice.
- **MAY** — optional permission.

A user agent that does not implement a MUST is non-conformant. An author who violates a MUST creates an invalid document — but the parser still produces a DOM (the forgiveness contract).

---

## 10. Accessibility Tree

The accessibility tree (or **a11y tree**) is a *separate* tree built by the user agent from the DOM and used by assistive technologies (screen readers, voice control, switch access). It is not the DOM — it has its own node types, its own subset of the document, and its own semantics.

### What the Browser Does

For every visible DOM element, the browser computes:

- **Role** — what kind of UI element (button, link, heading, region, etc.)
- **Name** — the accessible label (computed via `aria-labelledby`, `aria-label`, associated `<label>`, text content, alt text, or title)
- **Description** — supplementary context (`aria-describedby`)
- **State** — checked, expanded, disabled, busy, selected
- **Properties** — required, autocomplete, level (heading/list), valuemin/max/now (range)

### Native vs ARIA Roles

| HTML | Implicit Role | ARIA Equivalent |
|:---|:---|:---|
| `<button>` | button | `role="button"` |
| `<a href>` | link | `role="link"` |
| `<a>` (no href) | (none) | needs `role="link"` if interactive |
| `<input type="checkbox">` | checkbox | `role="checkbox"` |
| `<input type="radio">` | radio | `role="radio"` |
| `<input type="range">` | slider | `role="slider"` |
| `<select>` | combobox | `role="combobox"` |
| `<nav>` | navigation | `role="navigation"` |
| `<main>` | main | `role="main"` |
| `<aside>` | complementary | `role="complementary"` |
| `<h1>`–`<h6>` | heading | `role="heading"` aria-level |
| `<ul>`/`<ol>` | list | `role="list"` |
| `<li>` | listitem | `role="listitem"` |
| `<table>` | table | `role="table"` |
| `<tr>` | row | `role="row"` |
| `<th>` | columnheader/rowheader | `role="columnheader"` |
| `<dialog>` | dialog | `role="dialog"` |
| `<details>`/`<summary>` | group/button | (no exact equivalent) |
| `<div>`, `<span>` | (none — generic) | (none unless added) |

### The Cardinal Rule

*Use semantic HTML first; reach for ARIA only when no native element fits.* `<button onclick>` is universally accessible — `<div role="button" tabindex="0" onclick onkeydown>` is a leaky reimplementation that misses focus rings, form participation, and Enter/Space differences across user agents.

### Accessible Name Computation

The W3C **Accessible Name and Description Computation 1.2** spec defines the precedence:

1. `aria-labelledby` (highest priority) — points to one or more elements whose text becomes the name.
2. `aria-label` — direct string.
3. Native labelling — `<label for>`, `<label>` wrap, table `<caption>`, fieldset `<legend>`, `alt` on `<img>`.
4. Text content (for many elements).
5. `title` attribute (lowest, often ignored by screen readers).

### Accessibility Object Model (AOM) Proposal

A draft proposal lets script set accessibility properties directly without ARIA attributes:

```javascript
const internals = this.attachInternals();
internals.role = 'button';
internals.ariaLabel = 'Close dialog';
internals.ariaPressed = 'false';
```

This is shipped today as part of `ElementInternals` (form-associated custom elements section) — the broader AOM remains experimental.

---

## 11. Event Model

Events are central to the DOM. An event flows in three phases: capture (root-down), at-target, bubble (target-up).

### Phase Diagram

```
                document
                   |
                   v   <-- capture phase
                  body
                   |
                   v
                  div
                   |
                   v
                button   <-- at-target
                   |
                   ^   <-- bubble phase
                  div
                   ^
                  body
                   ^
                document
```

### addEventListener Options

```javascript
button.addEventListener('click', handler);                    // bubble phase by default
button.addEventListener('click', handler, true);              // capture phase
button.addEventListener('click', handler, { capture: true });
button.addEventListener('click', handler, { once: true });    // auto-remove after first fire
button.addEventListener('click', handler, { passive: true }); // promises not to preventDefault

const ac = new AbortController();
button.addEventListener('click', handler, { signal: ac.signal });
ac.abort();  // removes the listener
```

### Stopping Propagation

```javascript
e.preventDefault();             // cancels default action (form submit, anchor nav)
e.stopPropagation();            // stops further bubbling/capturing
e.stopImmediatePropagation();   // also stops other listeners on same target/phase
```

### Trusted vs Synthetic

```javascript
button.addEventListener('click', e => {
  console.log(e.isTrusted);  // true for real user clicks, false for dispatchEvent
});

// Synthetic — isTrusted false:
button.dispatchEvent(new MouseEvent('click', { bubbles: true }));
```

Many sensitive APIs (Clipboard, FullScreen, Notification permission) require user activation — only trusted events count. This is why you cannot simulate a click to bypass permission prompts.

### CustomEvent

```javascript
class Cart extends HTMLElement {
  add(item) {
    this.items.push(item);
    this.dispatchEvent(new CustomEvent('item-added', {
      detail: { item, total: this.items.length },
      bubbles: true,
      composed: true,        // crosses shadow DOM boundary
    }));
  }
}

document.addEventListener('item-added', e => {
  console.log(e.detail.total);
});
```

`composed: true` is required for events fired from inside a shadow tree to escape the shadow boundary and bubble through the host's ancestors.

### Passive Listeners and Scroll Performance

```javascript
// Bad — blocks scrolling on every wheel event:
window.addEventListener('wheel', handler);

// Good — promises not to call preventDefault, browser can scroll immediately:
window.addEventListener('wheel', handler, { passive: true });
```

By default `touchstart`, `touchmove`, `wheel` are non-passive — the browser must wait for the handler to complete before deciding whether to scroll, because the handler might call `preventDefault()`. With `passive: true` the browser scrolls on the compositor thread without waiting. Modern Chrome treats `touchstart` and `touchmove` as passive by default for documents added to the root.

---

## 12. Microtasks vs Macrotasks

The HTML event loop has two queue tiers: **task queue(s)** and the **microtask queue**. Understanding the difference is critical for predicting code execution order.

### The Loop Pseudocode

```javascript
while (true) {
  const task = dequeueOldestTask();   // one macrotask per turn
  execute(task);
  while (microtaskQueue.length > 0) {
    execute(microtaskQueue.shift()); // drain entire microtask queue
  }
  if (renderingOpportunity()) {
    runResizeObservers();
    runIntersectionObservers();
    requestAnimationFrameCallbacks();
    style();
    layout();
    paint();
  }
}
```

### Microtasks

Created by:

- `Promise.then`/`catch`/`finally` callbacks
- `queueMicrotask(fn)`
- `MutationObserver` callbacks
- The `await` resumption in async functions

Microtasks run **immediately after** the current synchronous code completes, *before* any other macrotask, *before* rendering.

### Macrotasks

Created by:

- `setTimeout`, `setInterval`
- `setImmediate` (legacy IE)
- `requestAnimationFrame` (special — runs in rendering opportunity)
- I/O callbacks (XHR, fetch microtask queues then settles macrotask)
- UI events (click, keydown)
- `postMessage` (cross-window) and `MessageChannel`

### Canonical Example

```javascript
console.log('1');

setTimeout(() => console.log('2'), 0);

Promise.resolve().then(() => console.log('3'));

queueMicrotask(() => console.log('4'));

console.log('5');

// Output: 1 5 3 4 2
//
// 1, 5: synchronous, executed inline
// 3, 4: microtasks, drained after current sync completes
// 2: macrotask (setTimeout), waits for next loop turn
```

### Why setTimeout(fn, 0) Is Slower Than queueMicrotask

`setTimeout` schedules a macrotask — the current task must complete, the entire microtask queue drains, rendering may run, *then* the timer task fires. `queueMicrotask` runs in the same task. For "do this just-after-current-call" semantics, `queueMicrotask` is one to two orders of magnitude faster.

### requestAnimationFrame

```javascript
function animate(timestamp) {
  // timestamp is DOMHighResTimeStamp (sub-millisecond)
  element.style.transform = `translateX(${(timestamp / 16) % 100}px)`;
  requestAnimationFrame(animate);
}
requestAnimationFrame(animate);
```

`requestAnimationFrame` callbacks run during the rendering opportunity, just *before* style/layout/paint. They are guaranteed to run at most once per frame (typically 60Hz = every 16.67ms, or 120Hz = every 8.33ms on high-refresh displays). This is the canonical place to read layout (`getBoundingClientRect`) and write styles together — both happen within one frame.

---

## 13. Mutation Observers

Mutation observers (DOM section 4.3.4) replace the deprecated **Mutation Events** (DOMNodeInserted, DOMNodeRemoved, DOMSubtreeModified) which fired synchronously on every change and tanked performance.

### The API

```javascript
const observer = new MutationObserver(mutations => {
  for (const m of mutations) {
    console.log(m.type);             // 'childList', 'attributes', 'characterData'
    console.log(m.target);
    console.log(m.addedNodes);
    console.log(m.removedNodes);
    console.log(m.attributeName);
    console.log(m.oldValue);
  }
});

observer.observe(document.body, {
  childList: true,            // observe direct children
  attributes: true,           // observe attribute changes
  characterData: true,        // observe text node content changes
  subtree: true,              // observe entire subtree
  attributeOldValue: true,    // record old attribute values
  characterDataOldValue: true,
  attributeFilter: ['class', 'data-state'],  // only these attributes
});

// Later:
observer.disconnect();
const pending = observer.takeRecords();   // any unconsumed mutations
```

### Batched Delivery

Mutation observer callbacks run as **microtasks** — they are batched within a single event-loop turn and delivered together. A series of synchronous DOM mutations produces one callback with all the records, not one callback per mutation. This makes them cheap enough to use in production framework reconciliation.

```javascript
const observer = new MutationObserver(records => {
  console.log(`Got ${records.length} mutations`);
});
observer.observe(document.body, { childList: true });

// All synchronous — one batched callback:
document.body.appendChild(document.createElement('div'));
document.body.appendChild(document.createElement('div'));
document.body.appendChild(document.createElement('div'));
// → "Got 3 mutations" (microtask)
```

### Common Use Cases

- **Live-collection replacement** — observe a container, maintain a derived list.
- **Framework reconciliation** — React's hydration uses MutationObserver to detect server/client divergence.
- **Third-party widget cleanup** — observe the host page for the widget being removed.
- **Lazy initialization** — instantiate a component when it appears in the DOM.

```javascript
// Lazy-init pattern:
new MutationObserver(records => {
  for (const r of records) {
    for (const node of r.addedNodes) {
      if (node.nodeType === 1 && node.matches('[data-component=carousel]')) {
        initCarousel(node);
      }
    }
  }
}).observe(document.body, { childList: true, subtree: true });
```

---

## 14. Resource Loading

### The Speculative Parser (Preload Scanner)

When the main parser blocks on a synchronous `<script>` (waiting for it to download and execute), a **preload scanner** runs ahead of the main parser tokenizing the rest of the document looking for `<link rel="stylesheet">`, `<script src>`, and `<img src>`. Discovered resources are kicked off downloading immediately. This is why `<script>` blocking-the-parser does *not* block resource discovery — the speculative parser is already ahead.

### Script Load Modes

```html
<!-- Default — parser-blocking, executed in order -->
<script src="a.js"></script>

<!-- Async — downloaded in parallel, executed as soon as ready (out of order) -->
<script src="a.js" async></script>

<!-- Defer — downloaded in parallel, executed in order after parsing complete -->
<script src="a.js" defer></script>

<!-- Module — defer-by-default, supports import/export -->
<script src="a.js" type="module"></script>

<!-- Module with explicit async -->
<script src="a.js" type="module" async></script>

<!-- Import map -->
<script type="importmap">
{ "imports": { "lodash": "https://cdn.skypack.dev/lodash" } }
</script>
```

### Ordering Rules Table

| Mode | Downloaded | Executed | Blocks Parser |
|:---|:---|:---|:---:|
| `<script>` | When encountered | Immediately on download | Yes |
| `<script async>` | In parallel | As soon as downloaded (out-of-order) | No |
| `<script defer>` | In parallel | After DOMContentLoaded, in source order | No |
| `<script type="module">` | In parallel | After DOMContentLoaded, in source order | No |
| `<script type="module" async>` | In parallel | As soon as ready | No |

### Link Resource Hints

```html
<!-- Pre-resolve DNS, no connection -->
<link rel="dns-prefetch" href="//cdn.example.com">

<!-- DNS + TCP + TLS, ready for first byte -->
<link rel="preconnect" href="https://cdn.example.com" crossorigin>

<!-- High-priority, current-navigation download -->
<link rel="preload" href="/critical.js" as="script">
<link rel="preload" href="/font.woff2" as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/hero.jpg" as="image">

<!-- Module preload (ES modules with their dependency tree) -->
<link rel="modulepreload" href="/app.js">

<!-- Low-priority, next-navigation -->
<link rel="prefetch" href="/next-page.html">

<!-- Stylesheet — parser-blocking for layout, render-blocking by default -->
<link rel="stylesheet" href="/styles.css">

<!-- Stylesheet with non-blocking trick -->
<link rel="preload" href="/styles.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
```

### Parser-Blocking vs Render-Blocking

- **Parser-blocking**: a synchronous `<script>` halts HTML parsing until it downloads and executes. The DOM tree stops growing.
- **Render-blocking**: a CSS `<link>` stops the browser from rendering anything until it loads. The DOM continues to grow but no pixels paint.

The two are *not* the same. CSS does not block HTML parsing — but a `<script>` that runs *after* a stylesheet *will* block parsing, because synchronous script execution must wait for pending stylesheets to apply (the script might query styles).

```html
<link rel="stylesheet" href="big.css">     <!-- render-blocking, not parser-blocking -->
<p>This renders only after big.css loads, but parsing continues</p>
<script>console.log(getComputedStyle(p));</script>  <!-- parser-blocked until big.css loads -->
```

---

## 15. Critical Rendering Path

The pipeline from byte to pixel:

```
Bytes  ──> HTML parse ──> DOM
                              \
                               > Render tree ──> Layout ──> Paint ──> Composite ──> Pixels
                              /
Bytes  ──> CSS parse  ──> CSSOM
```

### Stages

1. **HTML → DOM** — tokenize, build tree.
2. **CSS → CSSOM** — same model as DOM but for stylesheets. Render-blocking.
3. **Render tree** — DOM ∩ CSSOM, excluding `display:none` and elements with no visual representation. Pseudo-elements (`::before`, `::after`) are added.
4. **Layout (reflow)** — geometry: compute the box-model size and position of every render-tree node.
5. **Paint** — generate display list (commands like "fill rect", "draw text").
6. **Composite** — assemble layers (transform, opacity, position:fixed) on the GPU and present.

### Triggers

| Change | Triggers |
|:---|:---|
| `width`, `height`, `padding` | Layout + paint + composite |
| `color`, `background-color`, `box-shadow` | Paint + composite |
| `transform`, `opacity` (on a layer) | Composite only |
| `top`/`left`/`bottom`/`right` | Layout + paint + composite |
| `left` on `position:absolute` with `transform` hack | Composite only |

The performance optimization **"animate transform/opacity, not left/top"** comes from this table. Animating `transform` skips layout and paint, running entirely on the compositor at full frame rate. Animating `left` triggers full reflow every frame.

### Web Vitals — User-Centric Metrics

| Metric | Measures | Target |
|:---|:---|:---|
| **FCP** (First Contentful Paint) | First text/image paint | < 1.8s |
| **LCP** (Largest Contentful Paint) | Hero image / heading paint | < 2.5s |
| **CLS** (Cumulative Layout Shift) | Sum of unexpected layout shifts | < 0.1 |
| **INP** (Interaction to Next Paint) | Worst input-to-paint latency | < 200ms |
| **TTFB** (Time to First Byte) | Server response | < 0.8s |
| **FID** (deprecated, replaced by INP) | First-input delay | n/a |

### The "Layout Thrash" Anti-Pattern

```javascript
// BAD: triggers layout per iteration (read forces layout, write invalidates)
for (let i = 0; i < items.length; i++) {
  items[i].style.width = items[i].offsetWidth + 10 + 'px';
  // ^ read, ^ write, read, write, read, write... each read forces layout
}

// GOOD: batch reads, then batch writes
const widths = items.map(i => i.offsetWidth);   // one layout
items.forEach((i, idx) => i.style.width = (widths[idx] + 10) + 'px');  // one layout
```

---

## 16. Iframes and Cross-Origin

### The Navigable / Browsing-Context Model

Each `<iframe>` is a separate **navigable** (formerly *browsing context* in older spec terminology) with its own document, its own JavaScript realm, its own event loop tier (process-isolated in modern browsers). The DOM tree of the parent does *not* contain the iframe's DOM — only the `<iframe>` element. To access the iframe's DOM (when same-origin):

```javascript
const frame = document.querySelector('iframe');
const innerDoc = frame.contentDocument;     // null if cross-origin
const innerWin = frame.contentWindow;
```

### sandbox Attribute

```html
<!-- Maximum restriction -->
<iframe src="x.html" sandbox></iframe>

<!-- Partial restoration -->
<iframe src="x.html" sandbox="allow-scripts allow-same-origin"></iframe>
```

| Flag | Effect |
|:---|:---|
| `allow-scripts` | Permits JavaScript to run |
| `allow-forms` | Permits form submission |
| `allow-popups` | Permits opening new windows |
| `allow-same-origin` | Treats as same-origin (otherwise unique opaque origin) |
| `allow-top-navigation` | Permits changing the top window's location |
| `allow-pointer-lock` | Permits pointer lock |
| `allow-modals` | Permits alert/confirm/prompt |

A naked `sandbox` (no flags) creates a maximally restricted iframe — no scripts, no forms, treated as a unique origin.

### postMessage — Cross-Origin Communication

```javascript
// Parent
const frame = document.querySelector('iframe');
frame.contentWindow.postMessage({ type: 'ping', data: 42 }, 'https://child.example.com');

window.addEventListener('message', e => {
  if (e.origin !== 'https://child.example.com') return;
  console.log('reply:', e.data);
});

// Child
window.addEventListener('message', e => {
  if (e.origin !== 'https://parent.example.com') return;
  e.source.postMessage({ type: 'pong', data: e.data.data * 2 }, e.origin);
});
```

The second argument to `postMessage` is the **target origin** — a strict origin string or `*` (avoid `*` for sensitive data, anyone listening on the other side could intercept).

### Same-Origin Policy

Two URLs are same-origin if they share **scheme + host + port**. Any difference produces a cross-origin pair, which restricts:

- DOM access via `iframe.contentDocument`
- Script reading XHR/fetch responses
- Reading `<canvas>` pixel data (taint)
- Reading `<img>` pixel data via canvas

### CORS — Cross-Origin Resource Sharing

A server opts-in to cross-origin reads by setting:

```
Access-Control-Allow-Origin: https://app.example.com
```

For non-simple requests (POST with JSON, custom headers), browsers send a **preflight** OPTIONS request first:

```
OPTIONS /api HTTP/1.1
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: content-type, x-token
```

The server responds with the allowed methods/headers; the actual POST follows.

### CORP, COOP, COEP — Cross-Origin Isolation

| Header | Purpose |
|:---|:---|
| **CORP** (Cross-Origin-Resource-Policy) | A resource opts in to being embedded by other origins |
| **COOP** (Cross-Origin-Opener-Policy) | A top-level document opts out of sharing a browsing context group with cross-origin openers |
| **COEP** (Cross-Origin-Embedder-Policy) | A top-level document refuses to load any cross-origin resource that does not opt in via CORP/CORS |

When a page sets both COOP and COEP appropriately, the browser puts it in **cross-origin isolation**, which unlocks `SharedArrayBuffer`, high-resolution `performance.now()`, and `Performance.measureUserAgentSpecificMemory()`. Spectre/Meltdown mitigations (process isolation) require this opt-in.

```http
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
Cross-Origin-Resource-Policy: same-origin
```

```javascript
// Inside a cross-origin-isolated page:
console.log(crossOriginIsolated);  // true
const sab = new SharedArrayBuffer(1024);  // works
```

---

## 17. Forms — the WHATWG Spec Layer

Forms are covered in the practical `html-forms` sheet — here is the cross-reference of the deeper internals.

### Implicit Submission

A form with multiple inputs and a single text input submits when Enter is pressed in that input. With multiple text inputs, Enter in one input submits the form via the first `type="submit"` button, or fails silently if none exists. This is spec-defined behavior, not a quirk.

### The formdata Event

```javascript
form.addEventListener('formdata', e => {
  // Modify the FormData before submission
  e.formData.append('csrf', getCsrfToken());
  e.formData.delete('honeypot');
});
form.requestSubmit();  // triggers formdata then submit
```

Custom elements with `formAssociated = true` participate via `attachInternals().setFormValue()` — their value lands in the FormData automatically.

### Constraint Validation API

```javascript
input.required = true;
input.pattern = '\\d{4}';
input.minLength = 4;
input.maxLength = 4;

console.log(input.validity);
// {
//   valueMissing: false,
//   typeMismatch: false,
//   patternMismatch: true,
//   tooShort: false,
//   tooLong: false,
//   rangeUnderflow: false,
//   rangeOverflow: false,
//   stepMismatch: false,
//   badInput: false,
//   customError: false,
//   valid: false,
// }

input.setCustomValidity('Must be exactly 4 digits');
input.reportValidity();   // shows tooltip
input.checkValidity();    // returns boolean, fires invalid event
```

---

## 18. Common Performance Patterns

### Read-Then-Write Discipline

```javascript
// BAD: forces layout 100 times
for (const el of elements) {
  el.style.width = el.offsetWidth + 10 + 'px';
}

// GOOD: forces layout twice
const widths = elements.map(el => el.offsetWidth);
elements.forEach((el, i) => el.style.width = (widths[i] + 10) + 'px');
```

### Batched DOM Insertion via DocumentFragment

```javascript
// BAD: layout/style recalc per iteration
for (let i = 0; i < 1000; i++) {
  const li = document.createElement('li');
  li.textContent = `Item ${i}`;
  list.appendChild(li);   // each one triggers work
}

// GOOD: build off-document, attach once
const frag = document.createDocumentFragment();
for (let i = 0; i < 1000; i++) {
  const li = document.createElement('li');
  li.textContent = `Item ${i}`;
  frag.appendChild(li);
}
list.appendChild(frag);   // single attachment
```

### IntersectionObserver — Visibility Without Polling

```javascript
const io = new IntersectionObserver(entries => {
  for (const e of entries) {
    if (e.isIntersecting) {
      e.target.src = e.target.dataset.src;   // lazy-load
      io.unobserve(e.target);
    }
  }
}, { rootMargin: '100px', threshold: 0 });

document.querySelectorAll('img[data-src]').forEach(img => io.observe(img));
```

### ResizeObserver — Element Size Without Layout Thrash

```javascript
const ro = new ResizeObserver(entries => {
  for (const e of entries) {
    e.target.dataset.width = e.contentRect.width;
  }
});
ro.observe(document.querySelector('.responsive'));
```

ResizeObserver fires after layout but before paint, in the rendering opportunity. Modifications inside the callback may re-trigger layout in the same frame — the spec defines a depth bound to prevent infinite loops.

### content-visibility: auto

```css
.long-list-item { content-visibility: auto; contain-intrinsic-size: 200px; }
```

Skips rendering work for offscreen elements until they approach the viewport. Saves layout, paint, and even style computation. Pair with `contain-intrinsic-size` to prevent layout shifts when items haven't been laid out yet.

### will-change — Sparingly

```css
.about-to-animate { will-change: transform; }
```

Hints to the browser to put this element on its own composited layer. Overuse blows up GPU memory — apply only just before the animation starts and remove afterwards.

```javascript
button.addEventListener('mouseenter', () => button.style.willChange = 'transform');
button.addEventListener('animationend', () => button.style.willChange = 'auto');
```

### OffscreenCanvas — Workers

```javascript
// main thread
const canvas = document.querySelector('canvas');
const offscreen = canvas.transferControlToOffscreen();
const worker = new Worker('renderer.js');
worker.postMessage({ canvas: offscreen }, [offscreen]);

// renderer.js
self.onmessage = e => {
  const ctx = e.data.canvas.getContext('2d');
  function frame() {
    ctx.fillStyle = 'red';
    ctx.fillRect(0, 0, 100, 100);
    requestAnimationFrame(frame);
  }
  frame();
};
```

Offloads rendering from the main thread, keeping the UI responsive even under heavy paint.

---

## 19. Idioms at the Internals Depth

### Treat the DOM as Canonical State

Frameworks like React maintain a virtual DOM as a *diff buffer*, but the truth lives in the real DOM. When code reads from the virtual shadow without flushing the real DOM, bugs follow. The discipline:

1. Mutate the DOM (or your virtual layer) deterministically.
2. Read back from the DOM only after the mutation has been applied (next animation frame, microtask, or after explicit flush).
3. Never assume a property you set still equals what you set, especially `value` on form elements which the user can change.

### Create-Fragment-Then-Append-Once

The fragment idiom (above) is so important it warrants its own section. The win is not just performance — it is *atomicity*. Half-built trees are never visible to other code (event handlers, MutationObservers, IntersectionObserver). The whole subtree appears in one tick.

### requestAnimationFrame for Synced Visual Updates

```javascript
// Read layout in rAF, write in the same rAF — no thrash
requestAnimationFrame(() => {
  const rect = el.getBoundingClientRect();    // layout once
  el.style.transform = `translate(${rect.width}px, 0)`;  // write
});
```

For coordinated animations across multiple elements, use a single rAF and update all elements together.

### structuredClone for Deep-Clone

```javascript
const original = { a: 1, b: { c: 2 }, d: [1, 2, 3] };
const copy = structuredClone(original);
copy.b.c = 999;
console.log(original.b.c);  // 2 — original unchanged
```

Replaces the `JSON.parse(JSON.stringify(x))` hack — handles Map, Set, Date, ArrayBuffer, RegExp, Blob, File, ImageData, and is part of the structured clone algorithm used by postMessage and IndexedDB.

### Element.cloneNode(deep) Pitfalls

```javascript
const original = document.querySelector('input');
original.value = 'hello';                       // user typed this
const clone = original.cloneNode(true);
console.log(clone.value);                       // empty! — value is property, not attribute
console.log(original.getAttribute('value'));    // also empty (or default)
```

`cloneNode` clones *attributes*, not properties. For form elements, the user's typed value lives in the property, not the attribute. To clone with the runtime value, write the property to the attribute first or copy properties manually.

### Avoiding Silent Failures with attachShadow

```javascript
class MyEl extends HTMLElement {
  constructor() {
    super();
    if (this.shadowRoot) return;   // already upgraded? skip re-attach
    this.attachShadow({ mode: 'open' });
  }
}
```

Calling `attachShadow` twice throws — defend against accidental upgrades during hot-reload.

### Defining `is` for SSR Hydration

```html
<!-- Server-rendered -->
<button is="fancy-button" class="fancy">Click</button>
```

The element is parsed before the script runs. When `customElements.define('fancy-button', FancyButton, { extends: 'button' })` finally executes, the parser-built `<button is="fancy-button">` upgrades. This is the canonical SSR-friendly customized-built-in pattern.

### Querying Inside Shadow Boundaries

```javascript
// document.querySelector does NOT cross shadow boundaries:
document.querySelector('my-card .button');   // null — inside shadow

// Walk explicitly:
const card = document.querySelector('my-card');
const button = card.shadowRoot.querySelector('.button');
```

For library authors writing utilities that traverse the *flattened* tree:

```javascript
function deepQuerySelectorAll(root, selector) {
  const results = [];
  function walk(node) {
    if (node.matches?.(selector)) results.push(node);
    if (node.shadowRoot) walk(node.shadowRoot);
    for (const child of node.children ?? []) walk(child);
  }
  walk(root);
  return results;
}
```

---

## Prerequisites

- HTML basics — tags, attributes, the document tree (see the practical `html` sheet)
- JavaScript fundamentals — classes, prototypes, async/await, Promises
- CSS box model and selectors at the basic level
- Understanding of the HTTP request/response cycle and resource loading

## Complexity

- **Beginner:** DOM tree shape, getElementById/querySelector basics, addEventListener, defer/async on `<script>`
- **Intermediate:** Live vs static collections, the event loop and microtasks, the critical rendering path, MutationObserver for change detection, basic shadow DOM with attachShadow and slots
- **Advanced:** WHATWG parser state machine, insertion modes, the adoption agency algorithm, foster parenting, custom-element form association, ElementInternals, constructible stylesheets, cross-origin isolation (COOP/COEP), accessibility tree traversal vs DOM traversal
- **Expert:** Implementing a conformant HTML parser, designing a custom element library that survives SSR hydration without flashing, debugging layout thrash via Performance.measure, instrumenting paint/composite via the Long Tasks API and PerformanceObserver

## See Also

- html (sheet) — the practical HTML reference (tags, attributes, syntax)
- html-forms — forms, inputs, validation, the constraint validation API in depth
- css — selector engine, specificity, cascade, CSSOM
- css-layout — flexbox, grid, the box model, formatting contexts
- javascript — the language layer, async/await, Promises, the event loop
- polyglot — language comparison reference

## References

- WHATWG HTML Living Standard — `https://html.spec.whatwg.org/` — section 13 covers parsing in full detail
- WHATWG DOM Living Standard — `https://dom.spec.whatwg.org/` — defines Node, Element, Document, ShadowRoot, MutationObserver, the event dispatch algorithm
- WHATWG Infra Standard — `https://infra.spec.whatwg.org/` — defines the primitive types (lists, strings, ordered maps) used by the other specs
- W3C ARIA 1.2 — `https://www.w3.org/TR/wai-aria-1.2/` — accessibility roles, states, properties
- W3C Accessible Name and Description Computation — `https://www.w3.org/TR/accname-1.2/`
- web.dev/learn — `https://web.dev/learn/` — Google's web platform learning path including performance, accessibility, web components
- MDN Web Docs — `https://developer.mozilla.org/en-US/docs/Web/HTML` — practical HTML reference
- MDN DOM reference — `https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model`
- MDN Web Components — `https://developer.mozilla.org/en-US/docs/Web/API/Web_components`
- HTML Standard, section 4.13 — Custom Elements
- HTML Standard, section 13.2.5 — Tokenization (state-by-state spec)
- HTML Standard, section 13.2.6 — Tree construction (insertion modes)
- HTML Standard, section 8.1 — The HTML Event Loop
- DOM Standard, section 4.8 — Shadow trees and slottable composition
- DOM Standard, section 4.3.4 — Mutation Observers
- Patrick Brosset — *"Inside look at modern web browser"* (Google web.dev series)
- Paul Lewis — *"What Forces Layout / Reflow"* — the canonical list of layout-triggering DOM properties
