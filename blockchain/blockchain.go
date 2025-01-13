package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"

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
	fmt.Println("[BLOCK] Initializing Blockchain with Genesis Block...")

	// DB 인스턴스 가져오기
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %v", err)
	}

	// 이미 블록이 있는지 확인
	lastBlockData, err := dbInstance.Get([]byte("lastblock"), nil)
	if err == nil && len(lastBlockData) > 0 {
		// lastblock이 있다면 패스
		fmt.Println("[BLOCK] already initialized, Skipping genesis block creation...")
		return nil
	}

	// 제네시스 블록 생성
	genesisBlock := Block{
		Number:      1,
		Hash:        "",                                           // 아래에서 계산
		ParentHash:  "0x0",                                        // 부모 해시는 없음
		Timestamp:   0,                                            // 0으로 고정
		MerkleRoot:  "0x0",                                        // 트랜잭션이 없으므로 Merkle Root는 0x0
		Transaction: []Transaction{},                              // 빈 트랜잭션 리스트
		Miner:       "0xde589C867174C349d00e9b582867aF5c13A74679", // Test 주소 생성 및 코인 입금
	}

	// 블록 해시 계산
	blockHashData := fmt.Sprintf("%d%s%s%s%d", genesisBlock.Number, genesisBlock.ParentHash, genesisBlock.MerkleRoot, genesisBlock.Miner, genesisBlock.Timestamp)
	genesisBlock.Hash = utils.BytesToHex(utils.Keccak256([]byte(blockHashData)))

	// Genesis Miner 계정 생성 및 초기 코인 입금
	initialBalance := big.NewInt(10000)
	success, err := account.StoreAccountForGenesisMiner(genesisBlock.Miner, initialBalance)
	if err != nil || !success {
		return fmt.Errorf("failed to init miner account: %v", err)
	}
	fmt.Printf("[BLOCK] Genesisblock miner initialized, address : %s, balance : %s", genesisBlock.Miner, initialBalance.String())
	fmt.Println("")

	// 제네시스 블록 LevelDB에 저장
	blockJSON, err := json.Marshal(genesisBlock)
	if err != nil {
		return fmt.Errorf("failed to serialize genesis block: %v", err)

	}
	batch := new(db.Batch)
	batch.Put([]byte(genesisBlock.Hash), blockJSON)
	batch.Put([]byte("lastblock"), blockJSON)

	err = dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("failed to store genesis block in DB: %v", err)

	}

	fmt.Printf("[BLOCK] Genesis block created, hash: %s\n", genesisBlock.Hash)

	return nil
}

// p2p, rpc에서 blockchain으로 보내는 채널의 메시지를 수신해 트랜잭션, 블록을 검증하고 전파
func StartBlockchainProcessor() {
	mediatorInstance := mediator.GetMediatorInstance()

	go func() {
		for message := range mediatorInstance.P2PToBlockchain {

			if len(message) == 0 {
				fmt.Println("Received empty message, skipping...")
				continue
			}

			// 첫 바이트로 메시지 타입 판별
			messageBytes := []byte(message)
			messageType := messageBytes[0]
			messageContent := string(messageBytes[1:]) // 프로토콜 ID 제거 및 트림 처리

			switch messageType {

			// 0x01 : 피어로부터, rpc서버로 부터 트랜잭션을 받았을 때
			case protocol_constants.P2PTransactionMessage:
				fmt.Printf("[TX] Received Transaction, Processing... : %s\n", messageContent)
				// 트랜잭션 처리 (검증 및 멤풀 저장)
				processedMessage, err := ProcessTransaction(messageContent)
				if err != nil {
					fmt.Printf("[TX] Validation failed : %v\n", err)
					continue
				}
				
				// Blockchain => p2p 전달
				mediatorInstance.BlockchainToP2P <- fmt.Sprintf("%c%s", 0x01, processedMessage)

			// 0x02 : 블록을 다른 피어로부터 받았을 때
			case protocol_constants.P2PBlockMessage:
				fmt.Printf("[BLOCK] Recevied Block, Processing... : %s\n", messageContent)

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
					fmt.Printf("[BLOCK] Validation failed : %v\n", err)
					continue
				}

				fmt.Println("[BLOCK] Validation completes!")

				// Miner 보상 지금
				err = RewardToMiner(receivedBlock.Miner)
				if err != nil {
					fmt.Printf("Failed to reward miner : %v\n", err)
				}

				// 3. 블록 저장
				err = StoreBlock(&receivedBlock)
				if err != nil {
					fmt.Printf("Failed to store block :%v\n", err)
					continue
				}
				fmt.Printf("[BLOCK] Validated and Stored : %s\n", receivedBlock.Hash)

				// 4. 트랜잭션 실행
				err = ExecuteTransactions(receivedBlock.Transaction)
				if err != nil {
					fmt.Printf("Failed to execute transaction: %v\n", err)
					// TODO : 트랜잭션이 실패될 경우 블록 저장을 어떻게 롤백할 것인가
				}
				fmt.Println("[TX] Execution transactions in this block completed")

				// 5. 멤풀에서 이미 처리한 트랜잭션 제거
				defaultMempool.CleanMempoolAfterReceiveBlock(receivedBlock.Transaction)
				fmt.Println("[Mempool] Cleaned after processing block")

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
				fmt.Printf("[MESSAGE] Recevied from peer : %s\n", messageContent)
			}

		}
	}()
}

