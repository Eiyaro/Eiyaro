package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/client"
	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/pb"
	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/utils"
)

func createUnsignedTransaction(conf *createUnsignedTransactionConfig) error {
	daemonClient, tearDown, err := client.Connect(conf.DaemonAddress)
	if err != nil {
		return err
	}
	defer tearDown()

	ctx, cancel := context.WithTimeout(context.Background(), daemonTimeout)
	defer cancel()

	sendAmountSey, err := utils.EYToSey(conf.SendAmount)
	if err != nil {
		return err
	}

	response, err := daemonClient.CreateUnsignedTransactions(ctx, &pb.CreateUnsignedTransactionsRequest{
		From:                     conf.FromAddresses,
		Address:                  conf.ToAddress,
		Amount:                   sendAmountSey,
		IsSendAll:                conf.IsSendAll,
		UseExistingChangeAddress: conf.UseExistingChangeAddress,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Created unsigned transaction")
	fmt.Println(encodeTransactionsToHex(response.UnsignedTransactions))

	return nil
}
