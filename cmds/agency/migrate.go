package agency

import (
	"io"
	"path/filepath"

	ag "github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type MigrateCmd struct {
	EnclaveKey string

	InputReg  string
	OutputReg string
}

type _ /*seedAgent*/ struct {
	RootDID  string
	Name     string
	CADID    string
	CAVerKey string
}

func (c MigrateCmd) Validate() (err error) {
	defer err2.Handle(&err)

	assert.NotEmpty(c.InputReg, "need the input file")
	assert.NotEmpty(c.OutputReg, "need the output file")

	return nil
}

func (c MigrateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Handle(&err)

	try.To(c.sealedBox())
	try.To(ag.Register.Load(c.InputReg))

	cmds.Fprintln(w, "This migrate cmd is outdated.")

	//registeredWallets := make(map[string]seedAgent)

	// cmds.Fprintln(w, "Processing starts, please wait..")
	// ag.Register.EnumValues(func(rootDid string, values []string) (next bool) {
	// 	// dont let crash on panics
	// 	defer err2.Catch(func(err error) {
	// 		cmds.Fprintln(w, "enum register error:", err)
	// 	})
	// 	next = true // default is to continue even on error
	// 	email := values[0]
	// 	caDid := values[1]

	// 	rippedEmail := strings.Replace(email, "@", "_", -1)
	// 	_, walletExist := registeredWallets[rippedEmail]
	// 	if !walletExist {
	// 		key, err := enclave.WalletKeyByEmail(email)
	// 		keyByDid, error2 := enclave.WalletKeyByDID(rootDid)
	// 		if err != nil || error2 != nil {
	// 			cmds.Fprintln(w, "cannot get wallet key:", err, email, caDid)
	// 			return true
	// 		}
	// 		if key != keyByDid {
	// 			cmds.Fprintln(w, "keys don't match", key, keyByDid)
	// 		}

	// 		aw := ssi.NewRawWalletCfg(rippedEmail, key)
	// 		if !aw.Exists(false) {
	// 			cmds.Fprintf(w, "wallet '%s' not exist\n", rippedEmail)
	// 			return true
	// 		}

	// 		seed := cloud.NewSeedAgent(rootDid, caDid, "", aw)
	// 		h := try.To1(seed.Migrate())

	// 		a := h.(comm.Receiver)
	// 		vk := a.MyDID().VerKey()

	// 		registeredWallets[rippedEmail] = seedAgent{
	// 			Name:     rippedEmail,
	// 			RootDID:  rootDid,
	// 			CADID:    caDid,
	// 			CAVerKey: vk,
	// 		}

	// 	} else {
	// 		cmds.Fprintln(w, "Duplicate registered wallet:", email)
	// 	}
	// 	return true
	// })
	// cmds.Fprintln(w, "Processing done")

	// cmds.Fprintln(w, "Saving starts")
	// for _, sa := range registeredWallets {
	// 	ag.Register.Add(sa.RootDID, sa.Name, sa.CADID, sa.CAVerKey)
	// }
	// try.To(ag.Register.Save(c.OutputReg))
	// cmds.Fprintln(w, "All ready")

	return nil, nil
}

func (c MigrateCmd) sealedBox() (err error) {
	defer err2.Handle(&err)

	home := utils.IndyBaseDir()
	sealedBoxPath := filepath.Join(home, ".indy_client/enclave.bolt")

	return enclave.InitSealedBox(sealedBoxPath, "", c.EnclaveKey)
}
