package issuecredential

import (
	"encoding/base64"
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/dto"
)

var IssueCreator = &IssueFactor{}

type IssueFactor struct{}

func (f *IssueFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Issue{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewIssue(m)
}

func (f *IssueFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewIssueMsg(data)
}

func init() {
	gob.Register(&IssueImpl{})
	aries.Creator.Add(pltype.IssueCredentialIssue, IssueCreator)
}

func NewIssue(r *Issue) *IssueImpl {
	return &IssueImpl{Issue: r}
}

func NewIssueMsg(data []byte) *IssueImpl {
	var mImpl IssueImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func CredentialAttach(p *Issue) (data []byte, err error) {
	return base64.StdEncoding.DecodeString(p.CredentialsAttach[0].Data.Base64)
}

func (p *IssueImpl) checkThread() {
	p.Issue.Thread = decorator.CheckThread(p.Issue.Thread, p.Issue.ID)
}

// MARK: Struct
type IssueImpl struct {
	*Issue
}

func NewCredentialsAttach(attach []byte) []decorator.Attachment {
	data := decorator.AttachmentData{
		Base64: base64.StdEncoding.EncodeToString(attach)}
	rp := []decorator.Attachment{{
		ID:       "libindy-cred-0",
		MimeType: "application/json",
		Data:     data,
	}}
	return rp
}

func (p *IssueImpl) ID() string {
	return p.Issue.ID
}

func (p *IssueImpl) Type() string {
	return p.Issue.Type
}

func (p *IssueImpl) SetID(id string) {
	p.Issue.ID = id
}

func (p *IssueImpl) SetType(t string) {
	p.Issue.Type = t
}

func (p *IssueImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *IssueImpl) Thread() *decorator.Thread {
	//if p.Issue.Thread == nil {}
	return p.Issue.Thread
}

func (p *IssueImpl) FieldObj() interface{} {
	return p.Issue
}
