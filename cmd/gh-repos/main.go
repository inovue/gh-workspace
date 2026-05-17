package main

import (
	"os"

	"github.com/inovue/gh-repos-extension/internal/app"
)

func main() {
	os.Exit(app.New(app.Config{
		IsTerminal: isTerminal(os.Stdin),
	}).Run(os.Args[1:]))
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
