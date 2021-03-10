package completionhelp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lainio/err2"
)

func WalletLocations() []string {
	defer err2.Catch(func(err error) {
		_, _ = fmt.Fprintln(os.Stderr, err)
	})

	home := err2.String.Try(os.UserHomeDir())
	indyWallets := filepath.Join(home, ".indy_client/wallet")

	return []string{indyWallets}
}
