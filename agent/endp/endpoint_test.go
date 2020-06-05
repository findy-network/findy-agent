package endp

import (
	"reflect"
	"testing"
)

func TestNewEndpointAddress(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantEa *Addr
	}{
		{"basic", args{"/agency/endpoint/transport"},
			&Addr{Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport"},
		},
		{"long", args{"/agency/endpoint/transport/did/token"},
			&Addr{Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport", RcvrDID: "did", EdgeToken: "token"},
		},
		{"long", args{"/agency/6PpcwtwDJ5TJYnianLgYbn/RHLDsziT56McTZVFXKV5Pk/2H6KuFvaeZxPtuqMoSa6ri/token"},
			&Addr{
				Service:   "agency",
				PlRcvr:    "6PpcwtwDJ5TJYnianLgYbn",
				MsgRcvr:   "RHLDsziT56McTZVFXKV5Pk",
				RcvrDID:   "2H6KuFvaeZxPtuqMoSa6ri",
				EdgeToken: "token"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEa := NewServerAddr(tt.args.s); !reflect.DeepEqual(gotEa, tt.wantEa) {
				t.Errorf("NewServerAddr() = %v, want %v", gotEa, tt.wantEa)
			}
		})
	}
}

func TestEndpointAddress_GetHandlerEndpoint(t *testing.T) {
	type fields struct {
		AgencyName   string
		EndpointName string
		TransportDID string
		DID          string
		EdgeToken    string
		BasePath     string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"only endpoint", fields{AgencyName: "agency", EndpointName: "endpoint"}, "endpoint"},
		{"transport",
			fields{AgencyName: "agency", EndpointName: "endpoint", TransportDID: "transport"},
			"transport"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Addr{
				Service:   tt.fields.AgencyName,
				PlRcvr:    tt.fields.EndpointName,
				MsgRcvr:   tt.fields.TransportDID,
				RcvrDID:   tt.fields.DID,
				EdgeToken: tt.fields.EdgeToken,
				BasePath:  tt.fields.BasePath,
			}
			if got := e.ReceiverDID(); got != tt.want {
				t.Errorf("Addr.ReceiverDID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointAddress_Address(t *testing.T) {
	type fields struct {
		service   string
		myPl      string
		theirPl   string
		msgRsvr   string
		EdgeToken string
		BasePath  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"Easy", fields{
			"agency",
			"endpoint",
			"transport",
			"did",
			"token",
			"http://hostname",
		}, "http://hostname/agency/endpoint/transport/did/token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Addr{
				Service:   tt.fields.service,
				PlRcvr:    tt.fields.myPl,
				MsgRcvr:   tt.fields.theirPl,
				RcvrDID:   tt.fields.msgRsvr,
				EdgeToken: tt.fields.EdgeToken,
				BasePath:  tt.fields.BasePath,
			}
			if got := e.Address(); got != tt.want {
				t.Errorf("Addr.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointAddress_IsEncrypted(t *testing.T) {
	type fields struct {
		AgencyName   string
		EndpointName string
		TransportDID string
		DID          string
		EdgeToken    string
		BasePath     string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"Some word", fields{
			"agency",
			"endpoint",
			"transport",
			"did",
			"token",
			"http://base",
		}, true},
		{"Handshake", fields{
			"agency/",
			"handshake",
			"/transport/",
			"did",
			"token/",
			"http://base",
		}, false},
		{"Handshake", fields{
			"agency/",
			"ping",
			"/transport/",
			"did",
			"token/",
			"http://base",
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Addr{
				Service:   tt.fields.AgencyName,
				PlRcvr:    tt.fields.EndpointName,
				MsgRcvr:   tt.fields.TransportDID,
				RcvrDID:   tt.fields.DID,
				EdgeToken: tt.fields.EdgeToken,
				BasePath:  tt.fields.BasePath,
			}
			if got := e.IsEncrypted(); got != tt.want {
				t.Errorf("Addr.IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEndp(t *testing.T) {
	ea := &Addr{BasePath: "http://localhost:8090", Service: "findy", PlRcvr: "endpoint", MsgRcvr: "transport", RcvrDID: "did", EdgeToken: "token"}
	ea2 := &Addr{BasePath: "http://host", Service: "findyws", PlRcvr: "endpoint", MsgRcvr: "transport", RcvrDID: "did"}
	ea3 := &Addr{BasePath: "http://host", Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport"}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantEa *Addr
	}{
		{"1st", args{"http://localhost:8090/findy/endpoint/transport/did/token"}, ea},
		{"2nd", args{"http://host/findyws/endpoint/transport/did"}, ea2},
		{"3rd", args{"http://host/agency/endpoint/transport"}, ea3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEa := NewClientAddr(tt.args.s); !reflect.DeepEqual(gotEa, tt.wantEa) {
				t.Errorf("NewClientAddr() = %v, want %v", gotEa, tt.wantEa)
			}
		})
	}
}

func TestNewEndpAddr(t *testing.T) {
	ea := &Addr{Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport", RcvrDID: "did", EdgeToken: "token"}
	ea2 := &Addr{Service: "findyws", PlRcvr: "endpoint", MsgRcvr: "transport", RcvrDID: "did"}
	ea3 := &Addr{Service: "findy", PlRcvr: "endpoint", MsgRcvr: "transport"}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantEa *Addr
	}{
		{"1st", args{"/agency/endpoint/transport/did/token"}, ea},
		{"2nd", args{"/findyws/endpoint/transport/did"}, ea2},
		{"3rd", args{"/findy/endpoint/transport"}, ea3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEa := NewServerAddr(tt.args.s); !reflect.DeepEqual(gotEa, tt.wantEa) {
				t.Errorf("NewServerAddr() = %v, want %v", gotEa, tt.wantEa)
			}
		})
	}
}
