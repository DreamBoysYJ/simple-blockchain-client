package rpcserver

import (
	encodingJson "encoding/json"
	"fmt"

	"net/http"
	"simple_p2p_client/account"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/mediator"
	"simple_p2p_client/protocol_constants"
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

type SendRawTransactionArgs struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Nonce uint64 `json:"nonce"`
}

type SendRawTransactionReply struct {
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

type LastBlockArgs struct {
}

type LastBlockReply struct {
	LastBlock blockchain.Block `json:"lastBlock"`
}

func (s *RpcService) GetLastBlock(r *http.Request, args *LastBlockArgs, reply *LastBlockReply) error {
	// LevelDB에서 "lastblock" key 조회
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError("RPC: Failed to get lastblock")
		return fmt.Errorf("failed to access the database: %v", err)
	}

	// lastblock 가져오기
	lastblockData, err := dbInstance.Get([]byte("lastblock"), nil)
	if err != nil {
		utils.PrintError("RPC : Failed to retrieve lastblock value")
		return fmt.Errorf("failed to get lastblock:%v", err)
	}

	// JSON 문자열을 블록 구조체로 변환
	var lastBlock blockchain.Block
	err = encodingJson.Unmarshal(lastblockData, &lastBlock)
	if err != nil {
		utils.PrintError("RPC : Failed to unmarshal block data")
		return fmt.Errorf("failed to parse block data: %v", err)
	}

	// 응답에 블록 데이터 추가
	reply.LastBlock = lastBlock
	return nil

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

// func (s *RpcService) SendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {

// 	// 1. Transaction Fields 검증
// 	err := blockchain.ValidateTransactionFields(args.From, args.To, args.Value, args.Signature, args.Nonce)
// 	if err != nil {
// 		return fmt.Errorf("transaction validation failed : %w", err)
// 	}

// 	// 2. Sig 검증

// 	// Transaction message = from,to,value,nonce

// 	message := fmt.Sprintf("%s%s%s%d", args.From, args.To, args.Value, args.Nonce)

// 	messageHash := utils.Keccak256([]byte(message))

// 	hexSignature, err := hex.DecodeString(args.Signature)
// 	if err != nil {
// 		return fmt.Errorf("invalid signature format")
// 	}

// 	isValidSig, err := blockchain.VerifySignature(messageHash, hexSignature, args.From)

// 	if err != nil {
// 		return fmt.Errorf("signature verification failed : %v", err)
// 	}
// 	if !isValidSig {
// 		return fmt.Errorf("signature is invalid")
// 	}

// 	// 3. 계정 상태 확인

// 	err = account.CheckAccountState(args.From, args.To, args.Value, args.Nonce)
// 	if err != nil {
// 		return fmt.Errorf("account state validation failed : %v", err)
// 	}

// 	valueBigInt, err := utils.ConvertStringToBigInt(args.Value)
// 	if err != nil {
// 		return fmt.Errorf("failed to convert value string to int : %v", err)
// 	}

// 	// 4. 트랜잭션 생성
// 	tx, jsonRawTransactionStr, err := blockchain.CreateTransaction(args.From, args.To, args.Signature, valueBigInt, args.Nonce)
// 	if err != nil {
// 		return fmt.Errorf("failed to create transaction : %v", err)
// 	}

// 	// 5. Mempool에 저장
// 	blockchain.AppendToMempool(tx)

// 	// 6. 피어에 jsonRawTransaction 전파
// 	fmt.Println("JSON RAW TX ::: ", jsonRawTransactionStr)
// 	p2p.HandleSendingMessages(p2p.ConnectedPeers, protocol_constants.P2PTransactionMessage, jsonRawTransactionStr)

// 	// 7. 해시값 반환

// 	reply.TxHash = tx.Hash
// 	return nil
// }

func (s *RpcService) RefactorSendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {

	// 1. 트랜잭션 데이터를 JSON 형식으로 포맷

	valueBigInt, err := utils.ConvertStringToBigInt(args.Value)
	if err != nil {
		return fmt.Errorf("failed to convert value to big.int: %v", err)
	}

	rawTransaction := blockchain.RawTransaction{
		From:      args.From,
		To:        args.To,
		Value:     valueBigInt,
		Nonce:     args.Nonce,
		Signature: args.Signature,
	}

	rawTransactionBytes, err := encodingJson.Marshal(rawTransaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	// 2. 트랜잭션 메시지를 채널로 전달

	mediatorInstance := mediator.GetMediatorInstance()
	mediatorInstance.P2PToBlockchain <- fmt.Sprintf("%c%s", protocol_constants.P2PTransactionMessage, string(rawTransactionBytes))

	// 3. 트랜잭션 해시값 생성 및 반환
	// TODO : 메시지 전파 후 해시를 반환해야 되는데 지금은 그냥 반환 하게 됨.
	transactionHash := utils.Keccak256Hex(rawTransactionBytes)
	reply.TxHash = transactionHash

	// fmt.Println("Transaction sent to mediator. Hash:", transactionHash)
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
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error starting RPC Server: %v\n", err)
	}
	fmt.Printf("[RPC] Server is listening on : %d\n", port)

}

func (s *RpcService) SendRawTransaction(r *http.Request, args *SendRawTransactionArgs, reply *SendRawTransactionReply) error {

	// 1. 메시지 생성 (from, to, value, nonce를 연결)
	message := fmt.Sprintf("%s%s%s%d", args.From, args.To, args.Value, args.Nonce)

	// 2. 메시지 해싱 (Keccak256)
	messageHash := utils.Keccak256([]byte(message))

	// 3. 개인 키 로드 (개인 키를 안전하게 관리해야 함)
	privateKeyHex := "7ac125dda168b44ee9fc0d8db3a804ef86b3cc50206a0112b25373d622cf78f7" // 실제 서비스에서는 안전한 방식으로 관리 필요
	privateKey, err := account.LoadPrivateKey(privateKeyHex)                            // account 패키지에서 개인 키 로드 함수 구현 필요
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}

	// 4. 메시지 서명
	signature, err := account.SignMessage(messageHash, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	// 5. 서명 반환
	reply.Signature = signature
	return nil
}

type GetAccountArgs struct {
	Address string `json:"address"`
}

type GetAccountReply struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func (s *RpcService) GetAccountInfo(r *http.Request, args *GetAccountArgs, reply *GetAccountReply) error {
	// 입력된 주소 유효성 검사
	if !account.IsValidAddress(args.Address) {
		return fmt.Errorf("invalid address format: %s", args.Address)
	}

	// 계정 정보 조회
	accountData, err := account.GetAccount(args.Address)
	if err != nil {
		return fmt.Errorf("failed to retrieve account info: %v", err)
	}

	// 응답 데이터 설정
	reply.Address = args.Address
	reply.Balance = accountData.Balance.String() // big.Int -> 문자열 변환
	reply.Nonce = accountData.Nonce

	return nil
}
