package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/urfave/cli"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	nefapp "bitbucket.org/free5gc-team/nef/pkg/app"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/util/version"
)

func main() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.MainLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
	}()

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
	tlsKeyLogPath, err := initLogFile(cliCtx.String("log"), cliCtx.String("log5gc"))
	if err != nil {
		return err
	}

	logger.MainLog.Infoln("NEF version: ", version.GetVersion())

	cfg, err := factory.ReadConfig(cliCtx.String("config"))
	if err != nil {
		return err
	}

	nef, err := nefapp.NewApp(cfg, tlsKeyLogPath)
	if err != nil {
		return fmt.Errorf("New NEF err: %+v", err)
	}

	if err := nef.Run(); err != nil {
		return nil
	}

	return nil
}

func initLogFile(logNfPath, log5gcPath string) (string, error) {
	if err := logger.LogFileHook(logNfPath, log5gcPath); err != nil {
		return "", err
	}

	logTlsKeyPath := factory.NefDefaultTLSKeyLogPath
	if logNfPath != "" {
		nfDir, _ := filepath.Split(logNfPath)
		tmpDir := filepath.Join(nfDir, "key")
		if err := os.MkdirAll(tmpDir, 0775); err != nil {
			logger.InitLog.Errorf("Make directory %s failed: %+v", tmpDir, err)
			return "", err
		}
		_, name := filepath.Split(factory.NefDefaultTLSKeyLogPath)
		logTlsKeyPath = filepath.Join(tmpDir, name)
	}

	return logTlsKeyPath, nil
}
