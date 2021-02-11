package managed

// Wallet is a helper interface for managed wallets. You should always use this
// type instead of plain old indy SDK wallet handle. You present wallet
// configurations with ssi.Wallet and open them with ssi.Wallets.Open().
type Wallet interface {
	Close()
	Handle() int
	Config() WalletCfg
}

type Identifier interface {
	UniqueID() string
}

type WalletCfg interface {
	Identifier
	ID() string
	Key() string
}
