package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	var (
		base     = flag.String("base", "", "Redfish base URL (e.g. https://192.168.2.144)")
		user     = flag.String("user", "ADMIN", "BMC username")
		pass     = flag.String("pass", "ADMIN", "BMC password")
		entry    = flag.String("entry", "/redfish/v1/", "entry point path to start crawling")
		minSleep = flag.Duration("min-sleep", 1*time.Second, "minimum sleep between requests")
		maxSleep = flag.Duration("max-sleep", 5*time.Second, "maximum sleep between requests")
		timeout  = flag.Duration("timeout", 30*time.Second, "per-request timeout")
		maxDepth = flag.Int("max-depth", 0, "maximum crawl depth (0 = unlimited)")
	)
	flag.Parse()

	if *base == "" {
		fmt.Fprintln(os.Stderr, "error: -base is required")
		flag.Usage()
		os.Exit(1)
	}
	if *minSleep > *maxSleep {
		fmt.Fprintln(os.Stderr, "error: -min-sleep must be <= -max-sleep")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c := newClient(*base, *user, *pass, *timeout, *minSleep, *maxSleep)

	cr := newCrawler(c, *maxDepth)
	cr.run(ctx, *entry)

	if err := writeResults(cr.results); err != nil {
		fmt.Fprintf(os.Stderr, "error: write results: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "done: %d resources fetched (%d errors)\n",
		len(cr.results), cr.errorCount)
}
