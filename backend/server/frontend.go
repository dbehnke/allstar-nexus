package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// FileServer returns a simple file server for a subdirectory inside the embedded FS.
func FileServer(embeddedFS embed.FS, path string) (http.Handler, error) {
	sub, err := fs.Sub(embeddedFS, path)
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}

// SPAFileServer serves static assets from an embedded FS and falls back to index.html
// for any non-existent path, enabling client-side routing (Vue Router history mode).
func SPAFileServer(embeddedFS embed.FS, basePath string) (http.Handler, error) {
	sub, err := fs.Sub(embeddedFS, basePath)
	if err != nil {
		return nil, err
	}
	static := http.FS(sub)
	fileServer := http.FileServer(static)

	// Read index.html once so we can serve it directly on SPA fallbacks
	indexBytes, readErr := fs.ReadFile(sub, "index.html")
	if readErr != nil {
		return nil, readErr
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let API and WS routes be handled by other mux routes (they are more specific than "/").
		// This handler is only mounted at "/" as a catch-all for the SPA paths.

		// Try to serve the requested file; if it doesn't exist, rewrite to "/" to serve index.html.
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			// Serve index.html directly at root
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexBytes)
			return
		}

		// Attempt to open the requested path from the embedded FS
		if f, err := sub.Open(reqPath); err == nil {
			// If it's a file (not a directory), delegate to the file server
			if fi, statErr := f.Stat(); statErr == nil && fi != nil && !fi.IsDir() {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			_ = f.Close()
		}

		// Fallback: serve index.html directly to enable client-side routing
		// Avoid relying on FileServer directory logic which may 404 for unknown paths.
		// Preserve query/hash for client-side router in the browser.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Defensive: ensure consistent cache behavior for index.html
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
	}), nil
}
