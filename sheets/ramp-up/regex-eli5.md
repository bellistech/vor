# Regex — ELI5

> A regex is a fishing net you weave out of letters. The shape of the net decides what you catch. Cast it across a pile of words and only the words that match the holes come back stuck inside.

## Prerequisites

You should be comfortable enough with a terminal to run `grep`, `sed`, or `python3` and read what they print. If a `$` appears at the start of a code block, that means "type the rest of the line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. If you have never opened a terminal in your life, walk through `ramp-up/bash-eli5` first, then come back here.

You do **not** need to know how to program. You do **not** need to have read any "regex tutorial" before. By the end of this sheet you will know what a regex is, why it exists, what every weird symbol means, why some regex programs are slow and dangerous and others are fast and safe, and you will have actually run regexes against real text and watched them grab the right pieces.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

## Plain English

### Imagine you are fishing through a giant tub of letters

Picture a giant tub. The tub is full of letters. Just letters, floating around. There are billions of them. They are arranged into words, and words into sentences, and sentences into pages, but really, deep down, it's just an enormous tub of letters in a row.

You want to fish out certain things. Maybe you want to fish out every email address. Maybe you want to fish out every phone number. Maybe you want to fish out every line that says the word "error." You don't want to grab one specific email by name. You want to grab **anything that looks like an email**. You want to describe the **shape** of an email and let your tool find every letter-soup in the tub that matches that shape.

The tool you use to do this is called a **regex**. A regex is a tiny little **fishing net** that you weave by typing some symbols. The shape of the net decides what gets caught. The holes in your net are shaped like the things you want, and only the letters that fit through the holes come back when you pull the net out.

Let's make this concrete. Say the tub of letters is just one line:

```
my email is bob@example.com and stevie@bellis.tech and please write back
```

If your net is the regex `bob`, it grabs the letters `b`, `o`, `b` sitting next to each other. That's a tiny net with three holes shaped like b, o, b.

If your net is `\d+`, it grabs any chunk of one-or-more digits. That line above has no digits, so the net comes back empty. But if you cast it into:

```
order 1234 ships on day 5
```

then `\d+` catches `1234` and `5`. The net pulled back two fish.

If your net is `[a-z]+@[a-z.]+`, it grabs any chunk that looks like "a string of lowercase letters, then an @, then more lowercase letters and dots." That is the **shape** of a simple email. Cast that net into the original line and you fish out `bob@example.com` and `stevie@bellis.tech`. Two fish.

That is the whole big idea. **A regex is a description of a shape, and a regex engine is a fish-finder that runs your description against text and hands you back every chunk that matches.**

### Why we don't just look for exact words

You might be thinking, "couldn't I just type the words I'm looking for and search for them?" Yes — for one specific word, you can. If you only ever wanted to find the literal word `bob`, then a plain old "find" feature in any text editor would do it. You don't need regex for that.

But what if you don't know the word ahead of time? What if you want to find **all** the email addresses in a thousand-page document, and the email addresses are different in every line? You can't type each one in advance — you don't know what they are. You only know what they look like. They look like "letters, then @, then letters, then a dot, then more letters."

A regex lets you describe a **family of strings** instead of just one specific string. The family of all email addresses. The family of all phone numbers. The family of all dates. The family of all error messages that mention a particular word. The family of all lines that start with a capital letter. Anything you can describe with the words "what character could be here, and how many," you can write down as a regex.

That's why regex exists. It is the language for describing shapes of text.

### Imagine the net more literally

Imagine an actual fishing net laid flat on a table. The net has rows of holes, and each hole is shaped to let only certain letters slip through.

A net with a single round hole shaped like the letter `a` only catches the letter `a`. A net with a round hole shaped like `b` only catches `b`. So if I want a net that catches exactly the word `dog`, I weave three round holes side by side: one shaped like `d`, one like `o`, one like `g`. To catch the word, all three holes have to match in order. That is a regex made entirely of **literal characters**.

But what if I want a net where the first hole catches **any** of `a`, `b`, or `c`? I weave a triangular hole. The triangle is wide enough to let `a` through, or `b` through, or `c` through, but nothing else. The shorthand for "make this hole catch a, b, or c" is `[abc]`. The square brackets are the way regex says "any one of these." So `[abc]at` catches `aat`, `bat`, or `cat`, but not `dat`.

What if I want a hole that catches **any digit**? Instead of writing `[0123456789]` (which works fine), I can use the shorthand `\d`, which means "any single digit." Same idea, less typing.

What if I want a hole that catches **any lowercase letter**? I write `[a-z]`. The dash means "from-to." So `[a-z]` is "any one character in the range a through z."

What if I want a hole that catches **any character that is NOT a digit**? I write `[^0-9]` or its shorthand `\D`. The `^` inside square brackets means "not."

So far we have **single-character holes**: each hole catches exactly one character. The clever bit is when we say "I want this hole repeated."

### Repeating a hole

A row of holes side by side, each catching one character, is fine for short fixed words. But emails and phone numbers and IPs are different lengths. We need a way to say "match this kind of character, an unknown number of times in a row."

That is what **quantifiers** are for. A quantifier is a little symbol you put right after a hole that says "and again, and again."

- `*` means **zero or more times**. So `a*` matches "" (nothing), or `a`, or `aa`, or `aaa`, etc.
- `+` means **one or more times**. So `a+` matches `a`, `aa`, `aaa`, but NOT "" (nothing).
- `?` means **zero or one time**. So `a?` matches "" or `a`, but not `aa`.
- `{n}` means **exactly n times**. `a{3}` matches `aaa` and only `aaa`.
- `{n,}` means **at least n times**. `a{3,}` matches `aaa`, `aaaa`, `aaaaa`, etc.
- `{n,m}` means **between n and m times**. `a{2,4}` matches `aa`, `aaa`, or `aaaa`.

Pair quantifiers with the character classes from earlier and you can describe any shape:

- `\d+` = one or more digits in a row → `5`, `42`, `1234567`.
- `[a-z]+` = one or more lowercase letters → `hi`, `hello`, `aaaaa`.
- `\d{3}-\d{4}` = three digits, a dash, four digits → `555-1212`.
- `[a-zA-Z0-9]+@[a-z]+\.[a-z]+` = a basic email shape.

That last one is the email net we already saw, slightly tightened. Letters or digits, then an @, then letters, then a literal dot, then letters. The `\.` is important: a plain `.` in regex means "any character at all," so to match a real dot we have to **escape** it with a backslash.

### Why the dot is special

You will trip over this constantly: the symbol `.` in a regex does NOT mean "match a literal period." It means "match any one character at all." Any letter, any digit, any space, any punctuation — anything except (usually) a newline.

So `b.t` matches `bat`, `bit`, `but`, `b9t`, `b@t`, `b t`. Three characters, b-anything-t.

If you actually want to match a real dot in the text — like the dot in a domain name — you write `\.`. The backslash says "treat the next character as literal." So `example\.com` matches the exact string `example.com`, while `example.com` (without the backslash) matches `example.com` and also `examplexcom`, `example9com`, `example com`, etc.

This is one of the most common bugs. Almost every "my regex matches too much" bug starts with somebody forgetting to escape a dot.

### A second analogy: the cookie cutter

Another way to picture a regex is as a **cookie cutter** stamping into rolled-out dough. The dough is your text. The cookie cutter is your regex. The cutter's shape is the pattern. Wherever the cutter's shape lines up with the dough, you get a cookie. You stamp the whole sheet of dough and the regex engine hands you back every cookie.

A simple cookie cutter cuts one shape: `dog` cuts a piece of dough exactly shaped `dog`. A flexible cookie cutter has soft edges. `\d+` is a stretchy cutter that cuts whatever-length-of-digits it finds. `[a-z]+` is a stretchy cutter that grabs any run of lowercase letters.

Some cookie cutters have **anchors** — pegs that have to line up with the edge of the dough. `^` is a peg that has to touch the **start of the line**. `$` is a peg that has to touch the **end of the line**. So `^error` only cuts the word `error` if it is at the very start of a line. And `done$` only cuts `done` if it is at the very end. Anchors don't catch a character — they're just rules about where the rest of the cutter is allowed to land.

You can put anchors at both ends: `^\d{4}$` says "the entire line must be exactly four digits." Anchored at the start, anchored at the end, with `\d{4}` (four digits) in between.

### A third analogy: the police sketch

If you have ever seen a police sketch, an artist draws a face based on a witness's description. The witness says "tall guy, brown hair, scar over the left eye." The artist sketches a face. The police don't look for one exact known person — they look for **anyone matching the sketch**.

A regex is the sketch. It is not one specific suspect; it is a description that matches a whole family of suspects. The regex engine is the police officer with the sketch in hand walking through a crowd, comparing every face. Faces that match get pulled aside.

### What is "matching" really doing?

Let's get a tiny bit precise. When you ask a regex engine "does this regex match this text?", it tries to walk through the text and the regex side by side, character by character, and see if everything lines up.

Take regex `cat` against text `the cat sat`:

1. Start at position 0 (`t`). Compare to `c`. No match. Move on.
2. Start at position 1 (`h`). Compare to `c`. No match. Move on.
3. Start at position 2 (`e`). Compare to `c`. No match. Move on.
4. Start at position 3 (` `). Compare to `c`. No match. Move on.
5. Start at position 4 (`c`). Compare to `c`. Match. Move both forward.
6. Position 5 (`a`). Compare to `a`. Match. Move both forward.
7. Position 6 (`t`). Compare to `t`. Match. Move both forward. Regex done. Caught a fish: `cat`, at positions 4-6.

Then it can keep going from position 7 onward to see if there are more matches.

That is "matching." Walking the text, walking the regex, character by character, until you either fail (and try starting from the next character) or succeed (and either stop or continue looking for more matches).

When the regex has flexible parts like `*` or `+`, the engine has choices to make: how many characters should the `+` swallow? Most engines try to be **greedy** — they swallow as many characters as they can — and then back off if the rest of the regex doesn't fit. We will come back to this when we talk about **catastrophic backtracking** later, because that backing-off is where all the danger lives.

