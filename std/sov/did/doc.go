package did

import (
	"encoding/json"
	"time"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Doc struct {
	*DataDoc
}

func (d *Doc) MarshalJSON() (_ []byte, err error) {
	defer err2.Returnf(&err, "marshal sov doc")

	b := try.To1(json.Marshal(d.DataDoc))
	return b, nil
}

func (d *Doc) UnmarshalJSON(b []byte) (err error) {
	defer err2.Returnf(&err, "unmarshal sov doc")

	data := new(DataDoc)
	try.To(json.Unmarshal(b, data))
	d.DataDoc = data
	return nil
}

// Doc DataDID Document definition
type DataDoc struct {
	Context        string               `json:"@context,omitempty"`
	ID             string               `json:"id,omitempty"`
	PublicKey      []PublicKey          `json:"publicKey,omitempty"`
	Service        []Service            `json:"service,omitempty"`
	Authentication []VerificationMethod `json:"authentication,omitempty"`
	Created        *time.Time           `json:"created,omitempty"`
	Updated        *time.Time           `json:"updated,omitempty"`
	//	Proof          []Proof
}

// PublicKey DID doc public key
type PublicKey struct {
	ID              string `json:"id,omitempty"`
	Type            string `json:"type,omitempty"`
	Controller      string `json:"controller,omitempty"`
	PublicKeyBase58 string `json:"publicKeyBase58,omitempty"`
	// Value      []byte `json:"value,omitempty"`
}

// Service DID doc service
type Service struct {
	ID              string                 `json:"id,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Priority        uint                   `json:"priority,omitempty"`
	RecipientKeys   []string               `json:"recipientKeys,omitempty"`
	RoutingKeys     []string               `json:"routingKeys,omitempty"`
	ServiceEndpoint string                 `json:"serviceEndpoint"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

// VerificationMethod authentication verification method
type VerificationMethod struct {
	Type      string `json:"type,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
	//	PublicKey `json:"public_key,omitempty"`
}

func _(did core.DID, ae service.Addr) *DataDoc {
	didURI := did.URI()
	didURIRef := didURI + "#1"
	pubK := PublicKey{
		ID:              didURIRef,
		Type:            "Ed25519VerificationKey2018",
		Controller:      didURI,
		PublicKeyBase58: did.VerKey(),
	}
	service := Service{
		ID:              didURI,
		Type:            "IndyAgent",
		Priority:        0,
		RecipientKeys:   []string{did.VerKey()},
		ServiceEndpoint: ae.Endp,
	}
	return &DataDoc{
		Context:   "https://w3id.org/did/v1",
		ID:        didURI,
		PublicKey: []PublicKey{pubK},
		Service:   []Service{service},
		Authentication: []VerificationMethod{{
			Type:      "Ed25519SignatureAuthentication2018",
			PublicKey: didURIRef,
		}},
	}
}
