package gitsync

import (
	"context"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/sideband"
)

// Sync creates an ephemeral Synchronizer and performs a Sync with the given
// parameters. See Synchronizer.Sync for details.
func Sync(ctx context.Context, path, origin, branch string, progress sideband.Progress) error {
	return New(path, origin, branch, os.Stdout).Sync(ctx)
}
