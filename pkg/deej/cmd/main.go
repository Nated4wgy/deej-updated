package main

import (
	"flag"
	"fmt"
	"github.com/omriharel/deej/pkg/deej"
	"github.com/omriharel/deej/pkg/deej/settingsui"
	"github.com/omriharel/deej/pkg/deej/util"
	"os"
	"path/filepath"
)

var gitCommit, versionTag, buildType string

func main() {
	var verbose, settingsOnly, background, checkConfig bool
	var configFile string
	flag.BoolVar(&verbose, "verbose", false, "show verbose logs")
	flag.BoolVar(&verbose, "v", false, "show verbose logs")
	flag.BoolVar(&settingsOnly, "settings", false, "open the interface without starting another mixer")
	flag.BoolVar(&background, "background", false, "run in the tray without opening the interface")
	flag.BoolVar(&checkConfig, "check-config", false, "validate configuration and exit")
	flag.StringVar(&configFile, "config", "", "path to config.yaml (defaults beside the executable)")
	flag.Parse()
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	filename := filepath.Join(filepath.Dir(exe), "config.yaml")
	if configFile != "" {
		filename, err = filepath.Abs(configFile)
		if err != nil {
			panic(err)
		}
	}
	if filepath.Base(filename) != "config.yaml" {
		fmt.Fprintln(os.Stderr, "config filename must be config.yaml")
		os.Exit(1)
	}
	if err = os.Chdir(filepath.Dir(filename)); err != nil {
		panic(err)
	}
	if checkConfig {
		if _, err = deej.ReadSettings(filename); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Configuration valid")
		return
	}
	version := versionTag
	if version == "" {
		version = gitCommit
	}
	if version == "" {
		version = "development"
	}
	if settingsOnly {
		if err = settingsui.Show(filename, exe, version); err != nil {
			settingsui.ReportError(err)
		}
		return
	}
	cleanup, err := util.AcquireInstance(filepath.Dir(filename))
	if err != nil {
		if util.InstanceAlreadyRunning(err) {
			if !background {
				if err = deej.OpenSettings(filename); err != nil {
					settingsui.ReportError(err)
				}
			}
		} else {
			settingsui.ReportError(err)
		}
		return
	}
	defer cleanup()
	logger, err := deej.NewLogger(buildType)
	if err != nil {
		panic(err)
	}
	logger.Infow("Version info", "gitCommit", gitCommit, "versionTag", versionTag, "buildType", buildType)
	d, err := deej.NewDeej(logger, verbose)
	if err != nil {
		logger.Fatalw("Create mixer", "error", err)
	}
	d.SetVersion(version)
	d.SetOpenSettingsOnStart(!background)
	if err = d.Initialize(); err != nil {
		logger.Fatalw("Initialize mixer", "error", err)
	}
}
