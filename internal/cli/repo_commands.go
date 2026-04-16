package cli

import (
	"os"

	"github.com/forjd/aid/internal/config"
	"github.com/forjd/aid/internal/output"
)

func initCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "init does not accept arguments")
	}

	ctx := streams.context()
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
		return newError(ErrCodeUsage, "status does not accept arguments")
	}

	ctx := streams.context()
	env, err := discoverEnvironment(streams)
	if err != nil {
		return err
	}

	configExists := true
	if _, err := os.Stat(env.RepoConfigPath); err != nil {
		if os.IsNotExist(err) {
			configExists = false
		} else {
			return newError(ErrCodeInternal, "stat repo config: %v", err)
		}
	}

	result := output.StatusResult{
		RepoName:     env.RepoName,
		RepoPath:     env.RepoRoot,
		Branch:       env.Branch,
		DBPath:       env.DBPath,
		ConfigPath:   env.RepoConfigPath,
		ConfigExists: configExists,
	}

	exists, err := repoExists(env)
	if err != nil {
		return err
	}
	if !exists {
		return output.RenderStatus(streams.Out, streams.Options, result)
	}

	_, db, err := openStore(ctx, streams)
	if err != nil {
		return err
	}
	defer db.Close()

	repo, err := db.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		return err
	}

	result.Initialized = repo != nil
	if repo != nil {
		counts, err := db.StatusCounts(ctx, repo.ID)
		if err != nil {
			return err
		}
		result.Counts = counts
	}

	return output.RenderStatus(streams.Out, streams.Options, result)
}
