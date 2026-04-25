# CSS Layout (Box Model, Flexbox, Grid, Positioning)

Comprehensive reference for laying out web UI -- box model, flexbox, grid, positioning, container queries, intrinsic sizing, responsive patterns, and modern CSS layout primitives.

## The Box Model

Every element is a rectangular box with four parts: content, padding, border, margin. Outline draws on top and does NOT participate in the box.

```bash
# Anatomy of a box (outer to inner)
# margin    - transparent space outside the border (collapses with siblings)
# border    - the line drawn around padding+content
# padding   - transparent space inside the border (does NOT collapse)
# content   - the box's actual content area (text, child boxes)

.box {
  margin: 10px;          # outermost, transparent
  border: 2px solid red; # ring around padding+content
  padding: 8px;          # inside ring, between border and content
  width: 200px;          # default = content-box: applies to CONTENT only
  height: 100px;
  outline: 3px dashed blue;  # drawn outside border, NOT in box, doesn't shift layout
}
```

### content-box default vs border-box modern

```bash
# Default (W3C): width = content area only
# Total visual width = width + padding-left+right + border-left+right
.box { box-sizing: content-box; width: 200px; padding: 20px; border: 5px solid; }
# Actual rendered width = 200 + 40 + 10 = 250px  -> SURPRISING

# Modern: width = content + padding + border
# Total visual width = width  (predictable)
.box { box-sizing: border-box; width: 200px; padding: 20px; border: 5px solid; }
# Actual rendered width = 200px  -> what you typed is what you get
```

### The canonical universal reset

```bash
*, *::before, *::after {
  box-sizing: border-box;
}
# Apply to EVERYTHING including pseudo-elements. Universal sanity.
# Inheritance approach (lets components override):
html { box-sizing: border-box; }
*, *::before, *::after { box-sizing: inherit; }
```

### Outline does NOT participate in the box

```bash
# outline draws OUTSIDE border, takes NO layout space
# Use for focus rings without shifting siblings
button:focus-visible { outline: 2px solid blue; outline-offset: 2px; }

# border DOES take space; toggling on focus causes layout shift
button:focus { border: 2px solid blue; }  # BAD - shifts layout 2px
```

## Width and Height

Default sizing is `auto` which means "use intrinsic content size" or "fill container" depending on context.

### Default behavior

```bash
# Block elements: width: auto = fill parent (content-box)
div { /* takes full parent inline width */ }

# Inline elements: width is ignored - use display: inline-block to size
span { width: 200px; }   # IGNORED on inline
span { display: inline-block; width: 200px; }   # works

# height: auto = "as tall as content" - the default
div { /* grows to content height */ }
```

### Min and max constraints

```bash
.card {
  width: 50%;
  min-width: 320px;       # don't shrink below 320px
  max-width: 720px;       # don't grow beyond 720px
}

# Common pattern: responsive container
.container {
  width: 100%;
  max-width: 1280px;
  margin-inline: auto;    # center horizontally
  padding-inline: 1rem;
}
```

### Intrinsic sizing keywords

```bash
.box { width: min-content; }     # smallest the content can be (longest unbreakable word)
.box { width: max-content; }     # widest content needs without wrapping
.box { width: fit-content; }     # min(max-content, available) - shrinks to content but caps
.box { width: fit-content(20rem); }  # like fit-content but capped at 20rem
.box { width: auto; }            # context-dependent default
```

### aspect-ratio (modern, replaces padding-bottom hack)

```bash
# OLD hack:
.video { position: relative; padding-bottom: 56.25%; height: 0; }
.video > iframe { position: absolute; inset: 0; width: 100%; height: 100%; }

# MODERN:
.video { aspect-ratio: 16 / 9; width: 100%; }
.video > iframe { width: 100%; height: 100%; }

# Common ratios
.square    { aspect-ratio: 1; }           # or 1/1
.widescreen{ aspect-ratio: 16/9; }
.cinema    { aspect-ratio: 21/9; }
.portrait  { aspect-ratio: 3/4; }
```

## Margin

External space. Auto for centering. Negative allowed. Adjacent block siblings collapse.

```bash
.x { margin: 10px; }                  # all 4 sides
.x { margin: 10px 20px; }             # vertical | horizontal
.x { margin: 10px 20px 30px; }        # top | horizontal | bottom
.x { margin: 10px 20px 30px 40px; }   # top | right | bottom | left (clockwise)

# Auto on horizontal margins centers a block of fixed width
.center-block { width: 600px; margin: 0 auto; }
.center-block { width: 600px; margin-inline: auto; }   # logical equivalent

# Negative margins (valid, useful for overlap)
.bleed { margin-left: -1rem; margin-right: -1rem; }
```

### Margin collapse between adjacent block siblings

```bash
# Between siblings: only the LARGER of two adjacent vertical margins applies
.a { margin-bottom: 30px; }
.b { margin-top: 20px; }
# Resulting gap = 30px (NOT 50px). This is "margin collapse".

# Negative collapse: negative + positive = sum
.a { margin-bottom: -10px; }
.b { margin-top: 20px; }
# Resulting gap = 10px

# Two negatives: smaller (more negative) wins
```

### Margin collapse with parent (the BFC fix)

```bash
# A child's top margin escapes through the parent UNLESS parent forms a BFC
<div class="parent">
  <div class="child">  margin-top: 50px  </div>
</div>

# WITHOUT BFC: child's 50px margin "leaks" - parent appears pushed down 50px
.parent { /* margin-top of child escapes here */ }

# Fix 1 (modern, no side effects):
.parent { display: flow-root; }

# Fix 2 (legacy):
.parent { overflow: hidden; }   # has clipping side effect
.parent { padding-top: 1px; }   # fragile, fights designer
.parent { border-top: 1px solid transparent; }   # 1px shift
```

### Logical margins (i18n-friendly)

```bash
# margin-block-start / margin-block-end   replace top / bottom (in horizontal-tb)
# margin-inline-start / margin-inline-end replace left / right
.box {
  margin-block: 1rem;        # top + bottom
  margin-inline: auto;       # left + right (centers)
  margin-block-start: 2rem;
  margin-inline-end: 0.5rem;
}
# In RTL languages, margin-inline-start automatically flips to right side
```

## Padding

Internal space. Padding does NOT collapse. Affects content-box width unless box-sizing: border-box.

```bash
.x { padding: 10px; }
.x { padding: 10px 20px; }              # v | h
.x { padding: 10px 20px 30px 40px; }    # t | r | b | l

# Logical
.x { padding-block: 1rem; }       # top + bottom
.x { padding-inline: 2rem; }      # left + right
.x { padding-inline-start: 0; }
```

```bash
# Padding does NOT collapse - sibling paddings always stack
.a { padding-bottom: 20px; }
.b { padding-top: 30px; }
# Total visible gap = 50px (paddings don't collapse)
```

