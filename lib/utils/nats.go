package utils

import (
	"time"

	"github.com/nats-io/nats.go"
	"logur.dev/logur"
)

type NatsConf struct {
	URL     string
	Subject string
}

func NewNatsConn(conf NatsConf, logger logur.KVLoggerFacade) (*nats.Conn, error) {
	// connect options
	logger.Info("--- Connecting to Nats ---")
	opts := []nats.Option{nats.Name("nats conn")}
	opts = setupConnOptions(opts, logger)

	nc, err := nats.Connect(conf.URL, opts...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func setupConnOptions(opts []nats.Option, logger logur.KVLoggerFacade) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		if !nc.IsClosed() {
			logger.Info("Disconnected due to: ", "Error", err, "Attempt to reconnect in minute", totalWait.Minutes())
		}
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		logger.Info("Reconnected", "Url", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		if !nc.IsClosed() {
			logger.Error("Exiting: no servers available")
		} else {
			logger.Error("Exiting")
		}
	}))
	return opts
}
