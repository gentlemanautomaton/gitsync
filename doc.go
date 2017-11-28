// Package gitsync provides file tree mirroring via git. It is capable of
// cloning a remote repository and updating the local copy to match the head
// of a particular branch.
//
// This package assumes that the local copy is non-authoritative and that
// any local changes found may be discarded. It performs the equivalent of a
// "git reset hard" whenever the local copy is synchronized.
package gitsync
