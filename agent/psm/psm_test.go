package psm

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/lainio/err2/assert"
)

type TestJSON struct {
	Name string `json:"name"`
}

func TestPSM_data(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	registerGobs()

	s := State{
		PLInfo: PayloadInfo{Type: "type"},
	}
	subStates := []State{s}
	type fields struct {
		Key    StateKey
		States []State
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{"1st", fields{Key: StateKey{DID: "TEST", Nonce: "1234"}, States: subStates}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			p := PSM{
				Key:    tt.fields.Key,
				States: tt.fields.States,
			}
			dec := gob.NewDecoder(bytes.NewBuffer(p.Data()))
			var decodedP PSM
			_ = dec.Decode(&decodedP)
			if !reflect.DeepEqual(p, decodedP) {
				t.Errorf("data() = %v, want %v", decodedP, p)
			}
		})
	}
}

func Test_newPSM(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	p := PSM{
		Key: StateKey{
			DID:   mockStateDID,
			Nonce: mockStateNonce,
		},
		ConnID: "TEST",
		States: nil,
	}
	b := p.Data()
	type args struct {
		d []byte
	}
	tests := []struct {
		name string
		args args
		want *PSM
	}{
		{"1st",
			args{d: b},
			&PSM{Key: StateKey{DID: "TEST", Nonce: "1234"}, ConnID: "TEST"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			if got := NewPSM(tt.args.d); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newPSM() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_timestamp(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	type args struct {
		d *PSM
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{"zero",
			args{d: testPSM(0)},
			0,
		},
		{"value",
			args{d: testPSM(123)},
			123,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			if got := tt.args.d.Timestamp(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Timestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_next(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	type args struct {
		d *PSM
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"no protocol",
			args{d: testPSM(0)},
			"",
		},
		{"message",
			args{d: testPSM(123)},
			"message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			if got := tt.args.d.Next(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccept(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	p := PSM{
		Key: StateKey{
			DID:   mockStateDID,
			Nonce: mockStateNonce,
		},
		ConnID: "TEST",
		States: []State{
			{Sub: Received}, {Sub: Waiting}, {Sub: Waiting},
		},
	}
	accept := p.Accept(Received)
	assert.That(accept)

	accept = p.Accept(Sending)
	assert.That(accept)

	accept = p.Accept(ReadyACK) // important: JS agent's bug
	assert.That(accept, "waiting -> ready is ok for NOW")

	p.States = []State{{Sub: Received}, {Sub: Waiting}, {Sub: ReadyACK}}
	accept = p.Accept(Waiting)
	assert.ThatNot(accept)

	p.States = []State{{Sub: Received}, {Sub: Waiting}, {Sub: Failure}}
	accept = p.Accept(Waiting)
	assert.ThatNot(accept)

	p.States = []State{{Sub: Received}, {Sub: Decrypted}}
	accept = p.Accept(Sending)
	assert.That(accept)

	p.States = []State{{Sub: Received}}
	accept = p.Accept(ReadyACK)
	assert.That(accept)
}
