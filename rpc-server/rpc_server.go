package rpcserver

import (
	"fmt"
	"net/http"
	"regexp"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"strconv"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

// req & res struct

type BlockNumberArgs struct {
}

type BlockNumberReply struct {
	BlockNumber int `json:"blockNumber"`
}

type SendTransactionArgs struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
}

type SendTransactionReply struct {
	TxHash string `json:"txHash"`
}

type RpcService struct {
}

func (s *RpcService) GetBlockNumber(r *http.Request, args *BlockNumberArgs, reply *BlockNumberReply) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError("RPC : Failed to get lastblock")
		return fmt.Errorf("failed to get lastblock value : %v", err)
	}
	err = dbInstance.Put([]byte("lastblock"), []byte("234234234"), nil)
	if err != nil {
		fmt.Println("NO")
	}
	value, err := dbInstance.Get([]byte("lastblock"), nil)
	if err != nil {
		fmt.Println("ERRRO")
	}
	valueStr := string(value)
	intValue, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Println("HI")
	}

	reply.BlockNumber = intValue
	return nil

}

func isValidAddress(address string) bool {
	// 0x 시작 && 총 42자 && 40자 16진수
	// ^ : 시작
	// [0-9a-fA-f] : 이 부분은 괄호 안에 있는 문자들 중 하나를 허용
	// {40} 앞 패턴이 몇번 있어야 하는지
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}

func verifySignature()

func (s *RpcService) SendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {

	// 1. Transaction Fields 검증

	// 빈 인자가 있는지
	if args.From == "" || args.To == "" || args.Value == "" {
		return fmt.Errorf("missing required fields : 'from', 'to', or 'value'")
	}

	// Value가 양의 정수인지
	valueInt, err := strconv.Atoi(args.Value)
	if err != nil || valueInt <= 0 {
		return fmt.Errorf("invalid value : must be a positive integer")
	}

	// from, to 주소 양식이 올바른지
	if !isValidAddress(args.From) {
		return fmt.Errorf("invalid address : address 'from' format is wrong")
	}

	if !isValidAddress(args.To) {
		return fmt.Errorf("invalid address : address 'to' format is wrong")
	}

	// 2. Sig 검증

	// 3. 계정 상태 확인

	// 4. mempool에 저장

	// 5. 피어에 전파

	// 6. 해시값 반환
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
