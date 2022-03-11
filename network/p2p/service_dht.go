package p2p

import (
	"context"
	"math/rand"

	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbucket "github.com/libp2p/go-libp2p-kbucket"

	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
)

type P2PDHTService struct {
	ctx context.Context
	log tplog.Logger
	dht *dht.IpfsDHT
}

func NewP2PDHTService(ctx context.Context, log tplog.Logger, dht *dht.IpfsDHT) *P2PDHTService {
	return &P2PDHTService{
		ctx: ctx,
		log: tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PDHTService", log),
		dht: dht,
	}
}

func (ps *P2PDHTService) GetAllPeerIDs() ([]peer.ID, error) {
	rt := ps.dht.RoutingTable()
	peers := make([]peer.ID, 0, len(rt.GetPeerInfos()))
	for _, peerInfos := range rt.GetPeerInfos() {
		peers = append(peers, peerInfos.Id)
	}
	return peers, nil
}

func (ps *P2PDHTService) GetNearestPeerIDs(peerID peer.ID) ([]peer.ID, error) {
	peers := ps.dht.RoutingTable().NearestPeers(kbucket.ConvertPeerID(peerID), tpnetcmn.MaxBroadCastPeers)

	return peers, nil
}

func (ps *P2PDHTService) GetPeersWithFactor() ([]peer.ID, error) {
	factor := 0.5
	rt := ps.dht.RoutingTable()
	filterPeers := []peer.ID{}
	for _, peerInfos := range rt.GetPeerInfos() {
		peers := []peer.ID{}
		peers = append(peers, peerInfos.Id)
		peersSize := len(peers)
		step := int(1.0 / factor)
		splitSize := int(float64(peersSize) / (1.0 / factor))
		if peersSize == 0 {
			continue
		}
		pos := 0
		for pos = 0; pos < splitSize; pos++ {
			lastPos := pos * step
			for b := lastPos; b < lastPos+step && b < peersSize; b += step {
				randPos := rand.Intn(step) + lastPos
				filterPeers = append(filterPeers, peers[randPos])
			}
		}
		for a := pos * step; a < peersSize; a += 2 {
			filterPeers = append(filterPeers, peers[a])
		}
	}

	return filterPeers, nil
}

func (ps *P2PDHTService) Close() error {
	return ps.dht.Close()
}
