package lachesis

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type SocketLachesisProxy struct {
	nodeAddress string
	bindAddress string

	client *SocketLachesisProxyClientWebsocket
	server *SocketLachesisProxyServer
}

func NewSocketLachesisProxy(nodeAddr string,
	bindAddr string,
	timeout time.Duration,
	logger *logrus.Logger) (*SocketLachesisProxy, error) {

	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}

	client := NewSocketLachesisProxyClientWebsocket(nodeAddr, timeout)

	server, err := NewSocketLachesisProxyServer(bindAddr, timeout, logger)

	if err != nil {
		return nil, err
	}

	proxy := &SocketLachesisProxy{
		nodeAddress: nodeAddr,
		bindAddress: bindAddr,
		client:      client,
		server:      server,
	}

	go proxy.server.listen()

	return proxy, nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement LachesisProxy interface

func (p *SocketLachesisProxy) CommitCh() chan Commit {
	return p.server.commitCh
}

func (p *SocketLachesisProxy) SnapshotRequestCh() chan SnapshotRequest {
	return p.server.snapshotRequestCh
}

func (p *SocketLachesisProxy) RestoreCh() chan RestoreRequest {
	return p.server.restoreCh
}

func (p *SocketLachesisProxy) SubmitTx(tx []byte) error {
	err := p.client.SubmitTx(tx)
	if err != nil {
		return fmt.Errorf("Failed to deliver transaction to Lachesis: %v", err)
	}

	return nil
}