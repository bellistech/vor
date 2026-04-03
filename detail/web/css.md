# The Mathematics of CSS — Layout Engine and Rendering Internals

> *CSS layout is a constraint-solving system. The math covers the box model, specificity calculation, flexbox distribution, grid sizing, and the rendering pipeline performance model.*

---

## 1. The Box Model — Dimensional Arithmetic

### The Formula

$$\text{Total Width} = \text{margin-left} + \text{border-left} + \text{padding-left} + \text{content-width} + \text{padding-right} + \text{border-right} + \text{margin-right}$$

### box-sizing Impact

$$\text{content-box (default):} \quad \text{element width} = \text{content-width}$$

$$\text{border-box:} \quad \text{element width} = \text{content} + \text{padding} + \text{border}$$

### Worked Example

```css
.box { width: 300px; padding: 20px; border: 5px solid; margin: 10px; }
```

| box-sizing | Content | Padding | Border | Margin | Total |
|:---|:---:|:---:|:---:|:---:|:---:|
| content-box | 300px | 40px | 10px | 20px | 370px |
| border-box | 250px | 40px | 10px | 20px | 320px |

### Percentage Resolution

$$\text{Percentage Width} = \frac{\text{percentage}}{100} \times \text{Containing Block Width}$$

$$\text{Percentage Padding/Margin} = \frac{\text{percentage}}{100} \times \text{Containing Block Width (always width, even vertical)}$$

---

## 2. Specificity — The Scoring System

### The Formula

Specificity is calculated as a tuple $(a, b, c)$ compared left to right:

$$\text{Specificity} = (a, b, c)$$

Where:
- $a$ = count of ID selectors (`#id`)
- $b$ = count of class selectors (`.class`), attribute selectors (`[attr]`), pseudo-classes (`:hover`)
- $c$ = count of element selectors (`div`), pseudo-elements (`::before`)

### Comparison Algorithm

$$(a_1, b_1, c_1) > (a_2, b_2, c_2) \iff a_1 > a_2 \lor (a_1 = a_2 \land b_1 > b_2) \lor (a_1 = a_2 \land b_1 = b_2 \land c_1 > c_2)$$

### Worked Examples

| Selector | IDs (a) | Classes (b) | Elements (c) | Specificity |
|:---|:---:|:---:|:---:|:---:|
| `*` | 0 | 0 | 0 | (0,0,0) |
| `div` | 0 | 0 | 1 | (0,0,1) |
| `div p` | 0 | 0 | 2 | (0,0,2) |
| `.widget` | 0 | 1 | 0 | (0,1,0) |
| `div.widget` | 0 | 1 | 1 | (0,1,1) |
| `#main` | 1 | 0 | 0 | (1,0,0) |
| `#main .widget p` | 1 | 1 | 1 | (1,1,1) |
| `#main #sidebar .active` | 2 | 1 | 0 | (2,1,0) |
| `style=""` (inline) | | | | Always wins* |
| `!important` | | | | Overrides all* |

### Cascade Order (lowest to highest priority)

1. User agent stylesheet
2. Author stylesheet (specificity order)
3. Author `!important`
4. User `!important`
5. CSS animations (`@keyframes`)

---

## 3. Flexbox — Space Distribution Algorithm

### The Model

Flexbox distributes space along a main axis using flex-grow and flex-shrink factors.

### Flex-Grow Distribution

$$\text{Free Space} = \text{Container Size} - \sum \text{Item Base Sizes}$$

$$\text{Growth}_i = \text{Free Space} \times \frac{\text{flex-grow}_i}{\sum \text{flex-grow}_j}$$

$$\text{Final Size}_i = \text{Base Size}_i + \text{Growth}_i$$

### Worked Example

*"Container 600px, three items: A (100px, grow=1), B (150px, grow=2), C (100px, grow=1)."*

$$\text{Free Space} = 600 - (100 + 150 + 100) = 250\text{px}$$

$$\text{Total grow} = 1 + 2 + 1 = 4$$

| Item | Base | Grow Factor | Growth | Final Size |
|:---|:---:|:---:|:---:|:---:|
| A | 100px | 1/4 | 62.5px | 162.5px |
| B | 150px | 2/4 | 125px | 275px |
| C | 100px | 1/4 | 62.5px | 162.5px |
| **Total** | | | | **600px** |

### Flex-Shrink Distribution

