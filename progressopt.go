package gitsync

import "gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/sideband"

// Progress returns an option that sets the progress output.
func Progress(progress sideband.Progress) Option {
	return func(w *Worker) {
		w.progress = progress
	}
}
