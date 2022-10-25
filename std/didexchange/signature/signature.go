package signature

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/didexchange"
	didexchange0 "github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-agent/std/didexchange1"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

const connectionSigExpTime = 10 * 60 * 60

func Sign(r *didexchange0.Response, pipe sec.Pipe) (err error) {
	r.ConnectionSignature, err = newConnectionSignature(r.Connection, pipe)
	return err
}

func Verify(r *didexchange0.Response) (ok bool, err error) {
	r.Connection, err = verifySignature(r.ConnectionSignature, nil)
	ok = r.Connection != nil

	if ok {
		rawDID := common.ID(r.Connection.DIDDoc)
		r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	}

	return ok, err
}

func SignRequestV1(r *didexchange1.Request, ourDID core.DID) (err error) {
	c := ourDID.Packager().Crypto()
	kms := ourDID.Packager().KMS()
	kh := try.To1(kms.Get(ourDID.KID()))

	b58Key := ourDID.VerKey()
	pubKeyBytes := try.To1(base58.Decode(b58Key))
	pubKey := ed25519.PublicKey(pubKeyBytes)
	try.To(r.DIDDoc.Data.Sign(c, kh, pubKey, pubKeyBytes))
	return err
}

func VerifyRequestV1(r *didexchange1.Request, theirDID core.DID) (ok bool, err error) {
	// r.Connection, err = verifySignature(r.ConnectionSignature, nil)
	// ok = r.Connection != nil

	// if ok {
	// 	rawDID := common.ID(r.Connection.DIDDoc)
	// 	r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	// }
	// TODO
	try.To(r.DIDDoc.Data.Verify(theirDID.Packager().Crypto(), theirDID.Packager().KMS()))

	return ok, err
}

func SignResponseV1(r *didexchange1.Response, ourDID core.DID) (err error) {
	c := ourDID.Packager().Crypto()
	kms := ourDID.Packager().KMS()
	kh := try.To1(kms.Get(ourDID.KID()))

	b58Key := ourDID.VerKey()
	pubKeyBytes := try.To1(base58.Decode(b58Key))
	pubKey := ed25519.PublicKey(pubKeyBytes)
	try.To(r.DIDDoc.Data.Sign(c, kh, pubKey, pubKeyBytes))
	return err
}

func VerifyResponseV1(r *didexchange1.Response) (ok bool, err error) {
	// r.Connection, err = verifySignature(r.ConnectionSignature, nil)
	// ok = r.Connection != nil

	// if ok {
	// 	rawDID := common.ID(r.Connection.DIDDoc)
	// 	r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	// }

	return ok, err
}

func newConnectionSignature(connection *didexchange0.Connection, pipe sec.Pipe) (cs *didexchange0.ConnectionSignature, err error) {
	defer err2.Returnf(&err, "build connection sign")

	connectionJSON := try.To1(json.Marshal(connection))

	signedData, signature, verKey := try.To3(pipe.SignAndStamp(connectionJSON))

	return &didexchange0.ConnectionSignature{
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
	defer err2.Returnf(&err, "verify sign")

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
		// don't pollute logs with errors when we aren't treating this as an
		// error for now
		glog.Warningln("connection signature timestamp is invalid: ", timestamp, time.Unix(timestamp, 0))
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
