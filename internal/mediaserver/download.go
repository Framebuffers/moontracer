package mediaserver

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

/*
One-time download claim store.

Provides opaque, short-lived download URLs for user-generated files (e.g. player tokens)
without exposing the server's internal directory structure.

Flow (registration):
 1. Caller invokes Register(diskPath, name) after saving a file.
 2. A UUID claim token is generated and stored in globalClaims with a 10-minute TTL.
 3. Register returns a public URL of the form {downloadBase}/dl/{token}.

Flow (serving):
 1. User clicks the download link; browser GETs /dl/{token}.
 2. dlHandler (registered in server.go) calls globalClaims.pop(token).
    - If the token is unknown or expired: 404.
    - If valid: removes the entry (one-time use) and streams the file.
 3. Response includes Content-Disposition: attachment; filename="{sanitized name}.png"
    so the browser prompts a save dialog with the character's name, not a UUID.

Sweep:
 A background goroutine (started by Serve via startSweep) evicts expired entries
 every minute so the map does not grow unbounded even if claims are never redeemed.
*/

const claimTTL = 10 * time.Minute

/*
downloadBase is the public root URL used to build /dl/ links (e.g. "https://media.example.com").

Set via SetDownloadBase before Register is called.
*/
var downloadBase string

// SetDownloadBase sets the public server root URL for building download links.
func SetDownloadBase(url string) {
	downloadBase = strings.TrimRight(url, "/")
}

/*
Register creates a one-time opaque download claim for diskPath and returns its URL.

The claim expires after 10 minutes or on first use, whichever comes first.
*/
func Register(diskPath, name string) string {
	token := uuid.NewString()
	globalClaims.store(token, diskPath, name)
	return fmt.Sprintf("%s/dl/%s", downloadBase, token)
}

/*
	Claim Store
*/

type claimEntry struct {
	diskPath  string
	name      string
	expiresAt time.Time
}

type claimStore struct {
	mu sync.Mutex
	m  map[string]claimEntry
}

var globalClaims = &claimStore{m: make(map[string]claimEntry)}

func (s *claimStore) store(token, diskPath, name string) {
	s.mu.Lock()
	s.m[token] = claimEntry{diskPath: diskPath, name: name, expiresAt: time.Now().Add(claimTTL)}
	s.mu.Unlock()
}

func (s *claimStore) pop(token string) (claimEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[token]
	if !ok || time.Now().After(e.expiresAt) {
		delete(s.m, token)
		return claimEntry{}, false
	}
	delete(s.m, token)
	return e, true
}

func (s *claimStore) sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, e := range s.m {
		if now.After(e.expiresAt) {
			delete(s.m, token)
		}
	}
}

func startSweep() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			globalClaims.sweep()
		}
	}()
}

/*
dlHandler serves a claimed file exactly once, then invalidates the token.

URL: /dl/{token}
*/
func dlHandler(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/")

	// INFO: Reject anything that looks like a path traversal.
	if token == "" || strings.ContainsAny(token, "/\\") {
		http.NotFound(w, r)
		return
	}

	entry, ok := globalClaims.pop(token)
	if !ok {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(entry.diskPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	safe := sanitizeFilename(entry.name) + ".png"
	w.Header().Set("Content-Disposition", `attachment; filename="`+safe+`"`)
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, safe, fi.ModTime(), f)
}

/*
sanitizeFilename strips path separators and characters illegal in HTTP
Content-Disposition filenames, then truncates to 80 characters.
*/
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '/', '\n', '\r', '\t':
			return '_'
		}
		return r
	}, name)
	if len(name) > 80 {
		name = name[:80]
	}
	if name == "" {
		name = "token"
	}
	return name
}
