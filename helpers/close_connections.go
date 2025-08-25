package helpers

import (
	"github.com/retawsolit/WeMeet-recorder/pkg/config"
	"github.com/sirupsen/logrus"
)

func HandleCloseConnections() {
	if config.GetConfig() == nil {
		return
	}
	// close nats
	_ = config.GetConfig().NatsConn.Drain()
	config.GetConfig().NatsConn.Close()

	// close logger
	logrus.Exit(0)
}
