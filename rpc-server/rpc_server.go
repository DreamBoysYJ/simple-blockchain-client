package rpcserver

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/decred/dcrd/dcrec/secp256k1"
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

func PublicKeyToBytes(pubKey *secp256k1.PublicKey) []byte {
	return pubKey.SerializeUncompressed()
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
	address, err := PublicKeyToAddress(pubKey)
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

func PublicKeyToAddress(pubKey []byte) (string, error) {
	// pubKey는 압축되지 않은 공개키여야 함(len = 65, 첫 바이트 0x04)

	if len(pubKey) != 65 {
		return "", fmt.Errorf("invalid public key length, must be 65 bytes, but got %d bytes", len(pubKey))
	}

	if pubKey[0] != 0x04 {
		return "", fmt.Errorf("invalid public key format, must start with 0x04 for uncompressed keys, but got %v", pubKey[0])
	}

	// keccak256 계산
	hash := blockchain.Keccak256(pubKey[1:])

	// 마지막 20바이트를 이더리움 주소로 사용
	address := hash[len(hash)-20:]
	// 16진수로 출력하고 0x 붙이기
	return fmt.Sprintf("0x%x", address), nil
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
	if !isValidAddress(args.From) {
		return fmt.Errorf("invalid address : address 'from' format is wrong")
	}

	if !isValidAddress(args.To) {
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
