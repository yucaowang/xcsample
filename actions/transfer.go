package actions

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"github.com/xuperchain/xuperunion/pb"

	client "xcsample/client"
)

// TransferManager the manager of basic actions
type ActionManager struct {
	conf *Config
	xua  *XUserAccount
	xc   *client.XClient
}

// NewTransferManager create TransferManager instance
func NewTransferManager(conf *Config, keypath string) (*ActionManager, error) {
	ta := &ActionManager{
		conf: conf,
	}

	if err := ta.initClient(keypath); err != nil {
		return nil, err
	}

	return ta, nil
}

// Transfer is a common transaction which transfer given amount of xuper to another address
//   @param to: the receive address
//   @param amount: xuper amount in string
//   @param desc: comments on this transaction
//   @return txid string: the txid of tranction on blockchain
func (ta *ActionManager) Transfer(to string, amount string, desc []byte) (string, error) {
	status, err := ta.xc.Transfer(context.Background(), ta.conf.BcName, to, amount, "0", desc, 0)
	if err != nil {
		return "", err
	}
	txid := hex.EncodeToString(status.GetTxid())
	fmt.Println("normalTransfer success with txid:", txid)
	return txid, nil
}

// QueryTx is the interface to query a transation by txid
func (ta *ActionManager) QueryTx(txid string) (*pb.Transaction, error) {
	rawTxid, _ := hex.DecodeString(txid)
	txStatus := &pb.TxStatus{
		Txid:   rawTxid,
		Bcname: ta.conf.BcName,
	}
	tx, err := ta.xc.QueryTx(context.Background(), txStatus)
	if err != nil {
		return nil, err
	}

	return tx.GetTx(), nil
}

func (ta *ActionManager) loadKeys(path string) error {
	address, err := ioutil.ReadFile(path + "address")
	if err != nil {
		return err
	}
	sk, err := ioutil.ReadFile(path + "private.key")
	if err != nil {
		return err
	}
	pk, err := ioutil.ReadFile(path + "public.key")
	if err != nil {
		return err
	}
	xua := &XUserAccount{
		Address:    string(address),
		PublicKey:  pk,
		PrivateKey: sk,
	}
	ta.xua = xua
	return nil
}

func (ta *ActionManager) initClient(path string) error {
	host := ta.conf.TargetIP + ":" + ta.conf.TargetPort
	err := ta.loadKeys(path)
	if err != nil {
		return err
	}
	addrop := client.WithAddress(ta.xua.Address)
	pkop := client.WithPublicKey(string(ta.xua.PublicKey))
	skop := client.WithPrivateKey(string(ta.xua.PrivateKey))
	xc, err := client.NewXClientWithOpts(host, addrop, pkop, skop)
	if err != nil {
		return err
	}
	ta.xc = xc
	return nil
}
