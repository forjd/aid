package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/forjd/aid/internal/config"
	"github.com/forjd/aid/internal/output"
)

func initCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("init does not accept arguments")
	}

	ctx := context.Background()
	dbEnv, db, err := openStore(ctx, streams)
	if err != nil {
		return err
	}
	defer db.Close()

	repo, err := db.UpsertRepo(ctx, dbEnv.RepoRoot, dbEnv.RepoName)
	if err != nil {
		return err
	}

	configCreated, err := config.EnsureRepoConfig(dbEnv.RepoConfigPath)
	if err != nil {
		return err
	}

	return output.RenderInit(streams.Out, streams.Options, output.InitResult{
		RepoName:      repo.Name,
		RepoPath:      repo.Path,
		Branch:        dbEnv.Branch,
		DBPath:        dbEnv.DBPath,
		ConfigPath:    dbEnv.RepoConfigPath,
		ConfigCreated: configCreated,
	})
}

func statusCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("status does not accept arguments")
	}

	ctx := context.Background()
	dbEnv, db, err := openStore(ctx, streams)
	if err != nil {
		return err
	}
	defer db.Close()

	repo, err := db.FindRepoByPath(ctx, dbEnv.RepoRoot)
	if err != nil {
		return err
	}

	configExists := true
	if _, err := os.Stat(dbEnv.RepoConfigPath); err != nil {
		if os.IsNotExist(err) {
			configExists = false
		} else {
			return fmt.Errorf("stat repo config: %w", err)
		}
	}

	result := output.StatusResult{
		RepoName:     dbEnv.RepoName,
		RepoPath:     dbEnv.RepoRoot,
		Branch:       dbEnv.Branch,
		DBPath:       dbEnv.DBPath,
		ConfigPath:   dbEnv.RepoConfigPath,
		ConfigExists: configExists,
		Initialized:  repo != nil,
	}

	if repo != nil {
		counts, err := db.StatusCounts(ctx, repo.ID)
		if err != nil {
			return err
		}
		result.Counts = counts
	}

	return output.RenderStatus(streams.Out, streams.Options, result)
}
