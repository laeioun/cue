package main

import (
	"os"

	"github.com/laeioun/cue/internal/cli"
	"github.com/laeioun/cue/specs"
)

func main() {
	if err := cli.Execute(specs.FS); err != nil {
		os.Exit(1)
	}
}