```bash
# Padding shifts content-box layout in old default
.x { box-sizing: content-box; width: 200px; padding: 20px; }
# Element renders 240px wide - DON'T DO THIS

.x { box-sizing: border-box; width: 200px; padding: 20px; }
# Element renders 200px - PREDICTABLE
```

## Display Modes

`display` controls outer (how box participates in flow) and inner (how children lay out) display.

```bash
display: block;        # full inline width, breaks line; default for div, p, h1, section
display: inline;       # flows with text; height/width/vertical margin IGNORED; default for span, a, em
display: inline-block; # inline flow but accepts width/height/margin
display: flex;         # block-level flex container
display: inline-flex;  # inline-level flex container
display: grid;         # block-level grid container
display: inline-grid;  # inline-level grid container
display: table;        # behaves like a <table>
display: table-row;    # like <tr>
display: table-cell;   # like <td>; useful for vertical centering legacy
display: list-item;    # generates marker (default for li)
display: contents;     # element disappears as a box, children participate in parent layout
display: flow-root;    # block + creates new BFC (no margin collapse, no float bleed)
display: ruby;         # CJK ruby annotations
display: none;         # removed from layout AND accessibility tree
```

### display: none vs visibility: hidden vs opacity: 0

```bash
.a { display: none; }
# - removed from layout (no space taken)
# - removed from accessibility tree (screen readers skip)
# - cannot transition (instant)

.b { visibility: hidden; }
# - keeps space in layout (hole)
# - removed from accessibility tree
# - can be transitioned WITH visibility transition trick + delay

.c { opacity: 0; }
# - keeps space in layout
# - STILL accessible to screen readers (use aria-hidden too)
# - STILL receives clicks (use pointer-events: none too)
# - can transition smoothly
```

## Block Formatting Context (BFC)

A BFC isolates layout: floats inside don't escape, margins inside don't collapse with parent.

```bash
# Triggers a new BFC (any of):
display: flow-root;          # MODERN - no side effects, recommended
display: flex;               # flex containers form BFC
display: grid;               # grid containers form BFC
display: inline-block;
display: table-cell;
overflow: hidden;            # legacy - has clipping side effect
overflow: auto;              # legacy - may show scrollbars
overflow: scroll;
position: absolute;
position: fixed;
contain: layout;             # modern containment
column-count: 1;
```

### Modern: display: flow-root

```bash
# THE clean way to contain floats and stop margin-collapse-with-parent.
.parent { display: flow-root; }

# - children's vertical margins are contained
# - floats don't escape
# - no clipping (unlike overflow: hidden)
# - no scrollbars (unlike overflow: auto)
# - no positioning side effects (unlike absolute)
```

## Flexbox - Container

One-dimensional layout: a row (default) or column.

```bash
.flex {
  display: flex;
  flex-direction: row;          # row (default) | row-reverse | column | column-reverse
  flex-wrap: nowrap;            # nowrap (default) | wrap | wrap-reverse
  flex-flow: row wrap;          # shorthand: direction + wrap
  gap: 1rem;                    # row-gap + column-gap
  row-gap: 1rem;
  column-gap: 0.5rem;
}
```

### Axes

```bash
# main axis    = direction of flex-direction
# cross axis   = perpendicular to main axis
# justify-*    = main axis alignment
# align-*      = cross axis alignment

# flex-direction: row     -> main = horizontal, cross = vertical
# flex-direction: column  -> main = vertical,   cross = horizontal
```

## Flexbox - Justify and Align

```bash
.container {
  display: flex;
  /* main axis */
  justify-content: flex-start;     # default
  justify-content: flex-end;
  justify-content: center;
  justify-content: space-between;  # first/last touch edges, equal gaps between
  justify-content: space-around;   # half-gap on edges
  justify-content: space-evenly;   # equal gaps everywhere

  /* cross axis - single line */
  align-items: stretch;            # default - children fill cross axis
  align-items: flex-start;
  align-items: flex-end;
  align-items: center;
  align-items: baseline;           # text baselines align

  /* cross axis - multi line (only with flex-wrap: wrap) */
  align-content: stretch;
  align-content: flex-start;
  align-content: center;
  align-content: space-between;
  align-content: space-around;

  /* shorthands */
  place-items: center;             # align-items + justify-items (justify-items has no effect in flex)
  place-content: center;           # align-content + justify-content
}

/* per-item override */
.item-special { align-self: flex-end; }
```

## Flexbox - Items

Individual flex children control how they grow, shrink, and what their starting size is.

```bash
.item {
  flex-grow: 0;       # default - don't take extra space
  flex-shrink: 1;     # default - shrink if not enough room
  flex-basis: auto;   # default - use the item's content size or width

  /* shorthand */
  flex: 0 1 auto;     # default - DON'T grow, shrink if needed, basis from width/content
  flex: 1 1 auto;     # GROW to fill, shrink, basis auto
  flex: 1 0 auto;     # GROW, NEVER shrink below basis
  flex: 1;            # = 1 1 0%   (grow, shrink, basis 0) - common "split equally" pattern
  flex: auto;         # = 1 1 auto (grow, shrink, basis from content)
  flex: none;         # = 0 0 auto (rigid)
}
```

### flex: 1 vs flex: auto - the basis trap

```bash
# flex: 1   = 1 1 0%      basis 0 -> all items get equal width regardless of content
# flex: auto= 1 1 auto    basis auto -> items size by content first, then share leftover

# When you want equal-width columns:
.col { flex: 1; }      # equal columns even if content varies

# When you want content-aware sizing:
.col { flex: auto; }   # bigger content -> bigger column
```

### The min-width: 0 trap

```bash
# Flex items have an implicit min-width: auto = "shrink to min-content"
# A child with a long unbreakable word REFUSES to shrink below that word.

# Visible symptom: nav overflows viewport, text won't ellipsize
.nav { display: flex; }
.nav > .item { flex: 1; overflow: hidden; text-overflow: ellipsis; }  # STILL overflows

# FIX:
.nav > .item { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; }
# Now ellipsis works.
```

### order

```bash
# Flexbox order is VISUAL ONLY - DOM order persists for screen readers and tab focus
.a { order: 2; }
.b { order: 1; }   # renders first
.c { order: 3; }
# Default order = 0; negative values move toward start
# WARNING: changes visual order but NOT focus order - accessibility hazard
```

## Flexbox - Common Patterns

### Center perfectly

```bash
.parent { display: flex; justify-content: center; align-items: center; min-height: 100vh; }
# OR
.parent { display: flex; place-items: center; place-content: center; min-height: 100vh; }
# OR
.parent { display: grid; place-items: center; min-height: 100vh; }
```

### Horizontal nav with logo + items + actions

```bash
.nav { display: flex; align-items: center; gap: 1rem; }
.nav .logo { /* default flex: 0 1 auto - sized to content */ }
.nav .links { display: flex; gap: 1rem; margin-inline-start: auto; }   # push to right
.nav .actions { display: flex; gap: 0.5rem; }
```

### Sticky footer

