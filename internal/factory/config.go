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
	NEF_LOG_PATH       = path_util.Free5gcPath("free5gc/nefsslkey.log")
	NEF_PEM_PATH       = path_util.Free5gcPath("free5gc/support/TLS/nef.pem")
	NEF_KEY_PATH       = path_util.Free5gcPath("free5gc/support/TLS/nef.key")
	NEF_CONFIG_PATH    = path_util.Free5gcPath("free5gc/config/nefcfg.conf")
	NEF_DEFAULT_IPV4   = "127.0.0.1"
	NEF_DEFAULT_PORT   = "29505"
	NEF_DEFAULT_SCHEME = "https"
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
	Sbi             *Sbi      `yaml:"sbi,omitempty"`
	TimeFormat      string    `yaml:"timeFormat,omitempty"`
	DefaultBdtRefId string    `yaml:"defaultBdtRefId,omitempty"`
	NrfUri          string    `yaml:"nrfUri,omitempty"`
	ServiceList     []Service `yaml:"serviceList,omitempty"`
}

type Service struct {
	ServiceName string `yaml:"serviceName"`
	SuppFeat    string `yaml:"suppFeat,omitempty"`
}

type Sbi struct {
	Scheme       string `yaml:"scheme"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"` // IP that is registered at NRF.
	// IPv6Addr  string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"` // IP used to run the server in the node.
	Port        int    `yaml:"port,omitempty"`
}

func (c *Config) GetSbiScheme() string {
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Scheme != "" {
		return c.Configuration.Sbi.Scheme
	}
	return NEF_DEFAULT_SCHEME
}

func (c *Config) GetBindingAddr() string {
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

func (c *Config) GetRegisterAddr() string {
	var regAddr string
	if c.Configuration.Sbi.RegisterIPv4 != "" {
		regAddr = c.Configuration.Sbi.RegisterIPv4 + ":"
	} else {
		regAddr = NEF_DEFAULT_IPV4 + ":"
	}
	if c.Configuration.Sbi.Port != 0 {
		regAddr = regAddr + strconv.Itoa(c.Configuration.Sbi.Port)
	} else {
		regAddr = regAddr + NEF_DEFAULT_PORT
	}
	return regAddr
}

func (c *Config) GetSbiUri() string {
	return c.GetSbiScheme() + "://" + c.GetRegisterAddr()
}
