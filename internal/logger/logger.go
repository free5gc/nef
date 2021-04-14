package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/logger_util"
	NamfCommLogger "bitbucket.org/free5gc-team/openapi/Namf_Communication/logger"
	NamfEventLogger "bitbucket.org/free5gc-team/openapi/Namf_EventExposure/logger"
	NnssfNSSAIAvailabilityLogger "bitbucket.org/free5gc-team/openapi/Nnssf_NSSAIAvailability/logger"
	NnssfNSSelectionLogger "bitbucket.org/free5gc-team/openapi/Nnssf_NSSelection/logger"
	NsmfEventLogger "bitbucket.org/free5gc-team/openapi/Nsmf_EventExposure/logger"
	NsmfPDUSessionLogger "bitbucket.org/free5gc-team/openapi/Nsmf_PDUSession/logger"
	NudmEventLogger "bitbucket.org/free5gc-team/openapi/Nudm_EventExposure/logger"
	NudmParameterProvisionLogger "bitbucket.org/free5gc-team/openapi/Nudm_ParameterProvision/logger"
	NudmSubDataManagementLogger "bitbucket.org/free5gc-team/openapi/Nudm_SubscriberDataManagement/logger"
	NudmUEAuthLogger "bitbucket.org/free5gc-team/openapi/Nudm_UEAuthentication/logger"
	NudmUEContextManagLogger "bitbucket.org/free5gc-team/openapi/Nudm_UEContextManagement/logger"
	NudrDataRepositoryLogger "bitbucket.org/free5gc-team/openapi/Nudr_DataRepository/logger"
	openApiLogger "bitbucket.org/free5gc-team/openapi/logger"
)

var log *logrus.Logger
var MainLog *logrus.Entry
var InitLog *logrus.Entry
var CfgLog *logrus.Entry
var CtxLog *logrus.Entry
var GinLog *logrus.Entry
var SBILog *logrus.Entry
var ConsumerLog *logrus.Entry
var ProcessorLog *logrus.Entry
var TrafInfluLog *logrus.Entry
var PFDManageLog *logrus.Entry
var PFDFLog *logrus.Entry
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
	if fileErr, fullPath := logger_util.CreateFree5gcLogFile(log5gcPath); fileErr == nil {
		if fullPath != "" {
			free5gcLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
			if err != nil {
				return err
			}
			log.Hooks.Add(free5gcLogHook)
			openApiLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NamfCommLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NamfEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NnssfNSSAIAvailabilityLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NnssfNSSelectionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NsmfEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NsmfPDUSessionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmEventLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmParameterProvisionLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmSubDataManagementLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmUEAuthLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudmUEContextManagLogger.GetLogger().Hooks.Add(free5gcLogHook)
			NudrDataRepositoryLogger.GetLogger().Hooks.Add(free5gcLogHook)
		}
	} else {
		return fileErr
	}

	if fileErr, fullPath := logger_util.CreateNfLogFile(logNfPath, "nef.log"); fileErr == nil {
		selfLogHook, err := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if err != nil {
			return err
		}
		log.Hooks.Add(selfLogHook)
		openApiLogger.GetLogger().Hooks.Add(selfLogHook)
		NamfCommLogger.GetLogger().Hooks.Add(selfLogHook)
		NamfEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NnssfNSSAIAvailabilityLogger.GetLogger().Hooks.Add(selfLogHook)
		NnssfNSSelectionLogger.GetLogger().Hooks.Add(selfLogHook)
		NsmfEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NsmfPDUSessionLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmEventLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmParameterProvisionLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmSubDataManagementLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmUEAuthLogger.GetLogger().Hooks.Add(selfLogHook)
		NudmUEContextManagLogger.GetLogger().Hooks.Add(selfLogHook)
		NudrDataRepositoryLogger.GetLogger().Hooks.Add(selfLogHook)
	} else {
		return fileErr
	}

	return nil
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(bool bool) {
	log.SetReportCaller(bool)
}
