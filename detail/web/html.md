# The Mathematics of HTML — Document Parsing and DOM Internals

> *HTML is parsed into a DOM tree that the browser renders. The math covers DOM tree complexity, parsing performance, resource loading waterfall, and accessibility tree construction.*

---

## 1. DOM Tree Structure — Graph Theory

### The Model

The DOM is a **tree** (connected acyclic graph) where each node is an element, text node, or comment.

### Tree Properties

$$\text{Total Nodes} = \text{Elements} + \text{Text Nodes} + \text{Comments} + \text{Attributes (as nodes)}$$

$$\text{Depth} = \max(\text{distance from root to any leaf})$$

$$\text{Branching Factor (avg)} = \frac{\text{Total Children}}{\text{Non-Leaf Nodes}}$$

### Real-World DOM Statistics

| Website Type | Elements | Depth | Avg Branching |
|:---|:---:|:---:|:---:|
| Simple blog | 500-1,000 | 10-15 | 3-5 |
| News site | 2,000-5,000 | 15-25 | 5-10 |
| SPA (React/Vue) | 3,000-10,000 | 20-40 | 5-15 |
| Complex app | 10,000-50,000 | 30-60 | 10-20 |

### DOM Operation Complexity

| Operation | Complexity | Notes |
|:---|:---:|:---|
| `getElementById` | O(1) | Hash table lookup |
| `getElementsByClassName` | O(n) | Full tree traversal |
| `querySelector` | O(n) | Selector matching |
| `querySelectorAll` | O(n) | Full tree scan |
| `appendChild` | O(1) | Pointer update |
| `removeChild` | O(1) | Pointer update |
| `innerHTML =` | O(n) | Parse + build subtree |
| `cloneNode(true)` | O(s) | s = subtree size |

---

## 2. HTML Parsing — State Machine

### The Model

HTML parsing uses a **state machine** with ~80 states (per the spec). The tokenizer produces tokens that the tree builder consumes.

### Parsing Complexity

$$T_{parse} = O(n) \quad \text{where } n = \text{document size in bytes}$$

### Parsing Throughput

| Parser | Throughput | 100 KiB Page | 1 MiB Page |
|:---|:---:|:---:|:---:|
| HTML tokenizer | ~500 MiB/s | 0.2 ms | 2 ms |
| Tree construction | ~200 MiB/s | 0.5 ms | 5 ms |
| Total parse | ~150 MiB/s | 0.7 ms | 6.7 ms |

### Speculative Parsing (Preload Scanner)

The preload scanner runs ahead of the main parser to discover resources:

$$T_{resource\_discovery} = \min(T_{main\_parser}, T_{preload\_scan})$$

$$\text{Preload Savings} = T_{main\_parser} - T_{preload\_scan}$$

Resources discovered by the preload scanner start downloading immediately, potentially saving:

$$T_{saved} = T_{parse\_to\_discovery} - T_{preload\_time}$$

---

## 3. Resource Loading — Critical Path

### The Model

HTML loading follows a dependency waterfall. The **critical rendering path** determines time to first paint.

### Critical Path Length

$$T_{critical} = T_{HTML} + \max(T_{CSS\_blocking}, T_{JS\_blocking}) + T_{layout} + T_{paint}$$

### Resource Loading Formulas

$$T_{resource} = T_{DNS} + T_{TCP} + T_{TLS} + T_{TTFB} + \frac{\text{Size}}{\text{Bandwidth}}$$

### HTTP/2 Multiplexing

$$T_{serial} = \sum_{i=1}^{n} T_i \quad (\text{HTTP/1.1, 6 connections max})$$

$$T_{parallel} = \max(T_i) + \frac{\sum \text{Sizes}}{\text{BW}} \quad (\text{HTTP/2, single connection})$$

### Worked Example

*"Page with HTML (50 KiB), 3 CSS files (30 KiB each), 5 JS files (100 KiB each)."*

**HTTP/1.1** (6 parallel connections):

$$\text{Round 1:} \text{ HTML} \quad T = 50$$

$$\text{Round 2:} \text{ 3 CSS + 3 JS (6 connections)} \quad T = 100\text{ms}$$

$$\text{Round 3:} \text{ 2 JS} \quad T = 100\text{ms}$$

$$T_{total} = 250\text{ms}$$

**HTTP/2** (unlimited multiplexing):

$$\text{All resources in parallel after HTML}$$

$$T_{total} = T_{HTML} + \max(T_{largest}) = 50 + 100 = 150\text{ms}$$

---

## 4. Semantic Structure — Accessibility Tree

### The Model

The browser builds an **accessibility tree** from the DOM, mapping elements to ARIA roles.

### Role Mapping

