package wrapper

import (
	"flag"
	"os"
	"sync"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

var (
	testKey    = "key1"
	testValue  = []byte("value1")
	testConfig = Config{
		Key:       "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c",
		FileName:  "wrapper_test",
		FilePath:  ".",
		BucketIDs: []string{"id1", "id2"},
	}
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	try.To(flag.Set("logtostderr", "true"))
	try.To(flag.Set("stderrthreshold", "WARNING"))
	try.To(flag.Set("v", "10"))
	flag.Parse()
}

func tearDown() {
	os.RemoveAll(testConfig.FilePath + "/" + testConfig.FileName + ".bolt")
}

func TestOpen(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	s := New(testConfig)
	err := s.Init()
	assert.NoError(err)
	assert.INotNil(s)

	err = s.Close()
	assert.NoError(err)
}

func TestOpenStore(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	s := New(testConfig)
	err := s.Init()
	assert.NoError(err)
	assert.INotNil(s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	assert.NoError(err)
	assert.INotNil(store1)

	store2, err := s.OpenStore(testConfig.BucketIDs[1])
	assert.NoError(err)
	assert.INotNil(store2)

	store3, err := s.OpenStore("notExist")
	assert.Error(err)
	assert.INil(store3)

	err = s.Close()
	assert.NoError(err)
}

func TestReadData(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	s := New(testConfig)
	err := s.Init()
	assert.NoError(err)
	assert.INotNil(s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	assert.NoError(err)
	assert.INotNil(store1)

	var (
		key   = "key1"
		value = []byte("value1")
	)
	err = store1.Put(key, value)
	assert.NoError(err)

	got, err := store1.Get(key)
	assert.NoError(err)
	assert.DeepEqual(value, got)

	err = s.Close()
	assert.NoError(err)
}

func TestDeleteData(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	s := New(testConfig)
	err := s.Init()
	assert.NoError(err)
	assert.INotNil(s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	assert.NoError(err)
	assert.INotNil(store1)

	err = store1.Put(testKey, testValue)
	assert.NoError(err)

	got, err := store1.Get(testKey)
	assert.NoError(err)
	assert.DeepEqual(testValue, got)

	err = store1.Delete(testKey)
	assert.NoError(err)

	got, err = store1.Get(testKey)
	assert.Error(err)
	assert.SNil(got)

	err = s.Close()
	assert.NoError(err)
}

func TestConcurrentDataAccess(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	s := New(testConfig)
	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Init()
			assert.NoError(err)
			assert.INotNil(s)

			store1, err := s.OpenStore(testConfig.BucketIDs[0])
			assert.NoError(err)
			assert.INotNil(store1)

			err = store1.Put(testKey, testValue)
			assert.NoError(err)

			got, err := store1.Get(testKey)
			assert.NoError(err)
			assert.DeepEqual(testValue, got)
		}()
	}
	wg.Wait()
	err := s.Close()
	assert.NoError(err)
}
