package node

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/andrecronje/lachesis/crypto"
	"github.com/andrecronje/lachesis/poset"
)

type Core struct {
	id                  int
	key                 *ecdsa.PrivateKey
	pubKey              []byte
	hexID               string
	poset               *poset.Poset

	participants        map[string]int //[PubKey] => id
	reverseParticipants map[int]string //[id] => PubKey
	Head                string
	Seq                 int

	transactionPool    [][]byte
	blockSignaturePool []poset.BlockSignature

	logger *logrus.Entry
}

func NewCore(
	id int,
	key *ecdsa.PrivateKey,
	participants map[string]int,
	store poset.Store,
	commitCh chan poset.Block,
	logger *logrus.Logger) Core {

	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	logEntry := logger.WithField("id", id)

	reverseParticipants := make(map[int]string)
	for pk, id := range participants {
		reverseParticipants[id] = pk
	}

	core := Core{
		id:                  id,
		key:                 key,
		poset:               poset.NewPoset(participants, store, commitCh, logEntry),
		participants:        participants,
		reverseParticipants: reverseParticipants,
		transactionPool:     [][]byte{},
		blockSignaturePool:  []poset.BlockSignature{},
		logger:              logEntry,
		Head:                "",
		Seq:                 -1,
	}
	return core
}

func (c *Core) ID() int {
	return c.id
}

func (c *Core) PubKey() []byte {
	if c.pubKey == nil {
		c.pubKey = crypto.FromECDSAPub(&c.key.PublicKey)
	}
	return c.pubKey
}

func (c *Core) HexID() string {
	if c.hexID == "" {
		pubKey := c.PubKey()
		c.hexID = fmt.Sprintf("0x%X", pubKey)
	}
	return c.hexID
}

func (c *Core) SetHeadAndSeq() error {

	var head string
	var seq int

	last, isRoot, err := c.poset.Store.LastEventFrom(c.HexID())
	if err != nil {
		return err
	}

	if isRoot {
		root, err := c.poset.Store.GetRoot(c.HexID())
		if err != nil {
			return err
		}
		head = root.SelfParent.Hash
		seq = root.SelfParent.Index
	} else {
		lastEvent, err := c.GetEvent(last)
		if err != nil {
			return err
		}
		head = last
		seq = lastEvent.Index()
	}

	c.Head = head
	c.Seq = seq

	c.logger.WithFields(logrus.Fields{
		"core.Head": c.Head,
		"core.Seq":  c.Seq,
		"is_root":   isRoot,
	}).Debugf("SetHeadAndSeq")

	return nil
}

func (c *Core) Bootstrap() error {
	return c.poset.Bootstrap()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) SignAndInsertSelfEvent(event poset.Event) error {
	if err := event.Sign(c.key); err != nil {
		return err
	}
	if err := c.InsertEvent(event, true); err != nil {
		return err
	}
	return nil
}

func (c *Core) InsertEvent(event poset.Event, setWireInfo bool) error {
	if err := c.poset.InsertEvent(event, setWireInfo); err != nil {
		return err
	}
	if event.Creator() == c.HexID() {
		c.Head = event.Hex()
		c.Seq = event.Index()
	}
	return nil
}

