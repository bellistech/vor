# HTML (HyperText Markup Language)

Standard markup language for the web — document structure, semantics, accessibility, and embedded media. See html-forms for forms (input, validation, label, fieldset, etc.).

## Setup

No installation. HTML is parsed by every browser engine: Blink (Chrome/Edge), Gecko (Firefox), WebKit (Safari). The only "runtime" is a browser. Author files as `.html` (or `.htm`). Serve over HTTP(S) with `Content-Type: text/html; charset=UTF-8`.

```bash
# # Minimal canonical HTML5 page
# <!DOCTYPE html>
# <html lang="en">
# <head>
#   <meta charset="UTF-8">
#   <meta name="viewport" content="width=device-width, initial-scale=1">
#   <title>Hello, World</title>
# </head>
# <body>
#   <h1>Hello, World</h1>
# </body>
# </html>
```

```bash
# # Doctype is case-insensitive but the canonical form is uppercase DOCTYPE
# <!DOCTYPE html>          # # standards mode
# <!doctype html>          # # also valid (HTML5 is case-insensitive)
# (no doctype)             # # quirks mode — broken layout, avoid
```

```bash
# # The lang attribute is required for accessibility (screen readers pick voice)
# <html lang="en">         # # English
# <html lang="en-US">      # # English (United States)
# <html lang="ja">         # # Japanese
# <html lang="ar" dir="rtl"># # Arabic right-to-left
```

```bash
# # Serve locally for testing — any static server works
# python3 -m http.server 8000
# npx serve .
# php -S localhost:8000
# busybox httpd -f -p 8000
```

## Document Structure

The HTML document tree always has the same shape: `<html>` containing `<head>` (metadata, never rendered) and `<body>` (everything visible).

```bash
# # Canonical document tree
# <!DOCTYPE html>
# <html lang="en">
# <head>
#   <meta charset="UTF-8">
#   <meta name="viewport" content="width=device-width, initial-scale=1">
#   <meta name="description" content="One-line page summary for SEO">
#   <title>Title shown in browser tab and search results</title>
#   <link rel="icon" href="/favicon.ico">
#   <link rel="stylesheet" href="/styles.css">
# </head>
# <body>
#   <header>...</header>
#   <main>...</main>
#   <footer>...</footer>
#   <script src="/app.js" defer></script>
# </body>
# </html>
```

### Why each element matters

```bash
# <!DOCTYPE html>          # # opt out of quirks mode
# <html lang="...">        # # required for screen readers and language detection
# <meta charset="UTF-8">   # # MUST appear in first 1024 bytes
# <meta viewport>          # # mobile rendering — without it pages render at 980px wide
# <meta description>       # # search engine snippet (~155 chars)
# <title>                  # # tab text + search result heading
# <link rel="icon">        # # browser tab favicon
```

### Common boilerplate mistakes

```bash
# # BROKEN — missing lang, no viewport, charset late
# <!DOCTYPE html>
# <html>
# <head>
#   <title>My Page</title>
#   <meta charset="UTF-8">
# </head>

# # FIXED
# <!DOCTYPE html>
# <html lang="en">
# <head>
#   <meta charset="UTF-8">
#   <meta name="viewport" content="width=device-width, initial-scale=1">
#   <title>My Page</title>
# </head>
```

## Head Element Catalog

The `<head>` is a metadata container. None of its content is rendered (except `<title>` in the tab and `<noscript>` in fallback contexts).

### meta charset and viewport

```bash
# <meta charset="UTF-8">                           # # always UTF-8, always first
# <meta name="viewport"                            # # mobile responsive
#       content="width=device-width, initial-scale=1">
# <meta name="viewport"                            # # disable user zoom (avoid for accessibility)
#       content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
```

### meta name (page-level metadata)

```bash
# <meta name="description"     content="Page summary, ~155 chars max for SEO snippets">
# <meta name="keywords"        content="ignored by Google since 2009">
# <meta name="author"          content="Jane Doe">
# <meta name="generator"       content="Hugo 0.120">
# <meta name="application-name"content="My Web App">
# <meta name="theme-color"     content="#1a73e8">             # # browser chrome color
# <meta name="theme-color"     content="#000000" media="(prefers-color-scheme: dark)">
# <meta name="color-scheme"    content="light dark">           # # native form controls
# <meta name="referrer"        content="strict-origin-when-cross-origin">
# <meta name="robots"          content="index, follow">        # # search engine directive
# <meta name="robots"          content="noindex, nofollow">    # # block indexing
# <meta name="googlebot"       content="noarchive, nosnippet">
# <meta name="format-detection" content="telephone=no">        # # Safari iOS auto-tel
```

### meta http-equiv

```bash
# <meta http-equiv="refresh"          content="5; url=/new-page">  # # auto redirect
# <meta http-equiv="content-language" content="en">
# <meta http-equiv="X-UA-Compatible"  content="IE=edge">           # # legacy IE only
# <meta http-equiv="Content-Security-Policy"
#       content="default-src 'self'">                              # # PREFER server header
```

### Open Graph (Facebook, LinkedIn, Slack, Discord)

```bash
# <meta property="og:title"        content="Article title">
# <meta property="og:description"  content="One-paragraph summary">
# <meta property="og:image"        content="https://example.com/share.png">
# <meta property="og:image:alt"    content="Alt text for the share image">
# <meta property="og:image:width"  content="1200">
# <meta property="og:image:height" content="630">
# <meta property="og:url"          content="https://example.com/article">
# <meta property="og:type"         content="article">
# <meta property="og:site_name"    content="My Site">
# <meta property="og:locale"       content="en_US">
```

### Twitter / X cards

```bash
# <meta name="twitter:card"        content="summary_large_image">
# <meta name="twitter:site"        content="@mysite">
# <meta name="twitter:creator"     content="@author">
# <meta name="twitter:title"       content="Article title">
# <meta name="twitter:description" content="Summary">
# <meta name="twitter:image"       content="https://example.com/share.png">
# <meta name="twitter:image:alt"   content="Alt text">
```

### link rel catalog

```bash
# <link rel="stylesheet"   href="/styles.css">                      # # CSS
# <link rel="stylesheet"   href="/print.css" media="print">         # # only-print CSS
# <link rel="icon"         href="/favicon.ico">                     # # browser tab
# <link rel="icon"         href="/icon-192.png" sizes="192x192" type="image/png">
# <link rel="apple-touch-icon" href="/apple-touch-icon.png">        # # iOS home screen
# <link rel="manifest"     href="/manifest.webmanifest">            # # PWA install
# <link rel="canonical"    href="https://example.com/page">         # # preferred URL
# <link rel="alternate"    href="https://example.com/feed.xml" type="application/rss+xml">
# <link rel="alternate"    href="https://example.com/es/" hreflang="es">
# <link rel="preconnect"   href="https://fonts.googleapis.com">     # # warm DNS+TCP+TLS
# <link rel="dns-prefetch" href="https://api.example.com">          # # warm DNS only
# <link rel="preload"      href="/hero.jpg" as="image">              # # critical resource
# <link rel="preload"      href="/inter.woff2" as="font" type="font/woff2" crossorigin>
# <link rel="prefetch"     href="/next-page.html">                  # # likely-needed
# <link rel="modulepreload" href="/components/header.js">           # # ES module preload
# <link rel="search"       href="/opensearch.xml" type="application/opensearchdescription+xml">
# <link rel="author"       href="/about">
# <link rel="license"      href="/LICENSE">
```

### preload caveats

```bash
# # BROKEN — preload without crossorigin on fonts breaks
# <link rel="preload" href="/font.woff2" as="font">
# # FIXED
# <link rel="preload" href="/font.woff2" as="font" type="font/woff2" crossorigin>
```

## Headings

Six levels: `h1` (most important) through `h6` (least). Headings build the document outline that screen readers, search engines, and reader-mode views consume.

```bash
# <h1>Page or app primary title</h1>
# <h2>Top-level section</h2>
# <h3>Subsection</h3>
# <h4>Sub-subsection</h4>
# <h5>Rare</h5>
# <h6>Almost never used</h6>
```

### Outline algorithm

The original "HTML5 outline algorithm" — where `<section>` was supposed to scope `h1` like a fresh hierarchy — was **never implemented in any browser** and was removed from the spec. Use h1–h6 explicitly to express hierarchy.

```bash
# # WRONG (assumed sectioning would scope h1)
# <section><h1>About</h1></section>
# <section><h1>Contact</h1></section>
# # All become h1 — flat, no hierarchy.

# # RIGHT — use real heading levels
# <h1>Site or page title</h1>
# <section><h2>About</h2></section>
# <section><h2>Contact</h2></section>
```

### One h1 per page debate

WHATWG spec now allows multiple h1s (especially with sectioning roots). WAI-ARIA Authoring Practices and most accessibility audits still recommend exactly one h1 per page. Pragmatic rule: one h1 per page, no skipping levels (h1 → h2 → h3, never h1 → h3).

```bash
# # BROKEN — skipped level
# <h1>Title</h1>
# <h3>Subhead</h3>            # # screen reader announces "level 3" with no level 2

# # FIXED
# <h1>Title</h1>
# <h2>Subhead</h2>
```

### Tools to inspect outline

```bash
# # Chrome DevTools — Accessibility panel → Document outline
# # Firefox — Accessibility inspector → tab "Show tabbing order"
# # Browser extension: HTML5 Outliner, headingsMap
```

## Text-Level Semantics

```bash
# <p>Paragraph of text.</p>
# <br>                              # # line break (don't use for spacing — that's CSS)
# <hr>                              # # thematic break (story shift, scene change)
# <em>emphasis (italic by default)</em>
# <strong>strong importance (bold by default)</strong>
# <i>technical term, foreign phrase, thought, ship name (italic, no semantic emphasis)</i>
# <b>keyword in summary, product name (bold, no semantic importance)</b>
# <mark>highlighted/relevant text</mark>
# <small>fine print, side comment</small>
# <ins>inserted text (underlined)</ins>
# <del>deleted text (strikethrough)</del>
# <s>no longer accurate (strikethrough, not deletion)</s>
# <u>misspelling, proper name in CJK (underline, rarely used)</u>
# <sup>superscript</sup> and <sub>subscript</sub>
# <cite>title of a work</cite>
# <q>inline quotation</q>           # # browser adds curly quotes
# <blockquote cite="https://...">block quotation</blockquote>
# <pre>preformatted text</pre>      # # preserves whitespace and newlines
# <code>inline code</code>
# <kbd>Ctrl</kbd>+<kbd>C</kbd>      # # keyboard input
# <samp>program output</samp>
# <var>x</var> = <var>y</var> + 1   # # mathematical variable
# <time datetime="2026-04-25">April 25, 2026</time>
# <time datetime="2026-04-25T14:30:00Z">2:30 PM UTC</time>
# <abbr title="HyperText Markup Language">HTML</abbr>
# <dfn>defining instance of a term</dfn>
# <bdi>isolated bidirectional text</bdi>
# <bdo dir="rtl">override direction</bdo>
# <ruby>漢<rp>(</rp><rt>kan</rt><rp>)</rp></ruby>  # # East Asian annotation
# <wbr>                             # # optional word break opportunity (long URLs)
```

