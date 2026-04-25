# CSS (Cascading Style Sheets)

Stylesheet language for HTML — selectors, cascade, properties, custom props, units, media queries, and motion (see css-layout for flexbox, grid, positioning, container queries).

## Setup

No installation — CSS is interpreted directly by the browser. Three injection methods, in increasing specificity.

```bash
# External stylesheet (preferred — cacheable, parallel-loadable)
# In <head> of HTML document:
#   <link rel="stylesheet" href="/styles/main.css">
#   <link rel="stylesheet" href="/styles/print.css" media="print">
#   <link rel="preload" href="/styles/main.css" as="style">
```

```bash
# Inline <style> block (single-page or critical CSS)
# In <head>:
#   <style>
#     body { margin: 0; font-family: system-ui, sans-serif; }
#   </style>
```

```bash
# Inline style attribute (highest specificity except !important)
# <p style="color: red; margin-top: 1rem;">Inline-styled paragraph</p>
# Avoid in production — bypasses cascade, hurts maintenance.
```

### Order of stylesheets matters

```bash
# Last-loaded wins ties in specificity. The order:
#   <link rel="stylesheet" href="reset.css">          # 1. reset first
#   <link rel="stylesheet" href="vendor.css">         # 2. third-party
#   <link rel="stylesheet" href="components.css">     # 3. our base
#   <link rel="stylesheet" href="utilities.css">      # 4. overrides last
#   <style>...</style>                                # 5. critical inline
# Within a single file, later rules with equal specificity win.
```

### The @import directive

```bash
# Inside CSS file — chains a request, blocking parallel load.
# /* main.css */
# @import url("reset.css");           # parsed BEFORE main.css continues
# @import "components.css";           # url() optional
# @import "print.css" print;          # media-conditional
# @import "tablet.css" (min-width: 768px);
#
# Performance: @import is serial — browser must download main.css,
# parse, find @import, then download reset.css. Avoid in production.
# Prefer <link rel="stylesheet"> tags so the browser can parallelize.
#
# @layer-aware @import (modern):
# @import url("vendor.css") layer(vendor);
```

## Syntax

CSS is a list of rules. Each rule has a selector (or selector list) and a declaration block of property: value pairs.

```bash
# Basic syntax
# selector {
#   property: value;
#   property: value;
# }
#
# selector1, selector2, selector3 {       # comma = "OR"
#   shared-property: value;
# }
#
# Trailing semicolon on the LAST declaration is optional but recommended —
# adding new lines won't break syntax.
```

### Declarations

```bash
# Each declaration: property COLON value SEMICOLON
# .card {
#   color: #333;                # COLON separates property/value
#   padding: 1rem 2rem;          # SEMICOLON terminates declaration
#   background: white;           # last ; optional
# }
#
# Comments: /* ... */ — no // line comments in CSS!
# /* this is fine */
# // this is NOT a comment — entire next line is parsed as garbage
```

### At-rules

```bash
# At-rules start with @ and configure the stylesheet itself.
#
# @charset "UTF-8";                   # MUST be first, no comment before
# @import url("reset.css");           # bring in another stylesheet
# @namespace svg url("http://www.w3.org/2000/svg");
#
# @media (min-width: 768px) { ... }   # conditional rules
# @supports (display: grid) { ... }   # feature queries
# @container (min-width: 400px) { ... }   # container queries
#
# @font-face { font-family: ...; src: ...; }    # custom fonts
# @keyframes slideIn { ... }                     # animation timeline
# @page { margin: 1in; }                          # print page setup
#
# @layer reset, base, components, utilities;     # cascade layers
# @scope (.card) to (.footer) { ... }            # scoped rules (modern)
# @property --my-color { syntax: "<color>"; ... } # typed custom props
# @counter-style my-roman { ... }                 # custom list markers
```

## Selectors — Type, Class, ID

### Element / Type selector

```bash
# Targets every element of the given type.
# p   { line-height: 1.6; }      # all <p>
# h1  { font-size: 2rem; }       # all <h1>
# div { background: white; }     # all <div>
# Specificity: (0, 0, 0, 1) — one element selector.
```

### Class selector

```bash
# Targets elements with the given class attribute.
# .card        { padding: 1rem; }
# .card-large  { padding: 2rem; }
# .a.b         { ... }            # has BOTH classes (chained)
# .a .b        { ... }            # .b descendant of .a (space matters!)
#
# Specificity: (0, 0, 1, 0) — one class.
# Convention: kebab-case, BEM .block__elem--mod, or utility .text-red-500.
```

### ID selector

```bash
# Targets the element with that id attribute. IDs MUST be unique per page.
# #header     { height: 60px; }
# #login-form { ... }
#
# Specificity: (0, 1, 0, 0) — one ID beats 100 classes.
# Avoid in CSS — too specific, hard to override. Use classes instead.
# Useful for JS hooks (document.getElementById) and skip-links/anchors.
```

### Universal selector

```bash
# *      { box-sizing: border-box; }   # ALL elements
# *::before, *::after { box-sizing: inherit; }
# .panel * { color: inherit; }         # all descendants of .panel
#
# Specificity: (0, 0, 0, 0) — zero. Combinators add nothing either.
# Performance: rare standalone use. Often paired with reset patterns.
```

### Specificity ladder summary

```bash
# Highest to lowest:
#   1. inline style="..."          (1, 0, 0, 0)
#   2. id selector                 (0, 1, 0, 0)
#   3. class / attribute / pseudo-class  (0, 0, 1, 0)
#   4. element / pseudo-element    (0, 0, 0, 1)
#   5. universal *                 (0, 0, 0, 0)
# Tie? Last rule in source order wins.
# !important overrides this ladder entirely (use sparingly).
```

## Selectors — Attribute

Match elements by their attribute presence, value, or substring patterns.

```bash
# Presence
# [disabled]              { opacity: 0.5; }     # has any disabled attr
# [data-loading]          { cursor: wait; }
#
# Exact value
# [type="email"]          { border-color: blue; }
# [lang="en-US"]          { ... }
#
# Word match (whitespace-separated list)
# [class~="primary"]      { ... }   # one of the words is "primary"
# # Note: rarely used directly — use .primary instead.
#
# Language/locale prefix (= or hyphen-prefix)
# [lang|="en"]            { ... }   # matches "en" or "en-US", "en-GB"
#
# Starts with
# [href^="https://"]      { color: green; }
# [href^="mailto:"]::before { content: "✉ "; }
#
# Ends with
# [src$=".svg"]           { fill: currentColor; }
# [href$=".pdf"]::after   { content: " (PDF)"; }
#
# Contains substring
# [class*="btn"]          { cursor: pointer; }
# [href*="example.com"]   { ... }
```

### Case-insensitive flag

```bash
# Add `i` (insensitive) or `s` (sensitive, default) BEFORE the closing ].
# [type="EMAIL" i]        { ... }   # matches type=email, EMAIL, EmAiL
# [data-state="open" i]   { ... }
# [href$=".PDF" i]        { ... }   # matches .pdf, .PDF, .Pdf
#
# Specificity of attribute selectors: (0, 0, 1, 0) — same as a class.
```

## Selectors — Pseudo-Classes

Pseudo-classes describe an element's state or position. Single colon `:`.

### Interactive state

```bash
# a:link    { color: blue; }       # unvisited (rare — usually skipped)
# a:visited { color: purple; }     # visited (limited styling for privacy)
# a:hover   { text-decoration: underline; }   # mouse over
# a:active  { color: red; }        # being clicked (mousedown to mouseup)
# a:focus   { outline: 2px solid blue; }      # has keyboard focus
#
# LVHA order matters! :link, :visited, :hover, :active — otherwise
# later rules clobber earlier ones due to equal specificity.
```

### Modern focus pseudo-classes

```bash
# :focus            triggers on ANY focus (mouse click, tab, JS .focus())
# :focus-visible    triggers only when browser deems focus indicator helpful
#                   (typically keyboard / programmatic, NOT mouse click)
# :focus-within     element OR any descendant has focus
#
# button:focus            { outline: 2px solid blue; }    # mouse + kbd
# button:focus-visible    { outline: 2px solid blue; }    # kbd only
# button:focus:not(:focus-visible) { outline: none; }     # remove for mouse
# .form:focus-within      { background: #fafafa; }        # any input focused
```

### Form state

```bash
# :checked       checkbox/radio is checked, <option> is selected
# :disabled      form element is disabled
# :enabled       form element is NOT disabled
# :required      has required attribute
# :optional      does NOT have required
# :read-only     contenteditable=false / input readonly
# :read-write    editable
# :placeholder-shown   input is showing its placeholder (empty)
# :default       default option / submit button
# :valid         passes constraint validation
# :invalid       fails constraint validation
# :user-valid    valid AND user has interacted (modern, no flash on load)
# :user-invalid  invalid AND user has interacted
# :in-range      <input type=number> within min/max
# :out-of-range  outside min/max
#
# input:invalid     { border-color: red; }
# input:user-invalid { border-color: red; }   # better — no flash on empty
# input[type=checkbox]:checked + label { font-weight: bold; }
```

### Structural pseudo-classes

```bash
# :empty                 no children (not even whitespace)
# :first-child           first child of its parent
# :last-child            last child of its parent
# :only-child            only child of its parent
# :first-of-type         first sibling of its element type
# :last-of-type          last sibling of its element type
# :only-of-type          only sibling of its type
# :nth-child(n)          1-indexed; n = formula
# :nth-last-child(n)     counted from end
# :nth-of-type(n)        nth of its type
# :nth-last-of-type(n)
#
# nth-child formulas:
#   :nth-child(1)         first
#   :nth-child(odd)       1, 3, 5, ...
#   :nth-child(even)      2, 4, 6, ...
#   :nth-child(2n)        every 2nd starting at 0 → 2, 4, 6
#   :nth-child(2n+1)      odd
#   :nth-child(3n+2)      2, 5, 8, ...
#   :nth-child(-n+3)      first 3
#   :nth-child(n+4)       4th onward
```

### Document state

```bash
# :root          the root element (typically <html>) — best place for vars
# :target        element matching URL fragment (#id)
# :lang(en)      language matches
# :dir(ltr)      direction (modern)
#
# :root { --primary: #3b82f6; }
# h2:target { background: yellow; }   # highlight when URL is #section-id
```

### Logical pseudo-classes

```bash
# :is(s1, s2, s3)     matches if ANY selector matches; specificity = max
# :where(s1, s2, s3)  same as :is() but specificity is ZERO
# :not(s1, s2, s3)    matches if NONE match; specificity = highest of args
# :has(selector)      relational — parent matches if descendant matches
#
# :is(h1, h2, h3) a       { color: inherit; }
# :where(h1, h2, h3) a    { color: inherit; }   # zero specificity — easier override
# p:not(.intro)           { font-size: 0.9rem; }
# p:not(.a, .b)           { ... }                # multi-arg (modern)
# article:has(img)        { padding: 1rem; }     # parent of img
# li:has(> input:checked) { font-weight: bold; } # li containing checked input
# form:has(input:invalid) button[type=submit] { opacity: 0.5; }
```

## Selectors — Pseudo-Elements

Pseudo-elements style a part of an element. Double colon `::` (single colon also works for legacy `:before`, `:after`, `:first-line`, `:first-letter`).

```bash
# ::before          inserts content before the element's content
# ::after           inserts content after the element's content
# ::first-letter    first letter of block-level text
# ::first-line      first line of block-level text
# ::placeholder     <input> placeholder text
# ::selection       user-highlighted text
# ::backdrop        backdrop of <dialog>, fullscreen, popover
# ::marker          list-item bullet/number
# ::file-selector-button   the button inside <input type=file>
# ::cue             video/audio caption cue
# ::part(name)      a shadow DOM part exposed via part attribute
# ::slotted(s)      element slotted into a web component
# ::details-content the <details> content (modern)
```

