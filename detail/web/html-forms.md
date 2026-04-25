# The Internals of HTML Forms — Submission Algorithm, Validation, and Accessibility Tree

> *A `<form>` element is not a passive container. It is a participant in a precisely-specified algorithm that walks the DOM tree, builds a name-value entry list, encodes it into one of three wire formats, and triggers a navigation. Along the way it consults a constraint validation API, fires a cancellable submit event, dispatches a formdata event for last-mile mutation, exposes a ValidityState interface to JavaScript, and integrates with the platform accessibility tree. Modern web apps that "just use fetch and JSON" miss most of this — and pay for it in lost autofill, broken accessibility, and brittle validation. This document walks the spec.*

---

## 1. The Form Submission Algorithm

The HTML Living Standard defines form submission as a deterministic sequence. When a user clicks a submit button or presses Enter in a single-line input, the browser does not "send the form" — it runs an algorithm.

### 1.1 The Spec's Trigger Points

Submission is **triggered** by one of:

1. The user activating a submit button (`<button type=submit>`, `<input type=submit>`, `<input type=image>`).
2. **Implicit submission** — pressing Enter inside a single-line text input when the form has exactly one such input, or when the form has a default submit button.
3. A script calling `form.submit()` (skips validation, no submit event) or `form.requestSubmit(submitter?)` (runs full algorithm, fires submit event).

Each path produces a **submitter** — the element that initiated submission. For programmatic `requestSubmit()` you may pass the submitter explicitly; for `submit()` it is null. The submitter matters because per-button overrides (`formaction`, `formmethod`, `formenctype`, `formtarget`, `formnovalidate`) come from it.

### 1.2 The Eleven-Ish Steps

The spec's "form submission algorithm" runs in this order (paraphrased from §4.10.21.3):

1. **Locate the form**. If the submitter has a `form` attribute pointing to a different form by ID, use that.
2. **Check sandboxing**. If the form's document has `sandbox` without `allow-forms`, abort.
3. **Check submission already in progress**. If the form's `firing-submission-events` flag or `constructing-entry-list` flag is set, abort to prevent re-entrant submits.
4. **Static validation**. Unless `novalidate` (or `formnovalidate` on the submitter) is set, run "interactively validate the constraints". If invalid, fire `invalid` events and abort.
5. **Fire a `submit` event** at the form, with `submitter` set. Bubbles. Cancelable. If `event.preventDefault()` is called, abort.
6. **Construct the entry list**. Walk all submittable elements; build the name-value list. Fire `formdata` event during this step (after the list is constructed but before encoding).
7. **Determine the action URL**. Use `formaction` on submitter if present, else the form's `action`, else the document URL.
8. **Determine the method**. `formmethod` > `method` > `"get"`.
9. **Determine the encoding**. `formenctype` > `enctype` > `application/x-www-form-urlencoded`.
10. **Determine the target browsing context**. `formtarget` > `target` > the current browsing context.
11. **Encode and navigate**. Build the request body (or query string) and navigate.

### 1.3 Implicit Submission

The "press Enter to submit" behavior is precise:

> *A form element's default button is the first submit button in tree order whose form owner is that form element.*

> *If the user agent supports letting the user submit a form implicitly (for example, on some platforms hitting the "enter" key while a text field is focused implicitly submits the form), then doing so for a form, whose default button has activation behavior and is not disabled, must cause the user agent to fire a click event at that default button.*

If the form has **no submit button**, implicit submission still works **only** if the form has exactly one input of type `text`, `search`, `url`, `tel`, `email`, `password`, `date`, `month`, `week`, `time`, `datetime-local`, or `number`.

```html
<form action="/login">
  <input name="username" type="text" required>
  <input name="password" type="password" required>
  <!-- No submit button. Two text inputs. Enter does NOT submit. -->
</form>

<form action="/search">
  <input name="q" type="search">
  <!-- No submit button. One text input. Enter submits. -->
</form>
```

### 1.4 Per-Button Overrides

A submit button may override the form's submission attributes for the case where it is the submitter:

```html
<form action="/save" method="post" enctype="application/x-www-form-urlencoded">
  <input name="title" type="text">
  <button type="submit">Save</button>
  <button type="submit" formaction="/save-as-draft">Save as Draft</button>
  <button type="submit" formaction="/delete" formmethod="post">Delete</button>
  <button type="submit" formnovalidate>Save Without Validating</button>
</form>
```

The overrides on `<button>`/`<input type=submit>`:

| Attribute        | Overrides       | Example                        |
|------------------|-----------------|--------------------------------|
| `formaction`     | `action`        | `formaction="/draft"`          |
| `formmethod`     | `method`        | `formmethod="post"`            |
| `formenctype`    | `enctype`       | `formenctype="multipart/form-data"` |
| `formtarget`     | `target`        | `formtarget="_blank"`          |
| `formnovalidate` | (skip validation)| boolean                       |

### 1.5 The submit Event vs. the formdata Event

Two events fire during submission, in this order:

1. `submit` — at the form, before construct-entry-list. Cancelable. The event object has a `submitter` property pointing to the activating button (or null for `form.submit()`).
2. `formdata` — at the form, *after* construct-entry-list but *before* encoding. **Not** cancelable. The event object has a `formData` property (a `FormData` mutable view) for last-mile manipulation.

```javascript
form.addEventListener('submit', (e) => {
  console.log('submitter:', e.submitter); // <button> or null
  // e.preventDefault() to cancel
});

form.addEventListener('formdata', (e) => {
  // Last chance to modify the entry list
  e.formData.append('csrf-token', getCsrfToken());
  e.formData.delete('debug-only-field');
});
```

---

## 2. Form Data Set Construction

The algorithm that walks form-associated elements and builds the entry list is `construct the entry list` (§4.10.21.4).

### 2.1 Form-Associated Elements

A form-associated element is one of: `button`, `fieldset`, `input`, `object`, `output`, `select`, `textarea`, `img`, plus form-associated custom elements. Of these, the **submittable** subset is: `button`, `input`, `select`, `textarea`, plus form-associated custom elements with a value.

### 2.2 The Algorithm in Pseudocode

```
function constructEntryList(form, submitter, encoding):
    if form.constructingEntryList: return null  # re-entrancy guard
    form.constructingEntryList = true
    entryList = []

    for each element in form.elements (in tree order):
        if element is fieldset: continue
        if element is disabled: continue
        if element is a submit button and is not the submitter: continue
        if element is type="reset": continue
        if element is type="button": continue
        if element is type="image" and is not submitter: continue
        if element has no name attribute (or name is empty): continue
        if element is checkbox/radio and is not checked: continue
        if element is type="file":
            for each file in element.files: append (name, File) to entryList
            if no files and element has name: append (name, "") to entryList (empty file)
        elif element is select:
            for each option in element.options:
                if option.selected and not option.disabled:
                    append (name, option.value) to entryList
        else:
            append (name, element.value) to entryList

    # Fire the formdata event with entryList wrapped in a FormData
    fireFormDataEvent(form, entryList)

    form.constructingEntryList = false
    return entryList
```

### 2.3 The Skip Rules — Subtle Cases

**Disabled elements are skipped, including descendants of disabled fieldsets.**

```html
<fieldset disabled>
  <input name="hidden-because-fieldset" value="never-sent">
</fieldset>
```

**Readonly elements are NOT skipped.** A `readonly` text input still serializes its value:

```html
<input name="user-id" value="42" readonly>
<!-- Sent as user-id=42 -->
```

**Unchecked checkboxes and unselected radios are skipped entirely** — they do not appear in the entry list at all (not even as empty strings). This is the canonical "checkbox unchecked = absent from request" behavior.

```html
<input type="checkbox" name="newsletter" value="yes">
<!-- If unchecked: nothing in request. Server cannot tell unchecked from absent. -->
```

The conventional fix is a hidden companion:

```html
<input type="hidden" name="newsletter" value="no">
<input type="checkbox" name="newsletter" value="yes">
<!-- If unchecked: newsletter=no. If checked: newsletter=no&newsletter=yes (server takes last). -->
```

