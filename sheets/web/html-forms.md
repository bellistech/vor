# HTML Forms (Inputs, Validation, Accessibility)

Exhaustive reference for building accessible, validated, real-world HTML forms — every input type, every validation hook, every accessibility pattern, with the gotchas that actually bite.

## The form Element

The `<form>` element groups controls and defines how they are submitted. Every attribute has a default and most defaults bite at least once.

```bash
# Minimal form
# <form action="/login" method="post">
#   <label>Email <input type="email" name="email" required></label>
#   <button type="submit">Sign in</button>
# </form>
```

Attributes:

```bash
# action      — URL the form submits to. Empty/omitted = current URL.
# method      — get | post | dialog. Default get.
# enctype     — application/x-www-form-urlencoded (default)
#             | multipart/form-data (REQUIRED for file uploads)
#             | text/plain (debug only — never in production)
# target      — _self (default) | _blank | _parent | _top | named iframe
# novalidate  — disables browser constraint validation on submit
# autocomplete— on (default) | off — applies to all child inputs unless overridden
# name        — historical; rarely used
# accept-charset — default UTF-8; do not change
# rel         — link relationships (e.g. external)
```

GET vs POST in one breath:

```bash
# GET  — params in URL query string. Idempotent. Bookmarkable.
#        Used for searches, filters. Visible in server logs.
# POST — params in request body. Not idempotent.
#        Used for state changes. Required for file upload.
#        Required for sensitive data (passwords).
```

Encoding types:

```bash
# application/x-www-form-urlencoded
#   key=value&key2=value2 — URL encoded. Default. Cannot send files.
#
# multipart/form-data
#   Boundary-delimited parts. REQUIRED for <input type="file">.
#   Sets Content-Type: multipart/form-data; boundary=----WebKitFormBoundary...
#
# text/plain
#   Human-readable but ambiguous. Never use in production.
```

The canonical post-and-redirect (PRG) pattern — prevents duplicate submission on browser refresh:

```bash
# 1. Browser POSTs /orders
# 2. Server processes, returns 303 See Other with Location: /orders/42
# 3. Browser GETs /orders/42 — fresh page, no resubmit on refresh
```

Common mistakes:

```bash
# BROKEN — file upload without multipart
# <form action="/upload" method="post">
#   <input type="file" name="photo">
# </form>
# Server gets file NAME only, not bytes.
#
# FIXED
# <form action="/upload" method="post" enctype="multipart/form-data">
#   <input type="file" name="photo">
# </form>
```

## Submit Mechanics

A form submits when:

```bash
# 1. User clicks <button type="submit"> or <input type="submit">
# 2. User presses Enter inside a single text input (implicit submission)
# 3. JS calls form.submit() (NOTE: skips validation, skips submit event)
# 4. JS calls form.requestSubmit() (validates AND fires submit event — preferred)
```

Implicit submission rule (HTML spec):

```bash
# A form is implicitly submitted when:
#  - the form has exactly one <input> of type text/email/url/etc. AND
#  - user presses Enter while focused in that input
# To allow Enter on a multi-input form, include a submit button.
```

The `submit` event:

```bash
# form.addEventListener('submit', (e) => {
#   e.preventDefault();           // stop default GET/POST navigation
#   const data = new FormData(form);
#   const submitter = e.submitter; // which button triggered submit
#   if (submitter?.name === 'save') { ... }
# });
```

`event.submitter` distinguishes between multiple submit buttons:

```bash
# <button type="submit" name="action" value="save">Save</button>
# <button type="submit" name="action" value="publish">Publish</button>
# In submit handler: e.submitter.value === 'save' or 'publish'
```

Disabling double-submit (the spam-click defense):

```bash
# form.addEventListener('submit', (e) => {
#   const btn = e.submitter;
#   if (btn) btn.disabled = true;
#   // re-enable on error / re-render
# });
```

## The button Element

`<button>` defaults to `type="submit"` and this default has bitten every web developer alive at least once.

```bash
# DANGER — default type is submit
# <button>Open menu</button>   <!-- inside a <form>, this submits! -->
#
# FIXED — always specify type
# <button type="button">Open menu</button>
# <button type="submit">Save</button>
# <button type="reset">Reset</button>
```

Three legal types:

```bash
# type="submit"  — submits the form (default)
# type="reset"   — resets all form fields to default values (rarely useful)
# type="button"  — does nothing by default; use with JS click handlers
```

Per-button submission overrides — a button can override the parent form for its submission only:

```bash
# <form action="/save" method="post">
#   <button type="submit">Save</button>
#   <button type="submit"
#           formaction="/draft"
#           formmethod="post"
#           formenctype="multipart/form-data"
#           formtarget="_blank"
#           formnovalidate>
#     Save as draft (skip validation, open new tab)
#   </button>
# </form>
```

Override attributes:

```bash
# formaction      — overrides form's action
# formmethod      — overrides form's method
# formenctype     — overrides form's enctype
# formtarget      — overrides form's target
# formnovalidate  — disables validation for this button only
```

`<button>` vs `<input type="submit">`:

```bash
# <button>  — can contain HTML (icons, spans), semantic, preferred
# <input>   — value attribute is label, no nested content
#
# Modern: always prefer <button>.
```

Common gotchas:

```bash
# BROKEN — buttons in toolbar accidentally submit
# <form>
#   <button onclick="openMenu()">Menu</button>   <!-- type="submit" by default! -->
#   <input type="text" name="search">
# </form>
#
# FIXED
# <form>
#   <button type="button" onclick="openMenu()">Menu</button>
#   <input type="text" name="search">
# </form>
```

## The input Element — Overview

`<input>` is a void element (no closing tag). The `type` attribute decides everything about its behavior, validation, and rendering.

```bash
# Universal attributes (apply to most types):
# name         — key in submitted form data; required to be submitted
# value        — current value; for type=checkbox/radio it's what's submitted when checked
# placeholder  — hint text; NOT a label; disappears on input
# disabled     — not focusable, not submitted
# readonly     — focusable, IS submitted, not editable
# required     — must have a value (or be checked, for checkbox/radio)
# autocomplete — browser autofill hint; see Auto-completion section
# autofocus    — focus on page load (use sparingly; one per page)
# tabindex     — tab order (0=natural, -1=skip, >0=explicit, AVOID >0)
# form         — id of associated form (allows input outside <form>)
# list         — id of <datalist> for suggestions
# inputmode    — virtual keyboard hint (mobile)
```

Type list (every legal value):

```bash
# Text:     text password email url tel search
# Numeric:  number range
# Temporal: date time datetime-local week month
# Choice:   checkbox radio
# Files:    file
# Misc:     color hidden
# Buttons:  submit reset button image
```

The single most-forgotten rule:

```bash
# An input WITHOUT a name attribute is NOT submitted with the form.
# This is by design — it's how you create UI-only inputs.
```

## Input Type — text

The default. Single-line free text.

```bash
# <input type="text" name="username"
#        value=""
#        placeholder="janedoe"
#        minlength="3" maxlength="32"
#        size="20"
#        pattern="[A-Za-z0-9_]+"
#        spellcheck="false"
#        autocomplete="username"
#        inputmode="text"
#        list="user-suggestions"
#        required>
```

Attributes specific to text-like types:

```bash
# minlength    — minimum character count (validation)
# maxlength    — maximum character count (HARD limit; no characters past it)
# size         — visible width in characters (CSS width is better)
# pattern      — JavaScript regex (no slashes or flags); matched against full value
# spellcheck   — true | false; default browser-decided
# inputmode    — mobile keyboard hint
# list         — id of <datalist>
```

`pattern` examples — must match the ENTIRE value (anchors implicit):

```bash
# US ZIP            pattern="\d{5}(-\d{4})?"
# Slug              pattern="[a-z0-9-]+"
# UUID              pattern="[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
# Hex color         pattern="#[0-9A-Fa-f]{6}"
# Single emoji      not reliably possible — use server-side
```

`title` attribute pairs with `pattern` to provide the validation message:

```bash
# <input type="text" pattern="\d{5}"
#        title="Five-digit ZIP code"
#        required>
# Browser shows: "Please match the requested format. Five-digit ZIP code"
```

## Input Type — password

Same as text but value is masked. The DOM still readable — never assume secret.

```bash
# <input type="password" name="password"
#        autocomplete="current-password"
#        minlength="12"
#        required>
```

Critical autocomplete distinction:

```bash
# autocomplete="current-password"  — login forms (let manager fill saved password)
# autocomplete="new-password"      — signup/change forms (suggest a strong password)
# autocomplete="off"               — DO NOT USE; many browsers ignore for passwords
```

Pair username and password fields so password managers detect the form:

