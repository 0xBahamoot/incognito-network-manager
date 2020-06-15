package icnwmng

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type HostOption struct {
	IdentityKey     crypto.PrivKey
	Port            int
	NATdiscoverAddr string
	EnableRelay     bool
	UseRelayPeer    bool
	protocol        protocol.ID
}
