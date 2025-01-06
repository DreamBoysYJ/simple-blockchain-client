package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	db "github.com/syndtr/goleveldb/leveldb"

	"simple_p2p_client/account"
	"simple_p2p_client/constants"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/mediator"
	"simple_p2p_client/protocol_constants"

	"simple_p2p_client/utils"
	"time"
)

// 제네시스 블록 생성 및 저장
func InitGenesisBlock() error {
	fmt.Println("Initializing Blockchain with Genesis Block...")

	// DB 인스턴스 가져오기
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %v", err)
	}

	// 제네시스 블록 생성
	genesisBlock := Block{
		Number:      1,
		Hash:        "",               // 해시는 아래에서 계산
		ParentHash:  "0x0",            // 제네시스 블록의 부모 해시는 없음
		Timestamp:   0,                // 타임스탬프를 0으로 고정
		MerkleRoot:  "0x0",            // 제네시스 블록에 트랜잭션이 없으므로 Merkle Root는 0x0
		Transaction: []Transaction{},  // 빈 트랜잭션 리스트
		Miner:       "0xGenesisMiner", // 제네시스 블록의 마이너 주소 (예: "0xGenesisMiner")
	}

	// 블록 해시 계산
	blockHashData := fmt.Sprintf("%d%s%s%s%s", genesisBlock.Number, genesisBlock.ParentHash, genesisBlock.MerkleRoot, genesisBlock.Miner, genesisBlock.Timestamp)
	genesisBlock.Hash = utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))

	// 제네시스 블록 직렬화
	blockJSON, err := json.Marshal(genesisBlock)
	if err != nil {
		return fmt.Errorf("failed to serialize genesis block: %v", err)

	}

	// LevelDB에 저장
	batch := new(db.Batch)
	batch.Put([]byte(genesisBlock.Hash), blockJSON)
	batch.Put([]byte("lastblock"), blockJSON)

	err = dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("failed to store genesis block in DB: %v", err)

	}

	fmt.Println("Genesis Block created and stored successfully.")
	return nil
}

// p2p, rpc로부터 블록체인으로 보내는 채널의 메시지를 수신하고 트랜잭션, 블록 검증 및 재전파
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
			messageContent := string(messageBytes[1:]) // 프로토콜 ID 제거 및 트림 처리

			switch messageType {

			// 피어로부터, rpc서버로 부터 트랜잭션을 받았을 때
			case protocol_constants.P2PTransactionMessage:
				fmt.Printf("Raw message content: %q\n", messageContent)
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

				fmt.Printf("Received Block: %+v\n", receivedBlock)

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

				// 4. 트랜잭션 실행
				err = ExecuteTransactions(receivedBlock.Transaction)
				if err != nil {
					fmt.Printf("Failed to execute transaction: %v\n", err)
					// TODO : 트랜잭션이 실패될 경우 블록 저장을 어떻게 롤백할 것인가
				}
				fmt.Printf("Transactions executed : %v\n", receivedBlock.Transaction)

				// 5. 멤풀에서 이미 처리한 트랜잭션 제거
				defaultMempool.CleanMempoolAfterReceiveBlock(receivedBlock.Transaction)
				fmt.Println("Mempool cleaned after processing block")

				// 6. 블록을 채널에 전달 (피어에게 전파)
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

// 일정 주기로 멤풀을 확인하고 블록을 생성하는 로직
func StartBlockCreator() {
	ticker := time.NewTicker(constants.BlockCreationInterval)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("START Trying to create Block!")
		defaultMempool.SyncFutureToPending()
		fmt.Println("Sync complete, pending:", defaultMempool.pending)

		defaultMempool.mu.Lock()
		totalTransactions := 0
		for _, accountTxs := range defaultMempool.pending {
			totalTransactions += len(accountTxs)
		}

		fmt.Println("1단계요")

		if totalTransactions < constants.TransactionsPerBlock {
			defaultMempool.mu.Unlock()
			fmt.Println("Not enough transactions")
			continue
		}

		fmt.Println("2단계요")

		blockTxs := defaultMempool.ExtractTransactionsForBlock(constants.TransactionsPerBlock)
		fmt.Printf("Transactions for block: %v\n", blockTxs)
		defaultMempool.mu.Unlock()
		fmt.Println("3단계요")

		if len(blockTxs) == 0 {
			fmt.Println("No transactions extracted for block")
			continue
		}

		newBlock := CreateNewBlock(blockTxs)
		if newBlock == nil {
			fmt.Println("New block creation failed. Skipping block storage.")
			continue
		}
		fmt.Printf("New block created: %v\n", newBlock)

		// JSON 직렬화 후 데이터 확인
		blockJSON, err := json.Marshal(newBlock)
		if err != nil {
			fmt.Printf("Failed to serialize block to JSON: %v\n", err)
			continue
		}
		fmt.Printf("Serialized Block JSON: %s\n", blockJSON)

		// TODO : 생성했으면 그 다음엔?
		// 1. 블록 저장
		err = StoreBlock(newBlock)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Failed to store block: %v", err))
			continue
		}

		// 2. 블록 안 트랜잭션 실행
		err = ExecuteTransactions(newBlock.Transaction)
		if err != nil {
			fmt.Printf("Failed to execute transaction: %v\n", err)
			// TODO : 트랜잭션이 실패될 경우 블록 저장을 어떻게 롤백할 것인가
		}
		fmt.Printf("Transactions executed : %v\n", newBlock.Transaction)

		// 3. 블록을 채널에 전달 (피어에게 전파)
		blockJSON, err = json.Marshal(newBlock)
		if err != nil {
			fmt.Printf("Failed to serialize block to JSON : %v\n", err)
			continue
		}
		mediatorInstance := mediator.GetMediatorInstance()
		message := fmt.Sprintf("%c%s", protocol_constants.P2PBlockMessage, string(blockJSON))
		fmt.Printf("Message sent to channel: %s\n", message)

		mediatorInstance.BlockchainToP2P <- message
	}

}

