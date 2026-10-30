package main

import (
	"os"

	"github.com/mdhender/parser/internal/lemon"
)

func main() { os.Exit(lemon.Run(os.Args, os.Stdout, os.Stderr)) }
