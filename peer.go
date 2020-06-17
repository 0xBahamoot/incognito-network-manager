package icnwmng

import "github.com/libp2p/go-libp2p-core/peer"

type IncognitoPeer struct {
	PublicKey string
	peer.AddrInfo
}
