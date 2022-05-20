package didexchange_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	sov "github.com/findy-network/findy-agent/std/sov/did"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/require"
)

func TestConnection_ReadDoc(t *testing.T) {
	err2.StackTraceWriter = os.Stderr
	defer err2.CatchTrace(func(err error) {
		t.Error(err)
	})

	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args
		newFormat bool
	}{
		{"w3c sample", args{"./json/w3c-doc-sample.json"}, true},
		{"sov from afgo", args{"./json/sov.json"}, true},
		{"our peer did doc", args{"./json/our-peer-did-doc.json"}, true},
		{"acapy 160", args{"json/160-acapy.json"}, false},
		{"acapy", args{"json/sov/acapy.json"}, false},

		// AFGO format still has its problems
		//{"afgo def", args{"json/afgo-default.json"}, false},
		//{"afgo interop", args{"json/afgo-interop.json"}, false},

		{"dotnet", args{"json/dotnet.json"}, true},
		{"ours", args{"json/ours-160-prepared.json"}, true},
		{"js", args{"json/javascript.json"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc did.Doc
			d, err := ioutil.ReadFile(tt.filename)
			require.NoError(t, err)

			if tt.newFormat {
				require.NoError(t, json.Unmarshal(d, &doc))
			} else {
				doc := sov.Doc{}
				require.NoError(t, json.Unmarshal(d, &doc))

				b, err := json.Marshal(doc)
				require.NoError(t, err)
				require.NotNil(t, b)

				// second read from JSON
				require.NoError(t, json.Unmarshal(d, &doc))
			}
		})
	}
}
