// Package preview implements helpers for Aries issuing protocol.
package preview

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/data"
	"github.com/findy-network/findy-agent/std/issuecredential"
)

// StoreCredPreview copies credential attribute data to rep object
func StoreCredPreview(preview *issuecredential.PreviewCredential, rep *data.IssueCredRep) {
	rep.Attributes = make([]didcomm.CredentialAttribute, len(preview.Attributes))
	for index, value := range preview.Attributes {
		rep.Attributes[index] = didcomm.CredentialAttribute{
			Name:     value.Name,
			Value:    value.Value,
			MimeType: value.MimeType,
		}
	}
}
