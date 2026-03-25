# CSS (Cascading Style Sheets)

Stylesheet language for describing the presentation of HTML documents -- layout, colors, typography, and responsive design.

## Selectors

### Element, Class, ID, Universal

```css
p { color: black; }                      /* element */
.card { padding: 1rem; }                 /* class */
#header { height: 60px; }               /* ID (avoid -- high specificity) */
* { box-sizing: border-box; }            /* universal */
```

### Attribute Selectors

```css
[disabled] { opacity: 0.5; }             /* has attribute */
[type="email"] { border-color: blue; }   /* exact value */
[href^="https"] { color: green; }        /* starts with */
[src$=".png"] { border: none; }          /* ends with */
[class*="btn"] { cursor: pointer; }      /* contains substring */
```

### Pseudo-Classes and Pseudo-Elements

```css
a:hover { text-decoration: underline; }
input:focus { outline: 2px solid blue; }
li:first-child { margin-top: 0; }
li:nth-child(2n) { background: #f5f5f5; }  /* even rows */
p:not(.intro) { font-size: 0.9rem; }
:is(h1, h2, h3) { font-family: serif; }     /* matches any */
:where(h1, h2, h3) { margin-top: 1em; }     /* zero specificity */
input:required { border-left: 3px solid red; }

p::first-line { font-variant: small-caps; }
.tooltip::before { content: "Tip: "; font-weight: bold; }
.clearfix::after { content: ""; display: table; clear: both; }
::placeholder { color: #999; }
::selection { background: yellow; color: black; }
```

### Combinators

```css
article p { line-height: 1.6; }          /* descendant (any depth) */
ul > li { list-style: disc; }            /* child (direct only) */
h2 + p { margin-top: 0; }               /* adjacent sibling */
h2 ~ p { color: #333; }                 /* general sibling */
```

## Specificity

```css
/* (inline, IDs, classes/attrs/pseudo-classes, elements/pseudo-elements) */
/* 0,0,0,1 */  p { }
/* 0,0,1,0 */  .card { }
/* 0,0,1,1 */  p.intro { }
/* 0,1,0,0 */  #header { }
/* 1,0,0,0 */  style="..."   /* inline -- highest */
/* !important overrides all normal specificity -- use sparingly */
```

## Box Model

```css
.box {
  margin: 10px 20px 10px 20px;   /* top right bottom left (clockwise) */
  margin: 10px 20px;             /* vertical | horizontal */
  margin: 0 auto;                /* horizontal centering */
  padding: 1rem 2rem;
  border: 1px solid #ccc;
  border-radius: 8px;
}
/* border-box: width/height includes padding + border */
*, *::before, *::after { box-sizing: border-box; }
```

## Display and Positioning

```css
.block   { display: block; }        /* full width, stacks vertically */
.inline  { display: inline; }       /* flows with text, no width/height */
.ib      { display: inline-block; } /* inline flow, accepts width/height */
.none    { display: none; }         /* removed from flow entirely */

.static   { position: static; }     /* default -- normal flow */
.relative { position: relative; top: 10px; }  /* offset from normal */
.absolute { position: absolute; top: 0; right: 0; }  /* relative to positioned ancestor */
.fixed    { position: fixed; bottom: 0; width: 100%; }  /* relative to viewport */
.sticky   { position: sticky; top: 0; }  /* toggles relative/fixed on scroll */
```

## Flexbox

### Container Properties

```css
.flex-container {
  display: flex;
  flex-direction: row;            /* row | row-reverse | column | column-reverse */
  justify-content: center;       /* flex-start | flex-end | center | space-between | space-around | space-evenly */
  align-items: center;           /* flex-start | flex-end | center | stretch | baseline */
  flex-wrap: wrap;               /* nowrap | wrap | wrap-reverse */
  gap: 1rem;                     /* row-gap and column-gap shorthand */
}
```

### Item Properties

```css
.flex-item {
  flex: 1;                /* shorthand: flex-grow flex-shrink flex-basis */
  flex: 0 0 200px;       /* fixed 200px, no grow/shrink */
  order: -1;              /* visual order (default 0) */
  align-self: flex-end;   /* override container's align-items */
}
```

## Grid

### Container and Items

```css
.grid-container {
  display: grid;
  grid-template-columns: 200px 1fr 1fr;              /* fixed + fractional */
  grid-template-columns: repeat(3, 1fr);              /* 3 equal columns */
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));  /* responsive */
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));   /* collapses empties */
  grid-template-rows: auto 1fr auto;
  gap: 1rem;
}
.grid-item {
  grid-column: 1 / 3;            /* start / end line numbers */
  grid-column: span 2;           /* span 2 columns */
  grid-row: 1 / -1;              /* first to last line */
  grid-area: header;             /* named area */
}
/* Named grid areas */
.layout {
  grid-template-areas:
    "header header"
    "sidebar main"
    "footer footer";
}
```

## Typography

