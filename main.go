package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nef := nefApp.NewApp(ctx, cliCtx.String("config"))
	if nef == nil {
		logger.MainLog.Errorf("New NEF failed")
	}

	if err := nef.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown UPF
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Receive the interrupt signal
	logger.MainLog.Infof("Shutdown NEF ...")
	// Notify each goroutine and wait them stopped
	cancel()
	nef.WaitRoutineStopped()
	logger.MainLog.Infof("NEF exited")
	return nil
}

func initLogFile(logNfPath, log5gcPath string) error {
	if err := logger.LogFileHook(logNfPath, log5gcPath); err != nil {
		return err
	}
	return nil
}
