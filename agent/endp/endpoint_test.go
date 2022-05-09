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
		{"long", args{"/agency/endpoint/transport/connID/token"},
			&Addr{Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport", ConnID: "connID", EdgeToken: "token"},
		},
		{"long", args{"/agency-2/6PpcwtwDJ5TJYnianLgYbn/RHLDsziT56McTZVFXKV5Pk/670bc804-2c06-453c-aee6-48d3c929b488/token"},
			&Addr{
				Service:   "agency-2",
				PlRcvr:    "6PpcwtwDJ5TJYnianLgYbn",
				MsgRcvr:   "RHLDsziT56McTZVFXKV5Pk",
				ConnID:    "670bc804-2c06-453c-aee6-48d3c929b488",
				EdgeToken: "token",
				v2Api:     true,
			},
		},
		{"long edge token", args{"/agency-2/6PpcwtwDJ5TJYnianLgYbn/RHLDsziT56McTZVFXKV5Pk/670bc804-2c06-453c-aee6-48d3c929b488/670bc804-2c06-453c-aee6-48d3c929b488"},
			&Addr{
				Service:   "agency-2",
				PlRcvr:    "6PpcwtwDJ5TJYnianLgYbn",
				MsgRcvr:   "RHLDsziT56McTZVFXKV5Pk",
				ConnID:    "670bc804-2c06-453c-aee6-48d3c929b488",
				EdgeToken: "670bc804-2c06-453c-aee6-48d3c929b488",
				v2Api:     true,
			},
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
		ConnID       string
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
				ConnID:    tt.fields.ConnID,
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
		connID    string
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
			"connID",
			"token",
			"http://hostname",
		}, "http://hostname/agency/endpoint/transport/connID/token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Addr{
				Service:   tt.fields.service,
				PlRcvr:    tt.fields.myPl,
				MsgRcvr:   tt.fields.theirPl,
				ConnID:    tt.fields.connID,
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
		ConnID       string
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
			"connID",
			"token",
			"http://base",
		}, true},
		{"Handshake", fields{
			"agency/",
			"handshake",
			"/transport/",
			"connID",
			"token/",
			"http://base",
		}, false},
		{"Handshake", fields{
			"agency/",
			"ping",
			"/transport/",
			"connID",
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
				ConnID:    tt.fields.ConnID,
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
	ea := &Addr{BasePath: "http://localhost:8090", Service: "findy", PlRcvr: "endpoint", MsgRcvr: "transport", ConnID: "connID", EdgeToken: "token"}
	ea2 := &Addr{BasePath: "http://host", Service: "findyws", PlRcvr: "endpoint", MsgRcvr: "transport", ConnID: "connID"}
	ea3 := &Addr{BasePath: "http://host", Service: "agency", PlRcvr: "endpoint", MsgRcvr: "transport"}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		args   args
		wantEa *Addr
	}{
		{"1st", args{"http://localhost:8090/findy/endpoint/transport/connID/token"}, ea},
		{"2nd", args{"http://host/findyws/endpoint/transport/connID"}, ea2},
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
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"general wrong", args{"/agency/endpoint/transport/connID/token"}, false},
		{"from issue", args{"XXX/api/jsonws/invoke"}, false},
		{"wrong char in", args{"/a2a/MuYkMsV-jvH4Ryqvfoofre/MuYkMsVBjvH4Ryqvfoofre/6im1AuoExt4rT39XuJS94X"}, false},
		{"ok aries a2a", args{"/a2a/MuYkMsVBjvH4Ryqvfoofre/MuYkMsVBjvH4Ryqvfoofre/670bc804-2c06-453c-aee6-48d3c929b488"}, true},
		{"old api call", args{"/ca-api/9R7bAqCJQaDeq4u6UmBpyv/9R7bAqCJQaDeq4u6UmBpyv/670bc804-2c06-453c-aee6-48d3c929b488"}, true},
		{"agency api call", args{"/api/ping"}, true},
		{"was wrong because 21", args{"/a2a/2v5RGVnxaXAVDqvVkanB8h/2v5RGVnxaXAVDqvVkanB8h/670bc804-2c06-453c-aee6-48d3c929b488"}, true},
		{"was wrong l = 21", args{"/a2a/KexHf68PuUaWhr2KdcfFwW/KexHf68PuUaWhr2KdcfFwW/670bc804-2c06-453c-aee6-48d3c929b488"}, true},
		{"wrong l = 20", args{"/a2a/KexHf68PuUaWhr2KdcfFwW/KexHf68PuUaWhr2KdcfFwW/ktSyAAdJRGnZwKjxjjgS"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEa := NewServerAddr(tt.args.s); gotEa.Valid() != tt.want {
				t.Errorf("NewServerAddr() = %v, Valid() = %v", gotEa,
					gotEa.Valid())
			}
		})
	}
}

func TestAddr_Valid(t *testing.T) {
	type fields struct {
		ID        uint64
		Service   string
		PlRcvr    string
		MsgRcvr   string
		ConnID    string
		EdgeToken string
		BasePath  string
		VerKey    string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "1st",
			fields: fields{
				PlRcvr:  "MuYkMsVBjvH4Ryqvfoofre",
				MsgRcvr: "MuYkMsVBjvH4Ryqvfoofre",
				ConnID:  "670bc804-2c06-453c-aee6-48d3c929b488",
			},
			want: true,
		},
		{
			name: "2nd too short",
			fields: fields{
				PlRcvr:  "MuYkMsVBjvH4Ryqvfoofre",
				MsgRcvr: "MuYkMsVBjvH4Ryqvfoofre",
				ConnID:  "670bc804-2c06-453c-aee6-48d3c929b48",
			},
			want: false,
		},
		{
			name: "api/jsonws/invoke",
			fields: fields{
				PlRcvr:  "api",
				MsgRcvr: "jsonws",
				ConnID:  "invoke",
			},
			want: false,
		},
		{
			name: "valid edge token",
			fields: fields{
				PlRcvr:    "MuYkMsVBjvH4Ryqvfoofre",
				MsgRcvr:   "MuYkMsVBjvH4Ryqvfoofre",
				ConnID:    "670bc804-2c06-453c-aee6-48d3c929b488",
				EdgeToken: "670bc804-2c06-453c-aee6-48d3c929b488",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Addr{
				ID:        tt.fields.ID,
				Service:   tt.fields.Service,
				PlRcvr:    tt.fields.PlRcvr,
				MsgRcvr:   tt.fields.MsgRcvr,
				ConnID:    tt.fields.ConnID,
				EdgeToken: tt.fields.EdgeToken,
				BasePath:  tt.fields.BasePath,
				VerKey:    tt.fields.VerKey,
			}
			if got := e.Valid(); got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
