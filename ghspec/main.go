package main

import (
	"os"

	"github.com/nmiyake/pkg/errorstringer"
	"github.com/palantir/pkg/cli"

	"github.com/nmiyake/ghcli/common"
	"github.com/nmiyake/ghcli/ghspec/cmd"
)

func main() {
	app := cli.NewApp(cli.DebugHandler(errorstringer.SingleStack))
	app.Name = "ghspec"
	app.Usage = "write, verify and apply declarative specifications for GitHub repositories"
	app.Subcommands = []cli.Command{
		common.RateLimit(),
		cmd.CreateSpec(),
		cmd.VerifySpec(),
		cmd.ApplySpec(),
	}
	os.Exit(app.Run(os.Args))
}
