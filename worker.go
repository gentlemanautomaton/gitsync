package gitsync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/sideband"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// Worker is responsible for keeping local files in sync with a remote.
// It should be created by calling gitsync.New.
type Worker struct {
	path     string
	origin   string
	branch   plumbing.ReferenceName
	progress sideband.Progress
	auth     transport.AuthMethod
}

// New returns a Worker for the repository at the given path.
//
// The path should specify a file system directory to which the contents of
// the remote branch will be mirrored.
//
// The given origin will be used to access the remote.
//
// New is nondestructive. Calls to CloneOrPull will perform file system
// initialization and cloning as needed.
func New(path, origin string, options ...Option) *Worker {
	path, _ = filepath.Abs(path)
	s := &Worker{
		path:   path,
		origin: origin,
	}
	for _, opt := range options {
		opt(s)
	}
	return s
}

// CloneOrPull attempts to update the local file system to match a particular
// branch on the origin. It performs the equivalent of git clone, pull and
// checkout as necessary to accomplish this.
//
// CloneOrPull is destructive. Files within the local copy may be discarded in
// order for it to accomplish its goal. In the case of failure it may attempt to
// destroy the local copy and re-clone.
func (w *Worker) CloneOrPull(ctx context.Context) error {
	start := time.Now()

	repo, head, cloned, err := w.prepare(ctx)
	if err != nil {
		return err
	}

	if cloned {
		w.printf("Sync completed in %s\n", time.Now().Sub(start))
		return nil
	}

	err = w.updateOrigin(repo)
	if err != nil {
		return err
	}

	w.printf("Opening worktree\n")
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to open worktree: %v", err)
	}

	if head.Name() != w.branch {
		w.printf("Switching to %s branch\n", w.branch.Short())
		err = w.updateBranch(repo, worktree)
		if err != nil {
			return err
		}
	} else {
		w.printf("Already on %s branch\n", w.branch.Short())
	}

	w.printf("Pulling from %s\n", w.origin)
	err = worktree.Pull(&git.PullOptions{
		ReferenceName: w.branch,
		Progress:      w.progress,
		Auth:          w.auth,
		Force:         true,
	})
	switch err {
	case nil:
	case git.NoErrAlreadyUpToDate:
	default:
		return fmt.Errorf("unable to pull: %v", err)
	}

	w.printf("Sync completed in %s\n", time.Now().Sub(start))

	return nil
}

func (w *Worker) prepare(ctx context.Context) (repo *git.Repository, head *plumbing.Reference, cloned bool, err error) {
	const attempts = 2
	for i := 0; i < attempts; i++ {
		repo, cloned, err = w.openOrClone(ctx)
		if err != nil {
			return
		}

		head, err = repo.Head()
		if err == nil {
			return
		}

		err = fmt.Errorf("unable to determine repository HEAD reference: %v", err)

		if i < attempts {
			// The initial clone failed somehow, possibly on a previous attempt
			w.printf("The repository appears to be malformed\nAttempting delete and re-clone\n")
			err = w.delete()
			if err != nil {
				err = fmt.Errorf("unable to delete existing malformed repository: %v", err)
				return
			}
		}
	}

	return
}

func (w *Worker) openOrClone(ctx context.Context) (repo *git.Repository, cloned bool, err error) {
	w.printf("Opening repository at \"%s\"\n", w.path)
	repo, err = w.open()
	switch err {
	case nil:
	case git.ErrRepositoryNotExists:
		w.printf("Repository does not exist\nCloning from %s\n", w.origin)
		cloned = true
		repo, err = w.clone(ctx)
	default:
		err = fmt.Errorf("unable to open repository located at \"%s\": %v", w.path, err)
	}
	return
}

func (w *Worker) open() (repo *git.Repository, err error) {
	return git.PlainOpen(w.path)
}

func (w *Worker) clone(ctx context.Context) (repo *git.Repository, err error) {
	return git.PlainCloneContext(ctx, w.path, false, &git.CloneOptions{
		URL:           w.origin,
		ReferenceName: w.branch,
		Progress:      w.progress,
		Auth:          w.auth,
	})
}

// delete attempts to remove the git directory within w.path after performing
// some sanity checks.
func (w *Worker) delete() error {
	// Make sure it looks like a repository
	root, err := os.Stat(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to access path \"%s\": %v", w.path, err)
	}

	if !root.IsDir() {
		return fmt.Errorf("repository path \"%s\" is not a directory", w.path)
	}

	gitPath := filepath.Join(w.path, ".git")
	gitDir, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to access path \"%s\": %v", gitPath, err)
	}

	if !gitDir.IsDir() {
		return fmt.Errorf("repository path \"%s\" is not a directory", gitDir)
	}

	w.printf("Deleting repository at \"%s\"\n", gitPath)
	return os.RemoveAll(gitPath)
}

/*
func (w *Worker) fetch(ctx context.Context, repo *git.Repository) error {
	repo.FetchContext(ctx, &git.FetchOptions{
		Progress: w.progress,
	})
	return nil
}
*/

func (w *Worker) updateOrigin(repo *git.Repository) error {
	cfg := config.RemoteConfig{
		Name: "origin",
		URLs: []string{w.origin},
	}

	remote, err := repo.Remote("origin")
	switch err {
	case git.ErrRemoteNotFound:
		w.println("Creating origin")
		_, err = repo.CreateRemote(&cfg)
		return err
	case nil:
		urls := remote.Config().URLs
		if len(urls) == 1 && urls[0] == w.origin {
			return nil
		}
		w.println("Updating origin")
		err = repo.DeleteRemote("origin")
		if err != nil {
			return err
		}
		_, err = repo.CreateRemote(&cfg)
		return err
	default:
		return err
	}
}

func (w *Worker) updateBranch(repo *git.Repository, worktree *git.Worktree) error {
	// Test to see whether we already have the local branch
	var existingBranch bool
	_, err := repo.Reference(w.branch, false)
	if err == nil {
		existingBranch = true
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: w.branch,
		Create: !existingBranch,
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("unable to switch to %s branch: %v", w.branch.Short(), err)
	}
	return nil
}

func (w *Worker) printf(format string, v ...interface{}) {
	if w.progress == nil {
		return
	}
	fmt.Fprintf(w.progress, format, v...)
}

func (w *Worker) println(v ...interface{}) {
	if w.progress == nil {
		return
	}
	fmt.Fprintln(w.progress, v...)
}
