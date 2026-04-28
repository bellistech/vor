package custom

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Dir returns the custom sheets directory.
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "cs", "sheets")
}

// Load returns an os.DirFS for custom sheets, or nil if the dir doesn't exist.
func Load() fs.FS {
	dir := Dir()
	if dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	return os.DirFS(dir)
}

// Add copies a markdown file into the custom sheets directory under
// "uncategorized/". For category-aware placement, use AddTo.
func Add(path string) error {
	return AddTo(path, "")
}

// AddTo copies a markdown file into the custom sheets directory, placing
// it in the given category subdirectory (or "uncategorized" if category
// is the empty string). Performs three small UX checks before writing:
//
//  1. Confirms the source file exists and ends in `.md` (or appends .md).
//  2. Light markdown sanity check — warns to stderr if the file lacks an
//     H1 line (`# ...`). Doesn't fail; some users have legitimate
//     reasons (template, fragment, etc).
//  3. Conflict detection — refuses to overwrite an existing custom sheet
//     by default. Set ConfirmOverwrite to true to allow.
//
// Output on success names the destination path and lists existing
// categories under the custom dir so the user knows where else they
// could have placed the file.
func AddTo(path, category string) error {
	dir := Dir()
	if dir == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	if category == "" {
		category = "uncategorized"
	}
	// sanitize: only allow [a-z0-9-] in category names
	cat := strings.ToLower(strings.TrimSpace(category))
	for _, r := range cat {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("invalid category %q (allowed: a-z, 0-9, hyphen)", category)
		}
	}

	dest := filepath.Join(dir, cat)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create custom dir: %w", err)
	}

	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer src.Close()

	name := filepath.Base(path)
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	destPath := filepath.Join(dest, name)
	if _, statErr := os.Stat(destPath); statErr == nil && !ConfirmOverwrite {
		return fmt.Errorf(
			"refusing to overwrite existing custom sheet %s\n"+
				"  use `cs --edit %s` to edit it in $EDITOR, or remove and re-add",
			destPath, strings.TrimSuffix(name, ".md"),
		)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	defer out.Close()

	body, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	if !hasMarkdownH1(body) {
		fmt.Fprintf(os.Stderr,
			"warning: %s does not start with an H1 heading (`# Title`).\n"+
				"  Custom sheets render best with a single top-level H1.\n",
			name)
	}

	if _, err := out.Write(body); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Fprintf(os.Stderr, "added %s to %s\n", name, dest)
	if cat == "uncategorized" {
		if existing := listCategories(dir); len(existing) > 0 {
			fmt.Fprintf(os.Stderr, "  tip: existing custom categories are: %s\n",
				strings.Join(existing, ", "))
			fmt.Fprintf(os.Stderr, "  to place in a category, run: cs --add %s <category>\n", path)
		}
	}
	return nil
}

// ConfirmOverwrite, when true, allows AddTo to clobber an existing custom
// sheet at the destination. Default false (refuses overwrite). Callers can
// flip this when a CLI `--force` flag or interactive y/N is in effect.
var ConfirmOverwrite bool

// hasMarkdownH1 reports whether body has an H1 (`# ...`) on any non-blank
// line that is not inside a fenced code block. Cheap heuristic — handles
// the obvious cases without a full Markdown parser.
func hasMarkdownH1(body []byte) bool {
	inFence := false
	for _, line := range strings.Split(string(body), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(trim, "# ") {
			return true
		}
	}
	return false
}

// listCategories returns the names of the existing custom-sheet category
// subdirectories under dir. Used to nudge the user toward known categories.
func listCategories(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "uncategorized" {
			out = append(out, e.Name())
		}
	}
	return out
}

// Edit opens a topic in $EDITOR. If the topic only exists as embedded,
// it copies it to the custom dir first.
func Edit(topic string, embeddedFS fs.FS) error {
	dir := Dir()
	if dir == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Find existing custom file
	var target string
	customDir := Dir()
	filepath.WalkDir(customDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.TrimSuffix(d.Name(), ".md") == topic {
			target = path
		}
		return nil
	})

	// If not found in custom, copy from embedded
	if target == "" {
		var embeddedPath string
		fs.WalkDir(embeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.TrimSuffix(d.Name(), ".md") == topic {
				embeddedPath = path
			}
			return nil
		})

		if embeddedPath != "" {
			data, err := fs.ReadFile(embeddedFS, embeddedPath)
			if err != nil {
				return fmt.Errorf("read embedded %s: %w", topic, err)
			}

			// Preserve category directory
			cleaned := strings.TrimPrefix(embeddedPath, "sheets/")
			cat := filepath.Dir(cleaned)
			destDir := filepath.Join(customDir, cat)
			os.MkdirAll(destDir, 0755)

			target = filepath.Join(destDir, filepath.Base(embeddedPath))
			if err := os.WriteFile(target, data, 0644); err != nil {
				return fmt.Errorf("write custom %s: %w", target, err)
			}
			fmt.Fprintf(os.Stderr, "copied %s to %s for editing\n", topic, target)
		} else {
			// New custom sheet
			destDir := filepath.Join(customDir, "uncategorized")
			os.MkdirAll(destDir, 0755)
			target = filepath.Join(destDir, topic+".md")
			template := fmt.Sprintf("# %s\n\nDescription here.\n\n## Section\n\n```bash\n# example\n```\n", topic)
			if err := os.WriteFile(target, []byte(template), 0644); err != nil {
				return fmt.Errorf("write template: %w", err)
			}
		}
	}

	cmd := exec.Command(editor, target)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