### em vs i, strong vs b

```bash
# <em>     # # SEMANTIC stress emphasis ("She *was* there")
# <i>      # # offset text without emphasis (foreign phrase, ship names, taxonomy)
# <strong> # # SEMANTIC strong importance (warnings, key term in a heading)
# <b>      # # offset text without importance (keywords in a summary, product names)
```

### time element

```bash
# <time datetime="2026">2026</time>
# <time datetime="2026-04">April 2026</time>
# <time datetime="2026-04-25">April 25, 2026</time>
# <time datetime="14:30">2:30 PM</time>
# <time datetime="2026-04-25T14:30Z">April 25 at 2:30 PM UTC</time>
# <time datetime="P2DT3H">2 days, 3 hours</time>             # # ISO 8601 duration
```

### blockquote vs q

```bash
# <p>Per the docs, <q cite="https://example.com">use semantic markup</q>.</p>
# <blockquote cite="https://example.com">
#   <p>Use semantic markup.</p>
#   <footer>— <cite>Example Style Guide</cite></footer>
# </blockquote>
```

## Lists

```bash
# # Unordered list
# <ul>
#   <li>First</li>
#   <li>Second</li>
# </ul>

# # Ordered list
# <ol>
#   <li>Step one</li>
#   <li>Step two</li>
# </ol>

# # ol attributes
# <ol type="1">      # # 1, 2, 3 (default)
# <ol type="a">      # # a, b, c
# <ol type="A">      # # A, B, C
# <ol type="i">      # # i, ii, iii
# <ol type="I">      # # I, II, III
# <ol start="5">     # # start at 5
# <ol reversed>      # # 3, 2, 1
# <ol start="10" reversed>

# # value on a single li
# <ol>
#   <li>One</li>
#   <li value="5">Jumps to 5</li>
#   <li>Six</li>
# </ol>
```

### Description list

```bash
# <dl>
#   <dt>HTML</dt>
#   <dd>HyperText Markup Language</dd>
#
#   <dt>CSS</dt>
#   <dt>Cascading Style Sheets</dt>     # # multiple terms allowed
#   <dd>Stylesheet language</dd>
#   <dd>Used to style HTML</dd>         # # multiple descriptions allowed
# </dl>
```

### Nested lists

```bash
# <ul>
#   <li>Fruits
#     <ul>
#       <li>Apple</li>
#       <li>Banana</li>
#     </ul>
#   </li>
#   <li>Vegetables</li>
# </ul>

# # Nesting goes inside <li>, NEVER directly inside <ul>:
# # BROKEN
# <ul>
#   <li>Fruits</li>
#   <ul><li>Apple</li></ul>     # # invalid — <ul> child of <ul>
# </ul>
```

## Links

```bash
# <a href="https://example.com">External absolute</a>
# <a href="/about">Internal absolute path</a>
# <a href="about.html">Relative</a>
# <a href="../parent">Relative parent dir</a>
# <a href="#section-id">Same-page fragment</a>
# <a href="page.html#section">Fragment on another page</a>
# <a href="mailto:user@example.com">Email</a>
# <a href="mailto:user@example.com?subject=Hi&body=Hello">With subject and body</a>
# <a href="tel:+15551234567">Phone</a>
# <a href="sms:+15551234567">SMS</a>
# <a href="sms:+15551234567?body=Hi">SMS with body</a>
# <a href="javascript:void(0)">avoid this</a>      # # use <button> for actions
```

### target attribute

```bash
# <a href="..." target="_self">Same tab (default)</a>
# <a href="..." target="_blank">New tab</a>
# <a href="..." target="_parent">Parent frame</a>
# <a href="..." target="_top">Top-most frame</a>
# <a href="..." target="window-name">Named window (reuses if exists)</a>
```

### target=_blank security

`target="_blank"` without `rel="noopener"` lets the new tab call `window.opener.location = "phishing-site"`. Modern browsers (since Chrome 88, Firefox 79, Safari 12.1) imply `rel="noopener"` when `target="_blank"`, but explicit is still recommended.

```bash
# <a href="https://external.com" target="_blank" rel="noopener noreferrer">Safe</a>
# # noopener: prevents window.opener access
# # noreferrer: also strips Referer header (and implies noopener)
```

### download attribute

```bash
# <a href="/files/report.pdf" download>Download (uses original name)</a>
# <a href="/files/report.pdf" download="my-report.pdf">Custom filename</a>
# # download only works same-origin (or via Content-Disposition header)
```

### Other link attributes

```bash
# <a href="..." hreflang="es">Spanish version</a>
# <a href="..." type="application/pdf">PDF (hint, not enforced)</a>
# <a href="..." ping="https://analytics.example.com/track">Tracked click</a>
# <a href="..." referrerpolicy="no-referrer">No Referer header</a>
```

### rel attribute catalog

```bash
# rel="alternate"     # # alternate version (RSS feed, translated page)
# rel="author"        # # author of the page
# rel="bookmark"      # # permalink for nearest <article>
# rel="canonical"     # # preferred URL (used on <link>)
# rel="external"      # # external resource
# rel="help"          # # help document
# rel="license"       # # license terms
# rel="next" / "prev" # # paginated series
# rel="nofollow"      # # search engines should not follow
# rel="noopener"      # # block window.opener access
# rel="noreferrer"    # # strip Referer header
# rel="opener"        # # opt back IN to opener access (rare)
# rel="search"        # # search resource for the document
# rel="tag"           # # tag for the page
# rel="ugc"           # # user-generated content (anti-spam hint)
# rel="sponsored"     # # paid/sponsored link
```

### Relative vs absolute URLs

```bash
# https://example.com/path        # # absolute (full URL)
# //example.com/path              # # protocol-relative (rare; avoid)
# /path                           # # root-relative (recommended for internal)
# path                            # # document-relative
# ./path                          # # explicit current directory
# ../path                         # # parent directory
# #fragment                       # # same document, jump to id
```

## Images

```bash
# <img src="/photo.jpg" alt="Description of image content" width="800" height="600">
```

### Required attributes

```bash
# src      — URL of the image
# alt      — text alternative for accessibility (REQUIRED)
# width    — intrinsic width in pixels (NO unit)
# height   — intrinsic height in pixels (NO unit)
```

### alt text rules

```bash
# # Decorative image — empty alt (NOT missing alt)
# <img src="/divider.svg" alt="">

# # Functional image (icon-only link/button) — describe the function
# <a href="/cart"><img src="/cart-icon.svg" alt="Shopping cart"></a>

# # Informative image — describe the content
# <img src="/chart.png" alt="Sales rose 23% from Q1 to Q2">

# # Image of text (avoid this) — alt = the text
# <img src="/headline.png" alt="Annual Report 2026">

# # BROKEN
# <img src="/photo.jpg">                    # # missing alt
# <img src="/photo.jpg" alt="image">        # # uninformative
# <img src="/photo.jpg" alt="photo.jpg">    # # alt = filename
```

### Width and height to prevent CLS

Always set width and height. Browsers compute aspect ratio (`height / width`) and reserve space, eliminating Cumulative Layout Shift.

```bash
# <img src="/hero.jpg" alt="..." width="1600" height="900">
# # CSS: img { max-width: 100%; height: auto; } — keeps it responsive
```

### Resolution switching with srcset

```bash
# <img src="/image-800w.jpg"
#      srcset="/image-400w.jpg 400w,
#              /image-800w.jpg 800w,
#              /image-1600w.jpg 1600w"
#      sizes="(max-width: 600px) 100vw, 800px"
#      alt="..." width="800" height="600">
```

### Pixel-density descriptors

```bash
# <img src="/logo.png"
#      srcset="/logo.png 1x, /logo@2x.png 2x, /logo@3x.png 3x"
#      alt="Company logo">
```

### picture element (art direction)

```bash
# <picture>
#   <source media="(max-width: 600px)" srcset="/hero-mobile.jpg">
#   <source media="(max-width: 1200px)" srcset="/hero-tablet.jpg">
#   <img src="/hero-desktop.jpg" alt="..." width="1600" height="900">
# </picture>

# # Format negotiation (AVIF → WebP → JPEG)
# <picture>
#   <source type="image/avif" srcset="/hero.avif">
#   <source type="image/webp" srcset="/hero.webp">
#   <img src="/hero.jpg" alt="..." width="1600" height="900">
# </picture>
```

### Loading and priority

```bash
# <img src="..." loading="lazy" decoding="async">     # # offscreen images
# <img src="..." loading="eager" fetchpriority="high">  # # LCP image / above the fold
# <img src="..." referrerpolicy="no-referrer">
# <img src="..." crossorigin="anonymous">
```

### loading=lazy gotcha

```bash
# # BROKEN — lazy on hero image delays LCP
# <img src="/hero.jpg" loading="lazy" alt="Hero">

# # FIXED — only lazy-load below-the-fold
# <img src="/hero.jpg" fetchpriority="high" alt="Hero">
# <img src="/below-fold.jpg" loading="lazy" alt="...">
```

## Audio and Video

```bash
# <video src="/movie.mp4" controls width="640" height="360" poster="/thumb.jpg">
#   Your browser does not support video.
# </video>

# # Multiple sources for format fallback
# <video controls width="640" height="360" poster="/thumb.jpg" preload="metadata">
#   <source src="/movie.webm" type="video/webm">
#   <source src="/movie.mp4"  type="video/mp4">
#   <track kind="subtitles" src="/movie.en.vtt" srclang="en" label="English" default>
#   <track kind="subtitles" src="/movie.es.vtt" srclang="es" label="Español">
#   <track kind="captions"  src="/movie.cap.vtt" srclang="en">
#   <track kind="descriptions" src="/movie.desc.vtt" srclang="en">
#   <track kind="chapters"  src="/movie.chap.vtt" srclang="en">
#   Your browser does not support HTML5 video.
# </video>
```