### content property

```bash
# ::before / ::after REQUIRE content — even content: "" — to render.
#
# .tooltip::before {
#   content: "Tip: ";
#   font-weight: bold;
# }
# a[href^="https://"]::after {
#   content: " 🔗";
# }
# .clearfix::after {
#   content: "";
#   display: table;
#   clear: both;
# }
# blockquote::before {
#   content: open-quote;        # uses lang-aware quote
# }
# h2::before {
#   content: counter(section) ". ";   # counters!
# }
# img::after {
#   content: "Alt: " attr(alt);       # attribute value
# }
```

### Other pseudo-elements

```bash
# input::placeholder   { color: #999; opacity: 1; }
# ::selection          { background: yellow; color: black; }
# dialog::backdrop     { background: rgb(0 0 0 / 0.5); }
# li::marker           { color: red; font-weight: bold; }
# input[type=file]::file-selector-button {
#   background: #3b82f6;
#   color: white;
#   border: none;
#   padding: 0.5rem 1rem;
# }
# p::first-letter      { font-size: 3em; float: left; }
# p::first-line        { font-variant: small-caps; }
```

### Double colon convention

```bash
# CSS3 distinguishes pseudo-classes (:hover) from pseudo-elements (::before)
# via single vs double colon. Browsers still accept :before, :first-letter
# for legacy, but ::before is preferred for new code.
#
# /* legacy — works but discouraged */
# .x:before { content: "→"; }
# /* modern */
# .x::before { content: "→"; }
```

## Selectors — Combinators

Combinators relate selectors to each other in the DOM tree.

```bash
# Descendant     A B    — B inside A at any depth
# Child          A > B  — B is a DIRECT child of A
# Adjacent sib   A + B  — B is the IMMEDIATELY following sibling of A
# General sib    A ~ B  — B is any later sibling of A (after, same parent)
#
# article p           { line-height: 1.6; }    # any <p> inside <article>
# ul > li             { list-style: disc; }    # only direct <li> children
# h2 + p              { margin-top: 0; }       # <p> immediately after <h2>
# h2 ~ p              { color: #555; }         # any <p> after <h2> (siblings)
#
# .menu > li > a      { display: block; }      # exactly 2 levels deep
# article > * + *     { margin-top: 1em; }     # the "owl" — vertical rhythm
```

### CSS Nesting (2023+)

```bash
# Native nesting works in Chrome 112+, Safari 16.5+, Firefox 117+.
# & references the parent selector.
#
# .card {
#   padding: 1rem;
#   background: white;
#
#   & h2 {
#     margin: 0;
#   }
#
#   .child {                 # & is implied with descendant combinator
#     color: blue;
#   }
#
#   &:hover {
#     background: #f5f5f5;
#   }
#
#   &.active {
#     border-color: blue;
#   }
#
#   @media (min-width: 768px) {
#     padding: 2rem;
#   }
# }
#
# Compiles roughly to:
#   .card { padding: 1rem; ... }
#   .card h2 { margin: 0; }
#   .card .child { color: blue; }
#   .card:hover { ... }
#   .card.active { ... }
#   @media (min-width: 768px) { .card { padding: 2rem; } }
#
# Caveat: nested type selectors (& h2) need the & or are only valid via
# the relaxed grammar in latest browsers. When in doubt, use & explicitly.
```

## Specificity

Specificity is a 4-tuple `(a, b, c, d)`. Higher tuples win. Compare left-to-right.

```bash
# (a, b, c, d) where:
#   a = inline style="..."
#   b = number of IDs                  (#header)
#   c = number of classes / attrs / pseudo-classes  (.card, [type=text], :hover)
#   d = number of elements / pseudo-elements         (p, ::before)
#
# (0, 0, 0, 1)   p
# (0, 0, 0, 2)   p span
# (0, 0, 1, 0)   .card
# (0, 0, 1, 1)   p.intro
# (0, 0, 2, 0)   .a.b
# (0, 0, 2, 1)   .a .b span               # the space combinator adds no count
# (0, 1, 0, 0)   #header
# (0, 1, 1, 0)   #header .nav
# (1, 0, 0, 0)   <p style="...">
# !important     trumps the entire ladder (still ordered among !importants)
```

### What does NOT add to specificity

```bash
# - The universal selector *
# - Combinators >, +, ~, descendant space
# - Pseudo-classes :where(...) — always 0
# - The :is() / :not() / :has() outer wrappers don't add — but their
#   highest argument's specificity is used.
#
# :is(.a, #b)    → (0, 1, 0, 0)   # uses #b
# :where(.a, #b) → (0, 0, 0, 0)   # always zero
# :not(.a, #b)   → (0, 1, 0, 0)   # uses #b
# :has(#b)       → (0, 1, 0, 0)
```

### !important

```bash
# Adds a layer above normal specificity. Within !important, normal
# specificity rules still apply.
#
# .button { color: blue !important; }     # overrides plain rules
#
# Origin order with !important INVERTS:
#   1. user-agent !important
#   2. user !important
#   3. author !important       (your stylesheets)
#   4. animation
#   5. author normal           (your stylesheets)
#   6. user normal
#   7. user-agent normal
#
# Avoid !important — it's a maintenance nightmare. Fix by:
#   - lowering specificity of competing rule
#   - reorganizing source order
#   - using @layer for predictable cascade
```

### Cascade layers and specificity

```bash
# Within a layer, normal specificity applies. ACROSS layers,
# layer order wins — even a more specific rule in an earlier layer LOSES.
#
# @layer reset, base, utilities;
# @layer reset    { #header { color: red; } }     # ID, but earlier layer
# @layer utilities { .text-blue { color: blue; } } # class, later layer
# # Result: blue wins because utilities > reset, regardless of specificity.
#
# Unlayered rules WIN over layered rules (treated as a final implicit layer).
```

## The Cascade and Inheritance

The cascade decides which declaration applies when multiple match. Order:

```bash
# 1. Origin and importance
#    1.1 user-agent !important
#    1.2 user !important
#    1.3 author !important
#    1.4 transition (in-progress transitions)
#    1.5 animation (running @keyframes)
#    1.6 author normal
#    1.7 user normal
#    1.8 user-agent normal
# 2. Cascade layer (within an origin/importance)
# 3. Specificity (within a layer)
# 4. Source order (within equal specificity — last wins)
```

### Inheritance

```bash
# Some properties are AUTOMATICALLY inherited from parent to children:
#   color, font-family, font-size, font-style, font-weight, font-variant,
#   line-height, letter-spacing, word-spacing, text-align, text-indent,
#   text-transform, visibility, cursor, list-style, quotes, white-space,
#   direction
#
# Others are NOT inherited (each element starts fresh):
#   margin, padding, border, background, width, height, display,
#   position, top/right/bottom/left, opacity, transform, z-index
#
# .container { color: red; padding: 2rem; }
# .container > p { /* inherits color: red; does NOT inherit padding */ }
```

### Explicit cascade keywords

```bash
# inherit       — take the parent's COMPUTED value
# initial       — reset to the property's initial (CSS-spec) value
# unset         — inherit if inherited, else initial
# revert        — roll back to user-agent (or user) stylesheet value
# revert-layer  — roll back to the previous cascade layer
#
# .reset-children * {
#   all: unset;          # nuke every property to initial/inherited
# }
# .keep-color {
#   color: inherit;      # explicit inheritance for non-inherited props
# }
# .strip { color: initial; }   # back to spec default (often black)
#
# # Powerful: revert authors styles for a third-party widget
# .embed-area * { all: revert; }
```

## CSS Custom Properties (variables)

Custom properties are variables prefixed with `--`. Read with `var(--name)`. They cascade like normal properties and inherit through the DOM.

```bash
# Define on any element (typically :root for global)
# :root {
#   --color-primary: #3b82f6;
#   --color-text:    hsl(220 15% 20%);
#   --space-1:       0.25rem;
#   --space-2:       0.5rem;
#   --space-4:       1rem;
#   --space-8:       2rem;
#   --radius:        0.5rem;
#   --font-display:  "Inter", sans-serif;
# }
#
# Use anywhere a value is allowed
# .card {
#   color:           var(--color-text);
#   padding:         var(--space-4);
#   border-radius:   var(--radius);
#   font-family:     var(--font-display);
# }
```

### Fallback values

```bash
# var(--name, fallback) — used if --name is undefined or invalid.
#
# .item {
#   color: var(--color, #333);                       # fallback color
#   margin: var(--gap, 1rem);                        # fallback length
#   font-family: var(--font, var(--font-system, sans-serif));   # nested
# }
```

### Inheritance and scoping

```bash
# Custom properties cascade like color/font: inherited by descendants.
# Override per scope:
# :root         { --color-primary: blue; }
# .theme-dark   { --color-primary: skyblue; }
# .alert        { --color-primary: red; }
#
# .button       { background: var(--color-primary); }
# # Inside .theme-dark .button → skyblue.
# # Inside .alert .button       → red.
#
# Tip: define design tokens at :root, override on theme classes.
```

### Dynamic theming

```bash
# :root              { --bg: white; --fg: #111; }
# [data-theme=dark]  { --bg: #111;  --fg: #eee; }
#
# body { background: var(--bg); color: var(--fg); }
#
# // JS toggles:
# // document.documentElement.dataset.theme = "dark";
```

### @property — typed custom properties

```bash
# Plain custom properties are strings — can't be animated smoothly.
# @property registers a custom property with a TYPE so the browser can
# interpolate / validate / inherit it.
#
# @property --gradient-angle {
#   syntax: "<angle>";
#   inherits: false;
#   initial-value: 0deg;
# }
#
# .card {
#   background: linear-gradient(var(--gradient-angle), red, blue);
#   transition: --gradient-angle 1s linear;
# }
# .card:hover {
#   --gradient-angle: 360deg;       # NOW animates smoothly!
# }
#
# syntax values: <length>, <number>, <percentage>, <color>, <angle>,
# <time>, <length-percentage>, <integer>, <url>, <image>, <custom-ident>,
# or specific tokens like "auto | none". Use "*" for any string.
#
# Computed value vs registered: registered props are validated and
# can interpolate; unregistered are opaque strings.
```

## Units — Length

CSS lengths are absolute or relative.

### Absolute lengths

```bash
# px    pixel — 1/96 of an inch (the de-facto standard for screens)
# pt    point — 1/72 of an inch (print)
# pc    pica  — 12 points
# in    inch  — 96px
# cm    centimeter — 37.795px
# mm    millimeter — 3.7795px
# Q     quarter-mm — 0.25mm
#
# .doc { width: 8.5in; }                 # print sheet
# .label { font-size: 12pt; }            # typographic point
# .border { border-width: 1px; }         # screen — px is fine
#
# Caveat: only px is widely used for screens. The rest are for print.
```

### Font-relative lengths

```bash
# em       relative to the FONT-SIZE OF THIS ELEMENT (or parent for font-size)
# rem      relative to the ROOT element font-size (typically 16px)
# ex       x-height of the current font (~0.5em, font-dependent)
# ch       width of "0" character of the current font (monospace approx)
# ic       width of "水" (CJK water ideograph) — for CJK layouts
# rex/rch/ric  rem variants of the above (root-relative)
#
# html { font-size: 16px; }
# .a { font-size: 1.5rem; }       /* 24px */
# .b { font-size: 1.5em; }        /* 1.5 × parent font-size */
# .c { padding: 2em; }            /* 2 × this element's font-size */
# .d { width: 60ch; }             /* ~60-char readable column */
#
# Tip: rem for font-size and global spacing; em for component-internal
# spacing that should scale with text.
```

