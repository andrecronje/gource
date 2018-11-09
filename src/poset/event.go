package poset

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/andrecronje/lachesis/src/crypto"
	"github.com/andrecronje/lachesis/src/peers"
)

/*******************************************************************************
InternalTransactions
*******************************************************************************/
type TransactionType uint8

const (
	PEER_ADD TransactionType = iota
	PEER_REMOVE
)

type InternalTransaction struct {
	Type TransactionType
	Peer peers.Peer
}

func NewInternalTransaction(tType TransactionType, peer peers.Peer) InternalTransaction {
	return InternalTransaction{
		Type: tType,
		Peer: peer,
	}
}

//json encoding of body only
func (t *InternalTransaction) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(t); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func (t *InternalTransaction) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(t); err != nil {
		return err
	}
	return nil
}

/*******************************************************************************
EventBody
*******************************************************************************/

type EventBody struct {
	Transactions         [][]byte              // the payload
	InternalTransactions []InternalTransaction //peers add and removal internal consensus
	Parents              []string              // hashes of the event's parents, self-parent first
	Creator              []byte                // creator's public key
	Index                int64                   // index in the sequence of events created by Creator
	BlockSignatures      []BlockSignature      // list of Block signatures signed by the Event's Creator ONLY

	// wire
	// It is cheaper to send ints than hashes over the wire
	selfParentIndex       int64
	otherParentCreatorIDs []int64
	otherParentIndexes    []int64
	creatorID             int64
}

// json encoding of body only
func (e *EventBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) // will write to b
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *EventBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) // will read from b
	if err := dec.Decode(e); err != nil {
		return err
	}
	return nil
}

func (e *EventBody) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

/*******************************************************************************
Event
*******************************************************************************/

type Event struct {
	Body      EventBody
	Signature string // creator's digital signature of body

	topologicalIndex int64

	// used for sorting
	round            *int64
	lamportTimestamp *int64

	roundReceived *int64

	creator string
	hash    []byte
	hex     string

	// FlagTable stores connection information.
	FlagTable []byte

	// If the event is a witness, then stores the roots that it sees.
	WitnessProof []string
}

// NewEvent creates new block event.
func NewEvent(transactions [][]byte,
	internalTransactions []InternalTransaction,
	blockSignatures []BlockSignature,
	parents []string, creator []byte, index int64,
	flagTable map[string]int64) Event {

	body := EventBody{
		Transactions:         transactions,
		InternalTransactions: internalTransactions,
		BlockSignatures:      blockSignatures,
		Parents:              parents,
		Creator:              creator,
		Index:                index,
	}

	ft, _ := json.Marshal(flagTable)

	return Event{
		Body:      body,
		FlagTable: ft,
	}
}

// Round returns round of event.
func (e *Event) Round() int64 {
	if e.round == nil || *e.round < 0 {
		return -1
	}
	return *e.round
}

func (e *Event) Creator() string {
	if e.creator == "" {
		e.creator = fmt.Sprintf("0x%X", e.Body.Creator)
	}
	return e.creator
}

func (e *Event) SelfParent() string {
	if len(e.Body.Parents) > 0 {
		return e.Body.Parents[0]
	}
	return ""
}

func (e *Event) OtherParent(n int) string {
	if n >= 0 && n < len(e.Body.Parents)-1 {
		return e.Body.Parents[1:][n]
	}
	return ""
}

func (e *Event) OtherParents() []string {
	if len(e.Body.Parents) > 0 {
		return e.Body.Parents[1:]
	}
	return []string{}
}

func (e *Event) Transactions() [][]byte {
	return e.Body.Transactions
}

func (e *Event) Index() int64 {
	return e.Body.Index
}

func (e *Event) BlockSignatures() []BlockSignature {
	return e.Body.BlockSignatures
}

// True if Event contains a payload or is the initial Event of its creator
func (e *Event) IsLoaded() bool {
	if e.Body.Index == 0 {
		return true
	}

	hasTransactions := e.Body.Transactions != nil &&
		(len(e.Body.Transactions) > 0 || len(e.Body.InternalTransactions) > 0)

	return hasTransactions
}

// ecdsa sig
func (e *Event) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := e.Body.Hash()
	if err != nil {
		return err
	}
	R, S, err := crypto.Sign(privKey, signBytes)
	if err != nil {
		return err
	}
	e.Signature = crypto.EncodeSignature(R, S)
	return err
}

func (e *Event) Verify() (bool, error) {
	pubBytes := e.Body.Creator
	pubKey := crypto.ToECDSAPub(pubBytes)

	signBytes, err := e.Body.Hash()
	if err != nil {
		return false, err
	}

	r, s, err := crypto.DecodeSignature(e.Signature)
	if err != nil {
		return false, err
	}

	return crypto.Verify(pubKey, signBytes, r, s), nil
}

// json encoding of body and signature
func (e *Event) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *Event) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) // will read from b
	return dec.Decode(e)
}

// sha256 hash of body
func (e *Event) Hash() ([]byte, error) {
	if len(e.hash) == 0 {
		hash, err := e.Body.Hash()
		if err != nil {
			return nil, err
		}
		e.hash = hash
	}
	return e.hash, nil
}

