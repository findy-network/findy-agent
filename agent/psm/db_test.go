package psm

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/go-test/deep"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/assert"
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
	err2.Check(flag.Set("logtostderr", "true"))

	err2.Check(Open(dbPath))
}

func tearDown() {
	db.Close()

	os.Remove(dbPath)
}

func Test_addPSM(t *testing.T) {
	psm := testPSM(0)
	assert.NotNil(t, psm)

	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPSM(psm)
			if err != nil {
				t.Errorf("AddPSM() %s error %v", tt.name, err)
			}
			got, err := GetPSM(StateKey{DID: mockStateDID, Nonce: mockStateNonce})
			if diff := deep.Equal(psm, got); err != nil || diff != nil {
				t.Errorf("GetPSM() diff %v, err %v", diff, err)
			}
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
			err := AddRep(msgRep)
			if err != nil {
				t.Errorf("AddRep() %s error %v", tt.name, err)
			}
			got, err := GetRep(msgRep.Type(), msgRep.StateKey)
			if diff := deep.Equal(msgRep, got); err != nil || diff != nil {
				t.Errorf("GetRep() diff %v, err %v", diff, err)
			}
		})
	}
}