### Video attributes

```bash
# controls         # # show native player controls
# autoplay         # # auto-play (REQUIRES muted on most browsers)
# muted            # # start muted
# loop             # # restart at end
# poster="/img"    # # placeholder before play
# preload="none"   # # don't preload anything
# preload="metadata"# # preload duration/dimensions only
# preload="auto"   # # preload entire file (browser may ignore)
# playsinline      # # iOS: don't go fullscreen on play
# disablepictureinpicture  # # disable PiP
# crossorigin="anonymous"
```

### Autoplay rules (browsers since 2018)

```bash
# # WORKS — muted autoplay
# <video src="/clip.mp4" autoplay muted loop playsinline></video>

# # BLOCKED — autoplay with sound (user must click first)
# <video src="/clip.mp4" autoplay></video>
```

### Audio

```bash
# <audio controls preload="none">
#   <source src="/song.opus" type="audio/ogg; codecs=opus">
#   <source src="/song.mp3"  type="audio/mpeg">
#   Your browser does not support audio.
# </audio>
```

### track captions

```bash
# # WebVTT file (.vtt)
# WEBVTT
#
# 00:00:00.000 --> 00:00:04.000
# First caption line.
#
# 00:00:04.500 --> 00:00:08.000
# Second caption line.
```

### Canonical responsive video pattern

```bash
# <div style="aspect-ratio: 16/9;">
#   <video src="/movie.mp4" controls
#          style="width: 100%; height: 100%; object-fit: contain;"
#          poster="/thumb.jpg" preload="metadata"></video>
# </div>
```

## Tables

```bash
# <table>
#   <caption>Quarterly sales</caption>
#   <colgroup>
#     <col span="1">
#     <col span="2" style="background: #f0f0f0">
#   </colgroup>
#   <thead>
#     <tr>
#       <th scope="col">Quarter</th>
#       <th scope="col">Revenue</th>
#       <th scope="col">Growth</th>
#     </tr>
#   </thead>
#   <tbody>
#     <tr>
#       <th scope="row">Q1</th>
#       <td>$1.2M</td>
#       <td>+8%</td>
#     </tr>
#     <tr>
#       <th scope="row">Q2</th>
#       <td>$1.5M</td>
#       <td>+25%</td>
#     </tr>
#   </tbody>
#   <tfoot>
#     <tr>
#       <th scope="row">Total</th>
#       <td>$2.7M</td>
#       <td>+17%</td>
#     </tr>
#   </tfoot>
# </table>
```

### Spanning cells

```bash
# <td colspan="2">Spans two columns</td>
# <td rowspan="3">Spans three rows</td>
# <td colspan="2" rowspan="2">2x2 cell</td>
```

### scope attribute

```bash
# <th scope="col">     # # header for a column
# <th scope="row">     # # header for a row
# <th scope="colgroup">  # # spans a group of columns
# <th scope="rowgroup">  # # spans a group of rows
```

### headers attribute (complex tables)

```bash
# # When scope isn't enough (irregular table layouts)
# <table>
#   <tr>
#     <th id="name">Name</th>
#     <th id="q1">Q1</th>
#     <th id="q2">Q2</th>
#   </tr>
#   <tr>
#     <th id="alice" headers="name">Alice</th>
#     <td headers="alice q1">$100k</td>
#     <td headers="alice q2">$120k</td>
#   </tr>
# </table>
```

### Tables for tabular data only

```bash
# # BROKEN — using table for layout
# <table><tr><td>sidebar</td><td>main</td></tr></table>
# # FIXED — use CSS grid/flexbox
# <div class="layout"><aside>...</aside><main>...</main></div>
```

### Accessible data table checklist

```bash
# 1. <caption> describes the table.
# 2. <th scope="col"> on column headers.
# 3. <th scope="row"> on first cell of each data row (when meaningful).
# 4. <thead>, <tbody>, <tfoot> structure for long tables.
# 5. NEVER use <table> for layout.
```

## Semantic Sectioning

```bash
# <body>
#   <header>          <!-- top-of-page banner: logo, site nav -->
#     <nav>
#       <ul>
#         <li><a href="/">Home</a></li>
#         <li><a href="/about">About</a></li>
#       </ul>
#     </nav>
#   </header>
#
#   <main>            <!-- the main content (EXACTLY ONE per page) -->
#     <article>       <!-- self-contained piece (blog post, news story) -->
#       <header>
#         <h1>Article title</h1>
#         <p>By <a rel="author" href="/authors/jane">Jane</a> on
#            <time datetime="2026-04-25">April 25, 2026</time></p>
#       </header>
#
#       <section>     <!-- thematic group, MUST have heading -->
#         <h2>Background</h2>
#         <p>...</p>
#       </section>
#
#       <aside>       <!-- tangentially related (sidebar, callout) -->
#         <h2>Related</h2>
#       </aside>
#
#       <footer>
#         <p>Tags: <a rel="tag" href="...">html</a></p>
#       </footer>
#     </article>
#   </main>
#
#   <footer>          <!-- site footer: copyright, contact -->
#     <p>&copy; 2026 Example Corp.</p>
#   </footer>
# </body>
```

### Sectioning elements

```bash
# <header>      # # banner content (one per <article>, one for <body>)
# <nav>         # # major navigation block (NOT every group of links)
# <main>        # # primary content; exactly ONE per page
# <article>     # # self-contained, redistributable (RSS, syndication)
# <section>     # # thematic group; SHOULD have a heading
# <aside>       # # tangentially related; sidebar, callout, footnote
# <footer>      # # footer for nearest sectioning ancestor
# <search>      # # search form region (HTML 2024+)
# <figure>      # # standalone media with optional caption
# <figcaption>  # # caption for <figure>
# <details>     # # collapsible disclosure widget
# <summary>     # # heading for <details>
# <dialog>      # # modal/non-modal dialog
```

### figure with figcaption

```bash
# <figure>
#   <img src="/chart.png" alt="Revenue grew 40% YoY" width="800" height="400">
#   <figcaption>Figure 1. Revenue growth, 2024–2026.</figcaption>
# </figure>
#
# <figure>
#   <pre><code>console.log("hi")</code></pre>
#   <figcaption>Listing 2. Greeting in JavaScript.</figcaption>
# </figure>
```

### details / summary

```bash
# <details>
#   <summary>Click to expand</summary>
#   <p>Hidden content revealed on toggle.</p>
# </details>
#
# <details open>      # # initially expanded
#   <summary>FAQ: how do I cancel?</summary>
#   <p>Email support@example.com.</p>
# </details>
#
# <details name="accordion-1"><summary>Item A</summary>...</details>
# <details name="accordion-1"><summary>Item B</summary>...</details>
# # name attribute (2024+) — only one open at a time
```

### dialog element

```bash
# <dialog id="confirm">
#   <form method="dialog">
#     <p>Are you sure?</p>
#     <button value="cancel">Cancel</button>
#     <button value="confirm">Yes</button>
#   </form>
# </dialog>
#
# <button onclick="confirm.showModal()">Delete</button>
# # JS:
# # dialog.show()       — non-modal
# # dialog.showModal()  — modal (with backdrop, traps focus, blocks page)
# # dialog.close(value) — close with returnValue
# # CSS ::backdrop selects the backdrop layer
```

### main element rules

```bash
# # WRONG — multiple visible <main>
# <main>...</main>
# <main>...</main>

# # OK — only one IS visible at a time (others have hidden)
# <main hidden>...</main>
# <main>...</main>
```

## Generic Containers

```bash
# <div>             # # block-level generic container, no semantics
# <span>            # # inline generic container, no semantics
```

### When to use which

```bash
# # Use <div> when no semantic element fits AND you need a styling/layout hook.
# # Use <span> for the same, but inline.
# # Always reach for semantic elements FIRST.

# # BROKEN — div soup
# <div class="header">
#   <div class="nav">
#     <div class="link"><a href="/">Home</a></div>
#   </div>
# </div>

# # FIXED — semantic markup
# <header>
#   <nav>
#     <ul><li><a href="/">Home</a></li></ul>
#   </nav>
# </header>
```

## Embedded Content

### iframe

```bash
# <iframe src="https://example.com/embed"
#         width="640" height="360"
#         title="Required descriptive title"
#         loading="lazy"
#         referrerpolicy="no-referrer-when-downgrade"
#         allow="fullscreen; clipboard-write"
#         sandbox="allow-scripts allow-same-origin"
#         allowfullscreen></iframe>
```

### iframe attributes

```bash
# src              # # URL to embed
# srcdoc           # # inline HTML content (overrides src)
# title            # # accessible name (REQUIRED)
# width/height
# loading="lazy"   # # offscreen lazy-load
# referrerpolicy
# allow="..."      # # feature policy (camera, mic, fullscreen)
# allowfullscreen  # # allow Fullscreen API
# sandbox          # # security restrictions (see below)
# name             # # frame name (target attribute on links)
```

### iframe sandbox

`sandbox` with no value applies ALL restrictions. Add tokens to selectively allow.

```bash
# <iframe sandbox></iframe>                          # # max restrictions
# <iframe sandbox="allow-scripts"></iframe>          # # allow JS only
# <iframe sandbox="allow-scripts allow-same-origin"> # # JS + same-origin (DANGER pair)
```

### sandbox tokens

```bash
# allow-downloads
# allow-forms
# allow-modals
# allow-orientation-lock
# allow-pointer-lock
# allow-popups
# allow-popups-to-escape-sandbox
# allow-presentation
# allow-same-origin
# allow-scripts
# allow-storage-access-by-user-activation
# allow-top-navigation
# allow-top-navigation-by-user-activation
# allow-top-navigation-to-custom-protocols
```

### sandbox security gotcha

```bash
# # DANGER: combining allow-scripts + allow-same-origin lets the iframe
# # remove its own sandbox via DOM manipulation.
# <iframe sandbox="allow-scripts allow-same-origin" src="/untrusted.html"></iframe>
```

### embed and object (rarely used)

```bash
# <embed src="/file.pdf" type="application/pdf" width="600" height="800">
# <object data="/file.pdf" type="application/pdf" width="600" height="800">
#   <p>Fallback if PDF can't render.</p>
# </object>
```

