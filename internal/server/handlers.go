package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Mark-C-Hall/benefitshow/internal/ballot"
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

type votePageData struct {
	Locked bool
	Songs  []songs.Song
	Picks  []songs.Song
}

func (s *Server) handleVoteGet(w http.ResponseWriter, r *http.Request) {
	voted, err := s.store.HasVoted(userIDFrom(r))
	if err != nil {
		log.Printf("handleVoteGet: has voted: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if voted {
		ranks, err := s.store.GetOnlineBallot(userIDFrom(r))
		if err != nil {
			log.Printf("handleVoteGet: get ballot: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		picks := make([]songs.Song, 5)
		for i, id := range ranks {
			if song := songs.ByID(id); song != nil {
				picks[i] = *song
			}
		}
		s.render(w, "vote.html", votePageData{Locked: true, Picks: picks})
		return
	}
	s.render(w, "vote.html", votePageData{Locked: false, Songs: songs.All})
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
	if len(p.Ranks) != 5 {
		http.Error(w, "exactly 5 songs required", http.StatusBadRequest)
		return
	}
	seen := make(map[int]bool, 5)
	for _, id := range p.Ranks {
		if songs.ByID(id) == nil {
			http.Error(w, "invalid song id", http.StatusBadRequest)
			return
		}
		if seen[id] {
			http.Error(w, "duplicate song id", http.StatusBadRequest)
			return
		}
		seen[id] = true
	}

	var ranks [5]int
	copy(ranks[:], p.Ranks)

	if err := s.store.CreateOnlineBallot(userIDFrom(r), ranks); err != nil {
		if errors.Is(err, ballot.ErrAlreadyVoted) {
			http.Error(w, "already voted", http.StatusConflict)
			return
		}
		log.Printf("handleVotePost: create ballot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resultRow struct {
	Rank   int
	Title  string
	Artist string
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	s.render(w, "results.html", struct{ Results []resultRow }{})
}
