package psm

import (
	"bytes"
	"errors"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/lainio/err2"
	bolt "go.etcd.io/bbolt"
)

const (
	bucketPSM   = "PSM"
	bucketRawPL = "RawPL"

	bucketPairwise     = "Pairwise"
	bucketDeviceID     = "DeviceID"
	bucketBasicMessage = "BasicMessage"
	bucketIssueCred    = "IssueCred"
	bucketPresentProof = "PresentProof"
)

var (
	psmBucket   = []byte(bucketPSM)
	rawPayloads = []byte(bucketRawPL)

	pairwiseReps     = []byte(bucketPairwise)
	deviceIDReps     = []byte(bucketDeviceID)
	basicMessageReps = []byte(bucketBasicMessage)
	issueCredReps    = []byte(bucketIssueCred)
	presentProofReps = []byte(bucketPresentProof)
)

func toBytes(s string) []byte {
	return []byte(s)
}

type DB struct {
	db *bolt.DB
}

func OpenDb(filename string) (db *DB, err error) {
	db = &DB{}
	err = db.Open(filename)
	return db, err
}

// Open opens the database by name of the file. If it is already open it returns
// it, but it doesn't check the database name it isn't thread safe!
func (b *DB) Open(filename string) (err error) {
	if b.db != nil {
		return nil
	}
	defer err2.Return(&err)
	b.db, err = bolt.Open(filename, 0600, nil)
	err2.Check(b.db.Update(func(tx *bolt.Tx) (err error) {
		defer err2.Annotate("create bucket", &err)

		// we could add bucket to agentBucket to store all of the REST api calls
		err2.Try(tx.CreateBucketIfNotExists(psmBucket))
		err2.Try(tx.CreateBucketIfNotExists(rawPayloads))
		err2.Try(tx.CreateBucketIfNotExists(pairwiseReps))
		err2.Try(tx.CreateBucketIfNotExists(deviceIDReps))
		err2.Try(tx.CreateBucketIfNotExists(basicMessageReps))
		err2.Try(tx.CreateBucketIfNotExists(issueCredReps))
		err2.Try(tx.CreateBucketIfNotExists(presentProofReps))
		return nil
	}))
	return err
}

func (b *DB) addData(key []byte, value []byte, bucketName string) (err error) {
	defer err2.Annotate("add "+bucketName, &err)
	err2.Check(b.db.Update(func(tx *bolt.Tx) (err error) {
		defer err2.Annotate("update "+bucketName+" bucket", &err)
		b := tx.Bucket(toBytes(bucketName))
		err2.Check(b.Put(key, value))
		return err
	}))
	return err
}

func (b *DB) get(k StateKey, bucketName string) (m []byte, err error) {
	defer err2.Annotate("get "+bucketName, &err)
	err2.Check(b.db.View(func(tx *bolt.Tx) (err error) {
		defer err2.Annotate("read "+bucketName, &err)
		b := tx.Bucket(toBytes(bucketName))
		got := b.Get(k.Data())
		// byte slices returned from Bolt are only valid during a transaction,
		// so copy result slice
		m = append(got[:0:0], got...)
		if m == nil {
			err = errors.New("Object was not found with id " + string(k.Data()))
		}
		return err
	}))
	return m, err
}

func (b *DB) AddRawPL(addr *endp.Addr, data []byte) (err error) {
	return b.addData(addr.Key(), data, bucketRawPL)
}

func (b *DB) RmRawPL(addr *endp.Addr) (err error) {
	defer err2.Annotate("rm rawPL", &err)
	err2.Check(b.db.Update(func(tx *bolt.Tx) (err error) {
		defer err2.Annotate("update bucket", &err)
		b := tx.Bucket(rawPayloads)
		err2.Check(b.Delete(addr.Key()))
		return nil
	}))
	return nil
}

func (b *DB) addPSM(p *PSM) (err error) {
	return b.addData(p.Key.Data(), p.Data(), bucketPSM)
}

func (b *DB) getPSM(k StateKey) (m *PSM, err error) {
	data, err := b.get(k, bucketPSM)
	if data != nil {
		m = NewPSM(data)
	}
	return m, err
}

