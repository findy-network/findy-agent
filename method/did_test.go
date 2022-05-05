package method

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMethodString(t *testing.T) {
	tests := []struct {
		did, method string
	}{
		{did: "did:key",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso#",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso:test#",
			method: "key"},
		{did: "did:sov:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "sov"},
		{did: "did:indy:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "indy"},
	}

	for i, tt := range tests {
		name := fmt.Sprintf("test_%d", i)
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.method, String(tt.did))
		})
	}
}

func TestDIDType(t *testing.T) {
	tests := []struct {
		name, did string
		Type
	}{
		{"did key only prefix",
			"did:key",
			TypeKey,
		},
		{"did key",
			"did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			TypeKey,
		},
		{"did peer",
			"did:peer:1zQmQSLFWySB3LACeSrUpvM48QN9frMayNHypnsQjk4GhQKG",
			TypePeer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.Type, DIDType(tt.did))
		})
	}
}