**Submit buttons are only included if they are the submitter.** A form with three submit buttons sends only the value of the one that was clicked. The submitter's `name=value` is appended to the entry list at the position the button appeared in tree order.

### 2.4 Multi-select

A `<select multiple>` produces one entry per selected option:

```html
<select name="tags" multiple>
  <option value="a" selected>A</option>
  <option value="b">B</option>
  <option value="c" selected>C</option>
</select>
<!-- Entry list: [("tags", "a"), ("tags", "c")] -->
```

A non-multiple `<select>` always produces exactly one entry — the first option is selected by default if none has the `selected` attribute.

### 2.5 The Fixed Character Encoding

The browser determines the encoding of the entry list **once**, before iteration:

1. If the form has an `accept-charset` attribute, use the first supported charset listed.
2. Otherwise use the document's character encoding (almost always UTF-8 in modern docs).

This matters for legacy systems: if your page is in `windows-1252`, form values containing non-Latin-1 characters will be encoded with `&#NNN;` numeric character references in `application/x-www-form-urlencoded`. Modern advice: always serve UTF-8 documents.

---

## 3. Encoding Types — application/x-www-form-urlencoded

The default encoding. Used for `GET` requests (as query string) and `POST` with no `enctype` override.

### 3.1 The Algorithm

For each `(name, value)` entry:

1. Replace `0x20` (space) with `+` (NOT with `%20`, despite RFC 3986 — the form encoding pre-dates that RFC).
2. Percent-encode any byte that is not in the unreserved set: `A-Z a-z 0-9 * - . _`.
3. Concatenate as `name=value` joined by `&`.

```
[("title", "hello world"), ("body", "100% pure")]
→ title=hello+world&body=100%25+pure
```

### 3.2 The Special Characters

| Byte         | Encoded as |
|--------------|-----------|
| `0x20` space | `+`       |
| `0x0A` LF    | `%0A`     |
| `0x0D` CR    | `%0D`     |
| `&`          | `%26`     |
| `=`          | `%3D`     |
| `+`          | `%2B`     |
| `%`          | `%25`     |
| Multi-byte UTF-8 (e.g. é = 0xC3 0xA9) | `%C3%A9` |

### 3.3 The CR-LF Line-Ending Normalization

Multi-line `<textarea>` values normalize newlines to CR-LF before encoding:

```
"line1\nline2"  →  encoded as  line1%0D%0Aline2
```

This applies even on platforms where the user's keyboard inserts LF only. The spec mandates the normalization.

### 3.4 GET vs POST Wire Format

**GET** appends the encoded entry list to the action URL as a query string:

```http
GET /search?q=hello+world&page=2 HTTP/1.1
Host: example.com
```

**POST** sends the encoded entry list as the request body:

```http
POST /save HTTP/1.1
Host: example.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 27

title=hello+world&page=2
```

Note: GET cannot carry binary data; for file uploads you must use POST with `multipart/form-data`.

---

## 4. Encoding Types — multipart/form-data

Required for file uploads. Defined by RFC 7578 (which itself defers to RFC 2046 §5.1 for the multipart structure).

### 4.1 The Boundary

The browser generates a random boundary string (typically 30+ random characters prefixed with hyphens):

```
------WebKitFormBoundary7MA4YWxkTrZu0gW
```

The boundary appears in the `Content-Type` header and as a delimiter in the body. The boundary must not appear inside any field value (the browser ensures this by re-rolling on collision, though collision is astronomically unlikely with sufficient randomness).

### 4.2 Per-Part Anatomy

Each entry becomes a "part" with:

- A leading `--<boundary>` line.
- One or more headers, primarily `Content-Disposition: form-data; name="..."` (and `filename="..."` for files).
- A blank line.
- The raw value bytes.
- A trailing CR-LF.

The final boundary is `--<boundary>--` (with trailing hyphens) to mark end-of-multipart.

### 4.3 A Worked Example

For a form with `<input name="title" value="Trip">` and `<input type=file name="photo">` (file `cat.jpg`, content-type `image/jpeg`, 1234 bytes):

```http
POST /upload HTTP/1.1
Host: example.com
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Length: 1456

------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="title"

Trip
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="photo"; filename="cat.jpg"
Content-Type: image/jpeg

<1234 raw bytes of JPEG data>
------WebKitFormBoundary7MA4YWxkTrZu0gW--
```

Note:

- Text fields have NO `Content-Type` per-part header (defaults to `text/plain;charset=UTF-8` if needed).
- File fields include both `filename=` and a `Content-Type` reflecting the file's MIME (browser detects from extension or from File.type).
- The body bytes are NOT escaped or encoded — they are raw. The boundary is the only separator.

### 4.4 Multiple Files in One Field

For `<input type=file name="photos" multiple>`, each file becomes its own part with the same `name=`:

```http
------BOUNDARY
Content-Disposition: form-data; name="photos"; filename="a.jpg"
Content-Type: image/jpeg

<bytes>
------BOUNDARY
Content-Disposition: form-data; name="photos"; filename="b.jpg"
Content-Type: image/jpeg

<bytes>
------BOUNDARY--
```

Server-side, this is parsed as a list under the same field name.

### 4.5 Empty File Inputs

If `<input type=file name="x">` has no file selected, the part is still emitted with empty filename and empty body:

```http
------BOUNDARY
Content-Disposition: form-data; name="x"; filename=""
Content-Type: application/octet-stream

------BOUNDARY--
```

This is a notable difference from `application/x-www-form-urlencoded`, which would send `x=` (empty value).

---

## 5. Encoding Types — text/plain

A third, rarely-used encoding type defined by the HTML spec primarily for debugging and email-form scenarios.

### 5.1 Format

Each entry becomes one line of `name=value`, separated by CR-LF. **No escaping.** Spaces, equals signs, ampersands all appear literally.

```
title=hello world
body=100% pure
note=line1
line2
```

Yes — if a value contains a newline, it splits over multiple lines and there is **no way to disambiguate** field boundaries. This is why the spec calls it "for debugging" and warns it should not be used in production.

### 5.2 When You Might See It

- The historical `mailto:` form action (now blocked or limited in modern browsers).
- Simple internal tools where humans paste form output into other systems.
- Test fixtures.

### 5.3 Why Not To Use It

```html
<form action="/save" method="post" enctype="text/plain">
  <input name="comment" type="text">
</form>
```

If `comment` is `subject=injected&malicious=true`, the server reading text/plain naively will see two extra fields. There is no escape mechanism. **Never use text/plain for anything that crosses a trust boundary.**

---

## 6. The Constraint Validation API

The HTML spec defines a precise vocabulary for validation, distinct from "is this input valid" intuition.

### 6.1 The Categories

- **Barred from constraint validation** — the element is exempted entirely. Examples: `<button type=button>`, disabled elements, readonly elements (but only inputs that cannot have invalid value), `<input type=hidden>`, fieldsets, output, datalist descendants.
- **Candidate for constraint validation** — eligible. Has a non-trivial `willValidate` getter returning true.

```javascript
input.willValidate; // true if subject to validation
```

### 6.2 The Validation Methods

| Method                   | Triggers UI? | Returns                | Side Effects                    |
|--------------------------|:------------:|------------------------|---------------------------------|
| `checkValidity()`         | No           | boolean                | Fires `invalid` event if false  |
| `reportValidity()`        | Yes          | boolean                | Fires `invalid` + shows browser error UI |
| `setCustomValidity(msg)`  | No           | undefined              | Sets `customError`; pass `""` to clear |

```javascript
if (!input.checkValidity()) {
  // input.validationMessage is browser-localized
  showCustomError(input.validationMessage);
}

// Or let the browser show its own popup:
input.reportValidity();
```

### 6.3 The ValidityState Interface

`input.validity` returns a live `ValidityState` object with these boolean flags:

| Flag              | True when                                                                 |
|-------------------|---------------------------------------------------------------------------|
| `valueMissing`    | `required` set and value is empty/unselected                              |
| `typeMismatch`    | Value doesn't match `type=email`, `type=url` syntax                       |
| `patternMismatch` | Value doesn't match `pattern` regex                                       |
| `tooLong`         | Value exceeds `maxlength` (only for user-edits, not programmatic)         |
| `tooShort`        | Value below `minlength` and user has interacted                           |
| `rangeUnderflow`  | Numeric/date below `min`                                                  |
| `rangeOverflow`   | Numeric/date above `max`                                                  |
| `stepMismatch`    | Numeric/date not on step grid (e.g. `step=0.5` but value is `1.3`)        |
| `badInput`        | User input cannot be converted (e.g. "abc" in `type=number`)              |
| `customError`     | `setCustomValidity(message)` has been called with non-empty string        |
| `valid`           | All other flags false. Convenience.                                       |

```javascript
const v = input.validity;
if (v.valueMissing) showError('Required');
else if (v.typeMismatch) showError('Invalid format');
else if (v.patternMismatch) showError('Doesn\'t match pattern');
else if (v.rangeUnderflow) showError(`Min ${input.min}`);
else if (v.rangeOverflow) showError(`Max ${input.max}`);
else if (v.stepMismatch) showError(`Step of ${input.step} required`);
else if (v.tooShort) showError(`At least ${input.minLength} chars`);
else if (v.tooLong) showError(`At most ${input.maxLength} chars`);
else if (v.badInput) showError('Cannot parse');
```

### 6.4 setCustomValidity Semantics

`setCustomValidity(message)` is the only way to introduce app-specific validation rules into the platform's flow:

```javascript
const password = form.elements.password;
const confirm = form.elements.confirm;

confirm.addEventListener('input', () => {
  if (confirm.value !== password.value) {
    confirm.setCustomValidity('Passwords do not match');
  } else {
    confirm.setCustomValidity(''); // clear
  }
});
```

While `customError` is true, `validity.valid` is false and `reportValidity()` will fail. Pass `""` (empty string) to clear.

### 6.5 form.checkValidity / form.reportValidity

The form-level versions iterate all submittable elements:

```javascript
if (!form.checkValidity()) {
  // At least one element is invalid; invalid event fired on each
}

form.reportValidity(); // shows the browser UI on the first invalid element
```

---

## 7. The :user-valid / :user-invalid CSS Pseudoclasses

The original `:valid` and `:invalid` pseudoclasses match **immediately**, even before the user has interacted. This produces the worst UX in the world — a fresh empty form glowing red.

### 7.1 The Old Pseudoclasses

```css
input:invalid {
  border-color: red; /* Fires on page load for empty required fields */
}
```

### 7.2 The New Pseudoclasses (Baseline 2023+)

```css
input:user-invalid {
  border-color: red; /* Fires only after user has interacted */
}

input:user-valid {
  border-color: green;
}
```

The `:user-invalid` pseudoclass becomes active when:

1. The user has changed the element's value (`input` event fired), AND it is now invalid.
2. OR the user has attempted submit, AND it is invalid.
3. OR the element has been blurred while invalid.

### 7.3 Canonical Pattern

```css
/* Default: neutral */
input {
  border: 1px solid #ccc;
}

/* User has touched and it's wrong: red */
input:user-invalid {
  border-color: #d33;
}

/* User has touched and it's right: green */
input:user-valid {
  border-color: #393;
}

/* Show error message only when user-invalid */
input:user-invalid + .error-message {
  display: block;
}

input + .error-message {
  display: none;
}
```

### 7.4 Browser Support Note

`:user-invalid` is Baseline 2023. For older browsers, the polyfill is to manually add an `interacted` class on first `blur` / first `change`:

```javascript
form.addEventListener('blur', (e) => {
  if (e.target.matches('input, select, textarea')) {
    e.target.classList.add('interacted');
  }
}, true);
```

```css
input.interacted:invalid { border-color: red; }
```

---

## 8. Form Validation Events

The `invalid` event is the spec's hook for reacting to validation failures.

### 8.1 When invalid Fires

`invalid` fires on a form-associated element when:

1. Its `checkValidity()` or `reportValidity()` returns false.
2. The form's submission algorithm runs interactive validation and finds it invalid.

It is fired **per-invalid-element**, not once per form.

### 8.2 Bubbling and Cancelability

`invalid` does **not** bubble (per spec). However, it is cancelable: calling `event.preventDefault()` suppresses the browser's default error UI for that element.

Because it does not bubble, attaching a listener on the form does NOT capture it via the bubble phase. Use the **capture** phase:

```javascript
form.addEventListener('invalid', (e) => {
  e.preventDefault(); // suppress default tooltip
  showCustomErrorFor(e.target);
}, true); // capture = true is the trick
```

### 8.3 Canonical Pattern: Summary on Submit

```javascript
form.addEventListener('submit', (e) => {
  if (!form.checkValidity()) {
    e.preventDefault();
    // form.checkValidity() fired invalid events on each invalid element
    // The capture-phase listener has already collected them.
    focusFirstInvalid();
  }
});

const errors = [];
form.addEventListener('invalid', (e) => {
  e.preventDefault();
  errors.push({ field: e.target.name, message: e.target.validationMessage });
}, true);

function focusFirstInvalid() {
  const first = form.querySelector(':invalid');
  if (first) first.focus();
  renderErrorSummary(errors);
  errors.length = 0;
}
```

### 8.4 The change Event vs. input Event

These fire on form controls during user interaction (not validation, but related):

- `input` — fires on every keystroke / value change. Use for live validation.
- `change` — fires on `blur` after change (text inputs) or immediately (checkboxes, radios, selects).

```javascript
input.addEventListener('input', () => {
  // Fires on each keystroke
  if (input.checkValidity()) clearError(input);
});

input.addEventListener('blur', () => {
  // Fires when user leaves field
  if (!input.checkValidity()) showError(input);
});
```

---

## 9. The formdata Event

The `formdata` event is the spec's hook for last-mile mutation of the entry list, fired after construct-entry-list and before encoding.

### 9.1 When It Fires

Exactly once per submission, after step 6 of the submission algorithm. It does NOT fire when calling `new FormData(form)` outside a submit. (To trigger it programmatically without submitting, you have to dispatch a synthetic event or use `form.requestSubmit()` and cancel the submit event — clumsy.)

### 9.2 The FormDataEvent Interface

```javascript
form.addEventListener('formdata', (e) => {
  // e.formData is the live FormData object
  e.formData.append('csrf_token', getCsrfToken());
  e.formData.set('client_timestamp', Date.now());
  e.formData.delete('debug_only');
});
```

The `formData` property is the actual data being submitted — mutations affect the wire-format request.

### 9.3 The Canonical Use Cases

**1. CSRF token injection without DOM pollution:**

```javascript
form.addEventListener('formdata', (e) => {
  e.formData.append('_csrf', readCsrfFromMetaTag());
});
```

**2. Adding state from JS that isn't in the DOM:**

```javascript
form.addEventListener('formdata', (e) => {
  e.formData.append('client_id', appState.clientId);
  e.formData.append('session_start', appState.sessionStart);
});
```

**3. Removing fields conditionally:**

```javascript
form.addEventListener('formdata', (e) => {
  if (!e.formData.get('remember-me')) {
    e.formData.delete('remember-me'); // don't send unchecked checkbox even as ""
  }
});
```

### 9.4 Why It Beats DOM Mutation

The pre-2020 way to inject a CSRF token was:

```javascript
form.addEventListener('submit', () => {
  const hidden = document.createElement('input');
  hidden.type = 'hidden';
  hidden.name = '_csrf';
  hidden.value = getCsrfToken();
  form.appendChild(hidden); // mutates DOM, fires MutationObservers, etc.
});
```

`formdata` does the same job without touching the DOM, without firing observers, without leaving an artifact in the form.

---

## 10. Form-Associated Custom Elements

Web Components participating in form submission. Available since Chrome 77 / Firefox 98 / Safari 16.4 (2022).

### 10.1 The Opt-In Flag

A custom element opts in via the static `formAssociated` field:

