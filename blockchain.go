package main

import (
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/sha3"
)

// 트랜잭션 구조체
type Transaction struct {
	Hash  string
	From  string
	To    string
	Value *big.Int
	Nonce uint64
}

// 블록 구조체
type Block struct {
	Number      uint64
	Hash        string
	ParentHash  string
	Timestamp   time.Time
	Transaction []Transaction
	Miner       string
}

func createTx() {
	data := []byte("hello")

	hash := keccak256(data)
	// %x : 16진수
	fmt.Printf("Keccack 해시값 ::: %x\n ", hash)
}

func keccak256(data []byte) []byte {
	// hash 함수 생성
	hash := sha3.NewLegacyKeccak256()
	// hash 함수에 데이터 입력
	hash.Write(data)
	// 해시값을 계산하여 반환
	return hash.Sum(nil)

}
