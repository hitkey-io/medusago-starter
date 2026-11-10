// The MedusaGo backend: the Medusa CLI (develop, start, build, db:*, user,
// exec) running this project's configuration and customizations.
package main

import (
	"os"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Options{
		Config: medusaConfig,
	}))
}