### canvas

```bash
# <canvas id="game" width="800" height="600">
#   Fallback content for browsers without canvas (rare).
# </canvas>
# # Drawn via JS: canvas.getContext('2d') or 'webgl' or 'webgpu'
```

### Inline SVG

```bash
# <svg xmlns="http://www.w3.org/2000/svg"
#      viewBox="0 0 100 100"
#      width="100" height="100"
#      role="img" aria-label="Logo">
#   <circle cx="50" cy="50" r="40" fill="currentColor"/>
# </svg>
```

### MathML

```bash
# <math xmlns="http://www.w3.org/1998/Math/MathML">
#   <mfrac>
#     <mi>x</mi>
#     <mn>2</mn>
#   </mfrac>
# </math>
```

## Inline SVG

```bash
# <svg xmlns="http://www.w3.org/2000/svg"
#      viewBox="0 0 200 100"
#      role="img"
#      aria-label="Sales chart">
#   <title>Sales chart</title>
#   <desc>A bar chart showing quarterly sales.</desc>
#
#   <rect x="10" y="20" width="40" height="60" fill="steelblue"/>
#   <circle cx="100" cy="50" r="20" fill="tomato"/>
#   <line x1="0" y1="50" x2="200" y2="50" stroke="black" stroke-width="2"/>
#   <path d="M 10 80 L 50 20 L 100 80" stroke="green" fill="none"/>
#   <text x="100" y="90" text-anchor="middle" font-size="12">Quarter</text>
#   <polygon points="150,20 170,40 150,60" fill="purple"/>
#   <polyline points="0,0 10,10 20,5" stroke="orange" fill="none"/>
#   <ellipse cx="50" cy="50" rx="30" ry="15" fill="cyan"/>
# </svg>
```

### viewBox and preserveAspectRatio

```bash
# viewBox="min-x min-y width height"
# # The internal coordinate system. viewBox="0 0 100 100" means SVG
# # is 100×100 in its own units, regardless of width/height attributes.
#
# preserveAspectRatio="xMidYMid meet"   # # default: scale to fit, preserve ratio
# preserveAspectRatio="xMidYMid slice"  # # scale to fill, crop overflow
# preserveAspectRatio="none"            # # stretch to fit (distorts)
```

### Reusable defs / use

```bash
# <svg>
#   <defs>
#     <symbol id="icon-check" viewBox="0 0 24 24">
#       <path d="M5 12l5 5L20 7" stroke="currentColor" fill="none" stroke-width="2"/>
#     </symbol>
#   </defs>
#   <use href="#icon-check" width="24" height="24"/>
#   <use href="#icon-check" width="48" height="48" x="50"/>
# </svg>
```

### Styling SVG with CSS

```bash
# <svg class="icon" viewBox="0 0 24 24">
#   <circle cx="12" cy="12" r="10"/>
# </svg>
# /* CSS */
# .icon { width: 24px; height: 24px; }
# .icon circle { fill: currentColor; stroke: red; stroke-width: 2; }
```

### SVG accessibility

```bash
# # Decorative
# <svg aria-hidden="true">...</svg>

# # Informative
# <svg role="img" aria-label="Search">
#   <title>Search</title>
#   ...
# </svg>

# # Complex (chart, diagram)
# <svg role="img" aria-labelledby="title desc">
#   <title id="title">Q1–Q4 sales</title>
#   <desc id="desc">Sales rose steadily from $1M to $1.8M across the year.</desc>
#   ...
# </svg>
```

## Custom Data Attributes

`data-*` attributes are valid HTML and surface on `element.dataset` in JS (kebab-case → camelCase).

```bash
# <article data-id="42" data-author-id="7" data-published-at="2026-04-25">
#   ...
# </article>
```

### JS access

```bash
# const el = document.querySelector('article');
# el.dataset.id;            // "42"
# el.dataset.authorId;      // "7"  (data-author-id → authorId)
# el.dataset.publishedAt;   // "2026-04-25"
# el.dataset.newKey = 'x';  // <article data-new-key="x">
# delete el.dataset.id;     // remove
```

### CSS access

```bash
# /* attribute selector */
# [data-state="active"]   { color: green; }
# [data-state="error"]    { color: red; }
# [data-id]               { font-weight: bold; }   /* attribute exists */
# [data-tag~="featured"]  { border: 2px solid; }   /* word-list match */

# /* CSS attr() */
# .tooltip::after { content: attr(data-tooltip); }
```

### Naming rules

```bash
# # Allowed: lowercase letters, digits, hyphens
# data-user-id        ✓
# data-page_number    ✗ (no underscores)
# data-User-Id        ✗ (no uppercase in name)
# data-                ✗ (must have at least one character after data-)
```

## Microdata / Schema.org

Two ways to embed structured data: HTML microdata (`itemscope`/`itemprop`) or JSON-LD. **JSON-LD is preferred** by Google and easier to maintain (separate from markup).

### JSON-LD (recommended)

```bash
# <script type="application/ld+json">
# {
#   "@context": "https://schema.org",
#   "@type": "Article",
#   "headline": "How to learn HTML",
#   "author": {
#     "@type": "Person",
#     "name": "Jane Doe"
#   },
#   "datePublished": "2026-04-25",
#   "image": "https://example.com/cover.jpg"
# }
# </script>
```

### Microdata (legacy, but valid)

```bash
# <article itemscope itemtype="https://schema.org/Article">
#   <h1 itemprop="headline">How to learn HTML</h1>
#   <p>By
#     <span itemprop="author" itemscope itemtype="https://schema.org/Person">
#       <span itemprop="name">Jane Doe</span>
#     </span>
#     on <time itemprop="datePublished" datetime="2026-04-25">April 25</time>
#   </p>
#   <img itemprop="image" src="/cover.jpg" alt="Cover">
# </article>
```

### Common schemas

```bash
# Article, BlogPosting, NewsArticle
# Product, Offer, AggregateRating, Review
# Person, Organization, LocalBusiness
# BreadcrumbList, FAQPage, HowTo
# Recipe, Event, VideoObject
# WebSite (with SearchAction for sitelinks search box)
```

### Test with

```bash
# Google Rich Results Test:  https://search.google.com/test/rich-results
# Schema.org validator:      https://validator.schema.org/
```

## Boolean and Enumerated Attributes

### Boolean attributes

Presence = true. Value (if any) is ignored. The "false" form is to omit the attribute entirely.

```bash
# # All equivalent (all "true")
# <input disabled>
# <input disabled="">
# <input disabled="disabled">
# <input disabled="true">      # # works but misleading
# <input disabled="false">     # # STILL TRUE — presence = true

# # The list of HTML5 boolean attributes
# allowfullscreen   async         autofocus      autoplay
# checked           controls      default        defer
# disabled          formnovalidate hidden        inert
# ismap             itemscope     loop           multiple
# muted             nomodule      novalidate     open
# playsinline       readonly      required       reversed
# selected
```

### Common boolean gotcha

```bash
# # BROKEN
# <button disabled="false">Submit</button>     # # disabled (value ignored)

# # FIXED — omit the attribute
# <button>Submit</button>
```

### Enumerated attributes

Take a fixed set of string values. Invalid values fall back to a default.

```bash
# # crossorigin: "" | "anonymous" | "use-credentials"
# <script src="..." crossorigin="anonymous"></script>
# <script src="..." crossorigin="use-credentials"></script>

# # referrerpolicy: no-referrer | no-referrer-when-downgrade | origin |
# #                 origin-when-cross-origin | same-origin | strict-origin |
# #                 strict-origin-when-cross-origin | unsafe-url

# # loading: eager | lazy
# # decoding: sync | async | auto
# # fetchpriority: high | low | auto
# # contenteditable: true | false | plaintext-only
# # dir: ltr | rtl | auto
# # spellcheck: true | false
# # translate: yes | no
# # autocapitalize: off | on | sentences | words | characters
# # draggable: true | false
# # popover: auto | manual | hint
# # type (button): submit | reset | button
```

## Global Attributes

Apply to (almost) every element.

```bash
# id="unique"            # # one per document; used by anchors, JS, label[for], aria-*by
# class="card primary"   # # space-separated; CSS/JS hooks
# style="color: red"     # # inline CSS (avoid except for dynamic values)
# title="Hover tooltip"  # # tooltip; bad for mobile/keyboard users
# lang="en-US"           # # language of element subtree
# dir="rtl"              # # text direction
# hidden                 # # hidden from rendering AND a11y tree (display:none)
# hidden="until-found"   # # findable via Ctrl-F (modern browsers)
# inert                  # # un-clickable, un-tabbable, un-readable by AT
# contenteditable        # # makes element editable
# draggable="true"       # # native drag-and-drop source
# spellcheck="true"      # # browser spellcheck
# translate="no"         # # exclude from Google Translate
# autocapitalize="words" # # mobile keyboard hint (also on inputs)
# popover="auto"         # # popover API (2024+)
# tabindex="0"           # # focus order
# accesskey="s"          # # keyboard shortcut (Alt+S, varies by browser)
# slot="header"          # # web components slot assignment
# enterkeyhint="search"  # # mobile Enter key label (on contenteditable/input)
# is="my-element"        # # customized built-in element
# role="..."             # # ARIA role override (use sparingly)
# aria-*="..."           # # ARIA properties
# data-*="..."           # # custom data attributes
```

### hidden vs inert vs aria-hidden

```bash
# hidden          → display: none, removed from layout AND a11y tree
# inert           → still in layout, NOT focusable, NOT in a11y tree
# aria-hidden=true→ still in layout AND focusable, hidden from a11y tree only
# style="visibility:hidden" → in layout but invisible
# style="display:none"      → equivalent to hidden
```

### contenteditable

```bash
# <div contenteditable>Edit me</div>
# <div contenteditable="plaintext-only">No formatting</div>
# <div contenteditable="true" spellcheck="true"></div>
```

### tabindex

```bash
# tabindex="0"     # # focusable in source order
# tabindex="-1"    # # programmatically focusable (focus()) but NOT in tab order
# tabindex="1"     # # focused first (BAD — breaks natural order, AVOID)
# tabindex="2"+    # # higher = later in custom order (also BAD)
```

## Accessibility — ARIA Basics

WAI-ARIA (Accessible Rich Internet Applications) supplements semantic HTML when no native element fits.

### First rule of ARIA

