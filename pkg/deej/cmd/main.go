package main

import (
	"flag"
	"fmt"
 "os"
 "path/filepath"
 "github.com/omriharel/deej/pkg/deej/util"

	"github.com/omriharel/deej/pkg/deej"
)

var (
	gitCommit  string
	versionTag string
	buildType  string

	verbose bool
 configFile string
 checkConfig bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "show verbose logs (useful for debugging serial)")
	flag.BoolVar(&verbose, "v", false, "shorthand for --verbose")
	flag.StringVar(&configFile,"config","","path to config.yaml (defaults to executable directory)")
 flag.BoolVar(&checkConfig,"check-config",false,"validate config and exit without opening serial or audio")
 flag.Parse()
}

func main() {
 exe,err:=os.Executable();if err!=nil{panic(err)}
 directory:=filepath.Dir(exe)
 if configFile!=""{absolute,e:=filepath.Abs(configFile);if e!=nil{panic(e)};if filepath.Base(absolute)!="config.yaml"{panic("config filename must be config.yaml")};directory=filepath.Dir(absolute)}
 if err=os.Chdir(directory);err!=nil{panic(err)}
 if checkConfig {if _,err=deej.ReadSettings("config.yaml");err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};fmt.Println("Configuration valid");return}
 cleanup,err:=util.AcquireInstance(directory);if err!=nil{fmt.Fprintln(os.Stderr,err);return};defer cleanup()


	// first we need a logger
	logger, err := deej.NewLogger(buildType)
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	named := logger.Named("main")
	named.Debug("Created logger")

	named.Infow("Version info",
		"gitCommit", gitCommit,
		"versionTag", versionTag,
		"buildType", buildType)

	// provide a fair warning if the user's running in verbose mode
	if verbose {
		named.Debug("Verbose flag provided, all log messages will be shown")
	}

	// create the deej instance
	d, err := deej.NewDeej(logger, verbose)
	if err != nil {
		named.Fatalw("Failed to create deej object", "error", err)
	}

	// if injected by build process, set version info to show up in the tray
	if buildType != "" && (versionTag != "" || gitCommit != "") {
		identifier := gitCommit
		if versionTag != "" {
			identifier = versionTag
		}

		versionString := fmt.Sprintf("Version %s-%s", buildType, identifier)
		d.SetVersion(versionString)
	}

	// onwards, to glory
	if err = d.Initialize(); err != nil {
		named.Fatalw("Failed to initialize deej", "error", err)
	}
}
