# The Internals of CSS — Cascade, Specificity, Computed Values, and the Style Engine

> *CSS is not a language so much as a constraint-and-resolution pipeline. The browser ingests bytes, parses them into a CSSOM, matches selectors against the DOM, walks the cascade ladder to a winning declaration per property per element, then resolves that declaration through five formal value stages — declared, cascaded, specified, computed, used, and actual. Understanding the internals is understanding which stage owns which behavior. Layout topics (flexbox, grid, positioning) live in the sibling deep-dive `css-layout`; this page is exclusively about the engine.*

---

## 1. The Style Engine Pipeline

The CSS Cascading and Inheritance Module Level 5 (W3C `css-cascade-5`, sections 4 and 4.5) defines a strictly ordered pipeline. Each stage produces values consumed by the next. If you know the pipeline, every CSS surprise can be located on it.

### 1.1 The Six Stages

```
CSS bytes
  → tokenizer (CSS Syntax Module Level 3)
  → parser (constructs CSSOM)
  → selector matching (rules per element)
  → declared values (per property, per element, list of all matching declarations)
  → cascaded values (per property, per element, single winner from cascade)
  → specified values (cascaded value OR default if cascade produced nothing)
  → computed values (relative units → absolute, inherited where appropriate)
  → used values (after layout — percentages → pixels, auto → resolved)
  → actual values (used + device quantization, e.g. sub-pixel rounding)
```

Each transition is a documented algorithm. JavaScript's `getComputedStyle()` returns values from the **computed** stage — not the used stage — which is why `width: 50%` may come back as `"50%"` in some legacy browsers and as the resolved pixel value in modern ones (the spec was tightened in CSS 2.1 errata to return the used value for layout-sensitive properties).

### 1.2 Why the Stages Matter

```css
:root {
  --base: 16px;
}
.text {
  font-size: calc(var(--base) * 1.5); /* declared */
  width: 50%;                          /* declared */
  color: inherit;                      /* declared */
}
```

| Property      | Declared              | Cascaded              | Specified             | Computed                   | Used                  | Actual                |
|---------------|-----------------------|-----------------------|-----------------------|----------------------------|-----------------------|-----------------------|
| `font-size`   | `calc(var(--base)*1.5)` | (winner)            | same                  | `24px` (calc resolved)     | `24px`                | `24px`                |
| `width`       | `50%`                 | `50%`                 | `50%`                 | `50%`                      | `400px` (after layout)| `400px`               |
| `color`       | `inherit`             | `inherit`             | `inherit`             | parent's computed value    | same                  | same (gamut-mapped)   |

The width stays as `"50%"` through computed; only **after layout** does it become a pixel quantity. This is why `getComputedStyle(el).width` post-layout returns pixels — the spec resolves to used at that point.

### 1.3 Where Spec Numbering Lives

Spec sections you should be able to cite from memory:
- **css-cascade-5 §4** — Cascading.
- **css-cascade-5 §4.5** — Defaulting (initial / inherit / unset / revert).
- **css-cascade-5 §6** — Cascade layers.
- **css-cascade-6 §3** — `@scope`.
- **css-values-4 §4** — Computed value resolution.
- **css-values-4 §4.4** — Used value.
- **CSSOM §6** — Resolved value.

---

## 2. The Cascade Algorithm — Origin and Importance

The cascade is a **tournament** that takes all declarations matching one element, for one property, and picks one. The algorithm is exhaustively specified (`css-cascade-5 §6.1`). Each declaration carries a tuple of attributes; declarations are sorted by these attributes lexicographically.

### 2.1 The Origin Hierarchy

There are three origins:
1. **User-Agent** — the browser's built-in stylesheet (e.g. `body { display: block }`, `h1 { font-size: 2em; margin-block-start: 0.67em }`).
2. **User** — user-installed stylesheets (rare in modern browsers; mostly accessibility add-ons).
3. **Author** — the page's own stylesheets (CSS files, `<style>`, inline `style=""`).

Without `!important`, the precedence is **author > user > user-agent**.

### 2.2 The !important Reversal

`!important` flips the origin order (`css-cascade-5 §6.4.1`):
- Without important: UA < User < Author
- With important: Author!important < User!important < User-Agent!important

So `!important` in a user-agent stylesheet wins over `!important` in author code (this is how browsers force certain accessibility rules — e.g. `option { display: list-item !important }` in the UA sheet for `<select>` rendering).

### 2.3 The Canonical Priority Ladder

For one property on one element, the cascade resolves declarations in this exact order (`css-cascade-5 §6.5`):

1. Transition declarations (highest)
2. UA `!important`
3. User `!important`
4. Author `!important` (last layer, then earlier layers, then unlayered)
5. Animation declarations (`@keyframes`)
6. Author normal (unlayered first, then last layer back to first layer)
7. User normal
8. UA normal (lowest)

Note the inversion: **layers** order in opposite directions for normal vs. important declarations. This is intentional — see §5.

### 2.4 Practical Implication

`!important` exists for one legitimate purpose: overriding inline styles or third-party widgets. If you find yourself adding `!important` to your own code to "win," your problem is almost always **origin confusion** or **selector specificity** misuse. The modern fix: cascade layers (§5).

```css
/* Old "I just need to win" pattern */
.button {
  background: blue !important; /* code smell */
}

/* Modern: explicit layer ordering */
@layer base, theme, overrides;
@layer theme {
  .button { background: blue; }
}
```

---

## 3. Specificity — The Formal Calculation

Specificity is a 4-tuple — actually a 3-tuple in practice, since the inline-style component is handled separately as origin/importance — defined in `selectors-4 §17`.

### 3.1 The Tuple

The classic mnemonic is `(a, b, c, d)`:
- **a** — 1 if the declaration came from a `style=""` attribute, else 0 (often handled as a separate origin step in modern engines).
- **b** — count of ID selectors (`#main`).
- **c** — count of class selectors, attribute selectors, and pseudo-classes (`.btn`, `[type="text"]`, `:hover`).
- **d** — count of element selectors and pseudo-elements (`div`, `::before`).

Modern texts collapse to `(b, c, d)` and treat inline style + `!important` as separate cascade steps.

### 3.2 What Counts and What Doesn't

| Token                         | Contributes to | Notes                                              |
|-------------------------------|----------------|----------------------------------------------------|
| `*` (universal)               | nothing        | Specificity (0,0,0).                               |
| `>`, `+`, `~`, descendant    | nothing        | Combinators don't count.                           |
| `#id`                         | `b`            | One per ID, even if used in `:is()` arguments.     |
| `.class`                      | `c`            |                                                    |
| `[attr]`, `[attr="x"]`       | `c`            |                                                    |
| `:hover`, `:focus`, `:nth-child(n)` | `c`     | All pseudo-classes.                                |
| `tag`                         | `d`            |                                                    |
| `::before`, `::after`        | `d`            | Pseudo-elements count as `d`, not as `c`.         |
| `:is(X, Y)`                   | max(X, Y)      | The argument with the highest specificity.        |
| `:where(X, Y)`                | (0,0,0)        | **Always zero.** Hugely useful — see §4.          |
| `:not(X, Y)`                  | max(X, Y)      |                                                    |
| `:has(X, Y)`                  | max(X, Y)      |                                                    |
| `:nth-child(n of S)`          | (0,0,1) + S's specificity | The selector list inside contributes too.     |

### 3.3 Worked Examples

