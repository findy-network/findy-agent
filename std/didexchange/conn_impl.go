package didexchange

import (
	"encoding/json"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

func (c *Connection) ToJSON() (b []byte, err error) {
	defer err2.Return(&err)

	c.DIDDoc = try.To1(c.Doc.JSONBytes())

	return json.Marshal(c)
}
