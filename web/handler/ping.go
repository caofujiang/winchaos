package handler

import (
	"github.com/sirupsen/logrus"

	"github.com/caofujiang/winchaos/transport"
)

type PingHandler struct {
}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (ph *PingHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Info("Receive server ping request")
	return transport.ReturnSuccess()
}
