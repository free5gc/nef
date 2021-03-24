package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	nefApp "bitbucket.org/free5gc-team/nef/app"
	"bitbucket.org/free5gc-team/nef/internal/logger"
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
		cli.StringFlag{
			Name:  "log, l",
			Usage: "Output NF log to `FILE`",
		},
		cli.StringFlag{
			Name:  "log5gc, lc",
			Usage: "Output free5gc log to `FILE`",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.MainLog.Errorf("NEF Cli Run err: %v\n", err)
	}
}

func action(cliCtx *cli.Context) error {
	if err := initLogFile(cliCtx.String("log"), cliCtx.String("log5gc")); err != nil {
		logger.MainLog.Errorf("%+v", err)
		return err
	}

	logger.MainLog.Infoln("NEF version: ", version.GetVersion())

	nef := nefApp.NewApp(cliCtx.String("config"))
	if nef == nil {
		return fmt.Errorf("New NEF failed")
	}

	if err := nef.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
		return err
	}

	return nil
}

func initLogFile(logNfPath, log5gcPath string) error {
	if err := logger.LogFileHook(logNfPath, log5gcPath); err != nil {
		return err
	}
	return nil
}
