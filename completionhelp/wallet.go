package completionhelp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

func WalletLocations() []string {
	defer err2.Catch(err2.Err(func(err error) {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}))

	home := try.To1(os.UserHomeDir())
	indyWallets := filepath.Join(home, ".indy_client/wallet")

	return []string{indyWallets}
}
