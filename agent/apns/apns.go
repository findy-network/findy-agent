package apns

import (
	"crypto/tls"
	"errors"
	"os"

	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
)

var cert tls.Certificate
var client *apns2.Client
var enabled = false

// Init initializes APNS for push message sending.
func Init() (err error) {
	defer err2.Return(&err)

	certPath := utils.Settings.CertFileForAPNS()
	if certPath == "" {
		const oldCretPath = "FINDY_AGENT_CERT_PATH"
		glog.Warning("apns cert file name was empty.\n",
			"trying to use obsolete env:", oldCretPath)

		certPath = os.Getenv(oldCretPath)
	}
	if certPath == "" {
		const errorMsg = "no apns cert file in configuration, cannot send push notifications"
		glog.Error(errorMsg)
		return errors.New(errorMsg)
	}

	cert, err = certificate.FromP12File(certPath, "")
	err2.Check(err)
	client = apns2.NewClient(cert).Production()
	enabled = true
	return nil
}

// Push sends push notifications to all registered devices for the DID. The Init
// function must be called before Push calls.
func Push(did string) {
	if !enabled {
		return
	}

	defer err2.CatchTrace(func(err error) {
		glog.Error("Error in notifying devices: ", err)
	})

	ids, err := psm.GetAllDeviceIDRep(did)
	err2.Check(err)

	for _, id := range *ids {
		// dont terminate the loop ..
		if err := notifyDevice(id.DeviceToken); err != nil {
			glog.Error(err) // .. just log the error
		}
	}
}

func notifyDevice(token string) (err error) {
	defer err2.Return(&err)

	notification := newNotif(token)
	res, err := client.Push(notification)
	err2.Check(err)

	if res.StatusCode != 200 {
		glog.Errorf("Tried to notify device: %v %v %v\n",
			res.StatusCode, res.ApnsID, res.Reason)
	}
	return nil
}

func newNotif(token string) *apns2.Notification {
	return &apns2.Notification{
		DeviceToken: token,
		Topic:       "fi.findy.demo.agent.edge",
		Payload:     []byte(`{"aps":{"content-available" : 1}}`),
		PushType:    apns2.PushTypeBackground,
	}
}
