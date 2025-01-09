package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/account"
	"simple_p2p_client/leveldb"

	db "github.com/syndtr/goleveldb/leveldb"
)

var NodeAccount string // 프로그램을 실행하는 노드의 주소

// 노드 계정을 초기화하는 함수
func InitializeNodeAccount() error {
	// DB 접근
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("DB 접근 실패 : %v", err)
	}

	// 기존 노드 계정 로드
	nodeAccountKey := "nodeAccount"
	accountData, err := dbInstance.Get([]byte(nodeAccountKey), nil)
	if err == nil {
		// 기존 계정 있으면 불러오기
		NodeAccount = string(accountData)
		fmt.Printf("[ACCOUNT] Node Account load complete : %s\n", NodeAccount)
		return nil
	}

	// 없으면 생성
	_, address, err := account.CreateAccount()
	if err != nil {
		return fmt.Errorf("노드 게정 생성 실패 : %v", err)
	}

	// 새 계정 데이터 준비
	accountInfo := account.Account{
		Balance: big.NewInt(0),
		Nonce:   0,
	}

	accountJSON, err := json.Marshal(accountInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal accountInfo : %v", err)
	}

	// batch 생성
	batch := new(db.Batch)
	batch.Put([]byte(nodeAccountKey), []byte(address))
	batch.Put([]byte("account:"+address), accountJSON)

	// 배치 실행
	err = dbInstance.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("배치 작업 저장 실패 : %v", err)
	}

	NodeAccount = address
	fmt.Printf("[ACCOUNT] New Account for node created : %s\n", NodeAccount)
	return nil
}
