/*
 * NEF Configuration Factory
 */

package factory

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/davecgh/go-spew/spew"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/models"
	logger_util "bitbucket.org/free5gc-team/util/logger"
)

const (
	NefDefaultTLSKeyLogPath  = "./log/nefsslkey.log"
	NefDefaultTLSPemPath     = "./config/TLS/nef.pem"
	NefDefaultTLSKeyPath     = "./config/TLS/nef.key"
	NefDefaultConfigPath     = "./config/nefcfg.yaml"
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

const (
	ServiceTraffInflu string = "3gpp-traffic-influence"
	ServicePfdMng     string = "3gpp-pfd-management"
	ServiceNefPfd     string = string(models.ServiceName_NNEF_PFDMANAGEMENT)
	ServiceNefOam     string = "nnef-oam"
)

type Config struct {
	Info          *Info               `yaml:"info" valid:"required"`
	Configuration *Configuration      `yaml:"configuration" valid:"required"`
	Logger        *logger_util.Logger `yaml:"logger" valid:"optional"`
	mtx           sync.RWMutex
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
	for i, s := range c.ServiceList {
		switch {
		case s.ServiceName == ServiceNefPfd:
		case s.ServiceName == ServiceNefOam:
		default:
			err := errors.New("Invalid serviceList[" + strconv.Itoa(i) + "]: " +
				s.ServiceName + ", should be nnef-pfdmanagement or nnef-oam")
			return false, appendInvalid(err)
		}
	}
	result, err := govalidator.ValidateStruct(c)
	return result, appendInvalid(err)
}

type Sbi struct {
	Scheme       string `yaml:"scheme" valid:"scheme,required"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty" valid:"host,required"` // IP that is registered at NRF.
	// IPv6Addr  string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty" valid:"host,required"` // IP used to run the server in the node.
	Port        int    `yaml:"port,omitempty" valid:"port,optional"`
	Tls         *Tls   `yaml:"tls,omitempty" valid:"optional"`
}

func (s *Sbi) validate() (bool, error) {
	govalidator.TagMap["scheme"] = govalidator.Validator(func(str string) bool {
		return str == "https" || str == "http"
	})

	if tls := s.Tls; tls != nil {
		if result, err := tls.validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(s)
	return result, appendInvalid(err)
}

type Service struct {
	ServiceName string `yaml:"serviceName"`
	SuppFeat    string `yaml:"suppFeat,omitempty"`
}

type Tls struct {
	Pem string `yaml:"pem,omitempty" valid:"type(string),minstringlength(1),required"`
	Key string `yaml:"key,omitempty" valid:"type(string),minstringlength(1),required"`
}

func (t *Tls) validate() (bool, error) {
	result, err := govalidator.ValidateStruct(t)
	return result, err
}

func appendInvalid(err error) error {
	var errs govalidator.Errors
	if err == nil {
		return nil
	}
	es, ok := err.(govalidator.Errors)
	if ok {
		for _, e := range es.Errors() {
			errs = append(errs, fmt.Errorf("Invalid %w", e))
		}
	} else {
		errs = append(errs, err)
	}
	return error(errs)
}

func (c *Config) Print() {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	spew.Config.Indent = "\t"
	str := spew.Sdump(c.Configuration)
	logger.CfgLog.Infof("==================================================")
	logger.CfgLog.Infof("%s", str)
	logger.CfgLog.Infof("==================================================")
}

func (c *Config) Version() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}

func (c *Config) SbiScheme() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.Sbi.Scheme != "" {
		return c.Configuration.Sbi.Scheme
	}
	return NefSbiDefaultScheme
}

func (c *Config) SbiPort() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.Sbi.Port != 0 {
		return c.Configuration.Sbi.Port
	}
	return NefSbiDefaultPort
}

func (c *Config) SbiBindingIP() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	bindIP := "0.0.0.0"
	if c.Configuration.Sbi.BindingIPv4 != "" {
		if bindIP = os.Getenv(c.Configuration.Sbi.BindingIPv4); bindIP != "" {
			logger.CfgLog.Infof("Parsing ServerIPv4 [%s] from ENV Variable", bindIP)
		} else {
			bindIP = c.Configuration.Sbi.BindingIPv4
		}
	}
	return bindIP
}

func (c *Config) SbiBindingAddr() string {
	return c.SbiBindingIP() + ":" + strconv.Itoa(c.SbiPort())
}

func (c *Config) SbiRegisterIP() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.Sbi.RegisterIPv4 != "" {
		return c.Configuration.Sbi.RegisterIPv4
	}
	return NefSbiDefaultIPv4
}

func (c *Config) SbiRegisterAddr() string {
	return c.SbiRegisterIP() + ":" + strconv.Itoa(c.SbiPort())
}

func (c *Config) SbiUri() string {
	return c.SbiScheme() + "://" + c.SbiRegisterAddr()
}

func (c *Config) NrfUri() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.NrfUri != "" {
		return c.Configuration.NrfUri
	}
	return NefDefaultNrfUri
}

func (c *Config) ServiceList() []Service {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.ServiceList != nil && len(c.Configuration.ServiceList) > 0 {
		return c.Configuration.ServiceList
	}
	return nil
}

func (c *Config) TLSPemPath() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.Sbi.Tls != nil {
		return c.Configuration.Sbi.Tls.Pem
	}
	return NefDefaultTLSPemPath
}

func (c *Config) TLSKeyPath() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.Configuration.Sbi.Tls != nil {
		return c.Configuration.Sbi.Tls.Key
	}
	return NefDefaultTLSKeyPath
}

func (c *Config) NFServices() []models.NfService {
	versions := strings.Split(c.Version(), ".")
	majorVersionUri := "v" + versions[0]
	nfServices := []models.NfService{}
	for i, s := range c.ServiceList() {
		nfService := models.NfService{
			ServiceInstanceId: strconv.Itoa(i),
			ServiceName:       models.ServiceName(s.ServiceName),
			Versions: &[]models.NfServiceVersion{
				{
					ApiFullVersion:  c.Version(),
					ApiVersionInUri: majorVersionUri,
				},
			},
			Scheme:          models.UriScheme(c.SbiScheme()),
			NfServiceStatus: models.NfServiceStatus_REGISTERED,
			ApiPrefix:       c.SbiUri(),
			IpEndPoints: &[]models.IpEndPoint{
				{
					Ipv4Address: c.SbiRegisterIP(),
					Transport:   models.TransportProtocol_TCP,
					Port:        int32(c.SbiPort()),
				},
			},
			SupportedFeatures: s.SuppFeat,
		}
		nfServices = append(nfServices, nfService)
	}
	return nfServices
}

func (c *Config) ServiceUri(name string) string {
	switch name {
	case ServiceTraffInflu:
		return c.SbiUri() + TraffInfluResUriPrefix
	case ServicePfdMng:
		return c.SbiUri() + PfdMngResUriPrefix
	case ServiceNefPfd:
		return c.SbiUri() + NefPfdMngResUriPrefix
	case ServiceNefOam:
		return c.SbiUri() + NefOamResUriPrefix
	default:
		return ""
	}
}
