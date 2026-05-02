package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Mark-C-Hall/benefitshow/internal/songs"
)

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	t, ok := s.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	s.render(w, "landing.html", nil)
}

func (s *Server) handleVoteGet(w http.ResponseWriter, r *http.Request) {
	s.render(w, "vote.html", struct{ Songs []songs.Song }{Songs: songs.All})
}

type ballotPayload struct {
	Ranks []int `json:"ranks"`
}

func (s *Server) handleVotePost(w http.ResponseWriter, r *http.Request) {
	var p ballotPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	log.Printf("ballot received (M1 stub, not persisted): ranks=%v", p.Ranks)
	w.WriteHeader(http.StatusOK)
}

type resultRow struct {
	Rank   int
	Title  string
	Artist string
}

// mockResults returns a placeholder ranking so the /results template path is
// exercised before M4 wires the real STV output.
func mockResults() []resultRow {
	out := make([]resultRow, 0, 10)
	for i, s := range songs.All {
		if i >= 10 {
			break
		}
		out = append(out, resultRow{Rank: i + 1, Title: s.Title, Artist: s.Artist})
	}
	return out
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	s.render(w, "results.html", struct{ Results []resultRow }{Results: mockResults()})
}
