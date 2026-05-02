package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Mark-C-Hall/benefitshow/internal/config"
	"github.com/Mark-C-Hall/benefitshow/internal/server"
)

const usage = `usage: benefitshow {serve|import|tally}

  serve            run the web server
  import <path>    import paper ballots from CSV (not yet implemented)
  tally            run the STV tally (not yet implemented)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		runServe()
	case "import":
		fmt.Fprintln(os.Stderr, "import: not yet implemented")
		os.Exit(1)
	case "tally":
		fmt.Fprintln(os.Stderr, "tally: not yet implemented")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func runServe() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	handler, err := server.New(cfg)
	if err != nil {
		log.Fatalf("error building server: %v", err)
	}

	addr := cfg.ListenAddr()
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