```bash
body { min-height: 100vh; min-height: 100dvh; display: flex; flex-direction: column; }
main { flex: 1; }   # takes all remaining space, pushing footer down
footer { /* sits at bottom even on short pages */ }
```

### Equal-height columns

```bash
.row { display: flex; }
.col { flex: 1; }
# All children stretch to row's tallest by default (align-items: stretch is default)
```

### Main + sidebar with sidebar fixed width

```bash
.layout { display: flex; gap: 2rem; }
.sidebar { flex: 0 0 280px; }   # fixed 280px wide, never grows or shrinks
.main { flex: 1; min-width: 0; }   # min-width: 0 so children can shrink
```

### Media object (avatar + body)

```bash
.media { display: flex; gap: 1rem; }
.media .avatar { flex: 0 0 auto; }            # don't grow, don't shrink
.media .body { flex: 1; min-width: 0; }
```

## Flexbox - Gotchas

### Items refuse to shrink below content

```bash
# BROKEN
.box { display: flex; }
.box > * { flex: 1; }
# Long URL inside child overflows container

# FIXED
.box > * { flex: 1; min-width: 0; }
```

### flex: 1 vs flex: 1 1 auto

```bash
# BROKEN - expecting equal columns
.row { display: flex; }
.col { flex: 1 1 auto; }   # basis: auto -> bigger content gets bigger column

# FIXED
.col { flex: 1; }          # = flex: 1 1 0% -> equal columns
```

### Gap not supported in old Safari

```bash
# Modern (works everywhere modern):
.row { display: flex; gap: 1rem; }

# Polyfill for very old browsers (Safari < 14.1):
.row { display: flex; margin: -0.5rem; }
.row > * { margin: 0.5rem; }
# Don't bother unless you need iOS 14 - just use gap.
```

### margin: auto inside flex - the universal pusher

```bash
.nav .links li:last-child { margin-left: auto; }
# pushes the last link (and everything after) to the far right
# margin: auto eats all leftover free space on that axis
```

### flex-shrink: 1 default surprise

```bash
# By default, flex-shrink: 1 - items SHRINK below their declared width if needed
.box { width: 500px; }   # may render < 500px in a tight flex container

# Prevent shrinking:
.box { width: 500px; flex-shrink: 0; }
# OR
.box { flex: 0 0 500px; }
```

## Grid - Container

Two-dimensional: rows AND columns simultaneously.

```bash
.grid {
  display: grid;

  /* explicit tracks */
  grid-template-columns: 200px 1fr 1fr;          # 3 columns
  grid-template-rows: 50px auto 50px;            # 3 rows
  grid-template-areas:
    "header header header"
    "sidebar main main"
    "footer footer footer";

  /* shorthand */
  grid-template:
    "header header header" 50px
    "sidebar main main"    auto
    "footer footer footer" 50px
    / 200px 1fr 1fr;

  /* gaps */
  gap: 1rem;
  row-gap: 1rem;
  column-gap: 0.5rem;
}
```

### Track sizing keywords

```bash
grid-template-columns: 100px;            # absolute
grid-template-columns: 1fr;              # fraction of remaining space
grid-template-columns: auto;             # max-content for the items in the track
grid-template-columns: min-content;      # smallest content can shrink to
grid-template-columns: max-content;      # widest content needs
grid-template-columns: minmax(100px, 1fr);  # min 100px, max 1fr
grid-template-columns: fit-content(300px);  # like auto but capped at 300px
grid-template-columns: repeat(3, 1fr);   # 3 equal columns
grid-template-columns: repeat(3, minmax(0, 1fr));  # safe equal columns (handles overflow)
```

## Grid - Tracks and fr Unit

The fraction unit `fr` distributes leftover space after fixed/auto tracks resolve.

```bash
.grid { grid-template-columns: 1fr 1fr 1fr; }
# 3 equal columns

.grid { grid-template-columns: 1fr 2fr; }
# 1:2 ratio - second column is twice as wide

.grid { grid-template-columns: 200px 1fr 1fr; }
# 200px sidebar, two equal-width main columns

.grid { grid-template-columns: minmax(200px, 1fr) 3fr; }
# left col at least 200px, max 1fr; right col 3fr
```

### Responsive auto-fit/auto-fill

```bash
# Cards that fit as many as possible per row, min 250px each
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
}

# auto-fit  vs  auto-fill
# auto-fit  -> empty tracks COLLAPSE; existing cards stretch to fill row
# auto-fill -> empty tracks RESERVED; cards stay at min size on wide screens
```

## Grid - Item Placement

```bash
.item {
  grid-column-start: 1;
  grid-column-end: 3;
  grid-row-start: 2;
  grid-row-end: 4;

  /* shorthand: start / end */
  grid-column: 1 / 3;
  grid-row: 2 / 4;

  /* span N tracks */
  grid-column: 1 / span 2;        # start at line 1, span 2 columns
  grid-column: span 2;             # span 2 from auto position

  /* shorthand: row-start / column-start / row-end / column-end */
  grid-area: 2 / 1 / 4 / 3;

  /* named area */
  grid-area: header;
}
```

### Named lines

```bash
.grid {
  grid-template-columns: [main-start] 1fr 1fr 1fr [main-end aside-start] 200px [aside-end];
}
.featured { grid-column: main-start / main-end; }
.sidebar { grid-column: aside-start / aside-end; }
```

### Named areas

```bash
.layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  grid-template-rows: auto 1fr auto;
  grid-template-areas:
    "header header"
    "nav    main"
    "footer footer";
}
.layout > header { grid-area: header; }
.layout > nav    { grid-area: nav; }
.layout > main   { grid-area: main; }
.layout > footer { grid-area: footer; }
# A . (period) leaves a cell empty
```

## Grid - Implicit Tracks

Items placed beyond the explicit grid auto-create tracks. Control their sizing.

```bash
.grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-auto-rows: 200px;             # any auto-created row gets 200px
  grid-auto-columns: 100px;          # any auto-created column gets 100px
  grid-auto-flow: row;               # default - fill row by row
  grid-auto-flow: column;            # fill column by column
  grid-auto-flow: row dense;         # backfill earlier holes
}
```

### Dense packing

```bash
# Without dense: holes can appear when an item spans more tracks than available in current row
# With dense: later items backfill earlier holes
.gallery { grid-auto-flow: row dense; }
# WARNING: dense reorders items visually - DOM order persists for screen readers
```

## Grid - Justify and Align

```bash
.grid {
  /* container distributes tracks within itself (when total tracks < container) */
  justify-content: start | end | center | stretch | space-between | space-around | space-evenly;
  align-content: start | end | center | stretch | space-between | space-around | space-evenly;
  place-content: center;            # both

  /* container aligns items within their cells */
  justify-items: start | end | center | stretch;       # default stretch (inline axis)
  align-items: start | end | center | stretch;         # default stretch (block axis)
  place-items: center;                                 # both
}

.item {
  justify-self: start | end | center | stretch;        # override per item (inline axis)
  align-self: start | end | center | stretch;          # override per item (block axis)
  place-self: center;                                   # both
}
```

