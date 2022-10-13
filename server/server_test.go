package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/lainio/err2/assert"
)

func TestNotReadyHandler(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	checkReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	assert.NoError(err)
	assert.Equal("Not ready", string(data))
	assert.Equal(http.StatusServiceUnavailable, res.StatusCode)
}

func TestReadyHandler(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	agency.Ready.RegisteringComplete()
	checkReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	assert.NoError(err)
	assert.Equal("OK ready", string(data))
	assert.Equal(http.StatusOK, res.StatusCode)
}
