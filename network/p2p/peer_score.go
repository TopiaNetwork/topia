package p2p

import (
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerScoreCache struct {
	sync     sync.RWMutex
	scoreMap map[peer.ID]*pubsub.PeerScoreSnapshot
}

func NewPeerScoreCache() *PeerScoreCache {
	return &PeerScoreCache{
		scoreMap: make(map[peer.ID]*pubsub.PeerScoreSnapshot),
	}
}

func (pc *PeerScoreCache) Update(scores map[peer.ID]*pubsub.PeerScoreSnapshot) {
	pc.sync.Lock()
	defer pc.sync.Unlock()

	pc.scoreMap = scores
}

func (pc *PeerScoreCache) Fetch() map[peer.ID]*pubsub.PeerScoreSnapshot {
	pc.sync.RLock()
	defer pc.sync.RUnlock()

	return pc.scoreMap
}
