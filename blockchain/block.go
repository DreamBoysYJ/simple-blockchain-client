package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/account"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"strings"
	"time"

	db "github.com/syndtr/goleveldb/leveldb"
)

type Block struct {
	Number      uint64        `json:"number"`
	Hash        string        `json:"hash"`
	ParentHash  string        `json:"parentHash"`
	Timestamp   uint64        `json:"timestamp"`
	MerkleRoot  string        `json:"merkleRoot"`
	Transaction []Transaction `json:"transaction"`
	Miner       string        `json:"miner"`
}

// 이전 블록을 참고하여, Hash가 현재 블록의 ParentHash와 같은지 비교, timestamp는 더 큰지 비교
func ValidateBlockWithPrevBlock(newBlock *Block) error {
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

	// timestamp 비교
	// 이전 블록보다 시간이 큰지
	if newBlock.Timestamp <= lastBlock.Timestamp {
		return fmt.Errorf("invalid block: timestamp (%d) is not greater than or equal to the previous block's timestamp (%d)", newBlock.Timestamp, lastBlock.Timestamp)
	}

	// 새 블록과 이전 블록의 해시값 비교

	if lastBlock.Hash == newBlock.ParentHash {
		return nil
	} else {
		return fmt.Errorf("invalid block : parent hash : %s , last block hash : %s", newBlock.ParentHash, lastBlock.Hash)
	}

}

// 블록 구조체를 DB에 저장
func StoreBlock(block *Block) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get dbinstance: %v", err))
	}

	// 블록 구조체 json 직렬화
	blockJSON, err := json.Marshal(block)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to marshal block: %v", err))
	}

	// 3. Batch 작업 생성
	batch := new(db.Batch)

	// 4, Batch 작업 추가
	batch.Put([]byte(block.Hash), blockJSON)
	batch.Put([]byte("lastblock"), blockJSON)

	// 5. Batch 실행
	dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("failed to write batch to LevelDB: %w", err)
	}

	fmt.Printf("[BLOCK] Stored, Number : %v, Hash : %v\n", block.Number, block.Hash)
	return nil

}

// 트랜잭션들을 모아 블록 구조체 생성
func CreateNewBlock(transactions []Transaction) *Block {
	// 1. 이전 블록 정보 불러오기
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get dbinstance: %v", err))
		return nil
	}
	lastblockJSON, err := leveldb.GetLastBlock(dbInstance)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to get lastblock: %v", err))
		return nil
	}

	// json 데이터를 블록 구조체로 변환
	var lastBlock Block
	err = json.Unmarshal(lastblockJSON, &lastBlock)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to unmarshal last block : %v", err))
		return nil
	}

	// 2. 트랜잭션 해시 추출
	var transactionHashes []string
	for _, tx := range transactions {
		// 0x 접두사 제거
		txHash := strings.TrimPrefix(tx.Hash, "0x")
		if len(txHash) != 64 {
			utils.PrintError(fmt.Sprintf("Invalid transaction hash: %s", tx.Hash))
			return nil
		}
		transactionHashes = append(transactionHashes, txHash)
	}

	// 3. 머클루트 계산
	merkleRoot, err := BuildMerkleTree(transactionHashes)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to calcualter Merkle Root : %v", err))
		return nil
	}

	// 4. 블록 구조체 생성
	newBlock := &Block{
		Number:      lastBlock.Number + 1,
		ParentHash:  lastBlock.Hash,
		Timestamp:   uint64(time.Now().Unix()),
		Transaction: transactions,
		Miner:       NodeAccount, // 프로그램을 실행하는 노드의 주소
		MerkleRoot:  merkleRoot,
	}

	// 5. 블록 해시 계산
	blockHashData := fmt.Sprintf("%d%s%s%s%d", newBlock.Number, newBlock.ParentHash, merkleRoot, newBlock.Miner, newBlock.Timestamp)
	newBlock.Hash = utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))

	return newBlock
}

// 블록 유효성 검증
func validateReceivedBlock(block *Block) error {
	fmt.Println("[BLOCK] Starting Validation...")

	// 1. 이전 블록 해시 검증
	err := ValidateBlockWithPrevBlock(block)
	if err != nil {
		return fmt.Errorf("parent block hash mismatch : %v", err)
	}

	fmt.Println("[BLOCK] Validated against the Previous block")

	// 2. 현재 블록 해시 검증
	blockHashData := fmt.Sprintf("%d%s%s%s%d", block.Number, block.ParentHash, block.MerkleRoot, block.Miner, block.Timestamp)
	expectedHash := utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))
	if block.Hash != expectedHash {
		// TODO : 상대방에게 알리기
		return fmt.Errorf("invalid block hash : expected : %s, got %s", expectedHash, block.Hash)
	}

	fmt.Println("[BLOCK] Validated the hash of this block")

	// 3. 머클루트 검증
	var transactionHashes []string
	for _, tx := range block.Transaction {
		txHash := strings.TrimPrefix(tx.Hash, "0x") // 0x 제거
		transactionHashes = append(transactionHashes, txHash)
	}
	calculatedMerkleRoot, err := BuildMerkleTree(transactionHashes)
	if err != nil {
		return fmt.Errorf("failed to calculate Merkle root: %v", err)
	}
	if block.MerkleRoot != calculatedMerkleRoot {
		return fmt.Errorf("invalid Merkle root : expected %s, got %s", calculatedMerkleRoot, block.MerkleRoot)
	}

	fmt.Println("[BLOCK] Validated the merkleroot of this block")

	fmt.Println("[BLOCK] Starting validation of transactions in this block...")

	// 4. 블록 내 트랜잭션 검증
	for _, tx := range block.Transaction {

		_, err := ProcessTransactionFromBlock(tx)
		if err != nil {
			return fmt.Errorf("transaction validation failed for tx %s: %v", tx.Hash, err)
		}

	}

	return nil
}

// 블록 Miner에게 보상 지급
func RewardToMiner(minerAddress string) error {

	const rewardAmount = 1000
	rewardBigInt := big.NewInt(rewardAmount)

	// 주소 유효성 검증
	if !account.IsValidAddress(minerAddress) {
		return fmt.Errorf("invalid miner address : %s", minerAddress)
	}

	// Miner 계정 존재하는지 확인
	minerExists, err := account.AccountExists(minerAddress)
	if err != nil {
		return fmt.Errorf("failed to check miner account existence : %v", err)
	}

	// 없는 계정이면 db에 저장
	if !minerExists {
		_, err := account.StoreAccount(minerAddress)
		if err != nil {
			return fmt.Errorf("failed to create miner account : %v", err)
		}
	}

	// 계정 불러오기

	minerAccount, err := account.GetAccount(minerAddress)
	if err != nil {
		return fmt.Errorf("failed to retrieve miner account : %v", err)
	}

	// 잔액 업데이트
	minerAccount.Balance.Add(minerAccount.Balance, rewardBigInt)

	// 업데이트 된 계정 저장
	err = account.UpdateAccount(minerAddress, minerAccount)
	if err != nil {
		return fmt.Errorf("failed to update miner account: %v", err)
	}

	fmt.Printf("[BLOCK CREATOR] Rewarded %d to miner : %s\n", rewardAmount, minerAddress)
	return nil
}
