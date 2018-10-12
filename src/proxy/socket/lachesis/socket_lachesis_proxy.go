package lachesis

import (
	"fmt"
	"time"

	"github.com/andrecronje/lachesis/src/proxy/socket/proto"
	"github.com/sirupsen/logrus"
)

type SocketLachesisProxy struct {
	nodeAddress string
	bindAddress string

	handler proxy.ProxyHandler

	client *SocketLachesisProxyClient
	server *SocketLachesisProxyServer
}

func NewSocketLachesisProxy(
	nodeAddr string,
	bindAddr string,
	handler proxy.ProxyHandler,
	timeout time.Duration,
	logger *logrus.Logger,
) (*SocketLachesisProxy, error) {

	if logger == nil {
		logger = logrus.New()

		logger.Level = logrus.DebugLevel
	}

	client := NewSocketLachesisProxyClient(nodeAddr, timeout)

	server, err := NewSocketLachesisProxyServer(bindAddr, handler, timeout, logger)

	if err != nil {
		return nil, err
	}

	proxy := &SocketLachesisProxy{
		nodeAddress: nodeAddr,
		bindAddress: bindAddr,
		handler:     handler,
		client:      client,
		server:      server,
	}

	go proxy.server.listen()

	return proxy, nil
}

func (p *SocketLachesisProxy) SubmitTx(tx []byte) error {
	ack, err := p.client.SubmitTx(tx)

	if err != nil {
		return err
	}

	if !*ack {
		return fmt.Errorf("Failed to deliver transaction to Lachesis")
	}

	return nil
}
