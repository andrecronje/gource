package node

import (
	"math/rand"
	"sync"
	"time"

	"github.com/andrecronje/lachesis/src/peers"
)

type PeerSelector interface {
	Peers() *peers.Peers
	UpdateLast(peer string)
	UpdateLastN(peers []*peers.Peer)
	Next() *peers.Peer
	NextN(n int) []*peers.Peer
}

// +++++++++++++++++++++++++++++++++++++++
// RANDOM

type RandomPeerSelector struct {
	sync.RWMutex
	peers     *peers.Peers
	localAddr string
	last      string
	lastN     []*peers.Peer
}

func NewRandomPeerSelector(participants *peers.Peers, localAddr string) *RandomPeerSelector {
	rand.Seed(time.Now().UnixNano())
	return &RandomPeerSelector{
		localAddr: localAddr,
		peers:     participants,
	}
}

func (ps *RandomPeerSelector) Peers() *peers.Peers {
	ps.RLock()
	defer ps.RUnlock()
	return ps.peers
}

func (ps *RandomPeerSelector) UpdateLast(peer string) {
	ps.Lock()
	defer ps.Unlock()
	ps.last = peer
}

func (ps *RandomPeerSelector) UpdateLastN(peers []*peers.Peer) {
	ps.Lock()
	defer ps.Unlock()
	ps.lastN = peers
}

func (ps *RandomPeerSelector) Next() *peers.Peer {
	ps.RLock()
	defer ps.RUnlock()
	selectablePeers := ps.peers.ToPeerSlice()

	if len(selectablePeers) > 1 {
		_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.localAddr)

		if len(selectablePeers) > 1 {
			_, selectablePeers = peers.ExcludePeer(selectablePeers, ps.last)
		}
	}

	i := rand.Intn(len(selectablePeers))

	peer := selectablePeers[i]

	return peer
}

func (ps *RandomPeerSelector) NextN(n int) []*peers.Peer {
	ps.Lock()
	defer ps.Unlock()
	selectablePeers := ps.peers.ToPeerSlice()

	if len(selectablePeers) > n*2 {
		selectablePeers = peers.ExcludePeers(selectablePeers, ps.lastN)
	}

	if len(selectablePeers) > n {
		rand.Shuffle(len(selectablePeers), func(i, j int) {
			selectablePeers[i], selectablePeers[j] = selectablePeers[j], selectablePeers[i]
		})
		return selectablePeers[:n]
	}

	return selectablePeers
}
