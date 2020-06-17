package icnwmng

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

func (man *IncognitoNetworkManager) inMessageHandler(peerID peer.ID, rw *bufio.ReadWriter) {
	for {
		bytes, err := readBytes(rw, delimMessageByte, spamMessageSize)
		if err != nil {
			fmt.Println(err)
		}
		man.processInMessage(peerID, bytes)
	}
}

func (man *IncognitoNetworkManager) sendMessage(peer peer.ID, msg []byte) error {
	for _, stream := range man.openedStreams[peer] {
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		_, err := rw.Writer.Write(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		err = rw.Writer.Flush()
		if err != nil {
			fmt.Println(err)
			continue
		}
		return nil
	}
	return nil
}

// readString - read data from received message on stream
// and convert to string format
func readBytes(rw *bufio.ReadWriter, delim byte, maxReadBytes int) ([]byte, error) {
	buf := make([]byte, 0)
	bufL := 0
	// Loop to read byte to byte
	for {
		b, err := rw.ReadByte()
		if err != nil {
			return nil, NewConnManagerError(ReadStringMessageError, err)
		}
		// break byte buf after get a delim
		if b == delim {
			break
		}
		// continue add read byte to buf if not find a delim
		buf = append(buf, b)
		bufL++
		if bufL > maxReadBytes {
			return nil, NewConnManagerError(LimitByteForMessageError, errors.New("limit bytes for message"))
		}
	}

	// convert byte buf to string format
	return buf, nil
}
