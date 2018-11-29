package node

import (
	"math/rand"
	"sync"

	"github.com/Fantom-foundation/go-lachesis/src/peers"
)

// PeerSelector provides an interface for the lachesis node to 
// update the last peer it gossiped with and select the next peer
// to gossip with 
// type PeerSelector interface {
// 	Peers() *peers.Peers
// 	UpdateLast(peer string)
// 	Next() *peers.Peer
// }

// +++++++++++++++++++++++++++++++++++++++
// Selection based on FlagTable of a randomly chosen undermined event

type SmartPeerSelector struct {
	sync.RWMutex
	peers        *peers.Peers
	localAddr    string
	last         string
	lastN        []*peers.Peer
	GetFlagTable func() (map[string]int64, error)
}

func NewSmartPeerSelector(participants *peers.Peers,
	localAddr string,
	GetFlagTable func() (map[string]int64, error)) *SmartPeerSelector {

	return &SmartPeerSelector{
		localAddr:    localAddr,
		peers:        participants,
		GetFlagTable: GetFlagTable,
	}
}

func (ps *SmartPeerSelector) Peers() *peers.Peers {
	ps.RLock()
	defer ps.RUnlock()
	return ps.peers
}

func (ps *SmartPeerSelector) UpdateLast(peer string) {
	ps.Lock()
	defer ps.Unlock()
	ps.last = peer
}

func (ps *SmartPeerSelector) UpdateLastN(peers []*peers.Peer) {
	ps.Lock()
	defer ps.Unlock()
	ps.lastN = peers
}

func (ps *SmartPeerSelector) Next() *peers.Peer {
	ps.RLock()
	defer ps.RUnlock()
	selectablePeers := ps.peers.ToPeerSlice()
	if len(selectablePeers) > 1 {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.localAddr)
		if len(selectablePeers) > 1 {
			_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.last)
			if len(selectablePeers) > 1 {
				if ft, err := ps.GetFlagTable(); err == nil {
					for id, flag := range ft {
						if flag == 1 && len(selectablePeers) > 1 {
							peers.ExcludePeer(selectablePeers, id)
						}
					}
				}
			}
		}
	}
	i := rand.Intn(len(selectablePeers))
	return selectablePeers[i]
}

func (ps *SmartPeerSelector) NextN(n int) []*peers.Peer {
	ps.Lock()
	defer ps.Unlock()

	selectablePeers := ps.peers.ToPeerSlice()
	if len(selectablePeers) > n {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.localAddr)
		if len(selectablePeers) > n {
			selectablePeers = peers.ExcludePeers(selectablePeers, ps.lastN)
			if len(selectablePeers) > n {
				if ft, err := ps.GetFlagTable(); err == nil {
					for id, flag := range ft {
						if flag == 1 && len(selectablePeers) > n {
							peers.ExcludePeer(selectablePeers, id)
						}
					}
				}
			}
		}
	}

	if len(selectablePeers) > n {
		rand.Shuffle(len(selectablePeers), func(i, j int) {
			selectablePeers[i], selectablePeers[j] = selectablePeers[j], selectablePeers[i]
		})
		return selectablePeers[:n]
	}

	return selectablePeers
}
