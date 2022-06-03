package indy

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
)

type KMS struct {
	storage api.AgentStorage

	kms struct {
		sync.Mutex
		verkeys map[string]string
	}
}

func NewKMS(storage api.AgentStorage) *KMS {
	return &KMS{storage: storage,
		kms: struct {
			sync.Mutex
			verkeys map[string]string
		}{
			verkeys: make(map[string]string),
		}}
}

func (k *KMS) handle() int {
	return k.storage.(*Indy).Handle
}

func (k *KMS) Add(KID, verKey string) {
	k.kms.Lock()
	defer k.kms.Unlock()

	k.kms.verkeys[KID] = verKey
}

func (k *KMS) Create(kt kms.KeyType) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) Get(KID string) (interface{}, error) {
	k.kms.Lock()
	defer k.kms.Unlock()

	verKey := k.kms.verkeys[KID]

	return &Handle{
		Wallet: k.handle(),
		VerKey: verKey,
	}, nil
}

func (k *KMS) Rotate(kt kms.KeyType, KID string) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) ExportPubKeyBytes(KID string) ([]byte, kms.KeyType, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) CreateAndExportPubKeyBytes(kt kms.KeyType) (string, []byte, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) PubKeyBytesToHandle(pubKey []byte, kt kms.KeyType) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) ImportPrivateKey(privKey interface{}, kt kms.KeyType, opts ...kms.PrivateKeyOpts) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}
