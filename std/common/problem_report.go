/*
copied from aries-framework-go
*/

package common

import "github.com/optechlab/findy-agent/std/decorator"

// ProblemReport problem report definition
// TODO: need to provide full ProblemReport structure https://github.com/hyperledger/aries-framework-go/issues/912
type ProblemReport struct {
	Type           string            `json:"@type"`
	ID             string            `json:"@id"`
	Description    Code              `json:"description"`
	ExplainLongTxt string            `json:"explain-ltxt,omitempty"` // ACApy
	Thread         *decorator.Thread `json:"~thread,omitempty"`
}

// Code represents a problem report code
type Code struct {
	Code string `json:"code"`
}
