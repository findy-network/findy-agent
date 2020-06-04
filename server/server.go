/*
Package server encapsulates http server entry points. It's the package for
agency services.
*/
package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/agency"
	"github.com/optechlab/findy-agent/agent/aries"
	"github.com/optechlab/findy-agent/agent/cloud"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/optechlab/findy-agent/agent/trans"
	"github.com/optechlab/findy-agent/agent/utils"
	"github.com/optechlab/findy-go/dto"
	"golang.org/x/net/websocket"
)

// StartHTTPServer starts the http server. The function blocks when it success.
// It builds the host address and writes it to utils.Settings. It takes a CA API
// path (serviceName), and a host port, a server port as an argument. The server
// port is the port to listen, and the host port is the actual port on the
// Internet, the port the world sees, and is assigned to endpoints.
func StartHTTPServer(serviceName string, serverPort uint) error {
	sp := fmt.Sprintf(":%v", serverPort)
	mux := http.NewServeMux()
	// We have mostly non-browser ws clients which don't send origin some remove default Handshake func
	wsServer := websocket.Server{Handler: trans.WsListen, Handshake: nil}
	wsPattern := fmt.Sprintf("/%sws/", serviceName)
	mux.Handle(wsPattern, wsServer)

	pattern := setHandler(serviceName, mux, caAPITransport)
	setHandler(utils.Settings.ServiceName2(), mux, protocolTransport)
	setHandler(agency.APIPath, mux, handleAgencyAPI)

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if glog.V(5) {
			glog.Info("/version requested")
		}
		_, _ = w.Write([]byte(utils.Version))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if glog.V(5) {
			glog.Info("testing the server")
			glog.Info(r.URL.Path)
		}
		_, _ = w.Write([]byte(utils.Version))
	})

	fs := http.FileServer(http.Dir(utils.Settings.ExportPath()))
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	if glog.V(1) {
		glog.Info(utils.Settings.VersionInfo())
		glog.Infof("HTTP Server on port: %v with handle patterns: \"%s\", \"%s\"\n", serverPort, pattern, wsPattern)
	}
	server := http.Server{
		Addr:    sp,
		Handler: mux,
	}
	return server.ListenAndServe()
}

func BuildHostAddr(hostPort uint) {
	// update the real server host name for agents' use, Yeah I know not a perfect!
	if hostPort != 80 {
		hostAddr := fmt.Sprintf("http://%s:%v", utils.Settings.HostAddr(), hostPort)
		utils.Settings.SetHostAddr(hostAddr)
	} else {
		hostAddr := fmt.Sprintf("http://%s", utils.Settings.HostAddr())
		utils.Settings.SetHostAddr(hostAddr)
	}
}

func setHandler(serviceName string,
	mux *http.ServeMux,
	handler func(http.ResponseWriter, *http.Request)) (pattern string) {

	pattern = fmt.Sprintf("/%s/", serviceName)
	mux.HandleFunc(pattern, handler)
	return pattern
}

func handleAgencyAPI(w http.ResponseWriter, r *http.Request) {
	defer err2.Catch(func(err error) {
		glog.Error("error:", err)
		errorResponse(w)
	})

	ourAddress := logRequestInfo("Agency API", r)
	body := err2.Bytes.Try(ioutil.ReadAll(r.Body))

	receivedPayload := mesg.PayloadCreator.NewFromData(body)
	responsePayload := agency.APICall(ourAddress, receivedPayload)
	data := responsePayload.JSON()

	w.Header().Set("Content-Type", "application/x-binary")
	_, _ = w.Write(data)
}

