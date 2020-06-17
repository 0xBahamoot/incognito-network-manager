package icnwmng

import (
	"fmt"

	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func (man *IncognitoNetworkManager) createRelayAddresses() []string {
	var result []string
	for _, peerID := range man.relayPeerConns {
		relayaddr, err := ma.NewMultiaddr("/p2p/" + peerID.Pretty() + "/p2p-circuit/p2p/" + man.server.host.ID().Pretty())
		if err != nil {
			panic(err)
		}
		result = append(result, relayaddr.String())
	}
	return result
}

func (man *IncognitoNetworkManager) findRelayPeers() {
	man.relayPeerCandidate = []peer.ID{}
	for _, peer := range man.peerList {
		if !checkPeerIDExist(man.relayPeerCandidate, peer.ID) {
			canHop, err := circuit.CanHop(man.ctx, man.server.host, peer.ID)
			if err != nil {
				fmt.Println(err)
			}
			if canHop {
				man.relayPeerCandidate = append(man.relayPeerCandidate, peer.ID)
			}
		}
	}
}

func (man *IncognitoNetworkManager) connectRelayPeer() {
	newRelayPeerCandidate := []peer.ID{}
	for _, peerID := range man.relayPeerCandidate {
		if checkPeerIDExist(man.relayPeerConns, peerID) {
			continue
		}
		if len(man.relayPeerConns) >= maxRelayPeer {
			newRelayPeerCandidate = append(newRelayPeerCandidate, peerID)
			continue
		}
		if checkPeerIDExist(man.connectedPeer, peerID) {
			man.relayPeerConns = append(man.relayPeerConns, peerID)
			continue
		}
		err := man.server.host.Connect(man.ctx, man.server.host.Peerstore().PeerInfo(peerID))
		if err != nil {
			fmt.Println(err)
		} else {
			man.relayPeerConns = append(man.relayPeerConns, peerID)
		}
	}
	man.relayPeerCandidate = newRelayPeerCandidate
}

func (man *IncognitoNetworkManager) getCurrentPeerRelay() []peer.ID {
	result := make([]peer.ID, len(man.relayPeerConns))
	copy(result, man.relayPeerConns)
	return result
}