```css
.text {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  font-size: 1rem;                /* 16px default; prefer rem/em */
  font-weight: 400;               /* 100-900, normal=400, bold=700 */
  line-height: 1.5;               /* unitless multiplier preferred */
  letter-spacing: 0.05em;
  text-transform: uppercase;      /* none | capitalize | uppercase | lowercase */
  text-align: center;             /* left | right | center | justify */
  white-space: nowrap;            /* prevents wrapping */
  text-overflow: ellipsis;        /* requires overflow:hidden + white-space:nowrap */
  overflow: hidden;
}
```

## Colors and Custom Properties

```css
:root {
  --primary: #3b82f6;                     /* custom properties (CSS variables) */
  --text: hsl(220, 15%, 20%);            /* hue, saturation, lightness */
  --bg: rgb(255, 255, 255);
  --shadow: rgba(0, 0, 0, 0.1);          /* rgb with alpha */
}
.element {
  color: var(--primary);
  color: var(--undefined, #fallback);     /* fallback value */
  border-color: currentColor;             /* inherits current text color */
  opacity: 0.8;                           /* 0 = transparent, 1 = opaque */
}
```

## Transitions and Animations

```css
.button {
  transition: background-color 0.3s ease, transform 0.2s ease-out;
  transition: all 0.3s ease;              /* shorthand for all properties */
}
.button:hover { background-color: darkblue; transform: scale(1.05); }

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-10px); }
  to   { opacity: 1; transform: translateY(0); }
}
.modal { animation: fadeIn 0.3s ease-out forwards; }
```

## Transforms

```css
.element {
  transform: translate(50px, -20px);       /* x, y */
  transform: scale(1.5);                   /* uniform scale */
  transform: rotate(45deg);
  transform: translate(-50%, -50%) scale(1.2);  /* chained */
  transform-origin: center center;         /* pivot point */
}
```

## Media Queries and Responsive

```css
/* Mobile-first breakpoints */
@media (min-width: 640px)  { /* sm */ }
@media (min-width: 768px)  { /* md */ }
@media (min-width: 1024px) { /* lg */ }
@media (min-width: 1280px) { /* xl */ }

@media (prefers-reduced-motion: reduce) {
  * { animation: none !important; transition-duration: 0.01ms !important; }
}
@media (prefers-color-scheme: dark) {
  :root { --bg: #1a1a1a; --text: #e5e5e5; }
}
@media print { nav, footer { display: none; } }
```

## Z-Index, Overflow

```css
/* z-index only works on positioned elements (not static) */
.dropdown { position: relative; z-index: 10; }
.modal-overlay { position: fixed; z-index: 100; }
.tooltip { position: absolute; z-index: 50; }

.container {
  overflow: visible;      /* default -- content spills out */
  overflow: hidden;       /* clips content */
  overflow: auto;         /* scrollbar only when needed */
  overflow-x: hidden;     /* horizontal only */
}
```

## Common Reset Patterns

```css
*, *::before, *::after { box-sizing: border-box; }
body { margin: 0; font-family: system-ui, sans-serif; line-height: 1.5; }
img, picture, video, svg { display: block; max-width: 100%; }
input, button, textarea, select { font: inherit; }
h1, h2, h3, h4, h5, h6, p { overflow-wrap: break-word; }
ul[role="list"], ol[role="list"] { list-style: none; padding: 0; margin: 0; }

/* Visually hidden but accessible */
.sr-only {
  position: absolute; width: 1px; height: 1px;
  padding: 0; margin: -1px; overflow: hidden;
  clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0;
}
```

## Tips

- Use `box-sizing: border-box` globally -- it makes layout math intuitive.
- Prefer `rem` for font sizes and `em` for component-relative spacing.
- Flexbox for one-dimensional layout (row or column), Grid for two-dimensional.
- `auto-fit` with `minmax()` in grid creates responsive layouts without media queries.
- Avoid `!important` -- fix specificity issues instead (lower selectors or restructure).
- Use custom properties for theming -- they cascade and can be overridden per-component.
- `currentColor` is useful for borders/shadows that should match text color.
- Always include `prefers-reduced-motion` for users who are sensitive to animation.
- Use logical properties (`margin-inline`, `padding-block`) for better internationalization.
- Mobile-first (`min-width`) media queries generally produce cleaner CSS than desktop-first.

## References

- [MDN CSS Reference](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference)
- [MDN CSS Flexbox Guide](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_flexible_box_layout/Basic_concepts_of_flexbox)
- [MDN CSS Grid Guide](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_grid_layout)
- [MDN CSS Selectors](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_selectors)
- [MDN CSS Custom Properties](https://developer.mozilla.org/en-US/docs/Web/CSS/Using_CSS_custom_properties)
- [MDN Media Queries](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_media_queries/Using_media_queries)
- [MDN CSS Transitions](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_transitions/Using_CSS_transitions)
- [MDN CSS Animations](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_animations/Using_CSS_animations)
- [W3C CSS Specifications](https://www.w3.org/Style/CSS/)
- [CSS Specificity Calculator](https://specificity.keegan.st/)
- [Can I Use (Browser Support Tables)](https://caniuse.com/)
