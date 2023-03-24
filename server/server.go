/*
Package server encapsulates http server entry points. It's the package for
agency services.
*/
package server

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

// StartHTTPServer starts the http server. The function blocks when it success.
// It builds the host address and writes it to utils.Settings. It takes a CA API
// path (serviceName), and a host port, a server port as an argument. The server
// port is the port to listen, and the host port is the actual port on the
// Internet, the port the world sees, and is assigned to endpoints.
func StartHTTPServer(serverPort uint) error {
	sp := fmt.Sprintf(":%v", serverPort)
	mux := http.NewServeMux()

	pattern := setHandler(utils.Settings.ServiceName(), mux, protocolTransport)
	pattern2 := buildNewTransportPath(pattern)
	mux.HandleFunc(pattern2, protocolTransport)
	mux.HandleFunc("/version", tellVersion)
	mux.HandleFunc("/ready", checkReady)
	mux.HandleFunc("/", tellVersion)

	if glog.V(1) {
		glog.Info(utils.Settings.VersionInfo())
		glog.Infof("HTTP Server on port: %v with handle pattern: \"%s\"",
			serverPort, pattern)
		glog.Infof("***** New DID-Server v2.0 Path: '%s' *******", pattern2)
	}
	server := http.Server{
		Addr:    sp,
		Handler: mux,
	}
	return server.ListenAndServe()
}

func buildNewTransportPath(pattern string) string {
	return strings.TrimSuffix(pattern, "/") + endp.Version2EndpSuffix + "/"
}

func BuildHostAddr(scheme string, hostPort uint) {
	// update the real server host name for agents' use, Yeah I know not a perfect!
	if hostPort != 80 {
		hostAddr := fmt.Sprintf("%s://%s:%v", scheme, utils.Settings.HostAddr(), hostPort)
		utils.Settings.SetHostAddr(hostAddr)
	} else {
		hostAddr := fmt.Sprintf("%s://%s", scheme, utils.Settings.HostAddr())
		utils.Settings.SetHostAddr(hostAddr)
	}
}

func tellVersion(w http.ResponseWriter, _ *http.Request) {
	defer err2.Catch(func(err error) {
		glog.Warningln(err)
	})
	if glog.V(12) {
		glog.Info("/version requested")
	}
	try.To1(w.Write([]byte(utils.Version)))
}

func checkReady(w http.ResponseWriter, _ *http.Request) {
	defer err2.Catch(func(err error) {
		glog.Warningln(err)
	})
	if agency.Ready.IsReady() {
		w.WriteHeader(http.StatusOK)
		try.To1(w.Write([]byte("OK ready")))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	try.To1(w.Write([]byte("Not ready")))
}

func setHandler(serviceName string,
	mux *http.ServeMux,
	handler func(http.ResponseWriter, *http.Request)) (pattern string) {

	pattern = fmt.Sprintf("/%s/", serviceName)
	mux.HandleFunc(pattern, handler)
	return pattern
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

	ourAddress := logRequestInfo("Incoming Aries TRANSPORT", r)

	data := try.To1(io.ReadAll(r.Body))

	canContinue := ourAddress != nil &&
		agency.IsHandlerInThisAgency(ourAddress.PlRcvr) &&
		saveIncoming(ourAddress, data)

	if !canContinue {
		errorResponse(w)
		return
	}

	go transportPL(ourAddress, data)

	w.Header().Set("Content-Type", "application/json")
}

func logRequestInfo(caption string, r *http.Request) *endp.Addr {
	ourAddress := endp.NewServerAddr(r.URL.Path)
	if !ourAddress.Valid() {
		glog.V(3).Infoln("------ address isn't valid:", r.URL.Path)
		return nil
	}
	ourAddress.BasePath = utils.Settings.HostAddr()
	if glog.V(1) {
		caption = fmt.Sprintf("===== %s (%s) =====", caption, r.Method)
		glog.Info(caption)
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
	defer err2.Catch(func(err error) {
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
	rcvrWA := rcvrCA.WEA()
	pipe := rcvrWA.SecPipe(ourAddress.ConnID)

	// In case of connection-invite, we use common EA/CA pipe
	if pipe.IsNull() {
		// TODO: this should never happen
		panic("invitations aren't transported thru these anymore")
		//pipe = agency.CurrentTr(ourAddress).PayloadPipe()
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

	try.To(comm.Proc.Process(packet))

	// no error, we can cleanup the received payload
	rmIncoming(packet.Address)
}
