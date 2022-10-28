//go:build embed

package main

import "embed"

//go:embed *.html Scripts Images Fonts Styles Audio
var resources embed.FS
