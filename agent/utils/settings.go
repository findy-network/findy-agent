package utils

import (
	"path/filepath"
	"time"

	"github.com/findy-network/findy-agent/method"
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

	gRPCAdmin string

	serviceName string        // name of the this service which is used in URLs, etc.
	hostAddr    string        // Ip host name of the server's host seen from internet
	versionInfo string        // Version number etc. in free format as a string
	timeout     time.Duration // timeout setting for http requests and connections
	exportPath  string        // wallet export path

	localTestMode bool // tells if are running unit tests, will be obsolete

	didMethod method.Type // the DID method to use as a default
}

func (h *Hub) DIDMethod() method.Type {
	return h.didMethod
}

func (h *Hub) SetDIDMethod(m method.Type) {
	h.didMethod = m
}
func (h *Hub) GRPCAdmin() string {
	return h.gRPCAdmin
}

func (h *Hub) SetGRPCAdmin(gRPCAdmin string) {
	h.gRPCAdmin = gRPCAdmin
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

// SetServiceName sets the service name for a2a communication
func (h *Hub) SetServiceName(n string) {
	h.serviceName = n
}

// ServiceName returns service name for a2a communication
func (h *Hub) ServiceName() string {
	return h.serviceName
}

// SetVersionInfo sets current version info of this agency. The info is shown in
// the certain API calls like Ping.
func (h *Hub) SetVersionInfo(info string) {
	h.versionInfo = info
}

// SetHostAddr sets current host name of this service agency. The host name is
// used in the URLs and endpoints.
func (h *Hub) SetHostAddr(ipName string) {
	glog.V(4).Infoln("setting host addr:", ipName)
	h.hostAddr = ipName
}

// SetExportPath sets path for wallet exports.
func (h *Hub) SetExportPath(exportPath string) {
	h.exportPath = exportPath
}

func (h *Hub) HostAddr() string {
	return h.hostAddr
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

func (h *Hub) ExportPath() string {
	return h.exportPath
}

func (h *Hub) WalletExportPath(filename string) (exportPath, url string) {
	return filepath.Join(h.exportPath, filename),
		h.hostAddr + filepath.Join("/static", filename)
}