### Why this all looks like ASCII soup at first

Let's be honest: regexes are ugly. They look like keyboard noise:

```
^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$
```

That is an IPv4 address regex. It is not pretty. But every symbol is doing real work. Let's read it:

- `^` — start of line
- `\d{1,3}` — between one and three digits
- `\.` — a literal dot
- `\d{1,3}` — between one and three digits
- `\.` — a literal dot
- `\d{1,3}` — between one and three digits
- `\.` — a literal dot
- `\d{1,3}` — between one and three digits
- `$` — end of line

So the whole thing says: "the entire line is four chunks of one-to-three digits separated by dots." That matches `192.168.1.1` and `8.8.8.8` and (alas) `999.999.999.999` (which is technically nonsense but matches the shape). Tightening the digit ranges so we only accept 0-255 is left as an exercise — it is much uglier still, and it is a fine example of why nobody writes IPv4 regexes by hand if they can avoid it.

The point is: each character is a tool. Once you learn the tools, the soup turns into sentences.

### Two big families of regex engines

Not every regex tool works the same way. There are two big families.

**The backtracking family** is what most languages use: Perl, Python's `re`, JavaScript, Java, .NET, PCRE (the engine in PHP, nginx, Apache, and a million other places), Ruby. These engines start matching, hit a flexible spot, try one option, and if the rest of the regex fails, **back up** and try another option. Backtracking engines are flexible — they support every regex feature you can dream up, including backreferences, lookahead, lookbehind, recursion. But they have a dangerous secret: certain regexes can cause them to back up so many times they take **forever** to give up. That is **catastrophic backtracking**, also called **ReDoS** when an attacker uses it to crash your server. We will see it later.

**The automaton family** is what Go's `regexp`, Rust's `regex`, RE2, and grep's default engine use. These engines build a thing called an **NFA** or **DFA** (a kind of state machine) and walk the text once, no backing up. They have **predictable, linear-time performance** — they cannot take forever, no matter what regex or input you throw at them. But they don't support every feature: in particular, **no backreferences** and (in classic RE2) no lookahead or lookbehind. The trade-off is speed and safety in exchange for a smaller feature set.

Knowing which family your tool is in matters a lot. If you write a regex with `(.+)+` and feed it to a Go program, it just runs in linear time and doesn't care. If you write the same regex in Java and feed it user-controlled input, an attacker can hang your server with a single string. Same regex, different planet.

### How a regex actually runs (the NFA picture)

Let's draw a simple regex as a state machine. Take the regex `ab*c` — meaning "a, then zero or more b's, then c."

```
        b (loop)
        |
   +----v----+
   |         |
[start] --a--> (S1) --c--> [accept]
```

A cleaner ASCII version:

```
                +-----+
                |  b  |   (self-loop: any number of b's)
                v     |
   start --a--> S1 ---+--c--> S2 (accept)
```

The little circle labelled `S1` has a loop on it labelled `b`. That means once you arrive at S1, you can take the `b` arrow back to yourself zero or more times. Then you take the `c` arrow to the accept state and you are done. Trace the input `ac` through this:

- Start. Read `a`. Take the `a` arrow. Now at S1.
- Read `c`. Take the `c` arrow. Now at S2 (accept). Done. **Match.**

Trace the input `abbbc`:

- Start. Read `a`. Take the `a` arrow. Now at S1.
- Read `b`. Take the `b` self-loop. Still at S1.
- Read `b`. Take the `b` self-loop. Still at S1.
- Read `b`. Take the `b` self-loop. Still at S1.
- Read `c`. Take the `c` arrow. Now at S2 (accept). Done. **Match.**

Trace `axc`:

- Start. Read `a`. Take the `a` arrow. Now at S1.
- Read `x`. No matching arrow from S1 (no `x`, no `c` until we read `c`, no `b`). **Fail.**

That is the NFA picture. Each state is a place in your regex, each arrow is a character to read, and you walk along reading input. RE2-style engines literally build this graph and walk it once per input character — that is why they cannot blow up.

### The DFA picture (same regex, different drawing)

A **DFA** (deterministic finite automaton) is the same idea but with the rule that from any state, for any input character, there is exactly one place you could go. No "maybe try this, maybe try that" choice.

For `ab*c`, the DFA looks identical to the NFA above because there are no real choices — you read `a`, you have to be at S1; you read `b`, you stay at S1; you read `c`, you go to accept. This regex is already deterministic.

For something like `a(b|bb)c` (literally "a, then b or bb, then c"), the NFA has a choice — at the moment you see a `b`, do you take the `b` arrow or the start of the `bb` arrow? — but the DFA equivalent rolls those choices into one state per "set of NFA states you could be in," so the DFA walks deterministically.

The Russ Cox articles at <https://swtch.com/~rsc/regexp/> are the canonical explanation of all this and absolutely worth reading once you are comfortable with the basics.

### What is "anchoring" really doing?

When we say a regex is "anchored," we mean it has a `^` or `$` (or `\A`, `\z`, `\Z` in some flavors) that pins it to a specific place in the input. An **unanchored** regex floats — the engine will slide it across the text trying every starting position. An anchored regex says "I must start here" or "I must end here."

There are also functions that effectively anchor for you. In Python:

- `re.match(p, s)` anchors `p` at the start of `s` (but not the end).
- `re.fullmatch(p, s)` anchors at both ends — the entire string must match.
- `re.search(p, s)` does NOT anchor — it slides looking for any match.

People mix these up constantly. If your regex matches "more than expected," check whether you wanted `match` vs `search` vs `fullmatch`. Often the fix is just switching functions, not fixing the pattern.

## Literal Characters

The simplest regex is just letters and digits and most punctuation that you want to match exactly. `dog` matches `dog`. `cat42` matches `cat42`. The regex engine doesn't do anything fancy: each character in the regex represents itself.

The exceptions are the **metacharacters** — characters with special meaning. They are:

```
.  \  *  +  ?  ^  $  |  (  )  [  ]  {  }
```

If you want any of these to match literally, you have to **escape** them with a backslash:

- `\.` matches a literal dot.
- `\\` matches a literal backslash.
- `\$` matches a literal dollar sign.
- `\(` matches a literal open paren.
- `\?` matches a literal question mark.

In some flavors, `{` and `}` only need escaping when they look like quantifiers. In some flavors, `(` and `)` are literal unless they are backslashed (BRE — basic regex — does this and confuses everyone). When in doubt, escape it; an unnecessary escape is harmless in most flavors.

## Character Classes

A **character class** is a set of allowed characters in one position. Square brackets `[...]` define one. Inside a character class, most metacharacters lose their special meaning — you don't need to escape `.` inside `[]`, for example.

### `[abc]` — match one of these

`[abc]` matches one character that is `a` or `b` or `c`. Three options, one character.

`[abc]at` matches `aat`, `bat`, or `cat`.

### `[a-z]` — match a range

`[a-z]` matches one lowercase letter. `[A-Z]` matches one uppercase. `[0-9]` matches one digit. You can combine: `[A-Za-z0-9]` matches one alphanumeric character.

Multiple ranges are fine: `[a-zA-Z0-9_]` matches a single "word" character (letters, digits, or underscore).

### `[^abc]` — negation

A `^` as the first character inside `[]` means "not these." `[^abc]` matches any one character that is NOT `a`, `b`, or `c`. Note: outside of `[]`, `^` means "start of line." Inside `[]` as the first character, it means negation. Two completely different jobs.

### `\d`, `\w`, `\s` — short forms

Some character classes are so common they have short forms (these are PCRE/Perl conventions; flavors vary):

- `\d` — a digit. Usually means `[0-9]`. In Unicode mode in some flavors, also matches Arabic-Indic digits and other Unicode digit characters.
- `\w` — a word character. Usually `[A-Za-z0-9_]`. In Unicode mode, includes any Unicode letter, digit, or underscore.
- `\s` — whitespace. Space, tab, newline, carriage return, form feed.

### `\D`, `\W`, `\S` — negated short forms

- `\D` — anything NOT a digit. Same as `[^0-9]`.
- `\W` — anything NOT a word character. Same as `[^A-Za-z0-9_]`.
- `\S` — anything NOT whitespace.

### POSIX bracket expressions

POSIX defines named character classes that work inside `[]`:

- `[[:alpha:]]` — letters
- `[[:digit:]]` — digits
- `[[:alnum:]]` — letters or digits
- `[[:space:]]` — whitespace
- `[[:upper:]]` — uppercase letters
- `[[:lower:]]` — lowercase letters
- `[[:xdigit:]]` — hex digits, `[0-9A-Fa-f]`
- `[[:punct:]]` — punctuation
- `[[:print:]]` — printable characters (incl. space)
- `[[:graph:]]` — printable characters (excl. space)
- `[[:cntrl:]]` — control characters
- `[[:blank:]]` — space and tab

Note the **double brackets**: `[[:digit:]]` means "a single character class made up of POSIX digit." You combine them like any other: `[[:alpha:][:digit:]]` is letters or digits. POSIX classes are nice because they respect the locale, so `[[:alpha:]]` in a German locale includes umlauts.

## Anchors

Anchors don't consume characters. They are zero-width assertions about position. They either succeed or fail at a given spot, but they don't advance through the input.

### `^` — start of line

`^cat` matches `cat` only when it appears at the start of a line. In multiline mode, "start of line" means after every newline, too. In single-line mode (default in many tools), it means start of the entire string.

### `$` — end of line

`done$` matches `done` only at the end of a line.

### `\A` and `\z` — start/end of whole string

In some flavors, `\A` is "start of string, no matter what mode you are in," and `\z` is "very end of string." `\Z` (capital Z) often means "end of string, but allow a trailing newline." These distinctions matter when you have text with multiple lines and you want to be precise about whether you mean line boundaries or string boundaries.

### `\b` — word boundary

