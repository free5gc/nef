/*
 * NEF Configuration Factory
 */

package factory

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"bitbucket.org/free5gc-team/logger_util"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/path_util"
	"github.com/asaskevich/govalidator"
)

// Path of HTTP2 key and log file
var (
	NefDefaultKeyLogPath = path_util.Free5gcPath("free5gc/nefsslkey.log")
	NefDefaultPemPath    = path_util.Free5gcPath("free5gc/support/TLS/nef.pem")
	NefDefaultKeyPath    = path_util.Free5gcPath("free5gc/support/TLS/nef.key")
	NefDefaultConfigPath = path_util.Free5gcPath("free5gc/config/nefcfg.yaml")
)

const (
	NefExpectedConfigVersion = "1.0.0"
	NefSbiDefaultIPv4        = "127.0.0.5"
	NefSbiDefaultPort        = 8000
	NefSbiDefaultScheme      = "https"
	NefDefaultNrfUri         = "https://127.0.0.10:8000"
	TraffInfluResUriPrefix   = "/3gpp-traffic-influence/v1"
	PfdMngResUriPrefix       = "/3gpp-pfd-management/v1"
	NefPfdMngResUriPrefix    = "/nnef-pfdmanagement/v1"
	NefOamResUriPrefix       = "/nnef-oam/v1"
)

type Config struct {
	Info          *Info               `yaml:"info" valid:"required"`
	Configuration *Configuration      `yaml:"configuration" valid:"required"`
	Logger        *logger_util.Logger `yaml:"logger" valid:"optional"`
}

func (c *Config) Validate() (bool, error) {
	if info := c.Info; info != nil {
		if result, err := info.validate(); err != nil {
			return result, err
		}
	}
	if configuration := c.Configuration; configuration != nil {
		if result, err := configuration.validate(); err != nil {
			return result, err
		}
	}
	if logger := c.Logger; logger != nil {
		if result, err := logger.Validate(); err != nil {
			return result, err
		}
	}
	result, err := govalidator.ValidateStruct(c)
	return result, appendInvalid(err)
}

type Info struct {
	Version     string `yaml:"version,omitempty" valid:"type(string)"`
	Description string `yaml:"description,omitempty" valid:"type(string)"`
}

func (i *Info) validate() (bool, error) {
	result, err := govalidator.ValidateStruct(i)
	return result, appendInvalid(err)
}

type Configuration struct {
	Sbi         *Sbi      `yaml:"sbi,omitempty" valid:"required"`
	NrfUri      string    `yaml:"nrfUri,omitempty" valid:"required"`
	ServiceList []Service `yaml:"serviceList,omitempty" valid:"required"`
}

func (c *Configuration) validate() (bool, error) {
	if sbi := c.Sbi; sbi != nil {
		if result, err := sbi.validate(); err != nil {
			return result, err
		}
	}
	for index, serviceName := range c.ServiceList {
		switch {
		case serviceName.ServiceName == "nnef-pfdmanagement":
		default:
			err := errors.New("Invalid serviceList[" + strconv.Itoa(index) + "]: " +
				serviceName.ServiceName + ", should be nnef-pfdmanagement.")
			return false, err
		}
	}
	result, err := govalidator.ValidateStruct(c)
	return result, appendInvalid(err)
}

type Sbi struct {
	Scheme       string `yaml:"scheme" valid:"scheme,required"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty" valid:"ipv4,required"` // IP that is registered at NRF.
	// IPv6Addr  string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty" valid:"ipv4,required"` // IP used to run the server in the node.
	Port        int    `yaml:"port,omitempty" valid:"port,optional"`
}

func (s *Sbi) validate() (bool, error) {
	govalidator.TagMap["scheme"] = govalidator.Validator(func(str string) bool {
		return str == "https" || str == "http"
	})
	result, err := govalidator.ValidateStruct(s)
	return result, appendInvalid(err)
}