func caAPITransport(w http.ResponseWriter, r *http.Request) {
	defer err2.Catch(func(err error) {
		glog.Error("error:", err)
		errorResponse(w)
	})

	ourAddress := logRequestInfo("C/SA API TRANSPORT", r)
	data := err2.Bytes.Try(ioutil.ReadAll(r.Body))

	if !agency.IsHandlerInThisAgency(ourAddress) {
		errorResponse(w)
		return
	}

	// Get Transport from Agency to 1. decrypt the input, 2. build packet to
	// process, 3. encrypt the response
	tr := agency.CurrentTr(ourAddress)

	// 1. decrypt payload
	inPL := mesg.NewPayload(tr.PayloadPipe().Decrypt(data))

	// Get handler CA and forward Payload to it
	ca := agency.RcvrCA(ourAddress)

	// 2. put payload to a Packet to be processed accordingly
	outPL := comm.CloudAgentAPI().Process(comm.Packet{
		Payload:  &mesg.PayloadImpl{Payload: inPL},
		Address:  ourAddress,
		Receiver: ca,
	})

	// 3. Encrypt output Payload with the transport
	data = tr.PayloadPipe().Encrypt(dto.ToJSONBytes(outPL))

	w.Header().Set("Content-Type", "application/x-binary")
	_, _ = w.Write(data)
}

func errorResponse(w http.ResponseWriter) {
	glog.V(2).Info("Returning 500")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte("500 - Error"))
}

func protocolTransport(w http.ResponseWriter, r *http.Request) {
	defer err2.Catch(func(err error) {
		glog.Error("error:", err)
		errorResponse(w)
	})

	ourAddress := logRequestInfo("Aries TRANSPORT", r)

	data := err2.Bytes.Try(ioutil.ReadAll(r.Body))

	if !agency.IsHandlerInThisAgency(ourAddress) || !saveIncoming(ourAddress, data) {
		errorResponse(w)
		return
	}

	go transportPL(ourAddress, data)

	w.Header().Set("Content-Type", "application/json")
}

func logRequestInfo(caption string, r *http.Request) *endp.Addr {
	ourAddress := endp.NewServerAddr(r.URL.Path)
	ourAddress.BasePath = utils.Settings.HostAddr()
	if glog.V(1) {
		caption = fmt.Sprintf("===== %s =====", caption)
		glog.Info(caption, r.Method)
		glog.Info(ourAddress.Address())
		glog.Info("=====")

	}
	return ourAddress
}

func saveIncoming(addr *endp.Addr, data []byte) (ok bool) {
	addr.ID = utils.ReserveNonce(utils.NewNonce())
	if err := psm.AddRawPL(addr, data); err != nil {
		utils.DisposeNonce(addr.ID)
		return false
	}
	return true
}

func rmIncoming(addr *endp.Addr) {
	if err := psm.RmRawPL(addr); err != nil {
		glog.Error("could not rm incoming: ", err)
		return
	}
	utils.DisposeNonce(addr.ID)
}

func transportPL(ourAddress *endp.Addr, data []byte) {
	defer err2.CatchAll(func(err error) {
		glog.Error("transport payload error:", err)
	}, func(exception interface{}) {
		if utils.Settings.LocalTestMode() {
			panic(exception)
		} else {
			glog.Error(exception)
			debug.PrintStack()
		}
	})

	// First find the security pipe for the correct crypto. Then unpack the
	// envelope. Finally build the packet and forward it for handling. Packet
	// includes all the needed data for processing.

	// Most cases security pipe comes from wEA's pairwise endpoints
	rcvrCA := agency.ReceiverCA(ourAddress).(*cloud.Agent)
	pipe := rcvrCA.WEA().SecPipe(ourAddress.RcvrDID)

	// In case of connection-invite, we use common EA/CA pipe
	if pipe.IsNull() {
		pipe = agency.CurrentTr(ourAddress).PayloadPipe()
	}

	d, vk, err := pipe.Unpack(data)
	if err != nil {
		// Send error result to the other end, IF we SOME how can
		// In most cases we cannot, so ..

		// for now
		glog.Error("cannot unpack the envelope", err)
		panic(err)
	}

	inPL := aries.PayloadCreator.NewFromData(d)
	ourAddress.VerKey = vk // set associated verkey to our endp

	// Get handler CA and forward unpacked and typed Payload to it
	ca := agency.RcvrCA(ourAddress).(*cloud.Agent)

	// Put payload to a Packet and let communication processor handle it
	packet := comm.Packet{
		Payload:  inPL,
		Address:  ourAddress,
		Receiver: ca.WEA(), // worker EA handles the packet
	}
	err2.Check(comm.Proc.Process(packet))

	// no error, we can cleanup the received payload
	rmIncoming(packet.Address)
}
