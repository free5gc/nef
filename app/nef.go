package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/nef/internal/consumer"
	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/notifier"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/nef/internal/sbi"
	openApiLogger "bitbucket.org/free5gc-team/openapi/logger"
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

func NewApp(ctx context.Context, cfgPath string) *NefApp {
	nef := &NefApp{ctx: ctx, cfg: &factory.Config{}}

	if err := nef.initConfig(cfgPath); err != nil {
		switch errType := err.(type) {
		case govalidator.Errors:
			validErrs := err.(govalidator.Errors).Errors()
			for _, validErr := range validErrs {
				logger.InitLog.Errorf("%+v", validErr)
			}
		default:
			logger.InitLog.Errorf("%+v", errType)
		}
		logger.InitLog.Errorf("[-- PLEASE REFER TO SAMPLE CONFIG FILE COMMENTS --]")
		return nil
	}
	if nef.nefCtx = nefctx.NewNefContext(); nef.nefCtx == nil {
		return nil
	}
	if nef.consumer = consumer.NewConsumer(nef.cfg, nef.nefCtx); nef.consumer == nil {
		return nil
	}
	if nef.notifier = notifier.NewNotifier(); nef.notifier == nil {
		return nil
	}
	if nef.proc = processor.NewProcessor(nef.cfg, nef.nefCtx, nef.consumer, nef.notifier); nef.proc == nil {
		return nil
	}
	if nef.sbiServer = sbi.NewServer(nef.cfg, nef.proc); nef.sbiServer == nil {
		return nil
	}
	return nef
}

func (n *NefApp) initConfig(cfgPath string) error {
	if err := factory.InitConfigFactory(cfgPath, n.cfg); err != nil {
		return fmt.Errorf("initConfig [%s] Error: %+v", cfgPath, err)
	}
	if err := factory.CheckConfigVersion(n.cfg); err != nil {
		return err
	}
	if _, err := n.cfg.Validate(); err != nil {
		return err
	}

	n.cfg.Print()
	n.setLogLevel()
	return nil
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
	if cLogger.OpenApi != nil {
		setLoggerLogLevel("OpenApi", cLogger.OpenApi.DebugLevel, cLogger.OpenApi.ReportCaller,
			openApiLogger.SetLogLevel, openApiLogger.SetReportCaller)
	}
}

func (n *NefApp) Run() error {
	n.wg.Add(1)
	/* Go Routine is spawned here for listening for cancellation event on
	 * context */
	go n.listenShutdownEvent()

	if err := n.sbiServer.Run(n.ctx, &n.wg); err != nil {
		return err
	}
	return nil
}

func (n *NefApp) listenShutdownEvent() {
	defer n.wg.Done()
	<-n.ctx.Done()
	n.sbiServer.Stop(n.ctx, &n.wg)
}

func (n *NefApp) WaitRoutineStopped() {
	n.wg.Wait()
}
