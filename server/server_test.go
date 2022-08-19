package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/stretchr/testify/assert"
)

func TestNotReadyHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	checkReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Not ready", string(data))
	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}

func TestReadyHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	agency.Ready.RegisteringComplete()
	checkReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "OK ready", string(data))
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
