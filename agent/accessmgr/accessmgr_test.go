package accessmgr

import (
	"reflect"
	"testing"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go/wallet"
)

func Test_buildExportCredentials(t *testing.T) {
	DateTimeInName = false
	utils.Settings.SetWalletBackupPath("TEST_PATH/EXPORT/")
	cfgBase := ssi.Wallet{
		Config: wallet.Config{
			ID:            "WALLET_NAME",
			StorageConfig: &wallet.StorageConfig{Path: "WALLET/PATH"},
		},
		Credentials: wallet.Credentials{
			Path:                "",
			Key:                 "WALLET_KEY",
			KeyDerivationMethod: "RAW",
		},
	}
	type env struct {
		walletName string
		walletKey  string
	}
	tests := []struct {
		name string
		env  env
		want wallet.Credentials
	}{
		{name: "first", env: env{"NAME", "KEY"}, want: wallet.Credentials{
			Path:                "TEST_PATH/EXPORT/NAME",
			Key:                 "KEY",
			KeyDerivationMethod: "RAW",
		}},
		{name: "second", env: env{"NAME2", "KEY-second"}, want: wallet.Credentials{
			Path:                "TEST_PATH/EXPORT/NAME2",
			Key:                 "KEY-second",
			KeyDerivationMethod: "RAW",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := cfgBase
			cfg.Config.ID = tt.env.walletName
			cfg.Credentials.Key = tt.env.walletKey
			if got := buildExportCredentials(&cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildExportCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}
