/*
 * NEF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"bitbucket.org/free5gc-team/nef/internal/logger"
)

// TODO: Support configuration update from REST api
func InitConfigFactory(f string, cfg *Config) error {
	if f == "" {
		//Use default config path
		f = NEF_CONFIG_PATH
	}

	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Factory] %+v", err)
	} else {
		logger.CfgLog.Infof("Read config from [%s]", f)
		if yamlErr := yaml.Unmarshal([]byte(content), cfg); yamlErr != nil {
			return fmt.Errorf("[Factory] %+v", yamlErr)
		}
	}

	return nil
}

func CheckConfigVersion(cfg *Config) error {
	currentVersion := cfg.GetVersion()
	if currentVersion != NEF_EXPECTED_CONFIG_VERSION {
		return fmt.Errorf("config version is [%s], but expected is [%s].",
			currentVersion, NEF_EXPECTED_CONFIG_VERSION)
	}
	logger.CfgLog.Infof("config version [%s]", currentVersion)

	return nil
}
