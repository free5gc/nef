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
	consumer  *consumer.Consumer
	notifier  *notifier.Notifier
	proc      *processor.Processor
	sbiServer *sbi.Server
}

func NewApp(cfg *factory.Config, tlsKeyLogPath string) (*NefApp, error) {
	var err error
	nef := &NefApp{cfg: cfg}

	nef.setLogLevel()
	if nef.nefCtx, err = nefctx.NewNefContext(nef); err != nil {
		return nil, err
	}
	if nef.consumer, err = consumer.NewConsumer(nef); err != nil {
		return nil, err
	}
	if nef.notifier, err = notifier.NewNotifier(); err != nil {
		return nil, err
	}
	if nef.proc, err = processor.NewProcessor(nef); err != nil {
		return nil, err
	}
	if nef.sbiServer, err = sbi.NewServer(nef, tlsKeyLogPath); err != nil {
		return nil, err
	}
	return nef, nil
}

func (a *NefApp) Config() *factory.Config {
	return a.cfg
}

func (a *NefApp) Context() *nefctx.NefContext {
	return a.nefCtx
}

func (a *NefApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *NefApp) Notifier() *notifier.Notifier {
	return a.notifier
}

func (a *NefApp) Processor() *processor.Processor {
	return a.proc
}

func (a *NefApp) SbiServer() *sbi.Server {
	return a.sbiServer
}

func (a *NefApp) setLogLevel() {
	cLogger := a.cfg.Logger
	if cLogger == nil {
		logger.InitLog.Warnln("NEF config without log level setting!!!")
		return
	}
	if cLogger.NEF != nil {
		setLoggerLogLevel("NEF", cLogger.NEF.DebugLevel, cLogger.NEF.ReportCaller,
			logger.SetLogLevel, logger.SetReportCaller)
	}
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

func (a *NefApp) Run() error {
	var cancel context.CancelFunc
	a.ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	a.wg.Add(1)
	/* Go Routine is spawned here for listening for cancellation event on
	 * context */
	go a.listenShutdownEvent()

	if err := a.sbiServer.Run(a.ctx, &a.wg); err != nil {
		return err
	}

	if err := a.consumer.RegisterNFInstance(); err != nil {
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
	a.WaitRoutineStopped()
	logger.MainLog.Infof("NEF exited")
	return nil
}

func (a *NefApp) listenShutdownEvent() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}

		a.wg.Done()
	}()

	<-a.ctx.Done()
	a.sbiServer.Stop(a.ctx, &a.wg)
}

func (a *NefApp) WaitRoutineStopped() {
	a.wg.Wait()
}

func (a *NefApp) Start() {
	if err := a.Run(); err != nil {
		logger.MainLog.Errorf("NEF Run err: %v", err)
	}
}

func (a *NefApp) Terminate() {
	logger.MainLog.Infof("Terminating NEF...")
	logger.MainLog.Infof("NEF terminated")
}
