package commons

import (
	"reflect"
	"testing"
)

func run_config_test(path string, expected Config, t *testing.T) {
	conf, err := LoadConfig(path)
	if err != nil {
		t.Error("Error loading config:", err)
	} else {
		if !reflect.DeepEqual(conf, expected) {
			t.Errorf("Error loading config.\nExpected: %+v\n Got: %+v\n", expected, conf)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	conf_file := "../test/config/good_conf.yaml"

	expected_conf := Config{
		AgentXProtocol:  "udp",
		AgentXAddress:   "127.1.1.5:890",
		DiscoverTimeout: 123,
		GetTimeout:      456,
		EndpointAddress: "0.0.0.0:11111",
		DbFile:          "data/test.db",
		Logging: map[string]map[string]string{
			"stdout": map[string]string{"level": "DEBUG", "format": "json"},
			"file":   map[string]string{"level": "INFO", "format": "plain", "dir": "/var/log/iway/snmp-subagent"},
		},
	}

	run_config_test(conf_file, expected_conf, t)
}

func TestLoadEmptyConfig(t *testing.T) {
	conf_file := "../test/config/empty_config.yaml"

	expected_conf := Config{
		AgentXProtocol:  "tcp",
		AgentXAddress:   "localhost:705",
		DiscoverTimeout: 500,
		GetTimeout:      300,
		EndpointAddress: "localhost:8080",
		DbFile:          "data/applications.db",
	}

	run_config_test(conf_file, expected_conf, t)
}

func TestLoadSomeValues(t *testing.T) {
	conf_file := "../test/config/some_vals.yaml"

	expected_conf := Config{
		AgentXProtocol:  "udp",
		AgentXAddress:   "localhost:705",
		DiscoverTimeout: 500,
		GetTimeout:      456,
		EndpointAddress: "localhost:8080",
		DbFile:          "data/applications.db",
	}

	run_config_test(conf_file, expected_conf, t)
}

func TestUnexistingFile(t *testing.T) {
	conf_file := "../test/config/does_not_exist.yaml"

	_, err := LoadConfig(conf_file)
	if err == nil {
		t.Error("Should have raised an error")
	}
}

func TestWrongFormat(t *testing.T) {
	conf_file := "../test/config/something.conf"

	_, err := LoadConfig(conf_file)
	if err == nil {
		t.Error("Shouldn't have loaded a conf file")
	}
}

func TestLoadWrongLogging(t *testing.T) {
	conf_files := []string{
		"../test/config/wrong_logging1.yaml",
		"../test/config/wrong_logging2.yaml",
		"../test/config/wrong_logging3.yaml",
		"../test/config/wrong_logging4.yaml",
	}

	for _, conf_file := range conf_files {
		_, err := LoadConfig(conf_file)
		if err == nil {
			t.Error("Shouldn't have loaded a conf file")
		}
	}
}

func TestValidateLogging(t *testing.T) {
	test_data := []struct {
		conf           Config
		error_expected bool
	}{
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "DEBUG"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "INFO"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "NOTICE"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "WARNING"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "ERROR"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "CRITICAL"}}}, true},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{}}}, true},
		{Config{Logging: map[string]map[string]string{"file": map[string]string{"level": "INFO", "dir": "/var/log"}}}, false},
		{Config{Logging: map[string]map[string]string{"file": map[string]string{"level": "INFO"}}}, true},
		{Config{Logging: map[string]map[string]string{"file": map[string]string{"dir": "/var/log"}}}, true},
		{Config{Logging: map[string]map[string]string{"kafka": map[string]string{"level": "INFO"}}}, true},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "DEBUG", "format": "json"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "DEBUG", "format": "plain"}}}, false},
		{Config{Logging: map[string]map[string]string{"stdout": map[string]string{"level": "DEBUG", "format": "yaml"}}}, true},
	}

	for _, test_row := range test_data {
		err := validateLogging(test_row.conf)
		if err != nil && !test_row.error_expected {
			t.Errorf("Error was not expected but got %v", err)
		} else if err == nil && test_row.error_expected {
			t.Errorf("Error was expected but got nil")
		}
	}
}
