package app

import (
	"time"

	"github.com/andrecronje/lachesis/src/log"
	"github.com/andrecronje/lachesis/src/poset"
	"github.com/sirupsen/logrus"
)

type SocketAppProxy struct {
	clientAddress string
	bindAddress   string

	client *SocketAppProxyClient
	server *SocketAppProxyWebsocketServer

	logger *logrus.Logger
}

func NewSocketAppProxy(clientAddr string, bindAddr string, timeout time.Duration, logger *logrus.Logger) (*SocketAppProxy, error) {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
		lachesis_log.NewLocal(logger, logger.Level.String())
	}

	client := NewSocketAppProxyClient(clientAddr, timeout, logger)

	server, err := NewSocketAppProxyWebsocketServer(bindAddr, logger)

	if err != nil {
		return nil, err
	}

	proxy := &SocketAppProxy{
		clientAddress: clientAddr,
		bindAddress:   bindAddr,
		client:        client,
		server:        server,
		logger:        logger,
	}

	return proxy, nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement AppProxy Interface

func (p *SocketAppProxy) SubmitCh() chan []byte {
	return p.server.submitCh
}

func (p *SocketAppProxy) CommitBlock(block poset.Block) ([]byte, error) {
	return p.client.CommitBlock(block)
}

func (p *SocketAppProxy) GetSnapshot(blockIndex int) ([]byte, error) {
	return p.client.GetSnapshot(blockIndex)
}

func (p *SocketAppProxy) Restore(snapshot []byte) error {
	return p.client.Restore(snapshot)
}