> **Don't use ARIA when a semantic HTML element exists.**

```bash
# # BROKEN
# <div role="button" tabindex="0" onclick="...">Save</div>
# # FIXED
# <button>Save</button>
```

### Second rule: don't change native semantics unless you must

```bash
# # BROKEN
# <h1 role="button">Title</h1>           # # heading is no longer a heading
# # FIXED
# <button><h1>Title</h1></button>        # # not great either; rethink the design
```

### role attribute

```bash
# <div role="alert">Error</div>
# <div role="status">Saved</div>
# <div role="dialog" aria-modal="true">...</div>
# <div role="tablist">...</div>
# <div role="tab" aria-selected="true">...</div>
# <div role="tabpanel">...</div>
# <ul role="menu">...</ul>
# <li role="menuitem">...</li>
# <span role="img" aria-label="Cat">🐱</span>
```

### aria-label vs aria-labelledby

```bash
# # aria-label = string label
# <button aria-label="Close">×</button>

# # aria-labelledby = id reference (one or many)
# <h2 id="dialog-title">Confirm</h2>
# <div role="dialog" aria-labelledby="dialog-title">...</div>

# # Multiple labelledby ids concatenate
# <input aria-labelledby="label-1 label-2">
```

### aria-describedby

```bash
# <input id="pw" type="password" aria-describedby="pw-hint">
# <p id="pw-hint">Must be at least 8 characters.</p>
```

### aria-hidden

```bash
# <span aria-hidden="true">★</span>Rated 5 stars
# # The star is hidden from AT but visible to sighted users.

# # NEVER aria-hidden a focusable element — creates a "ghost focus"
# # BROKEN
# <button aria-hidden="true">Click me</button>     # # button still tabbable!
```

### aria-live regions

```bash
# # polite: announce when idle
# <div aria-live="polite" aria-atomic="true">2 of 10 items loaded</div>

# # assertive: interrupt current speech (for errors)
# <div aria-live="assertive" role="alert">Connection lost</div>

# # off: disabled (default)
# <div aria-live="off">...</div>

# # aria-atomic: read entire region on change vs only the changed part
# # aria-relevant: additions | removals | text | all
```

### aria-current

```bash
# <a href="/products" aria-current="page">Products</a>      # # current page
# <li aria-current="step">Step 3</li>                       # # current step
# <a aria-current="location">You are here</a>
# <a aria-current="date">Today</a>
# <a aria-current="time">Now</a>
# <a aria-current="true">Current item</a>
```

### aria-expanded, aria-controls

```bash
# <button aria-expanded="false" aria-controls="menu">Menu</button>
# <ul id="menu" hidden>...</ul>
# # Toggle aria-expanded and hidden via JS together.
```

### aria-pressed (toggle button)

```bash
# <button aria-pressed="false">Bold</button>
# # Toggle "true"/"false" on click.
```

### "No ARIA is better than bad ARIA"

```bash
# # WRONG — wrong role
# <a href="/" role="button">Home</a>             # # screen reader: "Home, button"
#                                                # # but Enter behavior is link, not button
# # RIGHT
# <a href="/">Home</a>
```

## Accessibility — Landmarks

Screen-reader users navigate by landmarks (NVDA: D, JAWS: R, VoiceOver: rotor).

```bash
# Native element       → ARIA role
# <header>            → banner       (only when child of <body>)
# <nav>               → navigation
# <main>              → main
# <aside>             → complementary
# <section>           → region        (only with accessible name)
# <article>           → article
# <footer>            → contentinfo  (only when child of <body>)
# <form>              → form          (only with accessible name)
# <search>            → search
# (no native)         → region
```

### Best practices

```bash
# # Use NATIVE elements first (they work without ARIA)
# <header>...</header>           # # ✓
# <div role="banner">...</div>   # # works but unnecessary

# # If multiple of same landmark, label them
# <nav aria-label="Primary">...</nav>
# <nav aria-label="Footer">...</nav>

# # Skip-link to main content (top of page)
# <a href="#main" class="skip-link">Skip to main content</a>
# ...
# <main id="main" tabindex="-1">...</main>
```

### section needs an accessible name

```bash
# # No name — NOT a landmark
# <section>...</section>

# # Has name — IS a landmark
# <section aria-labelledby="s-title">
#   <h2 id="s-title">News</h2>
# </section>
# <section aria-label="News">...</section>
```

## Accessibility — Focus Management

### What's focusable by default

```bash
# <a href="...">          ✓ (must have href)
# <button>                ✓
# <input>, <select>, <textarea>  ✓ (unless disabled)
# <area href="...">       ✓
# <iframe>                ✓
# elements with tabindex="0" or "-1"
# elements with contenteditable
```

### tabindex values

```bash
# tabindex="0"   # # add to natural tab order at element's source position
# tabindex="-1"  # # focusable via .focus() but NOT via Tab
# tabindex="1+"  # # AVOID — overrides natural order, accessibility nightmare
```

### :focus-visible (CSS)

```bash
# /* Show ring only on keyboard focus, not mouse click */
# button:focus-visible { outline: 2px solid blue; }
# button:focus { outline: none; }    /* DON'T do this without focus-visible */
```

### Common focus mistakes

```bash
# # BROKEN — outline: none with no replacement
# *:focus { outline: none; }

# # FIXED — provide a visible alternative
# *:focus-visible { outline: 2px solid currentColor; outline-offset: 2px; }
```

### Focus traps in dialogs

```bash
# # When a modal opens:
# 1. Save the previously focused element.
# 2. Move focus into the dialog (often the close button or first input).
# 3. Trap focus inside the dialog (Tab/Shift+Tab cycle stays inside).
# 4. On close, return focus to the saved element.

# # <dialog>.showModal() handles 2 and 3 natively.
```

### Focus on SPA route change

```bash
# // After client-side route change
# const heading = document.querySelector('h1');
# heading.setAttribute('tabindex', '-1');
# heading.focus();
# // Or focus a "skip to content" / <main> with tabindex=-1.
```

### autofocus pitfalls

```bash
# # BROKEN — autofocus on page load can disorient screen readers
# <input autofocus>

# # OK — autofocus inside a modal that just opened
# <dialog open><input autofocus></dialog>
```

## Web Components Basics

Custom elements let you define your own tags. Always namespaced (must contain a hyphen).

### Defining a custom element

```bash
# class MyButton extends HTMLElement {
#   constructor() {
#     super();
#   }
#
#   connectedCallback() {
#     // Called when the element is inserted into the DOM
#     this.innerHTML = `<button>${this.textContent}</button>`;
#   }
#
#   disconnectedCallback() {
#     // Cleanup when removed from the DOM
#   }
#
#   static get observedAttributes() { return ['label']; }
#
#   attributeChangedCallback(name, oldVal, newVal) {
#     // Called when an observed attribute changes
#   }
# }
#
# customElements.define('my-button', MyButton);
```

### Use it in HTML

```bash
# <my-button label="Save">Save</my-button>
```

### Customized built-in (less common)

```bash
# class FancyButton extends HTMLButtonElement { ... }
# customElements.define('fancy-button', FancyButton, { extends: 'button' });
# # HTML:
# <button is="fancy-button">Click</button>
# # NOTE: Safari does not support customized built-ins.
```

### Shadow DOM

