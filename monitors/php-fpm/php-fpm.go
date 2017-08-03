package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rfjaimes/snmp_agent_extras/commons"
)

var log = commons.GetLogger()

func main() {

	commons.SetBasicLogger()
	log.Info("Starting CPRN snmp subagent")

	config_flag := flag.String("config", "../../conf/config.yaml.in", "Path to config file")
	flag.Parse()

	config, err := commons.LoadConfig(*config_flag)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	commons.ResetLogging(config)

	commons.StartSubAgent(config)

	sm := NewStatsManager()
	sm.Run(60)

	base_oid := "1.3.6.1.4.1.25934.128.4.11.1.1.1"

	stats_handler := NewStatsSNMPHandler(sm, base_oid)

	commons.RegisterSubAgent(base_oid, stats_handler)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
