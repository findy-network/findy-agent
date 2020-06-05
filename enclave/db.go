package enclave

import (
	"errors"
	"fmt"

	"github.com/lainio/err2"
	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB

// ErrNotExists is an error for key not exist in the enclave.
var ErrNotExists = errors.New("key not exists")

// ErrSealBoxAlreadyExists is an error for enclave sealed box already exists.
var ErrSealBoxAlreadyExists = errors.New("enclave sealed box exists")

func assertDB() {
	if db == nil {
		panic("don't forget init the seal box")
	}
}

func open(filename string) (err error) {
	if db != nil {
		return ErrSealBoxAlreadyExists
	}
	defer err2.Return(&err)

	db, err = bolt.Open(filename, 0600, nil)
	err2.Check(err)

	err2.Check(db.Update(func(tx *bolt.Tx) (err error) {
		defer err2.Annotate("create buckets", &err)

		err2.Try(tx.CreateBucketIfNotExists([]byte(emailBucket)))
		err2.Try(tx.CreateBucketIfNotExists([]byte(didBucket)))
		err2.Try(tx.CreateBucketIfNotExists([]byte(masterSecretBucket)))
		return nil
	}))
	return err
}

// Close closes the sealed box of the enclave. It can be open again with
// InitSealedBox.
func Close() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println(err)
	})
	assertDB()

	err2.Check(db.Close())
	db = nil
}

func addKeyValueToBucket(bucket, keyValue, index string) (err error) {
	assertDB()

	defer err2.Annotate("add key", &err)

	err2.Check(db.Update(func(tx *bolt.Tx) (err error) {
		defer err2.Return(&err)

		b := tx.Bucket([]byte(bucket))
		err2.Check(b.Put([]byte(index), []byte(keyValue)))
		return nil
	}))
	return nil
}

func getKeyValueFromBucket(bucket, index string) (keyValue string, err error) {
	assertDB()

	defer err2.Return(&err)

	err2.Check(db.View(func(tx *bolt.Tx) (err error) {
		defer err2.Return(&err)

		b := tx.Bucket([]byte(bucket))
		d := b.Get([]byte(index))
		if d == nil {
			return ErrNotExists
		}
		keyValue = string(d)
		return nil
	}))
	return keyValue, nil
}