```bash
# <form>
#   <label>Email
#     <input type="email" name="email" autocomplete="username">
#   </label>
#   <label>Password
#     <input type="password" name="password" autocomplete="current-password">
#   </label>
#   <button type="submit">Sign in</button>
# </form>
```

Note: autocomplete="username" is correct on the email/text input that holds the username.

Show/hide password toggle:

```bash
# <input type="password" id="pw" name="password">
# <button type="button" aria-label="Show password"
#         onclick="const i=document.getElementById('pw');
#                  i.type = i.type==='password' ? 'text' : 'password';">
#   Show
# </button>
```

Common errors:

```bash
# BROKEN — manager won't autofill
# <input type="password" name="pwd">                  (no autocomplete hint)
#
# FIXED
# <input type="password" name="pwd" autocomplete="current-password">
```

## Input Type — email

Validates basic email shape (must contain `@` and `.`-suffixed domain).

```bash
# <input type="email" name="email"
#        autocomplete="email"
#        required>
```

Multiple emails (comma-separated):

```bash
# <input type="email" name="cc" multiple
#        autocomplete="email">
# Accepts:  alice@example.com, bob@example.org
```

Validation is lenient by HTML spec — `a@b` passes. For stricter checks:

```bash
# Use pattern attribute alongside type=email
# <input type="email"
#        pattern="[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$"
#        required>
# Or accept the lenient default and validate server-side (recommended).
```

Mobile keyboard:

```bash
# type="email" automatically shows email keyboard
# (with @ and . easily accessible) on iOS/Android.
# No need for inputmode="email" on top of it.
```

## Input Type — url, tel, search

`url` validates protocol and host:

```bash
# <input type="url" name="website"
#        autocomplete="url"
#        placeholder="https://example.com"
#        required>
# Valid:    https://example.com  https://example.com/path
# Invalid:  example.com  www.example.com   (no protocol)
```

`tel` does NOT validate format (formats vary worldwide):

```bash
# <input type="tel" name="phone"
#        inputmode="tel"
#        autocomplete="tel"
#        pattern="\+?[\d\s\-()]{7,20}"
#        placeholder="+1 (555) 123-4567">
# inputmode="tel" shows numeric keyboard on mobile.
```

`search` shows a clear-button (×) on Chrome/Safari and triggers search-specific UX:

```bash
# <input type="search" name="q"
#        results="5"
#        autocomplete="off"
#        placeholder="Search...">
# autocomplete="off" stops browser remembering past searches if undesired.
```

Common gotcha:

```bash
# BROKEN — tel field doesn't validate
# <input type="tel" required>
# User enters "abc" — browser accepts.
#
# FIXED
# <input type="tel" required
#        pattern="\+?[\d\s\-()]{7,20}"
#        title="7-20 digits, may include + - ( ) and spaces">
```

## Input Type — number

Numeric only. Spinner buttons. Triple-attribute validation: min, max, step.

```bash
# <input type="number" name="age"
#        min="0" max="120" step="1"
#        value="18"
#        inputmode="numeric"
#        required>
```

Step rules:

```bash
# step="1"     — integers only (default)
# step="0.01"  — two decimal places allowed
# step="any"   — disables stepping; allows arbitrary precision
# step="0.5"   — half-integers
```

Validation messages:

```bash
# value < min     — "Value must be greater than or equal to <min>"
# value > max     — "Value must be less than or equal to <max>"
# value % step!=0 — "Please enter a valid value. The two nearest valid values are X and Y"
```

`inputmode` distinction:

```bash
# inputmode="numeric"  — phone-style keypad (digits only)
# inputmode="decimal"  — decimal keypad (digits + .)
# Use decimal when fractional values expected (price, weight).
```

Localization gotcha — number input ALWAYS uses period as decimal separator regardless of locale:

```bash
# In de-DE locale, user expects "1,5" but type=number requires "1.5".
# BROKEN — frustrates European users
# <input type="number" step="0.1">
#
# FIXED — use text + pattern + manual parsing
# <input type="text" inputmode="decimal"
#        pattern="\d+([.,]\d+)?"
#        title="Number with optional decimal">
# Parse server-side: replace ',' with '.' before parsing.
```

## Input Type — range

Slider input. Same min/max/step/value as number.

```bash
# <input type="range" name="volume"
#        min="0" max="100" step="1"
#        value="50">
```

Always show current value (sliders are not self-labeling for visual users):

```bash
# <label>Volume <span id="vol-display">50</span>
#   <input type="range" id="vol" min="0" max="100" value="50"
#          oninput="document.getElementById('vol-display').textContent = this.value">
# </label>
# Better with <output>:
# <output for="vol">50</output>
```

Styling the slider (cross-browser pain):

```bash
# input[type="range"] {
#   -webkit-appearance: none;
#   appearance: none;
#   height: 4px;
#   background: #ccc;
# }
# input[type="range"]::-webkit-slider-thumb {
#   -webkit-appearance: none;
#   appearance: none;
#   width: 20px; height: 20px;
#   border-radius: 50%;
#   background: #007bff;
#   cursor: pointer;
# }
# input[type="range"]::-moz-range-thumb {
#   width: 20px; height: 20px;
#   border-radius: 50%;
#   background: #007bff;
#   cursor: pointer;
# }
```

## Input Type — date / time / datetime-local / week / month

Always ISO 8601. Browser shows a native picker.

```bash
# date            <input type="date" name="dob" min="1900-01-01" max="2099-12-31">
#                 Format: YYYY-MM-DD
#
# time            <input type="time" name="t" min="09:00" max="17:00" step="60">
#                 Format: HH:MM (or HH:MM:SS if step < 60)
#                 step in seconds.
#
# datetime-local  <input type="datetime-local" name="dt">
#                 Format: YYYY-MM-DDTHH:MM   (NO timezone)
#                 Old "datetime" type is dead — never use.
#
# week            <input type="week" name="w">
#                 Format: YYYY-Www  (e.g. 2026-W17)
#
# month           <input type="month" name="m">
#                 Format: YYYY-MM
```

Setting bounds:

```bash
# <input type="date" name="appt"
#        min="2026-04-25" max="2026-12-31">
# Min/max use the same ISO 8601 format as the value.
```

Browser support:

```bash
# date / time / month — Chrome, Edge, Safari (12.1+), Firefox
# week                — Chrome/Edge only; Firefox/Safari fall back to text
# datetime-local      — Chrome/Edge/Safari (14.1+); Firefox supports
# Always test, and fallback gracefully:
```

```bash
# Feature detect
# const test = document.createElement('input');
# test.type = 'date';
# const supportsDate = test.type === 'date';
# if (!supportsDate) {
#   // attach JS date picker library
# }
```

Mobile shows native picker (great UX). Use these types liberally.

Common gotcha:

```bash
# BROKEN — value mistakenly localized
# <input type="date" value="04/25/2026">
# Browser ignores invalid value silently.
#
# FIXED
# <input type="date" value="2026-04-25">
```

## Input Type — color

Color picker. Returns hex.

```bash
# <input type="color" name="bg" value="#ff8800">
# Default value: #000000 (black)
# Returns:       always 7-character #rrggbb (lowercase)
```

Limitations:

```bash
# - No alpha channel
# - No HSL or other formats
# - Picker UI varies wildly across browsers
# - Some browsers / mobile show simple swatches only
```

For richer color picking, use a JS library on top of `<input type="text">`.

## Input Type — file

File upload. Requires `enctype="multipart/form-data"` on the form.

```bash
# <input type="file" name="avatar"
#        accept="image/png, image/jpeg, .pdf"
#        multiple
#        capture="user">
```

Attributes:

```bash
# accept    — comma-separated MIME types and/or extensions
#             image/*  audio/*  video/*  .pdf  application/json
# multiple  — allow selecting multiple files
# capture   — user (front camera) | environment (rear camera) on mobile
```

JS access:

```bash
# const input = document.querySelector('input[type=file]');
# input.addEventListener('change', () => {
#   for (const file of input.files) {
#     console.log(file.name, file.size, file.type, file.lastModified);
#   }
# });
```

FormData usage:

```bash
# const fd = new FormData();
# fd.append('avatar', input.files[0]);
# fetch('/upload', { method: 'POST', body: fd });
# // Browser auto-sets Content-Type with boundary. Don't set it manually.
```

Reading file contents in browser:

```bash
# const reader = new FileReader();
# reader.onload = (e) => { console.log(e.target.result); };
# reader.readAsDataURL(input.files[0]);   // base64 data URL
# reader.readAsText(input.files[0]);      // text content
# reader.readAsArrayBuffer(input.files[0]); // binary
```

Browser does NOT enforce maximum file size:

