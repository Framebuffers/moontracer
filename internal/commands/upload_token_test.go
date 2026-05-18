package commands

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
parseHexColor coverage.

The function gates a color.RGBA value that is later passed straight into
the gradient generator. It currently accepts:

  - 6 hex chars, with or without leading "#"
  - Any case

It rejects everything else. These tests pin behavior so a future change
(e.g. accepting #RGB shorthand or named colors) is intentional.
*/
func TestParseHexColor(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    color.RGBA
		wantErr bool
	}{
		{"with hash", "#ff0000", color.RGBA{R: 255, A: 255}, false},
		{"without hash", "00ff00", color.RGBA{G: 255, A: 255}, false},
		{"uppercase", "#0000FF", color.RGBA{B: 255, A: 255}, false},
		{"empty", "", color.RGBA{}, true},
		{"only hash", "#", color.RGBA{}, true},
		{"too short", "#abc", color.RGBA{}, true},
		{"too long", "#aabbccdd", color.RGBA{}, true},
		{"non-hex chars", "#zz0000", color.RGBA{}, true},
		{"whitespace", "  ff0000  ", color.RGBA{}, true},
		{"injection-ish", "#ff0000;rm -rf /", color.RGBA{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseHexColor(c.in)
			if c.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

/*
extOrDefault must not let user-supplied filenames inject path separators
or traversal into the on-disk filename.

When:

	A user uploads a file whose name contains slashes or "..".

Expected:

	The returned extension is bounded - at minimum, no separators. Currently
	uses filepath.Ext which only returns text after the last dot, so most
	traversal payloads collapse safely; but ".png/../etc" -> "" -> default,
	and "foo.png\x00bad" returns ".png\x00bad" on Linux. Pin behavior.
*/
func TestExtOrDefault(t *testing.T) {
	cases := []struct {
		name, filename, def, want string
	}{
		{"plain", "cat.png", ".jpg", ".png"},
		{"no ext uses default", "cat", ".jpg", ".jpg"},
		{"trailing dot", "cat.", ".jpg", "."},
		{"hidden file", ".gitignore", ".jpg", ".gitignore"},
		{"traversal in name", "../etc/passwd", ".jpg", ".jpg"},
		{"double ext keeps last", "cat.png.exe", ".jpg", ".exe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extOrDefault(c.filename, c.def)
			assert.Equal(t, c.want, got, "input %q", c.filename)
			assert.NotContains(t, got, "/", "extension must not contain a path separator")
			assert.NotContains(t, got, "\\", "extension must not contain a backslash")
		})
	}
}
