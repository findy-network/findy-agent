package pltype

import (
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
)

// name constants
const (
	HandshakePairwiseName = "HANDSHAKE"
)

// Protocol constants
const (
	Terminate = ""
	Nothing   = ""
	Agent     = "urn:indy:sov:agent:message_type:findy.fi" // This will be for old and EA/CA PW
	Aries     = "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec"      // This will be for all Aries protocols
	CA        = "urn:indy:sov:agent_api:message_type:findy.fi"
	SA        = "urn:indy:sov:service_agent_api:message_type:findy.fi"

	LibindyRequestPresentationID = "libindy-request-presentation-0"
	LibindyPresentationID        = "libindy-presentation-0"

	UserAction = "user-action"

	ConnectionTrustAgent = "CONNECTION_TRUST_AGENT" // internal use only

	ProtocolConnection = "connection"
	Connection         = Agent + "/" + ProtocolConnection

	// these were for the Indy agent protocol, new Aries constants are in the
	// protocol files.
	ConnectionResponse  = Connection + "/1.0/response"
	ConnectionRequest   = Connection + "/1.0/request"
	ConnectionOffer     = Connection + "/1.0/offer"
	ConnectionHandshake = Connection + "/1.0/invite"
	ConnectionOk        = Connection + "/1.0/ok"    // terminates Acknowledgement cycle, internal use
	ConnectionError     = Connection + "/1.0/error" // if error occurs we send error payload, especially handy with ws
	ConnectionPing      = Connection + "/1.0/ping"
	ConnectionMsg       = Connection + "/1.0/msg"
	ConnectionAck       = Connection + "/1.0/acknowledgement"
)

const (
	ProtocolNotification      = "notification"
	HandlerProblemReport      = "problem-report"
	HandlerAck                = "ack"
	ProblemReport             = Aries + "/" + ProtocolNotification
	NotificationProblemReport = ProblemReport + "/1.0/" + HandlerProblemReport
	NotificationAck           = ProblemReport + "/1.0/" + HandlerAck
)

// Issue Credential protocol constants
const (
	ProtocolIssueCredential          = "issue-credential"
	HandlerIssueCredentialPropose    = "propose-credential"
	HandlerIssueCredentialOffer      = "offer-credential"
	HandlerIssueCredentialRequest    = "request-credential"
	HandlerIssueCredentialIssue      = "issue-credential"
	HandlerIssueCredentialACK        = "ack"
	HandlerIssueCredentialNACK       = "nack"
	ObjectTypeCredentialPreview      = "credential-preview"
	HandlerIssueCredentialUserAction = UserAction
	IssueCredential                  = Aries + "/" + ProtocolIssueCredential
	IssueCredentialPropose           = IssueCredential + "/1.0/" + HandlerIssueCredentialPropose
	IssueCredentialOffer             = IssueCredential + "/1.0/" + HandlerIssueCredentialOffer
	IssueCredentialUserAction        = IssueCredential + "/1.0/" + HandlerIssueCredentialUserAction
	IssueCredentialRequest           = IssueCredential + "/1.0/" + HandlerIssueCredentialRequest
	IssueCredentialIssue             = IssueCredential + "/1.0/" + HandlerIssueCredentialIssue
	IssueCredentialACK               = IssueCredential + "/1.0/" + HandlerIssueCredentialACK
	IssueCredentialNACK              = IssueCredential + "/1.0/" + HandlerIssueCredentialNACK
	IssueCredentialCredentialPreview = IssueCredential + "/1.0/" + ObjectTypeCredentialPreview
)

// DID exchange aka Connection related constants
const (
	Invitation                = "invitation"
	HandlerOffer              = "offer"
	HandlerRequest            = "request"
	HandlerResponse           = "response"
	AriesProtocolConnection   = "connections"
	AriesConnection           = Aries + "/" + AriesProtocolConnection
	AriesConnectionInvitation = AriesConnection + "/1.0/" + Invitation
	AriesConnectionRequest    = AriesConnection + "/1.0/" + HandlerRequest
	AriesConnectionOffer      = AriesConnection + "/1.0/" + HandlerOffer
	AriesConnectionResponse   = AriesConnection + "/1.0/" + HandlerResponse
)

// Present Proof protocol constants
const (
	ProtocolPresentProof            = "present-proof"
	HandlerPresentProofPropose      = "propose-presentation"
	HandlerPresentProofRequest      = "request-presentation"
	HandlerPresentProofPresentation = "presentation"
	HandlerPresentProofACK          = "ack"
	HandlerPresentProofNACK         = "nack"
	HandlerPresentUserAction        = UserAction
	ObjectTypePresentationPreview   = "presentation-preview"
	PresentProof                    = Aries + "/" + ProtocolPresentProof
	PresentProofPropose             = PresentProof + "/1.0/" + HandlerPresentProofPropose
	PresentProofRequest             = PresentProof + "/1.0/" + HandlerPresentProofRequest
	PresentProofPresentation        = PresentProof + "/1.0/" + HandlerPresentProofPresentation
	PresentProofUserAction          = PresentProof + "/1.0/" + HandlerPresentUserAction
	PresentProofACK                 = PresentProof + "/1.0/" + HandlerPresentProofACK
	PresentProofNACK                = PresentProof + "/1.0/" + HandlerPresentProofNACK
	PresentationPreviewObj          = PresentProof + "/1.0/" + ObjectTypePresentationPreview
)

