package rpcserver

import (
	"fmt"
	"net/http"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

// req & res struct

type BlockNumberArgs struct {
}

type BlockNumberReply struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SendTransactionArgs struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Nonce uint64 `json:"nonce"`
}

type SendTransactionReply struct {
	TxHash string `json:"txHash"`
}

type RpcService struct {
}

func (s *RpcService) GetBlockNumber(r *http.Request, args *BlockNumberArgs, reply *BlockNumberReply) error {
	// dbInstance, err	:= leveldb.GetDBInstance()
	// if err != nil {
	// 	utils.PrintError("RPC : Failed to get lastblock")
	// 	return fmt.Errorf("Failed to get lastblock value : %v", err)
	// }
	// var lastBlock *BlockNumberReply
	// lastblock ,err := leveldb.GetLastBlock(dbInstance)
	// reply.BlockNumber = lastBlock.BlockNumber
	reply.BlockNumber = 100000
	return nil

}

func (s *RpcService) SendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {
	reply.TxHash = "sew2342342343"
	return nil
}

func StartRpcServer(port int) {

	// Create Gorilla RPC server
	server := rpc.NewServer()

	// Register JSON Codec
	server.RegisterCodec(json.NewCodec(), "application/json")

	// Register RPC Service
	server.RegisterService(new(RpcService), "")

	// Set HTTP Handler
	http.Handle("/rpc", server)

	// Start Server
	fmt.Printf("Starting RPC Server on Port %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error starting RPC Server: %v\n", err)
	}

}
