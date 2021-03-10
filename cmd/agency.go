package cmd

import (
	"log"
	"os"
	"time"

	"github.com/findy-network/findy-agent/cmds/agency"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

// AgencyCmd represents the agency command
var AgencyCmd = &cobra.Command{
	Use:   "agency",
	Short: "Parent command for starting and pinging agency",
	Long: `
Parent command for starting and pinging agency
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

var agencyStartEnvs = map[string]string{
	"apns-p12-file":            "APNS_P12_FILE",
	"host-address":             "HOST_ADDRESS",
	"host-port":                "HOST_PORT",
	"server-port":              "SERVER_PORT",
	"service-name":             "SERVICE_NAME",
	"pool-name":                "POOL_NAME",
	"pool-protocol":            "POOL_PROTOCOL",
	"steward-seed":             "STEWARD_SEED",
	"psm-database-file":        "PSM_DATABASE_FILE",
	"reset-register":           "RESET_REGISTER",
	"register-file":            "REGISTER_FILE",
	"steward-wallet-name":      "STEWARD_WALLET_NAME",
	"steward-wallet-key":       "STEWARD_WALLET_KEY",
	"steward-did":              "STEWARD_DID",
	"protocol-path":            "PROTOCOL_PATH",
	"salt":                     "SALT",
	"admin-id":                 "ADMIN_ID",
	"grpc":                     "GRPC",
	"grpc-port":                "GRPC_PORT",
	"grpc-cert-path":           "GRPC_CERT_PATH",
	"grpc-jwt-secret":          "GRPC_JWT_SECRET",
	"enclave-path":             "ENCLAVE_PATH",
	"enclave-backup":           "ENCLAVE_BACKUP",
	"enclave-backup-time":      "ENCLAVE_BACKUP_TIME",
	"enclave-key":              "ENCLAVE_KEY",
	"host-scheme":              "HOST_SCHEME",
	"register-backup":          "REGISTER_BACKUP",
	"register-backup-interval": "REGISTER_BACKUP_INTERVAL",
	"wallet-backup":            "WALLET_BACKUP",
	"wallet-backup-time":       "WALLET_BACKUP_TIME",
}

// startAgencyCmd represents the agency start subcommand
var startAgencyCmd = &cobra.Command{
	Use:   "start",
	Short: "Command for starting agency",
	Long: `
Start command for findy agency server.

Example
	findy-agent-cli agency start \
		--pool-name findy \
		--steward-wallet-name sovrin_steward_wallet \
		--steward-wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--steward-did Th7MpTaRZVRYnPiabds81Y \
		--salt mySalt
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(agencyStartEnvs, "AGENCY")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)

		err2.Check(aCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(aCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var agencyPingEnvs = map[string]string{
	"base-address": "PING_BASE_ADDRESS",
}

// pingAgencyCmd represents the agency ping subcommand
var pingAgencyCmd = &cobra.Command{
	Use:   "ping",
	Short: "Command for pinging agency",
	Long: `
Pings agency.
If agency works fine, ping ok with server's host address is printed.

Example
	findy-agent-cli agency ping \
		--base-address http://localhost:8080
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(agencyPingEnvs, "AGENCY")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		err2.Check(paCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(paCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var (
	aCmd  = agency.DefaultValues
	paCmd = agency.PingCmd{}
)

const registerBackupInterval = 12 * time.Hour

func init() {
	defer err2.CatchTrace(func(err error) {
		log.Println(err)
	})

	aCmd.VersionInfo = "findy-agent-cli v. 0.1"

	flags := startAgencyCmd.Flags()
	flags.StringVar(&aCmd.APNSP12CertFile, "apns-p12-file", "", flagInfo("APNS certificate p12 file", AgencyCmd.Name(), agencyStartEnvs["apns-p12-file"]))
	flags.StringVar(&aCmd.HostAddr, "host-address", "localhost", flagInfo("host address", AgencyCmd.Name(), agencyStartEnvs["host-address"]))
	flags.UintVar(&aCmd.HostPort, "host-port", 8080, flagInfo("host port", AgencyCmd.Name(), agencyStartEnvs["host-port"]))
	flags.UintVar(&aCmd.ServerPort, "server-port", 8080, flagInfo("server port", AgencyCmd.Name(), agencyStartEnvs["server-port"]))
	flags.StringVar(&aCmd.ServiceName, "service-name", "ca-api", flagInfo("service name", AgencyCmd.Name(), agencyStartEnvs["service-name"]))
	flags.StringVar(&aCmd.PoolName, "pool-name", "findy-pool", flagInfo("pool name", AgencyCmd.Name(), agencyStartEnvs["pool-name"]))
	flags.Uint64Var(&aCmd.PoolProtocol, "pool-protocol", 2, flagInfo("pool protocol", AgencyCmd.Name(), agencyStartEnvs["pool-protocol"]))
	flags.StringVar(&aCmd.StewardSeed, "steward-seed", "000000000000000000000000Steward1", flagInfo("steward seed", AgencyCmd.Name(), agencyStartEnvs["steward-seed"]))
	flags.StringVar(&aCmd.PsmDb, "psm-database-file", "findy.bolt", flagInfo("state machine database's filename", AgencyCmd.Name(), agencyStartEnvs["psm-database-file"]))
	flags.BoolVar(&aCmd.ResetData, "reset-register", false, flagInfo("reset handshake register", AgencyCmd.Name(), agencyStartEnvs["reset-register"]))
	flags.StringVar(&aCmd.HandshakeRegister, "register-file", "findy.json", flagInfo("handshake registry's filename", AgencyCmd.Name(), agencyStartEnvs["register-file"]))
	flags.StringVar(&aCmd.WalletName, "steward-wallet-name", "", flagInfo("steward wallet name", AgencyCmd.Name(), agencyStartEnvs["steward-wallet-name"]))
	flags.StringVar(&aCmd.WalletPwd, "steward-wallet-key", "", flagInfo("steward wallet key", AgencyCmd.Name(), agencyStartEnvs["steward-wallet-key"]))
	flags.StringVar(&aCmd.StewardDid, "steward-did", "", flagInfo("steward DID", AgencyCmd.Name(), agencyStartEnvs["steward-did"]))
	flags.StringVar(&aCmd.ServiceName2, "protocol-path", "a2a", flagInfo("URL path for A2A protocols", AgencyCmd.Name(), agencyStartEnvs["protocol-path"])) // agency.ProtocolPath is available
	flags.StringVar(&aCmd.Salt, "salt", "", flagInfo("salt", AgencyCmd.Name(), agencyStartEnvs["salt"]))
	flags.BoolVar(&aCmd.AllowRPC, "grpc", false, flagInfo("enable grpc", AgencyCmd.Name(), agencyStartEnvs["grpc"]))
	flags.IntVar(&aCmd.GRPCPort, "grpc-port", 50051, flagInfo("grpc server port", AgencyCmd.Name(), agencyStartEnvs["grpc-port"]))
	flags.StringVar(&aCmd.TlsCertPath, "grpc-cert-path", "", flagInfo("folder path for grpc server tls certificates", AgencyCmd.Name(), agencyStartEnvs["grpc-cert-path"]))
	flags.StringVar(&aCmd.JWTSecret, "grpc-jwt-secret", "", flagInfo("secure string for JWT token generation", AgencyCmd.Name(), agencyStartEnvs["grpc-jwt-secret"]))

	flags.StringVar(&aCmd.GRPCAdmin, "admin-id", aCmd.GRPCAdmin, flagInfo("agency's admin ID", AgencyCmd.Name(), agencyStartEnvs["admin-id"]))
	flags.StringVar(&aCmd.HostScheme, "host-scheme", aCmd.HostScheme, flagInfo("scheme of the agency's host address", AgencyCmd.Name(), agencyStartEnvs["host-scheme"]))
	flags.StringVar(&aCmd.EnclaveKey, "enclave-key", "", flagInfo("SHA-256 32 bytes in hex ascii", AgencyCmd.Name(), agencyStartEnvs["enclave-key"]))
	flags.StringVar(&aCmd.EnclavePath, "enclave-path", "", flagInfo("Enclave full file name", AgencyCmd.Name(), agencyStartEnvs["enclave-path"]))
	flags.StringVar(&aCmd.EnclaveBackupName, "enclave-backup", "", flagInfo("Base name for enclave backup file", AgencyCmd.Name(), agencyStartEnvs["enclave-backup"]))
	flags.StringVar(&aCmd.EnclaveBackupTime, "enclave-backup-time", "03:00", flagInfo("Time to start enclave backup in HH:MM[:SS]", AgencyCmd.Name(), agencyStartEnvs["enclave-backup-time"]))
	flags.StringVar(&aCmd.RegisterBackupName, "register-backup", "findy.json.bak", flagInfo("handshake registry backup file", AgencyCmd.Name(), agencyStartEnvs["register-backup"]))
	flags.DurationVar(&aCmd.RegisterBackupInterval, "register-backup-interval", registerBackupInterval, flagInfo("Duration between handshake registry backups", AgencyCmd.Name(), agencyStartEnvs["register-backup-interval"]))
	flags.StringVar(&aCmd.WalletBackupPath, "wallet-backup", "", flagInfo("Path for wallet backups", AgencyCmd.Name(), agencyStartEnvs["wallet-backup"]))
	flags.StringVar(&aCmd.WalletBackupTime, "wallet-backup-time", "04:00", flagInfo("Time to start wallet backups for dirty ones", AgencyCmd.Name(), agencyStartEnvs["wallet-backup-time"]))

	p := pingAgencyCmd.Flags()
	p.StringVar(&paCmd.BaseAddr, "base-address", "http://localhost:8080", flagInfo("base address of agency", AgencyCmd.Name(), agencyPingEnvs["base-address"]))

	rootCmd.AddCommand(AgencyCmd)
	AgencyCmd.AddCommand(startAgencyCmd)
	AgencyCmd.AddCommand(pingAgencyCmd)
}
