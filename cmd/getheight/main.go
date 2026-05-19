package main

import (
	"fmt"
	"log"

	"github.com/Eiyaro/Eiyaro/infrastructure/network/rpcclient"
)

func main() {
	// Connect to RPC server
	rpcClient, err := rpcclient.NewRPCClient("127.0.0.1:42420")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer rpcClient.Close()

	// Get block count
	response, err := rpcClient.GetBlockCount()
	if err != nil {
		log.Fatalf("Failed to get block count: %v", err)
	}

	fmt.Println("========================================")
	fmt.Printf("Current Block Height: %d\n", response.BlockCount-1)
	fmt.Printf("Total Blocks: %d\n", response.BlockCount)
	fmt.Printf("Header Count: %d\n", response.HeaderCount)
	fmt.Println("========================================")
}
