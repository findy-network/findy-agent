package server

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"time"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/agency"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/agent/trans"
	"github.com/optechlab/findy-agent/agent/utils"
	"github.com/optechlab/findy-go/did"
	"github.com/optechlab/findy-go/pool"
	"golang.org/x/net/websocket"
)

const testServiceName = agency.CAAPIPath
const testServiceName2 = agency.ProtocolPath

var mux *http.ServeMux

func StartTestHTTPServer() {
	mux = http.NewServeMux()
	// We have mostly non-browser ws clients which don't send origin some remove default Handshake func
	wsServer := websocket.Server{Handler: trans.WsListen, Handshake: nil}
	wsPattern := fmt.Sprintf("/%sws/", testServiceName)
	mux.Handle(wsPattern, wsServer)

	mux.HandleFunc("/api/", handleAgencyAPI)
	pattern := fmt.Sprintf("/%s/", testServiceName)
	mux.HandleFunc(pattern, caAPITransport)
	pattern = fmt.Sprintf("/%s/", testServiceName2)
	mux.HandleFunc(pattern, protocolTransport)

	fs := http.FileServer(http.Dir(utils.Settings.ExportPath()))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	comm.SendAndWaitReq = testSendAndWaitHTTPRequest
	comm.FileDownload = testDownloadFile
}

func StartTestHTTPServer2() *httptest.Server {
	mux = http.NewServeMux()
	// We have mostly non-browser ws clients which don't send origin some remove default Handshake func
	wsServer := websocket.Server{Handler: trans.WsListen, Handshake: nil}
	wsPattern := fmt.Sprintf("/%sws/", testServiceName)
	mux.Handle(wsPattern, wsServer)

	mux.HandleFunc("/api/", handleAgencyAPI)
	pattern := fmt.Sprintf("/%s/", testServiceName)
	mux.HandleFunc(pattern, caAPITransport)
	pattern = fmt.Sprintf("/%s/", testServiceName2)
	mux.HandleFunc(pattern, protocolTransport)

	fs := http.FileServer(http.Dir(utils.Settings.ExportPath()))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	srv := httptest.NewServer(mux)

	utils.Settings.SetHostAddr(srv.URL)
	return srv

	//comm.SendAndWaitReq = testSendAndWaitHTTPRequest
	//comm.FileDownload = testDownloadFile
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
	out := err2.File.Try(os.Create(filename))
	defer func() {
		_ = resp.Body.Close()
		_ = out.Close()
	}()

	// Stream copy, can be used for large files as well
	err2.Empty.Try(io.Copy(out, resp.Body))

	return filename, nil
}

func ResetEnv(w *ssi.Wallet, exportPath string) {
	// Remove files
	err := os.RemoveAll(os.Getenv("HOME") + "/.indy_client")
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

	// Create pool
	const maxTimeout = 5 * time.Second
	const poolName = "myNewPool"

	currentPath, _ := os.Getwd()
	genesisPath := currentPath + "/../.circleci/genesis_transactions"
	config := pool.Config{
		GenesisTxn: genesisPath,
	}
	fmt.Println("Genesis path " + genesisPath)
	select {
	case r := <-pool.CreateConfig(poolName, config):
		if r.Err() != nil {
			fmt.Println("Pool create error, already exists?")
		}
	case <-time.After(maxTimeout):
		panic(errors.New("timeout exceeded"))
	}

	// Create wallet
	w.Create()
	handle := w.Open().Int()
	did.CreateAndStore(handle, did.Did{Seed: "000000000000000000000000Steward1"})
	w.Close(handle)
}
