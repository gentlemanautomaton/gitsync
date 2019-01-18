package gitsync

import "gopkg.in/src-d/go-git.v4/plumbing/transport"

// Auth returns an option that sets the given authentication method.
func Auth(auth transport.AuthMethod) Option {
	return func(s *Synchronizer) {
		s.auth = auth
	}
}