```javascript
class MyInput extends HTMLElement {
  static formAssociated = true;

  constructor() {
    super();
    this.internals_ = this.attachInternals();
  }

  // Reflect the value to the form's entry list
  set value(v) {
    this._value = v;
    this.internals_.setFormValue(v);
  }

  get value() { return this._value; }

  // Optional lifecycle callbacks:
  formAssociatedCallback(form) { /* attached to a form */ }
  formDisabledCallback(disabled) { /* fieldset disabled toggled */ }
  formResetCallback() { /* form.reset() called */ }
  formStateRestoreCallback(state, mode) { /* bfcache / autofill */ }
}

customElements.define('my-input', MyInput);
```

### 10.2 The ElementInternals Interface

`attachInternals()` returns an `ElementInternals` object with:

- `setFormValue(value, state?)` — write to entry list. `state` is for restore.
- `setValidity(flags, message?, anchor?)` — set ValidityState. `flags` is an object like `{ valueMissing: true }`. `anchor` is the element to focus on validation error.
- `checkValidity()` / `reportValidity()` — same as inputs.
- `validity`, `validationMessage`, `willValidate` — read-only mirror of platform contract.
- `form` — the associated form, or null.
- `labels` — `NodeList` of `<label>` elements.
- `states` — a `CustomStateSet` (see CSS `:state(...)` pseudoclass).

### 10.3 A Complete Form-Associated Element

```javascript
class StarRating extends HTMLElement {
  static formAssociated = true;
  static observedAttributes = ['name', 'required', 'value'];

  constructor() {
    super();
    this.internals_ = this.attachInternals();
    this.attachShadow({ mode: 'open' });
    this.shadowRoot.innerHTML = `
      <style>
        button { background: none; border: 0; cursor: pointer; }
        button[aria-pressed="true"] { color: gold; }
      </style>
      <button type="button" data-rating="1">★</button>
      <button type="button" data-rating="2">★</button>
      <button type="button" data-rating="3">★</button>
      <button type="button" data-rating="4">★</button>
      <button type="button" data-rating="5">★</button>
    `;
    this.shadowRoot.addEventListener('click', (e) => {
      const r = e.target.dataset.rating;
      if (r) this.value = r;
    });
  }

  get value() { return this._value || ''; }

  set value(v) {
    this._value = String(v);
    this.internals_.setFormValue(this._value);
    this._updateValidity();
    this._updateUI();
  }

  _updateValidity() {
    if (this.hasAttribute('required') && !this._value) {
      this.internals_.setValidity(
        { valueMissing: true },
        'Please select a rating',
        this.shadowRoot.querySelector('button')
      );
    } else {
      this.internals_.setValidity({});
    }
  }

  _updateUI() {
    this.shadowRoot.querySelectorAll('button').forEach(b => {
      b.setAttribute('aria-pressed', b.dataset.rating === this._value);
    });
  }

  formResetCallback() { this.value = ''; }
  formStateRestoreCallback(state) { this.value = state; }
}

customElements.define('star-rating', StarRating);
```

```html
<form>
  <label>Rate: <star-rating name="rating" required></star-rating></label>
  <button type="submit">Submit</button>
</form>
```

### 10.4 The Pre-2022 Workaround

Before form-associated custom elements, the only way to participate in form submission was to project a hidden `<input>` into light DOM:

```javascript
class OldStarRating extends HTMLElement {
  connectedCallback() {
    this._hidden = document.createElement('input');
    this._hidden.type = 'hidden';
    this._hidden.name = this.getAttribute('name');
    this.appendChild(this._hidden);
  }
}
```

This works for submission but does not integrate with `form.elements`, validation, labels, reset, or state restore. The new API replaces all of this.

---

## 11. Auto-completion — The Spec

The `autocomplete` attribute is documented in HTML §4.10.18.7 and is far more structured than "yes/no". It is parsed as a list of tokens.

### 11.1 The Token Grammar

```
autocomplete = "off" | "on" | autofill-detail-tokens

autofill-detail-tokens = [section-prefix] [shipping|billing] [home|work|mobile|fax|pager] field-name [webauthn]

section-prefix = "section-" + arbitrary-string
field-name = "name" | "given-name" | "family-name" | "email" | "username" | ...
```

### 11.2 The Vocabulary (Selected)

| Token             | For                                                |
|-------------------|----------------------------------------------------|
| `name`            | Full name                                          |
| `given-name`      | First name                                         |
| `family-name`     | Last name                                          |
| `nickname`        | Display name                                       |
| `email`           | Email                                              |
| `username`        | Login                                              |
| `current-password`| Existing password (login form)                     |
| `new-password`    | New password (signup, password change)             |
| `one-time-code`   | OTP (triggers SMS-OTP autofill on iOS / Android)   |
| `organization`    | Company                                            |
| `street-address`  | Multi-line street address                          |
| `address-line1`   | Street address line 1                              |
| `address-line2`   | Street address line 2                              |
| `address-level1`  | State / region                                     |
| `address-level2`  | City                                               |
| `country`         | Country code                                       |
| `country-name`    | Country name                                       |
| `postal-code`     | ZIP / postcode                                     |
| `cc-name`         | Cardholder name                                    |
| `cc-number`       | Credit card number                                 |
| `cc-exp`          | Expiration                                         |
| `cc-exp-month`    | Expiration month                                   |
| `cc-exp-year`     | Expiration year                                    |
| `cc-csc`          | CVC / CVV                                          |
| `tel`             | Full telephone                                     |
| `tel-national`    | Phone without country code                         |
| `bday`            | Birthday                                           |
| `language`        | Preferred language                                 |
| `url`             | Website                                            |

### 11.3 Section Prefixes

For multi-address forms, prefix with `section-<name>`:

```html
<fieldset>
  <legend>Shipping</legend>
  <input autocomplete="section-shipping shipping street-address" name="ship_address">
  <input autocomplete="section-shipping shipping postal-code" name="ship_zip">
</fieldset>

<fieldset>
  <legend>Billing</legend>
  <input autocomplete="section-billing billing street-address" name="bill_address">
  <input autocomplete="section-billing billing postal-code" name="bill_zip">
</fieldset>
```

The browser's password manager / autofill keeps the two sections separate.

### 11.4 Login vs. Signup Discrimination

```html
<!-- Login form -->
<input autocomplete="username" name="user">
<input autocomplete="current-password" type="password" name="pw">

<!-- Signup form -->
<input autocomplete="username" name="user">
<input autocomplete="new-password" type="password" name="pw">
<input autocomplete="new-password" type="password" name="pw_confirm">

<!-- Password change form -->
<input autocomplete="username" type="hidden" name="user">
<input autocomplete="current-password" type="password" name="old_pw">
<input autocomplete="new-password" type="password" name="new_pw">
```

The token-level distinction lets the browser's password manager:
- Suggest the existing password for login fields.
- Generate a new strong password for signup.
- Update the saved credential after a successful password-change submission.

### 11.5 SMS One-Time-Code Autofill

```html
<input autocomplete="one-time-code" inputmode="numeric" name="otp">
```

On iOS Safari and Android Chrome, when an SMS arrives matching the format `Your code is 123456` (or specific origin-bound formats), the browser surfaces the code on the input's keyboard suggestion bar. With Origin-Bound One-Time Codes (`@example.com #123456`) the autofill is automatic.

### 11.6 Disabling Autofill — When and How

`autocomplete="off"` is widely ignored by Chrome for password fields (security override). The reliable disable is:

```html
<input autocomplete="new-password" name="random_token">
<!-- Browser will offer to generate a password, won't autofill an existing one -->
```

For non-sensitive fields where autofill is unwanted (search, ephemeral data):

```html
<input autocomplete="off" name="search-query">
```

---

## 12. Inputs Deep Dive

### 12.1 input[type=number]

```html
<input type="number" name="qty" min="0" max="100" step="0.5" value="0">
```

- `min`, `max`, `step` participate in `rangeUnderflow`, `rangeOverflow`, `stepMismatch`.
- `step="any"` disables step validation.
- `valueAsNumber` returns the parsed number (or `NaN` for empty/invalid):

