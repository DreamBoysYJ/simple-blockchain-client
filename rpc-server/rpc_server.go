package rpcserver

import (
	"encoding/hex"
	"fmt"

	encodingJson "encoding/json"
	"net/http"
	"simple_p2p_client/account"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/p2p"
	"simple_p2p_client/utils"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

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

func VerifySignature(messageHash []byte, signature []byte, fromAddress string) (bool, error) {

	// 공개키 복구
	pubKey, err := crypto.Ecrecover(messageHash, signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key : %v", err)
	}

	// 공개키를 바이트 배열로 변환 (압축되지 않은 형식)
	fmt.Printf("Recovered public key (uncompressed): %x\n", pubKey)

	// 복구된 공개키로부터 주소 생성
	address, err := account.PublicKeyToAddress(pubKey)
	fmt.Printf("Address: %s\n", address)

	if err != nil {
		return false, fmt.Errorf("failed to create address from public key : %v", err)
	}

	// from과 비교
	if strings.EqualFold(address, fromAddress) {
		return true, nil
	} else {
		return false, nil
	}

}

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
	if !account.IsValidAddress(args.From) {
		return fmt.Errorf("invalid address : address 'from' format is wrong")
	}

	if !account.IsValidAddress(args.To) {
		return fmt.Errorf("invalid address : address 'to' format is wrong")
	}

	// 2. Sig 검증

	// message = from,to,value,nonce

	message := fmt.Sprintf("%s%s%s%d", args.From, args.To, args.Value, args.Nonce)
	fmt.Println("MESSAGE :::", message)

	messageHash := blockchain.Keccak256([]byte(message))
	fmt.Println("MESSAGE HASH :::", messageHash)

	signature, err := hex.DecodeString(args.Signature)
	fmt.Println("SIGNATURE :::", signature)

	if err != nil {
		return fmt.Errorf("invalid signature format")
	}
	isValidSig, err := VerifySignature(messageHash, signature, args.From)

	if err != nil {
		return fmt.Errorf("signature verification failed : %v", err)
	}
	if !isValidSig {
		return fmt.Errorf("signature is invalid")
	}

	// 3. 계정 상태 확인

	// from : 새로 만든 계정이면 안됨
	fromExists, err := account.AccountExists(args.From)
	if err != nil {
		return fmt.Errorf("값을 big.Int로 변환하는 데 실패했습니다: %v", err)
	}
	if !fromExists {
		return fmt.Errorf("from must be already stored")
	}

	// from.balance >= value인지 확인
	fromAccount, err := account.GetAccount(args.From)
	if err != nil {
		return fmt.Errorf("값을 big.Int로 변환하는 데 실패했습니다: %v", err)
	}

	valueBigInt, err := utils.ConvertStringToBigInt(args.Value)
	if err != nil {
		return fmt.Errorf("값을 big.Int로 변환하는 데 실패했습니다: %v", err)
	}
	if fromAccount.Balance.Cmp(valueBigInt) < 0 {
		return fmt.Errorf("not enough money, you have balance : %d", fromAccount.Balance)
	}

	// from.nonce == nonce 확인
	// if fromAccount.Nonce != args.Nonce {
	// 	return fmt.Errorf("from nonce must be same")
	// }

	// to AccountExists 아니면 저장해주기
	toExists, err := account.AccountExists(args.To)
	if err != nil {
		return fmt.Errorf("값을 big.Int로 변환하는 데 실패했습니다: %v", err)
	}
	// to 없는 계정이면
	if !toExists {
		// 만들어주기
		_, err := account.StoreAccount(args.To)
		if err != nil {
			return fmt.Errorf("to account made failed")
		}
	}

	// 4. mempool에 저장

	// tx 해시값 구하기
	rawTransaction := blockchain.RawTransaction{
		From:  args.From,
		To:    args.To,
		Value: valueBigInt,
		Nonce: args.Nonce,
	}

	jsonRawTransaction, err := encodingJson.Marshal(rawTransaction)
	if err != nil {
		return fmt.Errorf("failed to encoding json raw tx")
	}
	jsonRawTransactionHash := blockchain.Keccak256(jsonRawTransaction)
	jsonRawTransactionHashStr := hex.EncodeToString(jsonRawTransactionHash)

	fullTransaction := blockchain.Transaction{
		Hash:  jsonRawTransactionHashStr,
		From:  args.From,
		To:    args.To,
		Value: valueBigInt,
		Nonce: args.Nonce,
	}

	blockchain.Mempool = append(blockchain.Mempool, fullTransaction)

	// 5. 피어에 jsonRawTransaction 전파
	p2p.HandleSendingMessages(p2p.ConnectedPeers, string(jsonRawTransaction))

	// 6. 해시값 반환

	reply.TxHash = "0x" + jsonRawTransactionHashStr

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
