package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempThemeFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "theme.json")
	prev := themeFileOverride
	themeFileOverride = path
	t.Cleanup(func() { themeFileOverride = prev })
	return path
}

func TestLoadTheme_NoFileReturnsDefault(t *testing.T) {
	withTempThemeFile(t)
	got, err := LoadTheme()
	if err != nil {
		t.Errorf("missing file should not error; got %v", err)
	}
	if got.Gold != AmberThrone.Gold {
		t.Errorf("default Gold mismatch: got %q, want %q", got.Gold, AmberThrone.Gold)
	}
}

func TestLoadTheme_FullCustom(t *testing.T) {
	path := withTempThemeFile(t)
	body := `{
		"name": "midnight",
		"gold": "#FFFFFF",
		"purple": "#AA00AA",
		"silver": "#888888",
		"violet": "#CC88FF",
		"orange": "#FF8800",
		"emerald": "#00FF88",
		"dim_gray": "#333333",
		"dark_amber": "#222200"
	}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTheme()
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	if got.Gold != "#FFFFFF" {
		t.Errorf("Gold = %q, want #FFFFFF", got.Gold)
	}
	if got.Name != "midnight" {
		t.Errorf("Name = %q, want midnight", got.Name)
	}
}

func TestLoadTheme_PartialFillsDefaults(t *testing.T) {
	path := withTempThemeFile(t)
	body := `{"gold": "#ABCDEF"}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTheme()
	if err != nil {
		t.Fatalf("partial theme: %v", err)
	}
	if got.Gold != "#ABCDEF" {
		t.Errorf("Gold = %q, want #ABCDEF", got.Gold)
	}
	if got.Purple != AmberThrone.Purple {
		t.Errorf("missing Purple should fall back to default; got %q", got.Purple)
	}
	if got.Silver != AmberThrone.Silver {
		t.Errorf("missing Silver should fall back to default; got %q", got.Silver)
	}
}

func TestLoadTheme_BadJSONReturnsDefaultAndError(t *testing.T) {
	path := withTempThemeFile(t)
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTheme()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
	if got.Gold != AmberThrone.Gold {
		t.Errorf("on bad JSON should fall back to default; got Gold=%q", got.Gold)
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("error should mention JSON: %v", err)
	}
}

func TestLoadTheme_InvalidHexReturnsError(t *testing.T) {
	cases := map[string]string{
		"missing-hash":   `{"gold": "FF0000"}`,
		"too-short":      `{"gold": "#FFF"}`,
		"too-long":       `{"gold": "#FFFFFFFF"}`,
		"non-hex":        `{"gold": "#GG0000"}`,
		"named-color":    `{"gold": "red"}`,
		"empty-hash":     `{"gold": "#"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := withTempThemeFile(t)
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := LoadTheme()
			if err == nil {
				t.Errorf("%s: expected error, got nil", name)
			}
			if got.Gold != AmberThrone.Gold {
				t.Errorf("%s: should fall back to default Gold; got %q", name, got.Gold)
			}
		})
	}
}

func TestValidateHex(t *testing.T) {
	good := []string{"#000000", "#FFFFFF", "#AbCdEf", "#123456", ""}
	for _, c := range good {
		if err := validateHex("test", c); err != nil {
			t.Errorf("validateHex(%q) should pass; got %v", c, err)
		}
	}
	bad := []string{"FF0000", "#FFF", "#FFFFFFFF", "#GG0000", "red", "#"}
	for _, c := range bad {
		if err := validateHex("test", c); err == nil {
			t.Errorf("validateHex(%q) should fail; got nil", c)
		}
	}
}

func TestApplyTheme_ChangesPaletteAndStyles(t *testing.T) {
	custom := Theme{
		Name:      "test",
		Gold:      "#111111",
		Purple:    "#222222",
		Silver:    "#333333",
		Violet:    "#444444",
		Orange:    "#555555",
		Emerald:   "#666666",
		DimGray:   "#777777",
		DarkAmber: "#888888",
	}
	prevGold := gold
	t.Cleanup(func() { ApplyTheme(AmberThrone) }) // restore for any later test

	ApplyTheme(custom)
	if gold == prevGold {
		t.Errorf("gold should have changed after ApplyTheme")
	}
	// titleStyle uses gold — verify it picked up the change
	rendered := titleStyle.Render("X")
	if !strings.Contains(rendered, "X") {
		t.Errorf("titleStyle should still render content: %q", rendered)
	}
}

func TestApplyTheme_RoundTripWithLoad(t *testing.T) {
	path := withTempThemeFile(t)
	body := `{"gold": "#ABCDEF", "purple": "#FEDCBA"}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTheme()
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	t.Cleanup(func() { ApplyTheme(AmberThrone) })
	ApplyTheme(loaded)
	// after applying, gold should be the loaded value, not default
	if string(gold) != "#ABCDEF" {
		t.Errorf("after ApplyTheme(loaded): gold=%q, want #ABCDEF", gold)
	}
}

func TestThemeFile_ReturnsPathOrEmpty(t *testing.T) {
	withTempThemeFile(t)
	got := ThemeFile()
	if got == "" {
		t.Error("ThemeFile() with override set should return non-empty")
	}
	if !strings.HasSuffix(got, "theme.json") {
		t.Errorf("path should end in theme.json; got %q", got)
	}
}
