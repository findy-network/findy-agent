package psm

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const (
	dbPath = "db_test.bolt"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})

	// We don't want logs on file with tests
	try.To(flag.Set("logtostderr", "true"))

	try.To(Open(dbPath))
}

func tearDown() {
	db.Close()

	os.Remove(dbPath)
}

func Test_addPSM(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	psm := testPSM(0)
	assert.NotNil(psm)

	tests := []struct {
		name string
	}{
		{"add"},
		{"add with cipher"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()

			err := AddPSM(psm)
			assert.NoError(err)

			got, err := GetPSM(StateKey{DID: mockStateDID, Nonce: mockStateNonce})
			assert.NoError(err)
			assert.DeepEqual(psm, got)
		})
	}
}

type testRep struct {
	StateKey
}

func (t *testRep) Key() StateKey {
	return t.StateKey
}

func (t *testRep) Data() []byte {
	return dto.ToGOB(t)
}

func (t *testRep) Type() byte {
	return BucketPSM // just use any type
}

func NewTestRep(d []byte) Rep {
	p := &testRep{}
	dto.FromGOB(d, p)
	return p
}

func Test_addBaseRep(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	msgRep := &testRep{
		StateKey: StateKey{DID: mockStateDID, Nonce: mockStateNonce},
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	Creator.Add(BucketPSM, NewTestRep)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()

			err := AddRep(msgRep)
			assert.NoError(err)

			got, err := GetRep(msgRep.Type(), msgRep.StateKey)
			assert.NoError(err)
			assert.DeepEqual(msgRep, got)
		})
	}
}

func Test_Close(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	path := "close-" + dbPath
	err := Open(path)
	assert.INil(err)

	Close()
}
