package util

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"strings"
)

const startupKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func StartupEnabled(executable string) bool {
	k, e := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.QUERY_VALUE)
	if e != nil {
		return false
	}
	defer k.Close()
	v, _, e := k.GetStringValue("DeejUpdated")
	return e == nil && strings.EqualFold(v, fmt.Sprintf("%q", executable))
}
func SetStartup(executable string, enabled bool) error {
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
		return k.SetStringValue("DeejUpdated", fmt.Sprintf("%q", executable))
	}
	err = k.DeleteValue("DeejUpdated")
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