type Service struct {
	ServiceName string `yaml:"serviceName"`
	SuppFeat    string `yaml:"suppFeat,omitempty"`
}

func (c *Config) Print() {
	logger.CfgLog.Infof("==================================================")
	if c.Info != nil {
		logger.CfgLog.Infof("Info -")
		logger.CfgLog.Infof("  Version: %s", c.Info.Version)
		logger.CfgLog.Infof("  Description: %s", c.Info.Description)
	}
	if c.Configuration != nil {
		logger.CfgLog.Infof("Configuration -")
		if c.Configuration.Sbi != nil {
			logger.CfgLog.Infof("  Sbi -")
			logger.CfgLog.Infof("    Scheme: %s", c.Configuration.Sbi.Scheme)
			logger.CfgLog.Infof("    RegisterIPv4: %s", c.Configuration.Sbi.RegisterIPv4)
			logger.CfgLog.Infof("    BindingIPv4: %s", c.Configuration.Sbi.BindingIPv4)
			logger.CfgLog.Infof("    Port: %d", c.Configuration.Sbi.Port)
		}
		logger.CfgLog.Infof("  NrfUri: %s", c.Configuration.NrfUri)
		if c.Configuration.ServiceList != nil {
			logger.CfgLog.Infof("ServiceList -")
			for _, s := range c.Configuration.ServiceList {
				logger.CfgLog.Infof("ServiceName: %s, SuppFeat: %s", s.ServiceName, s.SuppFeat)
			}
		}
	}
	logger.CfgLog.Infof("==================================================")
}

func (c *Config) GetVersion() string {
	if c.Info != nil && c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}

func (c *Config) GetSbiScheme() string {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Scheme != "" {
		return c.Configuration.Sbi.Scheme
	}
	return NefSbiDefaultScheme
}

func (c *Config) GetSbiPort() int {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Port != 0 {
		return c.Configuration.Sbi.Port
	}
	return NefSbiDefaultPort
}

func (c *Config) GetSbiBindingIP() string {
	bindIP := "0.0.0.0"
	if c.Configuration == nil || c.Configuration.Sbi == nil {
		return bindIP
	}
	if c.Configuration.Sbi.BindingIPv4 != "" {
		if bindIP = os.Getenv(c.Configuration.Sbi.BindingIPv4); bindIP != "" {
			logger.CfgLog.Infof("Parsing ServerIPv4 [%s] from ENV Variable", bindIP)
		} else {
			bindIP = c.Configuration.Sbi.BindingIPv4
		}
	}
	return bindIP
}

func (c *Config) GetSbiBindingAddr() string {
	return c.GetSbiBindingIP() + ":" + strconv.Itoa(c.GetSbiPort())
}

func (c *Config) GetSbiRegisterIP() string {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.RegisterIPv4 != "" {
		return c.Configuration.Sbi.RegisterIPv4
	}
	return NefSbiDefaultIPv4
}

func (c *Config) GetSbiRegisterAddr() string {
	return c.GetSbiRegisterIP() + ":" + strconv.Itoa(c.GetSbiPort())
}

func (c *Config) GetSbiUri() string {
	return c.GetSbiScheme() + "://" + c.GetSbiRegisterAddr()
}

func (c *Config) GetNrfUri() string {
	if c.Configuration != nil && c.Configuration.NrfUri != "" {
		return c.Configuration.NrfUri
	}
	return NefDefaultNrfUri
}

func (c *Config) GetServiceList() []Service {
	if c.Configuration != nil && c.Configuration.ServiceList != nil && len(c.Configuration.ServiceList) > 0 {
		return c.Configuration.ServiceList
	}
	return nil
}

func appendInvalid(err error) error {
	var errs govalidator.Errors
	if err == nil {
		return nil
	}
	es := err.(govalidator.Errors).Errors()
	for _, e := range es {
		errs = append(errs, fmt.Errorf("Invalid %w", e))
	}
	return error(errs)
}
