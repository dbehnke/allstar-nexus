package server

import (
	"embed"
	"io/fs"
	"net/http"
)

func FileServer(embeddedFS embed.FS, path string) (http.Handler, error) {
	sub, err := fs.Sub(embeddedFS, path)
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}
