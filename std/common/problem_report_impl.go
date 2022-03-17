package common

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
)

var ProblemReportCreator = &ProblemReportFactor{}

type ProblemReportFactor struct{}

func (f *ProblemReportFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &ProblemReport{
		Type:        init.Type,
		ID:          init.AID,
		Description: Code{Code: init.Info},
		Thread:      decorator.CheckThread(init.Thread, init.AID),
	}
	return NewProblemReport(m)
}

func (f *ProblemReportFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewProblemReportMsg(data)
}

func init() {
	gob.Register(&ProblemReportImpl{})
	aries.Creator.Add(pltype.NotificationProblemReport, ProblemReportCreator)
	aries.Creator.Add(pltype.DIDOrgNotificationProblemReport, ProblemReportCreator)
}

func NewProblemReport(r *ProblemReport) *ProblemReportImpl {
	return &ProblemReportImpl{ProblemReport: r}
}

func NewProblemReportMsg(data []byte) *ProblemReportImpl {
	var mImpl ProblemReportImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func (p *ProblemReportImpl) checkThread() {
	p.ProblemReport.Thread = decorator.CheckThread(p.ProblemReport.Thread, p.ProblemReport.ID)
}

// MARK: Struct
type ProblemReportImpl struct {
	*ProblemReport
}

func (p *ProblemReportImpl) ID() string {
	return p.ProblemReport.ID
}

func (p *ProblemReportImpl) Type() string {
	return p.ProblemReport.Type
}

func (p *ProblemReportImpl) SetID(id string) {
	p.ProblemReport.ID = id
}

func (p *ProblemReportImpl) SetType(t string) {
	p.ProblemReport.Type = t
}

func (p *ProblemReportImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *ProblemReportImpl) Thread() *decorator.Thread {
	//if p.ProblemReport.Thread == nil {}
	return p.ProblemReport.Thread
}

func (p *ProblemReportImpl) FieldObj() interface{} {
	return p.ProblemReport
}