```javascript
input.valueAsNumber; // 42.5 (no parseFloat needed)
input.valueAsNumber = 100; // sets value="100"
```

**The locale-formatting issue**: the input renders the value using the user's locale (`1.234,56` in German), but `value` and `valueAsNumber` are always in canonical form (`1234.56`). This decouples display from wire format.

```javascript
// User in DE sees "1.234,56", types into the field
input.value;          // "1234.56" (canonical)
input.valueAsNumber;  // 1234.56
```

### 12.2 input[type=date], type=time, type=datetime-local

```html
<input type="date" name="dob" min="1900-01-01" max="2099-12-31">
<input type="time" name="appt" step="60">
<input type="datetime-local" name="meeting">
```

Wire format is always ISO 8601:

| Type             | Format                  | Example              |
|------------------|-------------------------|----------------------|
| `date`           | `YYYY-MM-DD`            | `2026-04-25`         |
| `time`           | `HH:MM` or `HH:MM:SS`   | `14:30`              |
| `datetime-local` | `YYYY-MM-DDTHH:MM`      | `2026-04-25T14:30`   |
| `month`          | `YYYY-MM`               | `2026-04`            |
| `week`           | `YYYY-Www`              | `2026-W17`           |

**The missing time-zone story**: `datetime-local` is, as the name suggests, a local time *with no time zone*. There is no `type=datetime` (it was removed from the spec). To collect a time-zoned timestamp you must combine `datetime-local` with a separate `select` for time zone, or compute the user's zone in JS:

```javascript
const localValue = input.value;      // "2026-04-25T14:30"
const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
// Server stores localValue + tz, or convert client-side via Date
```

```javascript
input.valueAsDate;        // Date object (UTC midnight for type=date)
input.valueAsNumber;      // Unix ms since epoch
```

### 12.3 input[type=file]

```html
<input type="file" name="photo" accept="image/*" multiple capture="environment">
```

Attributes:

- `accept` — comma-separated list of MIME types or extensions: `image/*`, `.pdf`, `audio/mp3,.wav`. Filters the OS file picker but does NOT validate post-selection (server must re-check).
- `multiple` — allows selecting multiple files.
- `capture="user" | "environment"` — on mobile, hints to use front (`user`) or back (`environment`) camera.

The DOM properties:

```javascript
input.files;          // FileList (array-like, not Array)
input.files[0];       // File (extends Blob)
input.files[0].name;  // "cat.jpg"
input.files[0].size;  // bytes
input.files[0].type;  // "image/jpeg"
input.files[0].lastModified; // Unix ms
```

**The File interface extends Blob**, so you can read it:

```javascript
const file = input.files[0];

// As ArrayBuffer (binary)
const buf = await file.arrayBuffer();

// As text
const text = await file.text();

// As stream
const reader = file.stream().getReader();

// Legacy FileReader
const fr = new FileReader();
fr.onload = (e) => console.log(e.target.result);
fr.readAsDataURL(file);
```

**Building FormData with files** (manual, e.g., for fetch):

```javascript
const fd = new FormData();
fd.append('title', 'My trip');
fd.append('photo', input.files[0], 'optional-override-name.jpg');

await fetch('/upload', { method: 'POST', body: fd });
// Browser auto-sets Content-Type: multipart/form-data with boundary
```

**Security: `input.value` for type=file cannot be set programmatically.** Only the user can populate it. Setting `input.value = ""` resets it (allowed) but `input.value = "/etc/passwd"` is a no-op (silently fails).

### 12.4 input[type=checkbox]

```html
<input type="checkbox" name="terms" value="accepted" required>
```

The `value` attribute is what gets sent if checked. If omitted, the value defaults to `"on"`.

```javascript
input.checked;       // boolean
input.value;         // string sent on submit
input.defaultChecked; // initial state
input.indeterminate; // tri-state (set in JS only)
```

**The `indeterminate` state is JS-only** — there is no HTML attribute. Use it for "select-all" checkboxes that are partially-selected:

```javascript
const all = document.querySelector('#select-all');
const items = document.querySelectorAll('.item');

function update() {
  const checked = [...items].filter(i => i.checked).length;
  if (checked === 0) {
    all.checked = false;
    all.indeterminate = false;
  } else if (checked === items.length) {
    all.checked = true;
    all.indeterminate = false;
  } else {
    all.indeterminate = true;
  }
}
```

The `indeterminate` state does NOT affect submission — checked/unchecked is what matters.

### 12.5 input[type=radio]

```html
<input type="radio" name="size" value="s" id="s">
<input type="radio" name="size" value="m" id="m" checked>
<input type="radio" name="size" value="l" id="l">
```

Radios are grouped by `name=`. The browser ensures only one in a group is checked at a time. **Tree order matters**: the first radio in tree order with `checked` wins on initial render.

`required` on **any** radio in the group makes the **whole group** required. The submission algorithm sends only the checked one's value (or skips the group entirely if none is checked).

```javascript
form.elements.size; // RadioNodeList (special)
form.elements.size.value; // value of the checked radio, or "" if none
form.elements.size.value = 'l'; // sets the matching radio to checked
```

### 12.6 input[type=range]

```html
<input type="range" name="volume" min="0" max="100" step="5" value="50">
```

A slider. Submits its current value.

**Event timing**:

- `input` fires continuously while the user drags.
- `change` fires on release (or arrow-key change).

```javascript
range.addEventListener('input', () => {
  // Live: every pixel of drag
  preview(range.value);
});

range.addEventListener('change', () => {
  // Once on release: commit
  commit(range.value);
});
```

This is unique to range — for text inputs `input` fires per-keystroke and `change` on blur.

### 12.7 select multiple

```html
<select name="tags" multiple size="5">
  <option value="a">A</option>
  <option value="b">B</option>
  <option value="c">C</option>
</select>
```

`size` controls the number of visible options (default for multiple is 4).

`selected` on multiple options sets initial state. JavaScript:

```javascript
[...select.selectedOptions].map(o => o.value); // ["a", "c"]
select.options[1].selected = true;
```

Each selected option produces a separate entry in the form data set with the same `name=`.

---

## 13. Accessibility Tree Integration

The platform builds an accessibility tree alongside the DOM, and form controls participate via specific attributes.

### 13.1 The Accessible Name Calculation

For a form control, the accessibility name is computed in this priority order (per ARIA Authoring Practices):

1. `aria-labelledby="ID1 ID2"` — concatenate text content of referenced elements.
2. `aria-label="..."` — use literal string.
3. `<label for="ID">` — the associated `<label>` text.
4. Implicit label (input wrapped by label): `<label>Name <input></label>` — the label text.
5. `title="..."` — fallback, not great UX (only shown on hover).
6. `placeholder="..."` — last resort, not a real label.

```html
<!-- Best: explicit label -->
<label for="username">Username</label>
<input id="username" name="user">

<!-- Also fine: wrapping label -->
<label>
  Username
  <input name="user">
</label>

<!-- For complex labels, aria-labelledby -->
<h3 id="addr-h">Shipping Address</h3>
<input aria-labelledby="addr-h addr-line" name="line1">
<span id="addr-line">Line 1</span>

<!-- Fallback (suboptimal): aria-label -->
<input aria-label="Search" name="q">
```

### 13.2 Implicit Labels Have a Catch

Wrapping a single input is fine. **Wrapping multiple is not** — only the first is associated:

```html
<label>
  Name
  <input name="first"> <!-- labelled "Name" -->
  <input name="last">  <!-- NOT labelled -->
</label>
```

Always use explicit `for=` for multi-input fields:

```html
<label for="first">First</label>
<input id="first" name="first">
<label for="last">Last</label>
<input id="last" name="last">
```

### 13.3 Describing the Field — aria-describedby

```html
<label for="pw">Password</label>
<input id="pw" type="password" aria-describedby="pw-help pw-error">
<p id="pw-help">At least 12 characters, including a number.</p>
<p id="pw-error" role="alert" hidden>Password is too short.</p>
```

Screen readers announce the help text after the label. The `role="alert"` on the error region causes screen readers to announce changes when the error becomes visible.

### 13.4 Required vs aria-required