```bash
# class MyCard extends HTMLElement {
#   constructor() {
#     super();
#     const shadow = this.attachShadow({ mode: 'open' });    // 'open' | 'closed'
#     shadow.innerHTML = `
#       <style>:host { display: block; padding: 1rem; }</style>
#       <slot></slot>
#     `;
#   }
# }
# customElements.define('my-card', MyCard);
```

### Shadow DOM scoping

```bash
# # Styles inside shadow root DO NOT leak out, and outer styles DO NOT leak in
# # (except inheritable properties like color, font, and CSS custom properties).
```

## Templates and Slots

### template element

```bash
# <template id="card-template">
#   <article class="card">
#     <h2></h2>
#     <p></p>
#   </article>
# </template>
#
# // JS
# const tpl = document.getElementById('card-template');
# const clone = tpl.content.cloneNode(true);
# clone.querySelector('h2').textContent = 'Title';
# clone.querySelector('p').textContent = 'Body';
# document.body.append(clone);
```

The contents of `<template>` are NOT rendered, NOT loaded (images don't fetch), and NOT executed (scripts don't run). Inert until cloned.

### slots

```bash
# # In the component (shadow DOM):
# <style>...</style>
# <header><slot name="title"></slot></header>
# <main><slot></slot></main>          <!-- default slot -->
# <footer><slot name="footer"></slot></footer>
#
# # When used:
# <my-card>
#   <h2 slot="title">Title</h2>        <!-- goes into slot name="title" -->
#   <p>Body content</p>                 <!-- goes into default slot -->
#   <button slot="footer">OK</button>
# </my-card>
```

## Document Metadata

### Canonical URL

```bash
# <link rel="canonical" href="https://example.com/page">
# # Tells search engines: even if there are URL variants
# # (?utm=, http vs https, /page vs /page/), this is THE URL.
```

### sitemap.xml hint

```bash
# # In robots.txt at site root
# User-agent: *
# Allow: /
# Sitemap: https://example.com/sitemap.xml
```

### robots.txt

```bash
# # /robots.txt at site root (NOT inside HTML)
# User-agent: *
# Disallow: /private/
# Disallow: /tmp/
# Allow: /
# Sitemap: https://example.com/sitemap.xml
```

### opensearch.xml

```bash
# <link rel="search"
#       type="application/opensearchdescription+xml"
#       title="My Site"
#       href="/opensearch.xml">
# # /opensearch.xml file describes search endpoint and lets browsers add it.
```

### manifest.json (PWA)

```bash
# <link rel="manifest" href="/manifest.webmanifest">
#
# # /manifest.webmanifest:
# {
#   "name": "My App",
#   "short_name": "App",
#   "start_url": "/",
#   "display": "standalone",
#   "background_color": "#ffffff",
#   "theme_color": "#1a73e8",
#   "icons": [
#     { "src": "/icon-192.png", "sizes": "192x192", "type": "image/png" },
#     { "src": "/icon-512.png", "sizes": "512x512", "type": "image/png" }
#   ]
# }
```

### Service worker registration

```bash
# # Done in JS, not HTML
# if ('serviceWorker' in navigator) {
#   navigator.serviceWorker.register('/sw.js');
# }
```

## Open Graph and Social Cards

The complete share-card meta block:

```bash
# <!-- Standard SEO -->
# <title>Article title</title>
# <meta name="description" content="One-paragraph summary">
# <link rel="canonical" href="https://example.com/article">
#
# <!-- Open Graph (Facebook, LinkedIn, Slack, Discord, iMessage) -->
# <meta property="og:type"        content="article">
# <meta property="og:title"       content="Article title">
# <meta property="og:description" content="One-paragraph summary">
# <meta property="og:url"         content="https://example.com/article">
# <meta property="og:site_name"   content="My Site">
# <meta property="og:locale"      content="en_US">
# <meta property="og:image"       content="https://example.com/share.png">
# <meta property="og:image:secure_url" content="https://example.com/share.png">
# <meta property="og:image:type"  content="image/png">
# <meta property="og:image:width" content="1200">
# <meta property="og:image:height" content="630">
# <meta property="og:image:alt"   content="Alt for screen readers and accessibility">
#
# <!-- Twitter / X -->
# <meta name="twitter:card"        content="summary_large_image">
# <meta name="twitter:site"        content="@mysite">
# <meta name="twitter:creator"     content="@author">
# <meta name="twitter:title"       content="Article title">
# <meta name="twitter:description" content="One-paragraph summary">
# <meta name="twitter:image"       content="https://example.com/share.png">
# <meta name="twitter:image:alt"   content="Alt text">
#
# <!-- Article-specific Open Graph -->
# <meta property="article:published_time" content="2026-04-25T10:00:00Z">
# <meta property="article:modified_time"  content="2026-04-25T12:00:00Z">
# <meta property="article:author"         content="https://example.com/authors/jane">
# <meta property="article:section"        content="Technology">
# <meta property="article:tag"            content="HTML">
```

### og:type values

```bash
# website, article, book, profile
# music.song, music.album, music.playlist, music.radio_station
# video.movie, video.episode, video.tv_show, video.other
```

### twitter:card values

```bash
# summary               # # small thumbnail
# summary_large_image   # # large hero image (preferred)
# app                   # # app install card
# player                # # video/audio player
```

### Image sizing

```bash
# Open Graph minimum:    600 × 315
# Open Graph recommended: 1200 × 630 (1.91:1 ratio)
# Twitter summary_large_image: 1200 × 628
# Max file size: 5 MB (Twitter), 8 MB (Facebook)
```

## Internationalization

### lang attribute

```bash
# <html lang="en">
# <html lang="en-US">
# <html lang="zh-Hans-CN">       # # simplified Chinese, China
# <html lang="ar">               # # Arabic
# <html lang="es-MX">            # # Mexican Spanish
#
# # Override on a substring
# <p>The Spanish word for hello is <span lang="es">hola</span>.</p>
```

### dir attribute

```bash
# <html lang="ar" dir="rtl">     # # whole document RTL
# <p dir="ltr">Override LTR for this paragraph.</p>
# <p dir="auto">Direction inferred from content.</p>
```

### bdi and bdo

```bash
# # bdi: bidirectional ISOLATE — protect surrounding text
# <p>User <bdi>إيان</bdi> posted a comment.</p>
#
# # bdo: bidirectional OVERRIDE — force a direction
# <p>This is <bdo dir="rtl">forced reverse</bdo>.</p>
```

### ruby annotations (East Asian)

```bash
# <ruby>
#   漢 <rp>(</rp><rt>kan</rt><rp>)</rp>
#   字 <rp>(</rp><rt>ji</rt><rp>)</rp>
# </ruby>
#
# # rt = ruby text (annotation above/beside)
# # rp = ruby parenthesis (fallback for browsers without ruby support)
```

### Locale-aware JS (Intl)

```bash
# new Intl.DateTimeFormat('de-DE').format(new Date())     // "25.4.2026"
# new Intl.NumberFormat('en-IN').format(1234567)          // "12,34,567"
# new Intl.RelativeTimeFormat('en').format(-3, 'day')     // "3 days ago"
# new Intl.ListFormat('en').format(['A','B','C'])         // "A, B, and C"
# new Intl.Collator('sv').compare('å', 'z')               // sort by Swedish rules
```

### lang on substrings = pronunciation

```bash
# <p>Visit our <span lang="fr">crèche</span>.</p>
# # Screen reader uses French voice for "crèche"
```

## Validation

### W3C Validator (online)

```bash
# https://validator.w3.org/
# https://validator.w3.org/nu/   # # Nu Html Checker (modern)
```

### Local validation with vnu.jar

```bash
# brew install vnu
# vnu --format gnu file.html
# vnu --format gnu --skip-non-html dir/
# # Or download:
# curl -LO https://github.com/validator/validator/releases/latest/download/vnu.jar
# java -jar vnu.jar file.html
# java -jar vnu.jar --format gnu --skip-non-html .
```

### html-validate (npm)

```bash
# npm install --save-dev html-validate
# npx html-validate src/**/*.html
# # Configure with .htmlvalidate.json
```

### Common validation errors

```bash
# Error: Element "div" not allowed as child of element "ul" in this context.
#   FIX: only <li> can be a direct child of <ul>/<ol>.
#
# Error: Stray end tag "div".
#   FIX: unbalanced tags — count opening/closing.
#
# Error: An "img" element must have an "alt" attribute, except under certain conditions.
#   FIX: add alt="" for decorative or alt="..." for informative.
#
# Error: Bad value "foo bar baz" for attribute "id" on element "div".
#   FIX: id can't contain spaces.
#
# Error: Duplicate ID "foo".
#   FIX: ids must be unique per document.
#
# Error: The "for" attribute of the "label" element must refer to a non-hidden form control.
#   FIX: ensure label[for=X] points to a real input id.
#
# Error: A "section" element with no heading.
#   FIX: add <h2> (or appropriate level) inside the <section>.
#
# Error: The "type" attribute is unnecessary for JavaScript resources.
#   FIX: drop type="text/javascript" — implied in HTML5.
#
# Error: Element "title" must not be empty.
#   FIX: <title> must contain non-whitespace characters.
#
# Error: Element "head" is missing a required instance of child element "title".
#   FIX: every page MUST have a <title>.
#
# Warning: This document appears to be written in English. Consider adding "lang="en"".
#   FIX: <html lang="en">.
```

## Common Element Pitfalls

### div with role=button vs button

```bash
# # BROKEN — missing keyboard support, focus, role announcement
# <div onclick="save()">Save</div>

# # PARTIAL — works for screen readers but no keyboard
# <div role="button" tabindex="0" onclick="save()">Save</div>

# # FIXED — native button does it all
# <button type="button" onclick="save()">Save</button>
```

### img without alt

```bash
# # BROKEN
# <img src="/photo.jpg">                       # # validator error

# # FIXED — informative
# <img src="/photo.jpg" alt="Sunset over mountains">

# # FIXED — decorative
# <img src="/divider.svg" alt="">
```

### Clicking on non-button non-link

```bash
# # BROKEN
# <span onclick="...">Click</span>             # # not focusable, no a11y

# # FIXED
# <button type="button" onclick="...">Click</button>
```

### Nested interactive elements

```bash
# # BROKEN — anchor inside anchor
# <a href="/post"><a href="/author">Author</a></a>      # # invalid

# # BROKEN — button inside link
# <a href="/post"><button>Read</button></a>             # # invalid

# # BROKEN — input inside label inside button
# <button><label><input></label></button>               # # invalid

# # FIX — restructure
# <a href="/post">Title</a>  <a href="/author">Author</a>
```

### Form input without label

```bash
# # BROKEN
# <input type="email" placeholder="Email">             # # placeholder ≠ label

# # FIXED — explicit
# <label for="email">Email</label>
# <input id="email" type="email">

# # FIXED — implicit
# <label>Email <input type="email"></label>

# # FIXED — aria-label (last resort)
# <input type="email" aria-label="Email">
```

### Click handler on div loses keyboard activation

```bash
# # BROKEN
# <div onclick="..." tabindex="0" role="button">Go</div>
# # Tab gets you there, but Enter/Space DON'T fire onclick.

# # FIXED — bind keydown too
# <div role="button" tabindex="0"
#      onclick="go()"
#      onkeydown="if (event.key==='Enter'||event.key===' ') go()">Go</div>

# # ACTUALLY FIXED — use a real button
# <button type="button" onclick="go()">Go</button>
```

### Auto-focus on page load

```bash
# # BROKEN — autofocus on the search input disorients screen readers
# <input autofocus type="search">

# # FIXED — let users start at the top, focus on user-initiated events
# <input type="search">
```

## Performance Hints

### Critical resource preload

```bash
# <link rel="preload" href="/critical.css" as="style">
# <link rel="preload" href="/inter.woff2" as="font" type="font/woff2" crossorigin>
# <link rel="preload" href="/hero.jpg" as="image" fetchpriority="high">
# <link rel="preload" href="/api/initial-data.json" as="fetch" crossorigin>
```

### Origin warming

```bash
# <link rel="dns-prefetch" href="//cdn.example.com">
# <link rel="preconnect"   href="https://api.example.com">
# <link rel="preconnect"   href="https://fonts.gstatic.com" crossorigin>
```

### Script loading

```bash
# <script src="/app.js"></script>             # # blocks parser (avoid in <head>)
# <script src="/app.js" defer></script>       # # parse, then execute in order
# <script src="/analytics.js" async></script> # # download in parallel, run when ready
# <script type="module" src="/app.js"></script>  # # ES module, deferred by default
# <link rel="modulepreload" href="/components/shared.js">
```

### defer vs async

```bash
# Tag                       Order        Execution
# <script>                  In order     Blocks parsing immediately
# <script async>            ANY order    As soon as fetched (NO parse blocking)
# <script defer>            In order     After DOM parsed, before DOMContentLoaded
# <script type="module">    In order     Deferred by default
```

### Image lazy loading

```bash
# <img src="..." loading="lazy" decoding="async" alt="...">
# # NEVER lazy-load above-the-fold (LCP) images.
```

### Core Web Vitals targets

```bash
# LCP (Largest Contentful Paint) — < 2.5s        # # speed
# INP (Interaction to Next Paint) — < 200ms      # # responsiveness (replaced FID in 2024)
# CLS (Cumulative Layout Shift)   — < 0.1        # # visual stability
```

### LCP optimization checklist

```bash
# 1. Preload the LCP image with fetchpriority="high".
# 2. Don't lazy-load the LCP image.
# 3. Compress and serve modern formats (AVIF/WebP).
# 4. Inline critical CSS in <head>.
# 5. Self-host fonts; preload the most-needed font weight.
# 6. Defer non-critical JS.
```

### CLS optimization checklist

```bash
# 1. Always set width/height on images and iframes.
# 2. Reserve space for ads and embeds.
# 3. Avoid inserting content above existing content (e.g., late-loaded banner).
# 4. Use font-display: swap and preload key fonts.
# 5. Use CSS aspect-ratio for placeholders.
```

### HTTP/2 server push (deprecated)

```bash
# # Chrome removed HTTP/2 server push in 2022. Use 103 Early Hints
# # with Link: rel=preload headers instead.
```

## Security

### Content Security Policy

Prefer the HTTP header `Content-Security-Policy` set by the server. The `<meta>` form has limitations (some directives like `frame-ancestors`, `report-uri`, and `sandbox` only work as headers).

```bash
# # As HTTP header (preferred)
# Content-Security-Policy: default-src 'self'; img-src 'self' https:; script-src 'self'

