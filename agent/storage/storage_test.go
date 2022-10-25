package storage

import (
	"flag"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/cfg"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type storageTest struct {
	name    string
	config  cfg.AgentStorage
	storage api.AgentStorage
}

var (
	kmsTestStorages []*storageTest
	afgoTestStorage *mgddb.Storage
)

const (
	nameAfgo = "afgo"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	try.To(flag.Set("logtostderr", "true"))
	try.To(flag.Set("stderrthreshold", "WARNING"))
	try.To(flag.Set("v", "10"))
	flag.Parse()

	kmsTestStorages = make([]*storageTest, 0)
	// AFGO
	kmsTestConfig := cfg.AgentStorage{AgentStorageConfig: api.AgentStorageConfig{
		AgentKey: mgddb.GenerateKey(),
		AgentID:  "MEMORY_agentID",
		FilePath: ".",
	}}

	afgoTestStorage = try.To1(mgddb.New(kmsTestConfig.AgentStorageConfig))

	kmsTestStorages = append(kmsTestStorages, &storageTest{
		name:    nameAfgo,
		config:  kmsTestConfig,
		storage: afgoTestStorage,
	})
}

func tearDown() {
	for _, testStorage := range kmsTestStorages {
		if err := testStorage.storage.Close(); err != nil {
			panic(err)
		}
		// afgo
		_ = os.RemoveAll(testStorage.config.AgentID + ".bolt")
	}
}

func TestConcurrentOpen(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			err := testCase.storage.Close()
			assert.NoError(err)

			wg := &sync.WaitGroup{}
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					err := testCase.storage.Open()
					assert.NoError(err)

					store := testCase.storage.KMS()
					keyID, keyBytes, err := store.CreateAndExportPubKeyBytes(kms.ED25519Type)
					assert.NoError(err)
					assert.NotEmpty(keyID)
					assert.SNotEmpty(keyBytes)
				}()
			}
			wg.Wait()
		})
	}
}

func TestDIDStore(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.DIDStorage()
			testDID := api.DID{
				ID:  "did:test:123",
				DID: "did:test:123",
			}
			err := store.SaveDID(testDID)
			assert.NoError(err)

			gotDID, err := store.GetDID(testDID.ID)
			assert.NoError(err)
			assert.DeepEqual(testDID, *gotDID)
		})
	}
}

func TestConnectionStore(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.ConnectionStorage()
			testConn := api.Connection{
				ID:            "123-uid",
				MyDID:         "did:test:123",
				TheirDID:      "did:test:456",
				TheirEndpoint: "https://example.com",
				TheirRoute:    []string{"routeKey"},
			}
			err := store.SaveConnection(testConn)
			assert.NoError(err)

			gotConn, err := store.GetConnection(testConn.ID)
			assert.NoError(err)
			assert.DeepEqual(testConn, *gotConn)

			testConn2 := testConn
			testConn2.ID = "456-uid"
			err = store.SaveConnection(testConn2)
			assert.NoError(err)

			conns, err := store.ListConnections()
			assert.NoError(err)
			assert.SLen(conns, 2)
			// key value storage doesn't guarantee order of the connections
			if reflect.DeepEqual(testConn, conns[0]) {
				assert.DeepEqual(testConn2, conns[1])
			} else {
				assert.DeepEqual(testConn, conns[1])
				assert.DeepEqual(testConn2, conns[0])
			}
		})
	}
}