## Grid - Subgrid

Nested grid that aligns its tracks with the parent grid.

```bash
.outer {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}
.outer > .card {
  display: grid;
  grid-template-rows: subgrid;        # rows align with outer grid rows
  grid-template-columns: subgrid;     # columns align with outer columns
  grid-row: span 3;                   # span 3 of outer's rows
}
# Card's children align across other cards' children - useful for "row of titles, row of images"
```

### Per-axis subgrid

```bash
# Subgrid only one axis - rows from parent, columns local
.card {
  display: grid;
  grid-template-rows: subgrid;
  grid-template-columns: 1fr 2fr;     # local columns
  grid-row: span 3;
}
```

### Browser support note

```bash
# subgrid - Chrome 117+, Safari 16+, Firefox 71+ (widely available 2023+)
```

## Grid - Common Patterns

### Holy-grail layout

```bash
.holy-grail {
  display: grid;
  min-height: 100dvh;
  grid-template-columns: 200px 1fr 200px;
  grid-template-rows: auto 1fr auto;
  grid-template-areas:
    "header  header  header"
    "left    main    right"
    "footer  footer  footer";
}
.hg-header { grid-area: header; }
.hg-left   { grid-area: left; }
.hg-main   { grid-area: main; }
.hg-right  { grid-area: right; }
.hg-footer { grid-area: footer; }
```

### 12-column responsive

```bash
.grid-12 {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 1rem;
}
.col-6 { grid-column: span 6; }
.col-4 { grid-column: span 4; }
.col-3 { grid-column: span 3; }

@media (max-width: 768px) {
  .col-6, .col-4, .col-3 { grid-column: span 12; }
}
```

### Magazine with named areas

```bash
.magazine {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: repeat(3, 200px);
  grid-template-areas:
    "hero hero hero side"
    "hero hero hero side"
    "a    b    c    d";
  gap: 1rem;
}
.hero { grid-area: hero; }
.side { grid-area: side; }
```

### Auto-fit gallery

```bash
.gallery {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(280px, 100%), 1fr));
  gap: 1rem;
}
# min(280px, 100%) prevents overflow when viewport < 280px
```

### Sticky header + scrolling content + footer

```bash
body { min-height: 100dvh; display: grid; grid-template-rows: auto 1fr auto; }
header { position: sticky; top: 0; }
main { overflow-y: auto; }
```

## Grid - Gotchas

### Items overflow because of min-content default

```bash
# BROKEN: long content makes column wider than 1fr
.grid { grid-template-columns: 1fr 1fr 1fr; }
# (row overflows when one cell has long unbreakable content)

# FIXED: minmax(0, 1fr) explicitly allows shrinking below content
.grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
```

### auto-fit vs auto-fill confusion

```bash
# Wide screen, only 2 cards
.cards { grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); }
# 2 cards stretch to fill viewport (cards become huge)

.cards { grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); }
# 2 cards stay at 250px each, rest of row is empty

# Most often you want auto-fit for "responsive grid that compacts"
# Use auto-fill when you want consistent card width regardless of count
```

### subgrid only on the axis you specify

```bash
# BROKEN: assuming subgrid: subgrid handles both
.child { grid-template-rows: subgrid; grid-template-columns: 1fr 2fr; }
# columns are LOCAL - won't align with parent

# FIXED: declare both for full subgrid
.child { grid-template-rows: subgrid; grid-template-columns: subgrid; }
```

### Absolutely-positioned grid items don't participate

```bash
.grid > .badge { position: absolute; top: 0; right: 0; }
# This element is removed from grid flow - grid-row/column placement IGNORED
# Use grid placement OR absolute, not both for layout

# Use grid placement to position within a cell:
.grid > .badge { grid-area: 1 / 4; align-self: start; justify-self: end; }
```

## Positioning

```bash
position: static;       # default - normal flow; top/right/bottom/left/z-index IGNORED
position: relative;     # offset from normal position; reserves original space; creates positioning context
position: absolute;     # removed from flow; positioned to nearest positioned ancestor (else viewport)
position: fixed;        # removed from flow; positioned to viewport (initial containing block)
position: sticky;       # relative until scroll threshold met, then fixed within scroll container
```

### Examples

```bash
.parent { position: relative; }     # makes children's absolute positioning relative to this
.child  { position: absolute; top: 10px; right: 10px; }

.fab    { position: fixed; bottom: 1rem; right: 1rem; }   # always visible, doesn't scroll

.relative-tweak { position: relative; top: 2px; }   # nudge without disturbing siblings

.sticky-header  { position: sticky; top: 0; }       # sticks at top while scrolling
```

### Inset shorthand

```bash
.modal { position: fixed; inset: 0; }   # = top:0; right:0; bottom:0; left:0
.modal { position: absolute; inset: 1rem; }   # 1rem from each edge

.x { inset-block: 0; }      # top + bottom
.x { inset-inline: 0; }     # left + right (logical)
```

### z-index requires non-static

```bash
# BROKEN
.over { z-index: 999; }   # IGNORED - position is default static

# FIXED
.over { position: relative; z-index: 999; }
```

## Sticky Positioning Deep Dive

`position: sticky` confuses everyone. Four conditions must hold for it to work.

```bash
# 1. Ancestor must SCROLL - the sticky element sticks within its closest scrolling ancestor
.parent { overflow-y: auto; height: 600px; }   # provides scroll context
.parent .sticky { position: sticky; top: 0; }

# 2. At least ONE of top/right/bottom/left must be set
.x { position: sticky; }       # BROKEN - no threshold
.x { position: sticky; top: 0; }   # WORKS

# 3. Sticky element's height must be < scroll container's height
# If sticky element is 100vh and container is 100vh, nothing to scroll past

# 4. Direct parent must NOT have overflow: hidden, scroll, or auto on the SAME axis
.outer { overflow: hidden; }
.outer .sticky { position: sticky; top: 0; }   # SILENTLY BROKEN
```

### "Why isn't sticky sticking?" checklist

```bash
# 1. Did you set top/right/bottom/left?
# 2. Does the closest scrolling ancestor have a fixed height (or is it the viewport)?
# 3. Are any ancestors clipping with overflow: hidden / auto / scroll?
# 4. Is the sticky element shorter than its scroll container?
# 5. Is the sticky element a direct child of the scroll container? (Need not be, but indirect parents like overflow:hidden break it.)
# 6. Is display: contents or table-row applied? Sticky doesn't work on table-row in some browsers.
```

```bash
# Common fix: remove unnecessary overflow on ancestors
.layout { /* removed overflow-x: hidden */ }
.section { overflow-x: clip; }   # clip allows sticky descendants in some browsers
```

## Z-Index and Stacking Contexts

`z-index` only matters within a stacking context. Many properties create new stacking contexts (often surprisingly).

