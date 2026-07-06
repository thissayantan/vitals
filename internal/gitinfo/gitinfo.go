// Package gitinfo provides cheap, cached git information for the git segment.
//
// Performance (DESIGN.md §3): prefer reading .git directly over execing git.
//   - Branch: read .git/HEAD (ref: refs/heads/<branch>) — no exec.
//   - Dirty: exec `git status --porcelain`, CACHED 5s keyed by repo path +
//     .git/index mtime; short timeout on the exec.
//   - Optional short SHA via .git/HEAD detached value or `git rev-parse --short`.
package gitinfo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thissayantan/vitals/internal/cache"
)

// Info is the resolved git state for a directory.
type Info struct {
	IsRepo bool
	Branch string
	SHA    string // short SHA (only populated when requested via GetSHA)
	Dirty  bool
}

// execTimeout bounds any git subprocess so a slow repo never blows the budget.
const execTimeout = 250 * time.Millisecond

// Get resolves git info for dir. branchHint (from worktree.branch in the session
// JSON) is preferred over reading .git/HEAD. Dirty state is cached for 5s keyed
// by the repo dir + .git/index mtime.
func Get(dir, branchHint string, c *cache.Store) Info {
	gitDir, ok := findGitDir(dir)
	if !ok {
		return Info{}
	}
	info := Info{IsRepo: true}

	info.Branch = branchHint
	if info.Branch == "" {
		info.Branch = branchFromHead(gitDir)
	}
	if info.Branch == "" {
		info.Branch = shortSHA(dir) // detached HEAD fallback
	}

	info.Dirty = cachedDirty(dir, gitDir, c)
	return info
}

// GetSHA is like Get but also resolves the short SHA (lazy: only when the git
// segment has showSha enabled).
func GetSHA(dir, branchHint string, c *cache.Store) Info {
	info := Get(dir, branchHint, c)
	if info.IsRepo {
		info.SHA = shortSHA(dir)
	}
	return info
}

// findGitDir walks up from dir looking for a .git entry. It returns the resolved
// git directory (handling the `gitdir: …` pointer used by worktrees/submodules).
func findGitDir(dir string) (string, bool) {
	d := dir
	for {
		gp := filepath.Join(d, ".git")
		if fi, err := os.Stat(gp); err == nil {
			if fi.IsDir() {
				return gp, true
			}
			// .git file pointing elsewhere (worktree/submodule).
			if data, err := os.ReadFile(gp); err == nil {
				line := strings.TrimSpace(string(data))
				if rest, ok := strings.CutPrefix(line, "gitdir: "); ok {
					target := rest
					if !filepath.IsAbs(target) {
						target = filepath.Join(d, target)
					}
					return target, true
				}
			}
			return gp, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}

// branchFromHead reads .git/HEAD and extracts the branch name (no exec).
func branchFromHead(gitDir string) string {
	data, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	if ref, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return ref
	}
	return ""
}

// cachedDirty returns the dirty flag, cached 5s by repo + index mtime.
func cachedDirty(dir, gitDir string, c *cache.Store) bool {
	var indexMtime int64
	if fi, err := os.Stat(filepath.Join(gitDir, "index")); err == nil {
		indexMtime = fi.ModTime().UnixNano()
	}
	key := "git-dirty:" + dir + ":" + strconv.FormatInt(indexMtime, 10)
	out := c.Memo(key, cache.TTLGit, func() []byte {
		if isDirty(dir) {
			return []byte("1")
		}
		return []byte("0")
	})
	return string(out) == "1"
}

// isDirty execs `git status --porcelain` with a short timeout.
func isDirty(dir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return false
	}
	return buf.Len() > 0
}

// shortSHA execs `git rev-parse --short HEAD` with a short timeout.
func shortSHA(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
