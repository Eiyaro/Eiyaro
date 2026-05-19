package main

import (
	"context"
	"fmt"

	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/client"
	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/pb"
	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/utils"
)

func balance(conf *balanceConfig) error {
	daemonClient, tearDown, err := client.Connect(conf.DaemonAddress)
	if err != nil {
		return err
	}
	defer tearDown()

	ctx, cancel := context.WithTimeout(context.Background(), daemonTimeout)
	defer cancel()
	response, err := daemonClient.GetBalance(ctx, &pb.GetBalanceRequest{})
	if err != nil {
		return err
	}

	pendingSuffix := ""
	if response.Pending > 0 {
		pendingSuffix = " (pending)"
	}
	if conf.Verbose {
		pendingSuffix = ""
		println("Address                                                                       Available  ")
		println("-----------------------------------------------------------------------------------------")
		for _, addressBalance := range response.AddressBalances {
			fmt.Printf("%s %s %s\n", addressBalance.Address, utils.FormatEY(addressBalance.Available), utils.FormatEY(addressBalance.Pending))
		}
		println("-----------------------------------------------------------------------------------------")
		print("                                                 ")
	}
	fmt.Printf("Total balance, EY %s %s%s\n", utils.FormatEY(response.Available), utils.FormatEY(response.Pending), pendingSuffix)

	return nil
}
