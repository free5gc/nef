package main

import (
	"os"

	"github.com/urfave/cli"

	"bitbucket.org/free5gc-team/nef/pkg"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/version"
)

func main() {
	app := cli.NewApp()
	app.Name = "nef"
	logger.MainLog.Infoln("NEF version: ", version.GetVersion())
	app.Usage = "-nefcfg nef configuration file"
	app.Action = action
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "nefcfg",
			Usage: "config file",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.MainLog.Errorf("NEF Cli Run err: %v", err)
	}
}

func action(cliCtx *cli.Context) {
	nefApp := nef.NewNEF(cliCtx.String("nefcfg"))
	if nefApp == nil {
		logger.MainLog.Errorf("New NEF failed")
		return
	}
	if err := nefApp.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
	}
}
