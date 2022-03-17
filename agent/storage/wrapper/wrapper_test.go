package wrapper

import (
	"flag"
	"os"
	"sync"
	"testing"

	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/require"
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
	s := New(testConfig)
	err := s.Init()
	require.NoError(t, err)
	require.NotNil(t, s)

	err = s.Close()
	require.NoError(t, err)
}

func TestOpenStore(t *testing.T) {
	s := New(testConfig)
	err := s.Init()
	require.NoError(t, err)
	require.NotNil(t, s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	require.NoError(t, err)
	require.NotNil(t, store1)

	store2, err := s.OpenStore(testConfig.BucketIDs[1])
	require.NoError(t, err)
	require.NotNil(t, store2)

	store3, err := s.OpenStore("notExist")
	require.Error(t, err)
	require.Nil(t, store3)

	err = s.Close()
	require.NoError(t, err)
}

func TestReadData(t *testing.T) {
	s := New(testConfig)
	err := s.Init()
	require.NoError(t, err)
	require.NotNil(t, s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	require.NoError(t, err)
	require.NotNil(t, store1)

	var (
		key   = "key1"
		value = []byte("value1")
	)
	err = store1.Put(key, value)
	require.NoError(t, err)

	got, err := store1.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, got)

	err = s.Close()
	require.NoError(t, err)
}

func TestDeleteData(t *testing.T) {
	s := New(testConfig)
	err := s.Init()
	require.NoError(t, err)
	require.NotNil(t, s)

	store1, err := s.OpenStore(testConfig.BucketIDs[0])
	require.NoError(t, err)
	require.NotNil(t, store1)

	err = store1.Put(testKey, testValue)
	require.NoError(t, err)

	got, err := store1.Get(testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, got)

	err = store1.Delete(testKey)
	require.NoError(t, err)

	got, err = store1.Get(testKey)
	require.Error(t, err)
	require.Nil(t, got)

	err = s.Close()
	require.NoError(t, err)
}

func TestConcurrentDataAccess(t *testing.T) {
	s := New(testConfig)
	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Init()
			require.NoError(t, err)
			require.NotNil(t, s)

			store1, err := s.OpenStore(testConfig.BucketIDs[0])
			require.NoError(t, err)
			require.NotNil(t, store1)

			err = store1.Put(testKey, testValue)
			require.NoError(t, err)

			got, err := store1.Get(testKey)
			require.NoError(t, err)
			require.Equal(t, testValue, got)
		}()
	}
	wg.Wait()
	err := s.Close()
	require.NoError(t, err)
}
