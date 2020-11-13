package nef

import (
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/nef/internal/consumer"
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/nef/internal/sbi"
	openApiLogger "bitbucket.org/free5gc-team/openapi/logger"
	pathUtilLogger "bitbucket.org/free5gc-team/path_util/logger"
)

type NefApp struct {
	cfg       *factory.Config
	nefCtx    *context.NefContext
	processor *processor.Processor
	sbiServer *sbi.SBIServer
	consumer  *consumer.Consumer
}

func NewNEF(nefcfgPath string) *NefApp {
	nef := &NefApp{cfg: &factory.Config{}}
	if err := nef.initConfig(nefcfgPath); err != nil {
		return nil
	}
	if nef.nefCtx = context.NewNefContext(); nef.nefCtx == nil {
		return nil
	}
	if nef.consumer = consumer.NewConsumer(nef.cfg, nef.nefCtx); nef.consumer == nil {
		return nil
	}
	if nef.processor = processor.NewProcessor(nef.cfg, nef.nefCtx, nef.consumer); nef.processor == nil {
		return nil
	}
	if nef.sbiServer = sbi.NewSBIServer(nef.cfg, nef.processor); nef.sbiServer == nil {
		return nil
	}
	return nef
}

func (n *NefApp) initConfig(nefcfgPath string) error {
	if err := factory.InitConfigFactory(nefcfgPath, n.cfg); err != nil {
		logger.InitLog.Errorf("initConfig [%s] Error: %+v", nefcfgPath, err)
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
	if cLogger.PathUtil != nil {
		setLoggerLogLevel("PathUtil", cLogger.PathUtil.DebugLevel, cLogger.PathUtil.ReportCaller,
			pathUtilLogger.SetLogLevel, pathUtilLogger.SetReportCaller)
	}
	if cLogger.OpenApi != nil {
		setLoggerLogLevel("OpenApi", cLogger.OpenApi.DebugLevel, cLogger.OpenApi.ReportCaller,
			openApiLogger.SetLogLevel, openApiLogger.SetReportCaller)
	}
}

func (n *NefApp) Run() error {
	return n.sbiServer.ListenAndServe(n.cfg.GetSbiScheme())
}
