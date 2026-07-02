package main

import (
	"os"

	"work-watch/internal/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:]))
}