### Viewport units

```bash
# vw       1% of viewport WIDTH
# vh       1% of viewport HEIGHT
# vmin     1% of the smaller dimension
# vmax     1% of the larger dimension
# vi       1% of viewport in inline direction (writing-mode aware)
# vb       1% of viewport in block direction
#
# .hero { min-height: 100vh; }   /* full viewport tall */
# .gutter { padding: 0 5vw; }
# .square { width: 50vmin; height: 50vmin; }
#
# Mobile chrome problem: 100vh on mobile includes the address bar that
# may collapse — so 100vh > visible area. Use the dynamic variants:
#
# svh      SMALL viewport — assumes UI chrome shown (smallest viewport)
# lvh      LARGE viewport — assumes UI chrome hidden (largest viewport)
# dvh      DYNAMIC viewport — updates as chrome shows/hides
#
# .hero { min-height: 100dvh; }  /* recommended for mobile heroes */
# Fallback: min-height: 100vh; min-height: 100dvh;
```

### Container query units

```bash
# cqw      1% of container's width
# cqh      1% of container's height
# cqi      1% of container's inline size
# cqb      1% of container's block size
# cqmin    smaller of cqi/cqb
# cqmax    larger
#
# .card-container { container-type: inline-size; }
# .card { padding: 5cqi; font-size: clamp(1rem, 4cqi, 1.5rem); }
#
# # See css-layout for full container query coverage.
```

## Units — Other

### Percentages

```bash
# % — relative to the parent (resolves differently per property!).
#   width: 50%       → 50% of parent's content-box width
#   margin: 10%      → 10% of parent's WIDTH (even for top/bottom!)
#   padding: 10%     → 10% of parent's WIDTH (even for top/bottom!)
#   top: 50%         → 50% of containing block's height (positioned)
#   font-size: 50%   → 50% of parent's font-size
#   line-height: 150% → 150% of own font-size
```

### Math functions

```bash
# calc() — arithmetic across units
# .col { width: calc(100% / 3 - 1rem); }
# .gap { margin-top: calc(1rem + 2vh); }
# # Operators: + - * /. Spaces REQUIRED around + and -.
# # Bad:  calc(100%-20px)    # no space → invalid
# # Good: calc(100% - 20px)
#
# min() — pick smallest
# .container { width: min(100%, 1200px); }   # never wider than 1200
#
# max() — pick largest
# .text { font-size: max(1rem, 16px); }      # never smaller than 16px
#
# clamp(min, preferred, max) — fluid value with bounds
# h1 { font-size: clamp(1.5rem, 2.5vw + 1rem, 3rem); }
# # min = 1.5rem, scales with viewport, capped at 3rem.
```

### Angles

```bash
# deg     degrees   (full turn = 360deg)
# grad    gradians  (full turn = 400grad)
# rad     radians   (full turn = 2π ≈ 6.2832rad)
# turn    turns     (full turn = 1turn)
#
# .arrow { transform: rotate(45deg); }
# .spin  { transform: rotate(0.25turn); }   # 90deg
# .skew  { transform: skewX(10deg); }
```

### Time

```bash
# s     seconds
# ms    milliseconds
#
# transition-duration: 300ms;
# animation-duration: 1.5s;
# transition: opacity .3s ease-out;       # leading dot is fine
```

### Frequency and resolution

```bash
# Hz, kHz — used by aural CSS (deprecated)
# dpi    dots per inch
# dpcm   dots per cm
# dppx   dots per px (1dppx = 96dpi)
#
# @media (min-resolution: 2dppx) {     # retina / high-DPI
#   .logo { background-image: url(logo@2x.png); }
# }
```

## Colors

CSS supports many color formats.

### Named colors

```bash
# 147 named colors from CSS spec.
# red, green, blue, white, black, gray, silver, maroon, olive, lime,
# aqua, teal, navy, fuchsia, purple, orange, pink, brown, gold, ...
# transparent — equivalent to rgb(0 0 0 / 0)
# currentColor — the current value of the `color` property
#
# .link { color: blue; }
# .border { border-color: currentColor; }   # matches text color
```

### RGB / RGBA

```bash
# Legacy comma syntax (still works)
# rgb(255, 0, 0)
# rgba(255, 0, 0, 0.5)
#
# Modern space-separated with slash for alpha
# rgb(255 0 0)
# rgb(255 0 0 / 0.5)
# rgb(255 0 0 / 50%)
#
# Each channel: 0–255 integer or 0%–100% percentage.
# .a { color: rgb(59 130 246); }        # blue-500-ish
# .b { background: rgb(0 0 0 / 0.05); } # 5% black overlay
```

### HSL / HSLA

```bash
# hsl(hue, saturation, lightness)
# hue: 0–360 degrees (0=red, 120=green, 240=blue)
# sat: 0%–100% (0=gray)
# light: 0%–100% (0=black, 50=color, 100=white)
#
# hsl(220, 90%, 60%)              # legacy comma
# hsl(220 90% 60%)                # modern space
# hsl(220 90% 60% / 0.8)          # with alpha
#
# Easy to derive variants:
# --primary: hsl(220 90% 60%);
# --primary-light: hsl(220 90% 75%);
# --primary-dark:  hsl(220 90% 45%);
```

### HWB

```bash
# hwb(hue whiteness blackness / alpha)
# Easier mental model for designers.
# hwb(220 20% 30%)        # blueish, 20% white, 30% black
```

### Modern color spaces — lab/lch/oklab/oklch

```bash
# lab() / lch() / oklab() / oklch() — perceptually uniform color spaces.
# Better gradients (no muddy gray midpoint), wider gamut (P3 displays).
#
# oklch(L C H)        L=0..1 lightness, C=0..0.4 chroma, H=0..360 hue
# oklab(L a b)        L=0..1, a/b ≈ -0.4..0.4
#
# .a { color: oklch(0.7 0.15 220); }              # vibrant blue
# .b { color: oklch(0.7 0.15 220 / 0.5); }
#
# # Smooth gradients
# linear-gradient(in oklch, red, blue)            # no muddy purple-gray midpoint
```

### color() function and predefined spaces

```bash
# color(srgb 1 0 0)              # explicit sRGB
# color(display-p3 1 0 0)        # wide gamut
# color(rec2020 1 0 0)           # ultra-wide
# color(xyz 1 0 0)
#
# # P3 fallback pattern
# .a { color: red; }
# @supports (color: color(display-p3 1 0 0)) {
#   .a { color: color(display-p3 1 0.2 0.2); }
# }
```

### Hex

```bash
# #RGB           shorthand   #f00 = #ff0000
# #RGBA          shorthand with alpha   #f008 = #ff000088
# #RRGGBB        full
# #RRGGBBAA      full with alpha
#
# .a { color: #3b82f6; }
# .b { background: #00000080; }   # 50% black
```

### color-mix() and light-dark()

```bash
# color-mix() — mix two colors in a chosen color space
# color-mix(in oklch, red 30%, blue)              # 30% red, 70% blue
# color-mix(in srgb, var(--c), white 20%)         # tint
# color-mix(in srgb, var(--c), black 20%)         # shade
#
# light-dark() — automatic theme switching
# :root { color-scheme: light dark; }
# .a { color: light-dark(black, white); }
# .b { background: light-dark(white, #111); }
# # Browser picks based on user preference + color-scheme.
```

## Backgrounds

```bash
# Shorthand: background: color image position / size repeat origin clip attachment;
#
# .a { background: #fff; }                            # color only
# .b { background: url("/img/bg.jpg"); }              # image only
# .c { background: #fff url("/img/bg.jpg") no-repeat center / cover; }
```

### background-color, background-image

```bash
# .a { background-color: #fff; }
# .b { background-image: url("/img/bg.jpg"); }
# .c { background-image: linear-gradient(red, blue), url("/bg.jpg"); }
#
# # Multiple comma-separated images stack — first is on top.
```

### Gradients

```bash
# linear-gradient(direction, color stops)
# linear-gradient(to right, red, blue)              # left → right
# linear-gradient(45deg, red, blue)                 # 45 degrees
# linear-gradient(red 0%, yellow 50%, blue 100%)    # explicit stops
# linear-gradient(red, blue 60%, green)             # blue at 60%
#
# radial-gradient(shape size at position, color stops)
# radial-gradient(circle, red, blue)
# radial-gradient(ellipse at top, red, blue)
# radial-gradient(circle 200px at 50% 50%, red, transparent)
#
# conic-gradient(from angle at position, color stops)
# conic-gradient(red, yellow, green, blue, red)     # color wheel
# conic-gradient(from 90deg, red, blue)
#
# # Repeating variants:
# repeating-linear-gradient(45deg, #eee 0 10px, #fff 10px 20px)   # stripes
# repeating-radial-gradient(circle, #eee 0 5px, #fff 5px 10px)
```

### Multiple backgrounds

```bash
# Comma-separated — each property must align (or be one value for all).
# .a {
#   background-image:    url(top.png), url(middle.png), url(bottom.png);
#   background-position: top,         center,           bottom;
#   background-repeat:   no-repeat,   no-repeat,        no-repeat;
# }
#
# Order: first listed is RENDERED ON TOP.
```

### background-position / size / repeat / attachment

```bash
# background-position
#   keywords: top | bottom | left | right | center
#   percentage: 50% 50%   length: 10px 20px
#   modern: top 1rem right 2rem (named offsets)
# .a { background-position: center; }
# .b { background-position: 100% 50%; }
#
# background-size
#   auto | length | percentage | cover | contain | length length
# .c { background-size: cover; }      # fill, may crop
# .d { background-size: contain; }    # fit, may letterbox
# .e { background-size: 200px auto; }
#
# background-repeat
#   repeat | no-repeat | repeat-x | repeat-y | space | round
# .f { background-repeat: no-repeat; }
#
# background-attachment
#   scroll | fixed | local
#   fixed = stays put as page scrolls (parallax)
#   local = scrolls with element's content (inside scroll container)
# .g { background-attachment: fixed; }
```

### background-origin and background-clip

```bash
# background-origin: where the image starts
#   border-box | padding-box | content-box (default for image)
#
# background-clip: where the background is painted
#   border-box (default) | padding-box | content-box | text
#
# # Famous trick: gradient text
# .gradient-text {
#   background: linear-gradient(45deg, #f00, #00f);
#   -webkit-background-clip: text;
#   background-clip: text;
#   color: transparent;
# }
```

## Borders

```bash
# Shorthand: border: width style color;
# .a { border: 1px solid #ccc; }
# .b { border: 2px dashed red; }
#
# Per-side
# .c { border-top: 1px solid #eee; }
# .d { border-left: 4px solid var(--accent); }   # left rail accent
#
# Individual longhands
# border-width: 1px 2px 1px 2px;     # T R B L
# border-style: solid;
# border-color: red green blue gold;  # T R B L
#
# Styles: none, hidden, solid, dashed, dotted, double, groove, ridge,
#         inset, outset
```

### border-radius

