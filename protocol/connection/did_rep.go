package connection

type didRep struct {
	DID    string
	VerKey string
	Wallet walletRep
	My     bool
	Endp   string
}
