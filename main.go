package main

import (
	"os"

	"github.com/urfave/cli"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg"
	"bitbucket.org/free5gc-team/version"
)

func main() {
	app := cli.NewApp()
	app.Name = "nef"
	app.Usage = "5G Network Exposure Function (NEF)"
	app.Action = action
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
		},
	}
	logger.MainLog.Infoln("NEF version: ", version.GetVersion())

	if err := app.Run(os.Args); err != nil {
		logger.MainLog.Errorf("NEF Cli Run err: %v", err)
	}
}

func action(cliCtx *cli.Context) {
	nefApp := nef.NewNEF(cliCtx.String("config"))
	if nefApp == nil {
		logger.MainLog.Errorf("New NEF failed")
		return
	}

	if err := nefApp.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
	}
}
