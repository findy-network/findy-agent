package utils

import (
	"os"
	"reflect"
	"testing"

	"github.com/findy-network/findy-common-go/dto"
	"github.com/stretchr/testify/assert"
)

func cloneMap(tgt, src regMapType) {
	for k, v := range src {
		tgt[k] = v
	}
}

func Test_newReg_and_toJsonBytes(t *testing.T) {
	testReg1 := Reg{}
	testReg2 := Reg{}
	testReg3 := Reg{}

	r1 := make(regMapType)
	r1["a"] = []string{"A", "a"}
	r1["b"] = []string{"B", "b"}
	testReg1.r = make(regMapType)
	cloneMap(testReg1.r, r1)
	jsonBytes1 := dto.ToJSONBytes(testReg1.r)

	r2 := make(regMapType)
	r2["c"] = []string{"C", "c"}
	testReg2.r = make(regMapType)
	cloneMap(testReg2.r, r2)
	jsonBytes2 := dto.ToJSONBytes(testReg2.r)

	r3 := buildRegistryData()
	testReg3.r = make(regMapType)
	cloneMap(testReg3.r, r3)
	jsonBytes3 := dto.ToJSONBytes(testReg3.r)

	type args struct {
		data []byte
	}
	tests := []struct {
		name  string
		args  args
		wantR *regMapType
	}{
		{"1st", args{data: jsonBytes1}, &r1},
		{"2nd", args{data: jsonBytes2}, &r2},
		{"3rd", args{data: jsonBytes3}, &r3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotR := newReg(tt.args.data); !reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("newReg() = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}

func Test_reg_save_and_load(t *testing.T) {
	r3 := buildRegistryData()

	filename := "42342342tmp4234.json"

	type fields struct {
		r regMapType
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"1", fields{r: r3}, args{filename}, false},
	}
	for index := range tests {
		tt := &tests[index]
		t.Run(tt.name, func(t *testing.T) {
			r := &Reg{
				r: tt.fields.r,
			}
			if err := r.Save(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Reg.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := r.Load(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Reg.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	_ = os.Remove(filename)
}

func buildRegistryData() regMapType {
	r3 := make(regMapType)
	r3["1"] = []string{"D", "ACDID"}
	r3["2"] = []string{"D jkalsdfjlksadf kkdf ksdf llsdl", "ACDID"}
	r3["3"] = []string{"Dajsdlkfas dlkfas;lkd fja;lksdf", "ACDID"}
	r3["4"] = []string{"Dasdjfasjdfl", "ACDID"}
	r3["5"] = []string{"D", "ACDID"}
	r3["6"] = []string{"D", "ACDID"}
	return r3
}

func TestReg_Exist(t *testing.T) {
	reg := Reg{r: buildRegistryData()}
	assert.True(t, reg.Exist("3"))
}

func Test_reg_enumValues(t *testing.T) {
	r3 := buildRegistryData()

	var currentK string
	var currentV string
	count := 0
	f := func(k keyDID, v []string) bool {
		count++
		if count == 3 {
			currentK = k
			currentV = v[0]
			return false
		}
		return true
	}
	type fields struct {
		r regMapType
	}
	type args struct {
		handler func(k keyDID, v []string) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"1", fields{r: r3}, args{handler: f}},
	}
	for index := range tests {
		tt := &tests[index]
		t.Run(tt.name, func(t *testing.T) {
			r := &Reg{
				r: tt.fields.r,
			}
			r.EnumValues(tt.args.handler)
			if r3[currentK][0] != currentV && count == 3 {
				t.Errorf("Reg.EnumValues() k: %s, v: %s, value: %s", currentK, currentV, r3[currentK])
			}
		})
	}
}

func Test_reg_Reset(t *testing.T) {
	r3 := buildRegistryData()

	filename := "42342342tmp4234.json"

	type fields struct {
		r regMapType
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"1", fields{r: r3}, args{filename}, false},
	}
	for index := range tests {
		tt := &tests[index]
		t.Run(tt.name, func(t *testing.T) {
			r := &Reg{
				r: tt.fields.r,
			}
			if len(r.r) <= 0 {
				t.Errorf("Reg.Reset() no data in registry: len = %v", len(r.r))
			}
			if err := r.Save(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Reg.Reset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := r.Load(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Reg.Reset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(r.r) <= 0 {
				t.Errorf("Reg.Reset() no data in registry: len = %v", len(r.r))
			}
			if err := r.Reset(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Reg.Reset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(r.r) != 0 {
				t.Errorf("Reg.Reset() no data in registry: len = %v", len(r.r))
			}
		})
	}
	_ = os.Remove(filename)
}
