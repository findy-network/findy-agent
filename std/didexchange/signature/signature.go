package signature

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

const connectionSigExpTime = 10 * 60 * 60

func Sign(r *didexchange.Response, pipe sec.Pipe) (err error) {
	r.ConnectionSignature, err = newConnectionSignature(r.Connection, pipe)
	return err
}

func Verify(r *didexchange.Response) (ok bool, err error) {
	r.Connection, err = verifySignature(r.ConnectionSignature, nil)
	ok = r.Connection != nil

	if ok {
		rawDID := common.ID(r.Connection.DIDDoc)
		r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	}

	return ok, err
}

func newConnectionSignature(connection *didexchange.Connection, pipe sec.Pipe) (cs *didexchange.ConnectionSignature, err error) {
	defer err2.Annotate("build connection sign", &err)

	connectionJSON := try.To1(json.Marshal(connection))

	signedData, signature, verKey := try.To3(pipe.SignAndStamp(connectionJSON))

	return &didexchange.ConnectionSignature{
		Type:       "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
		SignedData: base64.URLEncoding.EncodeToString(signedData),
		SignVerKey: verKey,
		Signature:  base64.URLEncoding.EncodeToString(signature),
	}, nil
}

func verifyTimestamp(data []byte) (timestamp int64, valid bool) {
	now := time.Now().Unix()
	tsIsValid := func(ts int64) bool {
		diff := now - ts
		return diff >= 0 && diff <= connectionSigExpTime
	}

	// preferred is big endian
	timestamp = int64(binary.BigEndian.Uint64(data))
	if tsIsValid(timestamp) {
		return timestamp, true
	}

	glog.Warningf("big endian encoded signature timestamp %s is invalid, try little endian", time.Unix(timestamp, 0))

	// accept also meaningful values found in little endian encoding
	// TODO: required format missing from spec
	// => confirm if we should support only preferred big endian
	timestamp = int64(binary.LittleEndian.Uint64(data))
	return timestamp, tsIsValid(timestamp)
}

// verifySignature verifies a signature inside the structure. If sec.Pipe is not
// given, it uses the key from the signature structure. If succeeded it returns
// a Connection structure, else nil.
func verifySignature(cs *didexchange.ConnectionSignature, pipe *sec.Pipe) (c *didexchange.Connection, err error) {
	defer err2.Annotate("verify sign", &err)

	if pipe != nil && pipe.Out.VerKey() != cs.SignVerKey {
		s := "programming error, we shouldn't be here"
		glog.Error(s)
		panic(s)
	} else if pipe == nil { // we need a tmp DID for a tmp Pipe
		did := ssi.NewDid("", cs.SignVerKey)
		pipe = &sec.Pipe{Out: did}
	}

	data := try.To1(utils.DecodeB64(cs.SignedData))
	if len(data) == 0 {
		s := "missing or invalid signature data"
		glog.Error(s)
		return nil, fmt.Errorf(s)
	}

	signature := try.To1(utils.DecodeB64(cs.Signature))

	ok, _ := try.To2(pipe.Verify(data, signature))
	if !ok {
		glog.Error("cannot verify signature")
		return nil, nil
	}

	timestamp, ok := verifyTimestamp(data)
	if !ok {
		glog.Errorln("connection signature timestamp is invalid: ", timestamp, time.Unix(timestamp, 0))
		// TODO: pass invalid timestamps on for now, as some agents do not fill it at all
		// should be fixed with new signature implementation
		// return nil, nil
	} else {
		glog.V(3).Info("verified connection signature w/ ts:", time.Unix(timestamp, 0))
	}

	connectionJSON := data[8:]

	var connection didexchange.Connection
	dto.FromJSON(connectionJSON, &connection)

	return &connection, nil
}
