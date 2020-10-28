package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/logger_conf"
	"bitbucket.org/free5gc-team/logger_util"
)

var log *logrus.Logger
var MainLog *logrus.Entry
var InitLog *logrus.Entry
var GinLog *logrus.Entry
var SBIServerLog *logrus.Entry
var ProcessorLog *logrus.Entry
var TrafInfluLog *logrus.Entry
var PFDManageLog *logrus.Entry
var OamLog *logrus.Entry

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	}

	free5gcLogHook, err := logger_util.NewFileHook(logger_conf.Free5gcLogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err == nil {
		log.Hooks.Add(free5gcLogHook)
	}

	selfLogHook, err := logger_util.NewFileHook(logger_conf.NfLogDir+"nef.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err == nil {
		log.Hooks.Add(selfLogHook)
	}

	MainLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Main"})
	InitLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Init"})
	GinLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "GIN"})
	SBIServerLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "SBIServer"})
	ProcessorLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Processor"})
	TrafInfluLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "TrafficInfluence"})
	PFDManageLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "PFDManagement"})
	OamLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "OAM"})
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(bool bool) {
	log.SetReportCaller(bool)
}
