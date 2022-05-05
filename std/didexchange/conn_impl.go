package didexchange

import (
	"github.com/lainio/err2"
)

func (c *Connection) ToJSON() (b []byte, err error) {
	defer err2.Return(&err)

	return nil, nil

	//	c.DIDDoc = try.To1(c.Doc.JSONBytes())

	//	return json.Marshal(c)
}
