package cli

import (
	"context"
	"fmt"

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

func openStore(ctx context.Context, streams Streams) (app.Environment, store.Store, error) {
	env, err := app.Discover(streams.Options.RepoPath)
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
		return nil, fmt.Errorf("repo not initialised; run \"aid init\" first")
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
