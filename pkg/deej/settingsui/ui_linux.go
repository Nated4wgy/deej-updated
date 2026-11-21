package settingsui

import "fmt"

func Show(filename, executable, version string) error {
	return fmt.Errorf("the desktop interface is available on Windows")
}
func ReportError(err error) { fmt.Println(err) }
