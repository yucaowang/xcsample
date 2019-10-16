package client

import (
	"context"
	"errors"
	"github.com/xuperchain/xuperunion/common"
	crypto_client "github.com/xuperchain/xuperunion/crypto/client"
	"github.com/xuperchain/xuperunion/crypto/hash"
	"github.com/xuperchain/xuperunion/global"
	"github.com/xuperchain/xuperunion/pb"
	"github.com/xuperchain/xuperunion/utxo"
	"github.com/xuperchain/xuperunion/utxo/txhash"
	"log"
	"math/big"
	"strings"
	"time"
)

type INotifier func(*pb.TxStatus) error

func (client *XClient) Transfer(ctx context.Context, bcname string, to string, amount string, fee string, desc []byte, frozen int64) (*pb.TxStatus, error) {
	in := &pb.TxData{
		Bcname:     bcname,
		FromAddr:   client.address,
		FromPubkey: client.publicKey,
		FromScrkey: client.privateKey,
		Nonce:      global.GenNonce(),
		Timestamp:  time.Now().UnixNano(),
		Desc:       desc,
		Header:     global.GHeader(),
	}
	account := pb.TxDataAccount{}
	account.Address = to
	if strings.HasPrefix(account.Address, "--") {
		return nil, ExtraErrInvalidAddress
	}
	account.Amount = amount
	account.FrozenHeight = frozen
	accounts := []*pb.TxDataAccount{&account}
	if fee != "0" {
		fee := &pb.TxDataAccount{Address: "$", Amount: fee}
		accounts = append(accounts, fee)
	}
	in.Account = accounts
	r, err := client.GenerateLocalTx(ctx, in, false)
	if err != nil {
		return nil, err
	}
	in2 := &pb.TxStatus{
		Header: r.Header,
		Bcname: in.Bcname,
		Txid:   r.Tx.Txid,
		Tx:     r.Tx,
	}
	r2, err := client.PostTx(ctx, in2)
	if err != nil {
		return nil, err
	}
	if r2.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, common.ServerError{r2.Header.Error}
	}
	return r, nil
}
func genInitSign(tx *pb.Transaction, in *pb.TxData) ([]*pb.SignatureInfo, error) {
	fromPubkey := in.FromPubkey
	fromScrkey := in.FromScrkey
	cryptoClient, err := crypto_client.CreateCryptoClient(crypto_client.CryptoTypeDefault)
	if err != nil {
		return nil, errors.New("Create crypto client error")
	}
	signTx, err := txhash.ProcessSignTx(cryptoClient, tx, []byte(fromScrkey))
	if err != nil {
		return nil, err
	}
	signInfo := &pb.SignatureInfo{
		PublicKey: fromPubkey,
		Sign:      signTx,
	}
	signInfos := []*pb.SignatureInfo{}
	signInfos = append(signInfos, signInfo)
	return signInfos, nil
}
func (client *XClient) GenerateLocalTx(ctx context.Context, in *pb.TxData, autogen bool) (*pb.TxStatus, error) {
	//组装txoutput
	if in.Header == nil {
		in.Header = global.GHeader()
	}
	bigZero := big.NewInt(0)
	totalNeed := bigZero
	tx := &pb.Transaction{
		Version:   utxo.TxVersion,
		Coinbase:  false,
		Desc:      in.Desc,
		Nonce:     in.Nonce,
		Timestamp: in.Timestamp,
		Autogen:   autogen,
	}
	for _, acc := range in.Account {
		amount, ok := big.NewInt(0).SetString(acc.Amount, 10)
		if !ok {
			return nil, ExtraErrInvalidAmount
		}
		if amount.Cmp(bigZero) < 0 {
			return nil, ExtraErrNegativeAmount
		}
		totalNeed.Add(totalNeed, amount)
		txOutput := &pb.TxOutput{}
		txOutput.ToAddr = []byte(acc.Address)
		txOutput.Amount = amount.Bytes()
		txOutput.FrozenHeight = acc.FrozenHeight
		tx.TxOutputs = append(tx.TxOutputs, txOutput)
	}
	//组装txinput
	txInputs, deltaTxOutput, err := client.MakeTxInputs(ctx, in, totalNeed)
	if err != nil {
		return nil, err
	}
	tx.TxInputs = txInputs
	if deltaTxOutput != nil {
		tx.TxOutputs = append(tx.TxOutputs, deltaTxOutput)
	}
	authRequire := in.FromAddr
	tx.AuthRequire = append(tx.AuthRequire, authRequire)
	tx.Initiator = in.FromAddr
	signInfos, err := genInitSign(tx, in)
	if err != nil {
		return nil, err
	}
	tx.InitiatorSigns = signInfos
	tx.AuthRequireSigns = signInfos
	tx.Txid, _ = txhash.MakeTransactionID(tx)
	//TODO 判断交易大小
	return &pb.TxStatus{
		Header: in.Header,
		Bcname: in.Bcname,
		Txid:   tx.Txid,
		Status: pb.TransactionStatus_UNCONFIRM,
		Tx:     tx,
	}, nil
}

