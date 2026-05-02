package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Mark-C-Hall/benefitshow/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	addr := cfg.ListenAddr()
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