```css
/* (0, 1, 0) */
.menu { }

/* (0, 1, 1) */
ul.menu { }

/* (1, 0, 0) */
#main { }

/* (1, 2, 0) */
#main .menu.active { }

/* (0, 0, 0) — :where forces zero */
:where(#main) .menu { }
/* Wait — this is (0, 1, 0), because :where(#main) contributes 0,
   and .menu contributes (0,1,0). */

/* (1, 0, 0) — :is takes max */
:is(#main, .sidebar) { }

/* (0, 1, 1) — :not's argument max */
button:not(.disabled, [aria-disabled="true"]) { }
/* button = (0,0,1); :not max = .disabled (0,1,0); total = (0,1,1) */

/* (0, 2, 1) — :nth-child(n of S) is unique */
li:nth-child(2 of .selected) { }
/* li = (0,0,1); :nth-child = (0,1,0); .selected inside = (0,1,0); total = (0,2,1) */
```

### 3.4 The Comparison

Specificities compare lexicographically left-to-right:
```
(0, 1, 0) > (0, 0, 99)      // a single class beats 99 elements
(1, 0, 0) > (0, 256, 256)   // an ID beats anything without an ID
```

This is why the old "256-class rule" was a myth — specificity doesn't carry, it lexicographically ranks.

### 3.5 Why Specificity Wars Are a Smell

Specificity wars indicate one of:
1. You're using IDs in selectors when you mean to apply a "single instance" style. Prefer classes.
2. Your global stylesheet is too aggressive. Prefer narrower selectors.
3. You need cascade layers (§5). Layer winning is **above** specificity in the cascade ladder.

---

## 4. The :where() Specificity-Hack

`:where()` is the pseudo-class that **forces zero specificity** on its argument list. Defined in `selectors-4 §17.2`.

### 4.1 The Idiom

Library code that wants to ship default styles which **author code can override with the simplest selector** uses `:where()`:

```css
/* Library reset — every author's `h1 { color: red }` will win */
:where(h1, h2, h3, h4, h5, h6) {
  margin-block: 0;
  font-weight: 600;
}

/* Author code — wins because :where forces (0,0,0) on the library */
h1 { margin-block: 1em; font-weight: 700; }
```

Without `:where()`, the library would have specificity `(0,0,1)` per heading, and the author would also need `(0,0,1)` — fine for `h1`, but ugly for compound selectors.

### 4.2 Contrast with :is()

```css
:where(#main .article p) { color: gray; }   /* specificity = (0,0,0) */
:is   (#main .article p) { color: gray; }   /* specificity = (1,1,1) */
```

Use `:is()` to deduplicate selector lists **without** changing specificity. Use `:where()` to deduplicate **and** zero out specificity.

### 4.3 Real-World Pattern: Modern Resets

Modern resets (`open-props.style/normalize`, `andy-bell/modern-css-reset`) use `:where()` extensively so that every selector in the reset has specificity `(0,0,0)`, and any author code wins:

```css
:where(*, *::before, *::after) {
  box-sizing: border-box;
}

:where(button, [type="button"], [type="submit"]) {
  cursor: pointer;
}

:where(img, picture, video) {
  max-inline-size: 100%;
  block-size: auto;
}
```

### 4.4 Anti-Patterns

```css
/* Don't do this — :where here erases your own intent */
:where(.danger) { color: red; }
.danger.large { color: red; font-size: 1.5em; }
/* Now `.danger` has specificity 0; your modifier needs to know that */
```

---

## 5. @layer — Cascade Layers

Cascade layers (`css-cascade-5 §6.4.5`) introduce a tournament step **above specificity**: declarations in a higher-priority layer beat declarations in a lower-priority layer regardless of the latter's specificity.

### 5.1 Declaration Order Defines Priority

```css
@layer reset, base, components, utilities;
```

This statement says "there are four layers, in order: reset (lowest priority), base, components, utilities (highest priority)." This is the **canonical** modern utility-CSS architecture. Tailwind v4, Open Props, every modern design system uses this pattern.

### 5.2 Adding Rules to Layers

```css
@layer reset {
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
}

@layer base {
  body { font-family: system-ui; line-height: 1.5; }
  h1   { font-size: 2rem; font-weight: 700; }
}

@layer components {
  .card {
    background: var(--surface);
    border-radius: 0.5rem;
    padding: 1rem;
  }
}

@layer utilities {
  .text-center { text-align: center; }
  .mt-4        { margin-top: 1rem; }
}
```

A `.text-center` declaration in `utilities` wins against a more-specific `.card .heading` declaration in `components`, even though `(0,2,0) > (0,1,0)` in raw specificity.

### 5.3 Layer-Internal Specificity Still Matters

Within a single layer, specificity decides. So:

```css
@layer components {
  .card { background: white; }
  .card.active { background: yellow; }   /* wins inside the layer */
}
```

### 5.4 The !important Reversal Inside Layers

Important declarations **flip the layer order** (`css-cascade-5 §6.4.5`):
- Normal: utilities > components > base > reset
- Important: reset!important > base!important > components!important > utilities!important

So `!important` in your `reset` layer beats `!important` in `utilities`. This is intentional: it lets a lower-priority layer say "this rule is critical" and not be overridden by a higher-priority normal-`!important`. In practice, **don't use `!important` inside layers** — the layer system already gives you priority.

### 5.5 Anonymous Layers and Nested Layers

```css
/* Anonymous layer — useful for one-shot overrides */
@layer { .legacy { color: red; } }

/* Nested layers — components.cards beats components.buttons */
@layer components {
  @layer buttons, cards;
  @layer buttons { .btn { ... } }
  @layer cards   { .card { ... } }
}
```

### 5.6 @import with Layers

`@import` integrates with layers via `layer()` syntax:

```css
@import url("framework.css") layer(framework);
@import url("theme.css") layer(theme);

@layer reset, framework, theme, components;
```

Now everything in `framework.css` lives in the `framework` layer, regardless of its own structure. This is how you wrap third-party CSS into your cascade architecture.

### 5.7 The Canonical Layer Stack

```css
@layer reset,        /* 1. Modern reset (Andy Bell-style) */
       base,         /* 2. Base elements (body, h1-h6, p, a) */
       tokens,       /* 3. Design tokens (custom props on :root) */
       layout,       /* 4. Layout primitives (.stack, .cluster, .grid) */
       components,   /* 5. Components (.card, .button, .nav) */
       utilities,    /* 6. Utility classes (.text-center, .mt-4) */
       overrides;    /* 7. Page-specific or one-off overrides */
```

This stack lets every layer do exactly its job, and "wins" gracefully without specificity wars.

---

## 6. @scope — The Scoping Mechanism

`@scope` (`css-cascade-6 §3`) lets you scope a stylesheet to a subtree of the DOM, with optional **lower bounds** (the scope ends at certain descendants). It also introduces a new tiebreaker — **scoping proximity**.

### 6.1 Basic Syntax

```css
@scope (.card) {
  /* Selectors here only match descendants of .card */
  :scope { border: 1px solid; }
  .title { font-size: 1.25rem; }
  .body  { line-height: 1.6; }
}
```

`:scope` refers to the scope root itself (`.card`). All other selectors implicitly require an ancestor match against `.card`.

### 6.2 Scoping with a Lower Bound

```css
@scope (.card) to (.card .nested) {
  .button { background: blue; }
}
```

This applies `.button { background: blue }` only to `.button` descendants of `.card` that are **not** inside a `.card .nested` subtree. The "to" selector ends scope traversal.

### 6.3 The Component-Without-Shadow-DOM Pattern

```html
<article class="card">
  <h2 class="title">Hello</h2>
  <div class="body">
    <p>Outer paragraph.</p>
    <article class="card">                <!-- nested card -->
      <h2 class="title">Nested title</h2>
    </article>
  </div>
</article>
```

```css
@scope (.card) to (.card .card) {
  .title { color: red; }
}
```