Use `required` (the HTML attribute) for actual validation. It is also exposed to the accessibility tree as `aria-required="true"`. Adding `aria-required` redundantly is harmless but unnecessary.

```html
<!-- Idiomatic -->
<label for="email">Email</label>
<input id="email" type="email" name="email" required>
```

For form-associated custom elements that don't have a real `required` attribute, you DO need `aria-required="true"`.

### 13.5 aria-invalid on Validation Failure

When validation fails, mirror the state to assistive technology:

```javascript
form.addEventListener('invalid', (e) => {
  e.target.setAttribute('aria-invalid', 'true');
}, true);

input.addEventListener('input', () => {
  if (input.checkValidity()) {
    input.removeAttribute('aria-invalid');
  }
});
```

Combined with `aria-describedby`:

```html
<label for="email">Email</label>
<input id="email" type="email" required
       aria-describedby="email-error" aria-invalid="true">
<p id="email-error" role="alert">Please enter a valid email.</p>
```

### 13.6 Focus Management on Submit-with-Errors

When submission fails validation, focus must move to the first invalid field (programmatically):

```javascript
form.addEventListener('submit', (e) => {
  if (!form.checkValidity()) {
    e.preventDefault();
    const first = form.querySelector(':invalid:not(fieldset)');
    if (first) first.focus();
  }
});
```

The browser's default `reportValidity()` does this for you. If you suppress it with `e.preventDefault()` in `invalid` handlers, you must do it yourself.

### 13.7 Fieldset and Legend

`<fieldset>` groups related controls with a `<legend>` as its accessible name. Screen readers announce the legend as context for each enclosed control:

```html
<fieldset>
  <legend>Notification preferences</legend>
  <label><input type="checkbox" name="notify" value="email"> Email</label>
  <label><input type="checkbox" name="notify" value="sms"> SMS</label>
</fieldset>
```

A screen reader reads: "Notification preferences, Email, checkbox, not checked." — it includes the legend.

---

## 14. CSRF Protection — Server-side Pairing

CSRF (Cross-Site Request Forgery) protection is necessary whenever a form modifies server state with cookies-as-credentials.

### 14.1 The Synchronizer Token Pattern

Server generates a per-session random token, stores it (session map), and renders it in the form:

```html
<form action="/transfer" method="post">
  <input type="hidden" name="csrf_token" value="a1b2c3d4...">
  <input name="amount" type="number">
  <button type="submit">Send</button>
</form>
```

On submit, the server compares `request.body.csrf_token` to the session-stored token. Mismatch → 403.

### 14.2 The Double-Submit Cookie Pattern

Server sets a cookie containing a random token AND injects the same token into the form. On submit, server compares `request.body.csrf_token === request.cookies.csrf_token`.

Advantage: stateless on server (no session map needed).

```http
Set-Cookie: csrf_token=a1b2c3d4; SameSite=Lax; Secure
```

```html
<input type="hidden" name="csrf_token" value="a1b2c3d4">
```

Server checks equality.

### 14.3 SameSite Cookies — The Modern Mitigation

`SameSite=Strict` cookies are not sent on cross-site requests (including cross-site form submissions):

```http
Set-Cookie: session=...; SameSite=Strict; Secure; HttpOnly
```

This **alone** prevents CSRF for state-changing endpoints, because a malicious site's submitted form arrives without the session cookie.

`SameSite=Lax` (the default in modern browsers) allows the cookie on top-level GET navigations but blocks it on POST cross-site, which prevents most CSRF.

### 14.4 Origin Header Check

Browsers send `Origin: https://example.com` on cross-origin requests. Server can verify:

```javascript
// Express
app.post('/transfer', (req, res) => {
  if (req.get('Origin') !== 'https://example.com') return res.sendStatus(403);
  // ...process
});
```

This is increasingly viable as `Origin` is sent on all CORS-eligible requests.

### 14.5 Framework Examples

**Express** (csurf middleware):

```javascript
const csrf = require('csurf');
app.use(csrf({ cookie: true }));
app.get('/form', (req, res) => res.render('form', { token: req.csrfToken() }));
```

**Django**:

```html
{% csrf_token %}
<!-- expands to <input type="hidden" name="csrfmiddlewaretoken" value="..."> -->
```

**Rails**:

```erb
<%= form_with(url: "/save") do |f| %>
  <!-- automatic <input type="hidden" name="authenticity_token" ...> -->
<% end %>
```

### 14.6 The formdata Pattern for SPAs

For client-rendered forms in SPAs, inject the token via `formdata`:

```javascript
form.addEventListener('formdata', (e) => {
  e.formData.append('csrf_token', getMetaContent('csrf-token'));
});
```

Combined with a server-rendered `<meta name="csrf-token" content="...">` tag, this avoids per-form template duplication.

---

## 15. The Browser Autofill UX

Autofill is a user-agent feature, but the form's markup determines the quality.

### 15.1 What the Browser Looks For

The browser/password manager scans:

- `autocomplete=` tokens (highest priority, most reliable).
- `name=` attributes matching common patterns (`username`, `password`, `email`, `firstname`).
- `id=` attributes (less reliable).
- `placeholder=` text (heuristic, fragile).
- Surrounding `<label>` text.

Heuristics work but are best-effort; explicit `autocomplete=` is the lever.

### 15.2 The Canonical Login Form

```html
<form method="post" action="/login">
  <label for="user">Username or email</label>
  <input id="user" name="user" type="text" autocomplete="username" required>

  <label for="pw">Password</label>
  <input id="pw" name="password" type="password" autocomplete="current-password" required>

  <label>
    <input type="checkbox" name="remember">
    Remember me
  </label>

  <button type="submit">Sign in</button>
</form>
```

This form will:

- Get autofill suggestions from the password manager on the user/password fields.
- Trigger "save password?" on successful submit (browser detects via XHR success or navigation).
- Surface saved credentials in the OS-level password manager (Touch ID / Face ID).

### 15.3 The Canonical Signup Form

```html
<form method="post" action="/signup">
  <label for="email">Email</label>
  <input id="email" name="email" type="email" autocomplete="email" required>

  <label for="user">Username</label>
  <input id="user" name="user" type="text" autocomplete="username" required>

  <label for="pw">Password</label>
  <input id="pw" name="password" type="password" autocomplete="new-password"
         minlength="12" required>

  <label for="pw2">Confirm password</label>
  <input id="pw2" name="password_confirm" type="password" autocomplete="new-password" required>

  <button type="submit">Create account</button>
</form>
```

The `autocomplete="new-password"` prompts the password manager to **generate** a strong password rather than offer existing ones.

### 15.4 The Multi-Step Wizard Pattern

When breaking a long form into steps, use `section-` prefixes so autofill works across steps:

```html
<!-- Step 1 -->
<form id="step1">
  <input autocomplete="section-shipping shipping name" name="ship_name">
  <input autocomplete="section-shipping shipping street-address" name="ship_addr">
</form>

<!-- Step 2 (different form, same section) -->
<form id="step2">
  <input autocomplete="section-shipping shipping postal-code" name="ship_zip">
  <input autocomplete="section-shipping shipping country" name="ship_country">
</form>
```

The browser treats `section-shipping` as a coherent unit even across forms.

### 15.5 The Hidden Username Trick

For password-change forms (where the user is already authenticated and the username is implicit), include a hidden `username` field:

```html
<form method="post" action="/change-password">
  <input type="hidden" name="username" value="<%= currentUser.email %>"
         autocomplete="username">

  <input name="old" type="password" autocomplete="current-password">
  <input name="new" type="password" autocomplete="new-password">
  <input name="confirm" type="password" autocomplete="new-password">
</form>
```

This lets the password manager associate the credential update with the right user.

---

## 16. The dialog Element Integration

`<dialog>` integrates with form submission via `method="dialog"`.

### 16.1 The Pattern

```html
<dialog id="confirm">
  <form method="dialog">
    <p>Delete this item?</p>
    <button value="cancel">Cancel</button>
    <button value="delete">Delete</button>
  </form>
</dialog>

<button onclick="document.getElementById('confirm').showModal()">Delete...</button>
```