//组装txinput
func (client *XClient) MakeTxInputs(ctx context.Context, in *pb.TxData, totalNeed *big.Int) ([]*pb.TxInput, *pb.TxOutput, error) {
	skByte, err := client.cryptoClient.GetEcdsaPrivateKeyFromJSON([]byte(client.privateKey))
	if err != nil {
		return nil, nil, err
	}
	client.cryptoClient.SignECDSA(
		skByte, hash.DoubleSha256([]byte(in.Bcname+in.FromAddr+totalNeed.String())))
	ui := &pb.UtxoInput{
		Header:    in.Header,
		Bcname:    in.Bcname,
		Address:   in.FromAddr,
		TotalNeed: totalNeed.String(),
		NeedLock:  true,
	}
	utxoRes, selectErr := client.SelectUTXO(ctx, ui)
	if selectErr != nil {
		return nil, nil, selectErr
	}
	if utxoRes.Header.Error != pb.XChainErrorEnum_SUCCESS {
		return nil, nil, common.ServerError{utxoRes.Header.Error}
	}
	var txTxInputs []*pb.TxInput
	var txOutput *pb.TxOutput
	for _, utxo := range utxoRes.UtxoList {
		//先检查目标转账地址的格式是否正确
		addrChkResult, _ := client.cryptoClient.CheckAddressFormat(string(utxo.ToAddr))
		if addrChkResult == false {
			return nil, nil, ExtraErrInvalidAddress
		}
		txInput := new(pb.TxInput)
		txInput.RefTxid = utxo.RefTxid
		txInput.RefOffset = utxo.RefOffset
		txInput.FromAddr = utxo.ToAddr
		txInput.Amount = utxo.Amount
		txTxInputs = append(txTxInputs, txInput)
	}
	utxoTotal, ok := big.NewInt(0).SetString(utxoRes.TotalSelected, 10)
	if !ok {
		return nil, nil, ExtraErrInvalidAmount
	}
	// 多出来的utxo需要再转给自己
	if utxoTotal.Cmp(totalNeed) > 0 {
		delta := utxoTotal.Sub(utxoTotal, totalNeed)
		txOutput = &pb.TxOutput{
			ToAddr: []byte(in.FromAddr), // 收款人就是汇款人自己
			Amount: delta.Bytes(),
		}
	}
	return txTxInputs, txOutput, nil
}
func (client *XClient) WatchTx(tx *pb.TxStatus, notifier INotifier) error {
	if len(client.notifierChan) >= CHAN_SIZE || tx == nil {
		return ExtraErrOverloadNotifier
	}
	client.notifierChan <- &Notifier{
		tx: tx,
		n:  notifier,
	}
	client.wgNi.Add(1)
	return nil
}
func (client *XClient) RunInBackground() {
	for {
		select {
		case <-client.stop:
			log.Println("quit")
			return
		case newtx := <-client.notifierChan:
			client.wg.Add(1)
			go client.run(newtx)
		}
	}
}
func (client *XClient) run(ni *Notifier) {
	defer client.wg.Done()
	status, err := client.QueryTx(context.Background(), ni.tx)
	if err != nil {
		log.Println(err)
		return
	}
	if status.Header.Error != pb.XChainErrorEnum_SUCCESS {
		log.Println(common.ServerError{status.Header.Error})
		return
	}
	//如果是为确认的交易，重复查询交易状态
	if status.Status == pb.TransactionStatus_UNCONFIRM || status.Status == pb.TransactionStatus_UNDEFINE {
		ni.qc += 1
		time.Sleep(1500 * time.Millisecond)
		client.notifierChan <- ni
	} else {
		//执行回调
		ni.tx = status
		ni.n(ni.tx)
		client.wgNi.Done()
	}
}
