package mgddb

import (
	"os"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/wrapper"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const (
	NameKey        = "kmsdb"
	NameDID        = "did"
	NameConnection = "connection"
	NameCredential = "credential"

	NameVDRPeer = "peer"
)

var bucketIDs = []string{
	NameKey,
	NameDID,
	NameConnection,
	NameCredential,
	NameVDRPeer,
}

type Storage struct {
	*wrapper.StorageProvider
	keyStorage *kmsStorage
	didStore   wrapper.Store
	connStore  wrapper.Store
}

func New(config api.AgentStorageConfig) (a *Storage, err error) {
	defer err2.Annotate("afgo storage new", &err)

	try.To(os.MkdirAll(config.FilePath, os.ModePerm))

	me := &Storage{
		wrapper.New(wrapper.Config{
			Key:       config.AgentKey,
			FileName:  config.AgentID,
			FilePath:  config.FilePath,
			BucketIDs: bucketIDs,
		}),
		nil,
		nil,
		nil,
	}

	try.To(me.Init())

	keyStorage := try.To1(newKmsStorage(me))
	me.keyStorage = keyStorage

	var ok bool
	didStore := try.To1(me.OpenStore(NameDID))
	me.didStore, ok = didStore.(wrapper.Store)
	assert.D.True(ok, "did store should always be wrapper store")

	connStore := try.To1(me.OpenStore(NameConnection))
	me.connStore, ok = connStore.(wrapper.Store)
	assert.D.True(ok, "conn store should always be wrapper store")

	return me, nil
}

func GenerateKey() string {
	// TODO
	return "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c"
}

// agent storage
func (s *Storage) Open() error {
	return s.Init()
}

func (s *Storage) KMS() kms.KeyManager {
	return s.keyStorage.KMS()
}

func (s *Storage) DIDStorage() api.DIDStorage {
	return s
}

func (s *Storage) ConnectionStorage() api.ConnectionStorage {
	return s
}

func (s *Storage) CredentialStorage() api.CredentialStorage {
	// TODO
	return nil
}

// DIDStorage
func (s *Storage) SaveDID(did api.DID) (err error) {
	return s.didStore.Put(did.ID, dto.ToGOB(did))
}

func (s *Storage) GetDID(id string) (did *api.DID, err error) {
	defer err2.Annotate("did storage get did", &err)

	bytes := try.To1(s.didStore.Get(id))

	did = &api.DID{}
	dto.FromGOB(bytes, did)
	return
}

// ConnectionStorage
func (s *Storage) SaveConnection(conn api.Connection) error {
	return s.connStore.Put(conn.ID, dto.ToGOB(conn))
}

func (s *Storage) GetConnection(id string) (conn *api.Connection, err error) {
	defer err2.Annotate("conn storage get conn", &err)

	bytes := try.To1(s.connStore.Get(id))

	conn = &api.Connection{}
	dto.FromGOB(bytes, conn)
	return

}

func (s *Storage) ListConnections() (res []api.Connection, err error) {
	defer err2.Annotate("conn storage list conn", &err)

	res = make([]api.Connection, 0)
	conn := &api.Connection{}
	try.To1(s.connStore.GetAll(func(bytes []byte) []byte {
		dto.FromGOB(bytes, conn)
		res = append(res, *conn)
		return bytes
	}))

	return res, nil
}
