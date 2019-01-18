package gitsync

import (
	"context"
)

// CloneOrPull creates an ephemeral Worker and performs a Pull with the given
// parameters. See Worker.CloneOrPull for details.
func CloneOrPull(ctx context.Context, path, origin string, options ...Option) error {
	return New(path, origin, options...).CloneOrPull(ctx)
}
