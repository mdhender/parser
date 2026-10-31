package main

import (
	"os"

	"github.com/mdhender/parser/internal/ll1"
)

func main() { os.Exit(ll1.Run(os.Args, os.Stdout, os.Stderr)) }
