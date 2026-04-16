package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/forjd/aid/internal/git"
)

type Environment struct {
	WorkingDir     string
	RepoRoot       string
	RepoName       string
	Branch         string
	AppDataDir     string
	DBPath         string
	RepoConfigDir  string
	RepoConfigPath string
}

func Discover(cwd string) (Environment, error) {
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Environment{}, fmt.Errorf("get working directory: %w", err)
		}
		cwd = wd
	}

	repoRoot, err := git.Root(cwd)
	if err != nil {
		return Environment{}, err
	}

	branch, err := git.Branch(repoRoot)
	if err != nil {
		return Environment{}, err
	}

	dataDir, err := dataDir()
	if err != nil {
		return Environment{}, err
	}

	return Environment{
		WorkingDir:     cwd,
		RepoRoot:       repoRoot,
		RepoName:       filepath.Base(repoRoot),
		Branch:         branch,
		AppDataDir:     dataDir,
		DBPath:         filepath.Join(dataDir, "aid.db"),
		RepoConfigDir:  filepath.Join(repoRoot, ".aid"),
		RepoConfigPath: filepath.Join(repoRoot, ".aid", "config.toml"),
	}, nil
}

func dataDir() (string, error) {
	raw, err := rawDataDir()
	if err != nil {
		return "", err
	}
	return resolveDataDir(raw)
}

func rawDataDir() (string, error) {
	if override := os.Getenv("AID_DATA_DIR"); override != "" {
		return override, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "aid"), nil
	case "windows":
		if localAppData := os.Getenv("LocalAppData"); localAppData != "" {
			return filepath.Join(localAppData, "aid"), nil
		}
		return filepath.Join(home, "AppData", "Local", "aid"), nil
	default:
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "aid"), nil
		}
		return filepath.Join(home, ".local", "share", "aid"), nil
	}
}

func resolveDataDir(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("resolve app data directory: empty path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve app data directory %q: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}
