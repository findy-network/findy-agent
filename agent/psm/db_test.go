package psm

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/findy-network/findy-common-go/crypto"
	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/go-test/deep"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dbPath  = "db_test.bolt"
	testKey = "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c"
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
	psm := testPSM(0)
	assert.NotNil(t, psm)

	k, err := hex.DecodeString(testKey)
	require.NoError(t, err)
	testCipher := crypto.NewCipher(k)

	tests := []struct {
		name   string
		cipher *crypto.Cipher
	}{
		{"add", nil},
		{"add with cipher", testCipher},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theCipher = tt.cipher

			err := AddPSM(psm)
			if err != nil {
				t.Errorf("AddPSM() %s error %v", tt.name, err)
			}
			got, err := GetPSM(StateKey{DID: mockStateDID, Nonce: mockStateNonce})
			if diff := deep.Equal(psm, got); err != nil || diff != nil {
				t.Errorf("GetPSM() diff %v, err %v", diff, err)
			}

			theCipher = nil
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
