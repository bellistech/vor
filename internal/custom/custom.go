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

// Add copies a markdown file into the custom sheets directory.
func Add(path string) error {
	dir := Dir()
	if dir == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	dest := filepath.Join(dir, "uncategorized")
	if err := os.MkdirAll(dest, 0755); err != nil {
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

	out, err := os.Create(filepath.Join(dest, name))
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	fmt.Fprintf(os.Stderr, "added %s to %s\n", name, dest)
	return nil
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
