package deej

import (
	"os"
	"os/exec"
	"path/filepath"
)

// OpenSettings reuses this executable in UI-only mode; it never opens a second serial connection.
func OpenSettings(configFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(configFile)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--settings", "--config", absolute)
	if err = cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
