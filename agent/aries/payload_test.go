package aries

import (
	"reflect"
	"testing"

	"github.com/optechlab/findy-agent/agent/didcomm"
)

func TestPayload_ReadWriteJSON(t *testing.T) {
	//thread := new(decorator.Thread)
	//dto.FromJSONStr(`{ "thid": "861129bd-1675-499a-b4a3-54c0eec2db42" }`, thread)
	//dto.FromJSONStr(`"~thread": { "thid": "861129bd-1675-499a-b4a3-54c0eec2db42" }`, thread)

	pl := PayloadCreator.New(didcomm.PayloadInit{
		ID:   "123",
		Type: "test-type",
	})
	//pl.SetThread(thread)
	data := pl.JSON()

	pl2 := PayloadCreator.NewFromData(data)
	if !reflect.DeepEqual(pl, pl2) {
		t.Errorf("%v to JSON from %v", pl, pl2)
	}
}