func (e *Event) Hex() string {
	if e.hex == "" {
		hash, _ := e.Hash()
		e.hex = fmt.Sprintf("0x%X", hash)
	}
	return e.hex
}

func (e *Event) SetRound(r int64) {
	if e.round == nil {
		e.round = new(int64)
	}
	*e.round = r
}

func (e *Event) SetLamportTimestamp(t int64) {
	if e.lamportTimestamp == nil {
		e.lamportTimestamp = new(int64)
	}
	*e.lamportTimestamp = t
}

func (e *Event) SetRoundReceived(rr int64) {
	if e.roundReceived == nil {
		e.roundReceived = new(int64)
	}
	*e.roundReceived = rr
}

func (e *Event) SetWireInfo(selfParentIndex int64,
	otherParentCreatorIDs []int64,
	otherParentIndexes []int64,
	creatorID int64) {
	e.Body.selfParentIndex = selfParentIndex
	e.Body.otherParentCreatorIDs = otherParentCreatorIDs
	e.Body.otherParentIndexes = otherParentIndexes
	e.Body.creatorID = creatorID
}

func (e *Event) WireBlockSignatures() []WireBlockSignature {
	if e.Body.BlockSignatures != nil {
		wireSignatures := make([]WireBlockSignature, len(e.Body.BlockSignatures))
		for i, bs := range e.Body.BlockSignatures {
			wireSignatures[i] = bs.ToWire()
		}

		return wireSignatures
	}
	return nil
}

func (e *Event) ToWire() WireEvent {

	return WireEvent{
		Body: WireBody{
			Transactions:          e.Body.Transactions,
			InternalTransactions:  e.Body.InternalTransactions,
			SelfParentIndex:       e.Body.selfParentIndex,
			OtherParentCreatorIDs: e.Body.otherParentCreatorIDs,
			OtherParentIndexes:    e.Body.otherParentIndexes,
			CreatorID:             e.Body.creatorID,
			Index:                 e.Body.Index,
			BlockSignatures:       e.WireBlockSignatures(),
		},
		Signature:    e.Signature,
		FlagTable:    e.FlagTable,
		WitnessProof: e.WitnessProof,
	}
}

// ReplaceFlagTable replaces flag table.
func (e *Event) ReplaceFlagTable(flagTable map[string]int) (err error) {
	e.FlagTable, err = json.Marshal(flagTable)
	return err
}

// GetFlagTable returns the flag table.
func (e *Event) GetFlagTable() (result map[string]int64, err error) {
	result = make(map[string]int64)
	err = json.Unmarshal(e.FlagTable, &result)
	return result, err
}

// MergeFlagTable returns merged flag table object.
func (e *Event) MergeFlagTable(
	dst map[string]int64) (result map[string]int64, err error) {
	src := make(map[string]int64)
	if err := json.Unmarshal(e.FlagTable, &src); err != nil {
		return nil, err
	}

	for id, flag := range dst {
		if src[id] == 0 && flag == 1 {
			src[id] = 1
		}
	}
	return src, err
}

func rootSelfParent(participantID int64) string {
	return fmt.Sprintf("Root%d", participantID)
}

/*******************************************************************************
Sorting
*******************************************************************************/

// ByTopologicalOrder implements sort.Interface for []Event based on
// the topologicalIndex field.
// THIS IS A PARTIAL ORDER
type ByTopologicalOrder []Event

func (a ByTopologicalOrder) Len() int      { return len(a) }
func (a ByTopologicalOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTopologicalOrder) Less(i, j int) bool {
	return a[i].topologicalIndex < a[j].topologicalIndex
}

// ByLamportTimestamp implements sort.Interface for []Event based on
// the lamportTimestamp field.
// THIS IS A TOTAL ORDER
type ByLamportTimestamp []Event

func (a ByLamportTimestamp) Len() int      { return len(a) }
func (a ByLamportTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLamportTimestamp) Less(i, j int) bool {
	it, jt := int64(-1), int64(-1)
	if a[i].lamportTimestamp != nil {
		it = *a[i].lamportTimestamp
	}
	if a[j].lamportTimestamp != nil {
		jt = *a[j].lamportTimestamp
	}
	if it != jt {
		return it < jt
	}

	wsi, _, _ := crypto.DecodeSignature(a[i].Signature)
	wsj, _, _ := crypto.DecodeSignature(a[j].Signature)
	return wsi.Cmp(wsj) < 0
}

/*******************************************************************************
 WireEvent
*******************************************************************************/

type WireBody struct {
	Transactions         [][]byte
	InternalTransactions []InternalTransaction
	BlockSignatures      []WireBlockSignature

	SelfParentIndex       int64
	OtherParentCreatorIDs []int64
	OtherParentIndexes    []int64
	CreatorID             int64

	Index int64
}

type WireEvent struct {
	Body         WireBody
	Signature    string
	FlagTable    []byte
	WitnessProof []string
}

func (we *WireEvent) BlockSignatures(validator []byte) []BlockSignature {
	if we.Body.BlockSignatures != nil {
		blockSignatures := make([]BlockSignature, len(we.Body.BlockSignatures))
		for k, bs := range we.Body.BlockSignatures {
			blockSignatures[k] = BlockSignature{
				Validator: validator,
				Index:     bs.Index,
				Signature: bs.Signature,
			}
		}
		return blockSignatures
	}
	return nil
}
