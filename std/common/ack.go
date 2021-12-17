/*
copied from aries-framework-go

*/

package common

import "github.com/findy-network/findy-agent/std/decorator"

// Ack acknowledgement struct
type Ack struct {
	Type   string            `json:"@type,omitempty"`
	ID     string            `json:"@id,omitempty"`
	Status string            `json:"status,omitempty"`
	Thread *decorator.Thread `json:"~thread,omitempty"`
}