| Element | Default Role | Accessible Name Source |
|:---|:---|:---|
| `<button>` | button | Text content |
| `<a href>` | link | Text content |
| `<input type="text">` | textbox | `<label>` or `aria-label` |
| `<img>` | img | `alt` attribute |
| `<nav>` | navigation | `aria-label` |
| `<main>` | main | Implicit |
| `<h1>`-`<h6>` | heading (level 1-6) | Text content |
| `<div>` | none (generic) | N/A |
| `<span>` | none (generic) | N/A |

### Heading Hierarchy

Valid heading structure forms a tree where:

$$\text{Level}_{child} > \text{Level}_{parent}$$

$$\text{Valid:} \quad h1 \rightarrow h2 \rightarrow h3$$

$$\text{Invalid:} \quad h1 \rightarrow h3 \quad (\text{skipped level})$$

### Landmark Region Coverage

$$\text{Coverage} = \frac{\text{Content in Landmarks}}{\text{Total Content}} \times 100\%$$

Target: 100% of visible content should be within a landmark region.

---

## 5. Document Size and Performance

### Byte Budget

$$\text{First Paint} \propto \text{Critical Resource Size}$$

| Metric | Target | HTML Budget |
|:---|:---:|:---:|
| FCP (First Contentful Paint) | < 1.8s | < 50 KiB HTML |
| LCP (Largest Contentful Paint) | < 2.5s | < 200 KiB total critical |
| TTI (Time to Interactive) | < 3.8s | < 300 KiB JS |

### DOM Size Limits

| DOM Elements | Impact |
|:---:|:---|
| < 800 | Excellent performance |
| 800-1,500 | Good performance |
| 1,500-3,000 | Acceptable |
| 3,000-5,000 | Performance degrades |
| > 5,000 | Layout thrashing, slow selectors |

### Selector Matching Cost

$$T_{selector} = \text{DOM Nodes} \times T_{match\_per\_node}$$

Browsers match selectors **right to left**:

`.container .item span` matches: find all `span`, filter parent `.item`, filter grandparent `.container`.

$$T_{match} = O(\text{spans} \times d) \quad \text{where } d = \text{selector depth}$$

---

## 6. Character Encoding and Entity Math

### UTF-8 Encoding Sizes

$$\text{Bytes per Character} = \begin{cases} 1 & \text{U+0000 to U+007F (ASCII)} \\ 2 & \text{U+0080 to U+07FF (Latin extended)} \\ 3 & \text{U+0800 to U+FFFF (most scripts)} \\ 4 & \text{U+10000 to U+10FFFF (emoji, rare)} \end{cases}$$

### HTML Entity Expansion

| Entity | Characters | Bytes (UTF-8) | Entity Length |
|:---|:---:|:---:|:---:|
| `&amp;` | 1 (`&`) | 1 | 5 |
| `&lt;` | 1 (`<`) | 1 | 4 |
| `&gt;` | 1 (`>`) | 1 | 4 |
| `&quot;` | 1 (`"`) | 1 | 6 |
| `&#x1F600;` | 1 (emoji) | 4 | 10 |

### Compression Effectiveness

HTML compresses well due to repetitive markup:

$$\text{Compression Ratio}_{HTML} \approx 5-10\times \text{ (gzip/brotli)}$$

| Raw HTML | Gzip | Brotli | Transfer Size |
|:---:|:---:|:---:|:---:|
| 50 KiB | 8 KiB | 6 KiB | 6 KiB |
| 200 KiB | 30 KiB | 22 KiB | 22 KiB |
| 1 MiB | 120 KiB | 90 KiB | 90 KiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| O(n) parse | Linear | HTML parsing |
| O(1) getElementById | Hash lookup | DOM query |
| $T_{HTML} + \max(T_{CSS}, T_{JS})$ | Critical path | First paint time |
| $\frac{\text{Content in Landmarks}}{\text{Total}}$ | Ratio | Accessibility coverage |
| 1-4 bytes per char | Encoding | UTF-8 size |
| $\text{Nodes} \times T_{match}$ | Linear | Selector cost |

---

*Every browser tab executes this pipeline — parse HTML into DOM, build CSSOM, merge into render tree, layout, paint, composite — a process that happens in under 16ms to maintain 60fps interaction.*

## Prerequisites

- Tree data structures (nodes, depth, traversal)
- Basic graph theory (DOM as a tree, accessibility tree)
- Network fundamentals (HTTP requests, resource loading)

## Complexity

- **Beginner:** DOM tree structure, element counting, basic parsing
- **Intermediate:** Resource loading waterfall, preload/defer/async timing, reflow cost
- **Advanced:** Parser state machine, speculative parsing, accessibility tree construction, critical rendering path optimization
