package history

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/forjd/aid/internal/git"
	"github.com/forjd/aid/internal/store"
)

type GitClient interface {
	AllCommitSHAs(startDir string) ([]string, error)
	CommitsBySHA(startDir string, shas []string) ([]git.Commit, error)
}

type CommitStore interface {
	SyncCommits(ctx context.Context, input store.SyncCommitsInput) (store.SyncCommitsResult, error)
}

type DefaultGitClient struct {
	Ctx context.Context
}

func (c DefaultGitClient) ctx() context.Context {
	if c.Ctx != nil {
		return c.Ctx
	}
	return context.Background()
}

func (c DefaultGitClient) AllCommitSHAs(startDir string) ([]string, error) {
	return git.AllCommitSHAs(c.ctx(), startDir)
}

func (c DefaultGitClient) CommitsBySHA(startDir string, shas []string) ([]git.Commit, error) {
	return git.CommitsBySHA(c.ctx(), startDir, shas)
}

type Service struct {
	Git   GitClient
	Store CommitStore
	Now   func() time.Time
}

type Result struct {
	Indexed int
	Added   int
	Updated int
	Removed int
	Initial bool
}

func (s Service) Index(ctx context.Context, repoRoot string, repoID int64, ignorePaths []string) (Result, error) {
	if s.Store == nil {
		return Result{}, fmt.Errorf("history store is required")
	}

	gitClient := s.Git
	if gitClient == nil {
		gitClient = DefaultGitClient{}
	}

	now := s.Now
	if now == nil {
		now = time.Now
	}

	reachableSHAs, err := gitClient.AllCommitSHAs(repoRoot)
	if err != nil {
		return Result{}, err
	}

	// Fetch commits fresh from git each run. This keeps the index consistent
	// with the current ignore_paths config: if paths that were previously
	// filtered are now included, they reappear on the next index run.
	freshCommits, err := gitClient.CommitsBySHA(repoRoot, reachableSHAs)
	if err != nil {
		return Result{}, err
	}

	freshBySHA := make(map[string]git.Commit, len(freshCommits))
	for _, commit := range freshCommits {
		freshBySHA[commit.SHA] = commit
	}

	storeCommits := make([]store.Commit, 0, len(reachableSHAs))
	totalReachable := len(reachableSHAs)
	for index, sha := range reachableSHAs {
		commit, ok := freshBySHA[sha]
		if !ok {
			continue
		}
		filtered, keep := filteredGitCommit(commit, ignorePaths)
		if keep {
			filtered.GitOrder = totalReachable - index - 1
			storeCommits = append(storeCommits, filtered)
		}
	}

	syncResult, err := s.Store.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    repoID,
		Commits:   storeCommits,
		IndexedAt: now().UTC(),
	})
	if err != nil {
		return Result{}, err
	}

	return Result{
		Indexed: len(storeCommits),
		Added:   syncResult.Added,
		Updated: syncResult.Updated,
		Removed: syncResult.Removed,
		Initial: syncResult.Initial,
	}, nil
}

func Mode(initial bool) string {
	if initial {
		return "initial sync"
	}
	return "incremental sync"
}

func filterChangedPaths(paths []string, ignorePaths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if matchesIgnoredPath(path, ignorePaths) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

func filteredGitCommit(commit git.Commit, ignorePaths []string) (store.Commit, bool) {
	paths := filterChangedPaths(commit.ChangedPaths, ignorePaths)
	if len(commit.ChangedPaths) > 0 && len(paths) == 0 {
		return store.Commit{}, false
	}

	return store.Commit{
		SHA:          commit.SHA,
		Author:       commit.Author,
		CommittedAt:  commit.CommittedAt,
		Message:      commit.Message,
		Summary:      commit.Summary,
		ChangedPaths: paths,
	}, true
}

func matchesIgnoredPath(path string, ignorePaths []string) bool {
	normalizedPath := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, prefix := range ignorePaths {
		normalizedPrefix := strings.TrimPrefix(strings.ReplaceAll(prefix, "\\", "/"), "./")
		if normalizedPrefix != "" && strings.HasPrefix(normalizedPath, normalizedPrefix) {
			return true
		}
	}
	return false
}