```bash
# Creates a stacking context:
position: relative; z-index: <integer>;
position: absolute; z-index: <integer>;
position: fixed;
position: sticky;
opacity: <less than 1>;        # opacity: 0.99 -> NEW context
transform: <not none>;          # transform: translate(0) -> NEW context
filter: <not none>;
backdrop-filter: <not none>;
will-change: <transform|opacity|...>;
isolation: isolate;             # the canonical fix
contain: layout | paint | strict | content;
mix-blend-mode: <not normal>;
mask: <something>;
clip-path: <something>;
```

### The "z-index: 9999 not working" canonical fix

```bash
# Symptom: sibling has transform/opacity, your high z-index has no effect
<div class="card">
  <img class="image" />            # transform: scale(1.05) on hover -> creates context
  <div class="badge"></div>         # z-index: 9999 inside card
</div>

# Each card has its own stacking context, so badge's z-index only competes WITHIN that card
# Solution if cards overlap and you want some badges above others:
.card { isolation: isolate; }      # explicit, clean stacking context
.card.featured { isolation: isolate; z-index: 2; }   # whole card pops above peers
```

### Debugging z-index

```bash
# Browser DevTools: 3D View / Layer panel shows stacking contexts
# Common surprises:
# - opacity: 0.99 silently creates context
# - transform: translate3d(0,0,0) creates context (the "GPU acceleration hack")
# - position: sticky creates context
# - filter: blur(0) creates context
```

## Float and Clear

Legacy 1-D layout system. Mostly replaced by flexbox/grid. Still useful for text wrapping.

```bash
.image { float: left; margin-right: 1rem; }
# Following text wraps around the floated image

.clear-both { clear: both; }
.clear-left { clear: left; }
.clear-right { clear: right; }

# Container collapses around floats - the clearfix
.parent::after { content: ""; display: table; clear: both; }
# OR modern:
.parent { display: flow-root; }     # contains floats without pseudo-element
```

### When to still use float

```bash
# Text wrap around image in article
article > img { float: right; margin-inline-start: 1rem; max-width: 40%; }
# That's basically it. For everything else, use flex/grid.
```

### Modern: shape-outside

```bash
img.circle { float: left; shape-outside: circle(); margin-right: 1rem; }
# Text wraps along the circle's edge instead of bounding box
```

## Container Queries

Style elements based on their parent's size, not the viewport. The breakthrough of 2023.

```bash
# 1. Mark an ancestor as a query container
.card-container {
  container-type: inline-size;        # query its inline (horizontal in LTR) size
  container-name: card;               # optional name
}

# 2. Query that container
@container card (min-width: 400px) {
  .card { display: grid; grid-template-columns: 120px 1fr; }
}
@container card (min-width: 600px) {
  .card { padding: 2rem; }
}

# Anonymous container query (queries nearest container)
@container (min-width: 400px) {
  .card { /* ... */ }
}
```

### container-type values

```bash
container-type: inline-size;    # query inline (horizontal in LTR) size only - most common
container-type: size;           # query both axes (requires defined block size on container)
container-type: normal;         # not a query container; default
```

### cqw / cqh / cqi / cqb units

```bash
# Container Query units - sized relative to query container
.card h2 { font-size: 5cqi; }      # 5% of container's inline size
# cqw = % of container width
# cqh = % of container height
# cqi = % of container inline size  (logical)
# cqb = % of container block size   (logical)
# cqmin / cqmax = min/max of cqi/cqb
```

### Component responsive to its parent (the win)

```bash
# Old way (media query): every component responsive to viewport
@media (min-width: 768px) { .card { display: grid; } }
# But same card placed in a sidebar at 320px wide breaks this assumption

# New way: component responsive to its actual parent
.sidebar, .main { container-type: inline-size; }
@container (min-width: 400px) { .card { display: grid; } }
# Card adapts whether it's in a 1280px main or 320px sidebar
```

## Container Style Queries

Query parent custom-property values to drive child styles. (2023+, limited support)

```bash
.theme {
  container-name: theme;
  --mode: dark;
}
.theme[data-state="open"] { --state: open; }

@container theme style(--state: open) {
  .panel { display: block; }
}
@container theme style(--mode: dark) {
  .text { color: white; }
}

# Limited support - check caniuse before relying. Chrome 111+, Safari 18+ (partial).
```

## Aspect Ratio

```bash
.video { aspect-ratio: 16 / 9; width: 100%; }
.square { aspect-ratio: 1; }
.portrait { aspect-ratio: 3/4; max-width: 400px; }

# Works on flex/grid items too
.flex-item { aspect-ratio: 1; }    # square card

# Replaces the OLD padding-bottom hack:
# OLD:
.video { padding-bottom: 56.25%; height: 0; position: relative; }
.video > iframe { position: absolute; inset: 0; width: 100%; height: 100%; }
# NEW:
.video { aspect-ratio: 16/9; }
.video > iframe { width: 100%; height: 100%; }
```

### Caveat: when content overflows

```bash
# If content forces height beyond the ratio, ratio is ignored
.x { aspect-ratio: 16/9; width: 400px; }
# Big content -> grows tall, breaking 16:9

# Fix: clip content
.x { aspect-ratio: 16/9; width: 400px; overflow: hidden; }
```

## Intrinsic Sizing

```bash
width: min-content;       # smallest the content can be (longest unbreakable token)
width: max-content;       # widest content needs without wrapping
width: fit-content;       # min(max-content, available) - shrinks to content but caps at parent
width: fit-content(20rem);  # function form - cap at 20rem
width: auto;              # context-dependent default

# Practical: tooltip that's only as wide as text needs but caps at 20em
.tooltip { width: max-content; max-width: 20em; }
# OR
.tooltip { width: fit-content(20em); }
```

### How min-width: 0 unlocks shrinking

```bash
# In flex/grid, items have implicit min-width: auto = "min-content"
# A long URL won't let the cell shrink below its width

# To allow shrinking below content min-size:
.flex-item { min-width: 0; }
.grid-cell { min-width: 0; }   # or grid-template-columns: minmax(0, 1fr)
```

## Logical Properties for Layout

Direction-agnostic. Auto-flip in RTL languages.

```bash
# block axis = vertical (in horizontal-tb writing mode); inline axis = horizontal

margin-block: 1rem;             # = margin-top + margin-bottom
margin-inline: auto;            # = margin-left + margin-right (centers)
margin-block-start: 1rem;       # = margin-top
margin-block-end: 0;            # = margin-bottom
margin-inline-start: 1rem;      # = margin-left in LTR, margin-right in RTL
margin-inline-end: 0;

padding-block: 1rem;
padding-inline: 1rem;

border-inline: 1px solid;
border-block-start: 2px solid;

inset-block: 0;                 # = top: 0; bottom: 0;
inset-inline: 0;                # = left: 0; right: 0;
inset: 0;                       # all four

# writing-mode interaction
.vertical-text { writing-mode: vertical-rl; }
# Now block axis = horizontal, inline axis = vertical
# margin-block-start now means margin-RIGHT
```

### The i18n flip win

