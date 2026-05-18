package mediaserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Coverage for the Download() helper.

Download() is invoked from /uploadtoken and /uploadcover with a Discord CDN
URL the user supplied via attachment. Discord vouches for those URLs, BUT:

	- Discord could serve a redirect to anywhere (Download follows redirects).
	- Discord CDN content is whatever the user uploaded - Discord does not
	  validate that an attachment with ContentType "image/png" is actually a
	  PNG. The bot must validate the magic bytes itself.
	- A pathological response (huge body, slow drip) can exhaust disk and
	  starve the 30s timeout.

These tests document the current behavior so a fix PR can flip them green.
*/

/*
Download writes the response body to disk.

When:

	A trivial server returns a small image-like body.

Expected:

	The file lands at diskPath, the mimeType is detected, no error.
*/
func TestDownload_HappyPath(t *testing.T) {
	body := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 64))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	dst := filepath.Join(t.TempDir(), "out.png")
	mime, err := Download(srv.URL+"/x", dst)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

/*
Download must reject non-image content.

When:

	The remote server returns a text/html or application/octet-stream body
	(e.g. a redirected phishing page or an arbitrary binary).

Expected:

	No file is written and an error is returned.

Development Note (v0.12.6, 20260511):

	Currently FAILS:
		Download detects the MIME but never rejects on it, so the bot would persist HTML
		or arbitrary binaries as if they were images and serve them back via
		the CDN.
*/
func TestDownload_RejectsNonImageMIME(t *testing.T) {
	htmlBody := []byte("<html><script>alert(1)</script></html>" + strings.Repeat(" ", 600))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(htmlBody)
	}))
	t.Cleanup(srv.Close)

	dst := filepath.Join(t.TempDir(), "evil.png")
	_, err := Download(srv.URL+"/x", dst)

	assert.Error(t, err, "Download must reject non-image MIME types")
	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr),
		"non-image content was persisted to disk at %s - Download lacks MIME enforcement", dst)
}

/*
Download must cap response size.

When:

	A malicious or misconfigured server streams a huge body.

Expected:

	Download stops at a sensible cap (e.g. 16 MiB matching the upload limit)
	rather than filling the disk.
	This test uses a 64 KiB cap target via a 100 KiB response to keep the
	test fast; flip the cap when adding the real guard.

Development Note (v0.12.6, 20260511):

	Currently FAILS - io.Copy is unbounded.
*/
func TestDownload_BoundsResponseSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// PNG magic prefix so MIME detection passes.
		w.Write([]byte("\x89PNG\r\n\x1a\n"))
		w.Write(make([]byte, MaxDownloadBytes+1024))
	}))
	t.Cleanup(srv.Close)

	dst := filepath.Join(t.TempDir(), "big.png")
	_, err := Download(srv.URL+"/x", dst)

	assert.Error(t, err, "Download must reject responses larger than MaxDownloadBytes")
	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr),
		"oversize body was persisted to disk at %s", dst)
}

/*
TokenPath must not allow callers to escape the player's token directory.

When:

	A caller (today: only trusted internal code, but future: any handler
	pulling suffix from user input) passes a suffix containing a slash or
	traversal sequence.

Expected:

	The returned disk path stays inside dataDir/<guildID>/tokens/<playerID>/.

Development Note (v0.12.6, 20260511):

	Currently FAILS:
		TokenPath uses filepath.Join with no validation, so
		"../../etc/passwd" as suffix climbs out of the player directory.
*/
func TestTokenPath_RejectsTraversalSuffix(t *testing.T) {
	dataDir := "/var/data"
	expectedPrefix := filepath.Join(dataDir, "guild-1", "tokens", "player-1") + string(filepath.Separator)

	cases := []struct {
		name, suffix, ext string
	}{
		{"dotdot suffix", "../../etc/passwd", ""},
		{"slash suffix", "src/../../../escape", ".png"},
		{"nul-ish ext", "out", "/../../escape"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			disk, _ := TokenPath(dataDir, "http://cdn", "guild-1", "player-1", c.suffix, c.ext)
			assert.True(t, strings.HasPrefix(disk, expectedPrefix),
				"suffix %q ext %q escaped the player dir: got %q (want prefix %q)",
				c.suffix, c.ext, disk, expectedPrefix)
		})
	}
}
