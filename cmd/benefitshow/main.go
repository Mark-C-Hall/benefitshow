package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Mark-C-Hall/benefitshow/internal/ballot"
	"github.com/Mark-C-Hall/benefitshow/internal/config"
	"github.com/Mark-C-Hall/benefitshow/internal/server"
	"github.com/Mark-C-Hall/benefitshow/internal/songs"
	"github.com/Mark-C-Hall/benefitshow/internal/stv"
)

const seats = 10

const usage = `usage: benefitshow {serve|import|tally}

  serve            run the web server
  import <path>    import paper ballots from a 5-column CSV
  tally            run the STV tally and persist the top ` + "10"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		runServe()
	case "import":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "import: missing CSV path")
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(2)
		}
		runImport(os.Args[2])
	case "tally":
		runTally()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func runServe() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	store, err := ballot.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	handler, err := server.New(cfg, store)
	if err != nil {
		log.Fatalf("error building server: %v", err)
	}

	addr := cfg.ListenAddr()
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func runImport(path string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	rows, err := readBallotCSV(path)
	if err != nil {
		log.Fatalf("import: %v", err)
	}
	if len(rows) == 0 {
		log.Fatalf("import: %s contained no ballot rows", path)
	}

	store, err := ballot.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	if err := store.ImportPaperBallots(rows); err != nil {
		log.Fatalf("import: %v", err)
	}
	fmt.Printf("imported %d paper ballot(s) from %s\n", len(rows), path)
}

func readBallotCSV(path string) ([][5]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = 5

	var rows [][5]int
	lineNum := 0
	for {
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		lineNum++

		row, parseErr := parseBallotRow(rec)
		if parseErr != nil {
			if lineNum == 1 {
				// First row failed validation: treat as a header and skip.
				continue
			}
			return nil, fmt.Errorf("row %d: %w", lineNum, parseErr)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseBallotRow(rec []string) ([5]int, error) {
	var row [5]int
	seen := make(map[int]bool, 5)
	for i, field := range rec {
		id, err := strconv.Atoi(field)
		if err != nil {
			return row, fmt.Errorf("column %d: %q is not an integer", i+1, field)
		}
		if songs.ByID(id) == nil {
			return row, fmt.Errorf("column %d: song id %d is not in the pool", i+1, id)
		}
		if seen[id] {
			return row, fmt.Errorf("column %d: song id %d is duplicated on this ballot", i+1, id)
		}
		seen[id] = true
		row[i] = id
	}
	return row, nil
}

func runTally() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	store, err := ballot.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	ballots, err := store.GetAllBallots()
	if err != nil {
		log.Fatalf("tally: %v", err)
	}
	if len(ballots) == 0 {
		log.Fatalf("tally: no ballots in database")
	}

	ordering := stv.Run(ballots, seats)
	if err := store.SaveTallyResults(ordering); err != nil {
		log.Fatalf("tally: %v", err)
	}

	fmt.Printf("counted %d ballot(s)\n\n", len(ballots))
	fmt.Println("Top 10:")
	for i, id := range ordering {
		if i >= seats {
			break
		}
		song := songs.ByID(id)
		if song == nil {
			fmt.Printf(" %2d. (unknown song id %d)\n", i+1, id)
			continue
		}
		fmt.Printf(" %2d. %s — %s\n", i+1, song.Title, song.Artist)
	}
}
