package history

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/forjd/aid/internal/git"
	"github.com/forjd/aid/internal/store"
)

func TestServiceIndexInitialSync(t *testing.T) {
	now := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
	gitClient := &fakeGitClient{
		reachable: []string{"bbb222", "aaa111"},
		commits: map[string]git.Commit{
			"aaa111": {
				SHA:          "aaa111",
				Author:       "Dan",
				CommittedAt:  now.Add(-time.Hour),
				Message:      "feat: add one",
				Summary:      "feat: add one",
				ChangedPaths: []string{"one.go"},
			},
			"bbb222": {
				SHA:          "bbb222",
				Author:       "Dan",
				CommittedAt:  now,
				Message:      "fix: add two",
				Summary:      "fix: add two",
				ChangedPaths: []string{"two.go"},
			},
		},
	}
	commitStore := &fakeCommitStore{
		syncResult: store.SyncCommitsResult{Added: 2, Total: 2, Initial: true},
	}

	service := Service{Git: gitClient, Store: commitStore, Now: func() time.Time { return now }}
	result, err := service.Index(context.Background(), "/tmp/repo", 42, nil)
	if err != nil {
		t.Fatalf("index history: %v", err)
	}

	if !reflect.DeepEqual(gitClient.requested, []string{"bbb222", "aaa111"}) {
		t.Fatalf("unexpected requested shas: %#v", gitClient.requested)
	}
	if result.Indexed != 2 || result.Added != 2 || !result.Initial {
		t.Fatalf("unexpected index result: %#v", result)
	}
	if commitStore.syncInput.RepoID != 42 || !commitStore.syncInput.IndexedAt.Equal(now) {
		t.Fatalf("unexpected sync input metadata: %#v", commitStore.syncInput)
	}
	if len(commitStore.syncInput.Commits) != 2 {
		t.Fatalf("unexpected synced commits: %#v", commitStore.syncInput.Commits)
	}
	if commitStore.syncInput.Commits[0].SHA != "bbb222" || commitStore.syncInput.Commits[0].GitOrder != 1 {
		t.Fatalf("unexpected first synced commit: %#v", commitStore.syncInput.Commits[0])
	}
	if commitStore.syncInput.Commits[1].SHA != "aaa111" || commitStore.syncInput.Commits[1].GitOrder != 0 {
		t.Fatalf("unexpected second synced commit: %#v", commitStore.syncInput.Commits[1])
	}
}

func TestServiceIndexFiltersIgnoredPathsAndSkipsFilteredCommits(t *testing.T) {
	now := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
	gitClient := &fakeGitClient{
		reachable: []string{"aaa111", "bbb222", "ccc333"},
		commits: map[string]git.Commit{
			"bbb222": {
				SHA:          "bbb222",
				Author:       "Dan",
				CommittedAt:  now.Add(-time.Minute),
				Message:      "chore: vendor update",
				Summary:      "chore: vendor update",
				ChangedPaths: []string{"vendor/deps.txt"},
			},
			"ccc333": {
				SHA:          "ccc333",
				Author:       "Dan",
				CommittedAt:  now,
				Message:      "feat: add handler",
				Summary:      "feat: add handler",
				ChangedPaths: []string{"internal/app/env.go", "vendor/deps.txt"},
			},
		},
	}
	commitStore := &fakeCommitStore{
		existing: []store.Commit{
			{
				SHA:          "aaa111",
				Author:       "Dan",
				CommittedAt:  now.Add(-2 * time.Minute),
				Message:      "feat: existing commit",
				Summary:      "feat: existing commit",
				ChangedPaths: []string{"internal/app/env.go", "vendor/old.txt"},
			},
		},
		syncResult: store.SyncCommitsResult{Added: 1, Updated: 1, Removed: 1, Total: 2, Initial: false},
	}

	service := Service{Git: gitClient, Store: commitStore, Now: func() time.Time { return now }}
	result, err := service.Index(context.Background(), "/tmp/repo", 42, []string{"vendor/"})
	if err != nil {
		t.Fatalf("index history with ignored paths: %v", err)
	}

	if !reflect.DeepEqual(gitClient.requested, []string{"bbb222", "ccc333"}) {
		t.Fatalf("unexpected requested shas: %#v", gitClient.requested)
	}
	if result.Indexed != 2 || result.Added != 1 || result.Updated != 1 || result.Removed != 1 || result.Initial {
		t.Fatalf("unexpected index result: %#v", result)
	}
	if len(commitStore.syncInput.Commits) != 2 {
		t.Fatalf("unexpected synced commits: %#v", commitStore.syncInput.Commits)
	}
	if commitStore.syncInput.Commits[0].SHA != "aaa111" || !reflect.DeepEqual(commitStore.syncInput.Commits[0].ChangedPaths, []string{"internal/app/env.go"}) {
		t.Fatalf("unexpected filtered existing commit: %#v", commitStore.syncInput.Commits[0])
	}
	if commitStore.syncInput.Commits[1].SHA != "ccc333" || !reflect.DeepEqual(commitStore.syncInput.Commits[1].ChangedPaths, []string{"internal/app/env.go"}) {
		t.Fatalf("unexpected filtered new commit: %#v", commitStore.syncInput.Commits[1])
	}
}

func TestMode(t *testing.T) {
	if Mode(true) != "initial sync" {
		t.Fatalf("unexpected initial mode: %q", Mode(true))
	}
	if Mode(false) != "incremental sync" {
		t.Fatalf("unexpected incremental mode: %q", Mode(false))
	}
}

type fakeGitClient struct {
	reachable []string
	commits   map[string]git.Commit
	requested []string
}

func (f *fakeGitClient) AllCommitSHAs(string) ([]string, error) {
	return append([]string(nil), f.reachable...), nil
}

func (f *fakeGitClient) CommitsBySHA(_ string, shas []string) ([]git.Commit, error) {
	f.requested = append([]string(nil), shas...)
	commits := make([]git.Commit, 0, len(shas))
	for _, sha := range shas {
		commits = append(commits, f.commits[sha])
	}
	return commits, nil
}

type fakeCommitStore struct {
	existing   []store.Commit
	syncInput  store.SyncCommitsInput
	syncResult store.SyncCommitsResult
}

func (f *fakeCommitStore) ListCommits(context.Context, int64, int) ([]store.Commit, error) {
	return append([]store.Commit(nil), f.existing...), nil
}

func (f *fakeCommitStore) SyncCommits(_ context.Context, input store.SyncCommitsInput) (store.SyncCommitsResult, error) {
	f.syncInput = input
	return f.syncResult, nil
}