// Basic Message protocol constants
const (
	ProtocolBasicMessage = "basicmessage"
	HandlerMessage       = "message"
	BasicMessage         = Aries + "/" + ProtocolBasicMessage
	BasicMessageSend     = BasicMessage + "/1.0/" + HandlerMessage
)

// Trust Ping protocol constants
const (
	ProtocolTrustPing   = "trust_ping"
	HandlerPing         = "ping"
	HandlerPingResponse = "ping_response"
	TrustPing           = Aries + "/" + ProtocolTrustPing
	TrustPingPing       = TrustPing + "/1.0/" + HandlerPing
	TrustPingResponse   = TrustPing + "/1.0/" + HandlerPingResponse
)

// SA API msg types
const (
	SAPing                         = SA + "/ping/1.0/ping"
	SAIssueCredential              = SA + "/issue_credential"
	SAIssueCredentialAcceptPropose = SAIssueCredential + "/1.0/accept_propose"
	SAPresentProof                 = SA + "/present_proof"
	SAPresentProofAcceptPropose    = SAPresentProof + "/1.0/accept_propose"
	SAPresentProofAcceptValues     = SAPresentProof + "/1.0/accept_values"
)

// CA API msg types
const (
	CASchema       = CA + "/schema"
	CASchemaCreate = CASchema + "/1.0/create"

	CACredDef       = CA + "/credential_definition"
	CACredDefCreate = CACredDef + "/1.0/create"

	CALedger           = CA + "/ledger"
	CALedgerWriteDid   = CALedger + "/1.0/write_did"
	CALedgerGetCredDef = CALedger + "/1.0/get_cred_def"
	CALedgerGetSchema  = CALedger + "/1.0/get_schema"

	CADID       = CA + "/did"
	CADIDVerKey = CADID + "/1.0/verkey"

	CAWallet    = CA + "/wallet"
	CAWalletGet = CAWallet + "/1.0/get"

	// Protocol launchers - protocol string must match Aries protocol
	CAPairwise           = CA + "/" + AriesProtocolConnection
	CAPairwiseInvitation = CAPairwise + "/1.0/invitation"
	CAPairwiseCreate     = CAPairwise + "/1.0/create"

	// Protocol launcher - protocol string must match Aries protocol
	CATrustPing = CA + "/" + ProtocolTrustPing + "/1.0/ping"

	// Protocol launcher - protocol string must match Aries protocol
	CAGetJWT = CA + "/" + "login" + "/1.0/jwt"

	CATask       = CA + "/task"
	CATaskStatus = CATask + "/1.0/status"
	CATaskReady  = CATask + "/1.0/ready"
	CATaskList   = CATask + "/1.0/list"

	CANotify           = CA + "/notify"
	CANotifyStatus     = CANotify + "/1.0/status"
	CANotifyUserAction = CANotify + "/1.0/user-action"

	// Protocol launchers - protocol string must match Aries protocol
	CACred        = CA + "/" + ProtocolIssueCredential
	CACredRequest = CACred + "/1.0/request" // TODO
	CACredOffer   = CACred + "/1.0/propose"

	// Protocol launchers - protocol string must match Aries protocol
	CAProof        = CA + "/" + ProtocolPresentProof
	CAProofPropose = CAProof + "/1.0/propose" // TODO
	CAProofRequest = CAProof + "/1.0/request"

	// Protocol launcher - protocol string must match Aries protocol
	CABasicMessage = CA + "/" + ProtocolBasicMessage + "/1.0/send"

	CAProblemReport = CA + "/notification/1.0/problem_report"

	CAPingOwnCA = CA + "/ping/1.0/own_ca"

	CAPingAPIEndp     = CA + "/ping/1.0/api_endp"
	CAAttachAPIEndp   = CA + "/attach/1.0/api_endp"
	CAAttachEADefImpl = CA + "/attach/1.0/ea_def_impl"

	CAContinuePresentProofProtocol    = CA + "/protocol/1.0/continue-present-proof"
	CAContinueIssueCredentialProtocol = CA + "/protocol/1.0/continue-issue-credential"
)

var protocolType = map[string]pb.Protocol_Type{
	AriesProtocolConnection: pb.Protocol_DIDEXCHANGE,
	ProtocolIssueCredential: pb.Protocol_ISSUE_CREDENTIAL,
	ProtocolPresentProof:    pb.Protocol_PRESENT_PROOF,
	ProtocolTrustPing:       pb.Protocol_TRUST_PING,
	ProtocolBasicMessage:    pb.Protocol_BASIC_MESSAGE,
}

func ProtocolTypeForFamily(family string) pb.Protocol_Type {
	if protocol, ok := protocolType[family]; ok {
		return protocol
	}
	glog.Warningf("no protocol type found for family %s", family)
	return pb.Protocol_NONE
}