```bash
# .a { border-radius: 8px; }
# .b { border-radius: 50%; }                   # circle (square element)
# .c { border-radius: 8px 16px; }              # TL/BR  TR/BL
# .d { border-radius: 8px 16px 24px 32px; }    # TL TR BR BL
#
# # Different horizontal vs vertical radii (ellipses)
# .e { border-radius: 50px / 20px; }           # all corners
# .f { border-radius: 50px 25px / 30px 10px; }
#
# # Per-corner longhands
# border-top-left-radius: 8px;
# border-top-right-radius: 8px;
# border-bottom-right-radius: 0;
# border-bottom-left-radius: 0;
#
# # Pill shape
# .pill { border-radius: 9999px; }   # or 50vw — guaranteed full-curve
```

### border-image

```bash
# Use an image / gradient as the border, with 9-slice scaling.
# .a {
#   border: 20px solid;            # width still required
#   border-image-source: url("frame.png");
#   border-image-slice: 30 fill;
#   border-image-repeat: round;
# }
# # Shorthand:
# .b {
#   border: 20px solid;
#   border-image: linear-gradient(red, blue) 1;
# }
```

### border-collapse on tables

```bash
# table { border-collapse: separate; }   # default — gaps with border-spacing
# table { border-collapse: collapse;  }  # adjacent cells share border
# table { border-spacing: 8px; }
```

## Outline

Outline is a focus-indicator border that does NOT take space in the layout.

```bash
# outline: width style color;
# .a:focus-visible { outline: 2px solid #3b82f6; }
# .b:focus { outline: 2px dashed red; outline-offset: 4px; }
#
# Per-property
# outline-width: 2px;
# outline-style: solid | dashed | dotted | double | none | auto;
# outline-color: blue;
# outline-offset: 4px;            # gap between element and outline
#
# Difference from border:
#   - outline does NOT affect box-model size — overlays the element
#   - outline does NOT support per-side or per-corner
#   - outline can have outline-offset (border can't)
#   - outline-style: auto = browser default (often a glow halo)
```

### Never remove focus without replacement

```bash
# # WRONG — kills keyboard accessibility
# button:focus { outline: none; }
#
# # RIGHT — replace with a custom indicator
# button { outline: none; }      # remove default
# button:focus-visible {         # only when keyboard-focused
#   outline: 2px solid #3b82f6;
#   outline-offset: 2px;
#   box-shadow: 0 0 0 4px rgb(59 130 246 / 0.3);
# }
#
# # Or just keep default outline:
# button:focus { outline: 2px solid; outline-offset: 2px; }
```

## Box Shadow

```bash
# box-shadow: x y blur spread color [inset];
# .a { box-shadow: 0 2px 4px rgb(0 0 0 / 0.1); }
# .b { box-shadow: 0 0 0 3px blue;             }   # solid 3px ring
# .c { box-shadow: inset 0 2px 4px rgb(0 0 0 / 0.2); }   # inner shadow
#
# Multiple shadows comma-separated, order = top to bottom:
# .layered {
#   box-shadow:
#     0 1px 2px rgb(0 0 0 / 0.05),
#     0 2px 4px rgb(0 0 0 / 0.05),
#     0 4px 8px rgb(0 0 0 / 0.05),
#     0 8px 16px rgb(0 0 0 / 0.05);
# }
#
# # Layered shadow trick — multiple low-opacity shadows = soft depth
```

### Performance

```bash
# Large blur radius (>20px) is expensive to repaint. To animate:
# .lift {
#   transition: transform 0.2s, box-shadow 0.2s;
#   box-shadow: 0 2px 4px rgb(0 0 0 / 0.1);
# }
# .lift:hover {
#   transform: translateY(-2px);
#   box-shadow: 0 8px 16px rgb(0 0 0 / 0.15);
# }
#
# # Trick: animate a pseudo-element's opacity instead of box-shadow
# .card { position: relative; }
# .card::after {
#   content: "";
#   position: absolute; inset: 0;
#   box-shadow: 0 16px 32px rgb(0 0 0 / 0.2);
#   opacity: 0; transition: opacity 0.2s;
# }
# .card:hover::after { opacity: 1; }
```

## Text

CSS controls every aspect of text rendering.

### Font

```bash
# font-family: stack with fallbacks. Quote multi-word names.
# body {
#   font-family: -apple-system, BlinkMacSystemFont, "Segoe UI",
#                Roboto, Helvetica, Arial, sans-serif;
# }
#
# # Generic families: serif, sans-serif, monospace, cursive, fantasy,
# # system-ui, ui-serif, ui-sans-serif, ui-monospace, ui-rounded, math
#
# # System font stack (modern preference)
# body { font-family: system-ui, sans-serif; }
#
# font-size: 16px;       # px, rem, em, %, smaller, larger, named (medium)
# font-weight: 100-900   # 400=normal, 700=bold, also "normal", "bold",
#                          "lighter", "bolder"
# font-style: normal | italic | oblique;
# font-variant: small-caps;     # also normal | all-small-caps | etc.
# font-stretch: 50% to 200% | ultra-condensed | normal | ultra-expanded
# font-display: swap | block | fallback | optional | auto;   # in @font-face
```

### Font shorthand

```bash
# font: style variant weight stretch size/line-height family;
# .a { font: italic small-caps bold 16px/1.5 "Helvetica", sans-serif; }
# .b { font: 1rem/1.5 system-ui, sans-serif; }     # minimal
# # Note: shorthand RESETS unspecified properties to initial.
```

### Spacing and decoration

```bash
# line-height        unitless preferred (1.5 not 24px) — scales with font
# letter-spacing     0.05em — tracking
# word-spacing       0.25em
# text-indent        first-line indent
# text-align         left | right | center | justify | start | end
# text-align-last    align of LAST line in justified text
# text-decoration    shorthand: line style color thickness
#   text-decoration-line:  underline | overline | line-through | none
#   text-decoration-style: solid | double | dotted | dashed | wavy
#   text-decoration-color: red
#   text-decoration-thickness: 2px | from-font | auto
#   text-underline-offset: 4px       # gap between text and underline
# text-transform     none | capitalize | uppercase | lowercase | full-width
#
# .a { text-decoration: underline wavy red 2px; }
# .b { text-decoration: none; }
# .c { line-height: 1.5; }              # ratio — preferred
# .d { letter-spacing: -0.02em; }       # tighten display headings
```

### Line breaking and wrapping

```bash
# white-space:
#   normal      — wraps at whitespace, collapses sequences
#   nowrap      — never wraps
#   pre         — preserves whitespace, no wrap
#   pre-wrap    — preserves, wraps as needed
#   pre-line    — collapses extra spaces, preserves \n, wraps
#   break-spaces — like pre-wrap but breaks before/after spaces too
#
# word-break: normal | break-all | keep-all | break-word
# overflow-wrap: normal | break-word | anywhere
# hyphens: none | manual | auto (requires lang attribute)
# tab-size: 4   # for <pre> / <code>
#
# # Truncate to one line with ellipsis
# .truncate {
#   overflow: hidden;
#   text-overflow: ellipsis;
#   white-space: nowrap;
# }
#
# # Truncate to N lines (line clamp)
# .clamp-3 {
#   display: -webkit-box;
#   -webkit-box-orient: vertical;
#   -webkit-line-clamp: 3;
#   line-clamp: 3;
#   overflow: hidden;
# }
#
# # Long URL safety
# .wrap-anywhere { overflow-wrap: anywhere; }
```

## Web Fonts

Use `@font-face` to load custom fonts.

```bash
# @font-face {
#   font-family: "Inter";
#   src: local("Inter"),
#        url("/fonts/inter.woff2") format("woff2"),
#        url("/fonts/inter.woff")  format("woff");
#   font-weight: 100 900;        /* variable font weight range */
#   font-style:  normal;
#   font-display: swap;
#   unicode-range: U+0000-00FF, U+0131, U+0152-0153;   /* subset */
# }
#
# body { font-family: "Inter", system-ui, sans-serif; }
```

### format() values

```bash
# format("woff2")          # WOFF2 — preferred (smallest)
# format("woff")           # WOFF — fallback (older browsers)
# format("truetype")       # TTF
# format("opentype")       # OTF
# format("embedded-opentype")  # EOT (legacy IE)
# format("svg")            # SVG fonts (deprecated)
#
# # Variable font hint
# format("woff2 supports variations")
# format("woff2-variations")
```

### local() then url() pattern

```bash
# Try the user's installed copy first; fall back to download.
# @font-face {
#   font-family: "Roboto";
#   src: local("Roboto"),                    /* skip download if installed */
#        local("Roboto-Regular"),
#        url("/fonts/Roboto.woff2") format("woff2");
# }
```

### font-display values

```bash
# auto      — browser default (usually block-ish)
# block     — invisible text up to ~3s while font loads, then swap
# swap      — show fallback immediately, swap when ready (FOUT)
# fallback  — short block (~100ms), then swap, then give up
# optional  — short block; if not ready, KEEP fallback (best for slow nets)
#
# # Usual choice:
# @font-face { ...; font-display: swap; }
```

## Typography Variables and Modern Features

```bash
# Variable fonts allow continuous weight/width/slant via axes.
# font-variation-settings sets named or custom axes.
#
# .heading {
#   font-family: "Inter", sans-serif;
#   font-variation-settings: "wght" 600, "wdth" 90, "slnt" -10;
# }
#
# # OpenType features (kerning, ligatures, etc.)
# .body {
#   font-feature-settings: "kern" 1, "liga" 1, "calt" 1;
#   font-kerning: normal;
#   font-variant-ligatures: common-ligatures contextual;
#   font-variant-numeric: tabular-nums;        /* aligned digit columns */
# }
#
# # Optical sizing (auto-adjusts micro-typography for size)
# .h1 { font-optical-sizing: auto; }
#
# # Modern text-edge / leading-trim — limited support but coming
# .title {
#   text-edge: cap alphabetic;       /* trim leading whitespace */
#   leading-trim: both;
# }
```

## Lists

```bash
# list-style-type
#   disc | circle | square | none
#   decimal | decimal-leading-zero
#   lower-alpha | upper-alpha
#   lower-roman | upper-roman
#   lower-greek | armenian | georgian | hebrew
#   ... plus many more language-specific
#
# ul { list-style-type: disc; }
# ol { list-style-type: decimal; }
# .clean { list-style: none; padding: 0; }     # nav menus
#
# list-style-image:  url("bullet.svg")
# list-style-position: outside (default) | inside
# list-style: type position image;             # shorthand
#
# .square { list-style-type: square; }
# .none   { list-style: none; }
```

### ::marker pseudo-element

```bash
# Style the bullet/number directly.
# li::marker {
#   color: red;
#   font-weight: bold;
#   content: "→ ";        # custom bullet text
# }
# ol li::marker {
#   content: counter(list-item) ". ";
#   font-variant-numeric: tabular-nums;
# }
```

### Canonical list reset

```bash
# /* Lists used for navigation/widgets — strip semantics */
# ul[role="list"], ol[role="list"] {
#   list-style: none;
#   padding: 0;
#   margin: 0;
# }
# # Add role="list" in HTML to keep list semantics for AT users.
```

## Tables

```bash
# table       { border-collapse: collapse; border-spacing: 0; }
# thead, tfoot { background: #f5f5f5; }
# th, td      { padding: 0.5rem 1rem; text-align: left; vertical-align: top; }
# th          { font-weight: 600; }
# tr:nth-child(even) td { background: #fafafa; }
#
# table       { border-collapse: separate; border-spacing: 8px; }   # gaps
# table       { table-layout: fixed; width: 100%; }                  # equal cols
# table       { caption-side: top | bottom; }
# table       { empty-cells: show | hide; }
#
# th[scope=col] { ... }     # respect scope attribute
```

### Common reset

```bash
# table {
#   width: 100%;
#   border-collapse: collapse;
#   border-spacing: 0;
# }
# th, td {
#   padding: 0.5rem 0.75rem;
#   border-bottom: 1px solid #eee;
#   text-align: left;
#   vertical-align: top;
# }
```

