package common

import (
	"flag"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	configFilePath  = flag.String("c", "", "Config file")
	configFilePath1 = flag.String("config", "", "Config file")
)

func LoadConfig(config interface{}, configFile string) error {
	if !flag.Parsed() {
		flag.Parse()
	}
	if *configFilePath != "" {
		configFile = *configFilePath
	}
	if *configFilePath1 != "" {
		configFile = *configFilePath1
	}
	if !strings.HasSuffix(configFile, ".yaml") {
		configFile += ".yaml"
	}
	Log.Verbose("reading configuration from '%s'...", configFile)
	file, err := os.Open(configFile)
	if err != nil {
		return Errorf("while reading %s: %v", configFile, err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return Errorf("while parse %v Error: %v", configFile, err)
	}
	return nil
}
