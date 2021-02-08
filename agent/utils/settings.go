package utils

import (
	"path/filepath"
	"time"

	"github.com/golang/glog"
)

const HTTPReqTimeout = 1 * time.Minute

var Settings = &Hub{}

type Hub struct {
	registerName           string        // name of the persistent register where agents and their wallets are stored
	registerBackupName     string        // cloud agent register's (above, json) backup file
	registerBackupInterval time.Duration // hours between backups

	walletBackupPath string
	walletBackupTime string

	serviceName   string        // name of the this service which is used in URLs, etc.
	serviceName2  string        // name of the this service which is used in URLs, etc.
	hostAddr      string        // Ip host name of the server's host seen from internet
	versionInfo   string        // Version number etc. in free format as a string
	wsServiceName string        // web socket service name, mostly for the ws CLI clients to use
	timeout       time.Duration // timeout setting for http requests and connections
	exportPath    string        // wallet export path

	localTestMode bool // tells if are running unit tests, will be obsolete

	certFileForAPNS string // APNS certification file in P12
}

func (h *Hub) WalletBackupTime() string {
	return h.walletBackupTime
}

func (h *Hub) SetWalletBackupTime(t string) {
	h.walletBackupTime = t
}

func (h *Hub) WalletBackupPath() string {
	return h.walletBackupPath
}

func (h *Hub) SetWalletBackupPath(path string) {
	h.walletBackupPath = path
}

func (h *Hub) RegisterBackupName() string {
	return h.registerBackupName
}

func (h *Hub) SetRegisterBackupName(name string) {
	h.registerBackupName = name
}

func (h *Hub) RegisterBackupInterval() time.Duration {
	return h.registerBackupInterval
}

func (h *Hub) SetRegisterBackupInterval(interval time.Duration) {
	h.registerBackupInterval = interval
}

func (h *Hub) CertFileForAPNS() string {
	return h.certFileForAPNS
}

func (h *Hub) SetCertFileForAPNS(certFileForAPNS string) {
	h.certFileForAPNS = certFileForAPNS
}

func (h *Hub) RegisterName() string {
	return h.registerName
}

func (h *Hub) SetRegisterName(registerName string) {
	h.registerName = registerName
}

func (h *Hub) LocalTestMode() bool {
	return h.localTestMode
}

func (h *Hub) SetLocalTestMode(localTestMode bool) {
	h.localTestMode = localTestMode
}

// SetTimeout sets the default timeout for HTTP and WS requests.
func (h *Hub) SetTimeout(to time.Duration) {
	h.timeout = to
}

// SetServiceName sets the service name of this agency. Service name is used in the
// URLs and endpoint addresses.
func (h *Hub) SetServiceName(n string) {
	h.serviceName = n
}

func (h *Hub) SetServiceName2(n string) {
	h.serviceName2 = n
}

// ServiceName2 returns service name to new worker EA based endpoints. This is
// something that will be refactored later.
func (h *Hub) ServiceName2() string {
	return h.serviceName2
}

// SetWsName sets web socket service name. It's in the different URL than HTTP.
func (h *Hub) SetWsName(n string) {
	h.wsServiceName = n
}

// SetVersionInfo sets current version info of this agency. The info is shown in
// the certain API calls like Ping.
func (h *Hub) SetVersionInfo(info string) {
	h.versionInfo = info
}

// SetHostAddr sets current host name of this service agency. The host name is
// used in the URLs and endpoints.
func (h *Hub) SetHostAddr(ipName string) {
	h.hostAddr = ipName
}

// SetExportPath sets path for wallet exports.
func (h *Hub) SetExportPath(exportPath string) {
	h.exportPath = exportPath
}

func (h *Hub) HostAddr() string {
	return h.hostAddr
}

func (h *Hub) ServiceName() string {
	if h.serviceName == "" && glog.V(3) {
		glog.Info("warning service name is empty")
	}
	return h.serviceName
}

func (h *Hub) VersionInfo() string {
	return h.versionInfo
}

func (h *Hub) Timeout() time.Duration {
	if h.timeout == 0 {
		return HTTPReqTimeout
	}
	return h.timeout
}

func (h *Hub) WsServiceName() string {
	return h.wsServiceName
}

func (h *Hub) ExportPath() string {
	return h.exportPath
}

func (h *Hub) WalletExportPath(filename string) (exportPath, url string) {
	return filepath.Join(h.exportPath, filename),
		h.hostAddr + filepath.Join("/static", filename)
}

// WebOnboardWalletName returns wallet name for web boarding wallet
func (h *Hub) WebOnboardWalletName() string {
	return "findy_web_wallet"
}

// WebOnboardWalletKey returns wallet key for web boarding wallet
func (h *Hub) WebOnboardWalletKey() string {
	// todo: we should get this from secrets. However, the whole wallet
	//  isn't important because it has only EA/CA pairwise we could remove
	//  it every time or ...
	return "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
}