```bash
# OLD: separate stylesheets for LTR/RTL or [dir="rtl"] overrides
[dir="ltr"] .card { padding-left: 1rem; border-left: 2px solid; }
[dir="rtl"] .card { padding-right: 1rem; border-right: 2px solid; padding-left: 0; border-left: 0; }

# NEW: write once, auto-flips
.card { padding-inline-start: 1rem; border-inline-start: 2px solid; }
```

## Multi-Column Layout

Newspaper-style flow across columns.

```bash
.article {
  column-count: 3;            # exactly 3 columns
  column-gap: 2rem;
  column-rule: 1px solid #ccc;   # vertical line between columns
}

.article {
  column-width: 20rem;        # as many columns as fit, each ~20rem wide
}

.article {
  columns: 3 20rem;           # shorthand: count + width
}

.article h2 {
  column-span: all;           # span across all columns (heading)
}

.figure {
  break-inside: avoid;        # don't split this element across columns
}
.heading {
  break-after: column;        # force new column after
  break-before: avoid;
}
```

### Use cases

```bash
# Good: long-form text articles, glossaries, address books
# Avoid: product UI - users hate scroll-down then scroll-up reading flow
```

## Responsive Design Patterns

### Breakpoint set (canonical)

```bash
# Mobile-first - default styles for smallest, layer up
.container { padding: 1rem; }

@media (min-width: 640px)  { .container { padding: 1.5rem; } }   # sm
@media (min-width: 768px)  { .container { padding: 2rem; } }     # md
@media (min-width: 1024px) { .container { padding: 3rem; } }     # lg
@media (min-width: 1280px) { .container { padding: 4rem; } }     # xl
@media (min-width: 1536px) { .container { padding: 5rem; } }     # 2xl
```

### Mobile-first vs desktop-first

```bash
# Mobile-first (PREFERRED)
.nav { display: flex; flex-direction: column; }
@media (min-width: 768px) { .nav { flex-direction: row; } }
# Smallest screens get the simplest CSS, larger layers on enhancements

# Desktop-first
.nav { display: flex; flex-direction: row; }
@media (max-width: 767px) { .nav { flex-direction: column; } }
# Less common - max-width queries can be confusing with overlapping ranges
```

### Container queries replacing media queries for component-level

```bash
# Instead of viewport-based responsive component:
@media (min-width: 768px) { .card { display: grid; } }

# Component responsive to ITSELF:
.card-wrap { container-type: inline-size; }
@container (min-width: 400px) { .card { display: grid; } }
```

### Fluid typography with clamp()

```bash
# clamp(MIN, PREFERRED, MAX) - scale fluidly between min and max
h1 { font-size: clamp(2rem, 5vw + 1rem, 4rem); }
# At 320px viewport: about 2.4rem; at 1280px: about 5rem; capped at 4rem

# Body type
p { font-size: clamp(1rem, 0.95rem + 0.25vw, 1.125rem); line-height: 1.6; }

# Spacing
section { padding-block: clamp(2rem, 5vw, 6rem); }
```

### Responsive images

```bash
img {
  max-width: 100%;
  height: auto;
  display: block;
}

# srcset / sizes (HTML)
<img src="hero-800.jpg"
     srcset="hero-400.jpg 400w, hero-800.jpg 800w, hero-1600.jpg 1600w"
     sizes="(max-width: 768px) 100vw, 800px"
     alt="">
```

## Modern Reset and Defaults

### Andy Bell modern reset (2023)

```bash
*, *::before, *::after { box-sizing: border-box; }

* { margin: 0; }

html, body { height: 100%; }

body {
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}

img, picture, video, canvas, svg {
  display: block;
  max-width: 100%;
}

input, button, textarea, select { font: inherit; }

p, h1, h2, h3, h4, h5, h6 {
  overflow-wrap: break-word;
}

#root, #__next { isolation: isolate; }
```

### Josh Comeau reset (similar, slightly more)

```bash
/* https://www.joshwcomeau.com/css/custom-css-reset/ */
*, *::before, *::after { box-sizing: border-box; }
* { margin: 0; }
body { line-height: 1.5; -webkit-font-smoothing: antialiased; }
img, picture, video, canvas, svg { display: block; max-width: 100%; }
input, button, textarea, select { font: inherit; }
p, h1, h2, h3, h4, h5, h6 { overflow-wrap: break-word; }
#root, #__next { isolation: isolate; }
```

### System font stack

```bash
body {
  font-family: system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
}

# Or just:
body { font-family: system-ui, sans-serif; }

# Mono
code, pre, kbd { font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace; }
```

### all: revert at top

```bash
# Reset specific element to user-agent default
.unstyled { all: revert; }
# Use sparingly - typically for embedded widgets that need to escape parent styles
```

## Display: Flow-Root

Modern, side-effect-free way to create a BFC.

```bash
.parent { display: flow-root; }
# Effect:
# - Floats inside don't escape (replaces clearfix)
# - Margin-collapse-with-parent prevented
# - No scrollbars (vs overflow: auto)
# - No clipping (vs overflow: hidden)
# - No positioning context (vs position: absolute)
# - No new layout/paint context (vs contain)

# Replaces all of these legacy patterns:
.parent::after { content: ""; display: table; clear: both; }    # clearfix
.parent { overflow: hidden; }                                   # contain floats
.parent { padding-top: 1px; }                                   # margin collapse hack
```

## Display: Contents

Removes the element's box but keeps its children in the parent's layout.

```bash
.unwrap { display: contents; }
# Useful when you have a wrapper required by component framework but don't want
# its box to participate in flex/grid layout.

<div class="grid">     <!-- display: grid -->
  <div class="unwrap">  <!-- display: contents -> children become grid items -->
    <div class="a"></div>
    <div class="b"></div>
  </div>
</div>
```

### Accessibility caveat

```bash
# In some browsers, display: contents removes the element from the accessibility tree
# Bug: button with display: contents loses its role
# Fix: avoid display: contents on interactive elements (button, a, input, etc.)
# Safe on: div, span, fragments
# Audit with screen reader before shipping
```

## Centering

The seven canonical ways to center.

### 1. Block element with auto margin (horizontal only)

```bash
.center { width: 600px; margin-inline: auto; }
```

### 2. Inline / inline-block via text-align

```bash
.parent { text-align: center; }
.child  { display: inline-block; }   # or just inline content
```

### 3. Flex - both axes

```bash
.parent { display: flex; justify-content: center; align-items: center; }
# or
.parent { display: flex; place-items: center; place-content: center; }
```

### 4. Grid - cleanest

```bash
.parent { display: grid; place-items: center; min-height: 100dvh; }
```

### 5. Absolute + transform

```bash
.parent { position: relative; }
.child  { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); }
```

### 6. Absolute + inset 0 + margin auto (fixed-size child)

```bash
.parent { position: relative; }
.child  { position: absolute; inset: 0; margin: auto; width: 200px; height: 100px; }
```

### 7. Single line of text - line-height

