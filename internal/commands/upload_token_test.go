package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
extOrDefault must not let user-supplied filenames inject path separators
or traversal into the on-disk filename.

When:

	A user uploads a file whose name contains slashes or "..".

Expected:

	The returned extension is bounded- at minimum, no separators. Currently
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