## Counters

CSS counters are programmatic numbering separate from HTML.

```bash
# Reset, increment, display.
# body { counter-reset: section; }              # init counter "section" to 0
# h2   { counter-increment: section; }
# h2::before { content: counter(section) ". "; }
#
# # Multiple counters and styles
# h2::before { content: counter(section, upper-roman) ". "; }
#
# # Nested
# ol { counter-reset: list-item; }
# li::before {
#   content: counters(list-item, ".") " ";    # plural counters() = nested
#   counter-increment: list-item;
# }
#
# # Reset to non-zero
# body { counter-reset: section 5; }
```

### @counter-style

```bash
# Define a fully custom counter style.
# @counter-style my-stars {
#   system: cyclic;
#   symbols: "★" "☆";
#   suffix: " ";
# }
# li { list-style: my-stars; }
#
# Systems: cyclic | numeric | alphabetic | symbolic | fixed | additive |
#          extends <name>
```

## Pseudo-Element Generated Content

The `content` property on `::before` / `::after` injects content not in the DOM.

```bash
# Strings
# .tooltip::before { content: "Tip: "; }
#
# Attribute values
# a[href]::after { content: " (" attr(href) ")"; }
#
# Counters
# h2::before { content: counter(section) ". "; }
#
# URL (image)
# .badge::before { content: url("/icons/star.svg"); }
#
# Combined
# .breadcrumb a::after {
#   content: " " attr(data-sep) " ";
#   color: #999;
# }
#
# Quotes (lang-aware)
# blockquote::before { content: open-quote; }
# blockquote::after  { content: close-quote; }
#
# Special:
#   normal — none for ::before/::after
#   none   — no content, pseudo doesn't generate
#   "" (empty string) — generates an empty box (useful for clearfix)
#
# .clearfix::after { content: ""; display: table; clear: both; }
```

### Accessibility

```bash
# Generated content is NOT in the accessibility tree by default —
# screen readers usually skip it. Use only for decorative text.
# For semantic content, put it in HTML.
#
# # Modern: alt text in content (Chromium 91+, Safari 17.4+)
# .icon::before {
#   content: url("/icon.svg") / "Search";   # alt text after slash
# }
```

## Visibility and Display

### display

```bash
# display values:
#   block          — full-width, stacks vertically
#   inline         — flows with text, no width/height
#   inline-block   — inline flow, accepts width/height/margin
#   none           — removed from layout entirely (and AT tree)
#   flex           — flexbox container (see css-layout)
#   inline-flex
#   grid           — grid container (see css-layout)
#   inline-grid
#   table | table-row | table-cell | table-header-group | ...
#   list-item
#   contents       — element acts as if it weren't there (children become parent's)
#   flow-root      — establishes a new BFC (clears floats)
#
# # Two-value display (modern)
# .a { display: block flex; }      /* outer block, inner flex */
# .b { display: inline grid; }     /* outer inline, inner grid */
```

### visibility

```bash
# visibility: visible | hidden | collapse;
# .a { visibility: hidden; }       # invisible BUT keeps space + AT tree
# tr { visibility: collapse; }     # collapses table rows (acts as none)
#
# Difference from display: none — visibility:hidden preserves layout.
# Difference from opacity:0 — visibility:hidden REMOVES from accessibility tree.
```

### Visual hiding comparison

```bash
# display: none       — gone from DOM render and AT tree, no space
# visibility: hidden  — invisible, keeps space, NOT in AT tree
# opacity: 0          — invisible, keeps space, IN AT tree, still clickable
# .sr-only            — visually hidden but kept in AT tree (screen reader only)
#
# # The canonical screen-reader-only hide:
# .sr-only {
#   position: absolute;
#   width: 1px;
#   height: 1px;
#   padding: 0;
#   margin: -1px;
#   overflow: hidden;
#   clip: rect(0, 0, 0, 0);
#   clip-path: inset(50%);
#   white-space: nowrap;
#   border: 0;
# }
```

### content-visibility

```bash
# content-visibility: visible | hidden | auto
# `auto` skips rendering offscreen content (huge perf win for long pages).
#
# .article {
#   content-visibility: auto;
#   contain-intrinsic-size: auto 800px;   # placeholder size hint
# }
```

## Cursor and User Interaction

```bash
# cursor: keyword | url(...) [x y] [, fallback...]
#
# Keywords:
#   default        — system default arrow
#   pointer        — hand (links, buttons)
#   text           — I-beam (text)
#   move           — 4-way move arrows
#   grab / grabbing — open / closed hand
#   wait           — hourglass (UI is busy)
#   progress       — busy with continued interaction allowed
#   help           — question mark
#   not-allowed    — circle with slash
#   crosshair
#   cell           — table cell
#   all-scroll     — pan all directions
#   col-resize / row-resize
#   ew-resize / ns-resize
#   nesw-resize / nwse-resize
#   n-resize / s-resize / e-resize / w-resize / ne-resize / etc.
#   none           — hide cursor
#   alias / copy / no-drop
#   zoom-in / zoom-out
#   context-menu
#
# button:disabled { cursor: not-allowed; }
# .draggable      { cursor: grab; }
# .draggable:active { cursor: grabbing; }
# .resize-handle  { cursor: ew-resize; }
#
# # Custom cursor
# .map { cursor: url("/cursors/crosshair.png") 16 16, crosshair; }
# # Numbers = hotspot (where the click registers).
```

### user-select

```bash
# user-select: none | auto | text | all | contain;
# .label { user-select: none; }      # can't be selected
# pre code { user-select: all; }      # one click selects everything
```

### pointer-events

```bash
# pointer-events: auto | none | visiblePainted | bounding-box | ...
# .overlay { pointer-events: none; }    # clicks pass through
# .overlay button { pointer-events: auto; }  # but children stay clickable
#
# Useful for SVG hit detection and disabled interactive elements
# (though aria-disabled + tabindex=-1 is better for forms).
```

## Overflow and Scrolling

```bash
# overflow:    visible | hidden | scroll | auto | clip
# overflow-x:  ...
# overflow-y:  ...
#
#   visible — content can overflow (default)
#   hidden  — clipped, no scrollbar
#   scroll  — clipped, scrollbar ALWAYS shown
#   auto    — clipped, scrollbar ONLY when needed
#   clip    — like hidden but no scrolling at all (paint-only clip)
#
# .a { overflow: hidden; }
# .b { overflow-x: auto; overflow-y: hidden; }
# .c { overflow: clip; overflow-clip-margin: 1rem; }
```

### Scroll behavior

```bash
# scroll-behavior: auto | smooth;
# html { scroll-behavior: smooth; }   # smooth scroll for anchor jumps
# # Note: this is global — JS scrollTo() also smooths.
#
# # Respect reduced motion
# @media (prefers-reduced-motion: no-preference) {
#   html { scroll-behavior: smooth; }
# }
```

### Scroll snap

```bash
# Container
# .carousel {
#   scroll-snap-type: x mandatory;       # snap on x-axis, mandatory
#   scroll-snap-type: y proximity;       # less strict
#   overflow-x: auto;
#   display: flex;
# }
# # Items
# .slide {
#   scroll-snap-align: start | center | end;
#   scroll-snap-stop: normal | always;   # always = can't skip past
#   scroll-margin: 1rem;                 # offset from snap line
# }
#
# # On the container — pad for snap calculations
# .carousel { scroll-padding-left: 2rem; }
```

### Horizontal scroll pattern

```bash
# .scroller {
#   display: flex;
#   gap: 1rem;
#   overflow-x: auto;
#   scroll-snap-type: x mandatory;
#   scrollbar-width: none;             # Firefox: hide scrollbar
# }
# .scroller::-webkit-scrollbar { display: none; }   # WebKit: hide scrollbar
# .scroller > * {
#   flex: 0 0 80%;
#   scroll-snap-align: start;
# }
```

### overflow-anchor

```bash
# overflow-anchor: auto | none
# # Browser auto-scrolls to keep visible content stable when DOM changes.
# # Disable on infinite-scroll feeds:
# .feed { overflow-anchor: none; }
```

## Transforms

`transform` applies geometric operations without affecting layout.

```bash
# 2D transforms
# transform: translate(10px, 20px);     # x, y
# transform: translateX(50px);
# transform: translateY(-50%);
# transform: scale(1.5);                # uniform
# transform: scale(2, 0.5);             # x scale, y scale
# transform: scaleX(-1);                # mirror horizontally
# transform: rotate(45deg);
# transform: skew(10deg, 5deg);         # x angle, y angle
# transform: skewX(10deg);
# transform: matrix(a, b, c, d, e, f);  # full 2D matrix
#
# # Combined — applied right-to-left
# transform: translate(-50%, -50%) rotate(45deg) scale(1.2);
#
# # Centering trick
# .centered {
#   position: absolute;
#   top: 50%; left: 50%;
#   transform: translate(-50%, -50%);
# }
```

### Modern individual transforms

```bash
# Independent properties (animatable separately!)
# translate: 10px 20px;        # equivalent to transform: translate(...)
# rotate:    45deg;
# scale:     1.5;
#
# .a { translate: 0 -10px; rotate: 5deg; scale: 1.05; }
# .a:hover { rotate: 0deg; }      # animate rotate independently
```

### 3D transforms

```bash
# transform: translateZ(100px);
# transform: translate3d(x, y, z);
# transform: rotateX(45deg);    # tilt forward
# transform: rotateY(45deg);    # turn sideways
# transform: rotateZ(45deg);    # = rotate()
# transform: rotate3d(x, y, z, angle);
# transform: scale3d(x, y, z);
# transform: scaleZ(0.5);
# transform: matrix3d(...);     # 16-value 4x4 matrix
# transform: perspective(800px) rotateX(20deg);
```

### transform-origin

```bash
# transform-origin: x y [z];
# .a { transform-origin: top left; }       # rotate around TL corner
# .a { transform-origin: 0 0; }            # same as top left
# .a { transform-origin: 50% 50%; }        # default — center
# .a { transform-origin: center bottom 0; }
```

### perspective and 3D context

```bash
# Perspective creates depth — must be on the PARENT to affect children.
# .scene {
#   perspective: 800px;          # smaller = more dramatic
#   perspective-origin: 50% 50%; # vanishing point
# }
# .card {
#   transform-style: preserve-3d;   # children render in 3D space
#   transform: rotateY(20deg);
# }
# .back-face {
#   transform: rotateY(180deg);
#   backface-visibility: hidden;    # hide when flipped away
# }
```

### will-change

```bash
# Hint that a property will animate — promotes to GPU layer.
# .modal { will-change: transform, opacity; }
#
# # Use SPARINGLY:
# # - too many will-change layers exhausts GPU memory
# # - apply just before animation, remove after
# # - for static elements, the perf cost > benefit
#
# .item:hover { will-change: transform; }    # hint on hover
# .item { will-change: auto; }                # remove hint
```

## Transitions

Transitions animate property changes triggered by state (e.g. :hover).

```bash
# transition-property:        all | none | <property-list>
# transition-duration:        300ms | 0.3s
# transition-timing-function: ease | linear | ease-in | ease-out |
#                             ease-in-out | cubic-bezier(...) | steps(...)
# transition-delay:           0s | 200ms | -100ms (negative = mid-animation start)
#
# Shorthand: transition: property duration timing delay;
# .button {
#   transition: background-color 0.3s ease-out, transform 0.2s ease;
# }
# .button:hover {
#   background-color: navy;
#   transform: translateY(-2px);
# }
```

