package util

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"strings"
)

const startupKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func startupCommand(executable string, configPath ...string) string {
	args := []string{executable}
	if len(configPath) > 0 && configPath[0] != "" && !strings.EqualFold(filepath.Dir(configPath[0]), filepath.Dir(executable)) {
		args = append(args, "--config", configPath[0])
	}
	return windows.ComposeCommandLine(args)
}
func StartupEnabled(executable string, configPath ...string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("DeejUpdated")
	return err == nil && strings.EqualFold(v, startupCommand(executable, configPath...))
}
func SetStartup(executable string, enabled bool, configPath ...string) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("startup executable must be an absolute path")
	}
	if enabled {
		if _, err := os.Stat(executable); err != nil {
			return err
		}
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enabled {
		return k.SetStringValue("DeejUpdated", startupCommand(executable, configPath...))
	}
	err = k.DeleteValue("DeejUpdated")
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
