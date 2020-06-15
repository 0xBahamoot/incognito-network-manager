package icnwmng

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	mrand "math/rand"

	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	config "github.com/libp2p/go-libp2p/config"
	nat "github.com/libp2p/go-nat"
	"github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

type Host struct {
	host          host.Host
	natType       network.Reachability
	broadcastAddr []string
	listenAddrs   []string
	listenPort    int
	natDevice     nat.NAT
	cancel        context.CancelFunc
	identityKey   crypto.PrivKey
	protocol      protocol.ID
	ctx           context.Context

	peerList           []peer.AddrInfo
	connectedPeer      []peer.ID
	relayPeerCandidate []peer.ID
	relayPeerConns     []peer.ID
	openedStreams      map[peer.ID]map[string]network.Stream
	useRelayPeer       bool

	connman *IncognitoNetworkManager
}

func createHost(pctx context.Context, option *HostOption, connman *IncognitoNetworkManager) (*Host, error) {
	if pctx == nil {
		pctx = context.Background()
	}
	ctx, cancel := context.WithCancel(pctx)

	host := Host{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: []string{},
		listenPort:    option.Port,
		cancel:        cancel,
		identityKey:   option.IdentityKey,
		ctx:           ctx,
		useRelayPeer:  option.UseRelayPeer,
		connman:       connman,
		openedStreams: make(map[peer.ID]map[string]network.Stream),
	}

	natDevice, err := checkNATDevice(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		host.natDevice = natDevice
	}

	hostAddrs := GetOutboundIP()

	var listenAddrs []string
	for _, addr := range hostAddrs {
		listenAddrs = append(listenAddrs, "/ip4/"+addr+"/tcp/"+strconv.Itoa(option.Port))
	}

	copy(host.listenAddrs, listenAddrs)

	if option.IdentityKey == nil {
		r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
		option.IdentityKey, _, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, r)
		if err != nil {
			panic(err)
		}
	}

	opts := []config.Option{}
	opts = append(opts, libp2p.ListenAddrStrings(listenAddrs...))
	opts = append(opts, libp2p.NATPortMap())
	opts = append(opts, libp2p.EnableNATService())
	opts = append(opts, libp2p.Transport(tcp.NewTCPTransport))
	opts = append(opts, libp2p.Identity(option.IdentityKey))
	opts = append(opts, libp2p.ConnectionManager(connman))

	if option.EnableRelay {
		opts = append(opts, libp2p.EnableRelay(circuit.OptHop))
	}
	noti := notifee{
		OnPeerConnected:    host.onPeerConnected,
		OnPeerDisconnected: host.onPeerDisconnected,
		OnPeerStreamOpened: host.onPeerStreamOpened,
		OnPeerStreamClosed: host.onPeerStreamClosed,
	}
	connman.notifee = &noti

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	host.host = h
	host.host.SetStreamHandler(option.protocol, host.StreamHandler)
	if option.NATdiscoverAddr != "" {
		err = host.ConnectPeer(option.NATdiscoverAddr)
		if err != nil {
			return nil, err
		}
		go func() {
			cSub, err := h.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
			if err != nil {
				panic(err)
			}
			defer cSub.Close()
			for {
				select {
				case stat := <-cSub.Out():
					if stat == network.ReachabilityUnknown {
						panic("After status update, client did not know its status")
					}
					t := stat.(event.EvtLocalReachabilityChanged)
					host.natType = t.Reachability
					err := host.updateBroadcastAddr()
					if err != nil {
						log.Fatal(err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	if host.useRelayPeer {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for {
				<-ticker.C
				host.findRelayPeers()
				host.connectRelayPeer()
				host.updateBroadcastAddr()
			}
		}()
	}
	if err := host.updateBroadcastAddr(); err != nil {
		return nil, err
	}
	return &host, nil
}

func (h *Host) GetNATType() network.Reachability {
	return h.natType
}

func (h *Host) GetBroadcastAddr() []string {
	result := make([]string, len(h.broadcastAddr))
	copy(result, h.broadcastAddr)
	return result
}

func (h *Host) GetHost() host.Host {
	return h.host
}

func (h *Host) Quit() {
	h.cancel()
}

func (h *Host) GetListeningPort() int {
	return h.listenPort
}

func (h *Host) updateBroadcastAddr() error {
	switch h.natType {
	case network.ReachabilityUnknown, network.ReachabilityPrivate:
		//behind router that is nested NATs or that not support PCP protocol
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
		var fullAddr []string
		for _, addr := range h.host.Addrs() {
			fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
		}
		if h.useRelayPeer {
			fullAddr = append(fullAddr, h.createRelayAddresses()...)
		}
		h.broadcastAddr = fullAddr
	case network.ReachabilityPublic:
		if h.natDevice == nil {
			//public IP case
			hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
			var fullAddr []string
			for _, addr := range h.host.Addrs() {
				fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
			}
			h.broadcastAddr = fullAddr
		} else {
			//behind public IP router that support PCP protocol
			for _, addr := range h.host.Addrs() {
				if manet.IsPublicAddr(addr) {
					hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
					var fullAddr []string
					for _, addr := range h.host.Addrs() {
						fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
					}
					h.broadcastAddr = fullAddr
					return nil
				}
			}

		}
	}
	return nil
}

func (h *Host) GetHostID() peer.ID {
	return h.host.ID()
}

func (h *Host) ConnectPeer(peerAddr string) error {
	peerInfo, err := PeerInfoFromString(peerAddr)
	if err != nil {
		return err
	}

	if err := h.host.Connect(context.Background(), *peerInfo); err != nil {
		return err
	}
	if !checkPeerIDExist(peerIDsFromPeerInfos(h.peerList), peerInfo.ID) {
		h.peerList = append(h.peerList, *peerInfo)
		h.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour)
	}
	return nil
}

func (h *Host) GetListenAddrs() []string {
	result := make([]string, len(h.listenAddrs))
	copy(result, h.listenAddrs)
	return result
}

func (h *Host) GetAllPeers() []peer.ID {
	result := []peer.ID{}
	for _, peer := range h.peerList {
		result = append(result, peer.ID)
	}
	return result
}

func (h *Host) onPeerConnected(pID peer.ID) {
	fmt.Println("onPeerConnected", pID)
	h.connectedPeer = append(h.connectedPeer, pID)
}
func (h *Host) onPeerDisconnected(pID peer.ID) {
	fmt.Println("onPeerDisconnected", pID)
	for idx, peerID := range h.connectedPeer {
		if peerID == pID {
			copy(h.connectedPeer[idx:], h.connectedPeer[idx+1:])
			h.connectedPeer[len(h.connectedPeer)-1] = ""
			h.connectedPeer = h.connectedPeer[:len(h.connectedPeer)-1]
		}
	}
	for idx, peerID := range h.relayPeerConns {
		if peerID == pID {
			copy(h.connectedPeer[idx:], h.connectedPeer[idx+1:])
			h.connectedPeer[len(h.connectedPeer)-1] = ""
			h.connectedPeer = h.connectedPeer[:len(h.connectedPeer)-1]
		}
	}
}
func (h *Host) onPeerStreamOpened(pID peer.ID, stream network.Stream) {
	fmt.Println("onPeerStreamOpened", pID)
	if len(h.openedStreams[pID]) == 0 {
		h.openedStreams[pID] = make(map[string]network.Stream)
	}
	h.openedStreams[pID][stream.ID()] = stream
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go h.inMessageHandler(stream.Conn().RemotePeer(), rw)
}
func (h *Host) onPeerStreamClosed(pID peer.ID, stream network.Stream) {
	fmt.Println("onPeerStreamClosed", pID)
	delete(h.openedStreams[pID], stream.ID())
}

func (h *Host) SetNotifee(n network.Notifiee) {
	h.host.Network().Notify(n)
}
