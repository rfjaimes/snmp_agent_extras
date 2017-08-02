package commons

import (
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	AgentXProtocol  string                       `yaml:"agentx_protocol"`
	AgentXAddress   string                       `yaml:"agentx_address"`
	DiscoverTimeout uint32                       `yaml:"discover_timeout"`
	GetTimeout      uint32                       `yaml:"get_timeout"`
	EndpointAddress string                       `yaml:"endpoint_address"`
	DbFile          string                       `yaml:"db_file"`
	Logging         map[string]map[string]string `yaml:"logging",flow`
}

func DefaultConfig() Config {
	c := Config{
		AgentXProtocol:  "tcp",
		AgentXAddress:   "localhost:705",
		DiscoverTimeout: 500,
		GetTimeout:      300,
		EndpointAddress: "localhost:8080",
		DbFile:          "data/applications.db",
	}
	return c
}

func LoadConfig(path string) (Config, error) {
	yamlstr, err := ioutil.ReadFile(path)
	// Initialize with default values
	yamlparsed := DefaultConfig()
	if err != nil {
		return yamlparsed, err
	}

	err = yaml.Unmarshal(yamlstr, &yamlparsed)
	if err != nil {
		return yamlparsed, err
	}

	err = validateLogging(yamlparsed)
	if err != nil {
		return yamlparsed, err
	}

	return yamlparsed, nil
}

func validateLogging(config Config) error {
	for log_appender, log_appender_info := range config.Logging {
		switch log_appender {
		case "stdout":

		case "file":
			if _, ok := log_appender_info["dir"]; !ok {
				return fmt.Errorf("Missing 'dir' in appender %v", log_appender)
			}
		default:
			return fmt.Errorf("Unsupported log appender '%v'", log_appender)
		}

		err := validateLogLevel(log_appender_info)
		if err != nil {
			return err
		}

		if format, ok := log_appender_info["format"]; ok {
			err = validateLogFormat(format)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func validateLogLevel(log_info map[string]string) error {
	level, ok := log_info["level"]
	if !ok {
		return errors.New("'level' not present in log appender")
	}

	if level == "DEBUG" ||
		level == "INFO" ||
		level == "NOTICE" ||
		level == "WARNING" ||
		level == "ERROR" {
		return nil
	} else {
		return fmt.Errorf("Unknown log level '%v'", level)
	}
}

func validateLogFormat(format string) error {
	if format == "plain" || format == "json" {
		return nil
	} else {
		return fmt.Errorf("Unknown log format '%v'", format)
	}
}
