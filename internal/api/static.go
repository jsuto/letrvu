package api

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:static
var staticFiles embed.FS

// spaHandler serves the embedded frontend. Requests for unknown paths fall
// back to index.html so the Vue router can handle client-side navigation.
func spaHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the file path the same way http.FileServer would.
		p := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if p == "." {
			p = "index.html"
		}
		if _, err := sub.Open(p); err != nil {
			// File not found — serve index.html for SPA client-side routing.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
