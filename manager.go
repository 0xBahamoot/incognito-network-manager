package icnwmng

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type IncognitoNetworkManager struct {
	ctx                 context.Context
	server              *Server
	notifee             *notifee
	PeerWithIncognitoID map[string][]peer.ID
	processInMessage    func(peer.ID, []byte)

	peerList           []peer.AddrInfo
	connectedPeer      []peer.ID
	relayPeerCandidate []peer.ID
	relayPeerConns     []peer.ID

	openedStreams map[peer.ID]map[string]network.Stream
	protocol      protocol.ID
}

type NetworkOption struct {
}

func InitNetwork(ctx context.Context, hop *ServerOption, protocol protocol.ID, processInMessage func(peer.ID, []byte)) (*IncognitoNetworkManager, error) {
	var man IncognitoNetworkManager
	var err error
	noti := notifee{
		OnPeerConnected:    man.onPeerConnected,
		OnPeerDisconnected: man.onPeerDisconnected,
		OnPeerStreamOpened: man.onPeerStreamOpened,
		OnPeerStreamClosed: man.onPeerStreamClosed,
	}
	man.ctx = ctx
	man.openedStreams = make(map[peer.ID]map[string]network.Stream)
	man.protocol = protocol
	man.notifee = &noti
	man.server, err = createServer(ctx, hop, &man)
	man.processInMessage = processInMessage

	if hop.UseRelayPeer {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for {
				<-ticker.C
				man.findRelayPeers()
				man.connectRelayPeer()
				man.server.updateBroadcastAddr()
			}
		}()
	}

	return &man, err
}

func (man *IncognitoNetworkManager) TagPeer(peer.ID, string, int)             {}
func (man *IncognitoNetworkManager) UntagPeer(peer.ID, string)                {}
func (man *IncognitoNetworkManager) UpsertTag(peer.ID, string, func(int) int) {}
func (man *IncognitoNetworkManager) GetTagInfo(peer.ID) *connmgr.TagInfo      { return &connmgr.TagInfo{} }
func (man *IncognitoNetworkManager) TrimOpenConns(ctx context.Context)        {}
func (man *IncognitoNetworkManager) Notifee() network.Notifiee                { return man.notifee }
func (man *IncognitoNetworkManager) Protect(peer.ID, string)                  {}
func (man *IncognitoNetworkManager) IsProtected(id peer.ID, tag string) (protected bool) {
	protected = false
	return
}
func (man *IncognitoNetworkManager) Unprotect(peer.ID, string) bool { return false }
func (man *IncognitoNetworkManager) Close() error                   { return nil }

func (man *IncognitoNetworkManager) onPeerConnected(pID peer.ID) {
	fmt.Println("onPeerConnected", pID)
	man.connectedPeer = append(man.connectedPeer, pID)
}
func (man *IncognitoNetworkManager) onPeerDisconnected(pID peer.ID) {
	fmt.Println("onPeerDisconnected", pID)
	for idx, peerID := range man.connectedPeer {
		if peerID == pID {
			copy(man.connectedPeer[idx:], man.connectedPeer[idx+1:])
			man.connectedPeer[len(man.connectedPeer)-1] = ""
			man.connectedPeer = man.connectedPeer[:len(man.connectedPeer)-1]
		}
	}
	for idx, peerID := range man.relayPeerConns {
		if peerID == pID {
			copy(man.connectedPeer[idx:], man.connectedPeer[idx+1:])
			man.connectedPeer[len(man.connectedPeer)-1] = ""
			man.connectedPeer = man.connectedPeer[:len(man.connectedPeer)-1]
		}
	}
}
func (man *IncognitoNetworkManager) onPeerStreamOpened(pID peer.ID, stream network.Stream) {
	fmt.Println("onPeerStreamOpened", pID)
	if len(man.openedStreams[pID]) == 0 {
		man.openedStreams[pID] = make(map[string]network.Stream)
	}
	man.openedStreams[pID][stream.ID()] = stream
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go man.inMessageHandler(stream.Conn().RemotePeer(), rw)
}
func (man *IncognitoNetworkManager) onPeerStreamClosed(pID peer.ID, stream network.Stream) {
	fmt.Println("onPeerStreamClosed", pID)
	delete(man.openedStreams[pID], stream.ID())
}

func (man *IncognitoNetworkManager) GetAllPeers() []peer.ID {
	result := []peer.ID{}
	for _, peer := range man.peerList {
		result = append(result, peer.ID)
	}
	return result
}
