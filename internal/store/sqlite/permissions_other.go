//go:build !unix

package sqlite

func repairDBPermissions(string) error {
	return nil
}
