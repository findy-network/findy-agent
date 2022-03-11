package wrapper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/findy-network/findy-common-go/crypto"
	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/lainio/err2"
)

const level7 = 7

type Store interface {
	storage.Store
	GetAll(transform db.Filter) ([][]byte, error)
}

type Config struct {
	Key       string
	FileName  string
	FilePath  string
	BucketIDs []string
}

type StorageProvider struct {
	l sync.RWMutex

	conf    Config
	db      *db.Mgd
	buckets map[string]bucket
	cipher  *crypto.Cipher
}

func New(config Config) *StorageProvider {
	s := &StorageProvider{
		l:       sync.RWMutex{},
		conf:    config,
		db:      nil,
		buckets: make(map[string]bucket),
	}

	var bucketKey byte
	for _, name := range s.conf.BucketIDs {
		s.buckets[name] = newBucket(s, bucketKey)
		bucketKey++
	}

	return s
}

func (s *StorageProvider) Init() (err error) {
	defer err2.Annotate("afgo storage open", &err)

	s.l.Lock()
	defer s.l.Unlock()

	if s.db != nil {
		glog.Warningf("skipping storage provider initialization for %s, already open", s.conf.FileName)
		return nil
	}

	k, err := hex.DecodeString(s.conf.Key)
	err2.Check(err)

	cipher := crypto.NewCipher(k)

	path := "."
	if s.conf.FilePath != "" {
		path = s.conf.FilePath
	}

	filename := path + "/" + s.conf.FileName + ".bolt"

	if len(s.conf.BucketIDs) == 0 {
		return fmt.Errorf("no buckets specified")
	}

	mgdBuckets := make([][]byte, 0)

	var bucketKey byte
	for range s.conf.BucketIDs {
		mgdBuckets = append(mgdBuckets, []byte{bucketKey})
		bucketKey++
	}

	// this will not open the file handle to db, just initializes it
	s.db = db.New(db.Cfg{
		Filename:   filename,
		Buckets:    mgdBuckets,
		BackupName: filename + "_backup",
	})

	s.cipher = cipher

	return nil
}

func (s *StorageProvider) ID() string {
	return s.conf.FileName
}

func (s *StorageProvider) Key() string {
	return s.conf.Key
}

// Used by AFGO through StorageProvider interface
func (s *StorageProvider) OpenStore(name string) (storage.Store, error) {
	glog.V(level7).Infoln("StorageProvider::OpenStore", s.ID(), name)

	if b, ok := s.buckets[name]; ok {
		return &b, nil
	}
	return nil, fmt.Errorf("store %s not found", name)
}

func (s *StorageProvider) Close() (err error) {
	defer err2.Annotate("afgo storage close", &err)

	s.l.RLock()
	defer s.l.RUnlock()

	if s.db == nil {
		glog.Warningf("skipping storage provider close for %s, already closed", s.conf.FileName)
		return nil
	}

	err2.Check(s.db.Close())
	s.db = nil
	return
}

func (s *StorageProvider) addData(bucketID byte, key, value []byte) (err error) {
	s.l.RLock()
	defer s.l.RUnlock()

	err = s.db.AddKeyValueToBucket([]byte{bucketID},
		&db.Data{
			Data: value,
			Read: s.encrypt,
		},
		&db.Data{
			Data: key,
			Read: s.hash,
		},
	)
	return err
}

func (s *StorageProvider) hash(key []byte) (k []byte) {
	// TODO: Please use salt when implementing this.
	if s.cipher != nil {
		h := md5.Sum(key)
		return h[:]
	}
	return append(key[:0:0], key...)
}

func (s *StorageProvider) encrypt(value []byte) (k []byte) {
	if s.cipher != nil {
		return s.cipher.TryEncrypt(value)
	}
	return append(value[:0:0], value...)
}

func (s *StorageProvider) decrypt(value []byte) (k []byte) {
	if s.cipher != nil {
		return s.cipher.TryDecrypt(value)
	}
	return append(value[:0:0], value...)
}

func (s *StorageProvider) getData(
	bucketID byte,
	key []byte,
) (
	value []byte,
	err error,
) {
	s.l.RLock()
	defer s.l.RUnlock()

	data := &db.Data{
		Write: s.decrypt,
		Use: func(d []byte) interface{} {
			value = d
			return nil
		},
	}
	_, err = s.db.GetKeyValueFromBucket([]byte{bucketID},
		&db.Data{
			Data: key,
			Read: s.hash,
		},
		data)

	return value, err
}

func (s *StorageProvider) deleteData(bucketID byte, key string) (err error) {
	s.l.RLock()
	defer s.l.RUnlock()

	err = s.db.RmKeyValueFromBucket([]byte{bucketID}, &db.Data{
		Data: []byte(key),
		Read: s.hash,
	})
	return
}

func (s *StorageProvider) getAll(bucketID byte, transform db.Filter) (res [][]byte, err error) {
	s.l.RLock()
	defer s.l.RUnlock()

	return s.db.GetAllValuesFromBucket([]byte{bucketID}, s.decrypt, transform)
}

// AFGO StorageProvider placeholder implementations
func (s *StorageProvider) SetStoreConfig(name string, config storage.StoreConfiguration) error {
	glog.V(level7).Infoln("StorageProvider::SetStoreConfig", name)
	panic("implement me")
}

func (s *StorageProvider) GetStoreConfig(name string) (storage.StoreConfiguration, error) {
	glog.V(level7).Infoln("StorageProvider::GetStoreConfig", name)
	panic("implement me")
}

func (s *StorageProvider) GetOpenStores() []storage.Store {
	glog.V(level7).Infoln("StorageProvider::GetOpenStores")
	panic("implement me")
}