```bash
# BROKEN — relying on accept for security
# <input type="file" accept="image/png">  <!-- user can rename evil.exe to .png -->
#
# FIXED — always validate on server
# const file = input.files[0];
# if (file.size > 5 * 1024 * 1024) { alert('Max 5 MB'); return; }   // client UX
# // Server MUST verify size, MIME type via magic bytes, scan for malware.
```

## Input Type — checkbox

Boolean. Or N-of-M selection when grouped.

```bash
# <input type="checkbox" name="terms" id="terms" value="accepted" required>
# <label for="terms">I accept the terms</label>
```

Checked vs value:

```bash
# value="accepted"  — what's submitted IF the box is checked
# checked           — initial state attribute (renders as checked)
# .checked property — live state via JS
```

Form data behavior:

```bash
# - If unchecked, the field is NOT in form data (not as empty string, not at all)
# - If no value attribute, defaults to value="on"
# - If checked and no name, NOT submitted (name is required)
```

Indeterminate state — only via JS:

```bash
# input.indeterminate = true;
# Visually a dash (-). Not part of HTML attribute.
# Use for "select all" checkboxes when partial selection.
```

Group of checkboxes (independent selections):

```bash
# <fieldset>
#   <legend>Pizza toppings</legend>
#   <label><input type="checkbox" name="topping" value="cheese"> Cheese</label>
#   <label><input type="checkbox" name="topping" value="pepperoni"> Pepperoni</label>
#   <label><input type="checkbox" name="topping" value="olives"> Olives</label>
# </fieldset>
# Server receives:  topping=cheese&topping=olives  (multiple keys)
```

Required behavior:

```bash
# required on a single checkbox makes it required to be checked.
# Useful for terms-of-service acceptance.
# required on a group of same-name checkboxes ALSO requires at least one.
```

## Input Type — radio

Single choice from a group. Group via `name`.

```bash
# <fieldset>
#   <legend>Plan</legend>
#   <label><input type="radio" name="plan" value="free" checked> Free</label>
#   <label><input type="radio" name="plan" value="pro"> Pro</label>
#   <label><input type="radio" name="plan" value="enterprise"> Enterprise</label>
# </fieldset>
```

Rules:

```bash
# - Same name = same group; only one can be checked
# - At most one checked attribute per group
# - required on ANY radio in the group makes the WHOLE group required
# - Tabbing into a radio group lands on the checked one (or first if none checked)
# - Arrow keys move between radios in the same group
# - Tab moves OUT of the group entirely (skips other radios)
```

Default selection:

```bash
# Always pre-select a sensible default.
# An unselected radio group requires the user to discover all options first.
```

Common error:

```bash
# BROKEN — name mismatch makes them all independent
# <input type="radio" name="size-s" value="small">
# <input type="radio" name="size-m" value="medium">  <!-- different name! -->
# Both can be selected. User confused.
#
# FIXED
# <input type="radio" name="size" value="small">
# <input type="radio" name="size" value="medium">
```

## Input Type — hidden

Carries a value invisible to the user. Always submitted.

```bash
# <input type="hidden" name="csrf_token" value="a4d23f...">
# <input type="hidden" name="redirect_to" value="/dashboard">
# <input type="hidden" name="user_id" value="42">
```

Use cases:

```bash
# - CSRF token
# - Server-supplied IDs (user_id, post_id)
# - Multi-step form state
# - Redirect destinations
```

NOT a security mechanism:

```bash
# BROKEN — assuming hidden inputs are tamper-proof
# <input type="hidden" name="price" value="9.99">
# User can edit DOM and change price to 0.01 before submitting.
#
# FIXED — server determines price from product ID
# <input type="hidden" name="product_id" value="42">
# // Server: lookup product 42, charge actual price.
```

## Input Type — submit, reset, button, image

Legacy button-shaped inputs. Prefer `<button>`.

```bash
# <input type="submit" value="Save">    legacy submit button
# <input type="reset" value="Reset">    resets form
# <input type="button" value="Click">   needs onclick
# <input type="image" src="go.png" alt="Submit">   submit-button image
```

`type="image"` quirk — submits the click coordinates with the form:

```bash
# <input type="image" name="map" src="map.png">
# Submits as:  map.x=42&map.y=87
# Used for image maps in old-school forms. Avoid in new code.
```

Modern alternative for everything above:

```bash
# <button type="submit">Save</button>
# <button type="reset">Reset</button>
# <button type="button" onclick="...">Click</button>
# <button type="submit"><img src="go.png" alt="Submit"></button>
```

## The textarea Element

Multi-line text. NOT an input — has open and close tags, content is the value.

```bash
# <textarea name="bio"
#           rows="4" cols="40"
#           minlength="10" maxlength="500"
#           wrap="soft"
#           placeholder="Tell us about yourself..."
#           required>Default content here</textarea>
```

Attributes:

```bash
# rows       — visible line count
# cols       — visible column count (use CSS width instead)
# wrap       — soft (default; submits as typed) | hard (submits with line breaks)
#               | off (no wrapping; horizontal scroll)
# minlength  — minimum chars
# maxlength  — maximum chars (HARD limit)
# placeholder, required, readonly, disabled, autocomplete — same as input
```

The no-self-closing rule:

```bash
# BROKEN
# <textarea name="bio" />
# Browser swallows everything until the next </textarea> as content.
#
# FIXED
# <textarea name="bio"></textarea>
```

Initial value via inner content (not value attribute):

```bash
# WRONG — value attribute ignored
# <textarea value="Hello"></textarea>
#
# RIGHT — inner content is initial value
# <textarea>Hello</textarea>
```

CSS resize:

```bash
# textarea {
#   resize: vertical;   /* default; allow vertical drag-to-resize */
#   /* horizontal | both | none */
# }
```

Auto-resize via JS:

```bash
# textarea.addEventListener('input', () => {
#   textarea.style.height = 'auto';
#   textarea.style.height = textarea.scrollHeight + 'px';
# });
```

## The select Element

Dropdown. Options inside.

```bash
# <select name="country" required>
#   <option value="">Choose a country</option>
#   <option value="us" selected>United States</option>
#   <option value="ca">Canada</option>
#   <option value="mx" disabled>Mexico (unavailable)</option>
# </select>
```

`<option>` attributes:

```bash
# value     — what's submitted (defaults to inner text)
# selected  — initial selection
# disabled  — unselectable
# label     — overrides display text (rarely used)
```

Grouping with `<optgroup>`:

```bash
# <select name="vehicle">
#   <optgroup label="Cars">
#     <option value="sedan">Sedan</option>
#     <option value="suv">SUV</option>
#   </optgroup>
#   <optgroup label="Trucks">
#     <option value="pickup">Pickup</option>
#   </optgroup>
# </select>
```

Multi-select:

```bash
# <select name="genres" multiple size="5">
#   <option>Rock</option>
#   <option>Jazz</option>
#   <option>Classical</option>
# </select>
# - multiple shows list-box (no dropdown)
# - size sets visible row count
# - Server receives multiple genres= values (or empty if nothing selected)
# - Cmd-click / Ctrl-click for multi; Shift-click for range
```

