package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewUsesXDGCacheHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", base)
	s := New("sess123")
	want := filepath.Join(base, "vitals", "sess123")
	if s.dir != want {
		t.Errorf("dir = %q, want %q", s.dir, want)
	}
}

func TestNewBlankSessionBucket(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", base)
	s := New("")
	want := filepath.Join(base, "vitals", "_")
	if s.dir != want {
		t.Errorf("blank session dir = %q, want %q", s.dir, want)
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s := New("s")
	if err := s.Set("k", []byte("value")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok := s.Get("k", time.Hour)
	if !ok || string(got) != "value" {
		t.Errorf("Get = (%q,%v), want (value,true)", got, ok)
	}
}

func TestGetMiss(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s := New("s")
	if _, ok := s.Get("absent", time.Hour); ok {
		t.Error("Get(absent) ok = true, want false")
	}
}

func TestGetStale(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s := New("s")
	if err := s.Set("k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	// Age the entry past its TTL deterministically (no sleep).
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(s.path("k"), old, old); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("k", time.Minute); ok {
		t.Error("stale entry returned ok=true, want false")
	}
	// Still valid under a generous TTL.
	if _, ok := s.Get("k", 2*time.Hour); !ok {
		t.Error("entry within TTL returned ok=false, want true")
	}
}

func TestMemoCachesAndReuses(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	s := New("s")
	calls := 0
	compute := func() []byte {
		calls++
		return []byte("computed")
	}
	if got := string(s.Memo("k", time.Hour, compute)); got != "computed" {
		t.Fatalf("first Memo = %q, want computed", got)
	}
	if got := string(s.Memo("k", time.Hour, compute)); got != "computed" {
		t.Fatalf("second Memo = %q, want computed", got)
	}
	if calls != 1 {
		t.Errorf("compute called %d times, want 1 (second call should hit cache)", calls)
	}
}

func TestMemoNilStoreAlwaysComputes(t *testing.T) {
	var s *Store // nil store: no caching
	calls := 0
	compute := func() []byte {
		calls++
		return []byte("x")
	}
	s.Memo("k", time.Hour, compute)
	s.Memo("k", time.Hour, compute)
	if calls != 2 {
		t.Errorf("nil-store compute called %d times, want 2", calls)
	}
}

func TestSafeKeyStableAndSafe(t *testing.T) {
	a := safeKey("git-dirty:/some/path:123")
	b := safeKey("git-dirty:/some/path:123")
	if a != b {
		t.Errorf("safeKey not deterministic: %q vs %q", a, b)
	}
	if a == "" || filepath.Base(a) != a {
		t.Errorf("safeKey %q is not a bare filesystem-safe name", a)
	}
	if safeKey("other") == a {
		t.Error("distinct keys hashed to the same name")
	}
}
