package v0_test

import (
	"encoding/json"
	"os"
	"testing"

	sov "github.com/findy-network/findy-agent/std/sov/did"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

func TestConnection_ReadDoc(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	err2.SetTracers(os.Stderr)
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
			assert.PushTester(t)
			defer assert.PopTester()
			var doc did.Doc
			d, err := os.ReadFile(tt.filename)
			assert.NoError(err)

			if tt.newFormat {
				assert.NoError(json.Unmarshal(d, &doc))
			} else {
				doc := sov.Doc{}
				assert.NoError(json.Unmarshal(d, &doc))

				b, err := json.Marshal(doc)
				assert.NoError(err)
				assert.INotNil(b)

				// second read from JSON
				assert.NoError(json.Unmarshal(d, &doc))
			}
		})
	}
}
