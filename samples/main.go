package main

import (
	"encoding/hex"
	"fmt"
	"time"

	"xcsample/actions"
)

// This is sample to show how to save data on xuperchain

func main() {
	conf := &actions.Config{
		BcName:     "xuper",
		TargetIP:   "localhost",
		TargetPort: "37101",
	}

	keypath := "./data/keys/"

	mgr, err := actions.NewTransferManager(conf, keypath)
	if err != nil {
		fmt.Println("New TransferManager failed, err=", err)
		return
	}

	// update data to blockchain
	msg := "this is a message I want to upload to blockchain"
	toAddr := "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	amount := "1"

	txid, err := mgr.Transfer(toAddr, amount, []byte(msg))
	if err != nil {
		fmt.Println("data upload failed, err=", err)
		return
	}

	fmt.Println("data post to blockchain successfully, txid=", txid)

	// wait for 3 seconds
	fmt.Println("wait 3 seconds to make sure the transation is confirmed in a block...")
	time.Sleep(time.Duration(3) * time.Second)

	// query data on blockchain
	tx, err := mgr.QueryTx(txid)
	if err != nil {
		fmt.Println("data query failed, err=", err)
		return
	}

	newMsg := tx.GetDesc()
	blockid := hex.EncodeToString(tx.GetBlockid())
	fmt.Printf("found data on blockchain, blockid=%s, txid=%s, message=%s\n", blockid, txid, string(newMsg))
}