When items overflow, shrink is weighted by flex-shrink AND base size:

$$\text{Shrink Factor}_i = \text{flex-shrink}_i \times \text{Base Size}_i$$

$$\text{Shrink}_i = \text{Overflow} \times \frac{\text{Shrink Factor}_i}{\sum \text{Shrink Factor}_j}$$

---

## 4. Grid Layout — Track Sizing Algorithm

### The Model

CSS Grid sizes tracks using a multi-pass algorithm with `fr` units as fractional distribution.

### fr Unit Calculation

$$1\text{fr} = \frac{\text{Available Space} - \text{Fixed Tracks}}{\sum \text{fr values}}$$

### Worked Example

```css
grid-template-columns: 200px 1fr 2fr;
```

*Container width = 1000px.*

$$\text{Available for fr} = 1000 - 200 = 800\text{px}$$

$$1\text{fr} = \frac{800}{1 + 2} = 266.67\text{px}$$

| Track | Size | Pixels |
|:---|:---|:---:|
| Column 1 | 200px | 200px |
| Column 2 | 1fr | 266.67px |
| Column 3 | 2fr | 533.33px |

### minmax() Resolution

$$\text{Track Size} = \max(\text{min}, \min(\text{max}, \text{available}))$$

For `minmax(200px, 1fr)`: Track is at least 200px, at most its fr share.

### auto-fill vs auto-fit

$$\text{auto-fill columns} = \lfloor \frac{\text{Container Width}}{\text{Min Column Width}} \rfloor$$

$$\text{auto-fit:} \text{ same count, but empty tracks collapse to 0}$$

---

## 5. Rendering Pipeline — Performance Model

### The Pipeline

$$\text{Frame Time} = T_{style} + T_{layout} + T_{paint} + T_{composite}$$

Target: 16.67ms for 60fps, 6.94ms for 144fps.

### Cost by CSS Property Change

| Property Change | Triggers | Cost |
|:---|:---|:---:|
| `width`, `height`, `margin` | Style + Layout + Paint + Composite | Highest |
| `color`, `background` | Style + Paint + Composite | Medium |
| `transform`, `opacity` | Composite only | Lowest |

### Layout Thrashing

$$T_{thrash} = n \times (T_{read} + T_{layout})$$

Where $n$ = interleaved read/write operations. Each read forces a synchronous layout.

| Operations | Without Thrashing | With Thrashing |
|:---:|:---:|:---:|
| 10 | 1 layout pass | 10 layout passes |
| 100 | 1 layout pass | 100 layout passes |
| 1,000 | 1 layout pass | 1,000 layout passes |

---

## 6. Media Queries — Breakpoint Math

### Viewport Units

$$1\text{vw} = \frac{\text{Viewport Width}}{100}$$

$$1\text{vh} = \frac{\text{Viewport Height}}{100}$$

$$1\text{vmin} = \min(1\text{vw}, 1\text{vh})$$

$$1\text{vmax} = \max(1\text{vw}, 1\text{vh})$$

### Fluid Typography Formula (clamp)

$$\text{font-size} = \text{clamp}(\text{min}, \text{preferred}, \text{max})$$

$$\text{preferred} = \text{min} + (\text{max} - \text{min}) \times \frac{\text{vw} - \text{min-vw}}{\text{max-vw} - \text{min-vw}}$$

### Common Breakpoints

| Device | Width | Columns (typical) |
|:---|:---:|:---:|
| Mobile (portrait) | 320-480px | 1 |
| Mobile (landscape) | 480-768px | 1-2 |
| Tablet | 768-1024px | 2-3 |
| Desktop | 1024-1440px | 3-4 |
| Large desktop | 1440px+ | 4-6 |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(a, b, c)$ tuple comparison | Lexicographic | Specificity |
| $\frac{\text{grow}_i}{\sum \text{grow}}$ | Proportional distribution | Flexbox growth |
| $\frac{\text{Available}}{\sum \text{fr}}$ | Division | Grid fr units |
| $\text{content} + \text{padding} + \text{border}$ | Addition | Box model |
| $\lfloor \frac{W}{\text{min}} \rfloor$ | Floor division | auto-fill columns |
| $T_{style} + T_{layout} + T_{paint}$ | Pipeline sum | Frame time |

---

*Every browser layout engine executes these algorithms for every frame — a constraint solver that turns declarative CSS rules into precise pixel positions 60+ times per second.*