```bash
.btn { height: 40px; line-height: 40px; text-align: center; }
# Trick: line-height = container height vertically centers a SINGLE line
# Doesn't work for multi-line content
```

## Common Layout Errors

### Browser dev tool messages and what they mean

```bash
# "Invalid value for grid-template-columns"
.grid { grid-template-columns: 1fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr 1fr; }
# Often means a typo or unsupported value; check for missing units

# "Layout calc thrash" / "Forced reflow"
# Fired in Performance panel when JS reads layout property after writing one
element.style.width = '100px';
const h = element.offsetHeight;   # forces synchronous reflow

# Fix: batch reads then writes
const h = element.offsetHeight;
element.style.width = '100px';
```

### "Stretching across overflow scroll" pattern

```bash
# BROKEN: nested flex/grid with horizontal scroll
<div class="page">          <!-- display: flex; flex-direction: column -->
  <div class="content">     <!-- flex: 1 -->
    <div class="table">     <!-- overflow-x: auto -->
      <table style="min-width: 2000px"></table>
    </div>
  </div>
</div>
# Symptom: page expands to 2000px wide, table doesn't scroll

# FIXED: min-width: 0 on flex/grid ancestors
.page { display: flex; flex-direction: column; }
.content { flex: 1; min-width: 0; }       # KEY
.table { overflow-x: auto; min-width: 0; } # KEY
```

## Common Gotchas

### Flex items can't shrink below content

```bash
# BROKEN
.flex-item { flex: 1; }
# Long URL inside refuses to shrink

# FIXED
.flex-item { flex: 1; min-width: 0; }
```

### Grid items overflow viewport

```bash
# BROKEN
.grid { grid-template-columns: 1fr 1fr 1fr; }   # 1fr has implicit min: auto

# FIXED
.grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
```

### Sticky inside overflow: hidden

```bash
# BROKEN
.outer { overflow: hidden; }
.outer .header { position: sticky; top: 0; }   # silently doesn't stick

# FIXED
.outer { overflow: clip; }    # clip allows sticky descendants (modern)
# OR remove the overflow entirely
```

### Transform creates a stacking context

```bash
# BROKEN: card has transform, child .modal with z-index can't escape card
.card { transform: scale(1); }   # creates context
.card .modal { position: fixed; z-index: 9999; }   # trapped within card

# FIXED: portal the modal to body, OR remove transform from card
```

### height: 100% needs parent with explicit height

```bash
# BROKEN
.parent { /* no height */ }
.child  { height: 100%; }   # 100% of WHAT?

# FIXED
html, body { height: 100%; }
.parent { height: 100%; }
.child  { height: 100%; }

# OR use flex/grid which provide implicit sizing
.parent { display: flex; flex-direction: column; height: 100dvh; }
.child  { flex: 1; }
```

### vh issues on mobile

```bash
# BROKEN: 100vh includes the dynamic browser chrome on mobile, causing scroll
.hero { height: 100vh; }

# FIXED: small/dynamic viewport units (2022+)
.hero { height: 100svh; }   # smallest viewport - never includes chrome
.hero { height: 100dvh; }   # dynamic - resizes as chrome shows/hides
.hero { height: 100lvh; }   # largest viewport

# Practical: cascade
.hero { height: 100vh; height: 100dvh; }   # fallback then modern
```

### align-items vs align-content in single-line flex

```bash
# In a SINGLE-line flex, align-content has NO EFFECT
.flex { display: flex; align-content: center; }   # IGNORED - single line
# Use align-items in single-line:
.flex { display: flex; align-items: center; }    # works

# align-content only matters in multi-line (flex-wrap: wrap with multiple rows)
```

## Performance

### Composite layers via transform / opacity

```bash
# These properties are GPU-accelerated (composited):
transform, opacity, filter
# Animating them is cheap - browser doesn't relayout or repaint

# These trigger layout (slow):
width, height, top, left, padding, margin, font-size

# BAD - layout-thrashing animation
.box { transition: left 0.3s; }
.box:hover { left: 100px; }   # forces layout per frame

# GOOD - composited animation
.box { transition: transform 0.3s; }
.box:hover { transform: translateX(100px); }
```

### will-change

```bash
# Hint to browser to promote element to its own layer
.smooth { will-change: transform; }
# WARNING: don't apply to many elements - GPU memory cost
# Apply briefly before animation, remove after:
element.style.willChange = 'transform';
element.addEventListener('transitionend', () => element.style.willChange = 'auto');
```

### content-visibility for offscreen optimization

```bash
.section { content-visibility: auto; contain-intrinsic-size: auto 1000px; }
# Browser skips rendering off-screen sections; massive speedup on long pages
# contain-intrinsic-size reserves layout space (so scrollbar accurate)
# Limited support: Chrome 85+, Safari 18+, Firefox 125+
```

### CSS containment

```bash
.widget { contain: layout; }    # widget's layout doesn't affect outside
.widget { contain: paint; }     # widget's paint doesn't bleed (clips)
.widget { contain: size; }      # widget has explicit size; ignore content
.widget { contain: content; }   # = layout + paint
.widget { contain: strict; }    # = layout + paint + size

# Browser optimization hint - skips work on isolated subtrees during reflow
```

## Accessibility - Layout Considerations

### Logical order matters

```bash
# Visual order can be reorderd (flexbox order, grid placement) but DOM/source order
# remains for tab focus and screen reader reading order

# Visual: [B] [A] [C]   but DOM: A, B, C
# Tab focus and screen reader: A, B, C (correct)
# Sighted users: B, A, C (visually confusing)

# RULE: keep visual and source order aligned unless you have a strong reason
```

### tabindex and DOM order

```bash
# tabindex="0"  - element joins tab order at its DOM position
# tabindex="-1" - element programmatically focusable but not in tab order
# tabindex >= 1 - DON'T USE - jumps element ahead of natural order; accessibility nightmare
```

### Tab focus and invisibility

```bash
# Hidden elements (display: none, hidden attribute) - skipped, correct
# Visibility hidden - skipped, correct
# Off-screen positioned (left: -9999px) - STILL FOCUSABLE - accessibility hazard
# .sr-only pattern - visually hidden but available to AT (correct):
.sr-only {
  position: absolute;
  width: 1px; height: 1px;
  padding: 0; margin: -1px;
  overflow: hidden;
  clip: rect(0,0,0,0);
  white-space: nowrap;
  border: 0;
}
```

### Antipattern: absolute positioned far away then visually hidden

```bash
# Don't:
.fake-hidden { position: absolute; left: -9999px; }
# Element is still tab-focusable - sighted keyboard users see focus ring scrolling off-screen

# Do (.sr-only above) or aria-hidden + tabindex=-1 if interactive
```

## Common Layout Patterns

### Sidebar + main

```bash
.shell { display: grid; grid-template-columns: 280px 1fr; min-height: 100dvh; }
.shell > aside { background: #f8f8f8; }
.shell > main { min-width: 0; padding: 2rem; }   # min-width: 0 for content overflow
```

### Cards in grid

