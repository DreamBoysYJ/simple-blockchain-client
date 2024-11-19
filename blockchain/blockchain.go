package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"time"

	db "github.com/syndtr/goleveldb/leveldb"

	"golang.org/x/crypto/sha3"
)

// 트랜잭션 구조체

type RawTransaction struct {
	From  string   `json:"from"`
	To    string   `json:"to"`
	Value *big.Int `json:"value"`
	Nonce uint64   `json:"nonce"`
}

type Transaction struct {
	Hash  string   `json:"hash"`
	From  string   `json:"from"`
	To    string   `json:"to"`
	Value *big.Int `json:"value"`
	Nonce uint64   `json:"nonce"`
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

func CreateTx() {
	data := []byte("hello")

	hash := Keccak256(data)
	// %x : 16진수
	fmt.Printf("Keccack 해시값 ::: %x\n ", hash)
}

func Keccak256(data []byte) []byte {
	// hash 함수 생성
	hash := sha3.NewLegacyKeccak256()
	// hash 함수에 데이터 입력
	hash.Write(data)
	// 해시값을 계산하여 반환
	return hash.Sum(nil)

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
		return fmt.Errorf("Failed to write batch to LevelDB: %w", err)
	}
	return nil

}

var Mempool []Transaction
