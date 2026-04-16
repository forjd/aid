package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/forjd/aid/internal/app"
	"github.com/forjd/aid/internal/config"
	"github.com/forjd/aid/internal/store"
	sqlitestore "github.com/forjd/aid/internal/store/sqlite"
)

type repoRuntime struct {
	env   app.Environment
	cfg   config.RepoConfig
	store store.Store
	repo  *store.Repo
}

// discoverEnvironment resolves the repo/data-dir environment without touching
// the database. It is safe to call before `aid init`.
func discoverEnvironment(streams Streams) (app.Environment, error) {
	return app.Discover(streams.Options.RepoPath)
}

func openStore(ctx context.Context, streams Streams) (app.Environment, store.Store, error) {
	env, err := discoverEnvironment(streams)
	if err != nil {
		return app.Environment{}, nil, err
	}

	sqliteStore, err := sqlitestore.Open(env.DBPath)
	if err != nil {
		return app.Environment{}, nil, err
	}

	if err := sqliteStore.Migrate(ctx); err != nil {
		_ = sqliteStore.Close()
		return app.Environment{}, nil, err
	}

	return env, sqliteStore, nil
}

// repoExists reports whether the sqlite database already exists. It never
// creates the file and can be used from read-only flows such as `aid status`.
func repoExists(env app.Environment) (bool, error) {
	if _, err := os.Stat(env.DBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat aid database: %w", err)
	}
	return true, nil
}

func openInitializedRepo(ctx context.Context, streams Streams) (*repoRuntime, error) {
	env, db, err := openStore(ctx, streams)
	if err != nil {
		return nil, err
	}

	repo, err := db.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if repo == nil {
		_ = db.Close()
		return nil, newError(ErrCodeNotInitialised, "repo not initialised; run \"aid init\" first")
	}

	cfg, err := config.LoadRepoConfig(env.RepoConfigPath)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &repoRuntime{
		env:   env,
		cfg:   cfg,
		store: db,
		repo:  repo,
	}, nil
}

func (runtime *repoRuntime) close() {
	_ = runtime.store.Close()
}
