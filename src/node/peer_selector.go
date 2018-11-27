package node

import (
	"math/rand"
	"sync"

	"github.com/Fantom-foundation/go-lachesis/src/peers"
)

// PeerSelector provides an interface for the lachesis node to
// update the last peer it gossiped with and select the next peer
// to gossip with
type PeerSelector interface {
	Peers() *peers.Peers
	UpdateLast(peer string)
	Next() *peers.Peer
}

//+++++++++++++++++++++++++++++++++++++++
//RANDOM

type RandomPeerSelector struct {
	peers     *peers.Peers
	localAddr string
	mtx       sync.Mutex
	last      string
}

func NewRandomPeerSelector(participants *peers.Peers, localAddr string) *RandomPeerSelector {
	return &RandomPeerSelector{
		localAddr: localAddr,
		peers:     participants,
	}
}

func (ps *RandomPeerSelector) Peers() *peers.Peers {
	return ps.peers
}

func (ps *RandomPeerSelector) UpdateLast(peer string) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.last = peer
}

func (ps *RandomPeerSelector) Next() *peers.Peer {
	selectablePeers := ps.peers.ToPeerSlice()

	if len(selectablePeers) > 1 {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.localAddr)

		if len(selectablePeers) > 1 {
			ps.mtx.Lock()
			_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.last)
			ps.mtx.Unlock()
		}
	}

	i := rand.Intn(len(selectablePeers))

	peer := selectablePeers[i]

	return peer
}