### Timing functions

```bash
# ease         — slow start, fast middle, slow end (default)
# linear       — constant speed
# ease-in      — slow start
# ease-out     — slow end
# ease-in-out  — slow both ends
# cubic-bezier(x1, y1, x2, y2)  — fully custom
# steps(n, jump-start | jump-end | jump-none | jump-both)
#
# .a { transition-timing-function: cubic-bezier(0.25, 0.1, 0.25, 1); }
# .b { transition-timing-function: steps(4, end); }   # 4 jumps
#
# Useful presets:
#   ease-out:      cubic-bezier(0, 0, 0.2, 1)        # decelerate
#   ease-in:       cubic-bezier(0.4, 0, 1, 1)        # accelerate
#   ease-in-out:   cubic-bezier(0.4, 0, 0.2, 1)
#   spring-bounce: cubic-bezier(0.5, -0.5, 0.5, 1.5) # overshoots
```

### Multiple transitions

```bash
# Comma-separated. Each transition can have its own duration/timing.
# .card {
#   transition:
#     transform   0.2s ease-out,
#     box-shadow  0.3s ease-out,
#     background  0.5s linear;
# }
```

### Performance — animate transform and opacity only

```bash
# These properties are CHEAP — handled on the GPU compositor:
#   transform, opacity, filter (mostly)
#
# These are EXPENSIVE — trigger layout/paint on every frame:
#   width, height, top/left/right/bottom, margin, padding,
#   border-width, font-size, line-height
#
# # SLOW
# .a { transition: width 0.3s; }
# .a:hover { width: 200px; }
#
# # FAST
# .a { transition: transform 0.3s; transform: scaleX(1); transform-origin: left; }
# .a:hover { transform: scaleX(1.5); }
```

### Transitioning to display:none

```bash
# # By default, display: none → block can't animate.
# # Modern: @starting-style + transition-behavior: allow-discrete
#
# .modal {
#   display: none;
#   opacity: 0;
#   transition: opacity 0.3s, display 0.3s allow-discrete;
# }
# .modal.open {
#   display: block;
#   opacity: 1;
# }
# @starting-style {
#   .modal.open { opacity: 0; }
# }
```

## Animations Basic

`@keyframes` define an animation timeline; the `animation` property runs it.

```bash
# Define
# @keyframes fadeIn {
#   from { opacity: 0; transform: translateY(-10px); }
#   to   { opacity: 1; transform: translateY(0); }
# }
#
# # With percentage stops
# @keyframes pulse {
#   0%   { transform: scale(1);   }
#   50%  { transform: scale(1.1); }
#   100% { transform: scale(1);   }
# }
#
# # Multi-property timeline
# @keyframes slideAndFade {
#   0%   { opacity: 0; transform: translateX(-100%); }
#   60%  { opacity: 1; transform: translateX(10%);   }
#   100% { opacity: 1; transform: translateX(0);     }
# }
```

### animation property

```bash
# animation: name duration timing-function delay iteration-count direction fill-mode play-state;
#
# .modal { animation: fadeIn 0.3s ease-out forwards; }
# .spinner { animation: spin 1s linear infinite; }
# .alert  { animation: pulse 1s ease-in-out infinite; }
#
# Multiple animations
# .multi {
#   animation:
#     fadeIn 0.3s ease-out,
#     slideUp 0.5s ease-in-out 0.1s;
# }
#
# Longhands
# animation-name:             fadeIn
# animation-duration:         0.3s
# animation-timing-function:  ease-out
# animation-delay:            100ms
# animation-iteration-count:  1 | infinite | 3
# animation-direction:        normal | reverse | alternate | alternate-reverse
# animation-fill-mode:        none | forwards | backwards | both
# animation-play-state:       running | paused
# animation-composition:      replace | add | accumulate
```

### animation-fill-mode

```bash
# none       — element returns to original style after animation
# forwards   — keeps the END (100%) state after animation
# backwards  — applies the START (0%) state during delay
# both       — backwards + forwards
#
# .a { animation: fadeIn 0.3s ease-out forwards; }
# # Without `forwards`, opacity reverts to 1 (or whatever) after fade.
```

### Pause / resume

```bash
# .spinner       { animation: spin 1s linear infinite; }
# .spinner.paused { animation-play-state: paused; }
#
# # Or via JS:
# // el.style.animationPlayState = "paused";
```

### Performance and respect for reduced-motion

```bash
# Always wrap animations
# @media (prefers-reduced-motion: reduce) {
#   *, ::before, ::after {
#     animation-duration: 0.01ms !important;
#     animation-iteration-count: 1 !important;
#     transition-duration: 0.01ms !important;
#     scroll-behavior: auto !important;
#   }
# }
```

## Filters and Backdrop

`filter` applies graphical effects to the element itself.

```bash
# filter: blur(5px);
# filter: brightness(1.2);            # 1 = identity, 0 = black, 2 = double
# filter: contrast(1.5);
# filter: drop-shadow(0 4px 8px rgb(0 0 0 / 0.2));   # like shadow but follows alpha
# filter: grayscale(100%);
# filter: hue-rotate(90deg);
# filter: invert(100%);
# filter: opacity(50%);
# filter: saturate(2);
# filter: sepia(60%);
# filter: url(#svg-filter-id);        # SVG filter
#
# # Multiple — applied in order
# .img { filter: grayscale(60%) blur(2px) brightness(0.8); }
```

### backdrop-filter

```bash
# Applies effect to what's BEHIND the element (frosted glass).
# .glass {
#   background: rgb(255 255 255 / 0.6);
#   backdrop-filter: blur(10px) saturate(1.2);
#   -webkit-backdrop-filter: blur(10px) saturate(1.2);
# }
# # Note: Safari needs the -webkit- prefix.
```

### Performance

```bash
# Filters force a new compositor layer — usually fine for static use.
# AVOID animating blur — it's expensive on every frame.
# Prefer animating opacity instead.
```

## Media Queries Basic

```bash
# @media [media-type] [and (feature)] {  ...  }
#
# Media types: all (default), screen, print, speech
#
# @media print { nav { display: none; } }
# @media screen and (min-width: 768px) { ... }
# @media (min-width: 600px) and (max-width: 1024px) { ... }
# @media (orientation: portrait) { ... }
# @media (hover: hover) and (pointer: fine) { ... }   # mouse, not touch
# @media (any-hover: none) { .tooltip { display: none; } }
```

### Width-based breakpoints

```bash
# Mobile-first canonical set
# /* base = mobile */
# @media (min-width: 640px)  { /* sm — large phone */ }
# @media (min-width: 768px)  { /* md — tablet */ }
# @media (min-width: 1024px) { /* lg — laptop */ }
# @media (min-width: 1280px) { /* xl — desktop */ }
# @media (min-width: 1536px) { /* 2xl — large desktop */ }
#
# # Modern range syntax
# @media (width >= 768px) { ... }
# @media (768px <= width < 1024px) { ... }
```

### Other features

```bash
# orientation:    portrait | landscape
# aspect-ratio:   16/9 | 1/1
# resolution:     2dppx | 192dpi
# hover:          none | hover
# pointer:        none | coarse | fine
# any-hover, any-pointer  (any input device)
# color:          number of bits per channel
# color-gamut:    srgb | p3 | rec2020
# update:         none | slow | fast (display refresh)
# scripting:      none | initial-only | enabled
# overflow-block: none | scroll | paged
# display-mode:   browser | standalone | fullscreen | minimal-ui   # PWA
```

### Preference media queries

```bash
# prefers-color-scheme: light | dark
# @media (prefers-color-scheme: dark) {
#   :root { --bg: #111; --fg: #eee; }
# }
#
# prefers-reduced-motion: no-preference | reduce
# @media (prefers-reduced-motion: reduce) {
#   * { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
# }
#
# prefers-contrast: no-preference | more | less | custom
# prefers-reduced-data: no-preference | reduce
# prefers-reduced-transparency: no-preference | reduce
# inverted-colors:   none | inverted
# forced-colors:     none | active                 # high contrast mode
# prefers-color-scheme: light | dark
```

## @supports Feature Queries

Test whether the browser supports a CSS feature.

```bash
# Property/value support
# @supports (display: grid) { ... }
# @supports (gap: 1rem) { ... }
# @supports (color: oklch(0 0 0)) { ... }
#
# # Negation
# @supports not (display: grid) {
#   .layout { /* float fallback */ }
# }
#
# # Combined
# @supports (display: grid) and (gap: 1rem) { ... }
# @supports (display: grid) or (display: flex) { ... }
#
# # Selector test (modern)
# @supports selector(:has(*)) {
#   .parent:has(img) { padding: 1rem; }
# }
# @supports selector(:focus-visible) {
#   .btn:focus-visible { outline: 2px solid; }
# }
#
# # Function test
# @supports (background: color-mix(in srgb, red, blue)) { ... }
```

### Progressive enhancement

```bash
# /* Baseline — works everywhere */
# .grid { display: flex; flex-wrap: wrap; gap: 1rem; }
#
# /* Enhanced — modern grid for capable browsers */
# @supports (display: grid) {
#   .grid {
#     display: grid;
#     grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
#     gap: 1rem;
#   }
# }
```

## @layer Cascade Layers

Declare cascade priority groups. Lower-listed = wins.

```bash
# Declare order (no rules — just establishes precedence)
# @layer reset, base, components, utilities;
#
# # Define rules in each
# @layer reset {
#   * { box-sizing: border-box; }
#   body { margin: 0; }
# }
#
# @layer base {
#   button { font: inherit; }
# }
#
# @layer components {
#   .card { padding: 1rem; border-radius: 8px; }
# }
#
# @layer utilities {
#   .text-center { text-align: center; }
#   .mt-0 { margin-top: 0; }
# }
#
# # Result: utilities WINS over components even if components has higher specificity.
```

### Anonymous and nested layers

```bash
# # Anonymous — single anonymous layer
# @layer { p { color: gray; } }
#
# # Nested
# @layer framework {
#   @layer reset { ... }
#   @layer base  { ... }
# }
# # Reference as: @layer framework.reset { ... }
#
# # Order: framework.reset wins over framework.base because order matters.
```

### Importing into a layer

```bash
# @import url("vendor.css") layer(vendor);
# @import url("tailwind.css") layer(utilities);
#
# # Now vendor styles can never beat your base, regardless of specificity.
```

### Unlayered styles

```bash
# Unlayered rules trump ALL layered rules.
# # Layered:
# @layer utilities { .red { color: red !important; } }
# # Unlayered:
# .red { color: blue; }
# # Unlayered wins → blue.
#
# Use ! important within layers to invert: !important within an EARLIER
# layer beats !important within a LATER layer (mirror of normal rules).
```

## CSS Nesting (2023+)

Native nesting matches Sass-like authoring without preprocessors.

```bash
# Basic
# .card {
#   padding: 1rem;
#
#   & h2 {
#     margin: 0;
#   }
#
#   & .child {
#     color: blue;
#   }
# }
#
# # & is the parent reference
# .button {
#   background: blue;
#
#   &:hover { background: navy; }
#   &.active { background: red; }
#   &[disabled] { opacity: 0.5; }
#
#   /* descendant of .button — & is implied */
#   .icon { margin-right: 0.5em; }
# }
```

### Nested at-rules

```bash
# .card {
#   padding: 1rem;
#
#   @media (min-width: 768px) {
#     padding: 2rem;
#   }
#
#   @supports (display: grid) {
#     display: grid;
#   }
#
#   @container (min-width: 400px) {
#     font-size: 1.25rem;
#   }
# }
```

