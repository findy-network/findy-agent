package agent

type Result struct{}

func (r Result) JSON() ([]byte, error) {
	return nil, nil
}
