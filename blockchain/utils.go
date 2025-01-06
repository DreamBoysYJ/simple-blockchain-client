package blockchain

import (
	"encoding/hex"
	"fmt"
	"simple_p2p_client/utils"
)

// 트랜잭션 해시들로 머클루트 반환
func BuildMerkleTree(transactionHashes []string) (string, error) {
	fmt.Printf("Transaction Hashes in BuildMerkleTree: %v\n", transactionHashes)

	if len(transactionHashes) == 0 {
		return "", fmt.Errorf("no transactions to build Merkle Tree")
	}

	// 트랜잭션 해시를 바이트 배열로 변환
	hashes := make([][]byte, len(transactionHashes))
	for i, txHash := range transactionHashes {
		if len(txHash) != 64 {
			return "", fmt.Errorf("invalid transaction hash length: %s (index %d)", txHash, i)
		}
		hash, err := hex.DecodeString(txHash)
		if err != nil {
			return "", fmt.Errorf("invalid transaction hash format: %s", txHash)
		}
		hashes[i] = hash
	}

	// 머클 트리 생성
	for level := 0; len(hashes) > 1; level++ {
		fmt.Printf("Level %d: %x\n", level, hashes)

		// 홀수일 경우 마지막 해시를 복제
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		// 부모 노드 배열 생성
		var parentHashes [][]byte
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			newHash := utils.Keccak256(combined)
			parentHashes = append(parentHashes, newHash)
		}

		// 부모 해시로 대체
		hashes = parentHashes
	}

	// 루트 해시 반환
	merkleRoot := utils.BytesToHex(hashes[0])
	fmt.Printf("Final Merkle Root: %s\n", merkleRoot)
	return merkleRoot, nil
}
