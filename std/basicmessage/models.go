package basicmessage

import (
	"errors"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

// todo: move this to right place later!
type AriesTime struct {
	time.Time
}

// use generate errors with ACAPy when sending basic messages
// const ISO8601 = "2006-01-02 15:04:05.999999999Z"
const ISO8601 = "2006-01-02 15:04:05.999999Z"

type Basicmessage struct {
	Type     string            `json:"@type,omitempty"`
	ID       string            `json:"@id,omitempty"`
	Thread   *decorator.Thread `json:"~thread,omitempty"`
	Content  string            `json:"content"`
	SentTime AriesTime         `json:"sent_time"`
}

func validateTimestamp(timeStr string) (t time.Time, err error) {
	acceptedFormats := []string{ISO8601, time.RFC3339}
	for _, fmt := range acceptedFormats {
		if t, err = time.Parse(fmt, timeStr); err == nil {
			break
		}
	}
	return
}

func (at *AriesTime) UnmarshalJSON(b []byte) (err error) {
	defer err2.Return(&err)

	t := try.To1(validateTimestamp(strings.Trim(string(b), "\"")))

	*at = AriesTime{Time: t}
	return
}

func (at AriesTime) MarshalJSON() ([]byte, error) {
	//	return (time.Time(t)).MarshalJSON()

	// below taken from Go standard lib
	t := at.Time
	if y := t.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(ISO8601)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, ISO8601)
	b = append(b, '"')
	return b, nil
}

func (at AriesTime) String() string {
	return at.Time.String()
}
