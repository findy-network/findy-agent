package pool

import (
	"reflect"
	"testing"

	"github.com/findy-network/findy-agent/agent/async"
	indyDto "github.com/findy-network/findy-wrapper-go/dto"
)

func TestLedgerConnection_Open(t *testing.T) {
	r := indyDto.Result{Data: indyDto.Data{Handle: 1}}
	pool.V = r
	pool.On = async.Consumed

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"1st", args{"myNewPool"}, 1},
		{"same name", args{"name1"}, 1},
		{"different name", args{"name2"}, 1},
		{"same different again", args{"name2"}, 1},
		{"1st name again", args{"name1"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Open(tt.args.name).Int(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LedgerConnection.Open() = %v, want %v", got, tt.want)
			}
		})
	}
}
