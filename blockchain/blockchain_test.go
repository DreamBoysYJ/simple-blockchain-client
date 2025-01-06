package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"testing"
)

func TestSyncFutureToPending(t *testing.T) {
	// 초기 멤풀 상태 설정
	mp := &Mempool{
		pending: map[string]map[uint64]Transaction{
			"account1": {
				1: {Nonce: 1},
				2: {Nonce: 2},
				3: {Nonce: 3},
			},
		},
		future: map[string]map[uint64]Transaction{
			"account1": {
				4: {Nonce: 4},
				5: {Nonce: 5},
				6: {Nonce: 6},
				8: {Nonce: 8}, // 연속되지 않은 Nonce
			},
		},
	}

	// SyncFutureToPending 호출
	mp.SyncFutureToPending()

	// 결과 확인
	expectedPending := map[uint64]Transaction{
		1: {Nonce: 1},
		2: {Nonce: 2},
		3: {Nonce: 3},
		4: {Nonce: 4},
		5: {Nonce: 5},
		6: {Nonce: 6},
	}
	expectedFuture := map[uint64]Transaction{
		8: {Nonce: 8},
	}

	// 검증: Pending 확인
	if len(mp.pending["account1"]) != len(expectedPending) {
		t.Errorf("Pending size mismatch: expected %d, got %d", len(expectedPending), len(mp.pending["account1"]))
	}
	for nonce, tx := range expectedPending {
		if _, exists := mp.pending["account1"][nonce]; !exists {
			t.Errorf("Pending missing transaction with nonce %d", nonce)
		}
		if mp.pending["account1"][nonce] != tx {
			t.Errorf("Pending transaction mismatch for nonce %d: expected %+v, got %+v", nonce, tx, mp.pending["account1"][nonce])
		}
	}

	// 검증: Future 확인
	if len(mp.future["account1"]) != len(expectedFuture) {
		t.Errorf("Future size mismatch: expected %d, got %d", len(expectedFuture), len(mp.future["account1"]))
	}
	for nonce, tx := range expectedFuture {
		if _, exists := mp.future["account1"][nonce]; !exists {
			t.Errorf("Future missing transaction with nonce %d", nonce)
		}
		if mp.future["account1"][nonce] != tx {
			t.Errorf("Future transaction mismatch for nonce %d: expected %+v, got %+v", nonce, tx, mp.future["account1"][nonce])
		}
	}

	// 테스트 결과 출력 (선택 사항)
	fmt.Println("Pending after Sync:", mp.pending["account1"])
	fmt.Println("Future after Sync:", mp.future["account1"])
}

func TestExtractTransactionsForBlock(t *testing.T) {
	// Initialize mempool
	mp := &Mempool{
		pending: map[string]map[uint64]Transaction{
			"A": {
				1: {Nonce: 1, From: "A"},
				2: {Nonce: 2, From: "A"},
				3: {Nonce: 3, From: "A"},
			},
			"B": {
				4: {Nonce: 4, From: "B"},
				5: {Nonce: 5, From: "B"},
				6: {Nonce: 6, From: "B"},
			},
			"C": {
				1: {Nonce: 1, From: "C"},
				2: {Nonce: 2, From: "C"},
				3: {Nonce: 3, From: "C"},
				4: {Nonce: 4, From: "C"},
				5: {Nonce: 5, From: "C"},
			},
		},
	}

	// Extract transactions for block
	blockTxs := mp.ExtractTransactionsForBlock(10)

	// Print transactions for block
	fmt.Println("Block Transactions:")
	for _, tx := range blockTxs {
		fmt.Printf("Account: %s, Nonce: %d\n", tx.From, tx.Nonce)
	}

	// Check pending state after extraction
	fmt.Println("Remaining Pending Transactions:")
	for account, accountTxs := range mp.pending {
		fmt.Printf("Account: %s\n", account)
		for nonce, tx := range accountTxs {
			fmt.Printf("  Nonce: %d, Transaction: %+v\n", nonce, tx)
		}
	}

	// Validate that 10 transactions were extracted
	if len(blockTxs) != 10 {
		t.Errorf("Expected 10 transactions, got %d", len(blockTxs))
	}
}

