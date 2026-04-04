# HTML (HyperText Markup Language)

Standard markup language for structuring web content -- documents, forms, media, and semantic page layout.

## Document Structure

### Boilerplate and Meta

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="description" content="Page description for SEO">
  <title>Page Title</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <script src="app.js" defer></script>
</body>
</html>

<!-- Open Graph (social sharing) -->
<meta property="og:title" content="Page Title">
<meta property="og:description" content="Description">
<meta property="og:image" content="https://example.com/image.jpg">
<meta property="og:url" content="https://example.com/page">
```

## Semantic Elements

```html
<header>          <!-- introductory content, nav, logo -->
<nav>             <!-- navigation links -->
<main>            <!-- primary content (one per page) -->
  <article>       <!-- self-contained content (blog post, news story) -->
  <section>       <!-- thematic grouping with heading -->
  <aside>         <!-- tangentially related (sidebar, callout) -->
</main>
<footer>          <!-- footer content, copyright, links -->
```

## Text Content

### Headings, Paragraphs, Inline

```html
<h1>Main heading</h1>     <!-- one per page, don't skip levels: h1 > h2 > h3 -->
<h2>Section</h2>
<h3>Subsection</h3>
<p>Text with <strong>strong</strong> and <em>emphasis</em>.</p>
<p><mark>highlight</mark>, <small>fine print</small>, <del>deleted</del>, <ins>inserted</ins></p>
<blockquote cite="https://example.com">Quoted text.</blockquote>
<pre><code>Preformatted code</code></pre>
<abbr title="HyperText Markup Language">HTML</abbr>
<time datetime="2026-03-25">March 25, 2026</time>
```

### Links and Images

```html
<a href="https://example.com">External link</a>
<a href="/about">Internal</a>
<a href="#section-id">Anchor</a>
<a href="mailto:user@example.com">Email</a>
<a href="/file.pdf" download>Download</a>
<a href="https://example.com" target="_blank" rel="noopener noreferrer">New tab</a>

<img src="photo.jpg" alt="Descriptive alt text" width="800" height="600">
<img src="decorative.svg" alt="" role="presentation">  <!-- decorative: empty alt -->
<figure>
  <img src="chart.png" alt="Sales chart showing 20% growth">
  <figcaption>Q4 2025 sales</figcaption>
</figure>
```

## Lists and Tables

```html
<ul><li>Unordered item</li></ul>
<ol start="1"><li>Ordered item</li></ol>
<dl><dt>Term</dt><dd>Definition</dd></dl>

<table>
  <caption>Monthly expenses</caption>
  <thead><tr><th scope="col">Month</th><th scope="col">Amount</th></tr></thead>
  <tbody><tr><td>January</td><td>$1,200</td></tr></tbody>
</table>
<td colspan="2">Spans two columns</td>
<td rowspan="3">Spans three rows</td>
```

## Forms

### Input Types

```html
<form action="/submit" method="POST">
  <label for="name">Name</label>
  <input type="text" id="name" name="name" required placeholder="Jane Doe">
  <input type="email" name="email" required>
  <input type="password" name="pass" minlength="8">
  <input type="number" name="qty" min="1" max="100" step="1">
  <input type="tel" name="phone" pattern="[0-9]{3}-[0-9]{4}">
  <input type="url" name="website">
  <input type="date" name="birthday">
  <input type="file" name="upload" accept=".pdf,.jpg" multiple>
  <input type="search" name="q">
  <input type="hidden" name="csrf" value="token123">
  <input type="range" name="volume" min="0" max="100">
  <input type="color" name="fav-color">
  <button type="submit">Submit</button>
  <button type="button">No default action</button>
</form>
```

### Select, Textarea, Checkbox, Radio

```html
<select name="country">
  <option value="" disabled selected>Choose...</option>
  <optgroup label="North America">
    <option value="us">United States</option>
  </optgroup>
</select>
<textarea name="message" rows="4" cols="50" maxlength="500"></textarea>

<fieldset>
  <legend>Interests</legend>
  <label><input type="checkbox" name="interest" value="code"> Coding</label>
  <label><input type="checkbox" name="interest" value="music"> Music</label>
</fieldset>
<fieldset>
  <legend>Size</legend>
  <label><input type="radio" name="size" value="s"> Small</label>
  <label><input type="radio" name="size" value="m" checked> Medium</label>
