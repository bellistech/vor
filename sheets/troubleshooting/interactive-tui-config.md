# Interactive TUI — Config & Persistence

How to customize the `vor -i` (or `cs -i`) interactive browser: keybindings, persistent filter history, custom color theme.

## What `vor -i` looks like

```bash
vor -i        # launch full-screen interactive browser
cs  -i        # same binary, legacy alias

# Top of the screen: header with sheet/category counts
# Body:             scrollable list of categories → topics → content
# Bottom:           status line with key hints
```

The TUI is a [bubbletea](https://github.com/charmbracelet/bubbletea) app rendered through [lipgloss](https://github.com/charmbracelet/lipgloss). All keystrokes are local; the interactive session never makes network calls (the `-so` Stack Overflow lookup is a separate one-shot CLI flag).

## Keybindings

```
Navigation
  j / ↓        next item
  k / ↑        previous item
  g / Home     jump to first
  G / End      jump to last
  space / PgDn scroll page (in content view)
  PgUp         scroll back a page (in content view)

Selection
  enter / l    open selected (category → topic → content)
  h / esc      go back one level
  d            show deep-dive detail page (if available)

Filter
  /            start filter (live as you type)
  enter        commit filter
  esc          clear filter
  ↑ / ↓        (during filter) recall older / newer history entry

Other
  ?            toggle full-screen help overlay (esc closes)
  q / Ctrl-C   quit (from root) or back (from deeper)
```

The `?` overlay shows the same table inside the TUI.

## Filter History (persistent)

Every time you commit a filter with **enter**, the value is pushed to history and saved to disk. Open the filter again and press **↑** to recall it.

```bash
~/.cache/cs/tui-history       # one entry per line, newest at the bottom
```

Behavior:

- Capped at **50 entries**; oldest roll off automatically.
- Consecutive duplicates are collapsed (typing the same query twice doesn't bloat history).
- Empty / whitespace-only entries are dropped.
- Atomic write (temp + rename) — concurrent TUI sessions won't corrupt the file.
- Failures are silently ignored (history is best-effort, never fatal to the TUI).

Inspect or edit the history file directly if you want:

```bash
cat ~/.cache/cs/tui-history
```

```bash
# Clear history
rm ~/.cache/cs/tui-history

# Or hand-curate
$EDITOR ~/.cache/cs/tui-history
```

Inside the filter input:

- **↑** walks toward older entries (newest-first stop at the top).
- **↓** walks back toward the newest, then a final **↓** clears the input back to a fresh prompt (live mode).
- Typing any character exits recall mode — your typed input becomes the live filter.

## Theme — Custom Colors

The default palette is **Amber Throne** (gold + purple + violet on a dim-gray ground). You can override any subset of the 8 named colors by writing `~/.config/cs/theme.json`.

### File location

```bash
mkdir -p ~/.config/cs
chmod 700 ~/.config/cs                   # not strictly required; tidy default
$EDITOR ~/.config/cs/theme.json
```

### Schema

```json
{
  "name":       "midnight",
  "gold":       "#FFFFFF",
  "purple":     "#5500AA",
  "silver":     "#888888",
  "violet":     "#CC88FF",
  "orange":     "#FF8800",
  "emerald":    "#00FF88",
  "dim_gray":   "#333333",
  "dark_amber": "#222200"
}
```

| Key | Default (Amber Throne) | Used by |
|-----|------------------------|---------|
| `gold`      | `#D4A017` | titles, selected items, full bar segments |
| `purple`    | `#7B2FBE` | borders |
| `silver`    | `#B0B0B0` | normal text, descriptions of selected items |
| `violet`    | `#C9A0DC` | counts, position indicators |
| `orange`    | `#FF6347` | filter prompt indicator |
| `emerald`   | `#50C878` | status line |
| `dim_gray`  | `#555555` | dimmed text, empty bar segments, dim descriptions |
| `dark_amber`| `#8B6914` | reserved (currently unused) |

All keys are **optional**. Missing keys fall back to the Amber Throne value, so a partial theme is valid:

```json
{ "name": "neon-teal", "gold": "#00FFEE", "purple": "#FF00AA" }
```

### Validation

The loader is strict on color format:

```
✓ #RRGGBB         (e.g. #1A2B3C, #aBcDeF)
✗ #RGB            (3-digit shorthand rejected)
✗ named colors    (e.g. "red", "blue" rejected)
✗ rgb()/hsl()     (function syntax not parsed)
✗ no leading #    (FF0000 without # rejected)
```

On any error (file missing, bad JSON, invalid color) the TUI silently falls back to Amber Throne — your TUI launch is never blocked by a bad theme file. Errors during JSON parse OR hex validation print a one-line warning to stderr; missing-file cases are silent.

### Example themes to copy-paste

**Solarized-ish dark**

```json
{
  "name":       "solarized-dark",
  "gold":       "#B58900",
  "purple":     "#6C71C4",
  "silver":     "#93A1A1",
  "violet":     "#D33682",
  "orange":     "#CB4B16",
  "emerald":    "#859900",
  "dim_gray":   "#586E75",
  "dark_amber": "#073642"
}
```

**Mono / accessibility**

```json
{
  "name":       "mono-accessible",
  "gold":       "#FFFFFF",
  "purple":     "#FFFFFF",
  "silver":     "#FFFFFF",
  "violet":     "#FFFFFF",
  "orange":     "#FFFFFF",
  "emerald":    "#FFFFFF",
  "dim_gray":   "#999999",
  "dark_amber": "#666666"
}
```

(Pair with `NO_COLOR=1` for cleaner pipe-and-redirect behavior.)

**Dracula-ish**

```json
{
  "name":       "dracula",
  "gold":       "#F1FA8C",
  "purple":     "#BD93F9",
  "silver":     "#F8F8F2",
  "violet":     "#FF79C6",
  "orange":     "#FFB86C",
  "emerald":    "#50FA7B",
  "dim_gray":   "#6272A4",
  "dark_amber": "#44475A"
}
```

## Troubleshooting

### Help overlay doesn't show

Press `?` from any state (categories / topics / content). If the bottom status line says `help — press ? or esc to close`, the overlay rendered. If not, check the terminal width — overlay needs ≥60 columns.

### Filter history not persisting

```bash
ls -la ~/.cache/cs/tui-history
```

If the file is missing after committing a filter:

- Confirm `~/.cache/cs/` is writable (`stat ~/.cache/cs/`)
- Check disk space (`df -h ~/.cache/`)
- Verify `HOME` is set in your shell (`echo $HOME`)

### Theme didn't load

```bash
# Sanity-check JSON
cat ~/.config/cs/theme.json | jq .

# Watch for the warning on stderr (run in a fresh terminal)
vor -i 2>/tmp/vor.err
# In another terminal:
cat /tmp/vor.err
```

If you see `warning: theme.json: invalid JSON: ...` or `warning: theme: <field> = "..." is not a #RRGGBB hex color`, the file fell back to Amber Throne. Fix the offending entry and relaunch.

### Reset everything

```bash
# Remove history + theme; next launch is pristine
rm -f ~/.cache/cs/tui-history
rm -f ~/.config/cs/theme.json
```

## Anti-features (NOT supported)

- Mouse interaction — TUI is keyboard-only by design.
- Multiple panels side-by-side — single-panel + drill-down stays fast and accessible.
- Animated transitions — TUI is meant to feel snappy, not flashy.
- Per-category theme overrides — global theme only.
- Theme hot-reload — the file is read once at TUI launch.

## See Also

- `troubleshooting/stack-overflow-cli` — the bonus `-so` flag setup
- `shell/bash` — `$EDITOR` and `~/.config` conventions
- `data-formats/json` — JSON syntax reference if your theme.json doesn't parse

## References

- bubbletea — <https://github.com/charmbracelet/bubbletea>
- lipgloss — <https://github.com/charmbracelet/lipgloss>
- glamour (the markdown renderer) — <https://github.com/charmbracelet/glamour>
- XDG Base Directory Specification — <https://specifications.freedesktop.org/basedir-spec/>
