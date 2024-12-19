package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/account"
	"simple_p2p_client/mediator"
	"simple_p2p_client/protocol_constants"

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
	MerkleRoot  string        `json:"merkleRoot"`
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
		return fmt.Errorf("invalid block : parent hash : %s , last block hash : %s", newBlock.ParentHash, lastBlock.Hash)
	}
}

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

func StartBlockchainProcessor() {
	mediatorInstance := mediator.GetMediatorInstance()

	go func() {
		for message := range mediatorInstance.P2PToBlockchain {

			if len(message) == 0 {
				fmt.Println("Received empty message, skipping...")
				continue
			}

			// 첫바이트로 메시지 타입 판별
			messageBytes := []byte(message)
			messageType := messageBytes[0]
			messageContent := string(messageBytes[1:])

			switch messageType {

			// 피어로부터 트랜잭션을 받았을 때
			case protocol_constants.P2PTransactionMessage:
				fmt.Printf("Processing transaction : %s\n", messageContent)
				// 트랜잭션 처리
				processedMessage, err := ProcessTransaction(messageContent)
				if err != nil {
					fmt.Printf("Transaction validation failed : %v\n", err)
					continue
				}
				// Blockchain => p2p 전달
				mediatorInstance.BlockchainToP2P <- fmt.Sprintf("%c%s", 0x01, processedMessage)

			// 블록을 다른 피어로부터 받았을 때
			case protocol_constants.P2PBlockMessage:
				fmt.Printf("Processing block : %s\n", messageContent)
				// TODO : 블록 검증

				// 1. json -> Block 구조체 변환
				var receivedBlock Block
				err := json.Unmarshal([]byte(messageContent), &receivedBlock)
				if err != nil {
					fmt.Printf("Failed to parse block : %v\n", err)
					continue
				}

				// 2. 블록 검증
				err = validateReceivedBlock(&receivedBlock)
				if err != nil {
					fmt.Printf("Block validation failed : %v\n", err)
					continue
				}

				// 3. 블록 저장
				err = StoreBlock(&receivedBlock)
				if err != nil {
					fmt.Printf("Failed to store block :%v\n", err)
					continue
				}

				fmt.Printf("Block validated and stored : %s\n", receivedBlock.Hash)

				// 4. 블록을 채널에 전달
				blockJSON, err := json.Marshal(receivedBlock)
				if err != nil {
					fmt.Printf("Failed to serialize block to JSON : %v\n", err)
					continue
				}
				mediatorInstance := mediator.GetMediatorInstance()
				message := fmt.Sprintf("%c%s", protocol_constants.P2PBlockMessage, string(blockJSON))
				mediatorInstance.P2PToBlockchain <- message

			default: // 알수 없는 메시지 타입
				fmt.Printf("message recevied : %s\n", messageContent)
			}

		}
	}()
}

// ProcessTransaction handles validation and storage of a transaction
func ProcessTransaction(rawTransactionMessage string) (string, error) {
	// 1. RawTransaction 구조체로 변환
	var rawTransaction RawTransaction
	err := json.Unmarshal([]byte(rawTransactionMessage), &rawTransaction)
	if err != nil {
		return "", fmt.Errorf("failed to parse raw transaction: %v", err)
	}

	// 2. 트랜잭션 필드 검증
	err = ValidateTransactionFields(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Signature, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction field validation failed: %v", err)
	}

	// 3. 서명 검증
	message := fmt.Sprintf("%s%s%s%d", rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
	messageHash := utils.Keccak256([]byte(message))
	decodedSignature, err := hex.DecodeString(rawTransaction.Signature)
	if err != nil {
		return "", fmt.Errorf("invalid signature format: %v", err)
	}
	isValidSig, err := VerifySignature(messageHash, decodedSignature, rawTransaction.From)
	if err != nil || !isValidSig {
		return "", fmt.Errorf("signature verification failed: %v", err)
	}

	// 4. 계정 상태 확인
	err = account.CheckAccountState(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("account state validation failed: %v", err)
	}

	// 5. 트랜잭션 생성
	tx, jsonRawTransactionStr, err := CreateTransaction(rawTransaction.From, rawTransaction.To, rawTransaction.Signature, rawTransaction.Value, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// 6. Mempool에 저장
	err = AppendToMempool(tx)
	if err != nil {
		return "", fmt.Errorf("failed to append transaction to mempool: %v", err)
	}

	// 7. 반환
	return jsonRawTransactionStr, nil
}

// Get Merkle root of transactions
func BuildMerkleTree(transactionHashes []string) (string, error) {
	// 1. 트랜잭션 해시 배열이 비어있는지 확인
	if len(transactionHashes) == 0 {
		return "", fmt.Errorf("no transactions to build Merkle Tree")
	}

	// 2. 트랜잭션 해시를 바이트 배열로 변환
	hashes := make([][]byte, len(transactionHashes))
	for i, txHash := range transactionHashes {
		hash, err := hex.DecodeString(txHash)
		if err != nil {
			return "", fmt.Errorf("invalid transaction hash format: %s", txHash)
		}
		hashes[i] = hash
	}

	// 3. 머클 트리 생성
	for len(hashes) > 1 {
		// 홀수일 경우 마지막 해시를 복제
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		// 부모 노드 배열 생성
		var parentHashes [][]byte
		for i := 0; i < len(hashes); i += 2 {
			// 두 개의 해시를 결합하여 새로운 해시 생성
			combined := append(hashes[i], hashes[i+1]...)
			newHash := utils.Keccak256(combined) // Keccak256 사용
			parentHashes = append(parentHashes, newHash)
		}

		// 부모 해시로 대체
		hashes = parentHashes
	}

	// 4. 루트 해시 반환
	merkleRoot := utils.BytesToHex(hashes[0])
	return merkleRoot, nil
}

// Create block
func StartBlockCreator() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		if len(Mempool) >= 10 {
			transactions := extractTransactionFromMemepool(10)
			newBlock := CreateNewBlock(transactions)

			// 블록 json 변환
			blockJSON, err := json.Marshal(newBlock)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Failed to serialize block to JSON : %v", err))
				mu.Unlock()
				continue
			}

			// 블록을 mediator 채널로 전송

			mediatorInstance := mediator.GetMediatorInstance()
			message := fmt.Sprintf("%c%s", protocol_constants.P2PBlockMessage, string(blockJSON))
			mediatorInstance.P2PToBlockchain <- message
		}
		mu.Unlock()
	}
}

