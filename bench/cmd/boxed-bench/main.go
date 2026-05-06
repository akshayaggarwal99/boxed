package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/akshayaggarwal99/boxed/bench/scenarios"
)

func main() {
	scenario := flag.String("scenario", "", "coldstart|throughput|overhead|agent")
	out := flag.String("out", "results", "output dir for CSVs")
	n := flag.Int("n", 1000, "iterations")
	conc := flag.Int("conc", 1, "concurrency")
	endpoint := flag.String("endpoint", "http://127.0.0.1:8080", "Boxed control plane URL")
	apiKey := flag.String("api-key", os.Getenv("BOXED_API_KEY"), "API key")
	flag.Parse()

	if *scenario == "" {
		log.Fatal("--scenario required (coldstart|throughput|overhead|agent)")
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	c := scenarios.Client{Endpoint: *endpoint, APIKey: *apiKey}

	switch *scenario {
	case "coldstart":
		scenarios.RunColdStart(c, *n, *out)
	case "throughput":
		scenarios.RunThroughput(c, *n, *conc, *out)
	case "overhead":
		scenarios.RunOverhead(c, *n, *out)
	case "agent":
		scenarios.RunAgentTrace(c, *n, *out)
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenario)
		os.Exit(2)
	}
}