Both `.title` elements still match `.card .title`, but the scope ends at `.card .card`, so the nested title is **not** styled. Without `@scope`, you'd need awkward `:not()` selectors or hand-built class systems. This is the "component-scoped without Shadow DOM" idiom.

### 6.4 Scoping Proximity — A New Tiebreaker

Where two declarations would otherwise tie on cascade order and specificity, the **scope-proximity tiebreaker** kicks in (`css-cascade-6 §3.7`): the rule whose scope root is **closer** to the matching element wins.

```css
@scope (.theme-light) { :scope a { color: navy; } }
@scope (.theme-dark)  { :scope a { color: cyan; } }
```

If a link is inside both (nested theming), the **inner** scope wins because its root is closer to the link.

---

## 7. Inherited vs Non-Inherited Properties

Inheritance is property-by-property. Each CSS property is defined as either inherited or not (`css-cascade-5 §3.7`).

### 7.1 The Canonical Inherited List

| Category    | Inherited Properties                                                                          |
|-------------|-----------------------------------------------------------------------------------------------|
| Color       | `color`, `caret-color`                                                                        |
| Font        | `font-family`, `font-size`, `font-style`, `font-weight`, `font-variant`, `font-stretch`, `font-feature-settings`, `font-kerning` |
| Text        | `text-align`, `text-indent`, `text-transform`, `text-shadow` (inherits), `letter-spacing`, `word-spacing`, `line-height`, `white-space`, `word-break`, `overflow-wrap` |
| Lists       | `list-style`, `list-style-type`, `list-style-position`, `list-style-image`                    |
| Direction   | `direction`, `writing-mode` (inherits), `unicode-bidi` (does NOT)                             |
| Visibility  | `visibility`, `cursor`                                                                        |
| Tables      | `border-collapse`, `border-spacing`, `caption-side`, `empty-cells`                            |
| Quotes      | `quotes`                                                                                      |

### 7.2 Non-Inherited (the Default)

