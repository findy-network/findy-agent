/*
copied from aries-framework-go

*/

package common

// Forward route forward message.
// nolint:lll // url in the next line is long
// https://github.com/hyperledger/aries-rfcs/blob/main/concepts/0094-cross-domain-messaging/README.md#corerouting10forward
type Forward struct {
	Type string                 `json:"@type,omitempty"`
	ID   string                 `json:"@id,omitempty"`
	To   string                 `json:"to,omitempty"`
	Msg  map[string]interface{} `json:"msg,omitempty"`
}
