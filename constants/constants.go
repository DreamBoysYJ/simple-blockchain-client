package constants

import "time"

const (
	TransactionsPerBlock  = 5                // 블록 당 트랜잭션 개수
	BlockCreationInterval = 10 * time.Second // 블록 생성 주기
	BootstrapNodeAddress  = "localhost:8282" // 하드코딩된 부트스트랩 노드 주소
)