Most box-model and layout properties **do not** inherit:
- `width`, `height`, `padding`, `margin`, `border`
- `display`, `position`, `top`/`right`/`bottom`/`left`
- `background-*` (yes — `background-color: red` on `<body>` does not propagate to children's computed `background-color`; the visual effect is because the body's painted background bleeds through children's transparent backgrounds)
- `overflow`, `z-index`, `transform`, `filter`, `opacity`, `box-shadow`

### 7.3 The Defaulting Keywords

Defined in `css-cascade-5 §4.5`:

| Keyword          | Effect                                                                                  |
|------------------|------------------------------------------------------------------------------------------|
| `inherit`        | Use the parent's computed value.                                                         |
| `initial`        | Use the property's spec-defined initial value (e.g. `color: initial` → black-ish).      |
| `unset`          | `inherit` if the property is inherited, else `initial`.                                 |
| `revert`         | Roll back to the value from the **previous origin** (author → user → UA). Useful for "undo my framework's declaration and use the UA's style." |
| `revert-layer`   | Roll back to the value from the **previous layer** in the cascade. Powerful inside `@layer`. |

### 7.4 The `all` Property

`all` is a shorthand for **every** property except `direction`, `unicode-bidi`, and custom properties. It accepts only the defaulting keywords:

```css
.reset-everything {
  all: revert;       /* drop everything, fall back to UA defaults */
}

.inherit-everything {
  all: inherit;      /* inherit every property from parent */
}
```

The `all: revert` reset is the modern "shadow-DOM-without-shadow-DOM" trick — it strips your component free of inherited styles in one declaration.

### 7.5 Inheritance + Inline Property Values

```css
:root { --brand: oklch(60% 0.2 250); }
.btn  { color: var(--brand); }
```

Custom properties (`--brand`) inherit by default. So `--brand` set on `:root` is available on every descendant. If you want a non-inherited custom property, you must register it via `@property` (§11).

---

## 8. Computed Values — Resolution

The cascade picks a winner. That winner is the **specified value**. The browser then resolves the specified value into a **computed value** (`css-values-4 §4`).

### 8.1 What "Compute" Does

The computed-value step performs three transformations:

1. **Defaulting keyword resolution** — `inherit` becomes the parent's computed value; `initial` becomes the property's initial value; `unset`/`revert`/`revert-layer` are resolved to the appropriate fallback.
2. **Relative-unit absolutization** — `em`, `rem`, `ex`, `ch`, `vw`, `vh`, percentages on certain properties, etc., are resolved against the appropriate reference (font-size, viewport, parent's font-size).
3. **Calc folding** — `calc(2 * 10px + 4em)` is folded into a single absolute number (where possible — calc with mixed units may stay symbolic until used-value time).

### 8.2 Worked Example

```css
:root         { font-size: 16px; }
.parent       { font-size: 1.25em; }   /* computed: 20px */
.child        { font-size: 1.5em;  }   /* computed: 30px (1.5 × 20) */
.grandchild   { font-size: inherit; }  /* computed: 30px */
```

Each `em` resolves against the **parent's** computed font-size at that level. This is why nesting `em`-based font-sizes compounds.

### 8.3 The getComputedStyle Contract

```javascript
const el = document.querySelector('.box');
const cs = getComputedStyle(el);
console.log(cs.fontSize);   // "16px"   — computed value
console.log(cs.color);      // "rgb(0, 0, 0)" — computed
console.log(cs.width);      // depends — see §9
```

`getComputedStyle` returns the **resolved value** (CSSOM §6.5), which is mostly the computed value, but for layout-sensitive properties (width, height, top/right/bottom/left, margin, padding) the spec demands the **used value**. So:

```javascript
// box has CSS: width: 50%; parent is 800px wide.
console.log(getComputedStyle(box).width);  // "400px" — used, not "50%"
```

### 8.4 @media and @supports — Where They Sit

Media queries (`@media`) and feature queries (`@supports`) are **rule-applicability filters**. They affect whether a rule's declarations enter the cascade in the first place; they do not affect the computed-value resolution. Once a rule is "in," its declarations cascade normally.

```css
@media (min-width: 768px) {
  .grid { grid-template-columns: repeat(3, 1fr); }
}

@supports (display: grid) {
  .layout { display: grid; }
}
```

If the media query is false, the rule is not collected at selector-matching time. This is why CSSOM's `cssRules` includes `CSSMediaRule` wrappers that you must traverse into to see the inner rules.

### 8.5 Inheritance Algorithm (Pseudocode)

```python
def computed_value(element, property):
    declared = matched_declarations(element, property)
    cascaded = cascade_winner(declared)             # from §2
    specified = cascaded if cascaded else default_for(property)

    if specified == "inherit":
        if element.parent is None:
            return initial_value_for(property)
        return computed_value(element.parent, property)

    if specified == "initial":
        return initial_value_for(property)

    if specified == "unset":
        return computed_value(element.parent, property) \
            if is_inherited(property) \
            else initial_value_for(property)

    return resolve_units_and_calc(specified, element)
```

This is the spec algorithm distilled. Real engines optimize heavily (style sharing, computed-style caching), but the semantic result is exactly this.

---

## 9. Used Values vs Actual Values

Two more stages sit after computed: **used** and **actual**.

### 9.1 Used Value

The used value is the value after **layout has run**. `width: 50%` becomes a pixel quantity once the parent's width is known. `height: auto` becomes a specific pixel height once the children's heights are known.

```css
.col {
  width: 50%;     /* computed: 50%; used: depends on parent layout */
  height: auto;   /* computed: auto; used: pixels after layout */
  margin: 1em;    /* computed: 16px (or whatever em resolves to); used: 16px */
}
```

The used-value step **requires layout**. This is why `width: 50%` does not have a used value before layout — it is a percentage of an unknown.

### 9.2 Actual Value

The actual value is the used value **after the device has quantized it**. Browsers paint to a pixel grid; sub-pixel widths get rounded. On HiDPI displays, the rounding is finer (often to half-pixel), but it still happens.

```css
.line { border-top-width: 0.6px; }
/* Used value: 0.6px
   Actual value: 1px (or 0px) on a 1× display, possibly 0.5px on a 2× display */
```

This matters for hairlines, for sub-pixel positioning of animated elements, and for tests that compare `getBoundingClientRect()` to expected pixel values.

### 9.3 The "Why JS Sees Used, Not Actual" Question

`getBoundingClientRect()` returns floating-point numbers because it returns **used** values (and sometimes used + transform), not actual. The actual value is a screen-quantized integer (or sub-pixel-quantized fraction). You can't read actual values from JavaScript; they live below the API.

### 9.4 Worked Pipeline Trace

```css
:root  { font-size: 16px; }
.card  { width: 50%; padding: 1em; border: 1px solid; }
```

For a `.card` whose parent is 803px wide on a 2× device:

| Stage      | width                  | padding             |
|------------|------------------------|---------------------|
| Declared   | `50%`                  | `1em`               |
| Cascaded   | `50%`                  | `1em`               |
| Specified  | `50%`                  | `1em`               |
| Computed   | `50%` (still symbolic) | `16px`              |
| Used       | `401.5px`              | `16px`              |
| Actual     | `401.5px` (2× display) | `16px`              |

On a 1× display, the actual width might be 401px or 402px depending on the painter's rounding.

---

## 10. CSS Custom Properties — Internals

Custom properties (`--name: value`) are defined in CSS Custom Properties for Cascading Variables Module Level 1 (`css-variables-1`).

### 10.1 The Substitution Model

A custom property's value is a **token stream** until the moment it is consumed via `var()`. The browser does **not** parse the token stream as a property value at definition time; it just stores it as text-like tokens.

```css
:root {
  --x: 4px solid red;     /* parsed as a list of tokens, not as a "border" value */
}

.box {
  border: var(--x);       /* substituted, then parsed as a border declaration */
  margin: var(--x);       /* substituted; "4px solid red" is not a valid margin */
                          /* Result: invalid, falls back to initial value */
}
```

This is the most counter-intuitive thing about custom properties: validation happens at **substitution** site, not definition site. You can store nonsense in a `--var` and the browser doesn't care until you use it.

### 10.2 var() and Fallbacks

```css
.btn {
  /* If --brand is unset, use blue */
  background: var(--brand, blue);

  /* Multi-fallback: try --user-brand, then --brand, then blue */
  color: var(--user-brand, var(--brand, blue));
}
```

The fallback can be **any** value, including another `var()`. This is how you build typed-fallback chains without `@property`.

### 10.3 Inheritance

Custom properties **inherit** by default. This is actually special: it makes them ideal for theming.

```css
:root          { --brand: oklch(60% 0.2 250); }
.theme-warm    { --brand: oklch(70% 0.18 50); }

.btn { background: var(--brand); }
```

Any `.btn` inside `.theme-warm` gets warm orange; any `.btn` outside gets the default blue. This is the canonical CSS theming pattern post-2020.

### 10.4 The Parsed-vs-Unparsed Distinction

| Custom Property Type          | Parsed at Definition? | Animatable? | Validated? |
|-------------------------------|-----------------------|-------------|------------|
| Unregistered (`--x: 5px`)     | No (token stream)     | No (interpolated as text) | At substitution |
| Registered (`@property --x`)  | Yes (typed)           | Yes (real interpolation)  | At definition |

This is the bridge into `@property`.

### 10.5 The Component API Pattern

```css
.button {
  /* Declare the "API" of the component */
  background: var(--button-bg, var(--brand, navy));
  padding: var(--button-padding, 0.5rem 1rem);
  border-radius: var(--button-radius, 0.25rem);
}

/* Caller customizes: */
.checkout .button {
  --button-bg: oklch(60% 0.2 140);   /* green */
  --button-padding: 1rem 2rem;
}
```

Custom properties become the public API of a component. The component documents its `--button-*` knobs; consumers set them per-instance.

---

## 11. @property — Typed Custom Properties

`@property` (CSS Properties and Values API Level 1, `css-properties-values-api-1`) registers a custom property with a **type**, an **inheritance flag**, and an **initial value**.

### 11.1 The Definition Block

```css
@property --brand-hue {
  syntax: "<number>";
  inherits: true;
  initial-value: 250;
}

@property --brand {
  syntax: "<color>";
  inherits: true;
  initial-value: oklch(60% 0.2 250);
}
```

| Field          | Meaning                                                                                             |
|----------------|-----------------------------------------------------------------------------------------------------|
| `syntax`       | A type-grammar like `<number>`, `<color>`, `<length>`, `<percentage>`, `<length-percentage>`, `<image>`, `<url>`, `<integer>`, `<angle>`, `<time>`, `<resolution>`, `<transform-function>`, `<custom-ident>`, or a `+` /`#` for lists, or `*` for any. |
| `inherits`     | `true` (default for unregistered) or `false`. Non-inheriting custom properties are useful for "scoped overrides that don't leak." |
| `initial-value`| Mandatory for typed properties (except `*`). The fallback when no value is set or when invalid.    |

### 11.2 Why Register?

Two huge wins:

#### 11.2.1 Animation Interpolation

Without `@property`, a custom property animates as a **swap at 50%**. With `@property`, it interpolates smoothly:

```css
@property --t {
  syntax: "<percentage>";
  inherits: false;
  initial-value: 0%;
}

.gradient {
  background: linear-gradient(to right, red, blue var(--t), white);
  transition: --t 1s;
}
.gradient:hover { --t: 100%; }
```

The hover transitions the gradient stop smoothly from 0% to 100%. Without `@property`, it would jump.

#### 11.2.2 Type Validation

```css
@property --width {
  syntax: "<length>";
  inherits: false;
  initial-value: 0px;
}

.box {
  --width: 200px;  /* valid */
  --width: red;    /* invalid — falls back to initial-value 0px */
}
```

The browser validates at definition site. Invalid assignments silently fall back.

### 11.3 The Animatable-Gradient-Stop Pattern

```css
@property --angle {
  syntax: "<angle>";
  inherits: false;
  initial-value: 0deg;
}

@keyframes spin {
  to { --angle: 360deg; }
}

.spinner {
  background: conic-gradient(from var(--angle), red, blue, red);
  animation: spin 4s linear infinite;
}
```

Without `@property --angle { syntax: "<angle>" }`, the `--angle` would interpolate as text and the conic gradient would freeze.

### 11.4 JavaScript Registration

Equivalent JS API:

```javascript
CSS.registerProperty({
  name: '--brand',
  syntax: '<color>',
  inherits: true,
  initialValue: 'oklch(60% 0.2 250)',
});
```

Use this when you want to register dynamically — e.g. a theme system that registers tokens at runtime.

---

## 12. Color Spaces and color() Function

CSS Color Module Level 4 (`css-color-4`) and Level 5 (`css-color-5`) replaced the sRGB-only world.

### 12.1 The Color Models

| Notation                          | Color Space  | Use Case                                                       |
|-----------------------------------|--------------|----------------------------------------------------------------|
| `#fff`, `rgb()`, `hsl()`         | sRGB         | Legacy default. Most current screens.                          |
| `color(srgb 1 0 0)`              | sRGB explicit| Same as above, just explicit.                                  |
| `color(display-p3 1 0 0)`        | Display P3   | Wide-gamut screens (modern Macs, iOS, modern Android).        |
| `color(rec2020 1 0 0)`           | Rec. 2020    | Even wider; not yet common on consumer devices.               |
| `lab(50% 40 30)`                  | CIE Lab      | Perceptually uniform (older, less behaved than oklab).         |
| `lch(50% 50 30)`                  | CIE LCh      | Cylindrical Lab.                                               |
| `oklab(0.5 0.1 0.1)`             | OKLab        | **Perceptually uniform**. The modern preferred space.          |
| `oklch(60% 0.2 250)`             | OKLCh        | Cylindrical OKLab. Hue is a single number — easiest theming.  |

OKLab/OKLCh (Björn Ottosson, 2020) are the perceptually-uniform color spaces that fix the well-known problems of CIE Lab on saturated colors. They are the default recommendation for design systems in 2024–2026.

### 12.2 color-mix()

`color-mix()` (`css-color-5 §4`) blends two colors in a chosen color space:

```css
.btn {
  /* 50/50 blend of red and blue in OKLCh */
  background: color-mix(in oklch, red, blue);

  /* 30% white, 70% brand — a tint */
  color: color-mix(in oklab, white 30%, var(--brand));

  /* Hover state derived from base */
}
.btn:hover {
  background: color-mix(in oklch, var(--brand), black 10%);
}
```

This eliminates entire color-system boilerplates: hover/active/disabled states are math, not hand-picked palettes.

### 12.3 light-dark() — Theme-Aware Colors

`light-dark()` (`css-color-5 §5`) returns the first argument in light mode, the second in dark mode. Activated by `color-scheme: light dark` on the element.

```css
:root {
  color-scheme: light dark;
  --bg: light-dark(white, oklch(20% 0.02 250));
  --fg: light-dark(black, oklch(95% 0.01 250));
}

body { background: var(--bg); color: var(--fg); }
```

No more `@media (prefers-color-scheme: dark)` blocks. The function reads the current scheme automatically.

### 12.4 The Gamut-Mapping Algorithm

When a color is **out-of-gamut** for the display, the browser must map it. Four strategies (`css-color-4 §13`):

1. **Clipping** — naive; clamps each channel. Visible color shifts.
2. **Chroma reduction** — preserves hue and lightness, reduces saturation until in-gamut. The CSS spec's default for OKLCh.
3. **Perceptual mapping** — uses a delta-E metric.
4. **Relative colorimetric** — used for printing.

For OKLCh, the spec's "css-color-4 algorithm" is chroma reduction in OKLCh space. This is why specifying colors in OKLCh produces the most predictable cross-device results — the browser's mapping matches the color space.

### 12.5 Color Interpolation

The `in <color-space>` modifier on gradients and `color-mix()` controls **how interpolation happens**:

```css
.bad  { background: linear-gradient(to right, red, blue); }                    /* sRGB */
.good { background: linear-gradient(in oklch, to right, red, blue); }          /* OKLCh */
```

In sRGB, the midpoint of red→blue is muddy purple. In OKLCh, it is a clean perceptual midpoint. Same end colors, very different middles.

---

## 13. Selector Matching Algorithm

Browsers match selectors **right-to-left** (`selectors-4 §19`). This is one of the most-cited but least-understood facts about CSS performance.

### 13.1 Why Right-to-Left

A selector like `.menu li a` could be matched by:
- **Left-to-right**: For each `.menu`, descend, find every `li`, descend, find every `a`. Cost: proportional to tree size.
- **Right-to-left**: For the **element being styled** (an `a`), check if it's an `a`. If so, walk up: is any ancestor an `li`? Yes — keep walking. Is any ancestor of that `li` a `.menu`? Yes — match.

The browser styles **every element in the document**. For each element, it has to ask "does this rule match?" Right-to-left lets the browser answer "no" quickly: the rightmost selector is the first filter.

### 13.2 The Cost Model

For each rule, for each element:
1. Match the rightmost simple selector. If no, **bail**.
2. If yes, walk the chain: each combinator step is one DOM walk.

Cost ≈ `O(rules × matching-elements × ancestor-depth)`.

### 13.3 The "ID Selectors Are Slow" Debate

A naive reading suggests `#main` is slow because the browser still does right-to-left matching. In practice, modern engines (Chromium's Blink, Firefox's Servo-derived, WebKit) maintain **reverse indexes** by ID, class, and tag — so the rightmost-selector check is `O(1)` lookups, not full walks. ID selectors are not slower than class selectors at match time.

### 13.4 What Actually Costs

The **expensive** patterns:
- **Universal as the rightmost**: `* { ... }` matches every element; the rightmost filter does nothing.
- **Descendant chains with very common right-most**: `.foo *` — every element checks whether any ancestor is `.foo`.
- **`:nth-child` recalculation**: changing the DOM near a `:nth-child` rule forces recalc on all siblings.
- **Attribute selectors with substring matching**: `[href*="example"]` requires string scans.

Best practice (still): **specific rightmost selectors**, **shallow chains**.

### 13.5 The Modern Hash Optimization

Engines index rules by:
- Last simple selector (a hash key).
- Up to ~3 ancestor classes/IDs (Bloom-filter pre-qualification — see §14).

A rule like `.menu .item .link` is keyed on `.link` plus a Bloom filter saying "the matching element's ancestor chain must contain `.menu` and `.item`."

---

## 14. Bloom Filters in Selector Matching

Servo (Mozilla's research engine, now living in Firefox style code) and Blink both use **per-element Bloom filters** to short-circuit ancestor-chain matching.

### 14.1 The Idea

Each element holds a Bloom filter representing the set of classes, IDs, and tag names of its **ancestors**. When the engine evaluates a rule like `.foo .bar .target`, it asks: "is `.foo` and `.bar` in my ancestor Bloom filter?"

- If the filter says **no** for `.foo` (with the false-positive rate of Bloom filters considered acceptable), bail. No DOM walk.
- If the filter says **yes**, do the actual walk to verify.

### 14.2 Why It's Cheap

Bloom filters are constant-size bitfields with a small number of hash functions. Adding a class is `O(k)` for `k` hashes (typically 2–4). Querying is `O(k)`. For a typical 100-deep DOM with thousands of classes in scope, the filter saves enormous amounts of DOM traversal.

### 14.3 The Cost-Model Implication

Selectors with **rare** ancestor classes are essentially free at non-match. Selectors with **very common** ancestor classes (e.g. `body .content` everywhere) lose the Bloom filter benefit because the filter always says "yes."

So the modern advice is: **prefer specific class names in your selector chains**, not deep selectors anchored on common ancestors.

### 14.4 Pseudo-Code

```python
class Element:
    ancestor_bloom: BloomFilter

    def matches_descendant_chain(self, ancestor_selectors):
        # Quick reject via Bloom
        for sel in ancestor_selectors:
            if sel.is_class_or_id() and not self.ancestor_bloom.maybe_contains(sel.name):
                return False        # definitely no match
        # Otherwise walk for real
        return self.walk_ancestors_for_chain(ancestor_selectors)
```

---

## 15. The Visual Formatting Model — Pre-Layout

Before layout runs, the browser establishes **formatting contexts** (`css-display-3 §3`, `css2 §9`).

### 15.1 The Formatting Contexts

| Context        | Triggered By                                     | Layout Behavior                              |
|----------------|--------------------------------------------------|----------------------------------------------|
| **Block FC**   | Block elements; `display: flow-root`             | Children stack vertically, margins collapse  |
| **Inline FC**  | Inline elements                                  | Children flow horizontally, line boxes form  |
| **Flex FC**    | `display: flex` / `inline-flex`                  | Flex algorithm applies                       |
| **Grid FC**    | `display: grid` / `inline-grid`                  | Grid track sizing applies                    |
| **Table FC**   | `display: table` / `<table>` element             | Table layout algorithm                       |
| **Multicol**   | `column-count` / `column-width`                  | Column flow                                  |

(Layout algorithms inside these contexts are documented in `detail/web/css-layout.md`.)

### 15.2 Block Formatting Context — Establishment Triggers

Anything from this list creates a **new** BFC:
- `float: left | right`
- `position: absolute | fixed`
- `display: inline-block`
- `display: flow-root` ← **the modern way**
- `overflow: auto | hidden | scroll | clip` (anything ≠ `visible`)
- `display: flex | grid` (these create their own FC, not BFC, but isolate similarly)
- Multicol containers
- Contain contexts: `contain: layout`, `contain: paint`, `contain: strict`

### 15.3 Why BFCs Matter

A new BFC:
- **Isolates float clearing**: floats inside the BFC don't escape; floats outside don't intrude.
- **Prevents margin collapse**: margins between siblings in different BFCs don't collapse.
- **Becomes a "block" for layout purposes**: its descendants don't affect outer layout.

The historical hack was `overflow: hidden` on a parent to clear floats. The modern equivalent is `display: flow-root` — same effect, no clipping side effect.

```css
/* Old hack — also clips overflow */
.clearfix { overflow: hidden; }

/* Modern — establishes BFC without clipping */
.clearfix { display: flow-root; }
```

### 15.4 Inline Formatting Context

Inline content lays out into **line boxes**. Each line box is the height of the tallest inline content. The `vertical-align` property positions inline children within the line box.

A common surprise: `<img>` inside a paragraph creates a line box taller than the surrounding text because the image baseline is the bottom of the image. Setting `vertical-align: middle` or `display: block` on the image fixes the gap.

---

## 16. The CSSOM

The CSS Object Model (`cssom-1`) exposes parsed stylesheets to JavaScript.

### 16.1 The Object Tree

```
document.styleSheets (StyleSheetList)
  → CSSStyleSheet
      .ownerNode      (the <link> or <style>)
      .cssRules       (CSSRuleList)
          → CSSStyleRule   (selector + declarations)
          → CSSMediaRule   (@media)
          → CSSLayerBlockRule  (@layer block form)
          → CSSLayerStatementRule (@layer name list)
          → CSSPropertyRule  (@property)
          → CSSScopeRule    (@scope)
          → CSSImportRule   (@import)
          → CSSGroupingRule (any rule that contains other rules — base class)
      .insertRule(rule, index)
      .deleteRule(index)
      .replaceSync(text)        // constructible stylesheets only
```

### 16.2 Reading Rules

```javascript
const sheet = document.styleSheets[0];
for (const rule of sheet.cssRules) {
  if (rule instanceof CSSStyleRule) {
    console.log(rule.selectorText, '→', rule.style.cssText);
  } else if (rule instanceof CSSMediaRule) {
    console.log('@media', rule.conditionText);
    for (const inner of rule.cssRules) {
      console.log('  ', inner.selectorText);
    }
  }
}
```

`CSSStyleRule.style` is a `CSSStyleDeclaration` — same interface as `element.style`. You can read declarations via property names (`rule.style.color`) or as a numeric list (`rule.style[0]`).

### 16.3 Writing Rules at Runtime

```javascript
sheet.insertRule('.theme-warm { --brand: oklch(70% 0.18 50); }', sheet.cssRules.length);
sheet.deleteRule(0);
```

`insertRule` is much faster than building a string and re-injecting `<style>`, because the browser doesn't reparse the entire sheet.

### 16.4 Constructible Stylesheets

```javascript
const sheet = new CSSStyleSheet();
sheet.replaceSync(`
  .toast { position: fixed; ... }
`);

// Apply globally:
document.adoptedStyleSheets = [...document.adoptedStyleSheets, sheet];

// Apply to a shadow root:
shadowRoot.adoptedStyleSheets = [sheet];
```

`adoptedStyleSheets` is the modern way to share a stylesheet between a document and its shadow roots **without** the browser cloning bytes per shadow root. It is the foundation of efficient Web Components in 2024–2026.

### 16.5 The Dynamic Stylesheet Pattern

```javascript
class ThemeSystem {
  constructor() {
    this.sheet = new CSSStyleSheet();
    document.adoptedStyleSheets.push(this.sheet);
  }

  apply(theme) {
    const declarations = Object.entries(theme)
      .map(([k, v]) => `--${k}: ${v};`)
      .join(' ');
    this.sheet.replaceSync(`:root { ${declarations} }`);
  }
}

const themes = new ThemeSystem();
themes.apply({ brand: 'oklch(60% 0.2 250)', surface: 'white' });
```

One stylesheet object, mutated. No string concatenation, no DOM thrash.

---

## 17. CSS-in-JS Strategies

The trade-off matrix between build-time and runtime CSS in JavaScript:

| Strategy           | When CSS is generated | Runtime cost  | Build cost | Type-safety | Theming runtime |
|--------------------|-----------------------|---------------|------------|-------------|-----------------|
| Plain `.css`       | None — author-written | Zero          | Zero       | None        | Custom props    |
| **CSS Modules**    | Build (compile)       | Zero          | Compile    | Class names | Custom props    |
| **Tailwind**       | Build (extract)       | Zero          | Extract    | None        | Custom props    |
| **Vanilla-extract**| Build (TS → CSS)      | Zero          | Compile    | Full        | TS objects      |
| **styled-components / emotion** | Runtime  | Per-render    | None       | Partial     | Theme provider  |
| **Linaria / Compiled** | Build (extract)   | Zero          | Extract    | Partial     | Custom props    |

### 17.1 CSS Modules

```javascript
// Button.module.css
.button { background: var(--brand); padding: 0.5rem 1rem; }

// Button.jsx
import styles from './Button.module.css';
<button className={styles.button}>Click</button>

// Compiled output: .Button_button__a3f9 { background: var(--brand); ... }
```

Build-time scoping. Class names are hashed; selectors don't collide. **No runtime cost.** Best for component-scoped CSS in a build pipeline.

### 17.2 Styled-Components / Emotion

```javascript
const Button = styled.button`
  background: ${props => props.theme.brand};
  padding: 0.5rem 1rem;
`;
```

Runtime: the tagged template is evaluated, a class name is generated, a `<style>` rule is inserted, and the class is applied. Cost: per-render reconciliation, hash computation, possible DOM mutation. Theming via context provider.

### 17.3 Vanilla-Extract

```typescript
// theme.css.ts
import { createTheme } from '@vanilla-extract/css';
export const [themeClass, vars] = createTheme({
  brand: 'oklch(60% 0.2 250)',
  spacing: { sm: '0.5rem', md: '1rem' },
});

// Button.css.ts
import { style } from '@vanilla-extract/css';
import { vars } from './theme.css';
export const button = style({
  background: vars.brand,
  padding: vars.spacing.sm,
});
```

Compile-time: the `.css.ts` file is run, generates a `.css` file. Runtime is zero. You get full TypeScript type safety on every variable.

### 17.4 Tailwind

```html
<button class="bg-brand px-4 py-2 rounded">Click</button>
```

Build-time: Tailwind scans files for class names and generates the matching CSS. Modern Tailwind v4 uses the cascade-layers system natively. Runtime is zero.

### 17.5 The Build-vs-Runtime Trade-Off

| Concern                     | Build-time wins | Runtime wins |
|------------------------------|-----------------|--------------|
| Performance                  | Yes             | No           |
| Initial bundle size          | Yes             | No           |
| Server-side rendering        | Yes             | Tricky       |
| Dynamic theming              | Custom props    | Native       |
| Per-component customization  | Custom props    | Direct       |

**The 2024–2026 consensus**: build-time + custom properties for runtime dynamism is the lowest-cost combination.

---

## 18. Style Recalc Performance

Style recalc is the browser stage that produces computed values from declared values. It runs:
- On initial load.
- On any DOM mutation that affects matched rules.
- On viewport resize (media query re-evaluation).
- On class/attribute changes.

### 18.1 What Triggers Recalc

| Mutation                     | Triggers Recalc on...           |
|------------------------------|----------------------------------|
| `element.classList.add('x')` | The element + its descendants if the class affects them |
| `element.setAttribute(...)`  | The element + descendants matching attribute selectors |
| Adding/removing a node       | The new tree's worth of style work |
| Hovering an element          | The element + ancestors if `:hover` is on a descendant selector |
| Window resize                | All elements, if any media query flipped |
| Custom property change at root | Any element using that property |

### 18.2 The "Don't Match Many Elements" Rule

A rule like `* { box-shadow: 0 1px 2px black }` matches every element. On a 10,000-element page, that is 10,000 declared values to track, 10,000 computed values to compute, 10,000 paint records.

```css
/* Bad — universal selector */
* { transition: all 200ms; }

/* Good — only the elements you actually transition */
.btn, .card { transition: background-color 200ms, transform 200ms; }
```

### 18.3 will-change — Use Sparingly

`will-change` tells the browser "I plan to animate this property; promote a layer now." But promoted layers cost memory.

```css
/* Don't apply globally */
* { will-change: transform; }   /* DON'T */

/* Apply just before animating, remove just after */
.modal { will-change: transform, opacity; }
.modal.idle { will-change: auto; }
```

Better: rely on the browser's own heuristics for transform/opacity animations. Reserve `will-change` for cases where you've measured a stutter and validated that promotion fixes it.

### 18.4 contain — Isolation

`contain` (`css-contain-2`) tells the engine "this subtree's effects are bounded." See §19.

### 18.5 Practical Profiling

Chromium DevTools → Performance → Recalculate Style. Each entry has:
- **Number of elements styled**.
- **Reason** (class change, attribute change, etc.).
- **Time taken**.

Targeting under 4ms per recalc is a reasonable budget for a 16ms frame.

---

## 19. Containment — the contain Property

`contain` (`css-contain-2 §3`) lets you tell the browser "the consequences of layout/style/paint inside this subtree don't propagate out."

### 19.1 The Four Containments

| Value         | Effect                                                                                       |
|---------------|----------------------------------------------------------------------------------------------|
| `layout`      | Layout inside doesn't affect outside layout. The container is its own layout root.          |
| `style`       | `counter-*` and `quotes` are scoped (small effect).                                          |
| `paint`       | Descendants don't paint outside the container. Acts like `overflow: hidden` for paint.      |
| `size`        | The container's size is determined without consulting descendants. Requires explicit sizing.|
| `inline-size` | Like `size`, but only on inline axis. Required for **container queries** (§19.4).           |
| `content`     | Shorthand for `layout style paint`. The most common.                                         |
| `strict`      | Shorthand for `layout style paint size`. Requires explicit dimensions.                       |

### 19.2 The Infinite-Scroll Pattern

```css
.list-item {
  contain: content;       /* layout + style + paint */
  content-visibility: auto;
  contain-intrinsic-size: auto 200px;  /* placeholder size for offscreen items */
}
```

The browser knows offscreen list items don't affect outside layout. With `content-visibility: auto`, offscreen items are not laid out at all. On a 100,000-row list, this is the difference between minutes of layout and milliseconds.

### 19.3 inline-size Containment

Container queries (`css-contain-3 §4`) require **at minimum** `inline-size` containment so the engine can size the container before its contents.

```css
.card {
  container-type: inline-size;
  /* equivalent to: contain: inline-size; */
}

@container (min-width: 400px) {
  .card { display: grid; grid-template-columns: 1fr 2fr; }
}
```

The engine sizes `.card` from its parent's layout, then re-runs styles on `.card`'s descendants based on the matched container query.

### 19.4 The Container-Query Layout Pass

Container queries introduce a chicken-and-egg problem: the styles depend on the size, but the size depends on the styles. The engine resolves this with a **two-pass approach** at the container boundary:
1. Layout the container based on its own constraints (parent layout).
2. Use the container's resolved inline-size to evaluate `@container` rules.
3. Apply the matched rules to descendants.
4. Layout the descendants.

Containment is the contract that makes this safe — the engine knows the container's size won't change as a function of the descendants.

---

## 20. Transitions and Animations — Internals

CSS Animations Module Level 1 (`css-animations-1`) and Transitions Module Level 1 (`css-transitions-1`).

### 20.1 The Interpolation Algorithm

For each animatable property, the browser computes:

```
interpolated_value = (1 - t) * from_value + t * to_value
```

…where `t` is the eased timing function output for the elapsed fraction. For non-numeric values (colors, lengths in different units, transforms), the spec defines per-property interpolation rules (`css-values-4 §11`).

### 20.2 Compositor-Friendly Properties

Only **`transform`** and **`opacity`** can be animated entirely on the compositor thread. Other properties go through:
1. Style on the main thread.
2. Layout (if affected).
3. Paint to a layer.
4. Composite.

Animating `width`, `height`, `top`, `left` triggers layout every frame. Animating `transform: translate(...)` and `opacity` only touches composite — typically GPU-accelerated.

### 20.3 The 60fps Budget

```
16.67ms per frame = 1000ms / 60fps

Style recalc       ≤ 4ms
Layout             ≤ 4ms
Paint              ≤ 4ms
Composite          ≤ 2ms
Headroom           ~3ms for JS
```

Animating compositor-friendly properties skips style/layout/paint, so the entire 16.67ms is available for one composite operation.

### 20.4 The will-change Promotion Story

```css
.card { will-change: transform; }
```

This tells the browser to put `.card` on its own compositor layer right now. Now `.card` can be transformed without re-painting. But:
- Each layer costs memory (texture).
- Too many layers cause overdraw.
- Once a layer exists, removing it later forces a re-paint.

Use `will-change` for elements that are about to animate and remove it when the animation ends.

### 20.5 ScrollTimeline and ViewTimeline

CSS Animations Level 2 introduces **scroll-driven animations**:

```css
@keyframes fade-in {
  from { opacity: 0; }
  to   { opacity: 1; }
}

.card {
  animation: fade-in linear;
  animation-timeline: view();        /* drive animation by element's view progress */
  animation-range: entry 0% cover 30%;
}
```

The animation progress is tied to **scroll position**, not time. On supporting browsers (Chrome 115+), this runs on the compositor — buttery-smooth scroll-linked effects without `requestAnimationFrame`.

### 20.6 @starting-style — Entry Animations

CSS Transitions Level 2 introduces `@starting-style` for "the value the element starts at, before normal styles apply":

```css
.toast {
  opacity: 1;
  transform: translateY(0);
  transition: opacity 300ms, transform 300ms;
}

@starting-style {
  .toast {
    opacity: 0;
    transform: translateY(20px);
  }
}
```

When `.toast` is added to the DOM, the browser:
1. Applies `@starting-style` first (opacity: 0, translateY: 20px).
2. Transitions to the normal style (opacity: 1, translateY: 0).

No JavaScript, no `requestAnimationFrame`, no double-rAF tricks. Pure declarative entry animation.

### 20.7 Keyframe Resolution

```css
@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50%      { transform: scale(1.05); }
}

.btn:hover { animation: pulse 600ms ease-in-out infinite; }
```

The keyframe positions are resolved at parse time. The interpolation between keyframes uses the easing function. The `animation-direction`, `animation-fill-mode`, and `animation-play-state` properties affect playback semantics, not interpolation.

---

## 21. Idioms at the Internals Depth

### 21.1 The Reset + Layer + Component Pattern

```css
@layer reset, base, tokens, layout, components, utilities, overrides;

@layer reset {
  :where(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :where(html) { color-scheme: light dark; }
}

@layer base {
  :where(body) {
    font-family: system-ui, sans-serif;
    line-height: 1.5;
    color: light-dark(black, white);
    background: light-dark(white, oklch(20% 0.02 250));
  }
}

@layer tokens {
  :root {
    --brand-h: 250;
    --brand: oklch(60% 0.2 var(--brand-h));
    --brand-tinted: color-mix(in oklch, var(--brand), white 30%);
    --space-1: 0.25rem;
    --space-2: 0.5rem;
    --space-3: 1rem;
    --space-4: 2rem;
  }
}

@layer components {
  .btn {
    background: var(--btn-bg, var(--brand));
    color: var(--btn-fg, white);
    padding: var(--space-2) var(--space-3);
    border-radius: 0.25rem;
  }
}

@layer utilities {
  .text-center { text-align: center; }
  .stack { display: flex; flex-direction: column; gap: var(--space-2); }
}
```

Every layer does exactly its job. Specificity stays at `(0,1,0)` everywhere because everything in `reset` and `base` uses `:where()`. No `!important`. Author code in `overrides` always wins.

### 21.2 The OKLCH Theme via Custom Properties

```css
@property --brand-h { syntax: "<number>"; inherits: true; initial-value: 250; }
@property --brand-c { syntax: "<number>"; inherits: true; initial-value: 0.2; }
@property --brand-l { syntax: "<percentage>"; inherits: true; initial-value: 60%; }

:root {
  --brand:        oklch(var(--brand-l) var(--brand-c) var(--brand-h));
  --brand-hover:  oklch(calc(var(--brand-l) - 5%) var(--brand-c) var(--brand-h));
  --brand-active: oklch(calc(var(--brand-l) - 10%) var(--brand-c) var(--brand-h));
}

.theme-warm  { --brand-h: 50; }
.theme-cool  { --brand-h: 200; }
.theme-error { --brand-h: 25; --brand-c: 0.25; }
```

Hue is a single number; lightness and chroma scale uniformly via OKLCh. Hover and active states are mathematically derived. A theme is a one-line `--brand-h` change.

### 21.3 The Contained Component via Container Queries + Custom-Property API

```css
.media-card {
  container-type: inline-size;
  contain: layout paint;

  /* Component API surface */
  --card-bg: var(--surface, white);
  --card-padding: 1rem;
  --card-radius: 0.5rem;
  --card-image-ratio: 16 / 9;

  background: var(--card-bg);
  padding: var(--card-padding);
  border-radius: var(--card-radius);
  display: grid;
  gap: 1rem;
}

.media-card > .image {
  aspect-ratio: var(--card-image-ratio);
  background-size: cover;
}

@container (min-width: 30rem) {
  .media-card {
    grid-template-columns: 1fr 2fr;
  }
}

@container (min-width: 50rem) {
  .media-card {
    --card-padding: 2rem;
    grid-template-columns: 1fr 3fr;
  }
}
```

The card is **self-contained**: container-typed for the queries, contain-isolated for paint/layout. Its appearance API is a set of `--card-*` variables. It responds to its own width, not the viewport. This is the modern component primitive.

### 21.4 The `revert-layer` Override Pattern

```css
@layer base {
  a { color: var(--brand); text-decoration: underline; }
}

@layer overrides {
  .nav a { all: revert-layer; color: white; text-decoration: none; }
}
```

`all: revert-layer` undoes the previous layer's declarations on this element. Useful when a component must be a clean slate against the underlying base styles.

---

## 22. Prerequisites

- **HTML and the DOM tree** — selectors traverse this graph.
- **Box model arithmetic** — covered in `css-layout`.
- **JavaScript event-loop basics** — for understanding when `getComputedStyle` flushes layout.
- **Browser rendering pipeline** — at least the high-level "parse → style → layout → paint → composite" sequence.

## Complexity

- **Beginner** — Origin order (UA < User < Author), specificity tuple, `!important`, basic inheritance.
- **Intermediate** — Cascade layers, `:where()` / `:is()`, custom properties + `var()`, computed-vs-used distinction, `getComputedStyle`.
- **Advanced** — `@property` typed registration, OKLCh + `color-mix()` + gamut mapping, `@scope` proximity, container queries with containment, scroll-driven animations, CSSOM constructible stylesheets.
- **Expert** — Bloom-filter selector matching internals, layer-internal `!important` reversal, the formal value-resolution algorithm (declared → cascaded → specified → computed → used → actual), engine-level style invalidation graphs.

## See Also

- `css` — practical day-to-day cheatsheet on selectors, properties, common patterns.
- `css-layout` — flexbox, grid, positioning, the box model arithmetic, rendering pipeline.
- `html` — the document tree CSS attaches to.
- `html-forms` — form-specific styling (`:user-invalid`, `accent-color`, etc.).
- `javascript` — DOM and CSSOM APIs.
- `polyglot` — comparative notes against other declarative styling systems.

## References

- W3C **CSS Cascading and Inheritance Module Level 5** — `https://www.w3.org/TR/css-cascade-5/` (cascade, layers, importance, defaulting keywords).
- W3C **CSS Cascading and Inheritance Module Level 6** — `https://www.w3.org/TR/css-cascade-6/` (`@scope`, scope-proximity tiebreaker).
- W3C **CSS Properties and Values API Level 1** — `https://www.w3.org/TR/css-properties-values-api-1/` (`@property`, registered custom properties).
- W3C **CSS Custom Properties for Cascading Variables Module Level 1** — `https://www.w3.org/TR/css-variables-1/`.
- W3C **CSS Color Module Level 4** — `https://www.w3.org/TR/css-color-4/` (oklab, oklch, color spaces, gamut mapping).
- W3C **CSS Color Module Level 5** — `https://www.w3.org/TR/css-color-5/` (`color-mix()`, `light-dark()`, relative color syntax).
- W3C **CSS Containment Module Level 2 / 3** — `https://www.w3.org/TR/css-contain-2/`, `https://www.w3.org/TR/css-contain-3/`.
- W3C **CSS Values and Units Module Level 4** — `https://www.w3.org/TR/css-values-4/` (computed-value resolution).
- W3C **Selectors Level 4** — `https://www.w3.org/TR/selectors-4/` (specificity, `:is()`, `:where()`, `:has()`).
- W3C **CSSOM** — `https://www.w3.org/TR/cssom-1/` (CSSStyleSheet, CSSRule subtypes, constructible stylesheets).
- W3C **CSS Animations Level 1** — `https://www.w3.org/TR/css-animations-1/`.
- W3C **CSS Animations Level 2** — `https://www.w3.org/TR/css-animations-2/` (`animation-timeline`, scroll-driven).
- W3C **CSS Transitions Level 2** — `https://www.w3.org/TR/css-transitions-2/` (`@starting-style`).
- MDN **CSS** reference — `https://developer.mozilla.org/en-US/docs/Web/CSS`.
- web.dev **Learn CSS** — `https://web.dev/learn/css/` (modern CSS curriculum).
- CSS Working Group — `https://www.csswg.org/` (drafts, GitHub, change log).
- Björn Ottosson, **A perceptual color space for image processing** (OKLab, 2020) — `https://bottosson.github.io/posts/oklab/`.
- WHATWG **DOM** — `https://dom.spec.whatwg.org/` (the underlying tree CSS styles).
