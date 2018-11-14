package peers

import (
	"encoding/hex"

	"github.com/andrecronje/lachesis/src/common"
)

const (
	jsonPeerPath = "peers.json"
)

func NewPeer(pubKeyHex, netAddr string) *Peer {
	peer := &Peer{
		PubKeyHex: pubKeyHex,
		NetAddr:   netAddr,
	}

	peer.computeID()

	return peer
}

func (this *Peer) Equals(that *Peer) bool {
	return this.ID == that.ID &&
		this.NetAddr == that.NetAddr &&
		this.PubKeyHex == that.PubKeyHex
}

func (p *Peer) PubKeyBytes() ([]byte, error) {
	return hex.DecodeString(p.PubKeyHex[2:])
}

func (p *Peer) computeID() error {
	// TODO: Use the decoded bytes from hex
	pubKey, err := p.PubKeyBytes()

	if err != nil {
		return err
	}

	p.ID = int64(common.Hash32(pubKey))

	return nil
}

// PeerStore provides an interface for persistent storage and
// retrieval of peers.
type PeerStore interface {
	// Peers returns the list of known peers.
	Peers() (*Peers, error)

	// SetPeers sets the list of known peers. This is invoked when a peer is
	// added or removed.
	SetPeers([]*Peer) error
}

// ExcludePeer is used to exclude a single peer from a list of peers.
func ExcludePeer(peers []*Peer, peer string) (int, []*Peer) {
	index := -1
	otherPeers := make([]*Peer, 0, len(peers))
	for i, p := range peers {
		if p.NetAddr != peer && p.PubKeyHex != peer {
			otherPeers = append(otherPeers, p)
		} else {
			index = i
		}
	}
	return index, otherPeers
}

// ExcludePeers is used to exclude multiple peers from a list of peers.
func ExcludePeers(peers []*Peer, excludedPeers []*Peer) []*Peer {
	otherPeers := make([]*Peer, 0, len(peers))
	for _, p := range peers {
		found := false
		for _, ex := range excludedPeers {
			if p.NetAddr == ex.NetAddr {
				found = true
				break
			}
		}
		if !found {
			otherPeers = append(otherPeers, p)
		}
	}
	return otherPeers
}
