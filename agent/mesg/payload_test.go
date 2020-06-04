package mesg

import (
	"testing"

	"github.com/optechlab/findy-agent/agent/pltype"
)

func TestPayload_protocol(t *testing.T) {
	type fields struct {
		ID      string
		Type    string
		Message Msg
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"connection", fields{"0001", pltype.ConnectionHandshake, Msg{}}, "connection"},
		{"pairwise", fields{"0001", pltype.TrustPingResponse, Msg{}}, "trust_ping"},
		{"connection", fields{"0001", pltype.ConnectionAck, Msg{}}, "connection"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pl := &Payload{
				ID:      tt.fields.ID,
				Type:    tt.fields.Type,
				Message: tt.fields.Message,
			}
			if got := pl.Protocol(); got != tt.want {
				t.Errorf("protocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayload_protocolMsg(t *testing.T) {
	type fields struct {
		ID      string
		Type    string
		Message Msg
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"request", fields{"0001", pltype.ConnectionRequest, Msg{}}, "request"},
		{"pong", fields{"0001", pltype.TrustPingResponse, Msg{}}, "ping_response"},
		{"acknowledgement", fields{"0001", pltype.ConnectionAck, Msg{}}, "acknowledgement"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pl := &Payload{
				ID:      tt.fields.ID,
				Type:    tt.fields.Type,
				Message: tt.fields.Message,
			}
			if got := pl.ProtocolMsg(); got != tt.want {
				t.Errorf("ProtocolMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}
