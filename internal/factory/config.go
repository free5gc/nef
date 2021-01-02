/*
 * NEF Configuration Factory
 */

package factory

import (
	"os"
	"strconv"

	"bitbucket.org/free5gc-team/logger_util"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/path_util"
)

// Path of HTTP2 key and log file
var (
	NEF_LOG_PATH    = path_util.Free5gcPath("free5gc/nefsslkey.log")
	NEF_PEM_PATH    = path_util.Free5gcPath("free5gc/support/TLS/nef.pem")
	NEF_KEY_PATH    = path_util.Free5gcPath("free5gc/support/TLS/nef.key")
	NEF_CONFIG_PATH = path_util.Free5gcPath("free5gc/config/nefcfg.yaml")
)

const (
	NEF_DEFAULT_VERSION        = "1.0.0"
	NEF_DEFAULT_IPV4           = "127.0.0.5"
	NEF_DEFAULT_PORT           = "8000"
	NEF_DEFAULT_PORT_INT       = 8000
	NEF_DEFAULT_SCHEME         = "https"
	NEF_DEFAULT_NRFURI         = "https://127.0.0.10:8000"
	TRAFF_INFLU_RES_URI_PREFIX = "/3gpp-traffic-influence/v1"
	PFD_MNG_RES_URI_PREFIX     = "/3gpp-pfd-management/v1"
	NEF_PFD_MNG_RES_URI_PREFIX = "/nnef-pfdmanagement/v1"
	NEF_OAM_RES_URI_PREFIX     = "/nnef-oam/v1"
)

type Config struct {
	Info          *Info               `yaml:"info"`
	Configuration *Configuration      `yaml:"configuration"`
	Logger        *logger_util.Logger `yaml:"logger"`
}

type Info struct {
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Configuration struct {
	Sbi         *Sbi      `yaml:"sbi,omitempty"`
	TimeFormat  string    `yaml:"timeFormat,omitempty"`
	NrfUri      string    `yaml:"nrfUri,omitempty"`
	ServiceList []Service `yaml:"serviceList,omitempty"`
}

type Sbi struct {
	Scheme       string `yaml:"scheme"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"` // IP that is registered at NRF.
	// IPv6Addr  string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"` // IP used to run the server in the node.
	Port        int    `yaml:"port,omitempty"`
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
		logger.CfgLog.Infof("  TimeFormat: %s", c.Configuration.TimeFormat)
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
	return NEF_DEFAULT_VERSION
}

func (c *Config) GetSbiScheme() string {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Scheme != "" {
		return c.Configuration.Sbi.Scheme
	}
	return NEF_DEFAULT_SCHEME
}

func (c *Config) GetSbiPort() int {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Port != 0 {
		return c.Configuration.Sbi.Port
	}
	return NEF_DEFAULT_PORT_INT
}

func (c *Config) GetSbiBindingAddr() string {
	var bindAddr string
	if c.Configuration == nil || c.Configuration.Sbi == nil {
		return "0.0.0.0:" + NEF_DEFAULT_PORT
	}
	if c.Configuration.Sbi.BindingIPv4 != "" {
		if bindIPv4 := os.Getenv(c.Configuration.Sbi.BindingIPv4); bindIPv4 != "" {
			logger.CfgLog.Infof("Parsing ServerIPv4 [%s] from ENV Variable", bindIPv4)
			bindAddr = bindIPv4 + ":"
		} else {
			bindAddr = c.Configuration.Sbi.BindingIPv4 + ":"
		}
	} else {
		bindAddr = "0.0.0.0:"
	}
	if c.Configuration.Sbi.Port != 0 {
		bindAddr = bindAddr + strconv.Itoa(c.Configuration.Sbi.Port)
	} else {
		bindAddr = bindAddr + NEF_DEFAULT_PORT
	}
	return bindAddr
}

func (c *Config) GetSbiRegisterIP() string {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.RegisterIPv4 != "" {
		return c.Configuration.Sbi.RegisterIPv4
	}
	return NEF_DEFAULT_IPV4
}

func (c *Config) GetSbiRegisterAddr() string {
	regAddr := c.GetSbiRegisterIP() + ":"
	if c.Configuration.Sbi.Port != 0 {
		regAddr = regAddr + strconv.Itoa(c.Configuration.Sbi.Port)
	} else {
		regAddr = regAddr + NEF_DEFAULT_PORT
	}
	return regAddr
}

func (c *Config) GetSbiUri() string {
	return c.GetSbiScheme() + "://" + c.GetSbiRegisterAddr()
}

func (c *Config) GetNrfUri() string {
	if c.Configuration != nil && c.Configuration.NrfUri != "" {
		return c.Configuration.NrfUri
	}
	return NEF_DEFAULT_NRFURI
}

func (c *Config) GetServiceList() []Service {
	if c.Configuration != nil && c.Configuration.ServiceList != nil && len(c.Configuration.ServiceList) > 0 {
		return c.Configuration.ServiceList
	}
	return nil
}
