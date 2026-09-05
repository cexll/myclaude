package bootstrap

import (
	"os"

	adaptercli "codeagent-wrapper/internal/adapter/cli"
	app "codeagent-wrapper/internal/app"
)

func Main() {
	os.Exit(adaptercli.Run(os.Args[1:], app.NewRootDeps()))
}
