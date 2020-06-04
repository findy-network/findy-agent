package steward

import (
	"testing"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/stretchr/testify/assert"
)

func TestCreateCmd_ValidateSeed(t *testing.T) {
	type fields struct {
		Seed string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"empty", fields{""}, false},
		{"too short", fields{"123"}, true},
		{"seed 31", fields{"0123456789012345678901234567890"}, true},
		{"seed 33", fields{"012345678901234567890123456789012"}, true},
		{"correct seed", fields{"000000000000000000000000Steward2"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cmds.ValidateSeed(tt.fields.Seed); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCmd_Build(t *testing.T) {
	invalid := CreateCmd{
		Cmd: cmds.Cmd{
			WalletName: "test_name_ttttt",
			WalletKey:  "wrong_key",
		},
		PoolName:    "test",
		StewardSeed: "seed",
	}
	err := invalid.Validate()
	assert.Error(t, err)

	c := CreateCmd{
		Cmd: cmds.Cmd{
			WalletName: "test_name_ttttt",
			WalletKey:  "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
		},
		PoolName:    "test_name",
		StewardSeed: "000000000000000000000000Steward2",
	}
	err = c.Validate()
	assert.NoError(t, err)
}
