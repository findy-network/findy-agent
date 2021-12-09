package agency

import (
	"io"
	"path/filepath"
	"strings"

	ag "github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

type MigrateCmd struct {
	EnclaveKey string

	InputReg  string
	OutputReg string
}

type seedAgent struct {
	RootDID  string
	Name     string
	CADID    string
	CAVerKey string
}

func (c MigrateCmd) Validate() (err error) {
	defer err2.Return(&err)

	assert.P.NotEmpty(c.InputReg, "need the input file")
	assert.P.NotEmpty(c.OutputReg, "need the output file")

	return nil
}

func (c MigrateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	err2.Check(c.sealedBox())
	err2.Check(ag.Register.Load(c.InputReg))

	registeredWallets := make(map[string]seedAgent)

	cmds.Fprintln(w, "Processing starts, please wait..")
	ag.Register.EnumValues(func(rootDid string, values []string) (next bool) {
		// dont let crash on panics
		defer err2.Catch(func(err error) {
			cmds.Fprintln(w, "enum register error:", err)
		})
		next = true // default is to continue even on error
		email := values[0]
		caDid := values[1]

		rippedEmail := strings.Replace(email, "@", "_", -1)
		_, walletExist := registeredWallets[rippedEmail]
		if !walletExist {
			key, err := enclave.WalletKeyByEmail(email)
			keyByDid, error2 := enclave.WalletKeyByDID(rootDid)
			if err != nil || error2 != nil {
				cmds.Fprintln(w, "cannot get wallet key:", err, email, caDid)
				return true
			}
			if key != keyByDid {
				cmds.Fprintln(w, "keys don't match", key, keyByDid)
			}

			aw := ssi.NewRawWalletCfg(rippedEmail, key)
			if !aw.Exists(false) {
				cmds.Fprintf(w, "wallet '%s' not exist\n", rippedEmail)
				return true
			}

			seed := cloud.NewSeedAgent(rootDid, caDid, aw)
			h, err := seed.Migrate()
			err2.Check(err)

			a := h.(comm.Receiver)
			vk := a.Trans().MessagePipe().In.VerKey()

			registeredWallets[rippedEmail] = seedAgent{
				Name:     rippedEmail,
				RootDID:  rootDid,
				CADID:    caDid,
				CAVerKey: vk,
			}

		} else {
			cmds.Fprintln(w, "Duplicate registered wallet:", email)
		}
		return true
	})
	cmds.Fprintln(w, "Processing done")

	cmds.Fprintln(w, "Saving starts")
	for _, sa := range registeredWallets {
		ag.Register.Add(sa.RootDID, sa.Name, sa.CADID, sa.CAVerKey)
	}
	err2.Check(ag.Register.Save(c.OutputReg))
	cmds.Fprintln(w, "All ready")

	return nil, nil
}

func (c MigrateCmd) sealedBox() (err error) {
	defer err2.Return(&err)

	home := utils.IndyBaseDir()
	sealedBoxPath := filepath.Join(home, ".indy_client/enclave.bolt")

	return enclave.InitSealedBox(sealedBoxPath, "", c.EnclaveKey)
}
