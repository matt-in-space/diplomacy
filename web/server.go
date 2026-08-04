package web

import (
	"io/fs"
	"net/http"
)

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embedded at build time; a failure here is a build bug
	}

	mux.HandleFunc("GET /", handleHome)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticRoot)))

	return mux
}