The "placeholder" pattern (HTML doesn't have a real placeholder for select):

```bash
# <select name="role" required>
#   <option value="" disabled selected hidden>Select a role</option>
#   <option value="admin">Admin</option>
#   <option value="user">User</option>
# </select>
# - disabled  — can't be re-selected
# - selected  — initial state
# - hidden    — doesn't appear in dropdown after first interaction
# - empty value="" + required — fails validation if not changed
```

Reading current selection:

```bash
# select.value             — selected option's value
# select.selectedOptions   — HTMLCollection of selected (multi-select)
# Array.from(select.selectedOptions).map(o => o.value)
```

Common errors:

```bash
# BROKEN — required on select doesn't fire if first option is empty-string-but-not-disabled
# <select required>
#   <option value="">Choose</option>
#   <option value="a">A</option>
# </select>
# This DOES work because empty value fails required.
#
# BROKEN — required does NOT fire if first option has a value
# <select required>
#   <option value="default">Default</option>
#   <option value="a">A</option>
# </select>
# User can submit "default" without realizing.
```

## The output Element

Live computation target. For attribute references inputs by id.

```bash
# <form oninput="result.value = parseFloat(a.value) + parseFloat(b.value)">
#   <input type="number" id="a" value="0">
#   +
#   <input type="number" id="b" value="0">
#   =
#   <output name="result" for="a b">0</output>
# </form>
```

Attributes:

```bash
# for   — space-separated list of input IDs that influence this output
# name  — submitted with form (rarely useful for output)
# form  — id of associated form (for output outside form element)
```

Accessibility win:

```bash
# <output> has implicit role="status" and aria-live="polite" by default.
# Screen readers announce changes WITHOUT user moving focus.
# Use it instead of <span> for live-updating values.
```

## The datalist Element

Suggestions for a text input, but input still accepts free text.

```bash
# <input list="browsers" name="browser">
# <datalist id="browsers">
#   <option value="Chrome">
#   <option value="Firefox">
#   <option value="Safari">
#   <option value="Edge">
# </datalist>
```

NOT a real dropdown — different from `<select>`:

```bash
# <select>     — restricts to listed values
# <datalist>   — suggests but accepts anything
```

Works with most input types:

```bash
# <input type="email" list="my-emails">
# <input type="url" list="my-urls">
# <input type="date" list="suggested-dates">
# <input type="color" list="brand-colors">
```

Hide the suggestion when needed:

```bash
# <input list="hints" autocomplete="off">
# autocomplete="off" suppresses browser-saved entries; datalist still shows.
```

## The fieldset and legend Elements

Group related fields. Provide an accessible group label.

```bash
# <fieldset>
#   <legend>Shipping address</legend>
#   <label>Street <input type="text" name="street"></label>
#   <label>City <input type="text" name="city"></label>
#   <label>ZIP <input type="text" name="zip"></label>
# </fieldset>
```

Why mandatory for groups:

```bash
# Screen readers announce: "Shipping address group, Street, edit text..."
# Without fieldset/legend the user hears each input out of context.
```

The disabled cascade:

```bash
# <fieldset disabled>
#   <input type="text" name="a">
#   <input type="text" name="b">
#   <button type="submit">Submit</button>
# </fieldset>
# All three are disabled (and not submitted).
# Useful for "step 2 unlocked after step 1" patterns.
```

Default styling reset (the historical border/padding bothers most designs):

```bash
# fieldset {
#   border: 0;
#   margin: 0;
#   padding: 0;
#   min-width: 0;   /* fixes flexbox layout issues with fieldset */
# }
```

When to use:

```bash
# - Radio button groups (ALWAYS)
# - Checkbox groups belonging to one concept (ALWAYS)
# - Address blocks (street + city + state + zip)
# - Date split into 3 inputs (day / month / year)
# - Card details (number + expiry + CVV)
```

## The label Element

Linking visible text to its input. ABSOLUTELY MANDATORY for every interactive input.

Two patterns — both equivalent:

```bash
# Explicit (preferred — works everywhere)
# <label for="email">Email</label>
# <input type="email" id="email" name="email">
#
# Implicit (no id needed)
# <label>Email <input type="email" name="email"></label>
```

Behavior of `<label>`:

```bash
# - Click on label focuses the input
# - Click on label toggles checkbox/radio
# - Screen reader announces label when input is focused
# - The label text is the accessible name of the input
```

The "no exceptions" rule:

```bash
# BROKEN — placeholder-only "label"
# <input type="text" placeholder="Email">
# Screen readers may not read placeholder. Once typed, hint is gone.
#
# FIXED
# <label for="email">Email</label>
# <input type="email" id="email" name="email" placeholder="you@example.com">
```

Visually-hidden label (when design demands no visible label):

```bash
# <label for="search" class="visually-hidden">Search</label>
# <input type="search" id="search" placeholder="Search...">
#
# .visually-hidden {
#   position: absolute;
#   width: 1px; height: 1px;
#   padding: 0; margin: -1px;
#   overflow: hidden;
#   clip: rect(0,0,0,0);
#   white-space: nowrap;
#   border: 0;
# }
```

Why aria-label is the LAST resort:

```bash
# aria-label hides text from sighted users (no visible label).
# Screen reader speaks it but no one else benefits.
# Visible label > visually-hidden label > aria-labelledby > aria-label.
```

## Constraint Validation

HTML5 validates without JavaScript. Browser shows native bubble on submit.

Validation attributes recap:

```bash
# required        any input    must have value (or be checked for boxes)
# pattern         text-types   regex match against full value
# min, max        number/date  numeric or date bounds
# minlength       text-types   min character count
# maxlength       text-types   max character count (also blocks past it)
# step            number/date  granularity
# type=email      email        format check
# type=url        url          format check
```

CSS pseudo-classes:

```bash
# :required        — has required attribute
# :optional        — does NOT have required attribute
# :valid           — passes all constraints
# :invalid         — fails any constraint (TRIGGERED ON LOAD — frustrating!)
# :user-valid      — passes constraints AFTER user interaction
# :user-invalid    — fails constraints AFTER user interaction
# :placeholder-shown — placeholder visible
# :read-only       — readonly attribute set
# :read-write      — editable
# :in-range        — number/date within min/max
# :out-of-range    — outside min/max
```

The `:invalid` premature-styling problem:

```bash
# BROKEN — angry red borders before user typed anything
# input:invalid { border-color: red; }
#
# FIXED — only after user interaction
# input:user-invalid { border-color: red; }
# (Modern browsers; fall back to JS class toggling for older.)
```

Disabling browser validation:

```bash
# Per form:    <form novalidate>
# Per button:  <button type="submit" formnovalidate>
# Per input:   no attribute — must use JS to ignore checkValidity()
```

## Validation API

JavaScript hooks for granular control.

```bash
# const form = document.querySelector('form');
# form.checkValidity();    // boolean — does NOT show UI
# form.reportValidity();   // boolean — DOES show UI bubble on first invalid
```

`input.validity` — the ValidityState object:

```bash
# input.validity.valid           — true if all checks pass
# input.validity.valueMissing    — required but empty
# input.validity.typeMismatch    — type=email/url format wrong
# input.validity.patternMismatch — pattern doesn't match
# input.validity.tooShort        — value < minlength (only after user-edit)
# input.validity.tooLong         — value > maxlength
# input.validity.rangeUnderflow  — value < min
# input.validity.rangeOverflow   — value > max
# input.validity.stepMismatch    — value not aligned with step
# input.validity.badInput        — input can't be parsed (e.g. "abc" in number)
# input.validity.customError     — set via setCustomValidity
```

Reading the message:

```bash
# input.validationMessage   — localized browser message
# // "Please fill out this field." or "Please enter a valid email address."
```

Setting custom validity:

```bash
# input.setCustomValidity("Username already taken");
# input.reportValidity();
# // Clear when user corrects:
# input.setCustomValidity("");
```

Listening to invalid:

```bash
# input.addEventListener('invalid', (e) => {
#   e.preventDefault();   // suppress browser bubble
#   showCustomError(input, input.validationMessage);
# });
```

## Custom Validation

Logic the spec can't express — username check, password match, server-side check.

Password confirmation match:

```bash
# const pw1 = document.getElementById('pw1');
# const pw2 = document.getElementById('pw2');
# function checkMatch() {
#   pw2.setCustomValidity(
#     pw1.value === pw2.value ? '' : 'Passwords do not match'
#   );
# }
# pw1.addEventListener('input', checkMatch);
# pw2.addEventListener('input', checkMatch);
```

Async validation (username exists?):

```bash
# let checkInFlight = null;
# username.addEventListener('input', async () => {
#   username.setCustomValidity('');   // assume valid mid-typing
#   clearTimeout(checkInFlight);
#   checkInFlight = setTimeout(async () => {
#     const taken = await fetch(`/api/usernames/${username.value}`)
#                       .then(r => r.json()).then(j => j.taken);
#     username.setCustomValidity(taken ? 'Username taken' : '');
#   }, 300);  // debounce 300ms
# });
```

Cross-field rules (end date after start date):

```bash
# function checkDates() {
#   end.setCustomValidity(
#     end.value && end.value < start.value ? 'End must be after start' : ''
#   );
# }
# start.addEventListener('change', checkDates);
# end.addEventListener('change', checkDates);
```

Debounce live feedback so the user doesn't see errors mid-keystroke:

```bash
# let timer;
# input.addEventListener('input', () => {
#   clearTimeout(timer);
#   timer = setTimeout(() => validate(input), 250);
# });
```

## Form Submission via JavaScript

The canonical AJAX form pattern.

```bash
# const form = document.getElementById('login');
# form.addEventListener('submit', async (e) => {
#   e.preventDefault();
#   if (!form.reportValidity()) return;   // shows browser errors
#
#   const data = new FormData(form);
#   try {
#     const res = await fetch(form.action, {
#       method: form.method || 'POST',
#       body: data,
#     });
#     if (!res.ok) throw new Error(await res.text());
#     window.location = '/dashboard';
#   } catch (err) {
#     showError(err.message);
#   }
# });
```

`FormData` essentials:

```bash
# const fd = new FormData(form);          // build from form element
# fd.append('extra', 'value');            // add a field
# fd.set('email', 'new@example.com');     // overwrite
# fd.delete('debug');
# fd.has('email');
# fd.get('email');
# fd.getAll('topping');                   // array of all "topping" values
# for (const [k, v] of fd.entries()) ...  // iterate
```

JSON submission:

```bash
# const data = Object.fromEntries(new FormData(form));
# fetch(form.action, {
#   method: 'POST',
#   headers: { 'Content-Type': 'application/json' },
#   body: JSON.stringify(data),
# });
# Note: Object.fromEntries collapses repeated keys (multi-checkbox).
# Use FormData iteration directly when keys repeat.
```

URL-encoded (no files):

```bash
# const params = new URLSearchParams(new FormData(form));
# fetch(form.action, {
#   method: 'POST',
#   headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
#   body: params,
# });
```

Multipart (files included) — automatic when body is FormData:

```bash
# fetch(form.action, { method: 'POST', body: new FormData(form) });
# Browser sets Content-Type: multipart/form-data; boundary=...
# DO NOT manually set the Content-Type header — boundary will be wrong.
```

## Files and Multipart Upload

```bash
# <form action="/upload" method="post" enctype="multipart/form-data">
#   <input type="file" name="files" multiple required>
#   <button type="submit">Upload</button>
# </form>
```

Iterating files:

```bash
# const input = document.querySelector('input[type=file]');
# for (const file of input.files) {
#   console.log(file.name, file.size, file.type, file.lastModified);
# }
```

Client-side size/type check (UX, not security):

```bash
# const MAX = 10 * 1024 * 1024;  // 10 MB
# for (const file of input.files) {
#   if (file.size > MAX) { alert(`${file.name} too large`); return; }
#   if (!/^image\//.test(file.type)) { alert(`${file.name} not an image`); return; }
# }
```

Upload with progress (XHR — fetch progress requires streams):

```bash
# const fd = new FormData(form);
# const xhr = new XMLHttpRequest();
# xhr.open('POST', form.action);
# xhr.upload.onprogress = (e) => {
#   if (e.lengthComputable) {
#     const pct = (e.loaded / e.total * 100).toFixed(1);
#     progressBar.value = pct;
#   }
# };
# xhr.onload = () => { /* done */ };
# xhr.send(fd);
```

Drag-and-drop file upload:

```bash
# dropzone.addEventListener('dragover', e => e.preventDefault());
# dropzone.addEventListener('drop', e => {
#   e.preventDefault();
#   const fd = new FormData();
#   for (const file of e.dataTransfer.files) {
#     fd.append('files', file);
#   }
#   fetch('/upload', { method: 'POST', body: fd });
# });
```

Server-side validation is mandatory — never trust the client:

```bash
# - Verify size against your real cap
# - Sniff MIME type from magic bytes (not from Content-Type header alone)
# - Generate new filename (prevent path traversal: ../../etc/passwd)
# - Scan with antivirus when storing
# - Store outside web root or behind auth
```

## Auto-completion

Tell the browser EXACTLY what each field is so password managers and autofill work.

```bash
# <input autocomplete="email" name="email" type="email">
# <input autocomplete="given-name" name="first">
# <input autocomplete="family-name" name="last">
# <input autocomplete="tel" name="phone" type="tel">
```

The full vocabulary (most common):

```bash
# Identity        name  given-name  additional-name  family-name  honorific-prefix
#                 honorific-suffix  nickname  username  bday  sex  organization
# Contact         email  tel  tel-country-code  tel-national  tel-area-code
# Address         street-address  address-line1  address-line2  address-line3
#                 address-level1 (state)  address-level2 (city)
#                 country  country-name  postal-code
# Login           current-password  new-password  one-time-code
# Payment         cc-name  cc-given-name  cc-additional-name  cc-family-name
#                 cc-number  cc-exp  cc-exp-month  cc-exp-year  cc-csc  cc-type
# Url/photo       url  photo  impp
```

Section prefixes for multiple of one type:

```bash
# Two addresses on one form (shipping and billing)
# <fieldset>
#   <legend>Shipping</legend>
#   <input autocomplete="section-shipping street-address" name="ship_street">
#   <input autocomplete="section-shipping postal-code" name="ship_zip">
# </fieldset>
# <fieldset>
#   <legend>Billing</legend>
#   <input autocomplete="section-billing street-address" name="bill_street">
#   <input autocomplete="section-billing postal-code" name="bill_zip">
# </fieldset>
```

One-time SMS code:

```bash
# <input type="text" inputmode="numeric"
#        autocomplete="one-time-code"
#        pattern="\d{6}"
#        maxlength="6"
#        required>
# Mobile browsers offer to autofill the code from SMS.
```

`autocomplete="off"` reality:

```bash
# - Most browsers IGNORE off on password fields (security feature for users)
# - Use specific tokens (current-password / new-password) instead
# - off is honored on regular text inputs
```

## Accessibility — Labels

Every interactive form control MUST have an accessible name.

The hierarchy of correctness:

```bash
# 1. Visible <label for="..."> linked to the input        BEST
# 2. Wrapping <label> (implicit association)              EQUALLY GOOD
# 3. aria-labelledby pointing at visible text             OK
# 4. aria-label (no visible label)                        LAST RESORT
# 5. placeholder-only                                     WRONG (broken for AT)
```

Examples:

```bash
# 1. <label for="em">Email</label> <input id="em" type="email">
# 2. <label>Email <input type="email"></label>
# 3. <h2 id="ttl">Email</h2> <input aria-labelledby="ttl">
# 4. <input type="search" aria-label="Site search">
```

Multi-source labels:

```bash
# <label id="amt-lbl">Amount</label>
# <label id="amt-cur">USD</label>
# <input type="number"
#        aria-labelledby="amt-lbl amt-cur"
#        aria-describedby="amt-help">
# <p id="amt-help">Enter a positive whole-dollar amount.</p>
```

The placeholder-as-label antipattern:

```bash
# BROKEN
# <input type="email" placeholder="Email">
# - Disappears once user types (no recall)
# - Not announced by all screen readers
# - Low contrast against background
#
# FIXED
# <label for="em">Email</label>
# <input id="em" type="email" placeholder="you@example.com">
```

## Accessibility — Error Messaging

Show errors clearly, link them to the field, and announce them.

Per-field error pattern:

```bash
# <label for="em">Email</label>
# <input type="email" id="em" name="email"
#        aria-describedby="em-err"
#        aria-invalid="true"
#        required>
# <p id="em-err" role="alert">Please enter a valid email.</p>
```

Attributes:

```bash
# aria-invalid="true"       — set when field has error; remove when fixed
# aria-describedby="ID"     — points at the error message element
# role="alert"              — interrupts and announces immediately
# aria-live="polite"        — announces when current speech ends
# aria-live="assertive"     — interrupts (use sparingly)
```

Form-level error summary (top of form):

```bash
# <div role="alert" aria-live="assertive" id="form-errors" tabindex="-1">
#   <h3>There were 2 errors with your submission:</h3>
#   <ul>
#     <li><a href="#em">Email is required</a></li>
#     <li><a href="#pw">Password too short</a></li>
#   </ul>
# </div>
```

Toggle on submit:

```bash
# form.addEventListener('submit', (e) => {
#   const invalids = form.querySelectorAll(':invalid');
#   if (invalids.length === 0) return;       // all good
#   e.preventDefault();
#   showErrorSummary(invalids);
#   invalids[0].focus();                     // focus first invalid
# });
```

Clearing errors:

```bash
# input.addEventListener('input', () => {
#   if (input.checkValidity()) {
#     input.removeAttribute('aria-invalid');
#     errorElement.textContent = '';
#   }
# });
```

## Accessibility — Required Indication

How to communicate a field is required.

The HTML way:

```bash
# <input required>
# Screen readers announce "required" automatically when the attribute is present.
```

Visible asterisk for sighted users:

```bash
# <label for="em">
#   Email <span aria-hidden="true">*</span>
#   <span class="visually-hidden">(required)</span>
# </label>
# <input id="em" type="email" required>
```

`aria-required` — usually redundant:

```bash
# <input required aria-required="true">  <!-- redundant but harmless -->
# Useful when:
# - Building a custom widget without native required (combobox, etc.)
# - Some legacy AT used to ignore native required (modern AT does not)
```

How AT announce:

```bash
# VoiceOver (iOS/macOS):  "Email, edit text, required"
# NVDA (Windows):         "Email edit required"
# JAWS:                   "Email edit required"
```

Marking optional fields instead (when most fields are required):

```bash
# <label>Phone (optional) <input type="tel"></label>
# More positive UX than asterisks everywhere.
```

## Accessibility — Fieldset / Legend Patterns

When inputs only make sense as a group, wrap them.

Mandatory cases for fieldset+legend:

```bash
# 1. Radio button groups
# <fieldset>
#   <legend>Notification frequency</legend>
#   <label><input type="radio" name="freq" value="daily"> Daily</label>
#   <label><input type="radio" name="freq" value="weekly"> Weekly</label>
# </fieldset>
#
# 2. Checkbox groups belonging to one concept
# <fieldset>
#   <legend>Accessibility needs</legend>
#   <label><input type="checkbox" name="needs" value="captions"> Captions</label>
#   <label><input type="checkbox" name="needs" value="signlang"> Sign language</label>
# </fieldset>
#
# 3. Address blocks (single concept across many inputs)
# <fieldset>
#   <legend>Shipping address</legend>
#   ...
# </fieldset>
#
# 4. Date split across inputs
# <fieldset>
#   <legend>Date of birth</legend>
#   <label>Day <input type="number" name="dob_d" min="1" max="31"></label>
#   <label>Month <input type="number" name="dob_m" min="1" max="12"></label>
#   <label>Year <input type="number" name="dob_y" min="1900" max="2099"></label>
# </fieldset>
```

How AT announce:

```bash
# "Shipping address group, Street, edit text, required, blank"
# Each input inherits the legend as additional context.
```

Visually hide the legend if design forbids it:

```bash
# <fieldset>
#   <legend class="visually-hidden">Notification frequency</legend>
#   ...
# </fieldset>
# Group is still announced even if legend is visually hidden.
```

## Accessibility — Inline vs Disabled Errors

The "submit and surface all errors" UX is the safer pattern.

Pattern A — inline errors on blur:

```bash
# input.addEventListener('blur', () => {
#   showOrClearInlineError(input);
# });
# - Only mark invalid AFTER user has interacted with that field
# - Don't show errors on still-empty fields they haven't touched
# - Screen reader users may be confused by mid-form announcements
```

Pattern B — submit-time validation (recommended for accessibility):

```bash
# form.addEventListener('submit', (e) => {
#   const invalids = form.querySelectorAll(':invalid');
#   if (invalids.length === 0) return;
#   e.preventDefault();
#   showErrorSummary(invalids);
#   invalids[0].focus();
# });
```

Pattern C — debounced live (rich UX but careful):

```bash
# - Wait until user stops typing for 500ms
# - Don't announce success on every keystroke
# - Announce errors politely (aria-live="polite")
# - Clear on every keystroke; re-validate on pause
```

The disabled error antipattern:

```bash
# BROKEN — disabling submit until form is valid
# button.disabled = !form.checkValidity();
# - User has no idea why the button is disabled
# - Screen reader users skip past disabled controls (lose discovery)
# - Doesn't tell user WHAT to fix
#
# FIXED — let them click, then surface errors clearly
# (Submit handler validates and shows error summary.)
```

## Accessibility — Focus Management

Tab order, focus traps, post-submit focus.

After failed submit:

```bash
# form.addEventListener('submit', (e) => {
#   const invalids = form.querySelectorAll(':invalid');
#   if (invalids.length === 0) return;
#   e.preventDefault();
#   invalids[0].focus();
#   invalids[0].scrollIntoView({ block: 'center', behavior: 'smooth' });
# });
```

After successful submit (SPA):

```bash
# // Either navigate (browser handles focus on new page)
# // OR show success message and focus the heading
# successMessage.focus();
# // Make sure successMessage has tabindex="-1" so it's focusable
```

Tab order rules:

```bash
# - DOM order should match visual order (avoid CSS reordering)
# - tabindex="0"  — natural place in tab order
# - tabindex="-1" — focusable via JS only, skipped by Tab
# - tabindex >= 1 — explicit order; AVOID — it overrides everything else
```

Skip past hidden/disabled:

```bash
# Browser already skips:
# - disabled inputs
# - aria-hidden="true" elements (eventually)
# - elements with display:none / visibility:hidden
```

Focus-trap antipattern in modals:

```bash
# BROKEN — trap focus, never release
# - User opens modal form
# - Tabs through fields; tabbing past last wraps to first (good)
# - Closes modal — focus is lost OR doesn't return to trigger button
#
# FIXED
# - On open, save trigger element: const trigger = document.activeElement;
# - Trap focus inside modal (Tab and Shift-Tab cycle)
# - Esc closes modal
# - On close: trigger.focus()
# - Use <dialog> element which handles much of this automatically
```

## Mobile Inputs

Pick the right `type` and `inputmode` so the right keyboard shows.

```bash
# inputmode="text"    — full keyboard (default)
# inputmode="none"    — no virtual keyboard (hardware keyboard / custom UI)
# inputmode="decimal" — decimal pad (with .)
# inputmode="numeric" — numeric pad (digits only, no .)
# inputmode="tel"     — phone keypad (with * # +)
# inputmode="search"  — keyboard with "Search" submit key
# inputmode="email"   — keyboard with @ and . easily accessible
# inputmode="url"     — keyboard with / and . easily accessible
```

Rule: inputmode is a hint, type is contractual:

```bash
# - type=number applies validation; can't be searched
# - type=text + inputmode=numeric shows numeric keyboard but allows free text
# - For pure numeric keyboard without validation:
#   <input type="text" inputmode="numeric" pattern="\d*">
```

Avoid iOS auto-zoom on focus:

```bash
# iOS Safari zooms in if the input's font-size < 16px.
# CSS:  input, textarea, select { font-size: 16px; }
# Or set a viewport meta with maximum-scale=1 (controversial — limits accessibility).
```

Numeric input cheatsheet:

```bash
# Card number     <input inputmode="numeric" pattern="\d{13,19}" autocomplete="cc-number">
# CVV             <input inputmode="numeric" pattern="\d{3,4}" maxlength="4" autocomplete="cc-csc">
# ZIP (US)        <input inputmode="numeric" pattern="\d{5}" autocomplete="postal-code">
# OTP             <input inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="\d{6}">
# Phone           <input type="tel" inputmode="tel" autocomplete="tel">
```

## Common Form Patterns

Login form:

```bash
# <form action="/login" method="post">
#   <label for="em">Email</label>
#   <input type="email" id="em" name="email"
#          autocomplete="username" required>
#
#   <label for="pw">Password</label>
#   <input type="password" id="pw" name="password"
#          autocomplete="current-password" required>
#
#   <label>
#     <input type="checkbox" name="remember" value="1">
#     Remember me
#   </label>
#
#   <a href="/forgot">Forgot password?</a>
#
#   <input type="hidden" name="csrf" value="...">
#   <button type="submit">Sign in</button>
# </form>
```

Registration form:

```bash
# <form action="/register" method="post">
#   <label for="name">Name</label>
#   <input type="text" id="name" name="name"
#          autocomplete="name" required>
#
#   <label for="em">Email</label>
#   <input type="email" id="em" name="email"
#          autocomplete="email" required>
#
#   <label for="pw">Password</label>
#   <input type="password" id="pw" name="password"
#          autocomplete="new-password" minlength="12" required>
#
#   <label for="pw2">Confirm password</label>
#   <input type="password" id="pw2" name="password_confirmation"
#          autocomplete="new-password" required>
#
#   <label>
#     <input type="checkbox" name="terms" value="1" required>
#     I accept the <a href="/terms">terms</a>
#   </label>
#
#   <button type="submit">Create account</button>
# </form>
```

Search form:

```bash
# <form role="search" action="/search" method="get">
#   <label for="q" class="visually-hidden">Search</label>
#   <input type="search" id="q" name="q"
#          autocomplete="off"
#          placeholder="Search documentation...">
#   <button type="submit">Search</button>
# </form>
```

Contact form:

```bash
# <form action="/contact" method="post">
#   <label>Name <input type="text" name="name" autocomplete="name" required></label>
#   <label>Email <input type="email" name="email" autocomplete="email" required></label>
#   <label>Message <textarea name="message" rows="5" required></textarea></label>
#   <button type="submit">Send</button>
# </form>
```

Multi-step / wizard form:

```bash
# - Show step 1, hide steps 2-N
# - On "Next" click: validate visible step's fields, hide step 1, show step 2
# - Use <fieldset disabled> on hidden steps so values still submit at the end
# - Show progress: "Step 2 of 4"
# - Allow back navigation
# - Persist state in sessionStorage in case of refresh
# - Final step: real submit button
```

## CSRF Protection

Cross-site request forgery — never trust the form's origin alone.

Synchronizer token pattern:

```bash
# 1. Server stores CSRF token in user's session
# 2. Server includes it in form HTML:
#    <input type="hidden" name="csrf_token" value="random-128-bits">
# 3. Server validates token == session.csrf on POST
# 4. Mismatch -> 403 Forbidden
```

Token generation (server-side; pseudocode):

```bash
# token = crypto_random_bytes(32).hex()
# session['csrf_token'] = token
# render template with token in hidden input
```

Double-submit cookie pattern:

```bash
# 1. Set cookie:  Set-Cookie: csrf=token; SameSite=Strict; Secure
# 2. Form includes:  <input type="hidden" name="csrf" value="token">
# 3. Server compares cookie value to form value
# 4. Match -> proceed; mismatch -> 403
```

Modern defense — SameSite cookies:

```bash
# Set-Cookie: session=abc; SameSite=Lax; Secure; HttpOnly
# - SameSite=Lax: cookie sent on same-origin GETs and top-level navigations
#                 NOT sent on cross-site POSTs (blocks CSRF for state changes)
# - SameSite=Strict: cookie never sent on cross-origin requests at all
# - Default in modern browsers is Lax.
```

The combination is best:

```bash
# - SameSite=Lax cookies (defense in depth)
# - CSRF tokens on all state-changing forms
# - Origin/Referer header check on critical endpoints
# - Re-prompt for password on sensitive actions (account delete, payment)
```

Never trust client-side validation alone:

```bash
# - Browser validation is for UX, not security
# - Server MUST re-validate everything
# - Client validation is bypassed by:
#   - Editing DOM in DevTools
#   - Sending raw HTTP requests (curl, Postman)
#   - Disabled JavaScript
#   - Old browsers
```

## Browser Autofill

Get autofill right and your forms feel magical. Get it wrong and users can't use them.

The basics:

```bash
# 1. Use semantic <input type="..."> values (email, tel, etc.)
# 2. Set autocomplete with the right token
# 3. Use sensible name attribute (browser sometimes uses name as fallback)
# 4. Wrap login fields in a single <form> element
```

Login form must include both fields:

```bash
# BROKEN — multi-step forms hide password field on step 1
# Step 1: <input type="email" autocomplete="username">
# Step 2: <input type="password" autocomplete="current-password">
# Many password managers can't bridge the steps.
#
# FIXED — single form with both fields visible (or visible-when-needed)
# <form>
#   <input type="email" autocomplete="username" required>
#   <input type="password" autocomplete="current-password" required>
#   <button type="submit">Sign in</button>
# </form>
```

Address autofill best practices:

```bash
# - Use full set of autocomplete tokens
# - One field per concept (don't combine name into one input)
# - Use section-* prefix when you have shipping AND billing
# - country and address-level1 (state) work best as <select>
```

One-time codes (SMS, TOTP):

```bash
# <input type="text"
#        inputmode="numeric"
#        autocomplete="one-time-code"
#        pattern="\d{6}"
#        maxlength="6">
# iOS will offer to autofill from incoming SMS.
# Android Chrome supports too.
```

Disabling autofill (rare but legitimate cases):

```bash
# - For a search box: autocomplete="off"  (browsers honor)
# - For a password: NEVER use off; use new-password to suggest a strong one
# - For a one-time-use field: autocomplete="off" (browsers usually honor)
```

## Common Errors and Fixes

Exact error texts you'll see and what they mean.

```bash
# "Please fill out this field."
#   Browser default for required + empty value.
#   Fix:  add value, OR remove required, OR use setCustomValidity to override.
```

```bash
# "Please match the requested format."
#   Pattern attribute didn't match.
#   Fix:  add title="..." to give a useful message.
#   <input pattern="\d{5}" title="5-digit ZIP code" required>
#   Now message is: "Please match the requested format. 5-digit ZIP code"
```

```bash
# "Please enter a valid email address."
#   type=email format check failed.
#   Fix:  guide user. Show pattern in placeholder.
```

```bash
# "An invalid form control with name='X' is not focusable."
#   Console warning. The form has an invalid required field that can't be focused —
#   typically because it's hidden or inside a hidden parent.
#   Fix:  remove required from hidden fields, OR remove the field from form on hide:
#   field.removeAttribute('required');
#   field.removeAttribute('name');   // also exclude from FormData
```

```bash
# "Form submission canceled because the form is not connected."
#   Form was removed from DOM during submit handler.
#   Often a timing issue — async work after preventDefault.
#   Fix:  don't remove the form before submission completes. Use FormData snapshot.
```

```bash
# "A form field element should have an id or name attribute."
#   Lighthouse warning. Field won't be picked up by autofill.
#   Fix:  add a name attribute (always) and id (when paired with label-for).
```

```bash
# "Password field is not contained in a form."
#   Browsers won't offer autofill for password outside <form>.
#   Fix:  wrap in a <form> (use action/method or just JS-handled).
```

```bash
# Server rejects with "CSRF token mismatch".
#   Token expired (session ended), missing, or wrong.
#   Fix:  re-render form with fresh token; verify HTML is being served with no caching.
```

## Common Gotchas

Each one shown broken AND fixed.

Gotcha 1 — `<button>` defaults to submit:

```bash
# BROKEN
# <form>
#   <button onclick="doSomething()">Click me</button>   <!-- submits form! -->
# </form>
#
# FIXED
# <form>
#   <button type="button" onclick="doSomething()">Click me</button>
# </form>
```

Gotcha 2 — Spam-click multiple submissions:

```bash
# BROKEN
# <button type="submit">Pay $99</button>
# User clicks 3 times before page navigates — 3 charges.
#
# FIXED
# form.addEventListener('submit', (e) => {
#   const btn = e.submitter;
#   if (btn.disabled) { e.preventDefault(); return; }
#   btn.disabled = true;
#   btn.textContent = 'Processing...';
# });
```

Gotcha 3 — Missing `name` attribute drops the field:

```bash
# BROKEN
# <input type="email" id="email" required>     <!-- no name! -->
# Submitting form: email field is NOT in FormData.
#
# FIXED
# <input type="email" id="email" name="email" required>
```

Gotcha 4 — Disabled inputs are NOT serialized:

```bash
# BROKEN
# <input type="hidden" name="user_id" value="42" disabled>
# Server doesn't receive user_id.
#
# FIXED — use readonly to keep it submitted, or just remove disabled
# <input type="hidden" name="user_id" value="42">
```

Gotcha 5 — Readonly inputs ARE serialized:

```bash
# Surprising but true:
# <input name="email" value="user@example.com" readonly>
# Form data: email=user@example.com   (yes, submitted)
# Use this for "show but don't let them edit" with the value still POSTing.
```

Gotcha 6 — `multiple` select isn't intuitive:

```bash
# BROKEN
# <select name="cities" multiple>...</select>
# Users don't know to Cmd-click. Lose selections accidentally.
#
# FIXED — use a different UI
# <fieldset>
#   <legend>Cities</legend>
#   <label><input type="checkbox" name="cities" value="nyc"> NYC</label>
#   <label><input type="checkbox" name="cities" value="la"> LA</label>
#   ...
# </fieldset>
# Or a dedicated multi-select widget like a tags input.
```

Gotcha 7 — Placeholder confused with label:

```bash
# BROKEN
# <input type="email" placeholder="Email">
# Once typed, no recall what the field is.
#
# FIXED
# <label for="em">Email</label>
# <input id="em" type="email" placeholder="you@example.com">
```

Gotcha 8 — Trusting client validation for security:

```bash
# BROKEN
# <input type="number" min="0" max="100" name="discount">
# Server: SELECT ... WHERE discount = $discount;
# Attacker bypasses HTML, sends discount=200.
#
# FIXED
# Server-side: assert 0 <= discount <= 100; reject otherwise.
# HTML validation is UX only.
```

Gotcha 9 — Numeric input local format:

```bash
# BROKEN — German user can't type comma decimals
# <input type="number" step="0.01">
# User types 1,5 — input shows 1,5 but value is empty (parse fail).
#
# FIXED
# <input type="text" inputmode="decimal"
#        pattern="\d+([.,]\d+)?"
#        title="Numbers with optional decimal">
# Parse server-side: replace ',' with '.' before parseFloat.
```

Gotcha 10 — Pattern requires full match (no anchors needed):

```bash
# BROKEN
# <input pattern="^\d{5}$">
# Anchors are implicit. ^...$ does no harm but is redundant.
#
# FIXED
# <input pattern="\d{5}">
```

Gotcha 11 — Submit on Enter for single text input:

```bash
# Surprising:
# <form>
#   <input type="text" name="q">           <!-- single text input -->
# </form>
# Pressing Enter SUBMITS even with no submit button. By design.
# To allow Enter to insert newline instead, use <textarea>.
```

Gotcha 12 — Empty form attribute prevents association:

```bash
# BROKEN
# <input type="text" form="">
# Empty form="" un-associates the input. NOT submitted with parent form.
#
# FIXED — remove the form attribute (default association is the parent <form>)
# <input type="text">
```

Gotcha 13 — `<input type="number">` allows scroll-to-change on focus:

```bash
# Frustrating UX bug: user scrolls page, page scrolls, but if focus is on number
# input, the value changes instead.
# FIX:
# input[type="number"]::-webkit-inner-spin-button { display: none; }
# input.addEventListener('wheel', e => e.target.blur());  // crude but works
```

Gotcha 14 — `select` doesn't have `placeholder`:

```bash
# BROKEN
# <select placeholder="Choose">  <!-- attribute does nothing -->
#   <option value="a">A</option>
# </select>
#
# FIXED
# <select required>
#   <option value="" disabled selected hidden>Choose</option>
#   <option value="a">A</option>
# </select>
```

## Performance

Forms can be heavy if validation runs on every keystroke.

Debouncing live validation:

```bash
# function debounce(fn, ms) {
#   let t;
#   return (...args) => {
#     clearTimeout(t);
#     t = setTimeout(() => fn(...args), ms);
#   };
# }
# input.addEventListener('input', debounce(validate, 250));
```

Defer expensive checks until blur:

```bash
# input.addEventListener('blur', async () => {
#   await checkUsernameTaken(input.value);
# });
# // Don't hit the server on every keystroke.
```

Minimize reflow during validation:

```bash
# - Batch DOM updates (don't toggle classes per field if many)
# - Use classList.toggle(name, condition) over add/remove pairs
# - Read all values first, then update all DOM (avoids interleaved reflow)
# - Use requestAnimationFrame for visual updates
```

Don't run heavy regex on every keystroke:

```bash
# BROKEN
# input.addEventListener('input', () => {
#   if (mySlowRegex.test(input.value)) { ... }   // ON EVERY KEY!
# });
#
# FIXED — debounce + simpler check
# input.addEventListener('input', debounce(() => {
#   if (mySlowRegex.test(input.value)) { ... }
# }, 250));
```

Lazy-load form sections:

```bash
# - Don't render every step of a wizard upfront
# - Use <template> or fetch the HTML for step N when reached
```

## Tools and Testing

Automated accessibility:

```bash
# axe-core / @axe-core/cli
#   npx axe https://example.com
#   Reports missing labels, color-contrast issues, ARIA misuse.
#
# Pa11y
#   pa11y https://example.com
#   Built on axe; outputs CLI-friendly results.
#
# Lighthouse
#   chrome://inspect or DevTools "Lighthouse" tab
#   Audits "Forms" category — labels, autocomplete tokens, etc.
#
# WAVE
#   webaim.org/wave — paste URL, see annotated screenshot.
```

Manual keyboard testing:

```bash
# 1. Tab through every focusable element. Order matches visual?
# 2. Activate every control with Enter / Space.
# 3. Use arrows on radio groups, sliders, selects.
# 4. Esc dismisses modals / dropdowns.
# 5. Verify focus trap on modals (Tab cycles inside).
# 6. Verify focus returns to trigger after closing modal.
```

Screen readers to test with:

```bash
# - VoiceOver (macOS / iOS) — Cmd+F5 to enable on Mac
# - NVDA (Windows) — free, install from nvaccess.org
# - JAWS (Windows) — paid, demo available
# - TalkBack (Android) — Settings > Accessibility
```

What to listen for:

```bash
# 1. Each input announces label, type, required, current value.
# 2. Errors are announced when they appear.
# 3. Group context (fieldset legend) is announced.
# 4. Submit feedback (success/error) is announced via aria-live.
```

Browser DevTools accessibility tree:

```bash
# Chrome:    DevTools > Elements > Accessibility tab
# Firefox:   DevTools > Accessibility tab
# Inspect each input — verify name, role, state.
```

## Modern Form Features

The `<dialog>` element (built-in modal):

```bash
# <dialog id="confirm">
#   <form method="dialog">
#     <p>Delete this item?</p>
#     <button value="cancel">Cancel</button>
#     <button value="confirm">Delete</button>
#   </form>
# </dialog>
# <button onclick="confirm.showModal()">Delete</button>
#
# // method="dialog" closes the dialog instead of submitting.
# // dialog.returnValue contains the clicked button's value.
# // <dialog> handles focus trap and Esc-to-close natively.
```

Form-associated custom elements:

```bash
# class MyInput extends HTMLElement {
#   static formAssociated = true;
#   #internals;
#   constructor() {
#     super();
#     this.#internals = this.attachInternals();
#   }
#   set value(v) {
#     this.#internals.setFormValue(v);
#   }
#   checkValidity() { return this.#internals.checkValidity(); }
# }
# customElements.define('my-input', MyInput);
# // Now <my-input name="x"> participates in form data.
```

`contenteditable=plaintext-only` (rich-input replacement, plain output):

```bash
# <div contenteditable="plaintext-only" role="textbox" aria-label="Notes"></div>
# - Strips formatting on paste
# - No HTML can be entered
# - Combined with role=textbox so AT treats it as input
```

Hidden submit-button alternative — HTML submitter property:

```bash
# // Multiple submit buttons
# <button type="submit" name="action" value="save">Save</button>
# <button type="submit" name="action" value="delete">Delete</button>
# // Read which one in handler:
# form.addEventListener('submit', e => {
#   if (e.submitter.value === 'delete') confirmDelete();
# });
```

The `inert` attribute (skip element from interaction & AT):

```bash
# <div inert>
#   <input type="text">       <!-- not focusable, not announced -->
# </div>
# Useful for inactive wizard steps without losing form association.
```

## Tips

```bash
# - Always use a <label>. Always.
# - Always specify type=button on non-submitting <button>s inside forms.
# - Always set autocomplete on auth and address fields.
# - Always validate server-side; HTML validation is UX only.
# - Always use enctype=multipart/form-data when uploading files.
# - Always use POST for state changes (not GET).
# - Always use SameSite cookies + CSRF tokens together.
# - Always disable submit button on first submit to prevent double-post.
# - Always show errors with aria-describedby + aria-invalid.
# - Always focus the first invalid field after a failed submit.
# - Avoid placeholder as the only label.
# - Avoid disabling the submit button to "guide" the user — show errors instead.
# - Avoid tabindex >= 1.
# - Avoid auto-submitting on every input change (jarring for AT users).
# - Prefer <button type="submit"> over <input type="submit">.
# - Prefer constraint validation attributes over JS regexes when possible.
# - Prefer fetch() with FormData over old XHR for AJAX submissions.
# - Test with keyboard only.
# - Test with a screen reader.
# - Test on slow networks (loading states matter).
# - Test on mobile with virtual keyboards.
# - Run axe-core or Lighthouse before shipping.
# - Read the WHATWG forms spec when in doubt — html.spec.whatwg.org/multipage/forms.html
```

## See Also

- html
- css
- css-layout
- javascript
- typescript
- polyglot
- regex

## References

- developer.mozilla.org/en-US/docs/Web/HTML/Element/form
- developer.mozilla.org/en-US/docs/Web/HTML/Element/input
- developer.mozilla.org/en-US/docs/Web/HTML/Element/button
- developer.mozilla.org/en-US/docs/Web/HTML/Element/textarea
- developer.mozilla.org/en-US/docs/Web/HTML/Element/select
- developer.mozilla.org/en-US/docs/Web/HTML/Element/fieldset
- developer.mozilla.org/en-US/docs/Web/HTML/Element/label
- developer.mozilla.org/en-US/docs/Web/HTML/Element/output
- developer.mozilla.org/en-US/docs/Web/HTML/Element/datalist
- developer.mozilla.org/en-US/docs/Web/HTML/Attributes/autocomplete
- developer.mozilla.org/en-US/docs/Web/HTML/Attributes/inputmode
- developer.mozilla.org/en-US/docs/Web/API/Constraint_validation
- developer.mozilla.org/en-US/docs/Web/API/FormData
- developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement
- developer.mozilla.org/en-US/docs/Web/API/ValidityState
- html.spec.whatwg.org/multipage/forms.html
- html.spec.whatwg.org/multipage/input.html
- www.w3.org/WAI/ARIA/apg/patterns/
- www.w3.org/WAI/tutorials/forms/
- web.dev/articles/learn/forms
- web.dev/articles/sign-in-form-best-practices
- web.dev/articles/sign-up-form-best-practices
- web.dev/articles/payment-and-address-form-best-practices
- developers.google.com/web/fundamentals/design-and-ux/input/forms
- accessibility.huit.harvard.edu/forms
