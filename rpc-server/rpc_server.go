package rpcserver

import (
	"encoding/hex"
	"fmt"

	"net/http"
	"simple_p2p_client/account"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/p2p"
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

type NewAccountArgs struct {
}

type NewAccountReply struct {
	PrivateKey string
	Address    string
}

func (s *RpcService) NewAccount(r *http.Request, args *NewAccountArgs, reply *NewAccountReply) error {

	privateKey, address, err := account.CreateAccount()
	if err != nil {
		return fmt.Errorf("making new account error : %v", err)
	}

	reply.Address = address
	reply.PrivateKey = privateKey
	return nil
}

func (s *RpcService) SendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {

	// 1. Transaction Fields 검증
	err := blockchain.ValidateTransactionFields(args.From, args.To, args.Value, args.Signature, args.Nonce)
	if err != nil {
		return fmt.Errorf("transaction validation failed : %w", err)
	}

	// 2. Sig 검증

	// Transaction message = from,to,value,nonce

	message := fmt.Sprintf("%s%s%s%d", args.From, args.To, args.Value, args.Nonce)

	messageHash := utils.Keccak256([]byte(message))

	hewSignature, err := hex.DecodeString(args.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature format")
	}

	isValidSig, err := blockchain.VerifySignature(messageHash, hewSignature, args.From)

	if err != nil {
		return fmt.Errorf("signature verification failed : %v", err)
	}
	if !isValidSig {
		return fmt.Errorf("signature is invalid")
	}

	// 3. 계정 상태 확인

	err = account.CheckAccountState(args.From, args.To, args.Value, args.Nonce)
	if err != nil {
		return fmt.Errorf("account state validation failed : %v", err)
	}

	// 4. mempool에 저장

	valueBigInt, err := utils.ConvertStringToBigInt(args.Value)
	if err != nil {
		return fmt.Errorf("failed to convert value string to int : %v", err)
	}

	// 4. 트랜잭션 생성
	tx, jsonRawTransactionStr, err := blockchain.CreateTransaction(args.From, args.To, valueBigInt, args.Nonce)
	if err != nil {
		return fmt.Errorf("failed to create transaction : %v", err)
	}

	// 5. Mempool에 저장
	blockchain.AppendToMempool(tx)

	// 6. 피어에 jsonRawTransaction 전파
	p2p.HandleSendingMessages(p2p.ConnectedPeers, jsonRawTransactionStr)

	// 7. 해시값 반환

	reply.TxHash = tx.Hash
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