`\b` matches at the boundary between a word character and a non-word character. So `\bcat\b` matches the standalone word `cat` but not the `cat` inside `category` or `bobcat`. The boundary itself is zero-width — it doesn't eat a character, it just asserts "you are at the edge of a word here."

Subtle gotcha: word-boundary semantics depend on what counts as a "word character," which differs by flavor and by Unicode mode. ASCII `\b` treats only `[A-Za-z0-9_]` as word characters; Unicode `\b` treats all Unicode letters and digits as word characters. This matters for non-English text.

### `\B` — NOT a word boundary

`\B` is the opposite. It matches at any position that is NOT a word boundary. So `\Bcat\B` matches `cat` only if it is in the middle of a longer word — like the `cat` in `vacation`.

## Quantifiers (greedy)

Quantifiers say "do the previous thing N times." By default in most flavors, quantifiers are **greedy** — they grab as much as they can, then back off if needed.

- `*` — zero or more.
- `+` — one or more.
- `?` — zero or one.
- `{n}` — exactly n.
- `{n,}` — at least n.
- `{n,m}` — between n and m.

Greedy in action: `<.+>` against `<a><b>` does **not** match just `<a>` — it greedily grabs `a><b` (everything between the first `<` and the last `>`). Greedy quantifiers want to swallow the world and only spit characters back when forced.

## Quantifiers (lazy)

Stick a `?` after a quantifier to make it **lazy** — match as few as possible, only growing when needed.

- `*?` — zero or more, lazy.
- `+?` — one or more, lazy.
- `??` — zero or one, lazy (yes, weird-looking).
- `{n,}?`, `{n,m}?` — same idea.

Lazy in action: `<.+?>` against `<a><b>` matches `<a>` first, then if you keep searching, matches `<b>` second. Lazy quantifiers try the smallest match first and only grow if that doesn't work.

People sometimes call this "greedy vs reluctant" or "greedy vs minimal." Same idea, different names.

## Grouping

Parentheses do two jobs in regex: they **group** stuff together and they **capture** the matched text.

### Capturing groups: `( ... )`

A plain `( ... )` is a capturing group. It groups for quantifiers (so `(ab)+` means "one or more `ab`s," matching `ab`, `abab`, `ababab`) AND it remembers whatever was matched inside, available later as `\1`, `\2`, etc., or as `$1`, `$2` in replacements.

Groups are numbered left to right by their opening paren. So in `(\d+)-(\w+)`, group 1 is the digits, group 2 is the word.

### Non-capturing groups: `(?: ... )`

If you only need grouping, not capturing, use `(?:...)`. It groups for quantifiers but doesn't remember anything. So `(?:ab)+` matches the same things as `(ab)+`, but doesn't reserve a backreference number.

Why bother? Two reasons. **Performance** — capturing has a small cost. **Numbering** — if you mix capturing and non-capturing, only the capturing ones take group numbers, which keeps the numbering clean.

## Backreferences

A **backreference** is a way to say "match the same text that group 1 matched." You write `\1` for group 1, `\2` for group 2, and so on (in PCRE/Perl/Python). In replacements (when you are doing a substitution, not just matching), the syntax is usually `$1`, `$2`, or sometimes `\1`, `\2`, depending on the tool.

Example: `(\w+) \1` matches any word followed by a space and the same word again. So it matches `the the` or `cat cat` but not `the dog`. The `\1` says "match exactly what group 1 matched here."

This is **not supported in RE2** (Go and Rust regex). RE2 makes a deliberate trade: it gives up backreferences in exchange for guaranteed linear-time matching. If you need backreferences in Go, use a third-party regex library or rethink the problem.

In substitutions, `$1` means "insert the text group 1 matched." So in Python:

```python
re.sub(r'(\w+) \1', r'\1', 'the the cat cat dog')  # -> 'the cat dog'
```

We collapsed every "word word" into a single word.

## Alternation

The pipe `|` means "or." `cat|dog` matches `cat` or `dog`. Use parentheses to scope alternation: `(cat|dog)s` matches `cats` or `dogs`, not `cat` or `dogs`. Without the parens, `cat|dogs` is "cat OR dogs," not what you wanted.

Alternation tries each option left to right and (in backtracking engines) takes the first one that works (Perl-style **leftmost-first**, not POSIX **longest-match**). This matters: `cat|category` against `category` matches `cat`, not `category`, in Perl-style engines. POSIX-style engines pick the longest match, returning `category`. Most modern languages use Perl-style.

## Escaping

To match a metacharacter literally, escape it with `\`.

- `\.` for a literal dot.
- `\\` for a literal backslash.
- `\$` for a literal dollar sign.
- `\^` for a literal caret.
- `\|` for a literal pipe.
- `\*`, `\+`, `\?` for literal quantifiers.
- `\(`, `\)`, `\[`, `\]`, `\{`, `\}` for literal brackets and braces.

Inside `[]`, most metacharacters are already literal — `[.+]` matches a literal dot or a literal plus. The metacharacters that still need escaping inside `[]` are usually `\`, `]` (which would close the class), `^` (which would mean negation if it's first), and `-` (which would mean range if it's between characters).

In your **source language**, you also have to deal with the language's own escape rules. In Python, write your regex in a raw string `r'...'` so backslashes survive: `r'\d+'` is the regex `\d+`, but `'\d+'` is a Python string containing the regex `\d+` (Python warns about that today but used to silently let it through). In JavaScript, regex literals `/\d+/` don't have this issue, but if you write `new RegExp('\\d+')` you have to double the backslash. Similar story in Java strings.

## Common Patterns

A grab bag of useful nets to copy and tweak. Each one is a **starting point**, not gospel — real-world data is messy and these will need tightening for your specific case.

### Email (basic)

```
[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}
```

This catches the common shape: name, @, domain, dot, TLD of two or more letters. It does NOT validate every legal email per RFC 5321/5322 (that regex is famously about a hundred lines long and almost nobody actually needs that level of strictness). Use this for "find email-shaped strings in text," not for "decide if this is a real RFC-valid email."

### URL (basic)

```
https?://[A-Za-z0-9.-]+(?:/[^\s]*)?
```

`https?://` matches `http://` or `https://` (the `?` makes the `s` optional). Then a domain. Then optionally a slash and any non-whitespace characters.

### IPv4

```
\b(?:\d{1,3}\.){3}\d{1,3}\b
```

Three "digits-dot" groups, then digits. Word boundaries on each end. This matches anything-shaped-like-an-IP, including invalid `999.999.999.999`. Tightening to legal 0-255 octets is uglier:

```
\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b
```

### MAC address

```
\b[0-9A-Fa-f]{2}(?:[:-][0-9A-Fa-f]{2}){5}\b
```

Six pairs of hex digits separated by colons or dashes.

### ISO-8601 date

```
\b\d{4}-\d{2}-\d{2}\b
```

Four digits, dash, two digits, dash, two digits. Doesn't validate that the date is real — `9999-99-99` matches the shape.

### UUID

```
\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b
```

Hex digits in 8-4-4-4-12 chunks separated by dashes.

### E.164 phone number

```
\+[1-9]\d{1,14}
```

E.164 says: leading plus, then a country code (no leading zero), total max 15 digits.

## Regex Flavors

Not every regex tool supports the same syntax. Here is the rough lay of the land.

### POSIX BRE (Basic Regular Expression)

Old. Used by classic `grep` (without `-E`), `sed` (without `-E`), `vi` in default mode. Quirks:

- `(`, `)`, `{`, `}` are literal unless escaped: you write `\(group\)` and `\{1,3\}`. Annoying.
- `+` and `?` may not be quantifiers — depends on the implementation.
- No `|` alternation in pure BRE.
- Anchors and `*` and bracket expressions all work normally.

### POSIX ERE (Extended Regular Expression)

Modernised POSIX. Used by `grep -E`, `egrep`, `sed -E`, `awk`. Closer to what most people expect:

