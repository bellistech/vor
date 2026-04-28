package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/lipgloss"
)

// Theme is the TUI color palette. All fields are 6-digit hex colors with the
// `#` prefix. Empty fields fall back to the corresponding Amber Throne default
// at apply-time, so a partial theme file still produces a usable TUI.
//
// Persisted as JSON at ~/.config/cs/theme.json. Tests can override the path
// via SetThemeFile().
type Theme struct {
	Name     string `json:"name,omitempty"` // human-readable label, ignored by code
	Gold     string `json:"gold,omitempty"`
	Purple   string `json:"purple,omitempty"`
	Silver   string `json:"silver,omitempty"`
	Violet   string `json:"violet,omitempty"`
	Orange   string `json:"orange,omitempty"`
	Emerald  string `json:"emerald,omitempty"`
	DimGray  string `json:"dim_gray,omitempty"`
	DarkAmber string `json:"dark_amber,omitempty"`
}

// AmberThrone is the default palette — matches the original hard-coded values.
var AmberThrone = Theme{
	Name:      "amber-throne",
	Gold:      "#D4A017",
	Purple:    "#7B2FBE",
	Silver:    "#B0B0B0",
	Violet:    "#C9A0DC",
	Orange:    "#FF6347",
	Emerald:   "#50C878",
	DimGray:   "#555555",
	DarkAmber: "#8B6914",
}

// themeFileOverride lets tests redirect the theme path.
var themeFileOverride string

// SetThemeFile overrides the theme path. Test-only.
func SetThemeFile(path string) { themeFileOverride = path }

// ThemeFile returns the resolved theme path (empty if HOME unavailable).
func ThemeFile() string {
	if themeFileOverride != "" {
		return themeFileOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "cs", "theme.json")
}

// hexColorRE matches a strict 6-digit hex color (#RRGGBB). Looser forms
// (#RGB, named CSS colors) are explicitly rejected — keep the spec narrow.
var hexColorRE = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateHex returns nil if s is a valid #RRGGBB hex color, error otherwise.
// Empty string is accepted (falls back to default at apply time).
func validateHex(field, s string) error {
	if s == "" {
		return nil
	}
	if !hexColorRE.MatchString(s) {
		return fmt.Errorf("theme: %s = %q is not a #RRGGBB hex color", field, s)
	}
	return nil
}

// LoadTheme reads ~/.config/cs/theme.json and validates each color. On any
// error (file missing, bad JSON, invalid color) returns the AmberThrone
// default. Returns the loaded-or-default theme along with a non-nil error
// only when the file existed but was malformed (caller may log it).
func LoadTheme() (Theme, error) {
	path := ThemeFile()
	if path == "" {
		return AmberThrone, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// missing file = silently use default
		return AmberThrone, nil
	}

	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return AmberThrone, fmt.Errorf("theme.json: invalid JSON: %w", err)
	}

	for _, c := range []struct {
		field, val string
	}{
		{"gold", t.Gold},
		{"purple", t.Purple},
		{"silver", t.Silver},
		{"violet", t.Violet},
		{"orange", t.Orange},
		{"emerald", t.Emerald},
		{"dim_gray", t.DimGray},
		{"dark_amber", t.DarkAmber},
	} {
		if err := validateHex(c.field, c.val); err != nil {
			return AmberThrone, err
		}
	}

	// Fill missing fields from default — partial themes are valid.
	if t.Gold == "" {
		t.Gold = AmberThrone.Gold
	}
	if t.Purple == "" {
		t.Purple = AmberThrone.Purple
	}
	if t.Silver == "" {
		t.Silver = AmberThrone.Silver
	}
	if t.Violet == "" {
		t.Violet = AmberThrone.Violet
	}
	if t.Orange == "" {
		t.Orange = AmberThrone.Orange
	}
	if t.Emerald == "" {
		t.Emerald = AmberThrone.Emerald
	}
	if t.DimGray == "" {
		t.DimGray = AmberThrone.DimGray
	}
	if t.DarkAmber == "" {
		t.DarkAmber = AmberThrone.DarkAmber
	}
	return t, nil
}

// ApplyTheme reassigns the package-level lipgloss styles to the given theme.
// Safe to call once at TUI startup before Run(). Not thread-safe — should
// not be called concurrently with View() rendering.
func ApplyTheme(t Theme) {
	gold = lipgloss.Color(t.Gold)
	purple = lipgloss.Color(t.Purple)
	silver = lipgloss.Color(t.Silver)
	violet = lipgloss.Color(t.Violet)
	orange = lipgloss.Color(t.Orange)
	emerald = lipgloss.Color(t.Emerald)
	dimGray = lipgloss.Color(t.DimGray)
	darkAmber = lipgloss.Color(t.DarkAmber)

	// Re-derive the styles that depend on the colors above.
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(gold)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(gold)
	normalStyle = lipgloss.NewStyle().Foreground(silver)
	dimStyle = lipgloss.NewStyle().Foreground(dimGray)
	countStyle = lipgloss.NewStyle().Foreground(violet)
	statusStyle = lipgloss.NewStyle().Foreground(emerald)
	filterStyle = lipgloss.NewStyle().Foreground(orange).Bold(true)
	barFull = lipgloss.NewStyle().Foreground(gold)
	barEmpty = lipgloss.NewStyle().Foreground(dimGray)
	borderColor = lipgloss.NewStyle().Foreground(purple)
	posStyle = lipgloss.NewStyle().Foreground(violet)
	descSelStyle = lipgloss.NewStyle().Foreground(silver)
	descDimStyle = lipgloss.NewStyle().Foreground(dimGray)

	// silence the linter for darkAmber (declared but currently only
	// assigned — kept available for future themed elements)
	_ = darkAmber
}
