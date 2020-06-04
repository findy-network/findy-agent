package psm

type DIDRep struct {
	DID    string
	VerKey string
	Wallet WalletRep
	My     bool
	Endp   string
}
