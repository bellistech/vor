package render

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// TermWidth returns the terminal width, defaulting to 80.
func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// TermHeight returns the terminal height, defaulting to 24.
func TermHeight() int {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return 24
	}
	return h
}

// Render renders markdown for terminal output.
func Render(content string) (string, error) {
	if !IsTTY() || os.Getenv("NO_COLOR") != "" {
		return content, nil
	}

	width := TermWidth()
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content, nil
	}
	return r.Render(content)
}

// Output renders and outputs content, using a pager if needed.
func Output(content string) error {
	rendered, err := Render(content)
	if err != nil {
		return err
	}

	if !IsTTY() {
		_, err = fmt.Print(rendered)
		return err
	}

	lines := strings.Count(rendered, "\n")
	if lines > TermHeight() {
		return pager(rendered)
	}

	_, err = fmt.Print(rendered)
	return err
}

func pager(content string) error {
	p := os.Getenv("PAGER")
	if p == "" {
		p = "less"
	}

	args := []string{}
	if p == "less" {
		args = append(args, "-R")
	}

	cmd := exec.Command(p, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Print(content)
		return nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Print(content)
		return nil
	}

	io.WriteString(stdin, content)
	stdin.Close()
	return cmd.Wait()
}
