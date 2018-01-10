package main

import (
	"fmt"
	"os"
	"strings"
)

func abort(err error) {
	fmt.Printf("%v\n", err)
	os.Exit(2)
}

func main() {
	usage := func(message string) {
		fmt.Fprintf(os.Stderr,
			"%s\n\n"+
				"usage: %s <command>\n"+
				"       %s mirror -repo <path> -origin <url> [-branch <branch>]\n",
			message, os.Args[0], os.Args[0])
		os.Exit(2)
	}

	if len(os.Args) < 2 {
		usage("No command specified.")
	}

	command := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch command {
	case "mirror":
		mirror(command, args)
	//case "update":
	//	update(command, args)
	default:
		usage(fmt.Sprintf("\"%s\" is an unrecognized command.", command))
	}
}