- `(`, `)`, `{`, `}`, `+`, `?`, `|` all work without escaping.
- No backreferences.
- No lookahead/lookbehind.
- POSIX longest-match semantics (different from Perl's leftmost-first).

### PCRE / PCRE2

Perl-Compatible Regular Expressions. The library used by PHP (`preg_*`), nginx, Apache, many editors, and a thousand other tools. Effectively the most common "rich" regex flavor. Has:

- All the standard character classes, quantifiers, anchors.
- Backreferences (`\1`, `(?P=name)`).
- Lookahead and lookbehind.
- Named groups (`(?P<name>...)` Python style or `(?<name>...)` Perl/PCRE style).
- Conditionals `(?(1)yes|no)`.
- Recursion `(?R)` and `(?1)`.
- Atomic groups `(?>...)`.
- Possessive quantifiers `*+`, `++`, `?+`.
- Inline modifiers `(?i)`, `(?m)`, etc.
- Comments `(?#...)`.
- Subroutine calls.

Backtracking engine. Powerful but ReDoS-prone if you're not careful.

### Perl regex

The original. PCRE chases this. Slightly more features than PCRE in some areas, fewer in others. Same basic vibe.

### Python `re`

Nearly PCRE-equivalent for everyday use. Some specifics:

- Named groups: `(?P<name>...)`, backreference `(?P=name)`, replacement `\g<name>`.
- `re.compile`, `re.search`, `re.match`, `re.fullmatch`, `re.findall`, `re.finditer`, `re.sub`, `re.subn`, `re.split`.
- Flags: `re.I` (ignore case), `re.M` (multiline), `re.S` (dotall — `.` matches newline), `re.X` (verbose, allows whitespace and comments in patterns), `re.U` (Unicode, default in 3.x).
- Backtracking engine.
- Variable-length lookbehind only since 3.7 — older Python required fixed-width lookbehinds.

There is also a third-party `regex` library on PyPI with more features (Unicode property classes, fuzzy matching, recursion).

### JavaScript `RegExp`

Built into the language. Quirks:

- Regex literals `/pattern/flags` and the constructor `new RegExp("pattern", "flags")`.
- Named groups `(?<name>...)`, backreference `\k<name>`, replacement `$<name>`.
- Flags: `i` (case insensitive), `g` (global — find all), `m` (multiline), `s` (dotall, ES2018+), `u` (Unicode, ES2015+), `y` (sticky), `d` (indices, ES2022+).
- Lookbehind: ES2018+, full PCRE-like.
- No POSIX classes, no inline modifiers (until very recently).
- `String.prototype.match`, `matchAll`, `replace`, `replaceAll`, `search`, `split`. `RegExp.prototype.test`, `exec`.
- Backtracking engine. ReDoS-prone.

### Java `java.util.regex`

`Pattern.compile(...)`, `Matcher.find()`, `Matcher.matches()`, `Matcher.group(0)`, `Matcher.group(name)`. Verbose. Most regex features. Slightly different syntax in places (named groups are `(?<name>...)`). Backtracking engine. PatternSyntaxException for compile errors.

### Go `regexp` (RE2)

Linear-time RE2 implementation. **No backreferences. No lookahead/lookbehind.** Trade-off: cannot blow up on bad input. `regexp.MustCompile("...")`, `re.FindString`, `re.FindAllString`, `re.FindStringSubmatch`, `re.ReplaceAllString`, `re.ReplaceAllStringFunc`. Named groups `(?P<name>...)` (Python-style).

### Rust `regex` crate

RE2-style, same trade-offs as Go. `Regex::new("...")`, `regex.find`, `regex.captures`, `regex.replace`, `regex.replace_all`. There is also a separate `fancy-regex` crate that adds backreferences and lookaround if you really need them — at the cost of linear-time guarantees.

### .NET `System.Text.RegularExpressions.Regex`

Backtracking engine, full PCRE-equivalent feature set, plus some unique extras like balancing groups (which can match nested structures, sort of). Compiled regexes are an option for performance.

### ICU regex

Unicode-aware, used by JavaScript engines, ICU consumers, etc.

### Boost.Regex / `std::regex`

C++ standard library has `std::regex` (variable performance, often slow). Boost has a richer implementation. Both are backtracking.

### Hyperscan

Intel's library for matching tens of thousands of regexes at line speed. Used in IDS systems, log scanning, etc. Not a regex engine you reach for casually — but worth knowing exists for high-performance batch matching.

### sed regex

By default, `sed` uses BRE. With `-E` (or `-r` on some systems), it uses ERE. `sed` is famously the worst tool to learn regex on because of BRE escaping rules.

### awk regex

ERE-ish. POSIX-style. Slightly different from `grep -E` in places.

### vim regex

Three modes: **nomagic** (`\v`-prefix-style escapes everywhere), **magic** (default — sort of BRE-like), and **very-magic** (`\v` mode, where everything is metacharacter unless escaped, much more PCRE-like). The mode you are in changes which characters need escaping. Most people set **very-magic** with `\v` at the start of every search to make vim behave like a normal regex engine.

## The Catastrophic Backtracking Problem

Here is the dragon.

Backtracking engines work by trying one path, and if it fails, trying another. For most regexes against most inputs, this is fine — there are not that many paths to try. But certain regex patterns combined with certain inputs cause the number of paths to explode to **exponential** in the size of the input. A regex that should match in a microsecond can take minutes, hours, years.

The classic example: `^(a+)+b$` against `aaaaaaaaaaX`.

Let's trace what happens. The regex says "start, one-or-more `a`s, repeated one-or-more times, then `b`, then end."

The engine walks through the input. The outer `+` and the inner `+` both want to swallow `a`s, and there are MANY ways to split the run of `a`s between them. For 10 `a`s, there are dozens of ways to split. For 20 `a`s, thousands. For 30 `a`s, millions. The engine tries every single way before it gives up because of the trailing `X` (which can never match `b$`).

```
Input: aaaaaaaaaaX
Regex: ^(a+)+b$

Try 1: outer takes (aaaaaaaaaa), inner takes 10 → no b at end → fail
Try 2: outer takes (aaaaaaaaa), then (a) → no b → fail
Try 3: outer takes (aaaaaaaa), then (aa) → no b → fail
Try 4: outer takes (aaaaaaaa), then (a), then (a) → no b → fail
Try 5: outer takes (aaaaaaa), then (aaa) → no b → fail
Try 6: outer takes (aaaaaaa), then (aa), then (a) → no b → fail
Try 7: outer takes (aaaaaaa), then (a), then (aa) → no b → fail
Try 8: outer takes (aaaaaaa), then (a), then (a), then (a) → no b → fail
... and so on, doubling each time we add an a.
```

Every additional `a` roughly doubles the work. 10 a's: a thousand tries. 30 a's: a billion. 50 a's: a quadrillion. The regex never fails fast — it tries every possible split before admitting defeat.

This is **catastrophic backtracking**. When an attacker sends user-controlled input designed to trigger this, it is **ReDoS** — Regular Expression Denial of Service. A single HTTP request with a crafted string can pin a server CPU for minutes.

### The shapes that go bad

The bad shapes share a common feature: **nested quantifiers where the inner thing can match the same character multiple ways.**

- `(a+)+`
- `(a*)*`
- `(a|aa)+`
- `(.*)*`
- `(.+)+`
- `(\w+)+`

Each one has a quantifier inside a quantifier with overlapping alternatives. The engine has many redundant ways to match the same input.

### How to defuse it

Several options:

1. **Use an RE2-style engine.** Go's regexp, Rust's regex, RE2 directly, ripgrep. These cannot blow up because they use NFA simulation, not backtracking. Linear time, always.

2. **Use atomic groups** (in PCRE, Perl, Java, .NET). An atomic group `(?>...)` says "once you match, don't back up into me." `(?>a+)+b` with input `aaaaaaaaaaX` fails fast because once the atomic group eats the a's, it can't give them back. Linear time.

3. **Use possessive quantifiers** (in PCRE, Perl, Java). `a++` is like `a+` but doesn't give back what it matched. `(a++)+b` is fast where `(a+)+b` is catastrophic.

4. **Rewrite the regex to be unambiguous.** `(a+)+b` can be written as `a+b` — same matches, no nested quantifier.

5. **Anchor the pattern** so the engine doesn't try every starting position: `^...$` or use `re.fullmatch`.

6. **Cap input length** before running the regex on user data.

### A real-world ReDoS

In 2019, Cloudflare had an outage caused by a regex `.*(?:.*=.*)`. It took down the whole network for 27 minutes. The pattern had a nested-quantifier shape. A specific user-controlled input triggered exponential backtracking. The CPU pinned, the WAF stalled, the network melted. The fix involved switching to a non-backtracking engine and rewriting the rule.

Lesson: if your regex runs over user-controlled input AND you are using a backtracking engine, every regex is a potential DoS bomb. Use RE2-class engines or use atomic groups religiously.

## Lookahead and Lookbehind

Sometimes you need to match something only if (or only if not) something else is around it, but you don't want to actually consume the surrounding characters. That is **lookaround** — zero-width assertions.

### Lookahead `(?= ...)` — positive

`foo(?=bar)` matches `foo` only if `bar` immediately follows. The `bar` is not part of the match — it is just a peek. So `foo(?=bar)` against `foobar` matches `foo` (length 3), not `foobar`.

### Negative lookahead `(?! ...)`

`foo(?!bar)` matches `foo` only if `bar` does NOT follow. So `foo(?!bar)` against `foobaz` matches `foo`, but against `foobar` it fails.

### Lookbehind `(?<= ...)` — positive

`(?<=foo)bar` matches `bar` only if `foo` came right before. The `foo` is not part of the match. So `(?<=foo)bar` against `foobar` matches `bar`.

### Negative lookbehind `(?<! ...)`

`(?<!foo)bar` matches `bar` only if `foo` did NOT come right before.

### Fixed-width vs variable-length lookbehind

In Python's `re`, lookbehind used to require **fixed width** — you couldn't say `(?<=\w+)` because the engine couldn't know how far back to look. Python 3.7+ supports variable-width lookbehind. PCRE traditionally only supports fixed-width. Java requires fixed-width-or-bounded. .NET supports variable-width. JavaScript (ES2018+) supports variable-width. Go's RE2 does not support lookbehind at all.

Lookaround is genuinely useful but can be slow in backtracking engines. Use sparingly.

## Named Groups

Numbered groups are fine for one or two captures, but past that, you stop being able to remember which is which. Named groups give them labels.

### Python `(?P<name>...)`

```python
m = re.search(r'(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})', '2026-04-27')
m.group('year')   # '2026'
m.group('month')  # '04'
m.group('day')    # '27'
```

Backreference: `(?P=year)`. Replacement: `\g<year>` or `\1` if numbered.

### PCRE `(?<name>...)` or `(?P<name>...)`

PCRE accepts both. Backreference: `\k<name>` or `(?P=name)`. Replacement varies by tool.

### JavaScript `(?<name>...)`

```javascript
'2026-04-27'.match(/(?<year>\d{4})-(?<month>\d{2})-(?<day>\d{2})/).groups
// { year: '2026', month: '04', day: '27' }
```

Backreference: `\k<name>`. Replacement: `$<name>`.

### Go `(?P<name>...)`

Python-style only. Access via `re.SubexpNames()` and the group index.

## Unicode Mode

By default, `\d`, `\w`, `\s`, and `[a-z]` use ASCII semantics. In Unicode mode (which is on by default in many modern languages), they expand:

- `\d` matches any character classified as a digit by Unicode — including Arabic-Indic digits `٠١٢٣٤٥٦٧٨٩`, Devanagari `०१२३४५६७८९`, fullwidth digits `０１２３４５６７８９`, and many more.
- `\w` matches any Unicode letter or digit or underscore.
- `\s` includes Unicode whitespace beyond ASCII.

You can also use **Unicode property escapes** `\p{...}`:

- `\p{L}` — any letter (any script).
- `\p{N}` — any number.
- `\p{P}` — any punctuation.
- `\p{Z}` — any separator.
- `\p{Lu}` — uppercase letter.
- `\p{Ll}` — lowercase letter.
- `\p{IsAlpha}`, `\p{Alpha}` — alphabetic.
- `\p{Script=Greek}` — letters in the Greek script.
- `\p{Script=Cyrillic}` — Cyrillic.
- `\p{Emoji}` — emoji codepoints.

Negated: `\P{L}` — anything NOT a letter.

These are godsend for non-English text. ASCII `[a-z]` does not match `é` or `ß` or `日`. `\p{L}` does.

In Python you need the third-party `regex` library (not stdlib `re`) for full `\p{...}` support. In JavaScript with the `u` flag, you get `\p{...}` natively. PCRE2 supports it. Go's RE2 supports a subset.

## Multiline ^ $ vs whole-string

By default in many flavors, `^` and `$` match start/end of the whole string. With the **multiline flag** (`re.M` / `m` / `(?m)`), they match start/end of each line.

```python
import re
text = 'first\nsecond\nthird'
re.findall(r'^\w+', text)            # ['first']
re.findall(r'^\w+', text, re.M)      # ['first', 'second', 'third']
```

This trips up everyone at least once.

There is also the **dotall flag** (`re.S` / `s` / `(?s)`). By default, `.` does NOT match newlines. With dotall, it does. So `<.*>` in default mode does not span multiple lines, but in dotall mode it can.

Anchor variants:

- `^` — start of line (in multiline mode) or start of string (otherwise).
- `$` — end of line (in multiline mode) or end of string (otherwise — usually right before a final newline).
- `\A` — start of string, always.
- `\z` — end of string, always.
- `\Z` — end of string, possibly before a final newline.

If you want "start of string and end of string, no matter the mode," use `\A` and `\z`.

## Common Errors (verbatim)

Real error messages you will see, with the canonical fix:

### Python

- **`re.error: bad escape \q at position 0`** — you wrote `\q` and there is no such escape. Fix: drop the backslash, or use a real escape.
- **`re.error: nothing to repeat at position 0`** — you wrote `*abc` or `+abc`, putting a quantifier where there is nothing to repeat. Fix: put the quantifier after a real expression.
- **`re.error: missing ), unterminated subpattern at position 0`** — opened a `(` and never closed it.
- **`re.error: unbalanced parenthesis at position 5`** — `)` without a matching `(`.
- **`re.error: multiple repeat at position 2`** — `a++` in stdlib `re` doesn't mean possessive (that's PCRE), it means "repeat a repeat" which is invalid.
- **`re.error: bad character range a-A at position 1`** — backwards range.