func TestBuildMerkleTree(t *testing.T) {
	// 유틸리티 함수: Keccak256 해시 계산
	hash := func(input string) string {
		return utils.BytesToHex(utils.Keccak256([]byte(input)))
	}

	// 유틸리티 함수: 두 개의 바이트 배열을 결합하여 해시 생성
	combineHashes := func(hash1, hash2 string) string {
		h1, _ := hex.DecodeString(hash1)
		h2, _ := hex.DecodeString(hash2)
		combined := append(h1, h2...)
		return utils.BytesToHex(utils.Keccak256(combined))
	}

	tests := []struct {
		name               string
		transactionHashes  []string
		expectedMerkleRoot string
		expectError        bool
	}{
		// 이전 테스트 케이스들 생략...

		{
			name: "Five transactions (odd number)",
			transactionHashes: []string{
				hash("tx1"),
				hash("tx2"),
				hash("tx3"),
				hash("tx4"),
				hash("tx5"),
			},
			// 루트 해시는 다음과 같이 계산:
			// Level 1: hash(tx1+tx2), hash(tx3+tx4), hash(tx5+tx5)
			// Level 2: hash(hash(tx1+tx2)+hash(tx3+tx4)), hash(hash(tx5+tx5)+hash(tx5+tx5))
			// Merkle Root: hash(hash(hash(tx1+tx2)+hash(tx3+tx4)) + hash(hash(tx5+tx5)+hash(tx5+tx5)))
			expectedMerkleRoot: combineHashes(
				combineHashes(
					combineHashes(hash("tx1"), hash("tx2")),
					combineHashes(hash("tx3"), hash("tx4")),
				),
				combineHashes(
					combineHashes(hash("tx5"), hash("tx5")),
					combineHashes(hash("tx5"), hash("tx5")),
				),
			),
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			merkleRoot, err := BuildMerkleTree(test.transactionHashes)
			if test.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect error but got: %v", err)
				}
				if merkleRoot != test.expectedMerkleRoot {
					t.Errorf("expected merkle root: %s, got: %s", test.expectedMerkleRoot, merkleRoot)
				}
			}
		})
	}
}

func TestCreateNewBlock(t *testing.T) {
	transactions := []Transaction{
		{Hash: "0x" + utils.BytesToHex(utils.Keccak256([]byte("tx1")))},
		{Hash: "0x" + utils.BytesToHex(utils.Keccak256([]byte("tx2")))},
		{Hash: "0x" + utils.BytesToHex(utils.Keccak256([]byte("tx3")))},
		{Hash: "0x" + utils.BytesToHex(utils.Keccak256([]byte("tx4")))},
		{Hash: "0x" + utils.BytesToHex(utils.Keccak256([]byte("tx5")))},
	}

	block := CreateNewBlock(transactions)
	if block == nil {
		t.Errorf("Failed to create new block")
	}

	t.Logf("New block created: %+v", block)
}

func TestCreateNewBlockTwo(t *testing.T) {
	// 테스트를 위한 유틸리티 함수: Keccak256 해시 계산
	hash := func(input string) string {
		return utils.BytesToHex(utils.Keccak256([]byte(input)))
	}

	// 이전 블록 설정 (제네시스 블록)
	genesisBlock := Block{
		Number:      1,
		Hash:        hash("genesis_block"),
		ParentHash:  "0x0",
		Timestamp:   0,
		MerkleRoot:  "0x0",
		Transaction: []Transaction{},
		Miner:       "0xGenesisMiner",
	}

	// LevelDB 초기화
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		t.Fatalf("Failed to initialize LevelDB: %v", err)
	}

	// 제네시스 블록 저장
	genesisBlockJSON, err := json.Marshal(genesisBlock)
	if err != nil {
		t.Fatalf("Failed to marshal genesis block: %v", err)
	}
	err = dbInstance.Put([]byte("lastblock"), genesisBlockJSON, nil)
	if err != nil {
		t.Fatalf("Failed to store genesis block: %v", err)
	}

	// 테스트 시나리오
	tests := []struct {
		name              string
		transactions      []Transaction
		expectValidBlock  bool
		expectedErrorText string
	}{
		{
			name: "Valid transactions",
			transactions: []Transaction{
				{Hash: hash("tx1")},
				{Hash: hash("tx2")},
			},
			expectValidBlock:  true,
			expectedErrorText: "",
		},
		{
			name:              "Empty transactions",
			transactions:      []Transaction{},
			expectValidBlock:  true, // 머클루트가 "0x0"으로 처리될 것으로 예상
			expectedErrorText: "",
		},
		{
			name: "Invalid transaction hash format",
			transactions: []Transaction{
				{Hash: "invalid_hash"},
			},
			expectValidBlock:  false,
			expectedErrorText: "Validation failed for created block",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 블록 생성
			newBlock := CreateNewBlock(test.transactions)

			// 결과 검증
			if test.expectValidBlock {
				if newBlock == nil {
					t.Errorf("Expected valid block but got nil")
				} else {
					// 블록 필드 검증
					if newBlock.Number != genesisBlock.Number+1 {
						t.Errorf("Expected block number %d, got %d", genesisBlock.Number+1, newBlock.Number)
					}
					if newBlock.ParentHash != genesisBlock.Hash {
						t.Errorf("Expected parent hash %s, got %s", genesisBlock.Hash, newBlock.ParentHash)
					}
					fmt.Printf("Created Block: %+v\n", newBlock)
				}
			} else {
				if newBlock != nil {
					t.Errorf("Expected block creation to fail, but got: %+v", newBlock)
				}
			}
		})
	}
}
