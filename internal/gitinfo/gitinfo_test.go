package gitinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitDirRealDir(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := findGitDir(root)
	if !ok || got != gitDir {
		t.Errorf("findGitDir = (%q,%v), want (%q,true)", got, ok, gitDir)
	}
}

func TestFindGitDirWalksUp(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := findGitDir(deep)
	if !ok || got != gitDir {
		t.Errorf("findGitDir(deep) = (%q,%v), want (%q,true)", got, ok, gitDir)
	}
}

func TestFindGitDirWorktreePointerAbsolute(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real-git-dir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	// A worktree's .git is a file: "gitdir: <path>".
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: "+target+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := findGitDir(root)
	if !ok || got != target {
		t.Errorf("findGitDir(abs pointer) = (%q,%v), want (%q,true)", got, ok, target)
	}
}

func TestFindGitDirWorktreePointerRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ./nested/gd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := findGitDir(root)
	want := filepath.Join(root, "nested", "gd")
	if !ok || got != want {
		t.Errorf("findGitDir(rel pointer) = (%q,%v), want (%q,true)", got, ok, want)
	}
}

func TestFindGitDirNotFound(t *testing.T) {
	if _, ok := findGitDir(t.TempDir()); ok {
		t.Error("findGitDir in a non-repo should return ok=false")
	}
}

func TestBranchFromHead(t *testing.T) {
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := branchFromHead(gitDir); got != "feature/x" {
		t.Errorf("branchFromHead = %q, want feature/x", got)
	}
}

func TestBranchFromHeadDetached(t *testing.T) {
	gitDir := t.TempDir()
	// Detached HEAD holds a raw SHA, not a ref — no branch name.
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("a1b2c3d4e5f6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := branchFromHead(gitDir); got != "" {
		t.Errorf("branchFromHead(detached) = %q, want empty", got)
	}
}

func TestBranchFromHeadMissing(t *testing.T) {
	if got := branchFromHead(t.TempDir()); got != "" {
		t.Errorf("branchFromHead(no HEAD) = %q, want empty", got)
	}
}
