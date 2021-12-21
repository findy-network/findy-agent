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
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
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
		rawDID := r.Connection.DIDDoc.ID
		r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	}

	return ok, err
}

func newConnectionSignature(connection *didexchange.Connection, pipe sec.Pipe) (cs *didexchange.ConnectionSignature, err error) {
	defer err2.Annotate("build connection sign", &err)

	connectionJSON := err2.Bytes.Try(json.Marshal(connection))

	signedData, signature, verKey := pipe.SignAndStamp(connectionJSON)

	return &didexchange.ConnectionSignature{
		Type:       "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
		SignedData: base64.URLEncoding.EncodeToString(signedData),
		SignVerKey: verKey,
		Signature:  base64.URLEncoding.EncodeToString(signature),
	}, nil
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

	data := err2.Bytes.Try(utils.DecodeB64(cs.SignedData))
	if len(data) == 0 {
		s := "missing or invalid signature data"
		glog.Error(s)
		return nil, fmt.Errorf(s)
	}

	signature := err2.Bytes.Try(utils.DecodeB64(cs.Signature))

	ok, _ := pipe.Verify(data, signature)
	if !ok {
		glog.Error("cannot verify signature")
		return nil, nil
	}

	timestamp := int64(binary.BigEndian.Uint64(data))
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 0 || diff > connectionSigExpTime {
		// try little endian - TODO: format missing from spec?
		glog.Warningf("signature timestamp %s is invalid for big endian encoding, try little endian", time.Unix(timestamp, 0))
		timestamp = int64(binary.LittleEndian.Uint64(data))
		diff = now - timestamp
	}

	if diff < 0 || diff > connectionSigExpTime {
		glog.Errorln("connection signature timestamp is invalid: ", timestamp, time.Unix(timestamp, 0))
		return nil, nil
	}

	glog.V(3).Info("verified connection signature w/ ts:", time.Unix(timestamp, 0))

	connectionJSON := data[8:]

	var connection didexchange.Connection
	dto.FromJSON(connectionJSON, &connection)

	return &connection, nil
}
