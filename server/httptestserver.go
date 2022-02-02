package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
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

	fs := http.FileServer(http.Dir(utils.Settings.ExportPath()))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	comm.SendAndWaitReq = testSendAndWaitHTTPRequest
	comm.FileDownload = testDownloadFile
}

func StartTestHTTPServer2() *httptest.Server {
	mux = http.NewServeMux()
	pattern := fmt.Sprintf("/%s/", TestServiceName)
	mux.HandleFunc(pattern, protocolTransport)

	fs := http.FileServer(http.Dir(utils.Settings.ExportPath()))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	srv := httptest.NewServer(mux)

	utils.Settings.SetHostAddr(srv.URL)
	return srv
}

func testSendAndWaitHTTPRequest(urlStr string, msg io.Reader, _ time.Duration) (data []byte, err error) {
	ea := endp.NewClientAddr(urlStr)
	request, _ := http.NewRequest("POST", ea.TestAddress(), msg)
	writer := httptest.NewRecorder()
	mux.ServeHTTP(writer, request)

	response := writer.Result()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %v", writer.Code)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	data, err = ioutil.ReadAll(response.Body)
	return data, err
}

func testDownloadFile(downloadDir, filepath, url string) (name string, err error) {
	defer err2.Annotate("TDD download file", &err)

	request, _ := http.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	mux.ServeHTTP(writer, request)

	resp := writer.Result()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download file: %s", resp.Status)
	}

	filename := filepath
	if filename == "" {
		filename = path.Base(request.URL.String())
	}
	filename = path.Join(downloadDir, filename)
	out := try.To1(os.Create(filename))
	defer func() {
		_ = resp.Body.Close()
		_ = out.Close()
	}()

	// Stream copy, can be used for large files as well
	try.To1(io.Copy(out, resp.Body))

	return filename, nil
}

func ResetEnv(w *ssi.Wallet, exportPath string) {
	// Remove files
	err := os.RemoveAll(utils.IndyBaseDir() + "/.indy_client")
	if err != nil {
		fmt.Println(err.Error())
	}

	err = os.RemoveAll(exportPath)
	if err != nil {
		fmt.Println(err.Error())
	}

	registry := []byte("{}")
	err = ioutil.WriteFile("./findy.json", registry, 0644)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Create wallet
	w.Create()
	handle := w.Open().Int()
	did.CreateAndStore(handle, did.Did{Seed: "000000000000000000000000Steward1"})
	w.Close(handle)
}
