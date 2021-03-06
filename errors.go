package icnwmng

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnexpectedError = iota
	GetPeerIdError
	ConnectError
	StartError
	StopError
	NotAcceptConnectionError

	PeerGenerateKeyPairError
	CreateP2PNodeError
	CreateP2PAddressError
	GetPeerIdFromProtocolError
	OpeningStreamP2PError
	HandleNewStreamError

	// PeerConn err
	HandleMessageCheckResponse
	HandleMessageCheck
	LimitByteForMessageError
	ReadStringMessageError
	HexDecodeMessageError
	UnzipMessageError
	HashToPoolError
	MessageTypeError
	CheckForwardError
	ParseJsonMessageError
	CacheMessageHashError
	UnhandleMessageTypeError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	UnexpectedError:          {-1, "Unexpected error"},
	GetPeerIdError:           {-2, "Get peer id fail"},
	ConnectError:             {-3, "Connect error"},
	StartError:               {-4, "Start error"},
	StopError:                {-5, "Stop errior"},
	NotAcceptConnectionError: {-6, "Not accept connection"},

	// -1xxx for peer
	PeerGenerateKeyPairError:   {-1001, "Can not generate key pair with reader"},
	CreateP2PNodeError:         {-1002, "Can not create libp2p node"},
	CreateP2PAddressError:      {-1003, "Can not create libp2p address for node"},
	GetPeerIdFromProtocolError: {-1004, "Can not get peer id from protocol"},
	OpeningStreamP2PError:      {-1005, "Fail in opening stream "},
	HandleNewStreamError:       {-1006, "Handle new stream error"},

	// -2xxx for peer connection
	HandleMessageCheckResponse: {-2001, "Handle message check response error"},
	HandleMessageCheck:         {-2002, "Handle message check error"},
	LimitByteForMessageError:   {-2003, "Limit byte for message"},
	ReadStringMessageError:     {-2004, "Read message error"},
	HexDecodeMessageError:      {-2005, "Hex decode message error"},
	UnzipMessageError:          {-2006, "Unzip message error"},
	HashToPoolError:            {-2007, "Insert hash of message to pool error"},
	MessageTypeError:           {-2008, "Can not find particular message for message cmd type"},
	CheckForwardError:          {-2009, "Check forward error"},
	ParseJsonMessageError:      {-2010, "Can not parse struct from json message"},
	CacheMessageHashError:      {-2011, "Cache messagse hash error"},
	UnhandleMessageTypeError:   {-2012, "Received unhandled message of type"},
}

type ConnManagerError struct {
	Code    int
	Message string
	err     error
}

func (e ConnManagerError) Error() string {
	return fmt.Sprintf("%d: %s %+v", e.Code, e.Message, e.err)
}

func NewConnManagerError(key int, err error) *ConnManagerError {
	return &ConnManagerError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].Message,
		err:     errors.Wrap(err, ErrCodeMessage[key].Message),
	}
}
