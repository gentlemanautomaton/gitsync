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
)

// Synchronizer is responsible for keeping local files in sync with a remote.
// It should be created by calling gitsync.New.
type Synchronizer struct {
	path     string
	origin   string
	branch   plumbing.ReferenceName
	progress sideband.Progress
}

// New returns a Synchronizer for the repository path, origin and branch.
// The path should specify a file system directory to which the contents of
// the remote branch will be mirrored.
//
// If progress is non-nil, textual progress updates will be written to it.
//
// New is nondestructive. Calls to Sync will perform file system initialization
// and cloning as needed.
func New(path, origin, branch string, progress sideband.Progress) *Synchronizer {
	path, _ = filepath.Abs(path)
	return &Synchronizer{
		path:     path,
		origin:   origin,
		branch:   plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
		progress: progress,
	}
}

// Sync attempts to update the local file system to match a particular branch
// on the origin. It performs the equivalent of git clone, pull and checkout as
// necessary to accomplish this.
//
// Sync is destructive. Files within the local copy may be discarded in order
// for sync to accomplish its goal. In the case of failure sync may attempt to
// destroy the local copy and re-clone.
func (s *Synchronizer) Sync(ctx context.Context) error {
	start := time.Now()

	repo, head, cloned, err := s.prepare(ctx)
	if err != nil {
		return err
	}

	if cloned {
		s.printf("Sync completed in %s\n", time.Now().Sub(start))
		return nil
	}

	err = s.updateOrigin(repo)
	if err != nil {
		return err
	}

	s.printf("Opening worktree\n")
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to open worktree: %v", err)
	}

	if head.Name() != s.branch {
		s.printf("Switching to %s branch\n", s.branch.Short())
		err = s.updateBranch(repo, worktree)
		if err != nil {
			return err
		}
	} else {
		s.printf("Already on %s branch\n", s.branch.Short())
	}

	s.printf("Pulling from %s\n", s.origin)
	err = worktree.Pull(&git.PullOptions{
		ReferenceName: s.branch,
		Progress:      os.Stdout,
	})
	switch err {
	case nil:
	case git.NoErrAlreadyUpToDate:
	default:
		return fmt.Errorf("unable to pull: %v", err)
	}

	s.printf("Sync completed in %s\n", time.Now().Sub(start))

	return nil
}

func (s *Synchronizer) prepare(ctx context.Context) (repo *git.Repository, head *plumbing.Reference, cloned bool, err error) {
	const attempts = 2
	for i := 0; i < attempts; i++ {
		repo, cloned, err = s.openOrClone(ctx)
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
			s.printf("The repository appears to be malformed\nAttempting delete and re-clone\n")
			err = s.delete()
			if err != nil {
				err = fmt.Errorf("unable to delete existing malformed repository: %v", err)
				return
			}
		}
	}

	return
}

func (s *Synchronizer) openOrClone(ctx context.Context) (repo *git.Repository, cloned bool, err error) {
	s.printf("Opening repository at \"%s\"\n", s.path)
	repo, err = s.open()
	switch err {
	case nil:
	case git.ErrRepositoryNotExists:
		s.printf("Repository does not exist\nCloning from %s\n", s.origin)
		cloned = true
		repo, err = s.clone(ctx)
	default:
		err = fmt.Errorf("unable to open repository located at \"%s\": %v", s.path, err)
	}
	return
}

func (s *Synchronizer) open() (repo *git.Repository, err error) {
	return git.PlainOpen(s.path)
}

func (s *Synchronizer) clone(ctx context.Context) (repo *git.Repository, err error) {
	return git.PlainCloneContext(ctx, s.path, false, &git.CloneOptions{
		URL:           s.origin,
		ReferenceName: s.branch,
		Progress:      s.progress,
	})
}

// delete attempts to remove the git directory within s.path after performing
// some sanity checks.
func (s *Synchronizer) delete() error {
	// Make sure it looks like a repository
	root, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to access path \"%s\": %v", s.path, err)
	}

	if !root.IsDir() {
		return fmt.Errorf("repository path \"%s\" is not a directory", s.path)
	}

	gitPath := filepath.Join(s.path, ".git")
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

	s.printf("Deleting repository at \"%s\"\n", gitPath)
	return os.RemoveAll(gitPath)
}

func (s *Synchronizer) fetch(ctx context.Context, repo *git.Repository) error {
	repo.FetchContext(ctx, &git.FetchOptions{
		Progress: s.progress,
	})
	return nil
}

func (s *Synchronizer) updateOrigin(repo *git.Repository) error {
	cfg := config.RemoteConfig{
		Name: "origin",
		URLs: []string{s.origin},
	}

	remote, err := repo.Remote("origin")
	switch err {
	case git.ErrRemoteNotFound:
		s.println("Creating origin")
		_, err = repo.CreateRemote(&cfg)
		return err
	case nil:
		urls := remote.Config().URLs
		if len(urls) == 1 && urls[0] == s.origin {
			return nil
		}
		s.println("Updating origin")
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

func (s *Synchronizer) updateBranch(repo *git.Repository, worktree *git.Worktree) error {
	// Test to see whether we already have the local branch
	var existingBranch bool
	_, err := repo.Reference(s.branch, false)
	if err == nil {
		existingBranch = true
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: s.branch,
		Create: !existingBranch,
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("unable to switch to %s branch: %v", s.branch.Short(), err)
	}
	return nil
}

func (s *Synchronizer) printf(format string, v ...interface{}) {
	if s.progress == nil {
		return
	}
	fmt.Fprintf(s.progress, format, v...)
}

func (s *Synchronizer) println(v ...interface{}) {
	if s.progress == nil {
		return
	}
	fmt.Fprintln(s.progress, v...)
}
