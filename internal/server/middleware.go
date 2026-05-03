package server

import (
	"context"
	"fmt"
	"net/http"
)

type ctxKey struct{}

var userIDKey ctxKey

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err != nil || c.Value == "" {
			s.render401(w)
			return
		}
		uid, err := s.store.UserBySessionToken(c.Value)
		if err != nil || uid == 0 {
			s.render401(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, uid)))
	})
}

func userIDFrom(r *http.Request) int64 {
	v, _ := r.Context().Value(userIDKey).(int64)
	return v
}

func (s *Server) render401(w http.ResponseWriter) {
	renderErrorPage(w, http.StatusUnauthorized, "Not signed in", "You need to be signed in to view this page.")
}

func renderErrorPage(w http.ResponseWriter, status int, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<link rel="stylesheet" href="/static/style.css">
</head><body class="page-landing"><main class="landing">
<h1>%s</h1><p>%s</p>
<a class="btn btn-primary" href="/">Back to home</a>
</main></body></html>`, title, title, body)
}
