//go:build !unix

package config

func repairRepoConfigPermissions(string) error {
	return nil
}
