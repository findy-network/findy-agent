package trans

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"

	"golang.org/x/net/websocket"
)

/*
wsListen is for server side WS connection accept. WS clients just listen. So
server is passive here as well.
*/
func WsListen(ws *websocket.Conn) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("ws listen error:", err)
	})

	r := ws.Request()

	cnxAddr := endp.NewServerAddr(r.URL.Path)
	cnxAddr.BasePath = r.URL.Host
	glog.V(2).Info("incoming WebSocket connection to: ", cnxAddr)

	if !agency.IsHandlerInThisAgency(cnxAddr.PlRcvr) || !cnxAddr.IsEncrypted() {
		glog.Warning("Accepting only safe connections")
		return
	}

	a, ok := agency.Handler(cnxAddr.ReceiverDID()).(comm.Receiver)
	if ok && a != nil {

		// Please notice that ws connection to CA is only made thru our
		// receiving Msg DID. We can have many ws connections to CA for many
		// different EA2EA pairwise.
		a.AddWs(a.WDID(), ws)

		// empty listen loop
		for {
			var data []byte
			if err := websocket.Message.Receive(ws, &data); err != nil {
				glog.V(3).Info("websocket is closed: ", err)
				return
			}
		}

	} else {
		err2.Check(ws.WriteClose(400))
	}
}

func WsConnect(cnxAddr *endp.Addr) (ws *websocket.Conn, err error) {
	// Do we need this or is there anything to avoid this? Our server
	// doesn't check it, but it seems that current Go API won't offer easy
	// way not to fill origin. We will monitor this and leave it to be.
	// Note! that this must be correct URL though.
	origin := "http://localhost/"

	url := cnxAddr.Address()
	return websocket.Dial(url, "", origin)
}
