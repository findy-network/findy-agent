package psm

import (
	"crypto/md5"
	"fmt"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-common-go/crypto"
	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/golang/glog"
	"github.com/lainio/err2/assert"
)

const (
	BucketPSM byte = 0 + iota
	BucketRawPL
	BucketPairwise
	BucketBasicMessage
	BucketIssueCred
	BucketPresentProof
)

var (
	buckets = [][]byte{
		{BucketPSM},
		{BucketRawPL},
		{BucketPairwise},
		{BucketBasicMessage},
		{BucketIssueCred},
		{BucketPresentProof},
	}

	theCipher *crypto.Cipher

	mgdDB db.Handle
)

type Rep interface {
	Key() StateKey
	Data() []byte
	Type() byte
}

type CreatorFunc func(d []byte) (rep Rep)

var Creator = &Factor{factors: make(map[byte]CreatorFunc)}

type Factor struct {
	factors map[byte]CreatorFunc
}

func (f *Factor) Add(t byte, factor CreatorFunc) {
	f.factors[t] = factor
}

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

func Close() {
	mgdDB.Close()
}

func addData(key []byte, value []byte, bucketID byte) (err error) {
	return mgdDB.AddKeyValueToBucket(buckets[bucketID],
		&db.Data{
			Data: value,
			Read: encrypt,
		},
		&db.Data{
			Data: key,
			Read: hash,
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
	return addData(addr.Key(), data, BucketRawPL)
}

func RmRawPL(addr *endp.Addr) (err error) {
	return mgdDB.RmKeyValueFromBucket(buckets[BucketRawPL],
		&db.Data{
			Data: addr.Key(),
			Read: hash,
		})
}

func AddPSM(p *PSM) (err error) {
	return addData(p.Key.Data(), p.Data(), BucketPSM)
}

// GetPSM get existing PSM from DB. If the PSM doesn't exist it returns error.
// See FindPSM for version which doesn't return error if the PSM doesn't exist.
func GetPSM(k StateKey) (m *PSM, err error) {
	var found bool
	found, err = get(k, BucketPSM, func(d []byte) {
		m = NewPSM(d)
	})
	if !found {
		assert.That(m == nil)
		return nil, fmt.Errorf("PSM with key %s/%s not found", k.DID, k.Nonce)
	}
	return m, err
}

// FindPSM doesn't return error if the PSM doesn't exist. Instead the returned
// PSM is nil.
func FindPSM(k StateKey) (m *PSM, err error) {
	_, err = get(k, BucketPSM, func(d []byte) {
		m = NewPSM(d)
	})
	return m, err
}

func AddRep(p Rep) (err error) {
	return addData(p.Key().Data(), p.Data(), p.Type())
}

func GetRep(repType byte, k StateKey) (m Rep, err error) {
	_, err = get(k, repType, func(d []byte) {
		factor, ok := Creator.factors[repType]
		if !ok {
			err = fmt.Errorf("no factor found for rep type %d", repType)
			return
		}
		m = factor(d)
	})
	return m, err
}

func RmPSM(p *PSM) (err error) {
	glog.V(1).Infoln("--- rm PSM:", p.Key)
	switch p.Protocol() {
	case pltype.ProtocolBasicMessage:
		err = rm(p.Key, BucketBasicMessage)
	case pltype.ProtocolConnection:
		err = rm(p.Key, BucketPairwise)
	case pltype.ProtocolIssueCredential:
		err = rm(p.Key, BucketIssueCred)
	case pltype.ProtocolPresentProof:
		err = rm(p.Key, BucketPresentProof)
	}
	if err != nil {
		return err
	}
	return rm(p.Key, BucketPSM)
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
