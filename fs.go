//go:build !embed

package main

import (
	"os"
)

var resources = os.DirFS(".")
