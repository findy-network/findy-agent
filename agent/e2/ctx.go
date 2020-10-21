package e2

import (
	"context"

	"github.com/lainio/err2"
)

type _Ctx struct{}

// Ctx is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Ctx _Ctx

// Try is a helper method to call func() (context.Context, error) functions
// with it and be as fast as Check(err).
func (o _Ctx) Try(v context.Context, err error) context.Context {
	err2.Check(err)
	return v
}
