// Package cache is a tiny session-keyed on-disk cache for expensive providers.
//
// See DESIGN.md §6. Files live under $XDG_CACHE_HOME/vitals/ (default
// ~/.cache/vitals/). Keys combine session_id and/or a path hash. Writes are
// atomic (temp file + rename). Concurrent renders are tolerated (last-writer-wins).
package cache

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// TTLs preserved from the prototype (DESIGN.md §6).
const (
	TTLGit     = 5 * time.Second
	TTLRuntime = time.Hour
	TTLCost    = 10 * time.Second
	TTLTasks   = 3 * time.Second
)

// Store is a session-scoped on-disk cache.
type Store struct {
	dir       string
	sessionID string
}

// New returns a Store rooted at $XDG_CACHE_HOME/vitals/<sessionID>/. A blank
// sessionID is tolerated (entries land in a shared "_" bucket).
func New(sessionID string) *Store {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".cache")
		} else {
			base = os.TempDir()
		}
	}
	bucket := sessionID
	if bucket == "" {
		bucket = "_"
	}
	return &Store{
		dir:       filepath.Join(base, "vitals", bucket),
		sessionID: sessionID,
	}
}

// safeKey turns an arbitrary key into a filesystem-safe filename.
func safeKey(key string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return strconv.FormatUint(h.Sum64(), 16)
}

func (s *Store) path(key string) string {
	return filepath.Join(s.dir, safeKey(key))
}

// Get returns the cached bytes for key if the entry exists and is younger than
// ttl. ok is false on miss, stale, or error.
func (s *Store) Get(key string, ttl time.Duration) (data []byte, ok bool) {
	p := s.path(key)
	fi, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if time.Since(fi.ModTime()) > ttl {
		return nil, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return b, true
}

// Set writes data for key atomically (temp file + rename). Errors are returned
// but callers may ignore them — a failed cache write only costs a recompute.
func (s *Store) Set(key string, data []byte) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, "tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path(key))
}

// Memo returns cached bytes for key when fresh, otherwise calls compute, stores
// the result, and returns it. A nil Store computes every time (no caching).
func (s *Store) Memo(key string, ttl time.Duration, compute func() []byte) []byte {
	if s == nil {
		return compute()
	}
	if b, ok := s.Get(key, ttl); ok {
		return b
	}
	b := compute()
	_ = s.Set(key, b)
	return b
}