// 일정 주기로 멤풀을 확인하고 블록 생성 시도
func StartBlockCreator() {
	// 실행 주기는 constants에서 설정 가능
	ticker := time.NewTicker(constants.BlockCreationInterval)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("[BLOCK CREATOR] Start trying to create block")
		defaultMempool.SyncFutureToPending()
		fmt.Println("[Mempool] Sync complete")

		defaultMempool.mu.Lock()
		totalTransactions := 0
		for _, accountTxs := range defaultMempool.pending {
			totalTransactions += len(accountTxs)
		}

		if totalTransactions < constants.TransactionsPerBlock {
			defaultMempool.mu.Unlock()
			fmt.Println("[BLOCK CREATOR] Cancelled, Not enough transactions")
			continue
		}

		blockTxs := defaultMempool.ExtractTransactionsForBlock(constants.TransactionsPerBlock)
		fmt.Printf("[BLOCK CREATOR] Transactions extracted for block : %v\n", blockTxs)
		defaultMempool.mu.Unlock()

		if len(blockTxs) == 0 {
			fmt.Println("No transactions extracted for block")
			continue
		}

		newBlock := CreateNewBlock(blockTxs)
		if newBlock == nil {
			fmt.Println("New block creation failed. Skipping block storage.")
			continue
		}
		fmt.Printf("[BLOCK CREATOR] New Block created: %v\n", newBlock)

		// JSON 직렬화 후 데이터 확인
		blockJSON, err := json.Marshal(newBlock)
		if err != nil {
			fmt.Printf("Failed to serialize block to JSON: %v\n", err)
			continue
		}

		// 블록 저장
		err = StoreBlock(newBlock)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Failed to store block: %v", err))
			continue
		}
		fmt.Printf("[BLOCK CREATOR] New Block stored: %v\n", newBlock)

		// 블록 안 트랜잭션 실행
		fmt.Printf("[BLOCK CREATOR] Transaction execution begins...\n")

		err = ExecuteTransactions(newBlock.Transaction)
		if err != nil {
			fmt.Printf("Failed to execute transaction: %v\n", err)
			// TODO : 트랜잭션이 실패될 경우 블록 저장을 어떻게 롤백할 것인가
		}
		fmt.Printf("[BLOCK CREATOR] Transactions executed : %v\n", newBlock.Transaction)

		// Miner 보상 지금
		err = RewardToMiner(newBlock.Miner)
		if err != nil {
			fmt.Printf("Failed to reward miner : %v\n", err)
		}

		// 블록을 채널에 전달 (피어에게 전파)

		mediatorInstance := mediator.GetMediatorInstance()
		message := fmt.Sprintf("%c%s", protocol_constants.P2PBlockMessage, string(blockJSON))
		fmt.Printf("[BLOCK CREATOR] Forwarding to broadcast block to peers: %s\n", message)

		mediatorInstance.BlockchainToP2P <- message
	}

}
