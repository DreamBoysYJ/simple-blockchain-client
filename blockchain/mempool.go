package blockchain

import (
	"fmt"
	"simple_p2p_client/account"
	"sort"
	"sync"
)

type Mempool struct {
	pending map[string]map[uint64]Transaction
	future  map[string]map[uint64]Transaction
	mu      sync.Mutex
}

var defaultMempool *Mempool

func InitMempool() {
	defaultMempool = &Mempool{
		pending: make(map[string]map[uint64]Transaction),
		future:  make(map[string]map[uint64]Transaction),
	}
	fmt.Println("[Mempool] Initialized...")
}

// 멤풀에 트랜잭션 추가 (Nonce에 따라 pending, future로 나눠서)
func (mp *Mempool) AddTransaction(tx Transaction, currentNonce uint64) error {

	// DB에서 account 정보 가져옴

	fromAccount, err := account.GetAccount(tx.From)
	if err != nil {
		return fmt.Errorf("failed to retrieve account from DB : %v", err)
	}

	// DB에서 currentNonce를 가져옴
	dbNonce := fromAccount.Nonce

	// fmt.Printf("Current nonce from DB IS : %v", dbNonce)

	// 멤풀 락
	// fmt.Println("Acquiring lock in AddTransaction...")
	mp.mu.Lock()
	// fmt.Println("Lock acquired in AddTransaction")
	defer func() {
		// fmt.Println("Releasing lock in AddTransaction")
		mp.mu.Unlock()
	}()
	// 계정별 맵 초기화
	if _, exists := mp.pending[tx.From]; !exists {
		mp.pending[tx.From] = make(map[uint64]Transaction)
	}
	if _, exists := mp.future[tx.From]; !exists {
		mp.future[tx.From] = make(map[uint64]Transaction)
	}

	// 중복 확인: pending과 future에 이미 존재하는 논스인지 확인
	if _, exists := mp.pending[tx.From][tx.Nonce]; exists {
		return fmt.Errorf("duplicate transaction: nonce %v already exists in pending", tx.Nonce)
	}
	if _, exists := mp.future[tx.From][tx.Nonce]; exists {
		return fmt.Errorf("duplicate transaction: nonce %v already exists in future", tx.Nonce)
	}

	// pending에서 from의 마지막 논스 계산
	highestPendingNonce := dbNonce
	for nonce := range mp.pending[tx.From] {
		if nonce > highestPendingNonce {
			highestPendingNonce = nonce
		}
	}

	// highestPendingNonce + 1이면 Pending, 아니면 future에 저장
	if tx.Nonce == highestPendingNonce+1 {
		// Pending queue에 저장
		mp.pending[tx.From][tx.Nonce] = tx
		fmt.Printf("[Mempool] : tx is stored in pending, nonce is : %v\n", tx.Nonce)
		fmt.Println()

	} else if tx.Nonce > highestPendingNonce {
		// Future queue에 저장
		mp.future[tx.From][tx.Nonce] = tx
		fmt.Printf("[Mempool] : tx is stored in future, nonce is : %v\n", tx.Nonce)
		fmt.Println()

	} else {
		// db 논스보다 작음
		return fmt.Errorf("invalid transaction: nonce too low")
	}

	return nil
}

// GetPendingTransactions 함수 : 특정 Account의 Pending 트랜잭션 가져오기
func (mp *Mempool) GetPendingTransactions(account string) []Transaction {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()
	txs := []Transaction{}
	if pendingTxs, exists := mp.pending[account]; exists {
		for _, tx := range pendingTxs {
			txs = append(txs, tx)
		}
	}
	return txs
}

// PromoteFutureToPending 함수 : Future 트랜잭션을 Pending으로 이동
func (mp *Mempool) PromoteFutureToPending(account string, currentNonce uint64) {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()
	if futureTxs, exists := mp.future[account]; exists {
		for nonce, tx := range futureTxs {
			if nonce == currentNonce+1 {
				delete(mp.future[account], nonce)
				mp.pending[account][nonce] = tx
			}
		}
	}
}

// SelectTransactionsForBlock : 블록 생성시 트랜잭션 선택
func (mp *Mempool) SelectTransactionsForBlock(account string, currentNonce uint64) ([]Transaction, error) {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()
	blockTxs := []Transaction{}

	// Pending queue에서 트랜잭션 선택
	if pendingTxs, exists := mp.pending[account]; exists {
		for nonce := currentNonce + 1; ; nonce++ {
			tx, exists := pendingTxs[nonce]
			if !exists {
				break
			}
			blockTxs = append(blockTxs, tx)
			delete(mp.pending[account], nonce)
		}
	}
	return blockTxs, nil
}

