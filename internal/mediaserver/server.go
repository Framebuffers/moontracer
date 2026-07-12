package mediaserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedExts = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".avif": {},
}

/*
Serve starts a read-only HTTP file server rooted at dataDir, listening on addr.
Files are accessible under the /api/v1/cdn/ prefix:

	GET /api/v1/cdn/[guild_id]/assets/campaigns/[campaign_id]/cover/[file]

This server runs in a background goroutine; does not block the Dispatcher or other goroutines.
*/
func Serve(mediaDir, addr string) {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/cdn/", http.StripPrefix("/api/v1/cdn/",
		extAllowListHandler(filesOnlyHandler(mediaDir))))

	go func() {
		log.Printf("mediaserver: listening on %s: /api/v1/cdn/", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("mediaserver: %v", err)
		}
	}()
}

func extAllowListHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		if _, ok := allowedExts[ext]; !ok {
			http.NotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

/*
filesOnlyHandler serves files under dataDir but 404s any directory request,
so the FileServer never auto-indexes and discloses filenames.
*/
func filesOnlyHandler(dataDir string) http.Handler {
	fs := http.FileServer(http.Dir(dataDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(dataDir, filepath.FromSlash(r.URL.Path))
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
}

/*
Probe checks that the local CDN endpoint is reachable after Serve() is called.
Retries up to 5 times with 100ms gaps to allow the goroutine to bind.
Logs the result.
*/
func Probe(addr string) {
	url := fmt.Sprintf("http://localhost%s/api/v1/cdn/", addr)
	client := &http.Client{Timeout: 2 * time.Second}

	var lastErr error
	for range 5 {
		time.Sleep(100 * time.Millisecond)
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		log.Printf("mediaserver: reachable (status %d)", resp.StatusCode)
		return
	}
	log.Printf("mediaserver: probe failed- CDN may not be reachable: %v", lastErr)
}
