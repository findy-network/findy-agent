package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

const TestServiceName = agency.ProtocolPath

var mux *http.ServeMux

func StartTestHTTPServer() {
	mux = http.NewServeMux()
	pattern := fmt.Sprintf("/%s/", TestServiceName)
	mux.HandleFunc(pattern, protocolTransport)

	comm.SendAndWaitReq = testSendAndWaitHTTPRequest
}

func StartTestHTTPServer2() *httptest.Server {
	mux = http.NewServeMux()
	pattern := fmt.Sprintf("/%s/", TestServiceName)
	mux.HandleFunc(pattern, protocolTransport)

	srv := httptest.NewServer(mux)

	utils.Settings.SetHostAddr(srv.URL)
	return srv
}

func testSendAndWaitHTTPRequest(urlStr string, msg io.Reader, _ time.Duration) (data []byte, err error) {
	ea := endp.NewClientAddr(urlStr)
	request, _ := http.NewRequestWithContext(context.TODO(), "POST", ea.TestAddress(), msg)
	writer := httptest.NewRecorder()
	mux.ServeHTTP(writer, request)

	response := writer.Result()
	response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %v", writer.Code)
	}

	data, err = io.ReadAll(response.Body)
	return data, err
}

func ResetEnv(w *ssi.Wallet, exportPath string) {
	defer err2.Catch(err2.Err(func(err error) {
		fmt.Println(err)
	}))
	// Remove files
	try.To(os.RemoveAll(utils.IndyBaseDir() + "/.indy_client"))
	try.To(os.RemoveAll(exportPath))
	registry := []byte("{}")
	try.To(os.WriteFile("./findy.json", registry, 0644))
	// Create wallet
	w.Create()
	handle := w.Open().Int()
	did.CreateAndStore(handle, did.Did{Seed: "000000000000000000000000Steward1"})
	w.Close(handle)
}
