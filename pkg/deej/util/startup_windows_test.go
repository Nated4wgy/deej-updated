package util

import (
	"golang.org/x/sys/windows"
	"reflect"
	"testing"
)

func TestStartupCommandRoundTripsWindowsPaths(t *testing.T) {
	executable := "C:/My Apps/deej/deej.exe"
	config := "C:/My Config/config.yaml"
	args, err := windows.DecomposeCommandLine(startupCommand(executable, config))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{executable, "--background", "--config", config}) {
		t.Fatalf("%q", args)
	}
	args, err = windows.DecomposeCommandLine(startupCommand(executable, "C:/My Apps/deej/config.yaml"))
	if err != nil || !reflect.DeepEqual(args, []string{executable, "--background"}) {
		t.Fatalf("%q %v", args, err)
	}
}
