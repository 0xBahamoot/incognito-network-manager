package icnwmng

import (
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

type Server struct {
	host          host.Host
	natType       network.Reachability
	broadcastAddr []string
	listenAddrs   []string
	listenPort    int
	natDevice     nat.NAT
	cancel        context.CancelFunc
	identityKey   crypto.PrivKey
	ctx           context.Context

	useRelayPeer bool

	connman *IncognitoNetworkManager
}

type ServerOption struct {
	IdentityKey     crypto.PrivKey
	Port            int
	NATdiscoverAddr string
	EnableRelay     bool
	UseRelayPeer    bool
}

func createServer(pctx context.Context, option *ServerOption, connman *IncognitoNetworkManager) (*Server, error) {
	if pctx == nil {
		pctx = context.Background()
	}
	ctx, cancel := context.WithCancel(pctx)

	server := Server{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: []string{},
		listenPort:    option.Port,
		cancel:        cancel,
		identityKey:   option.IdentityKey,
		ctx:           ctx,
		useRelayPeer:  option.UseRelayPeer,
		connman:       connman,
	}

	natDevice, err := checkNATDevice(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		server.natDevice = natDevice
	}

	hostAddrs := GetOutboundIP()

	var listenAddrs []string
	for _, addr := range hostAddrs {
		listenAddrs = append(listenAddrs, "/ip4/"+addr+"/tcp/"+strconv.Itoa(option.Port))
	}

	copy(server.listenAddrs, listenAddrs)

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

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	server.host = h
	if option.NATdiscoverAddr != "" {
		err = server.ConnectPeerByAddr(option.NATdiscoverAddr)
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
					server.natType = t.Reachability
					err := server.updateBroadcastAddr()
					if err != nil {
						log.Fatal(err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	if err := server.updateBroadcastAddr(); err != nil {
		return nil, err
	}
	return &server, nil
}

func (s *Server) GetNATType() network.Reachability {
	return s.natType
}

func (s *Server) GetBroadcastAddr() []string {
	result := make([]string, len(s.broadcastAddr))
	copy(result, s.broadcastAddr)
	return result
}

func (s *Server) GetHost() host.Host {
	return s.host
}

func (s *Server) Quit() {
	s.cancel()
}

func (s *Server) GetListeningPort() int {
	return s.listenPort
}

func (s *Server) updateBroadcastAddr() error {
	switch s.natType {
	case network.ReachabilityUnknown, network.ReachabilityPrivate:
		//behind router that is nested NATs or that not support PCP protocol
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", s.host.ID().Pretty()))
		var fullAddr []string
		for _, addr := range s.host.Addrs() {
			fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
		}
		if s.useRelayPeer {
			fullAddr = append(fullAddr, s.connman.createRelayAddresses()...)
		}
		s.broadcastAddr = fullAddr
	case network.ReachabilityPublic:
		if s.natDevice == nil {
			//public IP case
			hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", s.host.ID().Pretty()))
			var fullAddr []string
			for _, addr := range s.host.Addrs() {
				fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
			}
			s.broadcastAddr = fullAddr
		} else {
			//behind public IP router that support PCP protocol
			for _, addr := range s.host.Addrs() {
				if manet.IsPublicAddr(addr) {
					hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", s.host.ID().Pretty()))
					var fullAddr []string
					for _, addr := range s.host.Addrs() {
						fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
					}
					s.broadcastAddr = fullAddr
					return nil
				}
			}

		}
	}
	return nil
}

func (s *Server) GetHostID() peer.ID {
	return s.host.ID()
}

func (s *Server) ConnectPeerByAddr(peerAddr string) error {
	peerInfo, err := PeerInfoFromString(peerAddr)
	if err != nil {
		return err
	}

	if err := s.host.Connect(context.Background(), *peerInfo); err != nil {
		return err
	}
	// if !checkPeerIDExist(peerIDsFromPeerInfos(s.peerList), peerInfo.ID) {
	// 	s.peerList = append(s.peerList, *peerInfo)
	// 	s.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour)
	// }
	return nil
}

func (s *Server) GetListenAddrs() []string {
	result := make([]string, len(s.listenAddrs))
	copy(result, s.listenAddrs)
	return result
}

func (s *Server) createStream(ctx context.Context, peerID peer.ID, protocol protocol.ID, forceNew bool) (network.Stream, error) {
	if !forceNew {
		for _, peerConn := range s.host.Network().Conns() {
			if peerConn.RemotePeer() != peerID {
				continue
			}
			for _, stream := range peerConn.GetStreams() {
				if stream.Protocol() == protocol {
					return nil, fmt.Errorf("%s | protocol:%s | peerID:%s", ErrCreateStreamExist, protocol, peerID)
				}
			}
		}
	}
	return s.host.NewStream(ctx, peerID, protocol)
}