```bash
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}
.card { display: flex; flex-direction: column; gap: 0.5rem; }
.card .body { flex: 1; }   # body grows so footers align across cards
```

### Nav: logo + links + actions

```bash
.nav { display: flex; align-items: center; gap: 1rem; padding: 1rem; }
.nav .logo { /* default sizing */ }
.nav .links { display: flex; gap: 1rem; margin-inline-start: auto; }
.nav .actions { display: flex; gap: 0.5rem; }
```

### Sticky table headers

```bash
table { border-collapse: collapse; }
thead th { position: sticky; top: 0; background: white; z-index: 1; }
# Required: a scrolling ancestor or window scroll
```

### Full-page hero

```bash
.hero {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  text-align: center;
  padding: 2rem;
  background: linear-gradient(135deg, #667eea, #764ba2);
}
```

### Modal overlay

```bash
.modal-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 100; }
.modal {
  position: fixed; inset: 0; margin: auto;
  width: min(500px, calc(100% - 2rem));
  max-height: calc(100% - 2rem);
  background: white; border-radius: 8px;
  z-index: 101;
  overflow-y: auto;
}

# Even cleaner with <dialog>
dialog::backdrop { background: rgba(0,0,0,0.5); }
dialog { max-width: 500px; border: 0; border-radius: 8px; }
```

### Toast notifications

```bash
.toaster {
  position: fixed;
  inset-block-start: 1rem;
  inset-inline-end: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  z-index: 1000;
}
.toast { padding: 1rem; background: #333; color: white; border-radius: 4px; }
```

### Sticky footer

```bash
body { min-height: 100dvh; display: grid; grid-template-rows: auto 1fr auto; }
header, main, footer { /* normal */ }
# Or flex variant:
body { min-height: 100dvh; display: flex; flex-direction: column; }
main { flex: 1; }
```

## Print Layout

```bash
@media print {
  body { font-size: 12pt; color: black; background: white; }
  nav, footer, .no-print { display: none; }
  a { color: black; text-decoration: underline; }
  a[href]::after { content: " (" attr(href) ")"; font-size: 0.8em; }
}

@page {
  size: A4 portrait;       # or 'letter', 'A3 landscape', etc.
  margin: 2cm;
}
@page :first { margin-top: 4cm; }   # first-page-only margins

/* Page break control */
.section { page-break-before: always; }    # legacy
.section { break-before: page; }           # modern equivalent

.no-split { page-break-inside: avoid; }    # legacy
.no-split { break-inside: avoid; }         # modern

p { orphans: 3; widows: 3; }
# orphans: minimum lines at bottom of page
# widows: minimum lines at top of next page
```

### Print gotchas

```bash
# - Floats and complex grids often render differently in print
# - Background colors / images often disabled by browser default ("Background graphics" checkbox)
# - Use thicker borders if you rely on them (printers eat thin lines)
# - Test in browser print preview AND actual printed output (Chrome/Safari/Firefox differ)
```

## Idioms

### The modern reset block (paste at top)

```bash
*, *::before, *::after { box-sizing: border-box; }
* { margin: 0; }
html, body { height: 100%; }
body { line-height: 1.5; -webkit-font-smoothing: antialiased; }
img, svg, video { display: block; max-width: 100%; }
input, button, textarea, select { font: inherit; }
```

### aspect-ratio over padding-bottom hack

```bash
.video { aspect-ratio: 16/9; }      # CLEAN
# not
.video { padding-bottom: 56.25%; height: 0; position: relative; }
```

### clamp() for fluid type

```bash
h1 { font-size: clamp(2rem, 5vw + 1rem, 4rem); }
```

### Container queries for component independence

```bash
.section { container-type: inline-size; }
@container (min-width: 400px) { .card { display: grid; } }
```

### Intrinsic-grid auto-fit minmax for responsive cards

```bash
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; }
```

### Logical properties for i18n

```bash
.card { padding-inline: 1rem; margin-block: 1rem; border-inline-start: 2px solid; }
```

### display: flow-root for clean BFC

```bash
.parent { display: flow-root; }   # contains floats, prevents margin collapse, no side effects
```

### isolation: isolate to fix z-index woes

```bash
.card { isolation: isolate; }   # explicit stacking context, no transform/opacity hacks
```

### min-width: 0 on flex/grid ancestors

```bash
.flex-item { flex: 1; min-width: 0; }
.grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
```

### inset for shorthand positioning

```bash
.modal { position: fixed; inset: 0; }   # full-coverage overlay
```

### dvh / svh for mobile heights

```bash
.hero { min-height: 100dvh; }   # dynamic, accounts for browser chrome
```

## Tips

```bash
# - When debugging layout, add: * { outline: 1px solid red; }
# - DevTools Layout panel (Firefox/Chrome) shows grid lines and flex axes
# - For grid: use Firefox's grid inspector for area names overlay
# - For stacking issues: Chrome -> 3D View / Layers panel
# - Add `display: grid; grid: auto-flow / 1fr` for one-line equal-column quick test
# - Use `place-items: center` and `place-content: center` to short-circuit center debates
# - When something doesn't shrink: try min-width: 0 first
# - When something doesn't stretch: check min/max-width or align-items: stretch
# - When sticky doesn't stick: check overflow on ancestors
# - When z-index ignored: check for transform/opacity/filter on ancestors
# - When child margin escapes parent: display: flow-root on parent
# - When page horizontally scrolls: search for fixed widths > viewport, missing min-width: 0
# - When animations are janky: use transform/opacity, not width/height/top/left
# - Modern resets are 10-20 lines; pick one and use it consistently
# - Container queries for components, media queries for page-level layout
# - aspect-ratio for media; clamp() for fluid type; logical properties for i18n
# - Test in real RTL: <html dir="rtl">
# - Test in print preview as part of CI for documentation pages
```

## See Also

- css
- html
- html-forms
- javascript
- typescript
- polyglot

## References

- web.dev/learn/css/ -- Google's modern CSS course
- developer.mozilla.org/en-US/docs/Web/CSS -- MDN CSS reference
- css-tricks.com/snippets/css/complete-guide-grid/ -- canonical grid guide
- css-tricks.com/snippets/css/a-guide-to-flexbox/ -- canonical flexbox guide
- every-layout.dev -- Every Layout (Heydon Pickering, Andy Bell) -- composable layout primitives
- defensivecss.dev -- Ahmad Shadeed -- defensive patterns and gotchas
- smashingmagazine.com/category/css/ -- Smashing Magazine CSS articles
- ishadeed.com -- Ahmad Shadeed's blog -- deep dives on CSS layout
- joshwcomeau.com/css/ -- Josh Comeau's CSS posts
- kevinpowell.co -- Kevin Powell -- CSS video tutorials
- web.dev/learn/design/ -- responsive design principles
- caniuse.com -- browser support matrix
- developer.chrome.com/blog/has-m105/ -- :has() selector and modern CSS posts
- W3C CSS Working Group drafts -- drafts.csswg.org
