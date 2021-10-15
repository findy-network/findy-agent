package didexchange

import (
	"reflect"
	"testing"
)

func TestDecodeB64(t *testing.T) {
	const strPadded = "c3VyZS4="
	const strUnpadded = "c3VyZS4"

	res1, err := decodeB64(strPadded)
	if err != nil {
		t.Errorf("error in padded decode = %v", err)
	}

	res2, err := decodeB64(strUnpadded)
	if err != nil {
		t.Errorf("error in unpadded decode = %v", err)
	}

	if !reflect.DeepEqual(res1, res2) {
		t.Errorf("not equal, is (%v), want (%v)", res1, res2)
	}
}