```javascript
const dialog = document.getElementById('confirm');
dialog.addEventListener('close', () => {
  if (dialog.returnValue === 'delete') {
    performDelete();
  }
});
```

### 16.2 What method="dialog" Does

When the form is submitted (via button click or implicit submission):

1. The submission algorithm runs *up to* the navigation step.
2. Instead of navigating, the dialog's `close()` is called.
3. `dialog.returnValue` is set to the submitter button's `value=` attribute.
4. The `close` event fires on the dialog.

No HTTP request, no navigation. The form is purely an interactive UI for collecting a result.

### 16.3 Combining with Validation

Forms inside `<dialog method="dialog">` still run validation:

```html
<dialog id="rename">
  <form method="dialog">
    <label for="newname">New name</label>
    <input id="newname" name="newname" required minlength="1" maxlength="50">
    <button value="cancel" formnovalidate>Cancel</button>
    <button value="save">Save</button>
  </form>
</dialog>
```

`formnovalidate` on Cancel skips validation so the user can always escape.

### 16.4 With formdata

```javascript
dialog.querySelector('form').addEventListener('formdata', (e) => {
  console.log('user entered:', Object.fromEntries(e.formData));
});

dialog.addEventListener('close', () => {
  console.log('dialog closed with:', dialog.returnValue);
});
```

The `formdata` event still fires (after construct-entry-list, before "navigation"). Even though there's no real navigation, the event cycle runs.

---

## 17. Performance Patterns

### 17.1 Debounced Live Validation

Running `checkValidity()` on every keystroke is fine for simple inputs. For expensive checks (regex, server lookups), debounce:

```javascript
let t;
input.addEventListener('input', () => {
  clearTimeout(t);
  t = setTimeout(() => validate(input), 250);
});
```

For server-side uniqueness checks (e.g., username availability), debounce more aggressively (500ms+) and abort in-flight fetches:

```javascript
let abort;
input.addEventListener('input', () => {
  clearTimeout(t);
  if (abort) abort.abort();
  t = setTimeout(async () => {
    abort = new AbortController();
    try {
      const r = await fetch(`/check?u=${input.value}`, { signal: abort.signal });
      const { available } = await r.json();
      input.setCustomValidity(available ? '' : 'Username taken');
    } catch (e) { /* aborted */ }
  }, 500);
});
```

### 17.2 Defer Expensive Checks Until Blur

```javascript
input.addEventListener('blur', () => {
  if (input.value && !isExpensiveValid(input.value)) {
    input.setCustomValidity('...');
  }
});
input.addEventListener('input', () => {
  // Cheap checks only; clear customError as user fixes
  input.setCustomValidity('');
});
```

### 17.3 The Cost of validity-state Recomputation

Every keystroke triggers a re-evaluation of `validity` for that element. The cost is small for simple inputs (boolean checks against `min`/`max`/`pattern`) but adds up in large forms with hundreds of fields. The `:user-invalid` and `:user-valid` pseudoclasses also re-evaluate, potentially triggering style invalidation.

For very large forms (>200 inputs), consider:

- Using `pattern` sparingly (each pattern is a regex compile + match per keystroke).
- Avoiding `:invalid` selectors with expensive children (`input:invalid + .badge { ... animations ... }`).
- Validating on submit only, with field-level validation in submit handler.

### 17.4 FormData vs URLSearchParams vs JSON

Construction cost comparison for a 50-field form:

| API                        | Cost   | Notes                                         |
|----------------------------|--------|-----------------------------------------------|
| `new FormData(form)`       | O(N)   | Walks form.elements, builds entry list        |
| `new URLSearchParams(...)` | O(N)   | From an iterable; for url-encoded GET         |
| `JSON.stringify({...})`    | O(N)   | Manual object construction                    |

```javascript
// FormData (works with files, multipart automatic)
fetch(url, { method: 'POST', body: new FormData(form) });

// URLSearchParams (url-encoded body, no files)
const params = new URLSearchParams(new FormData(form));
fetch(url, {
  method: 'POST',
  headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: params,
});

// JSON (manual)
const obj = Object.fromEntries(new FormData(form));
fetch(url, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(obj),
});
```

`Object.fromEntries(new FormData(form))` collapses multi-valued fields (radios, multi-selects) to last-wins. To preserve all values, iterate manually.

### 17.5 Lazy DOM Reads in Validation

Re-reading `input.value` is cheap (cached property), but reading `getComputedStyle(input)` per keystroke is not. Cache layout reads outside hot paths.

---

## 18. Common Pitfalls

### 18.1 Button Without type Defaults to submit

**Bad**:

```html
<form>
  <input name="search">
  <button onclick="filter()">Filter</button>  <!-- defaults to type=submit -->
</form>
<!-- Clicking Filter SUBMITS the form, navigating away. -->
```

**Fixed**:

```html
<form>
  <input name="search">
  <button type="button" onclick="filter()">Filter</button>
</form>
```

Always specify `type=` on `<button>` inside forms. Default is `type=submit`.

### 18.2 Forgetting name= Attribute

**Bad**:

```html
<form>
  <input id="email" type="email" required>
  <!-- No name=. NOT serialized. Server gets nothing. -->
  <button type="submit">Send</button>
</form>
```

**Fixed**:

```html
<form>
  <input id="email" name="email" type="email" required>
  <button type="submit">Send</button>
</form>
```

### 18.3 Browser Default Validation Message Styling

