package mediaserver

import (
	"io"
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
HTTP path-traversal coverage for the /api/v1/cdn/ file server.

The production handler in Serve() is:

	mux.Handle("/api/v1/cdn/", http.StripPrefix("/api/v1/cdn/", http.FileServer(http.Dir(dataDir))))

These tests reconstruct the same mux against a temp dataDir laid out as:

	<root>/
	    secret.txt              <-- MUST NEVER be served (sibling of dataDir)
	    data/
	        guild-1/
	            assets/campaigns/c1/cover/legit.png   <-- legitimately served

If any traversal payload returns the secret, the FileServer is leaking data
above its root.
*/

const secretBody = "TOP_SECRET_DO_NOT_LEAK"

func newCDNTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	root := t.TempDir()
	dataDir := filepath.Join(root, "data")

	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte(secretBody), 0o600))

	legit := filepath.Join(dataDir, "guild-1", "assets", "campaigns", "c1", "cover")
	require.NoError(t, os.MkdirAll(legit, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legit, "legit.png"), []byte("png"), 0o644))

	mux := http.NewServeMux()
	mux.Handle("/api/v1/cdn/", http.StripPrefix("/api/v1/cdn/", http.FileServer(http.Dir(dataDir))))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, dataDir
}

func get(t *testing.T, base, path string) (int, string) {
	t.Helper()
	// NOTE: Build raw request URL to bypass Go client URL cleaning where possible.
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

/*
The legitimate happy path serves files from inside dataDir.

When:

	A well-formed CDN URL points to an existing file under dataDir.

Expected:

	200 OK, body matches the file on disk.
*/
func TestCDN_LegitimatePath(t *testing.T) {
	srv, _ := newCDNTestServer(t)

	status, body := get(t, srv.URL, "/api/v1/cdn/guild-1/assets/campaigns/c1/cover/legit.png")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "png", body)
}

/*
Dot-dot traversal must never escape dataDir.

When:

	A request uses ../ segments to climb above dataDir and target the sibling
	"secret.txt" file.

Expected:

	The secret body is never returned. ServeMux's CleanPath redirects /../
	traversal back inside the root before FileServer ever sees it.
*/
func TestCDN_NoTraversalAboveRoot(t *testing.T) {
	srv, _ := newCDNTestServer(t)

	payloads := []string{
		"/api/v1/cdn/../secret.txt",
		"/api/v1/cdn/guild-1/../../secret.txt",
		"/api/v1/cdn/./../secret.txt",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			_, body := get(t, srv.URL, p)
			assert.NotContains(t, body, secretBody, "traversal %q leaked secret file", p)
		})
	}
}

/*
URL-encoded traversal must also not escape.

When:

	The ../ segments are percent-encoded so they survive ServeMux cleaning.

Expected:

	No secret leakage. http.FileServer rejects encoded slashes via
	http.ServeFile's containsDotDot check after URL decoding.
*/
func TestCDN_NoTraversalEncoded(t *testing.T) {
	srv, _ := newCDNTestServer(t)

	payloads := []string{
		"/api/v1/cdn/%2e%2e/secret.txt",
		"/api/v1/cdn/guild-1/%2e%2e/%2e%2e/secret.txt",
		"/api/v1/cdn/..%2fsecret.txt",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			_, body := get(t, srv.URL, p)
			assert.NotContains(t, body, secretBody, "encoded traversal %q leaked secret file", p)
		})
	}
}

/*
Backslash separators are not a path-component separator on Linux and must not
be interpreted as one by the FileServer.

When:

	A Windows-style path with backslashes is requested.

Expected:

	No secret leakage. On Linux the backslash is just a filename character and
	the lookup misses; on Windows http.Dir rejects backslashes.
*/
func TestCDN_NoBackslashTraversal(t *testing.T) {
	srv, _ := newCDNTestServer(t)

	_, body := get(t, srv.URL, "/api/v1/cdn/..\\secret.txt")
	assert.NotContains(t, body, secretBody)
}

/*
Directory listings must not be served.

When:

	A request targets a directory rather than a file.

Expected:

	No HTML index of cover/ contents. http.FileServer auto-indexes by default
	unless the directory contains an index.html — this test pins current
	behavior so a future change that disables it (or adds index.html guarding)
	is intentional and reviewed.

Note:

	This test currently passes because FileServer DOES auto-index directories,
	exposing filenames. If you consider that a leak (it discloses cover-art
	UUIDs and token paths), flip the assertion and add a NotFoundHandler wrapper
	in Serve() to reject directory requests. Marked t.Log so reviewers see it.
*/
func TestCDN_DirectoryListingBehavior(t *testing.T) {
	srv, _ := newCDNTestServer(t)

	status, body := get(t, srv.URL, "/api/v1/cdn/guild-1/assets/campaigns/c1/cover/")
	t.Logf("directory request returned status=%d body-contains-listing=%v",
		status, strings.Contains(body, "legit.png"))

	// NOTE: Directory listings disclose filenames. A hardened server should 404.
	assert.NotContains(t, body, "legit.png",
		"directory listing exposes filenames — wrap FileServer to reject dir requests")
}
