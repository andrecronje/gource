package dummy

import (
	"time"

	"github.com/mosaicnetworks/babble/src/proxy/inapp"
	"github.com/sirupsen/logrus"
)

type DummyInappClient struct {
	state  *State
	proxy  *inapp.InappProxy
	logger *logrus.Logger
}

func NewDummyInappClient(logger *logrus.Logger) (*DummyInappClient, error) {
	proxy := inapp.NewInappProxy(1*time.Second, logger)

	state := State{
		stateHash: []byte{},
		snapshots: make(map[int][]byte),
		logger:    logger,
	}
	state.writeMessage([]byte("InappDummy"))

	client := &DummyInappClient{
		state:  &state,
		proxy:  proxy,
		logger: logger,
	}

	go client.Run()

	return client, nil
}

func (c *DummyInappClient) Run() {
	for {
		select {
		case commit := <-c.proxy.CommitCh():
			c.logger.Debug("CommitBlock")
			stateHash, err := c.state.CommitBlock(commit.Block)
			commit.Respond(stateHash, err)
		case snapshotRequest := <-c.proxy.SnapshotRequestCh():
			c.logger.Debug("GetSnapshot")
			snapshot, err := c.state.GetSnapshot(snapshotRequest.BlockIndex)
			snapshotRequest.Respond(snapshot, err)
		case restoreRequest := <-c.proxy.RestoreCh():
			c.logger.Debug("Restore")
			stateHash, err := c.state.Restore(restoreRequest.Snapshot)
			restoreRequest.Respond(stateHash, err)
		}
	}
}

func (c *DummyInappClient) SubmitTx(tx []byte) error {
	return c.proxy.SubmitTx(tx)
}
