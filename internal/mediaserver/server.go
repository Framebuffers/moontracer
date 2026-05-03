package mediaserver

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
Serve starts a read-only HTTP file server rooted at dataDir, listening on addr.
Files are accessible under the /api/v1/cdn/ prefix:

	GET /api/v1/cdn/[guild_id]/assets/campaigns/[campaign_id]/cover/[file]

This server runs in a background goroutine; does not block the Dispatcher or other goroutines.
*/
func Serve(dataDir, addr string) {
	fs := http.FileServer(http.Dir(dataDir))
	mux := http.NewServeMux()
	mux.Handle("/api/v1/cdn/", http.StripPrefix("/api/v1/cdn/", fs))

	go func() {
		log.Printf("mediaserver: listening on %s: /api/v1/cdn/", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("mediaserver: %v", err)
		}
	}()
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
	log.Printf("mediaserver: probe failed — CDN may not be reachable: %v", lastErr)
}