// 트랜잭션 실행(db 업데이트)
func ExecuteTransactions(transactions []Transaction) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("failed to get dbinstance : %v", err)
	}

	batch := new(db.Batch)

	for _, tx := range transactions {
		// 1. From 가져오기
		fromAccount, err := account.GetAccount(tx.From)
		if err != nil {
			return fmt.Errorf("failed to retrieve from account : %s : %v", tx.From, err)
		}

		// 2. To 가져오기 (없는 경우 생성)
		toAccount, err := account.GetAccount(tx.To)
		if err != nil {
			if err == db.ErrNotFound { // 없는 경우 생성
				_, createErr := account.StoreAccount(tx.To)
				if createErr != nil {
					return fmt.Errorf("failed to create to account %s: %v", tx.To, err)
				}
				toAccount, _ = account.GetAccount(tx.To)
			} else {
				return fmt.Errorf("failed to retrieve to account %s:%v", tx.To, err)
			}
		}

		// 3. 잔액 검증 및 트랜잭션 실행
		txValue := tx.Value
		if fromAccount.Balance.Cmp(txValue) < 0 {
			return fmt.Errorf("insufficient balance in from account %s", tx.From)
		}

		fromAccount.Balance.Sub(fromAccount.Balance, txValue)
		toAccount.Balance.Add(toAccount.Balance, txValue)
		fromAccount.Nonce++

		// 4. 계정 데이터 저장 (배치 작업 추가)
		fromAccountKey := append([]byte("account:"), []byte(tx.From)...)
		toAccountKey := append([]byte("account:"), []byte(tx.To)...)

		fromAccountJSON, err := json.Marshal(fromAccount)
		if err != nil {
			return fmt.Errorf("failed to serialize from account: %v", err)
		}

		toAccountJSON, err := json.Marshal(toAccount)
		if err != nil {
			return fmt.Errorf("failed to serialize to account: %v", err)
		}

		batch.Put(fromAccountKey, fromAccountJSON)
		batch.Put(toAccountKey, toAccountJSON)

	}

	// 5. 배치 실행
	err = dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("failed to execute batch write : %v", err)
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

// func CheckTxInMempool(txHash string) bool {
// 	if _, exists := Mempool[txHash]; exists {
// 		return true
// 	}
// 	return false
// }

// func AppendToMempool(tx Transaction) error {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	// 중복 tx인지 확인
// 	if _, exists := Mempool[tx.Hash]; exists {
// 		// 중복 트랜잭션
// 		return fmt.Errorf("tx already in mempool")

// 	}

// 	// 추가
// 	Mempool[tx.Hash] = tx
// 	fmt.Println("트랜잭션 멤풀에 들어갔따!")

// 	return nil

// }

// Peer로 부터 트랜잭션을 수신했을 때 검증
// Deprecated
// func ValidateTransaction(message string) (string, error) {
// 	// 1. RawTransaction 구조체 변환
// 	var rawTransaction RawTransaction
// 	err := json.Unmarshal([]byte(message), &rawTransaction)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to parse raw transaction : %v", err)

// 	}

// 	// 2. 트랜잭션 필드 검증
// 	err = ValidateTransactionFields(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Signature, rawTransaction.Nonce)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to validate transaction fields : %w", err)
// 	}

// 	// 3. 서명 검증
// 	txMessage := fmt.Sprintf("%s%s%s%d", rawTransaction.From, rawTransaction.To, rawTransaction.Value, rawTransaction.Nonce)
// 	txMessageHash := utils.Keccak256([]byte(txMessage))
// 	hexSignature, err := hex.DecodeString(rawTransaction.Signature)
// 	if err != nil {
// 		return "", fmt.Errorf("invalid signature format")
// 	}
// 	isValidSig, err := VerifySignature(txMessageHash, hexSignature, rawTransaction.From)

// 	if err != nil {
// 		return "", fmt.Errorf("signature verification failed : %v", err)
// 	}
// 	if !isValidSig {
// 		return "", fmt.Errorf("signature is invalid")
// 	}

// 	// 4. 계정 상태 확인
// 	err = account.CheckAccountState(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
// 	if err != nil {
// 		return "", fmt.Errorf("account state validation failed : %v", err)
// 	}

// 	// 5. 트랜잭션 생성
// 	tx, jsonRawTransactionStr, err := CreateTransaction(rawTransaction.From, rawTransaction.To, rawTransaction.Signature, rawTransaction.Value, rawTransaction.Nonce)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create transaction : %v", err)
// 	}

// 	// 6. Mempool에 저장
// 	err = defaultMempool.AddTransaction(tx, rawTransaction.Nonce)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to append to mempool : %v", err)
// 	}

// 	// 7. 반환
// 	return jsonRawTransactionStr, nil

// }
