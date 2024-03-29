package indy

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/mr-tron/base58"
)

type kmsStore struct {
	sync.RWMutex
	verKeyByKeyID map[string]string
	keyIDByVerKey map[string]string
}

type KMS struct {
	storage api.AgentStorage

	kms kmsStore
}

func NewKMS(storage api.AgentStorage) *KMS {
	return &KMS{storage: storage,
		kms: kmsStore{
			verKeyByKeyID: make(map[string]string),
			keyIDByVerKey: make(map[string]string),
		}}
}

func (k *KMS) handle() int {
	return k.storage.(*Indy).Handle
}

func (k *KMS) Add(KID, verKey string) {
	k.kms.Lock()
	defer k.kms.Unlock()

	k.kms.verKeyByKeyID[KID] = verKey
	k.kms.keyIDByVerKey[verKey] = KID
}

func (k *KMS) Create(_ kms.KeyType, _ ...kms.KeyOpts) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) Get(KID string) (interface{}, error) {
	k.kms.RLock()
	defer k.kms.RUnlock()

	verKey, ok := k.kms.verKeyByKeyID[KID]
	var handle *Handle

	if ok {
		handle = &Handle{
			Wallet: k.handle(),
			VerKey: verKey,
		}
	}

	return handle, nil
}

func (k *KMS) Rotate(_ kms.KeyType, _ string, _ ...kms.KeyOpts) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) ExportPubKeyBytes(_ string) ([]byte, kms.KeyType, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) CreateAndExportPubKeyBytes(_ kms.KeyType, _ ...kms.KeyOpts) (string, []byte, error) {
	//TODO implement me
	panic("implement me")
}

func (k *KMS) getKeyIDByVerKey(verKey string) string {
	k.kms.RLock()
	defer k.kms.RUnlock()
	keyID := k.kms.keyIDByVerKey[verKey]
	return keyID
}

func (k *KMS) PubKeyBytesToHandle(pubKey []byte, _ kms.KeyType, _ ...kms.KeyOpts) (interface{}, error) {
	verKey := base58.Encode(pubKey)
	keyID := k.getKeyIDByVerKey(verKey)
	if keyID == "" {
		keyID = verKey
		k.Add(keyID, verKey)
	}
	return k.Get(keyID)
}

func (k *KMS) ImportPrivateKey(_ interface{}, _ kms.KeyType, _ ...kms.PrivateKeyOpts) (string, interface{}, error) {
	//TODO implement me
	panic("implement me")
}