// 멤풀 정리 :  블록 생성 시도시 유효한 Future를 Pending으로 옮기고 필요없는 트랜잭션 정리
func (mp *Mempool) CleanMempool() {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()
	for account, futureTxs := range mp.future {
		if pendingTxs, exists := mp.pending[account]; exists {
			// 현재 pending 논스 계산
			currentNonce := uint64(0)
			for nonce := range pendingTxs {
				if nonce > currentNonce {
					currentNonce = nonce
				}
			}

			// Future에서 Pending으로 옮기기
			for nonce, tx := range futureTxs {
				if nonce == currentNonce+1 {
					mp.pending[account][nonce] = tx
					delete(mp.future[account], nonce)
					currentNonce++
				}
			}

			// Future 비우기
			delete(mp.future, account)
		}
	}
}

// 블록 생성 시도시, 유효한 future 트랜잭션들을 Pending으로 옮기는
func (mp *Mempool) SyncFutureToPending() {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()

	// 멤풀 내 모든 계정 순회
	for account, futureTxs := range mp.future {
		pendingTxs, pendingExists := mp.pending[account]
		if !pendingExists {
			// pending에는 없다면 초기화
			pendingTxs = make(map[uint64]Transaction)
			mp.pending[account] = pendingTxs
		}

		// Pending에서 가장 높은 nonce 찾기
		higestPendingNonce := uint64(0)
		for nonce := range mp.pending[account] {
			if nonce > higestPendingNonce {
				higestPendingNonce = nonce
			}
		}

		// Future 트랜잭션을 pending으로 이동
		for nonce := higestPendingNonce + 1; ; nonce++ {
			tx, exists := futureTxs[nonce]
			if !exists {
				break // 연속된 Nonce 없으면 종료
			}

			// Pending으로 이동
			pendingTxs[nonce] = tx
			delete(futureTxs, nonce)
		}

		// Future에서 해당 계정이 비어있으면 삭제
		if len(futureTxs) == 0 {
			delete(mp.future, account)
		}
	}
}

// 라운드로빈(주소별)으로 트랜잭션 추출
func (mp *Mempool) ExtractTransactionsForBlock(maxTxs int) []Transaction {

	var blockTxs []Transaction

	accounts := make([]string, 0, len(mp.pending))
	for account := range mp.pending {
		accounts = append(accounts, account)
	}

	for len(blockTxs) < maxTxs {
		noMoreTxs := true
		for _, account := range accounts {
			if len(blockTxs) >= maxTxs {
				break
			}

			pendingTxs := mp.pending[account]
			if len(pendingTxs) == 0 {
				fmt.Printf("No transactions for account: %s\n", account)
				continue
			}

			noMoreTxs = false
			nonces := make([]uint64, 0, len(pendingTxs))
			for nonce := range pendingTxs {
				nonces = append(nonces, nonce)
			}
			sort.Slice(nonces, func(i, j int) bool { return nonces[i] < nonces[j] })

			lowestNonce := nonces[0]
			fmt.Printf("[Mempool] Extracting transaction with nonce %d for account %s\n", lowestNonce, account)

			blockTxs = append(blockTxs, pendingTxs[lowestNonce])
			delete(pendingTxs, lowestNonce)

			if len(pendingTxs) == 0 {
				// fmt.Printf("All transactions for account %s processed. Removing from pending.\n", account)
				delete(mp.pending, account)
			}
		}

		if noMoreTxs {
			fmt.Println("No more transactions available for block.")
			break
		}
	}

	fmt.Printf("Final block transactions: %v\n", blockTxs)
	return blockTxs
}

// 블록을 피어로부터 수신 후 검증, 저장, 트랜잭션 실행을 마쳤을 경우, 멤풀에 중복 트랜잭션이 있을 경우 제거
func (mp *Mempool) CleanMempoolAfterReceiveBlock(blockTxs []Transaction) {

	mp.mu.Lock()

	defer func() {

		mp.mu.Unlock()
	}()
	for _, tx := range blockTxs {

		// pending 제거
		if accountTxs, exists := mp.pending[tx.From]; exists {
			delete(accountTxs, tx.Nonce)
			// 주소의 pending이 비었을 경우,
			if len(accountTxs) == 0 {
				delete(mp.pending, tx.From)
			}
		}

		// future 제거
		if futureTxs, exists := mp.future[tx.From]; exists {
			delete(futureTxs, tx.Nonce)
			// 주소 future이 비었을 경우
			if len(futureTxs) == 0 {
				delete(mp.future, tx.From)
			}
		}
	}
}
