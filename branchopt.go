package gitsync

import (
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Branch returns an option that sets the given branch name.
func Branch(name string) Option {
	return func(w *Worker) {
		w.branch = plumbing.ReferenceName("refs/heads/" + name)
	}
}