func (b *DB) getAllPSM(did string, tsSinceNs *int64) (m *[]PSM, err error) {

	// TODO: pagination logic

	defer err2.Annotate("get all psm", &err)
	err2.Check(b.db.View(func(tx *bolt.Tx) (err error) {
		res := make([]PSM, 0)
		defer err2.Annotate("read", &err)
		c := tx.Bucket(psmBucket).Cursor()
		prefix := []byte(did)
		var vPsm *PSM
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if v != nil {
				vPsm = NewPSM(v)
				if tsSinceNs == nil || vPsm.Timestamp() >= *tsSinceNs {
					res = append(res, *vPsm)
				}
			}
		}
		m = &res
		return nil
	}))
	return m, nil
}

func (b *DB) GetPSM(key StateKey) (s *PSM, err error) {
	defer err2.Annotate("get last psm", &err)
	m, _ := b.getPSM(key)
	if m == nil {
		return nil, errors.New("PSM with key " + key.DID + "/" + key.Nonce + " not found")
	}
	return m, nil
}

func (b *DB) IsPSMReady(key StateKey) (yes bool, err error) {
	defer err2.Annotate("is ready", &err)

	m, err := b.GetPSM(key)
	err2.Check(err)
	if m == nil {
		return false, err
	}

	return m.IsReady(), nil
}

func (b *DB) AllPSM(did string, tsSinceNs *int64) (m *[]PSM, err error) {
	defer err2.Annotate("get all psm", &err)

	m, err = b.getAllPSM(did, tsSinceNs)
	err2.Check(err)
	if m == nil {
		return nil, errors.New("No PSMs found with " + did)
	}
	return m, nil
}

func (b *DB) AddPairwiseRep(p *PairwiseRep) (err error) {
	return b.addData(p.KData(), p.Data(), bucketPairwise)
}

func (b *DB) GetPairwiseRep(k StateKey) (m *PairwiseRep, err error) {
	data, err := b.get(k, bucketPairwise)
	if data != nil {
		m = NewPairwiseRep(data)
	}
	return m, err
}

func (b *DB) AddDeviceIDRep(d *DeviceIDRep) (err error) {
	return b.addData(d.Key(), d.Data(), bucketDeviceID)
}

func (b *DB) GetAllDeviceIDRep(did string) (m *[]DeviceIDRep, err error) {

	// TODO: pagination logic

	defer err2.Annotate("get all device IDs", &err)
	err2.Check(b.db.View(func(tx *bolt.Tx) (err error) {
		res := make([]DeviceIDRep, 0)
		defer err2.Annotate("read", &err)
		c := tx.Bucket(deviceIDReps).Cursor()
		prefix := []byte(did)
		var vDeviceID *DeviceIDRep
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if v != nil {
				vDeviceID = NewDeviceIDRep(v)
				res = append(res, *vDeviceID)
			}
		}
		m = &res
		return nil
	}))
	return m, nil
}

func (b *DB) AddBasicMessageRep(p *BasicMessageRep) (err error) {
	return b.addData(p.KData(), p.Data(), bucketBasicMessage)
}

func (b *DB) GetBasicMessageRep(k StateKey) (m *BasicMessageRep, err error) {
	data, err := b.get(k, bucketBasicMessage)
	if data != nil {
		m = NewBasicMessageRep(data)
	}
	return m, err
}

func (b *DB) AddIssueCredRep(p *IssueCredRep) (err error) {
	return b.addData(p.KData(), p.Data(), bucketIssueCred)
}

func (b *DB) GetIssueCredRep(k StateKey) (m *IssueCredRep, err error) {
	data, err := b.get(k, bucketIssueCred)
	if data != nil {
		m = NewIssueCredRep(data)
	}
	return m, err
}

func (b *DB) AddPresentProofRep(p *PresentProofRep) (err error) {
	return b.addData(p.KData(), p.Data(), bucketPresentProof)
}

func (b *DB) GetPresentProofRep(k StateKey) (m *PresentProofRep, err error) {
	data, err := b.get(k, bucketPresentProof)
	if data != nil {
		m = NewPresentProofRep(data)
	}
	return m, err
}
