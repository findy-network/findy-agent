package prot

import (
	"crypto/tls"
	"os"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
)

func notifyDevice(cert *tls.Certificate, token string) error {
	notification := &apns2.Notification{}
	notification.DeviceToken = token
	notification.Topic = "fi.findy.demo.agent.edge"
	notification.Payload = []byte(`{"aps":{"content-available" : 1}}`)
	notification.PushType = apns2.PushTypeBackground

	client := apns2.NewClient(*cert).Production()
	//client := apns2.NewClient(*cert).Development()
	res, err := client.Push(notification)

	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		glog.Errorf("Tried to notify device: %v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
	}
	return nil
}

func notifyNewTasks(did string) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Error in notifying devices: ", err)
	})

	certPath := os.Getenv("FINDY_AGENT_CERT_PATH") // P12
	if len(certPath) > 0 {
		ids, err := psm.GetAllDeviceIDRep(did)
		err2.Check(err)

		cert, err := certificate.FromP12File(certPath, "")
		err2.Check(err)

		for _, id := range *ids {
			// dont terminate the loop ..
			if err := notifyDevice(&cert, id.DeviceToken); err != nil {
				glog.Error(err) // .. just log the error
			}
		}
	}
}
