package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	logger_util "bitbucket.org/free5gc-team/util/logger"
)

var (
	log          *logrus.Logger
	MainLog      *logrus.Entry
	InitLog      *logrus.Entry
	CfgLog       *logrus.Entry
	CtxLog       *logrus.Entry
	GinLog       *logrus.Entry
	SBILog       *logrus.Entry
	ConsumerLog  *logrus.Entry
	ProcessorLog *logrus.Entry
	TrafInfluLog *logrus.Entry
	PFDManageLog *logrus.Entry
	PFDFLog      *logrus.Entry
	OamLog       *logrus.Entry
)

const (
	FieldAFID       string = "af_id"
	FieldSubID      string = "sub_id"
	FieldPfdTransID string = "pfdTrans_id"
)

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category", FieldAFID, FieldSubID, FieldPfdTransID},
	}

	MainLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Main"})
	InitLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Init"})
	CfgLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "CFG"})
	CtxLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "CTX"})
	GinLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "GIN"})
	SBILog = log.WithFields(logrus.Fields{"component": "NEF", "category": "SBI"})
	ConsumerLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Consumer"})
	ProcessorLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "Processor"})
	TrafInfluLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "TrafficInfluence"})
	PFDManageLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "PFDManagement"})
	PFDFLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "PFDF"})
	OamLog = log.WithFields(logrus.Fields{"component": "NEF", "category": "OAM"})
}

func LogFileHook(logNfPath string, log5gcPath string) error {
	if fullPath, err := logger_util.CreateFree5gcLogFile(log5gcPath); err == nil {
		if fullPath != "" {
			free5gcLogHook, hookErr := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
			if hookErr != nil {
				return hookErr
			}
			log.Hooks.Add(free5gcLogHook)
		}
	} else {
		return err
	}

	if fullPath, err := logger_util.CreateNfLogFile(logNfPath, "nef.log"); err == nil {
		selfLogHook, hookErr := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if hookErr != nil {
			return hookErr
		}
		log.Hooks.Add(selfLogHook)
	} else {
		return err
	}

	return nil
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(enable bool) {
	log.SetReportCaller(enable)
}
