// Package server wires the HTTP routes for the voting app: it parses the
// embedded templates once at startup and returns an http.Handler that the
// main package mounts on its listener.
package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/Mark-C-Hall/benefitshow/internal/config"
	"github.com/Mark-C-Hall/benefitshow/web"
)

type Server struct {
	cfg       *config.Config
	templates map[string]*template.Template
}

func New(cfg *config.Config) (http.Handler, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	s := &Server{cfg: cfg, templates: templates}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleLanding)
	mux.HandleFunc("GET /vote", s.handleVoteGet)
	mux.HandleFunc("POST /vote", s.handleVotePost)
	mux.HandleFunc("GET /results", s.handleResults)

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		return nil, fmt.Errorf("subFS for static: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return logging(mux), nil
}

func parseTemplates() (map[string]*template.Template, error) {
	pages := []string{"landing.html", "vote.html", "results.html"}
	out := make(map[string]*template.Template, len(pages))
	for _, name := range pages {
		t, err := template.ParseFS(web.Templates, "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[name] = t
	}
	return out, nil
}

// statusRecorder captures the response status so the logging middleware can
// report it. Defaults to 200 for handlers that never call WriteHeader.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}
