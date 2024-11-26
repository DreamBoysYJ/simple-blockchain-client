package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/account"

	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	db "github.com/syndtr/goleveldb/leveldb"
)

// 트랜잭션 구조체

type RawTransaction struct {
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     *big.Int `json:"value"`
	Nonce     uint64   `json:"nonce"`
	Signature string   `json:"signature"`
}

type Transaction struct {
	Hash      string   `json:"hash"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     *big.Int `json:"value"`
	Nonce     uint64   `json:"nonce"`
	Signature string   `json:"signature"`
}

// 블록 구조체
type Block struct {
	Number      uint64        `json:"number"`
	Hash        string        `json:"hash"`
	ParentHash  string        `json:"parentHash"`
	Timestamp   time.Time     `json:"timestamp"`
	Transaction []Transaction `json:"transaction"`
	Miner       string        `json:"miner"`
}

// Mempool
var (
	Mempool = make(map[string]Transaction)
	mu      sync.Mutex
)

// 트랜잭션 관련 함수

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

func ValidateTransactionFields(from, to, value, signature string, nonce uint64) error {

	// 1. 빈 인자 없는지 확인
	if from == "" || to == "" || value == "" || signature == "" {
		return fmt.Errorf("missing required fields: 'from', 'to', 'value', 'signature'")
	}

	// 2. value > 0 인지
	valueInt, err := strconv.Atoi(value)
	if err != nil || valueInt <= 0 {
		return fmt.Errorf("invalid value : must be a positive integer")
	}

	// 3. nonce >=0
	// if nonce < 0 {
	// 	return fmt.Errorf("inavlid nonce : must be a non-negative integer")
	// }

	// 4. from, to 주소 양식이 올바른지
	if !account.IsValidAddress(from) {
		return fmt.Errorf("invalid address : address 'from' format is wrong")
	}
	if !account.IsValidAddress(to) {
		return fmt.Errorf("invalid address : address 'to' format is wrong")
	}

	return nil

}

// ///
func CreateTx() {
	data := []byte("hello")

	hash := utils.Keccak256(data)
	// %x : 16진수
	fmt.Printf("Keccack 해시값 ::: %x\n ", hash)
}

func ValidateBlock(newBlock *Block) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get dbinstance: %v", err))

	}
	lastblockJSON, err := leveldb.GetLastBlock(dbInstance)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get lastblock: %v", err))

	}

	// json 데이터를 블록 구조체로
	var lastBlock Block
	json.Unmarshal(lastblockJSON, &lastBlock)

	// 새 블록과 이전 블록의 해시값 비교

	if lastBlock.Hash == newBlock.ParentHash {
		return nil
	} else {
		return fmt.Errorf("Invalid block : parent hash : %s , last block hash : %s", newBlock.ParentHash, lastBlock.Hash)
	}
}

func StoreBlock(block *Block) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get dbinstance: %v", err))
	}

	blockJSON, err := json.Marshal(block)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to marshal block: %v", err))
	}

	batch := new(db.Batch)

	// 작업 추가
	batch.Put([]byte(block.Hash), blockJSON)
	batch.Put([]byte("lastblock"), blockJSON)

	// batch 실행
	dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("failed to write batch to LevelDB: %w", err)
	}
	return nil

}

func CheckTxInMempool(txHash string) bool {
	if _, exists := Mempool[txHash]; exists {
		return true
	}
	return false
}

func AppendToMempool(tx Transaction) error {
	mu.Lock()
	defer mu.Unlock()

	// 중복 tx인지 확인
	if _, exists := Mempool[tx.Hash]; exists {
		// 중복 트랜잭션
		return fmt.Errorf("tx already in mempool")

	}

	// 추가
	Mempool[tx.Hash] = tx
	fmt.Println("트랜잭션 멤풀에 들어갔따!")

	return nil

}

func CreateTransaction(from, to, signature string, value *big.Int, nonce uint64) (Transaction, string, error) {
	// 1. RawTransaction 생성
	rawTransaction := RawTransaction{
		From:      from,
		To:        to,
		Value:     value,
		Nonce:     nonce,
		Signature: signature,
	}

	// 2. JSON 직렬화
	jsonRawTransaction, err := json.Marshal(rawTransaction)
	if err != nil {
		return Transaction{}, "", fmt.Errorf("failed to encode raw transaction to JSON: %v", err)

	}

	// 3. 해시 생성
	jsonRawTransactionHash := utils.Keccak256(jsonRawTransaction)
	jsoonRawTransactionHashStr := fmt.Sprintf("0x%s", utils.BytesToHex(jsonRawTransactionHash))

	// 4. 트랜잭션 생성
	fullTransaction := Transaction{
		Hash:      jsoonRawTransactionHashStr,
		From:      from,
		To:        to,
		Value:     value,
		Nonce:     nonce,
		Signature: signature,
	}

	return fullTransaction, string(jsonRawTransaction), nil
}

// Peer로 부터 트랜잭션을 수신했을 때 검증
func ValidateTransaction(message string) (string, error) {
	// 1. RawTransaction 구조체 변환
	var rawTransaction RawTransaction
	err := json.Unmarshal([]byte(message), &rawTransaction)
	if err != nil {
		return "", fmt.Errorf("failed to parse raw transaction : %v", err)

	}

	// 2. 트랜잭션 필드 검증
	err = ValidateTransactionFields(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Signature, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to validate transaction fields : %w", err)
	}

	// 3. 서명 검증
	txMessage := fmt.Sprintf("%s%s%s%d", rawTransaction.From, rawTransaction.To, rawTransaction.Value, rawTransaction.Nonce)
	txMessageHash := utils.Keccak256([]byte(txMessage))
	hexSignature, err := hex.DecodeString(rawTransaction.Signature)
	if err != nil {
		return "", fmt.Errorf("invalid signature format")
	}
	isValidSig, err := VerifySignature(txMessageHash, hexSignature, rawTransaction.From)

	if err != nil {
		return "", fmt.Errorf("signature verification failed : %v", err)
	}
	if !isValidSig {
		return "", fmt.Errorf("signature is invalid")
	}

	// 4. 계정 상태 확인
	err = account.CheckAccountState(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("account state validation failed : %v", err)
	}

	// 5. 트랜잭션 생성
	tx, jsonRawTransactionStr, err := CreateTransaction(rawTransaction.From, rawTransaction.To, rawTransaction.Signature, rawTransaction.Value, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction : %v", err)
	}

	// 6. Mempool에 저장
	err = AppendToMempool(tx)
	if err != nil {
		return "", fmt.Errorf("failed to append to mempool : %v", err)
	}

	// 7. 반환
	return jsonRawTransactionStr, nil

}
