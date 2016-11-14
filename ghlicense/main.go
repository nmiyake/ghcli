package main

import (
	"os"

	"github.com/nmiyake/pkg/errorstringer"
	"github.com/palantir/pkg/cli"

	"github.com/nmiyake/ghcli/common"
	"github.com/nmiyake/ghcli/ghlicense/cmd"
)

func main() {
	app := cli.NewApp(cli.DebugHandler(errorstringer.SingleStack))
	app.Name = "ghlicense"
	app.Usage = "print, write, verify and fix licenses for GitHub repositories"
	app.Subcommands = []cli.Command{
		common.RateLimit(),
		cmd.List(),
		cmd.Print(),
		cmd.Write(),
		cmd.Verify(),
		cmd.Fix(),
	}
	os.Exit(app.Run(os.Args))
}
