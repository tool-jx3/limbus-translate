// Command lct builds the zh-TW language pack for Limbus Company. Subcommands
// register themselves with the cli package from files in this directory.
package main

import (
	"os"

	"github.com/tool-jx3/limbus-translate/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
