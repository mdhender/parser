package main

import (
	"os"

	"github.com/mdhender/parser/internal/lemon"
)

func main() { os.Exit(lemon.RunGo(os.Args, os.Stdout, os.Stderr)) }
