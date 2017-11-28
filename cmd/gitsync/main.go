package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gentlemanautomaton/gitsync"
)

func usage(message string) {
	fmt.Printf("%s\n\n", message)
	flag.Usage()
	os.Exit(2)
}

func abort(err error) {
	fmt.Printf("%v\n", err)
	os.Exit(2)
}

func main() {
	var (
		repo   string
		origin string
		branch string
	)

	flag.StringVar(&repo, "repo", "", "path of directory to sync")
	flag.StringVar(&origin, "origin", "", "URL of origin repository")
	flag.StringVar(&branch, "branch", "master", "branch to sync with")
	flag.Parse()

	if repo == "" {
		usage("No repository specified.")
	}

	if origin == "" {
		usage("No origin specified.")
	}

	if branch == "" {
		usage("No branch specified.")
	}

	err := gitsync.Sync(context.Background(), repo, origin, branch, os.Stdout)
	if err != nil {
		abort(err)
	}
}
