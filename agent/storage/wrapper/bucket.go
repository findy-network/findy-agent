package wrapper

import (
	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/lainio/err2"
)

type bucket struct {
	bucketID byte
	owner    *StorageProvider
}

func newBucket(owner *StorageProvider, bucketID byte) bucket {
	return bucket{
		owner:    owner,
		bucketID: bucketID,
	}
}

// Put stores the key + value pair along with the (optional) tags.
// If key is empty or value is nil, then an error will be returned.
func (b *bucket) Put(key string, value []byte, tags ...storage.Tag) (err error) {
	glog.V(7).Infoln("bucket::Put", key, tags)

	if len(tags) > 0 {
		panic("tags not supported")
	}

	return b.owner.addData(b.bucketID, []byte(key), value)
}

// Get fetches the value associated with the given key.
// If key cannot be found, then an error wrapping ErrDataNotFound will be returned.
// If key is empty, then an error will be returned.
func (b *bucket) Get(key string) (data []byte, err error) {
	defer err2.Return(&err)

	glog.V(7).Infoln("bucket::Get", key)

	data, err = b.owner.getData(b.bucketID, []byte(key))
	err2.Check(err)

	if len(data) == 0 {
		return nil, storage.ErrDataNotFound
	}

	return data, err
}

// Delete deletes the key + value pair (and all tags) associated with key.
// If key is empty, then an error will be returned.
func (b *bucket) Delete(key string) error {
	glog.V(7).Infoln("bucket::Delete", key)

	return b.owner.deleteData(b.bucketID, key)
}

func (b *bucket) GetAll(transform db.Filter) ([][]byte, error) {
	glog.V(7).Infoln("bucket::GetAll")

	return b.owner.getAll(b.bucketID, transform)
}

// Close closes this store object, freeing resources. For persistent store implementations, this does not delete
// any data in the underlying databases.
// Close can be called repeatedly on the same store multiple times without causing an error.
func (b *bucket) Close() error {
	glog.V(7).Infoln("bucket::Close")
	// skip this for now as Storage instance is handling closing
	return nil
}

// AFGO-placeholder
func (b *bucket) GetTags(key string) ([]storage.Tag, error) {
	glog.V(7).Infoln("bucket::GetTags", key)
	panic("implement me")
}

// AFGO-placeholder
func (b *bucket) GetBulk(keys ...string) ([][]byte, error) {
	glog.V(7).Infoln("bucket::GetBulk", keys)
	panic("implement me")
}

// AFGO-placeholder
func (b *bucket) Query(expression string, options ...storage.QueryOption) (storage.Iterator, error) {
	glog.V(7).Infoln("bucket::Query", expression)
	panic("implement me")
}

// AFGO-placeholder
func (b *bucket) Batch(operations []storage.Operation) error {
	glog.V(7).Infoln("bucket::Batch")
	panic("implement me")
}

// AFGO-placeholder
func (b *bucket) Flush() error {
	glog.V(7).Infoln("bucket::Flush")
	panic("implement me")
}