# # As meta tag (fallback only)
# <meta http-equiv="Content-Security-Policy"
#       content="default-src 'self'; img-src https:; script-src 'self'">
```

### CSP common directives

```bash
# default-src 'self'                           # # fallback for all
# script-src 'self' 'nonce-abc123'             # # scripts only from same origin + nonce
# style-src 'self' 'unsafe-inline'             # # avoid 'unsafe-inline' if possible
# img-src 'self' https: data:
# font-src 'self' https://fonts.gstatic.com
# connect-src 'self' https://api.example.com
# frame-ancestors 'none'                       # # blocks <iframe> embed (replaces X-Frame-Options)
# form-action 'self'
# base-uri 'self'
# object-src 'none'                            # # block <object>/<embed>
# upgrade-insecure-requests                    # # auto-upgrade http: → https:
# report-uri /csp-report                       # # legacy
# report-to default                            # # modern
```

### Subresource Integrity (SRI)

```bash
# <script src="https://cdn.example.com/lib.js"
#         integrity="sha384-oqVuAfXRKap7fdgcCY5uykM6+R9GqQ8K/uxy9rx7HNQlGYl1kPzQho1wx4JwY8wC"
#         crossorigin="anonymous"></script>
#
# <link rel="stylesheet"
#       href="https://cdn.example.com/style.css"
#       integrity="sha384-..."
#       crossorigin="anonymous">
#
# # Generate hash:
# openssl dgst -sha384 -binary file.js | openssl base64 -A
```

### HTTPS everywhere

```bash
# # Strict-Transport-Security header (server)
# Strict-Transport-Security: max-age=31536000; includeSubDomains; preload

# # Upgrade insecure mixed content
# <meta http-equiv="Content-Security-Policy" content="upgrade-insecure-requests">
```

### iframe sandbox + rel=noopener

```bash
# <iframe src="..." sandbox="allow-scripts" title="..."></iframe>
# <a href="..." target="_blank" rel="noopener noreferrer">External</a>
```

### Avoid inline JS

```bash
# # BROKEN — inline event handler (CSP often blocks this)
# <button onclick="save()">Save</button>

# # FIXED — addEventListener in JS
# <button id="save-btn">Save</button>
# <script>
#   document.getElementById('save-btn').addEventListener('click', save);
# </script>
```

### Other security headers (server-set)

```bash
# X-Content-Type-Options: nosniff
# X-Frame-Options: DENY                        # # legacy; CSP frame-ancestors preferred
# Referrer-Policy: strict-origin-when-cross-origin
# Permissions-Policy: camera=(), microphone=(), geolocation=()
# Cross-Origin-Opener-Policy: same-origin
# Cross-Origin-Embedder-Policy: require-corp
# Cross-Origin-Resource-Policy: same-site
```

## Document Charset and Encoding

### UTF-8 always

```bash
# # First in <head>, in the first 1024 bytes
# <meta charset="UTF-8">
#
# # The deprecated form (still valid but verbose):
# <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
```

### Charset gotcha

```bash
# # BROKEN — charset declared too late, browsers may have already guessed
# <head>
#   <title>Page with é and ü chars</title>
#   <meta charset="UTF-8">           # # too late
# </head>

# # FIXED — charset FIRST
# <head>
#   <meta charset="UTF-8">
#   <title>Page with é and ü chars</title>
# </head>
```

### BOM (Byte Order Mark)

```bash
# # UTF-8 BOM (EF BB BF) is technically allowed but causes problems:
# - PHP/Python/Node may emit it before <!DOCTYPE>, breaking parsing.
# - Older browsers misinterpret it.
# # Recommendation: save HTML files as UTF-8 WITHOUT BOM.
```

### HTML entities

```bash
# &amp;        # # &
# &lt;         # # <
# &gt;         # # >
# &quot;       # # "
# &apos;       # # ' (HTML5 only; XHTML safe via &#39;)
# &nbsp;       # # non-breaking space (U+00A0)
# &#160;       # # decimal numeric reference (also nbsp)
# &#xA0;       # # hexadecimal numeric reference (also nbsp)
# &copy;       # # ©
# &reg;        # # ®
# &trade;      # # ™
# &mdash;      # # — (em dash)
# &ndash;      # # – (en dash)
# &hellip;     # # …
# &laquo; &raquo; # # « »
# &lsquo; &rsquo; # # ‘ ’
# &ldquo; &rdquo; # # “ ”
# &bull;       # # •
# &middot;     # # ·
# &deg;        # # °
# &times;      # # ×
# &divide;     # # ÷
# &plusmn;     # # ±
# &micro;      # # µ
# &para;       # # ¶
# &sect;       # # §
# &dagger;     # # †
# &Dagger;     # # ‡
```

### When you MUST escape

```bash
# In text content:           &  <  >
# In attribute values:       &  "    (or ' if attribute uses ')
# In <script> / <style>:     none for text content (CDATA-like)
#                            but careful with </script> in JS strings
```

## Common Mistakes

### Self-closing void elements

In HTML5, void elements (img, br, hr, input, meta, link, area, base, col, embed, source, track, wbr) close themselves. The trailing slash is optional and has no effect.

```bash
# # All three are equivalent in HTML5:
# <br>
# <br/>
# <br />

# # ALL VALID. Pick a style and be consistent.
# # XML/XHTML required <br/>; HTML5 doesn't care.
```

### Missing alt attributes

```bash
# # BROKEN
# <img src="/photo.jpg">

# # FIXED — informative
# <img src="/photo.jpg" alt="Sunset">

# # FIXED — decorative
# <img src="/divider.svg" alt="">
```

### Using h1 multiple times

```bash
# # BROKEN
# <h1>Site Name</h1>
# ...
# <h1>Article Title</h1>

# # FIXED
# <h1>Article Title</h1>     <!-- one h1 per page -->
# <p>Posted on Site Name</p>
```

### Nesting <a> inside <a>

```bash
# # BROKEN (invalid HTML)
# <a href="/post">
#   Title <a href="/author">Author</a>
# </a>

# # FIXED — flatten
# <article>
#   <a href="/post">Title</a> by <a href="/author">Author</a>
# </article>
```

### div instead of fieldset

```bash
# # BROKEN — radio group with no semantic grouping
# <div>
#   <p>Size:</p>
#   <input type="radio" name="size" value="s"> S
#   <input type="radio" name="size" value="m"> M
# </div>

# # FIXED
# <fieldset>
#   <legend>Size</legend>
#   <label><input type="radio" name="size" value="s"> S</label>
#   <label><input type="radio" name="size" value="m"> M</label>
# </fieldset>
```

### section without heading

```bash
# # BROKEN — section is a landmark/region but has no name
# <section>
#   <p>Content</p>
# </section>

# # FIXED — add a heading
# <section>
#   <h2>News</h2>
#   <p>Content</p>
# </section>

# # OR — use <div> if it's just a styling wrapper
# <div class="card">
#   <p>Content</p>
# </div>
```

### Multiple <main>

```bash
# # BROKEN
# <main>...</main>
# <main>...</main>

# # FIXED — only one visible
# <main hidden>...</main>
# <main>...</main>
```

### Block elements inside inline

```bash
# # BROKEN
# <span><div>content</div></span>          # # span is inline; div is block

# # FIXED
# <div><span>content</span></div>
```

### <p> can't contain block elements

```bash
# # BROKEN
# <p>Quote: <blockquote>...</blockquote></p>     # # browser auto-closes <p>

# # FIXED
# <p>Quote:</p>
# <blockquote>...</blockquote>
```

### tbody auto-insertion

```bash
# # If you write:
# <table><tr><td>x</td></tr></table>

# # Browser inserts <tbody>:
# <table><tbody><tr><td>x</td></tr></tbody></table>

