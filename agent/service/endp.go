package service

// Addr is the public access point of an Agent.
type Addr struct {
	Endp string `json:"endpoint"`
	Key  string `json:"verkey"`
}