### Compound parent

```bash
# .card {
#   /* equivalent to .card.large */
#   &.large { padding: 2rem; }
#
#   /* equivalent to .large .card */
#   .large & { background: yellow; }
# }
```

### Specificity reminder

```bash
# .a {
#   & .b { ... }    /* specificity = .a + .b = (0, 0, 2, 0) */
# }
# # Behaves like :is(.a) .b → uses :is() specificity rules.
```

## Logical Properties

Logical properties replace physical (top/right/bottom/left) with writing-direction-relative ones. Critical for RTL/CJK layouts.

```bash
# Physical          Logical (in horizontal LTR)
# margin-top        margin-block-start
# margin-bottom     margin-block-end
# margin-left       margin-inline-start
# margin-right      margin-inline-end
# margin (T R B L)  margin-block (start end), margin-inline (start end)
#
# padding-*         padding-block-*, padding-inline-*
# border-*          border-block-*, border-inline-*
#
# top               inset-block-start
# bottom            inset-block-end
# left              inset-inline-start
# right             inset-inline-end
# inset: 0          all four sides
#
# width             inline-size
# height            block-size
# max-width         max-inline-size
# min-height        min-block-size
```

### Why use them?

```bash
# In RTL (e.g. Arabic), inline-start = RIGHT.
# In vertical writing modes, block direction is horizontal.
#
# Use logical:
# .card { margin-inline: 1rem; padding-block: 0.5rem; }
# # Auto-flips for [dir=rtl] without extra CSS.
#
# Use physical only when truly direction-locked
# (e.g. dropdown anchored to top of viewport, scrollbar position).
```

### Mixing

```bash
# OK to mix — use logical for content, physical for direction-locked.
# .modal {
#   inset-block-start: 1rem;     # logical (writing-mode aware)
#   inset-inline-end:  1rem;
# }
# .scrollbar { right: 0; }       # physical (always right)
```

## CSS Functions

```bash
# calc(expr)              arithmetic across units
#   .a { width: calc(100% - 2rem); }
#   .b { font-size: calc(1rem + 0.5vw); }
#
# min(a, b, c)            smallest
#   .c { width: min(100%, 1200px); }
#
# max(a, b, c)            largest
#   .d { font-size: max(1rem, 16px); }
#
# clamp(min, val, max)    bounded preferred value
#   h1 { font-size: clamp(1.5rem, 2vw + 1rem, 3rem); }
#
# env(safe-area-inset-top, fallback)
#   header { padding-top: env(safe-area-inset-top, 0); }   # iOS notch
#
# attr(name [, type, fallback])
#   .badge::before { content: attr(data-count); }
#   /* Modern with type — Chrome 133+ */
#   .x { width: attr(data-width type(<length>), 0); }
#
# counter(name [, style])  / counters(name, sep [, style])
#   h2::before { content: counter(section) ". "; }
#
# var(--name [, fallback])
#   .a { color: var(--c, black); }
#
# color-mix(in space, c1, c2)
#   .a { color: color-mix(in oklch, var(--brand) 50%, white); }
#
# light-dark(light, dark)  — auto theme
#   .a { color: light-dark(black, white); }
#
# # Gradients
#   linear-gradient(...)
#   radial-gradient(...)
#   conic-gradient(...)
#   repeating-*-gradient(...)
#
# # Transform functions
#   translate() translateX() translateY() translate3d() translateZ()
#   rotate() rotateX() rotateY() rotateZ() rotate3d()
#   scale() scaleX() scaleY() scaleZ() scale3d()
#   skew() skewX() skewY()
#   matrix() matrix3d()
#   perspective()
#
# # Filter functions (used in `filter` and `backdrop-filter`)
#   blur(), brightness(), contrast(), drop-shadow(),
#   grayscale(), hue-rotate(), invert(), opacity(),
#   saturate(), sepia()
#
# # Image functions
#   url("img.png")
#   image-set(url("1x.png") 1x, url("2x.png") 2x)
#   cross-fade(...) — fade between images
#
# # Selectors as functions (in @supports / queries)
#   selector(:has(*))
```

## Common Errors

CSS fails silently — typos don't throw, the property is just dropped.

### "Unknown property name"

```bash
# DevTools shows a yellow warning when a property name is unknown.
# /* Bad: */
# .a { colour: red; }                  /* British spelling */
# /* DevTools: "Unknown property: colour" */
#
# /* Fixed: */
# .a { color: red; }
```

### "Invalid value" / "Expected ..."

```bash
# /* Bad: */
# .a { color: bleu; }                  /* not a CSS color */
# .b { width: 100;  }                  /* missing unit (px/rem/%) */
# .c { padding: 1.5; }                 /* same — needs unit */
# /* DevTools: "Invalid property value" */
#
# /* Fixed: */
# .a { color: blue; }
# .b { width: 100px; }
# .c { padding: 1.5rem; }
```

### "Could not load font" / "Failed to load resource"

```bash
# /* Likely cause: wrong path or CORS */
# @font-face {
#   font-family: "Inter";
#   src: url("/fonts/Inter.woff2") format("woff2");
# }
# /* If served cross-origin: server must include
#    Access-Control-Allow-Origin: * for fonts. */
```

### Silent-fail nature

```bash
# /* CSS does NOT abort on errors — it skips the bad declaration. */
# .a {
#   color: blue;
#   colour: red;          /* dropped silently */
#   background: white;
# }
# /* Result: color=blue, background=white, no throw. */
#
# /* WORST: missing semicolon cascades into next */
# .a {
#   color: red             /* missing ; */
#   background: white;     /* now read as part of color value — BOTH dropped */
# }
# /* Always end declarations with ; */
```

## Common Gotchas

### Margin collapse

```bash
# Adjacent BLOCK margins collapse — the larger wins, not added.
#
# /* Broken: expected 30px gap */
# .a { margin-bottom: 20px; }
# .b { margin-top:    10px; }   /* gap is 20px, not 30px */
#
# /* Fixed — use padding, gap, or break the BFC */
# .container { display: flow-root; }   /* establishes BFC */
# /* OR use grid/flex — children don't collapse margins */
# .container { display: grid; gap: 30px; }
```

### Specificity surprises

```bash
# /* Broken — :hover doesn't apply */
# #header .nav a { color: blue; }
# .nav a:hover  { color: red; }      /* loses to ID specificity (0,1,1,1) > (0,0,1,2) */
#
# /* Fixed — match or beat the original specificity */
# #header .nav a:hover { color: red; }
# /* OR: lower the original */
# .header .nav a       { color: blue; }
# .nav a:hover         { color: red; }
```

### !important wars

```bash
# /* Broken — escalating !important */
# .a            { color: red !important; }
# .a.b          { color: blue !important; }   /* still loses if more specific */
#
# /* Fixed — use @layer or restructure */
# @layer base { .a { color: red; } }
# @layer override { .a.b { color: blue; } }   /* later layer always wins */
```

### :is() specificity

```bash
# /* Surprise — :is() takes its highest argument */
# :is(.card, #header) p { ... }       /* specificity = (0, 1, 0, 1) — uses #header */
#
# /* Use :where() for zero specificity */
# :where(.card, #header) p { ... }    /* specificity = (0, 0, 0, 1) */
```

### inherit on non-inherited properties

```bash
# /* Broken — padding doesn't inherit */
# .parent { padding: 1rem; }
# .child  { padding: inherit; }    /* OK — explicit inheritance */
#
# /* Avoid: all: inherit
#    Inherits EVERYTHING — usually breaks layout */
# .child { all: inherit; }         /* nukes the child's defaults */
#
# /* Fixed — be specific */
# .child { padding: inherit; color: inherit; }
```

### width: 100% with padding

```bash
# /* Broken — content-box default + padding overflows */
# .a { width: 100%; padding: 1rem; border: 1px solid; }
# /* total width = 100% + 2rem + 2px → overflows parent */
#
# /* Fixed — use border-box globally */
# *, *::before, *::after { box-sizing: border-box; }
# .a { width: 100%; padding: 1rem; border: 1px solid; }   /* now fits */
```

### Transitioning to/from display: none

```bash
# /* Broken — abrupt show/hide, no fade */
# .modal { display: none; opacity: 0; transition: opacity 0.3s; }
# .modal.open { display: block; opacity: 1; }
#
# /* Fixed — modern allow-discrete + @starting-style */
# .modal {
#   display: none;
#   opacity: 0;
#   transition: opacity 0.3s, display 0.3s allow-discrete;
# }
# .modal.open { display: block; opacity: 1; }
# @starting-style { .modal.open { opacity: 0; } }
#
# /* Or fallback — use visibility instead */
# .modal { visibility: hidden; opacity: 0; transition: opacity 0.3s, visibility 0.3s; }
# .modal.open { visibility: visible; opacity: 1; }
```

### z-index without position

```bash
# /* Broken — z-index ignored */
# .above { z-index: 100; }                  /* position is static (default) */
#
# /* Fixed — z-index requires non-static position */
# .above { position: relative; z-index: 100; }
# /* Or position: absolute | fixed | sticky */
```

### Stacking context surprises

```bash
# /* Broken — z-index: 9999 won't escape ancestor's stacking context */
# .parent { transform: translateZ(0); }   /* creates stacking context */
# .child  { z-index: 9999; }              /* trapped inside parent */
#
# /* Other things that create a stacking context: */
# /*   - position: fixed | sticky
#      - opacity < 1
#      - transform: any
#      - filter: any
#      - will-change: transform | opacity
#      - isolation: isolate
#      - mix-blend-mode != normal
#      - contain: paint | layout
# */
#
# /* Fixed — move .child outside or remove the stacking context root */
```

### flex shrink default vs grid

```bash
# /* Broken — flex item overflows because it's allowed to shrink */
# .item { flex: 1; }                       /* flex-shrink: 1 by default */
#
# /* Fixed — for items with fixed minimum content */
# .item { flex: 1; min-width: 0; overflow: hidden; }
# /* Or */
# .item { flex: 0 1 auto; }
#
# /* Grid items DON'T shrink by default — different behavior */
```

## Performance

```bash
# 1. Avoid expensive selectors
#    Bad:   * .a .b .c { ... }     # universal + descendant chain
#    Bad:   [class$="-something"] { ... }   # attribute substring scan
#    Good:  .specific-class { ... }
#
# 2. Animate only transform and opacity
#    These don't trigger layout or paint — pure GPU compositing.
#
# 3. Use will-change sparingly
#    .modal { will-change: transform, opacity; }   /* before animation */
#    /* DON'T leave it on static elements */
#
# 4. Use content-visibility: auto for offscreen
#    .article { content-visibility: auto;
#               contain-intrinsic-size: auto 800px; }
#
# 5. Use contain: paint | layout | strict
#    .widget { contain: layout paint; }   /* isolate to widget */
#
# 6. Minimize style recalculation triggers
#    Avoid frequently-changing classList toggles on root.
#
# 7. Reduce CLS (Cumulative Layout Shift)
#    img, video { aspect-ratio: 16/9; width: 100%; height: auto; }
#    /* Reserve space — no jump when media loads. */
#
# 8. Use system fonts when possible (no FOUT/FOIT)
#    body { font-family: system-ui, sans-serif; }
#
# 9. Subset fonts via unicode-range
#    @font-face { ...; unicode-range: U+0020-007F; }   /* Latin only */
#
# 10. Lazy-load offscreen styles
#     <link rel="stylesheet" href="below-fold.css" media="print" onload="this.media='all'">
```

### Rendering pipeline

