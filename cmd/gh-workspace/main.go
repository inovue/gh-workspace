package main

import (
	"os"
	"runtime/debug"

	"github.com/inovue/gh-workspace/internal/app"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = ""

func main() {
	os.Exit(app.New(app.Config{
		IsTerminal: isTerminal(os.Stdin) && isTerminal(os.Stderr),
		Version:    buildVersion(),
	}).Run(os.Args[1:]))
}

func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