# # CSS selector "table > tr" won't match — use "table tr" or write <tbody> explicitly.
```

## Conditional Comments

**DEAD as of IE 11+. Modern browsers ignore them entirely.**

```bash
# <!--[if IE]>
#   <p>You are using Internet Explorer.</p>
# <![endif]-->
#
# <!--[if lt IE 9]>
#   <link rel="stylesheet" href="ie8-fallback.css">
# <![endif]-->
```

### Modern alternatives

```bash
# # Feature detection in JS
# if ('IntersectionObserver' in window) { ... }
# if (CSS.supports('display: grid')) { ... }
#
# # Feature query in CSS
# @supports (display: grid) { ... }
# @supports not (display: grid) { ... }
#
# # No-JS fallback
# <noscript>
#   <p>Please enable JavaScript.</p>
# </noscript>
```

## Browser DevTools for HTML

### Chrome / Edge

```bash
# F12 or Cmd+Option+I (macOS) / Ctrl+Shift+I (Win/Linux)
#
# Elements panel        — live DOM tree, edit HTML/attrs
# Accessibility panel   — A11y tree, ARIA roles, contrast
# Computed panel        — final CSS values
# Lighthouse            — perf, a11y, SEO, best practices audit
# Application > Manifest— PWA manifest debugging
# Issues panel          — security, deprecation, accessibility flags
```

### Firefox

```bash
# F12
#
# Inspector             — DOM tree
# Accessibility tab     — Browse by role, contrast checker, simulate
# Style Editor          — live CSS editing
# Network               — request waterfall
```

### Safari

```bash
# Enable: Safari → Preferences → Advanced → Show Develop menu
# Cmd+Option+I
#
# Elements             — DOM tree
# Audits               — Apple's a11y/perf audit
```

### Browser extensions

```bash
# axe DevTools         — automated accessibility testing
# Wave                 — accessibility evaluation
# Lighthouse           — perf + a11y audit
# HeadingsMap          — page heading outline
# HTML5 Outliner       — sectioning element outline
```

### Lighthouse CLI

```bash
# npm install -g lighthouse
# lighthouse https://example.com --view
# lighthouse https://example.com --only-categories=accessibility,seo
# lighthouse https://example.com --output=json --output-path=report.json
```

## Idioms

### Semantic-first markup

```bash
# 1. Pick the most specific semantic element first.
# 2. Drop to <div>/<span> only when no semantic option fits.
# 3. Add ARIA only when semantic HTML can't express what you need.
```

### Document outline matters more than visual layout

```bash
# # The screen-reader user "sees" headings, landmarks, lists, and links.
# # If your h1 → h2 → h3 hierarchy is broken, sighted users may not notice
# # but screen-reader users will be lost.
```

### Progressive enhancement

```bash
# Layer 1: HTML (works without CSS or JS)
# Layer 2: CSS (visual presentation)
# Layer 3: JS (interactivity)
#
# # Each layer enhances; none replaces the previous.
```

### BEM convention

```bash
# # Block__Element--Modifier
# <article class="card card--featured">
#   <h2 class="card__title">Title</h2>
#   <p class="card__body">Body</p>
#   <button class="card__btn card__btn--primary">Read</button>
# </article>
```

### Layered enhancement example

```bash
# # HTML alone (works without CSS or JS)
# <details>
#   <summary>Show details</summary>
#   <p>...</p>
# </details>

# # CSS layer adds animation
# details[open] > p { animation: fade 200ms; }

# # JS layer adds analytics
# details.addEventListener('toggle', e => {
#   if (details.open) trackEvent('details-opened');
# });
```

### One-thing-per-element

```bash
# # Each element should serve a single semantic purpose.
# # If it's a heading AND a button, you've designed something wrong.
```

## Tools

### Formatting

```bash
# Prettier              — opinionated formatter (HTML, CSS, JS)
#   npm install --save-dev prettier
#   npx prettier --write "**/*.html"
#
# js-beautify           — older but configurable
#   npm install -g js-beautify
#   html-beautify file.html
```

### Validators

```bash
# vnu.jar               — official W3C Nu Html Checker (Java)
#   java -jar vnu.jar file.html
#
# html-validate         — npm-native validator (extensible)
#   npx html-validate src/**/*.html
#
# w3c-html-validator    — Node wrapper for W3C validator
#   npx w3c-html-validator file.html
```

### Linters

```bash
# HTMLHint              — fast lint, configurable rules
#   npm install -g htmlhint
#   htmlhint **/*.html
#
# stylelint             — for embedded <style> blocks
# eslint --plugin html  — for embedded <script> blocks
```

### Accessibility

```bash
# axe-core              — accessibility testing engine
# Pa11y                 — CLI accessibility runner
#   npx pa11y https://example.com
# axe DevTools          — browser extension
# Wave                  — browser extension
# Accessibility Insights— Microsoft's a11y suite
# IBM Equal Access      — IBM's tool
```

### Performance

```bash
# Lighthouse           — in DevTools or CLI
# WebPageTest          — webpagetest.org (deep waterfall, RUM)
# PageSpeed Insights   — pagespeed.web.dev
```

### SVG

```bash
# svgo                  — SVG optimizer
#   npm install -g svgo
#   svgo file.svg
#   svgo -f dir/        # # batch
```

### HTML to PDF

```bash
# wkhtmltopdf, weasyprint, puppeteer (headless Chrome).
```

## Modern HTML5 Features

### dialog element

```bash
# <dialog id="confirm-delete">
#   <h2>Delete this item?</h2>
#   <p>This cannot be undone.</p>
#   <form method="dialog">
#     <button value="cancel">Cancel</button>
#     <button value="confirm">Delete</button>
#   </form>
# </dialog>
#
# # JS API
# document.getElementById('confirm-delete').showModal();   // modal with backdrop
# dialog.show();                                            // non-modal
# dialog.close();
# dialog.close('confirm');                                  // returnValue = 'confirm'
# dialog.returnValue;                                       // read return value
```

### dialog backdrop styling

```bash
# /* CSS */
# dialog { border: none; border-radius: 8px; }
# dialog::backdrop { background: rgba(0,0,0,.5); }
```

### popover attribute (2024+)

```bash
# # Declarative popovers, no JS needed
# <button popovertarget="my-popover">Show</button>
# <div id="my-popover" popover>
#   <p>Popover content</p>
# </div>
#
# # Variants
# popover="auto"     # # default; light-dismiss + Esc closes
# popover="manual"   # # only closes via JS or popovertarget
# popover="hint"     # # tooltip-style
#
# # Triggers
# <button popovertargetaction="show"   popovertarget="x">Open</button>
# <button popovertargetaction="hide"   popovertarget="x">Close</button>
# <button popovertargetaction="toggle" popovertarget="x">Toggle</button>
```

### search element

```bash
# # New semantic landmark for search regions
# <search>
#   <form role="search" action="/search">
#     <label for="q">Search</label>
#     <input id="q" type="search" name="q">
#     <button>Go</button>
#   </form>
# </search>
```

### details element

```bash
# # Native disclosure widget
# <details>
#   <summary>Click to expand</summary>
#   <p>Details content.</p>
# </details>
#
# # Exclusive accordion (one open at a time)
# <details name="grp"><summary>A</summary>...</details>
# <details name="grp"><summary>B</summary>...</details>
```

### inert attribute

```bash
# # Make a subtree non-interactive (background of an open modal)
# <main inert>
#   <button>This button is unreachable</button>
# </main>
# <dialog open>...</dialog>
#
# # inert removes from tab order, hides from a11y tree, blocks pointer events.
# # Toggle via JS:
# main.inert = true;
# main.inert = false;
```

### Other recent additions

```bash
# loading="lazy" on iframes (since Chrome 77, Firefox 121)
# fetchpriority="high|low|auto" on <img>, <link>, <script>
# enterkeyhint="enter|done|go|next|previous|search|send" on inputs
# hidden="until-found" — Ctrl-F can find content inside (Chrome 102+)
# anchor positioning (CSS) — declarative tooltip/popover positioning
# customStateSet — :state() pseudo-class for web components
```

## Tips

- Always start with `<!DOCTYPE html>` — without it, browsers use quirks mode.
- Set `<meta charset="UTF-8">` first; it must appear in the first 1024 bytes.
- Set `<html lang="...">` for accessibility (screen reader voice selection).
- Set `<meta name="viewport" content="width=device-width, initial-scale=1">` for mobile.
- Use semantic elements (header, nav, main, article, section, aside, footer) over div.
- Exactly one `<h1>` per page; never skip heading levels (h1 → h2, not h1 → h3).
- Every `<img>` needs `alt`; use `alt=""` for decorative.
- Always include `width` and `height` on `<img>` and `<iframe>` to avoid CLS.
- Use `<button>` for actions and `<a href>` for navigation. Never `<div onclick>`.
- Prefer `defer` over `async` for scripts that depend on the DOM or each other.
- Always associate `<label>` with inputs (see html-forms).
- Use `rel="noopener noreferrer"` on `target="_blank"` links.
- Lazy-load offscreen images (`loading="lazy"`); never lazy-load the LCP image.
- Preload critical fonts with `crossorigin` and `type="font/woff2"`.
- Prefer JSON-LD over microdata for structured data.
- Use `<dialog>` for modals; it handles focus trap and Escape natively.
- Use `<details>`/`<summary>` for native disclosure widgets — no JS needed.
- Test with keyboard only (Tab, Shift+Tab, Enter, Space) — every interactive thing must be reachable.
- Run Lighthouse / axe / Wave at least once per page.
- Validate with the W3C Nu Html Checker before shipping.
- Don't use ARIA when a semantic element exists (first rule of ARIA).
- Don't change native semantics with `role=` unless you must.
- The `hidden` attribute removes from layout AND the a11y tree.
- The `inert` attribute makes a subtree non-interactive (great for modal backgrounds).
- The `popover` attribute (2024+) replaces most JS popover libraries.
- Self-closing slashes on void elements (`<br />`, `<br/>`, `<br>`) are all valid in HTML5.
- Conditional comments (`<!--[if IE]>`) are dead — use feature detection instead.
- For SPA route changes, manage focus manually (focus the new heading or main).
- Avoid `tabindex` greater than 0 — it overrides the natural tab order.
- Prefer `:focus-visible` over `:focus` for focus rings (no ring on mouse click).
- Test with screen readers (VoiceOver on macOS/iOS, NVDA on Windows, TalkBack on Android).

## See Also

- html-forms
- css
- css-layout
- javascript
- typescript
- polyglot
- regex

## References

- [MDN HTML Reference](https://developer.mozilla.org/en-US/docs/Web/HTML)
- [MDN HTML Elements Index](https://developer.mozilla.org/en-US/docs/Web/HTML/Element)
- [MDN Global Attributes](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes)
- [HTML Living Standard (WHATWG)](https://html.spec.whatwg.org/)
- [W3C HTML 5.2 (Recommendation)](https://www.w3.org/TR/html52/)
- [W3C Markup Validator](https://validator.w3.org/)
- [W3C Nu Html Checker](https://validator.w3.org/nu/)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [MDN ARIA](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA)
- [WCAG 2.2](https://www.w3.org/WAI/standards-guidelines/wcag/)
- [web.dev — Performance](https://web.dev/performance/)
- [web.dev — Accessibility](https://web.dev/accessible/)
- [web.dev — Core Web Vitals](https://web.dev/vitals/)
- [Schema.org](https://schema.org/)
- [Open Graph Protocol](https://ogp.me/)
- [Twitter Cards Documentation](https://developer.twitter.com/en/docs/twitter-for-websites/cards)
- [Can I Use](https://caniuse.com/)
- [Lighthouse](https://developer.chrome.com/docs/lighthouse/)
- [axe-core](https://github.com/dequelabs/axe-core)
- [Pa11y](https://pa11y.org/)
- [HTMLHint](https://htmlhint.com/)
- [Prettier](https://prettier.io/)
- [SVGO](https://github.com/svg/svgo)
