// Package songs holds the static candidate pool for the election.
//
// The pool will grow to 30+ entries before launch; the current set is
// placeholder data to develop against.
package songs

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Song struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	YouTubeURL string `json:"youtube"`
	SpotifyURL string `json:"spotify"`
}

//go:embed songs.json
var raw []byte

var All []Song
var byID map[int]*Song

func init() {
	if err := json.Unmarshal(raw, &All); err != nil {
		panic(fmt.Errorf("songs: parsing embedded songs.json: %w", err))
	}
	byID = make(map[int]*Song, len(All))
	for i := range All {
		byID[All[i].ID] = &All[i]
	}
}

// ByID returns the Song with the given ID, or nil if not found.
func ByID(id int) *Song {
	return byID[id]
}