func extractTransactionFromMemepool(count int) []Transaction {
	transactions := make([]Transaction, 0, count)
	i := 0
	for _, tx := range Mempool {
		if i >= count {
			break
		}
		transactions = append(transactions, tx)
		i++
	}
	return transactions
}

func CreateNewBlock(transactions []Transaction) *Block {
	// 1. 이전 블록 정보 불러오기
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

	// 2. 트랜잭션 해시 추출
	var transactionHashes []string
	for _, tx := range transactions {
		transactionHashes = append(transactionHashes, tx.Hash)
	}

	// 3. 머클루트 계산
	merkleRoot, err := BuildMerkleTree(transactionHashes)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to calcualter Merkle Root : %v", err))
	}

	// 4. 블록 구조체 생성
	newBlock := &Block{
		Number:      lastBlock.Number + 1,
		ParentHash:  lastBlock.Hash,
		Timestamp:   time.Now(),
		Transaction: transactions,
		Miner:       "0x영주", // TODO : 노드를 실행할 때 이 노드를 위한 계정 생성해두기 or 기존 불러오기
		MerkleRoot:  merkleRoot,
	}

	// 5. 블록 해시 계산
	blockHashData := fmt.Sprintf("%d%s%s%s%s", newBlock.Number, newBlock.ParentHash, merkleRoot, newBlock.Miner, newBlock.Timestamp)
	newBlock.Hash = utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))

	return newBlock
}

func validateReceivedBlock(block *Block) error {
	// 1. 이전 블록 해시 검증
	err := ValidateBlock(block)
	if err != nil {
		return fmt.Errorf("parent block hash mismatch : %v", err)
	}

	// 2. 현재 블록 해시 검증
	blockHashData := fmt.Sprintf("%d%s%s%s%s", block.Number, block.ParentHash, block.MerkleRoot, block.Miner, block.Miner)
	expectedHash := utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))
	if block.Hash != expectedHash {
		// TODO : 상대방에게 알리기
		return fmt.Errorf("invalid block hash : expected : %s, got %s", expectedHash, block.Hash)
	}

	// 3. 머클루트 검증
	var transactionHashes []string
	for _, tx := range block.Transaction {
		transactionHashes = append(transactionHashes, tx.Hash)
	}
	calculatedMerkleRoot, err := BuildMerkleTree(transactionHashes)
	if err != nil {
		return fmt.Errorf("failed to calculate Merkle root: %v", err)
	}
	if block.MerkleRoot != calculatedMerkleRoot {
		return fmt.Errorf("invalid Merkle root : expected %s, got %s", calculatedMerkleRoot, block.MerkleRoot)
	}

	// 4. 블록 내 트랜잭션 검증
	for _, tx := range block.Transaction {

		_, err := ProcessTransactionFromBlock(tx)
		if err != nil {
			return fmt.Errorf("transaction validation failed for tx %s: %v", tx.Hash, err)
		}
	}

	return nil
}

func ProcessTransactionFromBlock(tx Transaction) (string, error) {
	// 1. 트랜잭션 필드 검증
	err := ValidateTransactionFields(tx.From, tx.To, tx.Value.String(), tx.Signature, tx.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction field validation failed : %v", err)
	}

	// 2. 서명 검증
	txMessage := fmt.Sprintf("%s%s%s%d", tx.From, tx.To, tx.Value.String(), tx.Nonce)
	txMessageHash := utils.Keccak256([]byte(txMessage))
	decodedSignature, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return "", fmt.Errorf("invalid transaction signature format : %v", err)
	}
	isValidSig, err := VerifySignature(txMessageHash, decodedSignature, tx.From)
	if err != nil || !isValidSig {
		return "", fmt.Errorf("transaction signature validation failed : %v", err)
	}

	// 3. 계정 상태 확인
	err = account.CheckAccountState(tx.From, tx.To, tx.Value.String(), tx.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction state validation failed : %v", err)
	}

	// 4. 성공적으로 검증된 트랜잭션 반환
	return tx.Hash, nil
}