```bash
# Style → Layout → Paint → Composite
#
# Properties cost (cheaper later):
#   width, height, top/left      → Style + Layout + Paint + Composite
#   color, background-color      → Style + Paint + Composite
#   transform, opacity            → Style + Composite ONLY (GPU)
#
# Animate only the last category for 60fps.
```

## Accessibility

```bash
# 1. Visible focus indicators — never remove without replacement
#    button:focus-visible { outline: 2px solid currentColor; outline-offset: 2px; }
#
# 2. Respect prefers-reduced-motion
#    @media (prefers-reduced-motion: reduce) { ... }
#
# 3. Sufficient color contrast (WCAG 2.1 AA)
#    Text:        4.5:1 normal, 3:1 large (≥18pt or 14pt bold)
#    UI/icons:    3:1
#    Use tools — Chrome DevTools shows contrast ratio in color picker.
#
# 4. Don't rely on color alone — pair with icon/text
#    .error::before { content: "⚠ "; }
#    .error         { color: red; }
#
# 5. Sufficient touch targets (≥24×24 CSS px, recommend 44×44)
#    button { min-block-size: 44px; min-inline-size: 44px; }
#
# 6. Respect prefers-color-scheme
#    @media (prefers-color-scheme: dark) { :root { --bg: #111; ... } }
#
# 7. Respect prefers-contrast
#    @media (prefers-contrast: more) { :root { --border: #000; ... } }
#
# 8. Forced-colors mode (Windows High Contrast)
#    @media (forced-colors: active) {
#      button { border: 1px solid ButtonText; }   /* system colors */
#    }
#
# 9. Don't hide content from AT users with display: none if it's relevant
#    Use .sr-only pattern instead.
#
# 10. Support keyboard navigation visually — :focus-visible on all
#     interactive elements (button, a, input, [tabindex]).
```

## Print Styles

```bash
# @media print {
#   body { font-size: 11pt; color: black; background: white; }
#   nav, footer, .ad { display: none; }
#   a::after { content: " (" attr(href) ")"; font-size: 0.8em; color: #555; }
#   img { max-width: 100% !important; }
#   h2, h3 { page-break-after: avoid; }
#   pre, blockquote { page-break-inside: avoid; }
# }
#
# @page {
#   size: A4 portrait;       /* or letter, 8.5in 11in, etc. */
#   margin: 1in;
# }
# @page :first { margin-top: 2in; }
# @page :left  { margin-left: 1.5in; margin-right: 0.75in; }
# @page :right { margin-left: 0.75in; margin-right: 1.5in; }
#
# /* Page break controls */
# .new-page    { page-break-before: always; break-before: page; }
# .keep        { page-break-inside: avoid;  break-inside: avoid; }
# .no-break    { page-break-after:  avoid;  break-after: avoid; }
#
# /* Orphan/widow control — minimum lines before/after a break */
# p { orphans: 3; widows: 3; }
#
# /* Force background colors in print (Chrome) */
# body { print-color-adjust: exact; -webkit-print-color-adjust: exact; }
```

## CSS Architecture

### BEM (Block, Element, Modifier)

```bash
# .block         { ... }     # standalone component
# .block__elem   { ... }     # descendant of block
# .block--mod    { ... }     # modifier of block
# .block__elem--mod { ... }
#
# /* Example */
# .card           { ... }
# .card__title    { ... }
# .card__body     { ... }
# .card--featured { ... }
# .card--compact  { ... }
```

### ITCSS (Inverted Triangle CSS)

```bash
# Layer order from generic → specific:
#   1. Settings  — variables, config
#   2. Tools     — mixins, functions
#   3. Generic   — reset, normalize
#   4. Elements  — bare HTML element styles
#   5. Objects   — design patterns (e.g. .o-grid)
#   6. Components — UI components (.c-card)
#   7. Utilities  — helpers (.u-text-center)
#
# Modern equivalent: @layer settings, generic, elements, objects, components, utilities;
```

### OOCSS, SMACSS, atomic CSS

```bash
# OOCSS  — separate structure from skin (.btn .btn--primary)
# SMACSS — Base, Layout, Module, State, Theme categories
# Atomic — single-purpose utility classes (.text-center, .pa-3)
# Tailwind — utility-first by default, with @apply or @layer for components
```

### CSS-in-JS / CSS Modules

```bash
# CSS Modules (build-time scoped class names)
# /* button.module.css */
# .btn { color: blue; }
# /* JS: import s from './button.module.css'; <button class={s.btn}> */
# /* Output class: button_btn_3xN9k -- guaranteed unique */
#
# CSS-in-JS (runtime, e.g. styled-components, emotion)
# /* JS: const Btn = styled.button`color: blue;`; */
# /* Tradeoff: runtime overhead, but co-located styles. */
```

## Modern Tools

```bash
# PostCSS         — JS-based CSS transformations (autoprefixer, nesting,
#                   custom properties polyfill, modern features pipeline)
# Sass / Less     — CSS preprocessors with variables, nesting, mixins,
#                   functions; compile to CSS at build time.
# Stylus          — terser preprocessor (indent-based)
# Tailwind CSS    — utility-first framework, tree-shakes unused
# UnoCSS          — atomic CSS engine, fast
# Lightning CSS   — Rust-based CSS bundler/minifier
# CSS Modules     — scoped class names via build tool (Vite, webpack)
# CSS-in-JS       — styled-components, emotion, vanilla-extract
#
# Variables: CSS custom props (runtime, dynamic) vs Sass $vars (compile-time).
# Use CSS custom props for theming, Sass for static config or build math.
```

## Idioms

### Smart defaults reset

```bash
# *, *::before, *::after { box-sizing: border-box; }
# html { -webkit-text-size-adjust: 100%; tab-size: 4; }
# body {
#   margin: 0;
#   font-family: system-ui, sans-serif;
#   line-height: 1.5;
#   -webkit-font-smoothing: antialiased;
# }
# img, picture, video, canvas, svg { display: block; max-width: 100%; }
# input, button, textarea, select { font: inherit; }
# button { cursor: pointer; }
# h1, h2, h3, h4, h5, h6, p { overflow-wrap: break-word; }
# h1, h2, h3, h4, h5, h6 { text-wrap: balance; }
# p { text-wrap: pretty; }
```

### Centering

```bash
# Grid (one-shot)
# .center { display: grid; place-items: center; min-height: 100vh; }
#
# Flex
# .center { display: flex; align-items: center; justify-content: center; }
#
# Absolute (legacy but reliable)
# .parent { position: relative; }
# .child  { position: absolute; top: 50%; left: 50%;
#           transform: translate(-50%, -50%); }
#
# Margin auto (block-only)
# .container { max-width: 60rem; margin-inline: auto; }
```

### Aspect ratio

```bash
# .video { aspect-ratio: 16/9; }
# img    { aspect-ratio: 1/1; object-fit: cover; }
# /* Reserve space, prevent CLS — modern intrinsic ratio */
```

### Fluid typography with clamp()

```bash
# h1 { font-size: clamp(1.5rem, 2vw + 1rem, 3rem); }
# /* Scales with viewport between 1.5rem and 3rem */
#
# /* Pure-CSS fluid typography scale */
# :root {
#   --fs-100: clamp(0.875rem, 0.5vw + 0.7rem, 1rem);
#   --fs-200: clamp(1rem, 0.5vw + 0.85rem, 1.125rem);
#   --fs-300: clamp(1.125rem, 0.6vw + 0.95rem, 1.25rem);
#   --fs-500: clamp(1.5rem, 1.2vw + 1rem, 2rem);
#   --fs-700: clamp(2rem, 2.5vw + 1rem, 3rem);
#   --fs-900: clamp(3rem, 4vw + 1.5rem, 5rem);
# }
```

### Logical properties for i18n

```bash
# Use logical (margin-inline, padding-block) anywhere direction-agnostic
# is OK. The same stylesheet now flips for [dir=rtl] automatically.
```

### @layer for safe specificity

```bash
# @layer reset, base, theme, components, utilities;
# /* utilities ALWAYS wins, no !important needed */
# @layer utilities { .text-center { text-align: center; } }
```

### Container query container pattern

```bash
# .card-container { container-type: inline-size; }
# @container (min-width: 30rem) {
#   .card { padding: 2rem; font-size: 1.25rem; }
# }
# /* See css-layout for full container query coverage */
```

## Tips

- Use `*, *::before, *::after { box-sizing: border-box; }` globally — makes layout math sane.
- Prefer `rem` for font-size and global spacing; `em` for component-relative spacing that should scale with text.
- Use unitless `line-height` (e.g. `1.5`) — multiplies by font-size, scales correctly with nested text.
- Custom properties for theming — they cascade, can be overridden per-component, and update at runtime.
- Always include `prefers-reduced-motion` opt-out for animations.
- `@layer reset, base, components, utilities;` declares a predictable cascade — last wins, no !important needed.
- `:where()` zeros out specificity — perfect for "default" rules that any later style can override.
- `:has()` enables true parent selection — no JS needed for many UI states (`form:has(input:invalid)`).
- Use `oklch()` for color palettes — perceptually uniform, predictable lightness/chroma.
- `clamp()` with viewport-relative middle creates fluid type without media queries.
- Animate only `transform` and `opacity` for 60fps. Everything else triggers layout/paint.
- `aspect-ratio` on images and embeds prevents CLS and reserves space before media loads.
- Mobile-first (`min-width`) media queries produce cleaner CSS than desktop-first.
- `text-wrap: balance` for headings, `text-wrap: pretty` for paragraphs — modern typography.
- Use `currentColor` for SVG icons inside text — they inherit the text color automatically.
- `accent-color: <color>` styles native form controls (checkboxes, radios) with one line.
- `color-scheme: light dark` lets the browser style native scrollbars, form controls per theme.
- `:focus-visible` shows focus ring only when needed (kbd) — gives mouse users a clean look.
- The "stylesheet load order" mantra: reset → vendor → base → components → utilities → critical inline.
- Never `outline: none` without replacement — keyboard users lose all signal.

## See Also

- html
- html-forms
- css-layout
- javascript
- typescript
- polyglot
- regex

## References

- [MDN CSS Reference](https://developer.mozilla.org/en-US/docs/Web/CSS)
- [MDN CSS Selectors](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_selectors)
- [MDN CSS Cascade and Inheritance](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_cascade)
- [MDN CSS Custom Properties](https://developer.mozilla.org/en-US/docs/Web/CSS/Using_CSS_custom_properties)
- [MDN CSS Values and Units](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Values_and_Units)
- [MDN CSS Color](https://developer.mozilla.org/en-US/docs/Web/CSS/color_value)
- [MDN CSS Transitions](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_transitions/Using_CSS_transitions)
- [MDN CSS Animations](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_animations/Using_CSS_animations)
- [MDN Media Queries](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_media_queries/Using_media_queries)
- [MDN @supports](https://developer.mozilla.org/en-US/docs/Web/CSS/@supports)
- [MDN @layer](https://developer.mozilla.org/en-US/docs/Web/CSS/@layer)
- [MDN @property](https://developer.mozilla.org/en-US/docs/Web/CSS/@property)
- [MDN CSS Nesting](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_nesting)
- [MDN Logical Properties](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_logical_properties_and_values)
- [W3C CSS Specifications](https://www.w3.org/TR/CSS/)
- [CSS Working Group: Cascade Level 5](https://www.w3.org/TR/css-cascade-5/)
- [web.dev — Learn CSS](https://web.dev/learn/css/)
- [CSS-Tricks](https://css-tricks.com/)
- [Can I Use — Browser Compatibility](https://caniuse.com/)
- [CSS Specificity Calculator](https://specificity.keegan.st/)
