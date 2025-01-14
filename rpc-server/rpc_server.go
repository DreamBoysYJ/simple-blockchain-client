package rpcserver

import (
	"fmt"

	"net/http"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

type RpcService struct {
}

// 서버 초기화 및 공통 설정
func StartRpcServer(port int) {

	// Create Gorilla RPC server
	server := rpc.NewServer()

	// Register JSON Codec
	server.RegisterCodec(json.NewCodec(), "application/json")

	// Register RPC Service
	server.RegisterService(new(RpcService), "rpc")
	server.RegisterService(new(BlockAPI), "block")

	server.RegisterService(new(TransactionAPI), "transaction")

	server.RegisterService(new(AccountAPI), "account")

	// Set HTTP Handler
	http.Handle("/rpc", server)

	// Start Server
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error starting RPC Server: %v\n", err)
	}
	fmt.Printf("[RPC] Server is listening on : %d\n", port)

}
