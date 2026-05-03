package mediaserver

import (
	"log"
	"net/http"
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
		log.Printf("mediaserver: listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("mediaserver: %v", err)
		}
	}()
}