func (c *Core) KnownEvents() map[int]int {
	return c.poset.Store.KnownEvents()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) SignBlock(block poset.Block) (poset.BlockSignature, error) {
	sig, err := block.Sign(c.key)
	if err != nil {
		return poset.BlockSignature{}, err
	}
	if err := block.SetSignature(sig); err != nil {
		return poset.BlockSignature{}, err
	}
	return sig, c.poset.Store.SetBlock(block)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func (c *Core) OverSyncLimit(knownEvents map[int]int, syncLimit int) bool {
	totUnknown := 0
	myKnownEvents := c.KnownEvents()
	for i, li := range myKnownEvents {
		if li > knownEvents[i] {
			totUnknown += li - knownEvents[i]
		}
	}
	if totUnknown > syncLimit {
		return true
	}
	return false
}

func (c *Core) GetAnchorBlockWithFrame() (poset.Block, poset.Frame, error) {
	return c.poset.GetAnchorBlockWithFrame()
}

//returns events that c knows about and are not in 'known'
func (c *Core) EventDiff(known map[int]int) (events []poset.Event, err error) {
	unknown := []poset.Event{}
	//known represents the index of the last event known for every participant
	//compare this to our view of events and fill unknown with events that we know of
	// and the other doesn't
	for id, ct := range known {
		pk := c.reverseParticipants[id]
		//get participant Events with index > ct
		participantEvents, err := c.poset.Store.ParticipantEvents(pk, ct)
		if err != nil {
			return []poset.Event{}, err
		}
		for _, e := range participantEvents {
			ev, err := c.poset.Store.GetEvent(e)
			if err != nil {
				return []poset.Event{}, err
			}
			unknown = append(unknown, ev)
		}
	}
	sort.Sort(poset.ByTopologicalOrder(unknown))

	return unknown, nil
}

func (c *Core) Sync(unknownEvents []poset.WireEvent) error {

	c.logger.WithFields(logrus.Fields{
		"unknown_events":       len(unknownEvents),
		"transaction_pool":     len(c.transactionPool),
		"block_signature_pool": len(c.blockSignaturePool),
	}).Debug("Sync")

	otherHead := ""
	//add unknown events
	for k, we := range unknownEvents {
		ev, err := c.poset.ReadWireInfo(we)
		if err != nil {
			c.logger.WithField("WireEvent", we).Errorf("ReadingWireInfo")
			return err

		}
		if err := c.InsertEvent(*ev, false); err != nil {
			return err
		}
		//assume last event corresponds to other-head
		if k == len(unknownEvents)-1 {
			otherHead = ev.Hex()
		}
	}

	//create new event with self head and other head only if there are pending
	//loaded events or the pools are not empty
	return c.AddSelfEvent(otherHead)
}

func (c *Core) FastForward(peer string, block poset.Block, frame poset.Frame) error {

	//Check Block Signatures
	err := c.poset.CheckBlock(block)
	if err != nil {
		return err
	}

	//Check Frame Hash
	frameHash, err := frame.Hash()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(block.FrameHash(), frameHash) {
		return fmt.Errorf("Invalid Frame Hash")
	}

	err = c.poset.Reset(block, frame)
	if err != nil {
		return err
	}

	err = c.SetHeadAndSeq()
	if err != nil {
		return err
	}

	// lastEventFromPeer, _, err := c.poset.Store.LastEventFrom(peer)
	// if err != nil {
	// 	return err
	// }

	// err = c.AddSelfEvent(lastEventFromPeer)
	// if err != nil {
	// 	return err
	// }

	err = c.RunConsensus()
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) AddSelfEvent(otherHead string) error {

	//exit if there is nothing to record
	if otherHead == "" && len(c.transactionPool) == 0 && len(c.blockSignaturePool) == 0 {
		c.logger.Debug("Empty transaction pool and block signature pool")
		return nil
	}

	// Get flag tables from parents
	parentEvent,err := c.poset.Store.GetEvent(c.Head)
	if err != nil {
		return fmt.Errorf("Error retrieving parent: %s", err)
	}
	otherParentEvent,err := c.poset.Store.GetEvent(otherHead)
	if err != nil {
		return fmt.Errorf("Error retrieving parent: %s", err)
	}
	flagTable, flags := parentEvent.FlagTable()
	otherFlagTable,_ := otherParentEvent.FlagTable()
	// event flag table = parent 1 flag table OR parent 2 flag table
	for id, flag := range  otherFlagTable{
		if !flagTable[id] && flag {
			flagTable[id] = true
			flags++
		}
	}

	//create new event with self head and empty other parent
	//empty transaction pool in its payload
	newHead := poset.NewEvent(c.transactionPool,
		c.blockSignaturePool,
		[]string{c.Head, otherHead},
		c.PubKey(), c.Seq+1,
		flagTable, flags)

	if err := c.SignAndInsertSelfEvent(newHead); err != nil {
		return fmt.Errorf("error inserting new head: %s", err)
	}

	c.logger.WithFields(logrus.Fields{
		"transactions":     len(c.transactionPool),
		"block_signatures": len(c.blockSignaturePool),
	}).Debug("Created Self-Event")

	c.transactionPool = [][]byte{}
	c.blockSignaturePool = []poset.BlockSignature{}

	return nil
}

func (c *Core) FromWire(wireEvents []poset.WireEvent) ([]poset.Event, error) {
	events := make([]poset.Event, len(wireEvents), len(wireEvents))
	for i, w := range wireEvents {
		ev, err := c.poset.ReadWireInfo(w)
		if err != nil {
			return nil, err
		}
		events[i] = *ev
	}
	return events, nil
}

func (c *Core) ToWire(events []poset.Event) ([]poset.WireEvent, error) {
	wireEvents := make([]poset.WireEvent, len(events), len(events))
	for i, e := range events {
		wireEvents[i] = e.ToWire()
	}
	return wireEvents, nil
}

func (c *Core) RunConsensus() error {
	start := time.Now()
	err := c.poset.DivideRounds()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DivideRounds()")
	if err != nil {
		c.logger.WithField("error", err).Error("DivideRounds")
		return err
	}

	start = time.Now()
	err = c.poset.DecideFame()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DecideFame()")
	if err != nil {
		c.logger.WithField("error", err).Error("DecideFame")
		return err
	}

	start = time.Now()
	err = c.poset.DecideRoundReceived()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("DecideRoundReceived()")
	if err != nil {
		c.logger.WithField("error", err).Error("DecideRoundReceived")
		return err
	}

	start = time.Now()
	err = c.poset.ProcessDecidedRounds()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("ProcessDecidedRounds()")
	if err != nil {
		c.logger.WithField("error", err).Error("ProcessDecidedRounds")
		return err
	}

	start = time.Now()
	err = c.poset.ProcessSigPool()
	c.logger.WithField("duration", time.Since(start).Nanoseconds()).Debug("ProcessSigPool()")
	if err != nil {
		c.logger.WithField("error", err).Error("ProcessSigPool()")
		return err
	}

	return nil
}

func (c *Core) AddTransactions(txs [][]byte) {
	c.transactionPool = append(c.transactionPool, txs...)
}

func (c *Core) AddBlockSignature(bs poset.BlockSignature) {
	c.blockSignaturePool = append(c.blockSignaturePool, bs)
}

func (c *Core) GetHead() (poset.Event, error) {
	return c.poset.Store.GetEvent(c.Head)
}

func (c *Core) GetEvent(hash string) (poset.Event, error) {
	return c.poset.Store.GetEvent(hash)
}

func (c *Core) GetEventTransactions(hash string) ([][]byte, error) {
	var txs [][]byte
	ex, err := c.GetEvent(hash)
	if err != nil {
		return txs, err
	}
	txs = ex.Transactions()
	return txs, nil
}

func (c *Core) GetConsensusEvents() []string {
	return c.poset.Store.ConsensusEvents()
}

func (c *Core) GetConsensusEventsCount() int {
	return c.poset.Store.ConsensusEventsCount()
}

func (c *Core) GetUndeterminedEvents() []string {
	return c.poset.UndeterminedEvents
}

func (c *Core) GetPendingLoadedEvents() int {
	return c.poset.PendingLoadedEvents
}

func (c *Core) GetConsensusTransactions() ([][]byte, error) {
	txs := [][]byte{}
	for _, e := range c.GetConsensusEvents() {
		eTxs, err := c.GetEventTransactions(e)
		if err != nil {
			return txs, fmt.Errorf("consensus event not found: %s", e)
		}
		txs = append(txs, eTxs...)
	}
	return txs, nil
}

func (c *Core) GetLastConsensusRoundIndex() *int {
	return c.poset.LastConsensusRound
}

func (c *Core) GetConsensusTransactionsCount() int {
	return c.poset.ConsensusTransactions
}

func (c *Core) GetLastCommittedRoundEventsCount() int {
	return c.poset.LastCommitedRoundEvents
}

func (c *Core) GetLastBlockIndex() int {
	return c.poset.Store.LastBlockIndex()
}

func (c *Core) NeedGossip() bool {
	return c.poset.PendingLoadedEvents > 0 ||
		len(c.transactionPool) > 0 ||
		len(c.blockSignaturePool) > 0
}
