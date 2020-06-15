package icnwmng

import (
	"reflect"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
)

func TestInitNetwork(t *testing.T) {
	type args struct {
		hop              *HostOption
		processInMessage func(peer.ID, []byte)
	}
	tests := []struct {
		name    string
		args    args
		want    *IncognitoNetworkManager
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitNetwork(tt.args.hop, tt.args.processInMessage)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitNetwork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}
