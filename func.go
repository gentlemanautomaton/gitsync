package gitsync

import (
	"context"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/sideband"
)

// Pull creates an ephemeral Synchronizer and performs a Pull with the given
// parameters. See Synchronizer.Pull for details.
func Pull(ctx context.Context, path, origin, branch string, progress sideband.Progress) error {
	return New(path, origin, branch, os.Stdout).Pull(ctx)
}
