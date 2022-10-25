package vdr_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"os"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/cfg"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	myvdr "github.com/findy-network/findy-agent/agent/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type testCase struct {
	name     string
	registry vdr.VDR
}

var (
	storageTestConfig = cfg.AgentStorage{AgentStorageConfig: api.AgentStorageConfig{
		AgentKey: mgddb.GenerateKey(),
		AgentID:  "MEMORY_agentID",
		FilePath: ".",
	}}
	afgoTestStorage *mgddb.Storage
	tests           []testCase
	testKey         []byte
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

	// AFGO
	var err error
	afgoTestStorage, err = mgddb.New(storageTestConfig.AgentStorageConfig)
	assert.D.True(err == nil)
	assert.D.True(afgoTestStorage != nil)

	testVdr, err := myvdr.New(afgoTestStorage)
	assert.D.True(err == nil)
	assert.D.True(testVdr != nil)

	tests = append(
		tests,
		testCase{
			name:     "key",
			registry: testVdr.Key(),
		},
		testCase{
			name:     "peer",
			registry: testVdr.Peer(),
		},
	)

	testKey, _, err = ed25519.GenerateKey(rand.Reader)
	assert.D.True(err == nil)
}

func tearDown() {
	err := afgoTestStorage.Close()
	assert.D.True(err == nil)

	_ = os.RemoveAll(storageTestConfig.AgentID + ".bolt")
}

func TestVDRAccept(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			assert.That(tt.registry.Accept(tt.name))
			assert.ThatNot(tt.registry.Accept("invalid"))
		})
	}
}

func TestVDRCreateAndRead(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	doc := &did.Doc{
		VerificationMethod: []did.VerificationMethod{{
			Type:  "Ed25519VerificationKey2018",
			Value: testKey,
		}},
		Service: []did.Service{{
			Type:            "DidcCommServiceType",
			ServiceEndpoint: "http://example.com",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			docResolution, err := tt.registry.Create(doc)
			assert.NoError(err)
			assert.INotNil(docResolution)

			docResolution, err = tt.registry.Read(docResolution.DIDDocument.ID)
			assert.NoError(err)
			assert.INotNil(docResolution)
		})
	}
}

func TestVDRUpdate(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			assert.Equal(tt.registry.Update(nil).Error(), "not supported")
		})
	}
}

func TestVDRDeactivate(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			assert.Equal(tt.registry.Deactivate("").Error(), "not supported")
		})
	}
}

func TestVDRClose(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			assert.NoError(tt.registry.Close())
		})
	}
}
