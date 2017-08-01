package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"

	errgo "gopkg.in/errgo.v1"

	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/value"
	"github.com/rfjaimes/snmp_agent_extras/cpnr_monitor/stats"
)

var log = logging.MustGetLogger("iostat_monitor")
var format = logging.MustStringFormatter(
	`%{time:2006-02-01 15:04:05.000} %{level:.4s} %{message}`,
)

func main() {
	logging.SetFormatter(format)
	log_out_backend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(log_out_backend)

	log.Info("Running iostat monitor")

	client := &agentx.Client{
		Net:               "tcp",
		Address:           "localhost:705",
		Timeout:           1 * time.Minute,
		ReconnectInterval: 1 * time.Second,
	}

	if err := client.Open(); err != nil {
		log.Fatalf(errgo.Details(err))
	}

	session, err := client.Session()
	if err != nil {
		log.Fatalf(errgo.Details(err))
	}

	sm := stats.NewStatsManager()
	sm.Run(60)

	base_oid := "1.3.6.1.4.1.25934.128.1.11.1.1"

	stats_handler := stats.NewStatsSNMPHandler(sm, base_oid)

	session.Handler = stats_handler

	if err := session.Register(127, value.MustParseOID(base_oid)); err != nil {
		log.Fatalf(errgo.Details(err))
	}

	log.Info(base_oid, "successfully registered")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
