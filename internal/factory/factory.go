/*
 * NEF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
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
		if yamlErr := yaml.Unmarshal([]byte(content), cfg); yamlErr != nil {
			return fmt.Errorf("[Factory] %+v", yamlErr)
		}
	}

	return nil
}