**Bad** (cannot style the browser's tooltip):

```html
<input required>
<!-- Browser shows "Please fill in this field" in browser-native UI you cannot CSS. -->
```

**Fixed** — replace via custom validity:

```javascript
input.addEventListener('invalid', (e) => {
  e.preventDefault(); // suppress browser tooltip
  showBrandedError(input, input.validationMessage);
});

input.addEventListener('input', () => {
  input.setCustomValidity(''); // clear stale custom message
  if (input.validity.valueMissing) {
    input.setCustomValidity('Please enter your email');
  }
});
```

### 18.4 Multiple Submit Buttons + Enter Key

**Bad** (ambiguous):

```html
<form>
  <input name="q">
  <button type="submit" onclick="searchProducts()">Search</button>
  <button type="submit" onclick="searchHelp()">Help</button>
</form>
<!-- Pressing Enter triggers the FIRST submit button (Search), not the second. -->
```

The implicit submitter is the first submit button in tree order. To prevent this:

```html
<form>
  <input name="q">
  <button type="button" onclick="searchProducts()">Search</button>
  <button type="button" onclick="searchHelp()">Help</button>
</form>
<!-- Or: handle keydown on the form, prevent default -->
```

Or design with an explicit submit button hidden if needed:

```html
<form>
  <input name="q">
  <button type="submit" formaction="/search">Search</button>
  <button type="submit" formaction="/help">Help</button>
</form>
<!-- Enter triggers Search. Click on Help triggers Help. -->
```

### 18.5 input.value for Files

**Bad**:

```javascript
const fileInput = document.querySelector('input[type=file]');
fileInput.value = '/Users/me/Pictures/cat.jpg'; // silently ignored
```

You **cannot** programmatically set a file input's value. Only the user can pick a file (security constraint to prevent silent file uploads).

**The only allowed assignment**:

```javascript
fileInput.value = ''; // resets the file input
```

To pre-fill, you must use a `<input type=hidden>` companion:

```html
<input type="file" name="photo">
<input type="hidden" name="photo_id" value="<%= existingPhotoId %>">
```

The server uses `photo_id` if `photo` is empty.

### 18.6 Trusting Client-side Validation

**Bad** (security):

```javascript
form.addEventListener('submit', (e) => {
  if (input.value.length < 12) {
    e.preventDefault();
    showError('Password too short');
    return;
  }
  // proceed
});
```

The server MUST re-validate. Client-side validation is UX, not security.

### 18.7 Forgetting accept Doesn't Validate

```html
<input type="file" accept=".jpg,.png">
```

`accept=` filters the OS picker but **does not validate** if the user picks "All files" and selects `evil.exe`. Always re-check on the server (or even client-side post-select):

```javascript
input.addEventListener('change', () => {
  const file = input.files[0];
  if (!file.type.startsWith('image/')) {
    input.setCustomValidity('Must be an image');
    input.reportValidity();
    input.value = '';
  }
});
```

---

## 19. Idioms at the Internals Depth

### 19.1 The Robust Submit Pattern

Combines validation, fetch, error display, focus management:

```javascript
async function handleSubmit(form, e) {
  e.preventDefault();

  // 1. Run platform validation
  if (!form.checkValidity()) {
    form.reportValidity(); // browser shows error UI
    return;
  }

  // 2. Build request body
  const body = new FormData(form);

  // 3. Disable form during submit
  const btn = form.querySelector('button[type=submit]');
  btn.disabled = true;
  btn.textContent = 'Saving...';

  try {
    const r = await fetch(form.action, {
      method: form.method.toUpperCase(),
      body,
      headers: { 'Accept': 'application/json' },
    });

    if (!r.ok) {
      // Server returned validation errors (e.g., {field: "email", message: "Already taken"})
      const errors = await r.json();
      for (const { field, message } of errors) {
        const el = form.elements[field];
        if (el) el.setCustomValidity(message);
      }
      form.reportValidity();
      return;
    }

    // Success
    const data = await r.json();
    onSuccess(data);
  } catch (err) {
    showError('Network error');
  } finally {
    btn.disabled = false;
    btn.textContent = 'Save';
  }
}

form.addEventListener('submit', (e) => handleSubmit(form, e));

// Clear customError as the user types
form.addEventListener('input', (e) => {
  e.target.setCustomValidity('');
});
```

### 19.2 The Multi-Step Wizard via Hidden State

```html
<form method="post" action="/checkout" id="wizard">
  <input type="hidden" name="step" value="1">

  <div data-step="1">
    <label>Email <input name="email" type="email" required></label>
    <button type="submit" name="action" value="next">Next</button>
  </div>

  <div data-step="2" hidden>
    <label>Address <input name="addr" required></label>
    <button type="submit" name="action" value="back">Back</button>
    <button type="submit" name="action" value="next">Next</button>
  </div>

  <div data-step="3" hidden>
    <label>Payment <input name="cc" required></label>
    <button type="submit" name="action" value="back">Back</button>
    <button type="submit" name="action" value="confirm">Confirm</button>
  </div>
</form>
```

The server inspects `step` and `action` to render the correct next view. The form preserves all entered data on each round-trip via hidden inputs.

### 19.3 Routing via formaction

Use `formaction` to send the same form to different endpoints based on which button was clicked:

```html
<form method="post" action="/article/save">
  <textarea name="body"></textarea>

  <button type="submit">Publish</button>
  <button type="submit" formaction="/article/draft">Save as Draft</button>
  <button type="submit" formaction="/article/preview" formtarget="_blank">Preview</button>
  <button type="submit" formaction="/article/delete"
          formnovalidate
          onclick="return confirm('Delete?')">Delete</button>
</form>
```

The submitter button determines the action URL. `formnovalidate` on Delete bypasses the body-required validation. `formtarget="_blank"` opens preview in a new tab.

### 19.4 Conditional Required via formdata

When a field is required only sometimes (e.g., based on another field's value), the cleanest approach is to skip platform `required` and validate via `formdata`:

```html
<form>
  <select name="contact_method">
    <option value="email">Email</option>
    <option value="phone">Phone</option>
  </select>

  <input name="email" type="email" id="email">
  <input name="phone" type="tel" id="phone">

  <button type="submit">Submit</button>
</form>
```

```javascript
form.addEventListener('formdata', (e) => {
  const method = e.formData.get('contact_method');
  const value = e.formData.get(method);
  if (!value) {
    e.formData.append('_validation_error', `${method} required`);
    // But formdata can't cancel — use submit handler instead
  }
});

form.addEventListener('submit', (e) => {
  const method = form.elements.contact_method.value;
  const target = form.elements[method];
  if (!target.value) {
    e.preventDefault();
    target.setCustomValidity(`Please provide your ${method}`);
    target.reportValidity();
  }
});
```

### 19.5 Idempotent Server-side via formdata Token

```javascript
form.addEventListener('formdata', (e) => {
  // Generate a per-submit nonce so the server can dedupe duplicate submissions
  e.formData.append('_idempotency_key', crypto.randomUUID());
});
```

The server stores the key with the result; re-submission with the same key returns the cached result. Useful for double-click protection.

### 19.6 Progressive Enhancement

A form that works with no JavaScript at all, but enhances when JS is available:

```html
<form method="post" action="/save" data-enhance>
  <input name="title" required>
  <button type="submit">Save</button>
</form>
```

```javascript
document.querySelectorAll('form[data-enhance]').forEach(form => {
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const r = await fetch(form.action, { method: 'POST', body: new FormData(form) });
    if (r.ok) showSuccess(); else form.submit(); // fallback to native submit on error
  });
});
```

If JS fails to load, the form still submits the old-fashioned way and renders a server response. This is the original spirit of HTML.

---

## 20. Prerequisites

- HTML basics: forms, inputs, attributes (sheets/web/html.md).
- HTTP basics: GET vs POST, request/response, headers, status codes.
- DOM events: bubbling, capturing, `preventDefault()`.
- JavaScript fundamentals: addEventListener, async/await, fetch.
- ES6 collections: `FormData`, `URLSearchParams`, iterators.
- Basic accessibility model: ARIA roles, labels, focus.

---

## 21. Complexity

| Operation                              | Cost                       |
|----------------------------------------|----------------------------|
| `construct entry list`                 | O(N) where N = form controls |
| `application/x-www-form-urlencoded`    | O(M) where M = total bytes |
| `multipart/form-data` (no files)       | O(M)                       |
| `multipart/form-data` (with files)     | O(M + F) F = file bytes    |
| `checkValidity()` per element          | O(1) — flags are pre-computed by the input's intrinsic checks (regex match for pattern is O(L) for value length L) |
| `form.checkValidity()`                 | O(N)                       |
| `:user-invalid` style invalidation     | O(1) per change, but triggers descendant re-style |

For a typical login form (5 inputs, 100 bytes total), submission overhead is microseconds. For a large form with file uploads, the cost is dominated by network and file I/O, not the algorithm.

---

## 22. See Also

- html-forms (sheet)
- html
- css
- javascript
- polyglot

---

## 23. References

- HTML Living Standard, §4.10 Forms — https://html.spec.whatwg.org/multipage/forms.html
- HTML Living Standard, §4.10.5 The input element — https://html.spec.whatwg.org/multipage/input.html
- HTML Living Standard, §4.10.21 Form submission — https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#form-submission-2
- HTML Living Standard, §4.10.18.7 Autofill — https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#autofill
- RFC 7578: Returning Values from Forms: multipart/form-data — https://datatracker.ietf.org/doc/html/rfc7578
- RFC 3986: URI Generic Syntax (percent-encoding) — https://datatracker.ietf.org/doc/html/rfc3986
- MDN: HTML element reference — https://developer.mozilla.org/en-US/docs/Web/HTML/Element
- MDN: input element — https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input
- MDN: ValidityState — https://developer.mozilla.org/en-US/docs/Web/API/ValidityState
- MDN: FormData — https://developer.mozilla.org/en-US/docs/Web/API/FormData
- MDN: ElementInternals — https://developer.mozilla.org/en-US/docs/Web/API/ElementInternals
- MDN: HTMLFormElement.requestSubmit() — https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement/requestSubmit
- MDN: Constraint validation — https://developer.mozilla.org/en-US/docs/Web/HTML/Constraint_validation
- web.dev: Learn Forms — https://web.dev/articles/learn/forms
- WAI-ARIA Authoring Practices: Forms — https://www.w3.org/WAI/ARIA/apg/patterns/
- WHATWG DOM Standard, Events — https://dom.spec.whatwg.org/#events
- W3C ARIA 1.2: aria-invalid, aria-required — https://www.w3.org/TR/wai-aria-1.2/
