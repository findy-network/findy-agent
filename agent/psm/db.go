package psm

import (
	"crypto/md5"
	"errors"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-common-go/crypto"
	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

const (
	bucketPSM byte = 0 + iota
	bucketRawPL
	bucketPairwise
	bucketDeviceID
	bucketBasicMessage
	bucketIssueCred
	bucketPresentProof
)

var (
	buckets = [][]byte{
		{bucketPSM},
		{bucketRawPL},
		{bucketPairwise},
		{bucketDeviceID},
		{bucketBasicMessage},
		{bucketIssueCred},
		{bucketPresentProof},
	}

	theCipher *crypto.Cipher

	mgdDB *db.Mgd
)

// Open opens the database by name of the file. If it is already open it returns
// it, but it doesn't check the database name it isn't thread safe!
func Open(filename string) (err error) {
	mgdDB = db.New(db.Cfg{
		Filename:   filename,
		Buckets:    buckets,
		BackupName: filename + "_backup",
	})
	return nil
}

func addData(key []byte, value []byte, bucketID byte) (err error) {
	return mgdDB.AddKeyValueToBucket(buckets[bucketID],
		&db.Data{
			Data: value,
			Read: hash,
		},
		&db.Data{
			Data: key,
			Read: encrypt,
		},
	)
}

// get executes a read transaction by a key and a bucket. Instead of returning
// the data, it uses lambda for the result transport to prevent cloning the byte
// slice.
func get(
	k StateKey,
	bucketID byte,
	use func(d []byte),
) (
	found bool,
	err error,
) {
	value := &db.Data{
		Write: decrypt,
		Use: func(d []byte) interface{} {
			use(d)
			return nil
		},
	}
	found, err = mgdDB.GetKeyValueFromBucket(buckets[bucketID],
		&db.Data{
			Data: k.Data(),
			Read: hash,
		},
		value)

	return found, err
}

func rm(k StateKey, bucketID byte) (err error) {
	return mgdDB.RmKeyValueFromBucket(buckets[bucketID],
		&db.Data{
			Data: k.Data(),
			Read: hash,
		})
}

func AddRawPL(addr *endp.Addr, data []byte) (err error) {
	return addData(addr.Key(), data, bucketRawPL)
}

func RmRawPL(addr *endp.Addr) (err error) {
	return mgdDB.RmKeyValueFromBucket(buckets[bucketRawPL],
		&db.Data{
			Data: addr.Key(),
			Read: hash,
		})
}

func AddPSM(p *PSM) (err error) {
	return addData(p.Key.Data(), p.Data(), bucketPSM)
}

func getPSM(k StateKey) (m *PSM, err error) {
	found := false
	found, err = get(k, bucketPSM, func(d []byte) {
		m = NewPSM(d)
	})
	if !found {
		glog.Warningln("getPSM cannot find machine")
		m = nil
	}
	return m, err
}

func GetPSM(key StateKey) (s *PSM, err error) {
	defer err2.Annotate("get last psm", &err)
	m, _ := getPSM(key)
	if m == nil {
		return nil, errors.New("PSM with key " + key.DID + "/" + key.Nonce + " not found")
	}
	return m, nil
}

func IsPSMReady(key StateKey) (yes bool, err error) {
	defer err2.Annotate("is ready", &err)

	m, err := GetPSM(key)
	err2.Check(err)
	if m == nil {
		return false, err
	}

	return m.IsReady(), nil
}

func AddPairwiseRep(p *PairwiseRep) (err error) {
	return addData(p.KData(), p.Data(), bucketPairwise)
}

func GetPairwiseRep(k StateKey) (m *PairwiseRep, err error) {
	_, err = get(k, bucketPairwise, func(d []byte) {
		m = NewPairwiseRep(d)
	})
	return m, err
}

func AddDeviceIDRep(d *DeviceIDRep) (err error) {
	return addData(d.Key(), d.Data(), bucketDeviceID)
}

func AddBasicMessageRep(p *BasicMessageRep) (err error) {
	return addData(p.KData(), p.Data(), bucketBasicMessage)
}

func GetBasicMessageRep(k StateKey) (m *BasicMessageRep, err error) {
	_, err = get(k, bucketBasicMessage, func(d []byte) {
		m = NewBasicMessageRep(d)
	})
	return m, err
}

func AddIssueCredRep(p *IssueCredRep) (err error) {
	return addData(p.KData(), p.Data(), bucketIssueCred)
}

func GetIssueCredRep(k StateKey) (m *IssueCredRep, err error) {
	_, err = get(k, bucketIssueCred, func(d []byte) {
		m = NewIssueCredRep(d)
	})
	return m, err
}

func AddPresentProofRep(p *PresentProofRep) (err error) {
	return addData(p.KData(), p.Data(), bucketPresentProof)
}

func GetPresentProofRep(k StateKey) (m *PresentProofRep, err error) {
	_, err = get(k, bucketPresentProof, func(d []byte) {
		m = NewPresentProofRep(d)
	})
	return m, err
}

func RmPSM(p *PSM) (err error) {
	glog.V(1).Infoln("--- rm PSM:", p.Key)
	switch p.Protocol() {
	case pltype.ProtocolBasicMessage:
		err = rm(p.Key, bucketBasicMessage)
	case pltype.ProtocolConnection:
		err = rm(p.Key, bucketPairwise)
	case pltype.ProtocolIssueCredential:
		err = rm(p.Key, bucketIssueCred)
	case pltype.ProtocolPresentProof:
		err = rm(p.Key, bucketPresentProof)
	}
	if err != nil {
		return err
	}
	return rm(p.Key, bucketPSM)
}

// all of the following has same signature. They also panic on error

// hash makes the cryptographic hash of the map key value. This prevents us to
// store key value index (email, DID) to the DB aka sealed box as plain text.
// Please use salt when implementing this.
func hash(key []byte) (k []byte) {
	if theCipher != nil {
		h := md5.Sum(key)
		return h[:]
	}
	return append(key[:0:0], key...)
}

// encrypt encrypts the actual wallet key value. This is used when data is
// stored do the DB aka sealed box.
func encrypt(value []byte) (k []byte) {
	if theCipher != nil {
		return theCipher.TryEncrypt(value)
	}
	return append(value[:0:0], value...)
}

// decrypt decrypts the actual wallet key value. This is used when data is
// retrieved from the DB aka sealed box.
func decrypt(value []byte) (k []byte) {
	if theCipher != nil {
		return theCipher.TryDecrypt(value)
	}
	return append(value[:0:0], value...)
}

// noop function if need e.g. tests
func _(value []byte) (k []byte) {
	println("noop called!")
	return value
}
