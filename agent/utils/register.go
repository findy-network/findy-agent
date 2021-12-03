package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type (
	keyDID    = string
	valueType = []string
)

type regMapType map[keyDID]valueType

type Reg struct {
	r regMapType // stores Agents already on-boarded, has Email as key
	l sync.Mutex // sync obj for register
}

func newReg(data []byte) (r *regMapType) {
	r = new(regMapType)
	err := json.Unmarshal(data, r)
	if err != nil {
		panic(fmt.Sprintln("Error marshalling from JSON: ", err.Error()))
	}
	return
}

func (r *Reg) Exist(key keyDID) bool {
	r.l.Lock()
	defer r.l.Unlock()
	_, ok := r.r[key]
	return ok
}

func (r *Reg) Add(key keyDID, value ...string) {
	glog.V(3).Infof("Handshake register add: %s -> %s\n", key, value)
	r.l.Lock()
	defer r.l.Unlock()
	r.r[key] = value
}

func (r *Reg) Load(filename string) (err error) {
	defer err2.Return(&err)

	r.l.Lock()
	defer r.l.Unlock()

	if filename == "" {
		r.r = make(regMapType)
		return nil
	}

	data, err := readJSONFile(filename)
	if err != nil && os.IsNotExist(err) {
		err2.Check(writeJSONFile(filename, []byte("{}")))
		data, err = readJSONFile(filename)
	}
	err2.Check(err)

	r.r = *newReg(data)
	return nil
}

func (r *Reg) Save(filename string) (err error) {
	r.l.Lock()
	defer r.l.Unlock()

	var data []byte
	if data, err = json.MarshalIndent(r.r, "", "\t"); err != nil {
		return err
	}
	return writeJSONFile(filename, data)
}

func (r *Reg) EnumValues(handler func(k keyDID, v []string) bool) {
	r.l.Lock()
	defer r.l.Unlock()
	for k, v := range r.r {
		if !handler(k, v) {
			break
		}
	}
}

func (r *Reg) Reset(filename string) (err error) {
	defer err2.Annotate("resetting", &err)
	err2.Check(r.Load(""))       // reset data
	err2.Check(r.Save(filename)) // save reset data to file
	return err
}

func writeJSONFile(name string, json []byte) error {
	err := ioutil.WriteFile(name, json, 0644)
	return err
}

func readJSONFile(name string) ([]byte, error) {
	result, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return result, err
}
