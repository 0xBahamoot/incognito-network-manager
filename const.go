package icnwmng

import (
	"errors"
	"time"
)

type TraversalMethod string

const (
	TraversalPMP       = "pmp"
	TraversalHolePunch = "punching"
	TraversalHW        = "highway"
	TraversalNone      = "none"
)

var (
	//ErrShouldHaveIPAddress ...
	ErrShouldHaveIPAddress = errors.New("error machine should have an assigned IP address")
	//ErrNoNATDeviceFound ...
	ErrNoNATDeviceFound = errors.New("error no NAT devices found")
	//ErrCreatingHost ...
	ErrCreatingHost = errors.New("error creating host")
	//ErrCantUpdateBroadcastAddress ...
	ErrCantUpdateBroadcastAddress = errors.New("error cant update broadcast address")
	//ErrCantConnectToNATDiscoverAddress ...
	ErrCantConnectToNATDiscoverAddress = errors.New("error cant connect to NAT discover address")
	// ErrNoMapping signals no mapping exists for an address
	ErrNoMapping = errors.New("mapping not established")
	// ErrCantGetExternalAddress ...
	ErrCantGetExternalAddress = errors.New("error cant get external address")

	// ErrCreateStream ...
	ErrCreateStreamExist = errors.New("error stream of this protocol to this peerID already exist")
)

const (
	maxRelayPeer     = 10
	emptyString      = ""
	delimMessageByte = '\n'
	delimMessageStr  = "\n"
)

const (
	maxRetriesCheckHashMessage = 5
	maxTimeoutCheckHashMessage = time.Duration(10)
	heavyMessageSize           = 5 * 1024 * 1024  // 5 Mb
	spamMessageSize            = 50 * 1024 * 1024 // 50 Mb
)
