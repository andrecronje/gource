package dummy

import (
	"time"

  "github.com/andrecronje/lachesis/src/dummy/state"
  "github.com/andrecronje/lachesis/src/proxy/inapp"
	"github.com/sirupsen/logrus"
)

// DummyInappClient is an in-memory implmentation of the dummy app. It actually
// imlplements the AppProxy interface, and can be passed in the Lachesis
// constructor directly
type DummyInappClient struct {
	*inapp.InappProxy
	state  *state.State
	logger *logrus.Logger
}

//NewDummyInappClient instantiates a DummyInappClient
func NewDummyInappClient(logger *logrus.Logger) *DummyInappClient {
	state := state.NewState(logger)

	proxy := inapp.NewInappProxy(1*time.Second, logger)

  client := &DummyInappClient{
		InappProxy: proxy,
		state:      state,
		logger:     logger,
	}
 	go client.Run()
 	return client
}

//Run listens for messages from the Lachesis node via the InappProxy
func (c *DummyInappClient) Run() {
	for {
		select {
		case commit := <-c.InappProxy.CommitCh():
			c.logger.Debug("CommitBlock")
			stateHash, err := c.state.CommitHandler(commit.Block)
			commit.Respond(stateHash, err)
		case snapshotRequest := <-c.InappProxy.SnapshotRequestCh():
			c.logger.Debug("GetSnapshot")
			snapshot, err := c.state.SnapshotHandler(snapshotRequest.BlockIndex)
			snapshotRequest.Respond(snapshot, err)
		case restoreRequest := <-c.InappProxy.RestoreCh():
			c.logger.Debug("Restore")
			stateHash, err := c.state.RestoreHandler(restoreRequest.Snapshot)
			restoreRequest.Respond(stateHash, err)
		}
	}
}

//SubmitTx sends a transaction to the Lachesis node via the InappProxy
func (c *DummyInappClient) SubmitTx(tx []byte) error {
	return c.InappProxy.SubmitTx(tx)
}

//GetCommittedTransactions returns the state's list of transactions
func (c *DummyInappClient) GetCommittedTransactions() [][]byte {
	return c.state.GetCommittedTransactions()
}