### POSIX ERE

- **`grep: Invalid range end`** — `[z-a]` instead of `[a-z]`.
- **`grep: Trailing backslash`** — your pattern ends with `\` and there is nothing to escape.
- **`grep: Unmatched ( or \(`**

### PCRE / PHP

- **`preg_match(): Compilation failed: missing closing parenthesis at offset 5`** — unbalanced `(`.
- **`preg_match(): Compilation failed: nothing to repeat at offset 0`** — same as Python.
- **`preg_match(): Subject not valid UTF-8`** — your input is not valid UTF-8 and you're using `u` flag. Fix: validate or strip.
- **`preg_match(): regular expression is too large`** — pattern exceeds engine limits, often after compilation expansion.

### Java

- **`java.util.regex.PatternSyntaxException: Unclosed group near index 5`** — unbalanced `(`.
- **`java.util.regex.PatternSyntaxException: Dangling meta character '*'`** — same as Python's "nothing to repeat."
- **`java.util.regex.PatternSyntaxException: Illegal repetition`** — usually `{` without a number or `}`.

### sed / awk / grep generic

- **`sed: -e expression #1, char N: unknown command: \``\`** — usually wrong delimiter or unescaped slash inside a `s/.../.../` command.
- **`sed: -e expression #1, char N: extra characters after command`** — same general family.
- **`awk: bailing out at source line 1`** — usually a runaway regex with mismatched braces or quotes.
- **`grep: Trailing backslash`** — pattern ends with `\`.
- **`grep: Unmatched [`** — bracket class not closed.

### Generic

- **`no match for ...`** — your pattern compiled fine but found nothing in the input. Not a regex error per se, just "found nothing." Common confusions: forgot to anchor or accidentally anchored, forgot multiline flag, forgot to escape a metacharacter so it matched too much or too little.

### ReDoS / catastrophic backtracking

Not always a clean error message — usually the symptom is a regex that **never returns**. The CPU pegs at 100%, the function hangs. In Java, you might see a stack overflow or a thread dump showing the matcher in `match()` for many seconds. In Node.js, your event loop blocks. In Python, `re.search` just sits there. Profile and you'll see it spinning in NFA simulation. Fix: rewrite the pattern, switch engines, add timeouts, validate input.

## Hands-On

Type these into your terminal. The `$` at the start of each line means "shell prompt — don't type the dollar sign." Output is shown on the next line(s) without a `$`.

### Match a literal word with grep

```
$ echo "the cat sat on the mat" | grep -E 'cat'
the cat sat on the mat
```

`grep` printed the whole line because it contained `cat`. `-E` enables ERE (extended regex), which is what most modern users expect. Without `-E`, `grep` uses BRE.

### Find any 4-digit number

```
$ echo "order 1234 ships on day 5 with code 9876" | grep -oE '\d{4}'
```

Wait — `grep -E` doesn't always understand `\d`. POSIX ERE has no `\d`. Use:

```
$ echo "order 1234 ships on day 5 with code 9876" | grep -oE '[0-9]{4}'
1234
9876
```

`-o` means "only print the matching part, not the whole line." `-E` enables ERE.

### Use ripgrep (which speaks PCRE-ish via `-P` and \d natively in default)

```
$ echo "order 1234 ships on day 5 with code 9876" | rg '\d{4}'
order 1234 ships on day 5 with code 9876
```

ripgrep uses Rust's regex engine — RE2-style, linear time, and it understands `\d` out of the box.

### Print only the matches

```
$ echo "order 1234 ships on day 5 with code 9876" | rg -o '\d+'
1234
5
9876
```

### Match emails in a file with ripgrep

```
$ rg -o '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' contacts.txt
bob@example.com
stevie@bellis.tech
admin@unheaded.dev
```

### Use sed to replace text

```
$ echo "hello world" | sed -E 's/world/regex/'
hello regex
```

`s/pattern/replacement/` is sed's substitute command. `-E` enables ERE.

### Replace with a backreference

```
$ echo "Bellis, Stevie" | sed -E 's/(\w+), (\w+)/\2 \1/'
Stevie Bellis
```

We captured two groups and swapped them.

### Replace globally with `g` flag

```
$ echo "one one one" | sed -E 's/one/two/g'
two two two
```

Without `g`, sed only replaces the first match per line.

### awk pattern matching

```
$ echo -e "apple\nbanana\ncherry" | awk '/^c/'
cherry
```

awk with `/regex/` filters lines that match. Here, lines starting with `c`.

### awk extracting fields

```
$ echo "name: stevie age: 42" | awk -F'[ :]+' '{print $2, $4}'
stevie 42
```

`-F'[ :]+'` sets the field separator to "one or more spaces or colons."

### Perl one-liner

```
$ echo "the year 2026 is here" | perl -ne 'print $1 if /(\d{4})/'
2026
```

`perl -ne` runs the script for each line, `$1` is the first capture group.

### Perl substitute

```
$ echo "hello world" | perl -pe 's/world/regex/'
hello regex
```

`-pe` runs the script and prints each line. Same syntax as sed substitute, but Perl's regex engine has full PCRE features.

### Python one-liner

```
$ echo "the year 2026 is here" | python3 -c 'import re,sys; print(re.search(r"\d{4}", sys.stdin.read()).group())'
2026
```

### Python `findall`

```
$ python3 -c 'import re; print(re.findall(r"\d+", "a1 b22 c333"))'
['1', '22', '333']
```

### Python `sub`

```
$ python3 -c 'import re; print(re.sub(r"\d+", "N", "a1 b22 c333"))'
aN bN cN
```

### Python named groups

```
$ python3 -c 'import re; m=re.search(r"(?P<year>\d{4})-(?P<month>\d{2})", "2026-04"); print(m.group("year"), m.group("month"))'
2026 04
```

### Python `re.MULTILINE`

```
$ printf 'first\nsecond\nthird' | python3 -c 'import re,sys; print(re.findall(r"^\w+", sys.stdin.read(), re.M))'
['first', 'second', 'third']
```

### Python `re.fullmatch` vs `re.search`

```
$ python3 -c 'import re; print(bool(re.search(r"\d+", "abc123def")))'
True

$ python3 -c 'import re; print(bool(re.fullmatch(r"\d+", "abc123def")))'
False

$ python3 -c 'import re; print(bool(re.fullmatch(r"\d+", "123")))'
True
```

### Lookahead in Python

```
$ python3 -c 'import re; print(re.findall(r"\d+(?=px)", "10px 20em 30px"))'
['10', '30']
```

We grabbed numbers that are followed by `px`, but we did not include `px` in the match.

### Negative lookahead

```
$ python3 -c 'import re; print(re.findall(r"\d+(?!px)", "10px 20em 30px"))'
['1', '20', '3']
```

Numbers not followed by `px`. Note this is character-by-character — `10` becomes `1` because the `0` is followed by `p`, so the lookahead fails at `10` but succeeds at `1`.

### Lookbehind in Python

```
$ python3 -c 'import re; print(re.findall(r"(?<=\$)\d+", "price $10 vs €20 vs $30"))'
['10', '30']
```

Numbers preceded by a literal `$`.

### Use `jq` to filter JSON by regex

```
$ echo '[{"name":"alice"},{"name":"bob"},{"name":"charlie"}]' | jq -r '.[] | select(.name | test("^[abc]"))| .name'
alice
bob
charlie
```

`test()` in jq runs a PCRE-like regex over a string field.

### Vim search

In vim normal mode, type `/`, then your regex, then Enter. By default, vim is in **magic** mode (BRE-like). To get sane regex syntax, prefix with `\v`:

```
/\v\d+
```

That searches for one or more digits in PCRE-ish style.

### Vim substitute

```
:%s/\v(\w+) (\w+)/\2 \1/g
```

`%` means "every line." `s/pattern/replacement/g` is a global substitute. `\v` puts us in very-magic mode where `(`, `)`, `+`, `\d`, `\w` work the way most people expect.

### Replace in-place with sed

```
$ sed -i.bak -E 's/old/new/g' file.txt
```

`-i.bak` edits `file.txt` in place, saving a backup as `file.txt.bak`. On macOS sed, `-i` requires an argument — even an empty string: `-i ''`. On GNU sed, `-i` alone works.

```
# macOS:
$ sed -i '' -E 's/old/new/g' file.txt

# GNU:
$ sed -i -E 's/old/new/g' file.txt
```

Yes — this is the inconsistency that drives every Mac dev mad.

### Find files matching a pattern with ripgrep

```
$ rg -l 'TODO|FIXME' .
src/main.go
docs/notes.md
```

`-l` lists matching files only.

### grep with PCRE (`-P`, GNU only)

```
$ echo "abc123" | grep -P '(?<=abc)\d+'
abc123
```

`-P` enables PCRE in GNU grep — gives you lookahead, lookbehind, and `\d`. Not available in BSD/macOS grep by default.

### Replace with a function in JavaScript

```
$ node -e 'console.log("a1 b22 c333".replace(/\d+/g, m => `[${m}]`))'
a[1] b[22] c[333]
```

Replacement functions get the matched text and can compute the replacement.

### Multiline flag in Python verbose mode

```python
$ python3 -c 'import re; r = re.compile(r"""
... ^         # start of line
... (\w+)     # word
... \s+       # whitespace
... (\w+)     # second word
... $         # end of line
... """, re.X | re.M); print(r.findall("hello world\nfoo bar"))'
[('hello', 'world'), ('foo', 'bar')]
```

`re.X` (verbose) lets you write a regex with whitespace and comments.

### Atomic group in PCRE

```
$ echo "aaaaaaaaaaaaaaaaX" | perl -E 'print "match" if "aaaaaaaaaaaaaaaaX" =~ /^(?>a+)+b$/'
```

Without atomic, `(a+)+b` would catastrophically backtrack. With `(?>a+)+b`, it fails fast.

### Possessive quantifier in PCRE

```
$ echo "aaaaaaaaaaaaaaaaX" | perl -E 'print "match" if "aaaaaaaaaaaaaaaaX" =~ /^a++b$/'
```

`a++` is possessive — once matched, the engine refuses to give characters back.

## Common Confusions

A lot of regex pain is the same handful of confusions over and over. Here are the big ones.

### 1. Greedy vs lazy

`<.+>` against `<a><b>` greedily matches `<a><b>` (the whole thing) because `.+` swallows the world. `<.+?>` lazily matches `<a>` first because `.+?` only takes what it must. **Fix:** use `?` after a quantifier to make it lazy when you want minimal matches.

### 2. `.` matches "anything" — including too much

`example.com` (no escape) matches `examplexcom`, `exampleAcom`, etc. **Fix:** escape the dot: `example\.com`.

### 3. `^` outside `[]` vs `^` inside `[]`

Outside, `^` means "start of line." Inside `[...]` as the first character, `^` means "negation." `[^a]` is "not a." Outside any bracket, `^a` is "a at the start of a line." Two completely different jobs sharing one symbol.

### 4. Capturing group vs non-capturing group

`(abc)` captures and increments group number. `(?:abc)` groups for quantifiers but doesn't capture. If you use `(...)` just for grouping, you waste a group number. **Fix:** use `(?:...)` for groups you don't need to reference later.

### 5. `re.match` vs `re.search` vs `re.fullmatch` (Python)

- `match` anchors at the start.
- `fullmatch` anchors at both ends.
- `search` doesn't anchor.

People assume `match` checks the whole string. It doesn't. `re.match(r'\d+', '123abc')` succeeds.

### 6. sed `-i` on macOS vs GNU

GNU sed: `sed -i 's/a/b/' file`. macOS sed: `sed -i '' 's/a/b/' file` (empty string required as backup suffix). **Fix:** always use `sed -i.bak` or `sed -i '.bak'` and learn to delete `.bak` files later, since that works on both. Or use `gsed` on macOS (from `brew install gnu-sed`).

### 7. Forward slash inside slash-delimited regex

`s/http://example.com/X/` confuses sed because `/` is the delimiter. **Fix:** either escape the slashes (`s/http:\/\/example\.com/X/`) or change delimiter (`s|http://example\.com|X|` — sed accepts any character as delimiter).

### 8. Multiline flag

`^` and `$` only match line boundaries when multiline flag is on. **Fix:** turn on multiline (`re.M` in Python, `m` flag in JS, `(?m)` inline). Or use `\A` and `\z` for whole-string anchors.

### 9. Case-insensitive flag

`abc` does not match `ABC` unless you set the case-insensitive flag (`re.I`, `i` flag, `(?i)` inline). **Fix:** turn it on. Or use `[Aa][Bb][Cc]` (ugly but works everywhere).

### 10. `$` vs `\z`

`$` may match before a trailing newline or at the end of every line in multiline mode. `\z` means "absolute end of string, no exceptions." If you write `^pattern$` and your input has a trailing `\n`, `$` will match before the `\n` — fine for line matching, surprising for whole-string validation. **Fix:** use `\z` when you mean it.

### 11. `\b` semantics

`\b` is "word boundary." But what's a word character? In ASCII, `[A-Za-z0-9_]`. In Unicode mode, all Unicode letters and digits. So `\bcafé\b` in ASCII mode treats `é` as non-word, breaking the boundary at the `é`. In Unicode mode, `é` is a word character. Same regex, different planet.

### 12. `\d` is sometimes more than `[0-9]`

In ASCII mode (e.g., Python's `re` with `re.A`, or older versions), `\d` is exactly `[0-9]`. In Unicode mode (default in Python 3, JavaScript with `u`), `\d` matches all Unicode digit characters, including `٠`, `१`, `４`, etc. **Fix:** if you want strictly 0-9, write `[0-9]` explicitly.

### 13. RE2 doesn't support backreferences — and why

RE2 builds an NFA and walks it in O(n). Backreferences require remembering what was matched and comparing later, which breaks the linear-time guarantee. So RE2 (Go, Rust, ripgrep) refuses to support `\1`-style backreferences. If you need them in Go, use `github.com/dlclark/regexp2` — but you lose linear-time guarantees.

### 14. Anchored vs unanchored

By default, most regex engines try every starting position. So `\d+` against `abc123` matches `123` even though the `\d+` doesn't start at position 0. If you want to require the match to start at the beginning, anchor with `^` or use a function like `re.match` / `re.fullmatch`.

### 15. POSIX longest-match vs Perl leftmost-first

POSIX ERE picks the longest possible match: `cat|category` against `category` returns `category`. Perl-style picks the first alternative that matches: same regex returns `cat`. Most modern languages are Perl-style. `awk` and POSIX `grep -E` are POSIX-style. This catches people who switch between tools.

### 16. Whitespace in patterns

In default mode, whitespace is significant. `a b` matches `a`, space, `b`. In verbose mode (`re.X` in Python, `x` flag elsewhere), whitespace is ignored unless escaped or in a class, and `#` starts a comment. Convenient for long patterns. But people forget the flag and wonder why their `a b` regex doesn't match `ab`.

### 17. Backslashes in source code vs in regex

`'\d'` in Python is — well, used to be — interpreted as `\d`. But really, the regex engine wants the two-character sequence `\` then `d`. The safe path is to use raw strings: `r'\d'`. JavaScript regex literals (`/\d/`) avoid this. Java strings need `"\\d"` because `"\d"` is invalid. If your regex isn't behaving, check the language layer first.

### 18. Why `+` is greedy by default

Backtracking engines historically default to greedy because it tends to match more "natural" cases. But for things like HTML tags or strings between delimiters, greedy is wrong and you need lazy. **Heuristic:** any time your pattern says "everything from X to Y," try lazy first.

### 19. Lookbehind constraints

In some engines (PCRE traditional, Java with limits, older Python), lookbehind must be **fixed width** — `(?<=foo)` is fine, `(?<=\w+)` isn't. Modern engines (Python 3.7+, .NET, JS ES2018+) allow variable-width.

### 20. `-i` flag dash placement

`grep -i pattern` is different from `grep pattern -i` only in some implementations' parsing. Always put flags before the pattern to be safe.

## Vocabulary

| Term | Plain English |
|------|---------------|
| regex | Short for "regular expression." A description of a pattern of text. |
| regular expression | The formal name. A pattern that describes a set of strings. |
| pattern | The regex itself — what you are matching against. |
| match | When a regex finds a piece of text that fits its pattern. |
| NFA | Nondeterministic Finite Automaton — a state machine where multiple paths can be active at once. The model behind RE2. |
| DFA | Deterministic Finite Automaton — a state machine where at every step, exactly one next state is determined. |
| automaton | A state machine. The math behind regex engines. |
| deterministic | "Always one choice." DFAs make exactly one transition per input character. |
| nondeterministic | "Multiple paths possible." NFAs explore many possible states at once. |
| ε-transition | "Epsilon" transition — moving between states without consuming input. Used in NFA construction. |
| Thompson construction | The algorithm for building an NFA from a regex. Invented by Ken Thompson. |
| backtracking | When a regex engine tries one path, fails, undoes, and tries another. The technique most languages' engines use. |
| NFA simulation | Walking the NFA over input, tracking all possible states at once. The RE2 approach. |
| alternation | The `|` operator — "or." `a|b` matches `a` or `b`. |
| concatenation | Sticking patterns next to each other. `ab` is `a` concatenated with `b`. |
| Kleene star | The `*` operator. Zero or more. Named after Stephen Kleene. |
| plus | The `+` operator. One or more. |
| optional | The `?` operator. Zero or one. |
| repetition counts | The `{n}`, `{n,}`, `{n,m}` operators for fixed/bounded counts. |
| character class | A set of allowed characters at one position. `[abc]`, `\d`, etc. |
| range | Inside a character class, `a-z` means "from a to z." |
| negation | A `^` inside `[...]` flips the class to "anything except." |
| escaping | Using `\` to make a metacharacter literal. `\.` is a literal dot. |
| anchor | A zero-width assertion about position. `^`, `$`, `\b`. |
| word boundary | The position between a word character and a non-word character. `\b`. |
| line anchor | `^` or `$` — start or end of line. |
| string anchor | `\A` or `\z` — start or end of the whole string. |
| capturing group | A `(...)` that remembers the matched text for later. |
| non-capturing group | A `(?:...)` that groups for quantifiers but doesn't remember. |
| named group | A capturing group with a name: `(?P<name>...)` or `(?<name>...)`. |
| backreference | `\1`, `\2`, etc. Match the same text a previous group matched. |
| replacement | The text used to replace a match in a substitution. `$1` or `\1` references groups. |
| lookahead | A zero-width peek forward: `(?=...)` positive, `(?!...)` negative. |
| negative lookahead | `(?!...)` — match only if the inside does NOT follow. |
| lookbehind | A zero-width peek backward: `(?<=...)` positive, `(?<!...)` negative. |
| fixed-width lookbehind | A lookbehind with a known fixed length. Some engines require this. |
| variable-length lookbehind | A lookbehind with a flexible length. Modern engines allow it. |
| atomic group | `(?>...)` — once matched, never give back. ReDoS mitigation. |
| possessive quantifier | `*+`, `++`, `?+` — once matched, never give back. ReDoS mitigation. |
| recursion | A regex that refers to itself, like `(?R)` in PCRE. |
| conditional | `(?(1)yes|no)` — match `yes` if group 1 matched, else `no`. |
| comments | `(?#text)` — inline comment in a regex. |
| inline modifiers | `(?i)`, `(?m)`, `(?s)` — turn flags on/off mid-pattern. |
| Unicode property | `\p{L}`, `\p{N}` — match by Unicode category. |
| IsAlpha | A Unicode property class meaning "is alphabetic." |
| GeneralCategory | The top-level Unicode category (Letter, Number, etc). |
| BidiClass | Unicode bidirectional class — used for right-to-left scripts. |
| Script | Unicode script (Latin, Greek, Cyrillic, Han, etc). |
| GraphemeCluster | A user-perceived character — may be multiple codepoints. |
| codepoint | A single Unicode value, 0 to 0x10FFFF. |
| surrogate pair | UTF-16 way of encoding codepoints above 0xFFFF using two 16-bit values. |
| UTF-8 | Variable-width Unicode encoding, 1-4 bytes per codepoint. The web standard. |
| UTF-16 | Variable-width Unicode encoding, 2 or 4 bytes per codepoint. Used in JS, Java, Windows. |
| NFC | Normalization Form Canonical Composition. One way Unicode "the same string" is canonicalised. |
| NFD | Normalization Form Canonical Decomposition. The other canonical form. |
| BRE | Basic Regular Expression — POSIX, restrictive, used by classic grep/sed. |
| ERE | Extended Regular Expression — POSIX, modern, used by `grep -E`/`awk`. |
| PCRE | Perl-Compatible Regular Expression — a popular library used everywhere. |
| PCRE2 | The modern successor to PCRE. Most distros now ship PCRE2. |
| Perl regex | The original Perl regex. Slightly more features than PCRE. |
| Python re | Python's stdlib regex module. |
| Python regex | Third-party `regex` module on PyPI with extra features. |
| JavaScript RegExp | Built-in JS regex. Backtracking engine. |
| Java regex | `java.util.regex.Pattern`. Backtracking, with PatternSyntaxException for errors. |
| Go regexp | Go's stdlib package. RE2-style. Linear time, no backreferences. |
| Rust regex | Rust's `regex` crate. RE2-style. Same trade-offs as Go. |
| RE2 | Google's regex library. Linear time. |
| RE2J | Java port of RE2. |
| ICU regex | Unicode-aware regex from the ICU library. |
| Boost.Regex | C++ regex library, predates `std::regex`. |
| std::regex | C++ standard library regex. Often slow. |
| sed regex | sed's regex flavor, BRE by default, ERE with `-E`. |
| awk regex | awk's regex flavor, ERE-ish. |
| grep | Search files for patterns. BRE by default, `-E` for ERE, `-P` for PCRE on GNU. |
| egrep | Same as `grep -E`. |
| fgrep | Same as `grep -F` — fixed strings, no regex at all. |
| ripgrep | Modern grep replacement. Rust regex engine. Fast. RE2 trade-offs. |
| ag | "The silver searcher." Older grep replacement, PCRE-based. |
| ack | Perl-based grep replacement, predecessor to ag. |
| ackmate | Editor-friendly variant of ack. |
| fzf | Fuzzy finder. Not exactly regex, but in the same family of "filter text" tools. |
| peco | Another fuzzy filter. |
| jq -R 'test' | Run a regex test inside jq on JSON string fields. |
| test() (JS) | `RegExp.prototype.test()` — returns true/false. |
| match() (JS) | `String.prototype.match()` — returns match info. |
| exec() (JS) | `RegExp.prototype.exec()` — stateful match across iterations. |
| replace() (JS) | `String.prototype.replace()` — substitute. |
| re.search (Python) | Find first match anywhere in the string. |
| re.match (Python) | Match starting at the beginning of the string. |
| re.fullmatch (Python) | Match the entire string. |
| re.findall (Python) | Return all matches as a list. |
| re.finditer (Python) | Iterator of match objects. |
| re.sub (Python) | Substitute matches. |
| Pattern.compile (Java) | Compile a regex into a Pattern object. |
| Matcher.find (Java) | Find next match. |
| Matcher.group (Java) | Get a captured group. |
| regexp.MustCompile (Go) | Compile and panic on error. |
| FindAllString (Go) | Return all matched strings. |
| ReplaceAllStringFunc (Go) | Replace using a callback. |
| Regex::new (Rust) | Construct a regex. |
| captures (Rust) | Get capture groups. |
| replace_all (Rust) | Substitute all matches. |
| preg_match (PHP) | PHP's PCRE match function. |
| Regex (.NET) | .NET's regex class in `System.Text.RegularExpressions`. |
| re_search (Emacs) | Emacs Lisp regex search. |
| vim regex | Vim's regex with magic/very-magic/nomagic modes. |
| POSIX bracket expression | `[[:digit:]]`, `[[:alpha:]]`, etc. |
| [[:alpha:]] | POSIX class for letters. |
| [[:digit:]] | POSIX class for digits. |
| [[:space:]] | POSIX class for whitespace. |
| [[:alnum:]] | POSIX class for alphanumeric. |
| [[:upper:]] | POSIX class for uppercase. |
| [[:lower:]] | POSIX class for lowercase. |
| [[:xdigit:]] | POSIX class for hex digits. |
| [[:punct:]] | POSIX class for punctuation. |
| [[:print:]] | POSIX class for printable characters. |
| [[:graph:]] | POSIX class for non-space printable characters. |
| [[:cntrl:]] | POSIX class for control characters. |
| [[:blank:]] | POSIX class for space and tab. |
| anchored | A regex required to match at a specific position (start, end, both). |
| fullmatch | A match that covers the entire input. |
| partial match | A match that covers some but not all of the input. |
| longest-match | POSIX semantics — pick the longest possible match. |
| leftmost-first | Perl semantics — pick the first alternative that works. |
| ReDoS | Regular Expression Denial of Service — exploit catastrophic backtracking. |
| polynomial blowup | Performance degradation that scales as a polynomial in input size. |
| exponential blowup | Performance degradation that scales as 2^n in input size. The catastrophic kind. |
| atomic groups as ReDoS mitigation | `(?>...)` prevents backtracking into the group. |
| possessive *+ ++ ?+ as alternative | Quantifiers that don't give back what they matched. |
| regex fuzzing | Automatically generating inputs to test regex behavior. |
| regex testing tool | regex101.com, regexr.com, regextester — web-based regex sandboxes. |
| metacharacter | A character with special regex meaning: `.`, `*`, `+`, `?`, `(`, etc. |
| literal | A character that matches itself. Most letters and digits are literal. |
| greedy | Quantifier that matches as much as possible. Default in most flavors. |
| lazy | Quantifier that matches as little as possible. Marked with `?` after. |
| reluctant | Another name for "lazy." |
| zero-width assertion | Something that matches a position, not characters. Anchors and lookarounds. |
| group number | The index of a capturing group, counted left-to-right. |
| flags | Modifiers like case-insensitive, multiline, dotall. Apply to the whole pattern. |
| ASCII mode | Regex semantics restricted to ASCII characters. `\d` = `[0-9]`. |
| Unicode mode | Regex semantics covering all of Unicode. `\d` includes non-Latin digits. |
| dotall | A flag (`s`) that makes `.` match newlines too. |
| verbose mode | A flag (`x`) that lets you write multi-line patterns with whitespace and comments. |
| extended | Often a synonym for ERE in tools that have BRE/ERE modes. |
| compiled regex | A regex preprocessed into an internal form for faster repeated matching. |

## ASCII Diagrams

### NFA for `ab*c`

```
                     +-----+
                     |  b  |
                     v     |
   start --a--> ( S1 )-----+
                     |
                     | c
                     v
                ( S2 = accept )
```

Trace `abbbbc`:

```
position: a b b b b c
state:    S1→S1→S1→S1→S1→S1→S2
                           accept!
```

### DFA for the same regex

```
   start --a--> [ S1 ] ----b----+
                  |             |
                  | c           v
                  v          [ S1 ]  (loop back)
                [ S2 ]
                accept
```

(For this regex, the DFA is structurally the same as the NFA — there's only one nondeterministic choice at S1, which is "is the next character `b` (loop) or `c` (proceed)," and that choice is determined by the input character. So this is already deterministic.)

### Catastrophic backtracking trace for `^(a+)+b$` on `aaaaaX`

The regex is "start, one-or-more `a`'s, repeated one-or-more times, then `b`, then end."

```
Input: a a a a a X
       0 1 2 3 4 5

Try 1: outer (aaaaa), 1 iteration. Input pos 5. Need b. Got X. FAIL. Backtrack.
Try 2: outer (aaaa), 1 iter. Pos 4. Try inner again: (a), 2 iters. Pos 5. Need b. X. FAIL.
Try 3: outer (aaaa), 1 iter. Try ()+. Already 2 iters tried. Pos 4. Need b. FAIL. Backtrack outer.
Try 4: outer (aaa), 1 iter. Pos 3. Try (aa), 2 iter. Pos 5. Need b. X. FAIL.
Try 5: outer (aaa), 1 iter. Pos 3. Try (a)(a), 2 then 3 iter. Pos 5. X. FAIL.
Try 6: outer (aa), 1 iter. Pos 2. Try (aaa), 2 iter. Pos 5. X. FAIL.
Try 7: outer (aa), 1 iter. Try (aa)(a). Pos 5. X. FAIL.
Try 8: outer (aa), 1 iter. Try (a)(aa). Pos 5. X. FAIL.
Try 9: outer (aa), 1 iter. Try (a)(a)(a). Pos 5. X. FAIL.
Try 10: outer (a), 1 iter. Then (aaaa). Pos 5. X. FAIL.
... and so on through every partition of the 5 a's into ordered groups ...

For 5 a's: 16 paths. For 10 a's: 512 paths. For 20 a's: 524,288 paths.
For 50 a's: 562,949,953,421,312 paths. Each path takes nanoseconds.
The regex never returns.
```

The number of paths is roughly 2^(n-1) where n is the number of a's. Pure exponential blowup.

### Capture group structure tree for `(\w+)@(\w+)\.(\w+)`

```
Pattern: (\w+) @ (\w+) \. (\w+)
         ^^^^^   ^^^^^    ^^^^^
         group 1 group 2  group 3

Match on "stevie@bellis.tech":

   group 0 (whole): "stevie@bellis.tech"
   ├── group 1 (\w+): "stevie"
   ├── literal "@"
   ├── group 2 (\w+): "bellis"
   ├── literal "."
   └── group 3 (\w+): "tech"
```

Group 0 is always "the whole match." Groups 1, 2, 3 are the parenthesised subparts. Numbering follows the opening paren left-to-right.

### NFA construction for alternation `cat|dog`

```
                     +---c---a---t---+
                    /                 \
   start --ε--> ( split )            ( accept )
                    \                 /
                     +---d---o---g---+

   ε = epsilon (free) transition
```

The split state has two ε-transitions, one to each branch. Both are "active" simultaneously during NFA simulation.

### Backtracking engine path on the same regex

```
Input: dog

Try cat path first (left-to-right alternation):
  Position 0: read 'd'. Need 'c'. FAIL. Backtrack.

Try dog path:
  Position 0: read 'd'. Need 'd'. OK.
  Position 1: read 'o'. Need 'o'. OK.
  Position 2: read 'g'. Need 'g'. OK.
  ACCEPT.
```

Backtracking tries one branch fully before falling back. NFA simulation tries both branches simultaneously, no backing up.

### Anchored vs unanchored

```
Regex:     \d+
Unanchored search slides:
   Position 0: 'a' — fail
   Position 1: 'b' — fail
   Position 2: 'c' — fail
   Position 3: '1' — match starts. Eat '123'. Done. Match "123".

Regex:     ^\d+
Anchored search:
   Position 0 only: 'a' — fail. No more positions to try (^ pins to 0). Done. No match.
```

## Try This

Eight little exercises. The answers are at the end of each.

### 1. Match a US ZIP code

A US ZIP code is five digits, optionally followed by a dash and four more digits. Write a regex that matches `12345` or `12345-6789` but not `1234` or `123456`.

```
echo "ZIPs: 12345, 12345-6789, 90210, 1234, abc" | rg -o '\b\d{5}(-\d{4})?\b'
12345
12345-6789
90210
```

### 2. Find lines that don't have a comment

In a config file, lines starting with `#` are comments. Find all non-comment, non-blank lines.

```
echo -e '# comment\n\nname=stevie\n#another\nport=22' | rg -v '^\s*(#|$)'
name=stevie
port=22
```

`-v` inverts the match. The pattern matches lines that ARE blank or commented; `-v` keeps the rest.

### 3. Extract all hashtags from text

```
echo 'Loving #regex and #python today!' | rg -o '#\w+'
#regex
#python
```

### 4. Replace tabs with two spaces

```
$ printf 'col1\tcol2\tcol3\n' | sed -E $'s/\t/  /g'
col1  col2  col3
```

The `$'...'` syntax in bash interprets `\t` as a real tab.

### 5. Capture and reformat dates

Convert `2026-04-27` to `04/27/2026`.

```
$ echo '2026-04-27' | sed -E 's|(\d{4})-(\d{2})-(\d{2})|\2/\3/\1|'
```

Wait — sed's BRE/ERE doesn't have `\d`. Use:

```
$ echo '2026-04-27' | sed -E 's|([0-9]{4})-([0-9]{2})-([0-9]{2})|\2/\3/\1|'
04/27/2026
```

Note we used `|` as a delimiter so the slashes in the replacement don't conflict.

### 6. Find all IPv4-shaped strings in a log

```
$ rg -o '\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b' access.log
192.168.1.10
8.8.8.8
10.0.0.1
```

### 7. Validate a basic password (Python)

At least 8 characters, must contain at least one digit and one letter.

```python
$ python3 -c '
import re
pw = "abc12345"
ok = len(pw) >= 8 and re.search(r"\d", pw) and re.search(r"[A-Za-z]", pw)
print("ok" if ok else "no")
'
ok
```

### 8. Pull out URLs from HTML

```
$ echo '<a href="https://unheaded.dev">' | rg -oP '(?<=href=")[^"]+'
https://unheaded.dev
```

`-P` enables PCRE in GNU grep / ripgrep so we can use lookbehind. The lookbehind asserts `href="` came before, then we match everything until the next `"`.

## Where to Go Next

Once you can read regexes and write small ones from scratch, the natural next steps are:

- **`data-formats/regex`** — the operational reference sheet with all the syntax in one place, no analogies.
- **`data-formats/awk`**, **`data-formats/sed`**, **`data-formats/jq`** — the three classic Unix text tools that all use regex heavily.
- **`terminal/ripgrep`** — fast, modern grep with sane defaults.
- **`ramp-up/python-eli5`** — Python's `re` module is one of the most commonly-used regex APIs.
- **`ramp-up/vim-eli5`** — vim has its own regex flavor with quirks.
- **`ramp-up/bash-eli5`** — bash uses regex in `[[ string =~ pattern ]]`, with ERE semantics.

For deeper understanding:

- Read **Mastering Regular Expressions** by Jeffrey Friedl. The canonical regex book.
- Read **swtch.com/~rsc/regexp/** — Russ Cox's three-part series on RE2, NFA simulation, and the difference between linear-time and backtracking engines. If you build anything with regex over user input, read this.
- Try **regex101.com** — paste any regex and see it explained, animated, and tested against sample inputs. Best regex sandbox on the web.
- Try **regexr.com** — similar tool, slightly different vibe.

Once you are comfortable, look at production engines:

- The **RE2** source on GitHub (Google).
- Rust's **`regex` crate** docs.
- Go's **`regexp/syntax`** package — the syntax tree internals.

## See Also

- `data-formats/regex` — operational reference, no analogies.
- `data-formats/jq` — JSON wrangler with regex support via `test`/`match`/`capture`.
- `data-formats/awk` — record-oriented text processor, ERE-based.
- `data-formats/sed` — stream editor, BRE by default, ERE with `-E`.
- `terminal/ripgrep` — fast Rust-based grep.
- `ramp-up/bash-eli5` — bash basics, including `[[ =~ ]]` regex matching.
- `ramp-up/python-eli5` — Python's `re` module is one of the most-used regex APIs.
- `ramp-up/vim-eli5` — vim's own regex flavor.
- `ramp-up/linux-kernel-eli5` — the foundation under everything.

## References

- **Mastering Regular Expressions** by Jeffrey Friedl, 3rd edition, O'Reilly. The book.
- **regex101.com docs** — <https://regex101.com/help> — interactive regex sandbox documentation.
- **swtch.com/~rsc/regexp/** — Russ Cox's articles on RE2 and the regex matching trichotomy.
- **perlretut** — Perl's regex tutorial: `perldoc perlretut`.
- **perlretrap** — Perl's regex traps: `perldoc perlretrap`.
- **Python `re` docs** — <https://docs.python.org/3/library/re.html>.
- **Go `regexp/syntax` docs** — <https://pkg.go.dev/regexp/syntax>.
- **PCRE2 docs** — <https://www.pcre.org/current/doc/html/>.
- **JavaScript MDN RegExp** — <https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/RegExp>.
- **Java Pattern javadoc** — <https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/regex/Pattern.html>.
- **Rust `regex` crate** — <https://docs.rs/regex/>.
- **POSIX regex spec (IEEE Std 1003.1)** — <https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap09.html>.
- **RE2 wiki** — <https://github.com/google/re2/wiki/Syntax>.
- **OWASP ReDoS** — <https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS>.
- **Cloudflare 2019-07-02 outage post-mortem** — <https://blog.cloudflare.com/details-of-the-cloudflare-outage-on-july-2-2019/> — case study in real-world ReDoS.
