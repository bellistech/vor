package registry

import (
	"testing"
	"testing/fstest"
)

// Two sheets share the name "saml". Get can return only one of them;
// GetIn names the category, so each is reachable.
func twinFS() fstest.MapFS {
	return fstest.MapFS{
		"auth/saml.md":     {Data: []byte("# SAML (auth)\n\nThe single sign-on sheet, the longer of the two.\n")},
		"security/saml.md": {Data: []byte("# SAML (security)\n")},
		"shell/bash.md":    {Data: []byte("# Bash\n")},
	}
}

func TestGetIn(t *testing.T) {
	reg, err := New(twinFS())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		category, name string
		want           string // "" = nil; else the category of the sheet returned
	}{
		{"auth", "saml", "auth"},
		{"security", "SAML", "security"},
		{"security", "saml", "security"},
		{"shell", "BASH", "shell"},
		{"nosuch", "saml", ""},
		{"auth", "nosuch", ""},
		{"", "saml", ""},
		{"AUTH", "saml", ""}, // the category is exact
		{"auth", "sam", ""},  // no prefix matching
		{"auth", "", ""},
	}
	for _, c := range cases {
		s := reg.GetIn(c.category, c.name)
		switch {
		case c.want == "" && s != nil:
			t.Errorf("GetIn(%q, %q) = %s/%s, want nil", c.category, c.name, s.Category, s.Name)
		case c.want != "" && s == nil:
			t.Errorf("GetIn(%q, %q) = nil, want a sheet in %q", c.category, c.name, c.want)
		case c.want != "" && (s.Category != c.want || s.Name != "saml" && s.Name != "bash"):
			t.Errorf("GetIn(%q, %q) = %s/%s", c.category, c.name, s.Category, s.Name)
		}
	}
}
