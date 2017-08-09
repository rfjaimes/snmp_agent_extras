package commons

import (
	"time"

	errgo "gopkg.in/errgo.v1"

	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/value"
)

var client *agentx.Client

func StartSubAgent(config Config) {

	client = &agentx.Client{
		Net:               config.AgentXProtocol,
		Address:           config.AgentXAddress,
		Timeout:           1 * time.Minute,
		ReconnectInterval: 1 * time.Second,
	}

	if err := client.Open(); err != nil {
		log.Fatalf(errgo.Details(err))
	}

}

func RegisterSubAgent(base_oid string, stats_handler agentx.Handler) {

	session, err := client.Session()
	if err != nil {
		log.Fatalf(errgo.Details(err))
	}

	session.Handler = stats_handler

	if err := session.Register(127, value.MustParseOID(base_oid)); err != nil {
		log.Fatalf(errgo.Details(err))
	}

	log.Info(base_oid, "successfully registered")

}