</fieldset>
```

### Validation Attributes

```html
<input required>                        <!-- must be filled -->
<input minlength="3" maxlength="50">    <!-- text length -->
<input min="0" max="100">              <!-- numeric range -->
<input pattern="[A-Za-z]{3}">          <!-- regex pattern -->
```

## Media

```html
<img srcset="small.jpg 480w, medium.jpg 800w, large.jpg 1200w"
     sizes="(max-width: 600px) 480px, 800px"
     src="medium.jpg" alt="Responsive image">
<picture>
  <source media="(min-width: 800px)" srcset="wide.jpg">
  <img src="small.jpg" alt="Adaptive image">
</picture>
<video controls width="640" poster="thumb.jpg" preload="metadata">
  <source src="video.mp4" type="video/mp4">
</video>
<audio controls preload="none">
  <source src="audio.mp3" type="audio/mpeg">
</audio>
```

## Structural and Attributes

```html
<div>             <!-- generic block container -->
<span>            <!-- generic inline container -->
<br>              <!-- line break -->
<hr>              <!-- thematic break -->
<details><summary>Click to expand</summary><p>Hidden content.</p></details>

<!-- Global attributes -->
<div id="unique-id">              <!-- unique identifier -->
<div class="card primary">        <!-- space-separated classes -->
<div data-user-id="42">           <!-- custom data attributes -->
<div hidden>                      <!-- hidden from rendering -->
<div tabindex="0">                <!-- makes element focusable -->
```

## Script and Style Loading

```html
<link rel="stylesheet" href="styles.css">                    <!-- blocking -->
<link rel="preload" href="font.woff2" as="font" crossorigin> <!-- preload asset -->
<script src="app.js"></script>            <!-- blocking: stops HTML parsing -->
<script src="app.js" defer></script>      <!-- deferred: runs after parsing, in order -->
<script src="analytics.js" async></script> <!-- async: runs when ready, any order -->
<script type="module" src="mod.js"></script> <!-- ES module: deferred by default -->
```

## Accessibility

```html
<img src="logo.png" alt="Company name logo">
<nav aria-label="Primary navigation">
<button aria-expanded="false" aria-controls="menu">Menu</button>
<div aria-live="polite">Status updates appear here</div>
<div role="alert">Error: invalid input</div>
<a href="#main-content" class="sr-only">Skip to main content</a>
<input aria-invalid="true" aria-describedby="err1">
<span id="err1" role="alert">Email is required</span>
<!-- Landmark roles: role="banner" (header), "navigation" (nav), "main" (main) -->
```

## Entities

```html
&amp; &lt; &gt; &quot; &apos;   <!-- & < > " ' -->
&nbsp; &copy; &mdash; &ndash; &hellip;  <!-- non-breaking space, (c), --, -, ... -->
```

## Tips

- Always start with `<!DOCTYPE html>` -- without it, browsers use quirks mode.
- Use semantic elements (`header`, `nav`, `main`, `article`) over generic divs.
- One `<h1>` per page; never skip heading levels (h1 then h3).
- Every `<img>` needs `alt` -- use empty `alt=""` only for decorative images.
- Use `<button>` for actions and `<a>` for navigation -- never `<div onclick>`.
- Prefer `defer` over `async` for scripts that depend on DOM or each other.
- Always associate `<label>` with inputs via `for`/`id` pairing.
- Include `width` and `height` on images to prevent layout shift during load.
- Test with keyboard-only navigation -- everything interactive must be Tab-reachable.

## See Also

- css
- nginx
- caddy
- npm
- web-attacks

## References

- [MDN HTML Reference](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference)
- [HTML Living Standard (WHATWG)](https://html.spec.whatwg.org/)
- [MDN HTML Elements Reference](https://developer.mozilla.org/en-US/docs/Web/HTML/Element)
- [MDN HTML Forms Guide](https://developer.mozilla.org/en-US/docs/Learn/Forms)
- [MDN HTML Input Types](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input)
- [MDN HTML Global Attributes](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes)
- [MDN Accessibility (ARIA)](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA)
- [W3C HTML Validator](https://validator.w3.org/)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [Can I Use (Browser Support Tables)](https://caniuse.com/)
- [Web Content Accessibility Guidelines (WCAG)](https://www.w3.org/WAI/standards-guidelines/wcag/)
