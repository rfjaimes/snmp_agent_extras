package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"gitlab.intraway.com/sentinel/sentinel-snmp-subagent/snmp_subagent"
)

func main() {
	SetBasicLogger()
	log.Info("Starting sentinel snmp subagent")
	config_flag := flag.String("config", "conf/config.yaml", "Path to config file")
	flag.Parse()

	config, err := snmp_subagent.LoadConfig(*config_flag)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	ResetLogging(config)

	subagent := snmp_subagent.NewSNMPSubagent()
	log.Info("Initializing...")
	subagent.Initialize(config)

	log.Info("Running sentinel snmp subagent")

	subagent.Run()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Info("Exiting sentinel snmp subagent")
}
