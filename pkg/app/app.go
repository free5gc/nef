package app

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/nef/internal/consumer"
	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/notifier"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/nef/internal/sbi"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
)

type NefApp struct {
	ctx       context.Context
	wg        sync.WaitGroup
	cfg       *factory.Config
	nefCtx    *nefctx.NefContext
	proc      *processor.Processor
	sbiServer *sbi.Server
	consumer  *consumer.Consumer
	notifier  *notifier.Notifier
}

func NewApp(cfg *factory.Config, tlsKeyLogPath string) (*NefApp, error) {
	var err error
	nef := &NefApp{cfg: cfg}

	nef.setLogLevel()
	if nef.nefCtx, err = nefctx.NewNefContext(nef.cfg); err != nil {
		return nil, err
	}
	if nef.consumer, err = consumer.NewConsumer(nef.nefCtx); err != nil {
		return nil, err
	}
	if nef.notifier, err = notifier.NewNotifier(); err != nil {
		return nil, err
	}
	if nef.proc, err = processor.NewProcessor(nef.nefCtx, nef.consumer, nef.notifier); err != nil {
		return nil, err
	}
	if nef.sbiServer, err = sbi.NewServer(nef.nefCtx, nef.proc, tlsKeyLogPath); err != nil {
		return nil, err
	}
	return nef, nil
}

func setLoggerLogLevel(loggerName, DebugLevel string, reportCaller bool,
	logLevelFn func(l logrus.Level), reportCallerFn func(b bool)) {
	if DebugLevel != "" {
		if level, err := logrus.ParseLevel(DebugLevel); err != nil {
			logger.InitLog.Warnf("%s Log level [%s] is invalid, set to [info] level",
				loggerName, DebugLevel)
			logLevelFn(logrus.InfoLevel)
		} else {
			logger.InitLog.Infof("%s Log level is set to [%s] level", loggerName, level)
			logLevelFn(level)
		}
	} else {
		logger.InitLog.Infof("%s Log level is default set to [info] level", loggerName)
		logLevelFn(logrus.InfoLevel)
	}
	reportCallerFn(reportCaller)
}

func (n *NefApp) setLogLevel() {
	cLogger := n.cfg.Logger
	if cLogger == nil {
		logger.InitLog.Warnln("NEF config without log level setting!!!")
		return
	}
	if cLogger.NEF != nil {
		setLoggerLogLevel("NEF", cLogger.NEF.DebugLevel, cLogger.NEF.ReportCaller,
			logger.SetLogLevel, logger.SetReportCaller)
	}
}

func (n *NefApp) Run() error {
	var cancel context.CancelFunc
	n.ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	n.wg.Add(1)
	/* Go Routine is spawned here for listening for cancellation event on
	 * context */
	go n.listenShutdownEvent()

	if err := n.sbiServer.Run(n.ctx, &n.wg); err != nil {
		return err
	}

	// Wait for interrupt signal to gracefully shutdown UPF
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Receive the interrupt signal
	logger.MainLog.Infof("Shutdown NEF ...")
	// Notify each goroutine and wait them stopped
	cancel()
	n.WaitRoutineStopped()
	logger.MainLog.Infof("NEF exited")
	return nil
}

func (n *NefApp) listenShutdownEvent() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}

		n.wg.Done()
	}()

	<-n.ctx.Done()
	n.sbiServer.Stop(n.ctx, &n.wg)
}

func (n *NefApp) WaitRoutineStopped() {
	n.wg.Wait()
}

func (n *NefApp) Start() {
	if err := n.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
	}
}

func (n *NefApp) Terminate() {
	logger.MainLog.Infof("Terminating NEF...")
	logger.MainLog.Infof("NEF terminated")
}
