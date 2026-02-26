package main

import (
	"os"

	"github.com/todda86/yaks/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
