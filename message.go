package icnwmng

import "github.com/libp2p/go-libp2p-core/peer"

func (man *IncognitoNetworkManager) BroadcastMessage(msg []byte) {

}

func (man *IncognitoNetworkManager) SendMessageToPeerID(msg []byte, peerID peer.ID) {
	man.Host.sendMessage(peerID, msg)
}

func (man *IncognitoNetworkManager) ConnectToPeerID(peerID string) {

}
