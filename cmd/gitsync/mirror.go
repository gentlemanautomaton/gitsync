package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gentlemanautomaton/gitsync"
)

func mirror(command string, args []string) {
	var (
		repo   string
		origin string
		branch string
	)

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&repo, "repo", "", "path of directory to sync")
	fs.StringVar(&origin, "origin", "", "URL of origin repository")
	fs.StringVar(&branch, "branch", "master", "branch to sync with")
	fs.Parse(args)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s -repo <path> -origin <url> [-branch <branch>]\n", os.Args[0], command)
		fs.PrintDefaults()
	}

	usage := func(message string) {
		fmt.Printf("%s\n\n", message)
		fs.Usage()
		os.Exit(2)
	}

	if repo == "" {
		usage("No repository specified.")
	}

	if origin == "" {
		usage("No origin specified.")
	}

	if branch == "" {
		usage("No branch specified.")
	}

	err := gitsync.Pull(context.Background(), repo, origin, branch, os.Stdout)
	if err != nil {
		abort(err)
	}
}
