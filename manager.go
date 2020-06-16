package icnwmng

import (
	"context"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type IncognitoNetworkManager struct {
	Host                *Host
	notifee             *notifee
	PeerWithIncognitoID map[string][]peer.ID
	processInMessage    func(peer.ID, []byte)
}

func InitNetwork(ctx context.Context, hop *HostOption, processInMessage func(peer.ID, []byte)) (*IncognitoNetworkManager, error) {
	var man IncognitoNetworkManager
	var err error
	man.Host, err = createHost(ctx, hop, &man)
	man.processInMessage = processInMessage
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
