package commons

import (
	"os"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("sentinel-snmp-subagent")
var plain_format = logging.MustStringFormatter(
	`%{time:2006-02-01 15:04:05.000} %{level:.4s} %{message}`,
)

var json_format = logging.MustStringFormatter(
	`{"timestamp": "%{time:2006-02-01 15:04:05.000}", "severity": "%{level:.4s}", "message": "%{message}"}`,
)

func SetBasicLogger() {
	logging.SetFormatter(plain_format)
	stdout_backend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(stdout_backend)
}

func ResetLogging(config Config) {
	backends := make([]logging.Backend, 0)
	for log_appender, log_appender_info := range config.Logging {
		format := plain_format
		if log_format, ok := log_appender_info["format"]; ok {
			if log_format == "json" {
				format = json_format
			}
		}

		level := logging.INFO
		if log_level, ok := log_appender_info["level"]; ok {
			level, _ = logging.LogLevel(log_level)
		}
		var backend logging.Backend
		switch log_appender {
		case "stdout":
			backend = logging.NewLogBackend(os.Stdout, "", 0)
		case "file":
			log_file := log_appender_info["dir"] + "/snmp-subagent.log"
			if file_writter, err := os.OpenFile(log_file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
				log.Warningf("Couldn't create appender 'file': %v", err)
				continue
			} else {
				backend = logging.NewLogBackend(file_writter, "", 0)
			}
		}
		backend_formatter := logging.NewBackendFormatter(backend, format)
		backend_leveled := logging.AddModuleLevel(backend_formatter)
		backend_leveled.SetLevel(level, "")

		backends = append(backends, backend_leveled)
	}

	logging.SetBackend(backends...)
}
